package chunk

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"fmt"
	"io/ioutil"
	"sort"

	"github.com/google/uuid"
	"github.com/jsphweid/mir1/bucket"
	"github.com/jsphweid/mir1/chord"
	"github.com/jsphweid/mir1/constants"
	"github.com/jsphweid/mir1/model"
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
			binary.Write(dataBuf, binary.LittleEndian, chord.TicksOffset)
			binary.Write(dataBuf, binary.LittleEndian, chord.FileNum)
			dataOffset += 12
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
	filename := constants.OutDir + "/" + c.Filename
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

		// each chord will take up uint32 and uint64 == 12 bytes
		size += len(chords) * 12
		// each index will take up some vari length + uint32 == 28 bytes
		// NOTE: note completely accurate because we're encoding a map when we write
		size += len(key) + 4

		isLast := len(sortedKeys)-1 == i
		if size > constants.PreferredChunkSize || (isLast && force) {
			chunk := makeChunk(m, currKeys)
			createdChunks = append(createdChunks, chunk)
			size = 0
			currKeys = currKeys[:0]
		}
	}

	return createdChunks
}

func getBucketPaths() []string {
	files, err := ioutil.ReadDir(constants.OutDir)
	if err != nil {
		panic("Could not make chunks because out file not read:" + err.Error())
	}

	var res []string
	for _, file := range files {
		res = append(res, constants.OutDir+"/"+file.Name())
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
