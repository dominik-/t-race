package cmd

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gitlab.tubit.tu-berlin.de/dominik-ernst/tracer-benchmarks/benchmark"
	"gitlab.tubit.tu-berlin.de/dominik-ernst/tracer-benchmarks/provider"
)

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "t-race", "Config file name. Can be YAML, JSON or TOML format.")
	rootCmd.PersistentFlags().StringP("deployment", "d", "components.yaml", "Component descriptor file name. Must be a YAML file.")
	rootCmd.PersistentFlags().Int64P("runtime", "r", 60, "The runtime of the benchmark in seconds.")
	rootCmd.PersistentFlags().Int64P("baselineTP", "t", 10, "The target throughput per second, that arrives at the root component.")
	rootCmd.PersistentFlags().String("resultDirPrefix", "results-", "Prefix for the directory, to which results are written. Defaults to \"results-\". The start time is always appended.")
	bindToViper("deployment", rootCmd)
	bindToViper("runtime", rootCmd)
	bindToViper("baselineTP", rootCmd)
	bindToViper("resultDirPrefix", rootCmd)
}

var (
	cfgFile         string
	deploymentFile  string
	baseThroughput  int64
	runtime         int64
	workerPrefix    string
	resultDirPrefix string
)

func bindToViper(flagName string, cmd *cobra.Command) {
	viper.BindPFlag(flagName, cmd.PersistentFlags().Lookup(flagName))
	viper.BindEnv(flagName)
}

var rootCmd = &cobra.Command{
	Use:   "tracer-benchmarks",
	Short: "Benchmarking tool for distributed tracing systems",
	Long:  `Coordinator component for TRace, a benchmarking tool for distributed tracing systems.`,
	Run:   ExecuteBenchmark,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func ExecuteBenchmark(cmd *cobra.Command, args []string) {
	config := &benchmark.BenchmarkConfig{
		Throughput:      baseThroughput,
		Runtime:         runtime,
		WorkerPrefix:    workerPrefix,
		ResultDirPrefix: resultDirPrefix,
	}
	deployment, err := benchmark.ParseDeploymentDescription(deploymentFile)
	if err != nil {
		log.Fatalf("Parsing of component deployment description failed.")
	}
	workerPorts := []int{9001, 9002, 9003}
	sinkPorts := []int{9011}
	prov := provider.NewLocalStaticProvider(workerPorts, sinkPorts)
	prov.CreateEnvironments(deployment.Environments)
	sinkMap := prov.AllocateServices(deployment.Sinks)
	serviceMap := prov.AllocateSinks(deployment.Services)

	benchmark.SetupConnections(workers)
	benchmark.StartBenchmark(workers, config)
}

func initConfig() {
	configFileDir, configFileName := filepath.Split(cfgFile)
	fileNameNoExt := configFileName[:len(configFileName)-len(filepath.Ext(configFileName))]
	log.Printf("filename: %s, dirname: %s, noext: %s", configFileName, configFileDir, fileNameNoExt)
	viper.SetConfigName(fileNameNoExt)
	viper.AddConfigPath(configFileDir)
	viper.AddConfigPath(".")
	err := viper.ReadInConfig()
	if err != nil {
		serr, ok := err.(*viper.ConfigFileNotFoundError)
		if !ok {
			log.Printf("No config file: %v", err)
		} else {
			log.Fatalf("Configuration error: %v", serr)
		}
	}
	deploymentFile = viper.GetString("deployment")
	baseThroughput = viper.GetInt64("baselineTP")
	runtime = viper.GetInt64("runtime")
	workerPrefix = viper.GetString("workerPrefix")
	workers = viper.GetStringSlice("workers")
	resultDirPrefix = viper.GetString("resultDirPrefix")
}
