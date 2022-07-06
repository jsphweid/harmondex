package cmd

import (
	"github.com/jsphweid/harmondex/bucket"
	"github.com/jsphweid/harmondex/chunk"
	"github.com/jsphweid/harmondex/constants"
	"github.com/jsphweid/harmondex/model"
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
		run()
	},
}

func createFileNumMap(paths []string) model.FileNumToMidiPath {
	res := make(model.FileNumToMidiPath)
	for i, v := range paths {
		res[uint32(i)] = v
	}
	return res
}

func run() {
	util.RecreateOutputDir()
	paths := util.GatherAllMidiPaths("lmd_full")
	fileNumMap := createFileNumMap(paths[:10000]) // NOTE: temp
	bucket.ProcessAllMidiFiles(fileNumMap)
	chunks := chunk.CreateAll()
	util.CreateBinary(constants.OutDir+"/allChunks.dat", chunks)
	util.CreateBinary(constants.OutDir+"/indexToPath.dat", fileNumMap)
	bucket.DeleteAll()
}
