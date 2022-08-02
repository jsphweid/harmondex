package cmd

import (
	"strconv"

	"github.com/jsphweid/harmondex/bucket"
	"github.com/jsphweid/harmondex/chunk"
	"github.com/jsphweid/harmondex/constants"
	"github.com/jsphweid/harmondex/file"
	"github.com/jsphweid/harmondex/util"
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
		if len(args) < 1 {
			panic("Need at least 1 arg... the path to the content folder")
		}

		var maxNum int
		if len(args) == 2 {
			arg2, err := strconv.Atoi(args[1])
			if err != nil {
				panic(err)
			}
			maxNum = arg2
		}

		run(args[0], maxNum)
	},
}

func run(contentPath string, maxNum int) {
	util.RecreateOutputDir()
	paths := util.GatherAllMidiPaths(contentPath, maxNum)
	fileNumMap := file.CreateFileNumMap(paths)
	bucket.ProcessAllMidiFiles(fileNumMap)
	chunks := chunk.CreateAll()
	util.CreateBinary(constants.OutDir+"/allChunks.dat", chunks)
	util.CreateBinary(constants.OutDir+"/indexToPath.dat", fileNumMap)
	// bucket.DeleteAll()
}
