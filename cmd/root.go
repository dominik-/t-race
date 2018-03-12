package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	//cobra.OnInitialize(initConfig)
	//rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "tbench.yaml", "config file name.")
	rootCmd.PersistentFlags().Int64VarP(&interval, "interval", "i", 100, "The interval between single ticks, by which each worker generates a trace. In milliseconds")
	rootCmd.PersistentFlags().IntVarP(&workers, "workers", "w", 10, "The number of workers to start")
	//rootCmd.PersistentFlags().StringVar(&resultDirPrefix, "resultDirPrefix", "results-", "Prefix for the directory, to which results are written. Defaults to \"results-\". The start time is always appended.")
}

var (
	cfgFile         string
	interval        int64
	workers         int
	resultDirPrefix string
)

var rootCmd = &cobra.Command{
	Use:   "tracerbench",
	Short: "Benchmarking tool for distributed tracing systems",
	Long:  `So good it hurts`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

//currently unused
func initConfig() {
	// Don't forget to read config either from cfgFile or from home directory!
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// we use the working directory of the binary
		dir, err := os.Getwd()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		viper.AddConfigPath(dir)
		viper.SetConfigName("tbench.yaml")
	}

	if err := viper.ReadInConfig(); err != nil {
		fmt.Println("Can't read config:", err)
		os.Exit(1)
	}
}
