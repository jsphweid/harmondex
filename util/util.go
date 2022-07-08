package util

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/jsphweid/harmondex/constants"
	"golang.org/x/exp/constraints"
)

func RecreateOutputDir() {
	path, err := os.Getwd()
	if err != nil {
		panic("Could not RecreateOutputDir: " + err.Error())
	}
	dir := path + "/" + constants.OutDir
	os.RemoveAll(dir)
	os.MkdirAll(dir, 0777)
}

func GatherAllMidiPaths(path string) []string {
	var res []string
	walk := func(s string, d fs.DirEntry, err error) error {
		if err != nil {
			panic("Error walking: " + err.Error())
		}
		if !d.IsDir() {
			if strings.HasSuffix(s, ".mid") || strings.HasSuffix(s, ".midi") {
				res = append(res, s)
			}
		}
		return nil
	}
	filepath.WalkDir(path, walk)
	return res
}

func GetKeys[A constraints.Ordered, B any](m map[A]B) []A {
	keys := make([]A, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func CreateBinary(filename string, data any) {
	fmt.Printf("Creating binary for filename: %v\n", filename)
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

func ReadFileOrPanic(path string) *os.File {
	f, err := os.Open(path)
	if err != nil {
		panic("Couldn't read big file: " + err.Error())
	}
	return f
}

func FilterZeros[A constraints.Integer](nums []A) []A {
	var res []A
	for _, v := range nums {
		if v != 0 {
			res = append(res, v)
		}
	}
	return res
}

func ReadBinaryOrPanic[A any](path string) A {
	f, err := os.Open(path)
	if err != nil {
		panic("Could not load binary file: " + err.Error())
	}
	defer f.Close()

	var b []byte
	_, err = f.Read(b)
	if err != nil {
		panic("Could not read binary file: " + err.Error())
	}

	var data A
	decoder := gob.NewDecoder(f)
	err = decoder.Decode(&data)
	if err != nil {
		panic("Could not decode binary file: " + err.Error())
	}

	return data
}

func Min[A constraints.Integer](num1 A, num2 A) A {
	if num1 > num2 {
		return num2
	}
	return num1
}
