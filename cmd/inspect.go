package cmd

import (
	"fmt"

	"github.com/jsphweid/harmondex/chunk"
	"github.com/jsphweid/harmondex/util"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(inspectCmd)
}

var inspectCmd = &cobra.Command{
	Use:   "inspect",
	Short: "Inspects a chunk",
	Long:  `Inspects a chunk`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) != 1 {
			panic("Need 1 arg...")
		}
		inspect(args[0])
	},
}

func inspect(path string) {
	f := util.OpenFileOrPanic(path)
	index, _ := chunk.ReadIndexOrPanic(f)
	keys := util.GetKeys(index)
	for _, key := range keys {
		val, _ := index[key]
		fmt.Printf("key: %v\n", key)
		fmt.Printf("val: %v\n", val)
	}
}
