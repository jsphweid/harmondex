package cmd

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/jsphweid/mir1/chord"
	"github.com/jsphweid/mir1/model"
	"github.com/spf13/cobra"
	"gitlab.com/gomidi/midi/v2"
	_ "gitlab.com/gomidi/midi/v2/drivers/rtmididrv" // autoregisters driver
)

func init() {
	rootCmd.AddCommand(serverCmd)
}

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Runs server",
	Long:  `Runs server`,
	Run: func(cmd *cobra.Command, args []string) {
		startServer()
	},
}

func loadAllChunks() []model.Chunk {
	f, err := os.Open("out/allChunks.dat")
	if err != nil {
		panic("Could not load allChunks file: " + err.Error())
	}
	defer f.Close()

	var b []byte
	_, err = f.Read(b)
	if err != nil {
		panic(err)
	}

	var chunks []model.Chunk
	decoder := gob.NewDecoder(f)
	err = decoder.Decode(&chunks)
	if err != nil {
		panic("Could not decode allChunks file: " + err.Error())
	}

	return chunks
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

	var index model.Index
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

func parseResult(buf []byte) []model.RawResult {
	var res []model.RawResult
	for i := 0; i < len(buf); i += 12 {
		var rr model.RawResult
		rr.AbsTime = binary.LittleEndian.Uint64(buf[i : i+8])
		rr.FileId = binary.LittleEndian.Uint32(buf[i+8 : i+12])
		res = append(res, rr)
	}
	return res
}

func findChords(allChunks []model.Chunk, onNotes chord.OnNotes) {
	if len(onNotes) == 0 {
		return
	}

	// TODO: return something
	keys := make([]uint8, 0, len(onNotes))
	for k := range onNotes {
		keys = append(keys, k)
	}
	chordKey := chord.CreateChordKey(keys)
	for _, chunk := range allChunks {
		if chordKey >= chunk.Start && chordKey <= chunk.End {
			res := findChordsInChunk(chunk.Filename, chordKey)
			fmt.Printf("res!!!!!: %v\n", res)
			return // TODO
		}
	}

	// TODO
	// fmt.Printf("Could not find a chord for: %v\n", onNotes)
}

func startServer() {
	defer midi.CloseDriver()
	in, err := midi.InPort(0)
	if err != nil {
		fmt.Println("can't find VMPK")
		return
	}

	allChunks := loadAllChunks()
	onNotes := make(chord.OnNotes)

	// manage current midi notes
	// retrieve
	// concurrency, also, debounce

	stop, err := midi.ListenTo(in, func(msg midi.Message, timestampms int32) {
		var bt []byte
		var ch, key, vel uint8
		switch {
		case msg.GetSysEx(&bt):
			fmt.Printf("got sysex: % X\n", bt)
		case msg.GetNoteStart(&ch, &key, &vel):
			onNotes[key] = true
			findChords(allChunks, onNotes)
		case msg.GetNoteEnd(&ch, &key):
			delete(onNotes, key)
			findChords(allChunks, onNotes)
		default:
			// ignore
		}
	}, midi.UseSysEx())

	if err != nil {
		fmt.Printf("ERROR: %s\n", err)
		return
	}

	time.Sleep(time.Second * 5000) // lol
	stop()
}
