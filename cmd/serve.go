package cmd

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/jsphweid/harmondex/chord"
	"github.com/jsphweid/harmondex/chunk"
	"github.com/jsphweid/harmondex/constants"
	"github.com/jsphweid/harmondex/db"
	"github.com/jsphweid/harmondex/model"
	"github.com/jsphweid/harmondex/util"
	"github.com/rs/cors"
	"github.com/spf13/cobra"
)

var allChunks []model.ChunkOverview
var fileNumMap model.FileNumToMidiPath

func init() {
	rootCmd.AddCommand(serveCmd)
}

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "serves",
	Long:  `serves`,
	Run: func(cmd *cobra.Command, args []string) {
		serve()
	},
}

func parseResult(buf []byte) []model.RawResult {
	var res []model.RawResult
	for i := 0; i < len(buf); i += 8 {
		var rr model.RawResult
		rr.Offset = binary.LittleEndian.Uint32(buf[i : i+4])
		rr.FileId = binary.LittleEndian.Uint32(buf[i+4 : i+8])
		res = append(res, rr)
	}
	return res
}

func findChordsInChunk(filename string, chordKey string) []model.RawResult {
	// read chunk
	f := util.OpenFileOrPanic(filepath.Join(constants.GetIndexDir(), filename))
	index, _ := chunk.ReadIndexOrPanic(f)

	val, ok := index[chordKey]
	if ok {
		// advance file byte pointer to start position from current
		// TODO: add pagination
		f.Seek(int64(val.Start), os.SEEK_CUR)
		bytesToRead := val.End - val.Start
		buf := make([]byte, bytesToRead)
		_, err := io.ReadFull(f, buf)
		if err != nil {
			panic("Could not read from seeked positon: " + err.Error())
		}
		return parseResult(buf)
	}

	var emptyResults []model.RawResult
	return emptyResults
}

func findChords(notes model.Notes) []model.RawResult {
	var empty []model.RawResult

	if len(notes) == 0 {
		return empty
	}

	chordKey := chord.CreateChordKey(notes)
	for _, chunk := range allChunks {
		if chordKey >= chunk.Start && chordKey <= chunk.End {
			return findChordsInChunk(chunk.Filename, chordKey)
		}
	}

	return empty
}

func matchesToResults(matches []model.RawResult) []model.SearchResult {
	res := make([]model.SearchResult, 0)
	fileIdToMetadata := fetchMidiMetadata(matches)
	for _, rr := range matches {
		searchResult := model.SearchResult{
			FileId:       rr.FileId,
			Offset:       float32(rr.Offset) / 1000, // convert to seconds because it's clearer
			MidiMetadata: fileIdToMetadata[rr.FileId],
		}
		res = append(res, searchResult)
	}
	return res
}

func fetchMidiMetadata(matches []model.RawResult) map[uint32]*model.MidiMetadata {
	res := make(map[uint32]*model.MidiMetadata)
	uniqueFileIds := make(map[uint32]bool)
	for _, rr := range matches {
		uniqueFileIds[rr.FileId] = true
	}
	fileIds := util.GetKeys(uniqueFileIds)
	var filenames []string
	filenameToFileId := make(map[string]uint32)
	for _, fileId := range fileIds {
		filename := fileNumMap[fileId]
		filenames = append(filenames, filename)
		filenameToFileId[filename] = fileId
	}
	for filename, metadata := range db.GetMidiMetadatas(filenames) {
		res[filenameToFileId[filename]] = &metadata
	}
	return res
}

func handleGetFile(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	fileNum, err := strconv.Atoi(id)
	if err != nil {
		return
	}
	if filename, ok := fileNumMap[uint32(fileNum)]; ok {
		path := filepath.Join(constants.GetMediaDir(), filename)
		bytes, err := ioutil.ReadFile(path)
		if err != nil {
			fmt.Println("Error reading midi file: " + err.Error())
			return
		}
		w.Write(bytes)
	}
}

func handleSearch(w http.ResponseWriter, r *http.Request) {
	reqBody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		fmt.Println(w, "Kindly enter data with the event title and description only in order to update")
	}

	var input model.SearchRequestBody

	err = json.Unmarshal(reqBody, &input)
	if err != nil {
		fmt.Println("Could not unmarshal request body: " + err.Error())
	}

	if len(input.Chords) != 1 {
		http.Error(w, "Length of chords can only be 1 for now...", 400)
		return
	}

	matches := findChords(input.Chords[0])
	max_matches := util.Min(len(matches), 10)
	json.NewEncoder(w).Encode(matchesToResults(matches[:max_matches]))
}

func UnauthorizedHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(401)
	fmt.Fprintf(w, "401 Unauthorized\n")
}

func serve() {
	allChunks = util.ReadBinaryOrPanic[[]model.ChunkOverview](constants.GetIndexDir() + "/allChunks.dat")
	fileNumMap = util.ReadBinaryOrPanic[model.FileNumToMidiPath](constants.GetIndexDir() + "/indexToPath.dat")

	router := mux.NewRouter()
	router.HandleFunc("/search", handleSearch).Methods("POST")
	router.HandleFunc("/file/{id}", handleGetFile).Methods("GET")

	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"http://localhost:3500"},
		AllowCredentials: true,
	})

	handler := c.Handler(router)

	log.Fatal(http.ListenAndServe(":8080", handler))
}
