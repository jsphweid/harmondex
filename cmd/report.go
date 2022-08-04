package cmd

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"

	"github.com/jsphweid/harmondex/chunk"
	"github.com/jsphweid/harmondex/constants"
	"github.com/jsphweid/harmondex/util"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(reportCmd)
}

var reportCmd = &cobra.Command{
	Use:   "report",
	Short: "Creates a report",
	Long:  `Creates a report`,
	Run: func(cmd *cobra.Command, args []string) {
		report()
	},
}

type bucketsReport struct {
	numChords int64
	numFiles  int64
	numBytes  int64
}

type chunksReport struct {
	avgIndexPercent float32
	indexPercents   []float32
	chordsInIndexes []int64
	numFiles        int64
	numChords       int64
	totalBytes      int64
	dataBytes       int64
}

func analyzeBuckets() bucketsReport {
	var report bucketsReport

	files, err := ioutil.ReadDir(util.GetIndexDir())
	if err != nil {
		panic("Could not read dir because: " + err.Error())
	}

	r, _ := regexp.Compile(`^\d\d\d\.dat$`)
	for _, file := range files {
		filename := file.Name()
		if r.MatchString(filename) {
			report.numFiles += 1
			path := filepath.Join(util.GetIndexDir(), filename)
			f, err := os.Open(path)
			if err != nil {
				panic("Could not open file")
			}
			stats, _ := f.Stat()
			if err != nil {
				panic("Could not get file stats")
			}
			report.numBytes += stats.Size()
			report.numChords += (stats.Size() / constants.ChordSize)
		}
	}

	return report
}

func analyzeChunks() chunksReport {
	var report chunksReport
	files, err := ioutil.ReadDir(util.GetIndexDir())
	if err != nil {
		panic("Could not read dir because: " + err.Error())
	}

	r, _ := regexp.Compile("^[0-9a-fA-F]{8}-([0-9a-fA-F]{4}-){3}[0-9a-fA-F]{12}.dat$")
	for _, file := range files {
		filename := file.Name()
		if r.MatchString(filename) {
			report.numFiles += 1
			f := util.OpenFileOrPanic(filepath.Join(util.GetIndexDir(), filename))
			index, indexLength := chunk.ReadIndexOrPanic(f)

			// count chords
			var chordsInIndex int64
			keys := make([]string, 0, len(index))
			for k := range index {
				keys = append(keys, k)
			}
			for _, v := range keys {
				chordsInIndex += int64(index[v].End-index[v].Start) / 8
			}

			chordsInIndexes := report.chordsInIndexes
			chordsInIndexes = append(chordsInIndexes, chordsInIndex)
			report.chordsInIndexes = chordsInIndexes

			stats, _ := f.Stat()
			if err != nil {
				panic("Could not get file stats")
			}
			indexPercent := float32(indexLength+4) / float32(stats.Size())
			report.totalBytes += stats.Size()

			indexPercents := report.indexPercents
			indexPercents = append(indexPercents, indexPercent)
			report.indexPercents = indexPercents

			dataBytes := stats.Size() - int64(indexLength+4)
			report.dataBytes += dataBytes
			report.numChords += (dataBytes / 8)
			f.Close()
		}
	}
	avg := float32(report.totalBytes-report.dataBytes) / float32(report.totalBytes)
	report.avgIndexPercent = avg
	return report
}

func report() {
	// Analyze files in out/
	bucketsReport := analyzeBuckets()
	chunksReport := analyzeChunks()
	fmt.Printf("bucketsReport.numFiles: %v\n", bucketsReport.numFiles)
	fmt.Printf("chunksReport.numFiles: %v\n", chunksReport.numFiles)
	fmt.Printf("dataBytes is this many times more than bucketed size (should be less than 1) %v\n", float32(chunksReport.dataBytes)/float32(bucketsReport.numBytes))
	fmt.Printf("chunksReport.avgIndexPercent: %v\n", chunksReport.avgIndexPercent)
	fmt.Printf("chunksReport.chordsInIndexes: %v\n", chunksReport.chordsInIndexes)

	fmt.Printf("bucketsReport.numChords: %v\n", bucketsReport.numChords)
	numCalcedChords := util.Sum(chunksReport.chordsInIndexes)
	fmt.Printf("numCalcedChords from indexes: %v\n", numCalcedChords)

	fmt.Printf("bucketsReport.numBytes: %v\n", bucketsReport.numBytes)
	fmt.Printf("chunksReport.totalBytes: %v\n", chunksReport.totalBytes)
}
