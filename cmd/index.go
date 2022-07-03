package cmd

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"sort"

	"github.com/google/uuid"
	"github.com/jsphweid/mir1/chord"
	"github.com/jsphweid/mir1/midi"
	"github.com/jsphweid/mir1/model"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(indexCmd)
}

var indexCmd = &cobra.Command{
	Use:   "index",
	Short: "Creates index",
	Long:  `Creates index`,
	Run: func(cmd *cobra.Command, args []string) {
		run()
	},
}

const ChordSize = 28
const PreferredChunkSize = 64 * 1024 * 1024

type Chunk struct {
	Start    string
	End      string
	Filename string
}

type Pair struct {
	Start uint32
	End   uint32
}

func recreateOutputDir() {
	os.RemoveAll("out/")
	os.MkdirAll("out/", 0777)
}

func makeKey(chord model.Chord) string {
	notes := chord.Notes
	sort.Slice(notes, func(i, j int) bool {
		return notes[i] < notes[j]
	})
	var res string
	for i, note := range notes {
		res += fmt.Sprintf("%v", note)
		if i < len(notes)-1 {
			res += "-"
		}
	}
	return res
}

func createBinary(filename string, data any) {
	buf := new(bytes.Buffer)
	encoder := gob.NewEncoder(buf)

	// Encoding the map
	err := encoder.Encode(data)
	if err != nil {
		panic(err)
	}
	f, err := os.Create(filename)
	if err != nil {
		fmt.Println("Couldn't open file: "+filename, err)
	}
	defer f.Close()

	_, err = f.Write(buf.Bytes())
	if err != nil {
		fmt.Println("Write failed for file: "+filename, err)
	}
}

func maybeWriteChord(chord model.Chord) {
	// ignore really short or really long chords
	if len(chord.Notes) < 2 || len(chord.Notes) > 16 {
		return
	}

	// order them
	sort.Slice(chord.Notes, func(i, j int) bool {
		return chord.Notes[i] < chord.Notes[j]
	})

	// put them in notes
	var notes [16]uint8
	copy(notes[:], chord.Notes)

	filename := "out/" + fmt.Sprintf("%03d", notes[0]) + ".dat"
	f, err := os.OpenFile(filename, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0777)
	if err != nil {
		panic(err)
	}

	defer f.Close()

	var bytes [ChordSize]byte
	copy(bytes[:], notes[:])
	binary.LittleEndian.PutUint64(bytes[16:24], chord.AbsTime)
	binary.LittleEndian.PutUint32(bytes[24:28], chord.FileId)
	if _, err = f.Write(bytes[:]); err != nil {
		panic(err)
	}
}

func getKeysSorted(m map[string][]model.Chord) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		return keys[i] < keys[j]
	})
	return keys
}

func getEncodedMapSize(m map[string]Pair) uint32 {
	buf := new(bytes.Buffer)
	encoder := gob.NewEncoder(buf)
	err := encoder.Encode(m)
	if err != nil {
		panic("error getting map size: " + err.Error())
	}
	return uint32(len(buf.Bytes()))
}

func makeChunk(m map[string][]model.Chord, sortedKeys []string) Chunk {
	var c Chunk
	c.Filename = uuid.New().String() + ".dat"
	c.Start = sortedKeys[0]
	c.End = sortedKeys[len(sortedKeys)-1]

	// fill up index with 0 values
	index := make(map[string]Pair)
	for _, key := range sortedKeys {
		var p Pair
		index[key] = p
	}
	indexSize := getEncodedMapSize(index)
	dataOffset := indexSize + 4 // 4 bytes for encoding length of index

	// fill up data section
	dataBuf := new(bytes.Buffer)
	for i, key := range sortedKeys {
		chords := m[key]
		p := index[key]
		p.Start = uint32(dataOffset)
		if i > 0 {
			pp := index[sortedKeys[i-1]]
			pp.End = uint32(dataOffset)
		}
		for _, chord := range chords {
			binary.Write(dataBuf, binary.LittleEndian, chord.FileId)
			binary.Write(dataBuf, binary.LittleEndian, chord.AbsTime)
			dataOffset += 12
		}
	}
	// set last one
	p := index[sortedKeys[len(sortedKeys)-1]]
	p.End = uint32(dataOffset)

	// encode index
	indexBuf := new(bytes.Buffer)
	encoder := gob.NewEncoder(indexBuf)
	err := encoder.Encode(index)
	if err != nil {
		panic("error making chunk, couldn't encode to get size: " + err.Error())
	}

	sizeBuf := new(bytes.Buffer)
	binary.Write(sizeBuf, binary.LittleEndian, indexSize)

	var finalBytes []byte
	finalBytes = append(finalBytes, sizeBuf.Bytes()...)
	finalBytes = append(finalBytes, indexBuf.Bytes()...)
	finalBytes = append(finalBytes, dataBuf.Bytes()...)

	filename := "out/" + c.Filename
	f, err := os.Create(filename)
	if err != nil {
		fmt.Println("Couldn't open file: "+filename, err)
	}
	defer f.Close()
	_, err = f.Write(finalBytes)
	if err != nil {
		fmt.Println("Write failed for file: "+filename, err)
	}
	return c
}

func maybeMakeChunks(m map[string][]model.Chord, force bool) []Chunk {
	var currSize int
	var currKeys []string

	sortedKeys := getKeysSorted(m)
	var createdChunks []Chunk

	for i, key := range sortedKeys {
		if i != 0 && currSize > PreferredChunkSize && len(currKeys) > 0 {
			chunk := makeChunk(m, currKeys)
			createdChunks = append(createdChunks, chunk)
			currSize = 0
		}

		currKeys = append(currKeys, key)
		chords := m[key]
		// each chord will take up uint32 and uint64 == 12 bytes
		currSize += len(chords) * 12
		// each index will take up some vari length + uint32 == 28 bytes
		// NOTE: note completely accurate because we're encoding a map when we write
		currSize += len(key) + 4
	}

	if len(currKeys) > 0 && (currSize > PreferredChunkSize || force) {
		chunk := makeChunk(m, currKeys)
		createdChunks = append(createdChunks, chunk)
	}

	return createdChunks
}

func makeChunks() {
	// read big files
	files, err := ioutil.ReadDir("./out")
	if err != nil {
		fmt.Println("Could not make chunks because out file not read:" + err.Error())
	}

	m := make(map[string][]model.Chord)
	var allChunks []Chunk

	// make chunks
	for _, file := range files {
		filename := "out/" + file.Name()
		f, err := os.Open(filename)
		if err != nil {
			panic("Couldn't read big file: " + err.Error())
		}

		r := bufio.NewReader(f)

		for {
			buf := make([]byte, ChordSize)
			n, err := r.Read(buf)
			buf = buf[:n]
			if n == 0 { // reached end of file?
				if err != nil {
					fmt.Println(err)
					break
				}
				if err == io.EOF {
					break
				}
			}
			var chord model.Chord
			chord.Notes = buf[:16]
			chord.AbsTime = binary.LittleEndian.Uint64(buf[16:24])
			chord.FileId = binary.LittleEndian.Uint32(buf[24:28])
			key := makeKey(chord)
			bucket := m[key]
			bucket = append(bucket, chord)
			m[key] = bucket
		}
		chunks := maybeMakeChunks(m, false)
		allChunks = append(allChunks, chunks...)
		os.Remove(filename) // remove big file now that we're done with it
	}
	chunks := maybeMakeChunks(m, true)
	allChunks = append(allChunks, chunks...)

	createBinary("out/allChunks.dat", allChunks)
}

func run() {

	// delete all every time
	recreateOutputDir()

	// map index to filename and store in file
	indexToPath := make(map[int64]string)

	files, err := ioutil.ReadDir("./lmd_full/0")
	if err != nil {
		log.Fatal(err)
	}
	var numChords int

	// collate data in big files
	for i, f := range files {
		filename := f.Name()
		path := "/0/" + f.Name()
		indexToPath[int64(i)] = path
		fullpath := "/Users/jsphweid/git/mir1/lmd_full" + path
		parsed, err := midi.ReadMidiFile(fullpath)

		if err != nil {
			fmt.Printf("Skipping %v because error reading with file: %v\n", filename, err)
			continue
		}
		chords, err := chord.ParseChords(parsed)

		for _, chord := range chords {
			chord.FileId = uint32(i)
			maybeWriteChord(chord)
		}

		numChords += len(chords)
		if err != nil {
			log.Print(err)
		}

		// TODO: temp
		if i >= 100 {
			break
		}
	}

	makeChunks()

	createBinary("out/indexToPath.dat", indexToPath)
}
