package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(versionCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "t-race version info.",
	Long:  `Version of t-race.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("t-race benchmarking tool v0.9.1")
	},
}
