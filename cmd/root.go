package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func bindToViper(flagName string, cmd *cobra.Command) {
	viper.BindPFlag(flagName, cmd.Flags().Lookup(flagName))
	viper.BindEnv(flagName)
}

var rootCmd = &cobra.Command{
	Use:   "t-race",
	Short: "Benchmarking tool for distributed tracing systems",
	Long:  `t-race is a distributed workload generator for distributed tracing systems.`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
