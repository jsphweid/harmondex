package cmd

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"github.com/jsphweid/harmondex/chord"
	"github.com/jsphweid/harmondex/constants"
	"github.com/jsphweid/harmondex/model"
	"github.com/spf13/cobra"
	_ "gitlab.com/gomidi/midi/v2/drivers/rtmididrv" // autoregisters driver
)

var allChunks []model.ChunkOverview

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

func loadAllChunks() []model.ChunkOverview {
	f, err := os.Open(constants.OutDir + "/allChunks.dat")
	if err != nil {
		panic("Could not load allChunks file: " + err.Error())
	}
	defer f.Close()

	var b []byte
	_, err = f.Read(b)
	if err != nil {
		panic(err)
	}

	var chunks []model.ChunkOverview
	decoder := gob.NewDecoder(f)
	err = decoder.Decode(&chunks)
	if err != nil {
		panic("Could not decode allChunks file: " + err.Error())
	}

	return chunks
}

func parseResult(buf []byte) []model.RawResult {
	var res []model.RawResult
	for i := 0; i < len(buf); i += 12 {
		var rr model.RawResult
		rr.TicksOffset = binary.LittleEndian.Uint64(buf[i : i+8])
		rr.FileId = binary.LittleEndian.Uint32(buf[i+8 : i+12])
		res = append(res, rr)
	}
	return res
}

func findChordsInChunk(filename string, chordKey string) []model.RawResult {
	// read chunk
	f, err := os.Open("out/" + filename)
	if err != nil {
		panic("Could not open file: " + err.Error())
	}

	buf := make([]byte, 4)
	_, err = io.ReadFull(f, buf)
	if err != nil {
		panic("Could not read first 4 bytes: " + err.Error())
	}
	indexLength := binary.LittleEndian.Uint32(buf)

	buf = make([]byte, indexLength)
	_, err = io.ReadFull(f, buf)
	if err != nil {
		panic("Could not read first 4 bytes: " + err.Error())
	}

	var index model.ChunkIndex
	// NOTE: seems silly to have to do this
	decoder := gob.NewDecoder(bytes.NewReader(buf))
	err = decoder.Decode(&index)
	if err != nil {
		panic("Could not decode allChunks file: " + err.Error())
	}
	val, ok := index[chordKey]
	if ok {
		// advance file byte pointer to start position from current
		// TODO: add pagination
		f.Seek(int64(val.Start), os.SEEK_CUR)
		bytesToRead := val.End - val.Start
		buf = make([]byte, bytesToRead)
		_, err = io.ReadFull(f, buf)
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

	res := make([]model.SearchResult, 0)
	for _, rr := range matches {
		res = append(res, model.SearchResult{FileId: rr.FileId, TicksOffset: rr.TicksOffset})
	}
	json.NewEncoder(w).Encode(res)
}

func serve() {
	allChunks = loadAllChunks()

	router := mux.NewRouter().StrictSlash(true)
	router.HandleFunc("/search", handleSearch).Methods("POST")
	log.Fatal(http.ListenAndServe(":8080", router))
}
