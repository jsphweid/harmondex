package cmd

import (
	"strconv"

	"github.com/jsphweid/harmondex/bucket"
	"github.com/jsphweid/harmondex/chunk"
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
		var maxNum int
		if len(args) == 1 {
			arg1, err := strconv.Atoi(args[0])
			if err != nil {
				panic(err)
			}
			maxNum = arg1
		}

		run(maxNum)
	},
}

func run(maxNum int) {
	util.RecreateOutputDir()
	paths := util.GatherAllMidiPaths(maxNum)
	fileNumMap := file.CreateFileNumMap(paths)
	bucket.ProcessAllMidiFiles(fileNumMap)
	chunks := chunk.CreateAll()
	util.CreateBinary(util.GetAllChunksPath(), chunks)
	util.CreateBinary(util.GetFileNumToNamePath(), fileNumMap)
	// bucket.DeleteAll()
}
