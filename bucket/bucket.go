package bucket

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"regexp"
	"sort"

	"github.com/jsphweid/harmondex/chord"
	"github.com/jsphweid/harmondex/constants"
	"github.com/jsphweid/harmondex/midi"
	"github.com/jsphweid/harmondex/model"
	"github.com/jsphweid/harmondex/util"
)

func maybePutChordInBuckets(chord model.Chord) {
	// TODO: bucketize other methods? 1. transposed, 2. note classes

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

	filename := fmt.Sprintf("%v/%03d.dat", constants.OutDir, notes[0])
	f, err := os.OpenFile(filename, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0777)
	if err != nil {
		panic("Could not open bucket because: " + err.Error())
	}
	defer f.Close()

	// TODO: create easy mechanism for reading/writing chord
	var bytes [constants.ChordSize]byte
	copy(bytes[:], notes[:])
	binary.LittleEndian.PutUint32(bytes[16:20], chord.Offset)
	binary.LittleEndian.PutUint32(bytes[20:24], chord.FileNum)
	if _, err = f.Write(bytes[:]); err != nil {
		panic("Could not write chord to bucket because: " + err.Error())
	}
}

func processMidiFile(fileNum uint32, path string) {
	parsed, err := midi.ReadMidiFile(path)
	if err != nil {
		fmt.Printf("Skipping %v because: %v\n", path, err)
		return
	}

	chords, err := chord.GetChords(parsed)
	if err != nil {
		fmt.Printf("Skipping %v because: %v\n", path, err)
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
	files, err := ioutil.ReadDir(constants.OutDir)
	if err != nil {
		panic("Could not read dir because: " + err.Error())
	}

	r, _ := regexp.Compile(`^\d\d\d\.dat$`)
	for _, file := range files {
		filename := file.Name()
		if r.MatchString(filename) {
			os.Remove(constants.OutDir + "/" + filename)
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

		var c model.Chord
		c.Notes = util.FilterZeros(buf[:16])
		c.Offset = binary.LittleEndian.Uint32(buf[16:20])
		c.FileNum = binary.LittleEndian.Uint32(buf[20:24])
		res = append(res, c)
	}
	return res
}
