package cmd

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gitlab.tubit.tu-berlin.de/dominik-ernst/tracer-benchmarks/model"
)

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "tbench", "Config file name. Can be YAML, JSON or TOML format.")
	rootCmd.PersistentFlags().IntVarP(&workers, "workers", "w", 1, "The number of workers to start")
	rootCmd.PersistentFlags().DurationVarP(&runtime, "runtime", "r", 1*time.Minute, "The runtime of each worker.")
	rootCmd.PersistentFlags().DurationVarP(&delay, "delay", "d", 1000*time.Microsecond, "The delay that is used between opening and closing a span.")
	rootCmd.PersistentFlags().DurationVarP(&interval, "interval", "i", 10*time.Millisecond, "The interval between single ticks, by which each worker generates a trace")
	rootCmd.PersistentFlags().StringVarP(&workerPrefix, "workerPrefix", "p", "Worker", "Prefix for worker threads writing traces.")
	//rootCmd.PersistentFlags().StringVar(&resultDirPrefix, "resultDirPrefix", "results-", "Prefix for the directory, to which results are written. Defaults to \"results-\". The start time is always appended.")
	//configuration by file is not yet working - need to overwrite cobra flag defaults with config file apparently??
	bindToViper("workers", rootCmd)
	bindToViper("runtime", rootCmd)
	bindToViper("delay", rootCmd)
	bindToViper("interval", rootCmd)
	bindToViper("workerPrefix", rootCmd)
}

var (
	cfgFile         string
	interval        time.Duration
	workers         int
	runtime         time.Duration
	delay           time.Duration
	workerPrefix    string
	resultDirPrefix string
)

func bindToViper(flagName string, cmd *cobra.Command) {
	viper.BindPFlag(flagName, cmd.PersistentFlags().Lookup(flagName))
}

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
	ExecuteBenchmark()
}

func ExecuteBenchmark() {
	config := &model.BenchmarkConfig{
		Interval:        interval,
		Runtime:         runtime,
		Workers:         workers,
		WorkerPrefix:    workerPrefix,
		ResultDirPrefix: resultDirPrefix,
	}
	generator := &model.ConstantSpanGenerator{
		Counter:       0,
		Delay:         100,
		OperationName: "benchmark",
	}

	benchmark := model.NewBenchmark(config, generator)
	benchmark.RunBenchmark()

}

func initConfig() {
	viper.SetConfigName(cfgFile)
	viper.AddConfigPath("$HOME/.appname")
	viper.AddConfigPath(".")
	err := viper.ReadInConfig()
	if err != nil {
		serr, ok := err.(*viper.ConfigFileNotFoundError)
		if !ok {
			log.Printf("Config file Error: %v", err)
		} else {
			log.Fatalf("Configuration error: %v", serr)
		}
	}
}
