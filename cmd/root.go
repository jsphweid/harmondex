package cmd

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "mir1",
	Short: "MIR project 1",
	Long:  `MIR project 1... I'll come up with a better name later.`,
}

func Execute() {
	cobra.CheckErr(rootCmd.Execute())
}
