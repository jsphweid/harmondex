package chunk

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"

	"github.com/google/uuid"
	"github.com/jsphweid/harmondex/bucket"
	"github.com/jsphweid/harmondex/chord"
	"github.com/jsphweid/harmondex/constants"
	"github.com/jsphweid/harmondex/model"
	"github.com/jsphweid/harmondex/util"
)

type ChordKeyToChords = map[string][]model.Chord

func getKeysSorted(m ChordKeyToChords) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		return keys[i] < keys[j]
	})
	return keys
}

func getEncodedMapSize(m map[string]model.Pair) uint32 {
	buf := new(bytes.Buffer)
	encoder := gob.NewEncoder(buf)
	err := encoder.Encode(m)
	if err != nil {
		panic("error getting map size: " + err.Error())
	}
	return uint32(len(buf.Bytes()))
}

func makeChunkOverview(sortedKeys []string) model.ChunkOverview {
	var c model.ChunkOverview
	c.Filename = uuid.New().String() + ".dat"
	c.Start = sortedKeys[0]
	c.End = sortedKeys[len(sortedKeys)-1]
	return c
}

func makeChunk(m ChordKeyToChords, sortedKeys []string) model.ChunkOverview {
	c := makeChunkOverview(sortedKeys)
	chunkIndex := make(model.ChunkIndex)
	dataOffset := 0

	// fill up data section
	// NOTE: for now, chord instances have no ordering
	dataBuf := new(bytes.Buffer)
	for i, key := range sortedKeys {
		chords := m[key]
		chord.RankSortChords(chords)
		p := chunkIndex[key]
		p.Start = uint32(dataOffset)
		chunkIndex[key] = p
		// set End on the previous sortedKey in chunkIndex
		if i > 0 {
			pp := chunkIndex[sortedKeys[i-1]]
			pp.End = uint32(dataOffset)
			chunkIndex[sortedKeys[i-1]] = pp
		}
		for _, chord := range chords {
			binary.Write(dataBuf, binary.LittleEndian, chord.AbsTickOffset)
			binary.Write(dataBuf, binary.LittleEndian, chord.FileNum)
			dataOffset += 8
		}
	}

	// set End on the last sortedKey in chunkIndex
	p := chunkIndex[sortedKeys[len(sortedKeys)-1]]
	p.End = uint32(dataOffset)
	chunkIndex[sortedKeys[len(sortedKeys)-1]] = p

	// encode index into buffer
	indexBuf := new(bytes.Buffer)
	encoder := gob.NewEncoder(indexBuf)
	err := encoder.Encode(chunkIndex)
	if err != nil {
		panic("error making chunk, couldn't encode to get size: " + err.Error())
	}

	// encode size to a buffer
	sizeBuf := new(bytes.Buffer)
	indexSize := getEncodedMapSize(chunkIndex)
	binary.Write(sizeBuf, binary.LittleEndian, indexSize)

	// combine everything together
	var finalBytes []byte
	finalBytes = append(finalBytes, sizeBuf.Bytes()...)
	finalBytes = append(finalBytes, indexBuf.Bytes()...)
	finalBytes = append(finalBytes, dataBuf.Bytes()...)

	// save as a file
	filename := filepath.Join(util.GetIndexDir(), c.Filename)
	err = ioutil.WriteFile(filename, finalBytes, 0777)
	if err != nil {
		panic("Write failed for chunk file: " + err.Error())
	}
	return c
}

func maybeMakeChunks(m ChordKeyToChords, force bool) []model.ChunkOverview {
	var size int
	var currKeys []string

	sortedKeys := getKeysSorted(m)
	var createdChunks []model.ChunkOverview

	for i, key := range sortedKeys {
		currKeys = append(currKeys, key)
		chords := m[key]

		// each chord will take up uint32 and uint32 == 8 bytes
		size += len(chords) * 8
		// each index will take up some vari length + uint32 == 28 bytes?
		// NOTE: note completely accurate because we're encoding a map when we write
		size += len(key) + 4

		isLast := len(sortedKeys)-1 == i
		if size > constants.PreferredChunkSize || (isLast && force) {
			chunk := makeChunk(m, currKeys)
			createdChunks = append(createdChunks, chunk)
			size = 0
			for _, cKey := range currKeys {
				delete(m, cKey)
			}
			currKeys = currKeys[:0]
		}
	}

	return createdChunks
}

func getBucketPaths() []string {
	outDir := util.GetIndexDir()
	files, err := ioutil.ReadDir(outDir)
	if err != nil {
		panic("Could not make chunks because out file not read:" + err.Error())
	}

	var res []string
	for _, file := range files {
		res = append(res, filepath.Join(outDir, file.Name()))
	}
	return res
}

func CreateAll() []model.ChunkOverview {
	m := make(ChordKeyToChords)
	var res []model.ChunkOverview

	buckets := getBucketPaths()
	for i, bucketPath := range buckets {
		fmt.Printf("Processing %v of %v buckets\n", i+1, len(buckets))
		for _, c := range bucket.ReadChords(bucketPath) {
			chordKey := chord.CreateChordKey(c.Notes)
			currChords := m[chordKey]
			currChords = append(currChords, c)
			m[chordKey] = currChords
		}

		// check at end of every bucket to see if we can make chunks
		// we have to make chunks on bucket boundaries
		// if last bucket, we have to make sure we make the rest...
		isLastBucket := len(buckets)-1 == i
		res = append(res, maybeMakeChunks(m, isLastBucket)...)
	}

	return res
}

func ReadIndexOrPanic(f *os.File) (model.ChunkIndex, uint32) {
	// reads the index and returns it, advancing file to end of index point

	buf := make([]byte, 4)
	_, err := io.ReadFull(f, buf)
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
	return index, indexLength
}
