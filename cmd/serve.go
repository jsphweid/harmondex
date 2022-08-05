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
	f := util.OpenFileOrPanic(filepath.Join(util.GetIndexDir(), filename))
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

func fetchMidiMetadata(fileIds []uint32) map[uint32]model.MidiMetadata {
	res := make(map[uint32]model.MidiMetadata)
	var filenames []string
	filenameToFileId := make(map[string]uint32)
	for _, fileId := range fileIds {
		filename := fileNumMap[fileId]
		filenames = append(filenames, filename)
		filenameToFileId[filename] = fileId
	}
	filenameToMetadata := db.GetMidiMetadatas(filenames)
	for filename, metadata := range filenameToMetadata {
		res[filenameToFileId[filename]] = metadata
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
		path := filepath.Join(util.GetMediaDir(), filename)
		bytes, err := ioutil.ReadFile(path)
		if err != nil {
			fmt.Println("Error reading midi file: " + err.Error())
			return
		}
		w.Write(bytes)
	}
}

func sendSearchResponse(w http.ResponseWriter, matches []model.RawResult, start int) {
	var uniqueFileIds []uint32
	fileIdToOffsets := make(map[uint32][]float32)

	for _, match := range matches {
		offset := float32(match.Offset) / 1000 // convert to seconds because it's clearer
		if offsets, ok := fileIdToOffsets[match.FileId]; ok {
			fileIdToOffsets[match.FileId] = append(offsets, offset)
		} else {
			uniqueFileIds = append(uniqueFileIds, match.FileId)
			fileIdToOffsets[match.FileId] = []float32{offset}
		}
	}

	var resp model.SearchResponse
	resp.NumFiles = len(uniqueFileIds)
	resp.NumMatches = len(matches)
	resp.Start = start // TODO: is this really that valuable?
	resp.Results = []model.SearchResultV2{}

	if start >= len(uniqueFileIds) {
		json.NewEncoder(w).Encode(resp)
		return
	}

	ids := uniqueFileIds[start:util.Min(len(uniqueFileIds), start+10)]
	fileIdToMetadata := fetchMidiMetadata(ids)
	for _, id := range ids {
		var sr model.SearchResultV2
		sr.FileId = id
		sr.Offsets = fileIdToOffsets[id]
		sr.MidiMetadata = nil
		if _, ok := fileIdToMetadata[id]; ok {
			val := fileIdToMetadata[id]
			sr.MidiMetadata = &val
		}
		resp.Results = append(resp.Results, sr)
	}

	json.NewEncoder(w).Encode(resp)
}

func getStart(r *http.Request) int {
	start := r.URL.Query().Get("start")
	num, err := strconv.Atoi(start)

	if err != nil {
		return 0
	}

	return num
}

func HandleSearch(w http.ResponseWriter, r *http.Request) {
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
	start := getStart(r)
	sendSearchResponse(w, matches, start)
}

func UnauthorizedHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(401)
	fmt.Fprintf(w, "401 Unauthorized\n")
}

func LoadServeFiles() {
	// NOTE: this should be exposed but I don't immediately know a
	// better way to make this file easily testable than to do this
	allChunks = util.ReadBinaryOrPanic[[]model.ChunkOverview](util.GetAllChunksPath())
	fileNumMap = util.ReadBinaryOrPanic[model.FileNumToMidiPath](util.GetFileNumToNamePath())
}

func serve() {
	LoadServeFiles()
	router := mux.NewRouter()
	router.HandleFunc("/search", HandleSearch).Methods("POST")
	router.HandleFunc("/file/{id}", handleGetFile).Methods("GET")

	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"http://localhost:3500"},
		AllowCredentials: true,
	})

	handler := c.Handler(router)

	log.Fatal(http.ListenAndServe(":8080", handler))
}
