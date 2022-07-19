package cmd

import (
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
		if len(args) != 1 {
			panic("Need at least 1 arg... the path to the content folder")
		}

		run(args[0])
	},
}

func run(contentPath string) {
	util.RecreateOutputDir()
	paths := util.GatherAllMidiPaths(contentPath)
	fileNumMap := file.CreateFileNumMap(paths[:10000]) // NOTE: temp
	bucket.ProcessAllMidiFiles(fileNumMap)
	chunks := chunk.CreateAll()
	util.CreateBinary(constants.OutDir+"/allChunks.dat", chunks)
	util.CreateBinary(constants.OutDir+"/indexToPath.dat", fileNumMap)
	// bucket.DeleteAll()
}
