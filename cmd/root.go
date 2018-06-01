package cmd

import (
	"fmt"
	"log"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gitlab.tubit.tu-berlin.de/dominik-ernst/tracer-benchmarks/benchmark"
)

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "tbench", "Config file name. Can be YAML, JSON or TOML format.")
	rootCmd.PersistentFlags().StringVarP(&deploymentFile, "deployment", "d", "components.yaml", "Component descriptor file name. Must be a YAML file.")
	rootCmd.PersistentFlags().Int64VarP(&runtime, "runtime", "r", 60, "The runtime of the benchmark in seconds.")
	rootCmd.PersistentFlags().Int64VarP(&baseThroughput, "baselineTP", "t", 100, "The target throughput per second, that arrives at the root component.")
	rootCmd.PersistentFlags().StringVarP(&workerPrefix, "workerPrefix", "p", "Worker", "Prefix for worker threads writing traces.")
	rootCmd.PersistentFlags().StringSliceVarP(&workers, "workers", "w", []string{""}, "Comma-separated list of worker addresses. For manual benchmark setups.")
	//rootCmd.PersistentFlags().StringVar(&resultDirPrefix, "resultDirPrefix", "results-", "Prefix for the directory, to which results are written. Defaults to \"results-\". The start time is always appended.")
	//configuration by file is not yet working - need to overwrite cobra flag defaults with config file apparently??
	bindToViper("workers", rootCmd)
	bindToViper("runtime", rootCmd)
	bindToViper("delay", rootCmd)
	bindToViper("baseThroughput", rootCmd)
	bindToViper("workerPrefix", rootCmd)
}

var (
	cfgFile         string
	deploymentFile  string
	baseThroughput  int64
	runtime         int64
	workerPrefix    string
	resultDirPrefix string
	workers         []string
)

func bindToViper(flagName string, cmd *cobra.Command) {
	viper.BindPFlag(flagName, cmd.PersistentFlags().Lookup(flagName))
}

var rootCmd = &cobra.Command{
	Use:   "tracerbench",
	Short: "Benchmarking tool for distributed tracing systems",
	Long:  `Coordinator component for TRace, a benchmarking tool for distributed tracing systems.`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	ExecuteBenchmark()
}

func ExecuteBenchmark() {
	config := &benchmark.BenchmarkConfig{
		Throughput:      baseThroughput,
		Runtime:         runtime,
		WorkerPrefix:    workerPrefix,
		ResultDirPrefix: resultDirPrefix,
	}
	component, err := benchmark.ParseComponentDescription(deploymentFile)
	if err != nil {
		log.Fatalf("Parsing of component deployment description failed.")
	}
	workers := benchmark.AllocateWorkers(component, workers)
	benchmark.SetupConnections(workers)
	benchmark.StartBenchmark(workers, config)
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
