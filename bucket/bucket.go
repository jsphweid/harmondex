package bucket

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"sort"

	"github.com/jsphweid/harmondex/chord"
	"github.com/jsphweid/harmondex/constants"
	"github.com/jsphweid/harmondex/db"
	"github.com/jsphweid/harmondex/midi"
	"github.com/jsphweid/harmondex/model"
	"github.com/jsphweid/harmondex/util"
)

func maybePutChordInBuckets(c model.Chord) {
	// TODO: bucketize other methods? 1. transposed, 2. note classes

	// ignore really short or really long chords
	if len(c.Notes) < 2 || len(c.Notes) > 16 {
		return
	}

	// order them
	sort.Slice(c.Notes, func(i, j int) bool {
		return c.Notes[i] < c.Notes[j]
	})

	bytes := chord.Serialize(c)

	filename := fmt.Sprintf("%v/%03d.dat", constants.GetIndexDir(), c.Notes[0])
	f, err := os.OpenFile(filename, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0777)
	if err != nil {
		panic("Could not open bucket because: " + err.Error())
	}
	defer f.Close()

	if _, err = f.Write(bytes[:]); err != nil {
		panic("Could not write chord to bucket because: " + err.Error())
	}
}

func fileHasMetadata(filename string) bool {
	metadatas := db.GetMidiMetadatas([]string{filename})
	if _, ok := metadatas[filename]; ok {
		return true
	}
	return false
}

func processMidiFile(fileNum uint32, filename string) {
	path := filepath.Join(constants.GetMediaDir(), filename)
	parsed, err := midi.ReadMidiFile(path)
	if err != nil {
		fmt.Printf("Skipping %v because: %v\n", filename, err)
		return
	}

	hasMetadata := fileHasMetadata(filename)
	chords, err := chord.GetChords(parsed, hasMetadata)
	if err != nil {
		fmt.Printf("Skipping %v because: %v\n", filename, err)
		return
	}

	for _, chord := range chords {
		chord.FileNum = uint32(fileNum)
		maybePutChordInBuckets(chord)
	}
}

func ProcessAllMidiFiles(m model.FileNumToMidiPath) {
	keys := util.GetKeys(m)
	for i, num := range keys {
		fmt.Printf("Processing %v of %v midi files\n", i+1, len(keys))
		processMidiFile(num, m[num])
	}
}

func DeleteAll() {
	outDir := constants.GetIndexDir()
	files, err := ioutil.ReadDir(outDir)
	if err != nil {
		panic("Could not read dir because: " + err.Error())
	}

	r, _ := regexp.Compile(`^\d\d\d\.dat$`)
	for _, file := range files {
		filename := file.Name()
		if r.MatchString(filename) {
			os.Remove(filepath.Join(outDir, filename))
		}
	}
}

func ReadChords(path string) []model.Chord {
	var res []model.Chord
	bucketFile := util.OpenFileOrPanic(path)
	bucketReader := bufio.NewReader(bucketFile)
	for {
		buf := make([]byte, constants.ChordSize)
		_, err := io.ReadFull(bucketReader, buf)
		if err == io.EOF {
			break
		}
		if err != nil {
			panic("Could not read chord from file: " + err.Error())
		}

		buf = buf[:constants.ChordSize]
		c := chord.Deserialize(buf)
		res = append(res, c)
	}
	return res
}
