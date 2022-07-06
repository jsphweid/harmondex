package cmd

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/bep/debounce"
	"github.com/jsphweid/mir1/chord"
	"github.com/jsphweid/mir1/model"
	"github.com/jsphweid/mir1/sample"
	"github.com/spf13/cobra"
	"gitlab.com/gomidi/midi/v2"
	_ "gitlab.com/gomidi/midi/v2/drivers/rtmididrv" // autoregisters driver
	"gitlab.com/gomidi/midi/v2/smf"
)

func init() {
	rootCmd.AddCommand(serverCmd)
}

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Runs server",
	Long:  `Runs server`,
	Run: func(cmd *cobra.Command, args []string) {
		serve()
	},
}

func loadIndexToPath() map[uint32]string {
	f, err := os.Open("out/indexToPath.dat")
	if err != nil {
		panic("Could not load indexToPath file: " + err.Error())
	}
	defer f.Close()

	var b []byte
	_, err = f.Read(b)
	if err != nil {
		panic(err)
	}

	res := make(map[uint32]string)
	decoder := gob.NewDecoder(f)
	err = decoder.Decode(&res)
	if err != nil {
		panic("Could not decode indexToPath file: " + err.Error())
	}

	return res
}

func loadAllChunks() []model.ChunkOverview {
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

	var chunks []model.ChunkOverview
	decoder := gob.NewDecoder(f)
	err = decoder.Decode(&chunks)
	if err != nil {
		panic("Could not decode allChunks file: " + err.Error())
	}

	for _, c := range chunks {
		fmt.Printf("c.Filename: %v\n", c.Filename)
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

func findChords(allChunks []model.ChunkOverview, onNotes chord.OnNotes) []model.RawResult {
	var empty []model.RawResult

	if len(onNotes) == 0 {
		return empty
	}

	// TODO: return something
	keys := make([]uint8, 0, len(onNotes))
	for k := range onNotes {
		keys = append(keys, k)
	}
	chordKey := chord.CreateChordKey(keys)
	for _, chunk := range allChunks {
		if chordKey >= chunk.Start && chordKey <= chunk.End {
			fmt.Printf("found in ------- chunk: %v\n", chunk)
			fmt.Printf("chunk.Start: %v\n", chunk.Start)
			fmt.Printf("chunk.End: %v\n", chunk.End)
			return findChordsInChunk(chunk.Filename, chordKey)
		}
	}

	return empty
}

func playMidi(filename string, ticksOffset uint64) {
	fmt.Printf("filename: %v\n", filename)
	f, err := os.Open(filename)
	if err != nil {
		panic("Could not load " + filename + ": " + err.Error())
	}
	defer f.Close()

	out, err := midi.FindOutPort("FluidSynth")
	if err != nil {
		panic("Can't find qsynth")
	}

	// TODO: should be ticks...
	mf := smf.ReadTracksFrom(f).SMF()
	sample := sample.Create(mf, ticksOffset)
	var bf bytes.Buffer
	sample.WriteTo(&bf)
	reader := bytes.NewReader(bf.Bytes())
	smf.ReadTracksFrom(reader).Play(out)
}

func playSomething(results []model.RawResult, lookup map[uint32]string) {
	if len(results) == 0 {
		return
	}

	result := results[0]
	filename := lookup[result.FileId]
	playMidi("lmd_full"+filename, result.TicksOffset)
}

func startServer() {
	defer midi.CloseDriver()
	in, err := midi.InPort(0)
	if err != nil {
		fmt.Println("can't find VMPK")
		return
	}

	indexToPath := loadIndexToPath()
	allChunks := loadAllChunks()
	onNotes := make(chord.OnNotes)

	// manage current midi notes
	// retrieve
	// concurrency, also, debounce

	play := func() {
		playSomething(findChords(allChunks, onNotes), indexToPath)
	}

	debounced := debounce.New(100 * time.Millisecond)

	stop, err := midi.ListenTo(in, func(msg midi.Message, timestampms int32) {
		var bt []byte
		var ch, key, vel uint8
		switch {
		case msg.GetSysEx(&bt):
			fmt.Printf("got sysex: % X\n", bt)
		case msg.GetNoteStart(&ch, &key, &vel):
			onNotes[key] = true
			debounced(play)
		case msg.GetNoteEnd(&ch, &key):
			delete(onNotes, key)
			debounced(play)
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
