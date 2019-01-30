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
	rootCmd.PersistentFlags().StringP("services", "s", "components.yaml", "Component descriptor file name. Must be a YAML file.")
	rootCmd.PersistentFlags().Int64P("runtime", "r", 60, "The runtime of the benchmark in seconds.")
	rootCmd.PersistentFlags().Int64P("baselineTP", "t", 10, "The target throughput per second, that arrives at the root component.")
	rootCmd.PersistentFlags().String("resultDirPrefix", "results-", "Prefix for the directory, to which results are written. Defaults to \"results-\". The start time is always appended.")
	rootCmd.PersistentFlags().IntSlice("sinks", []int{}, "")
	rootCmd.PersistentFlags().IntSlice("workers", []int{}, "")
	bindToViper("services", rootCmd)
	bindToViper("runtime", rootCmd)
	bindToViper("baselineTP", rootCmd)
	bindToViper("resultDirPrefix", rootCmd)
	bindToViper("sinks", rootCmd)
	bindToViper("workers", rootCmd)
}

var (
	cfgFile         string
	sinks           []int
	workers         []int
	serviceFile     string
	baseThroughput  int64
	runtime         int64
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
		ResultDirPrefix: resultDirPrefix,
	}
	deployment, err := benchmark.ParseDeploymentDescription(serviceFile)
	if err != nil {
		log.Fatalf("Parsing of component deployment description failed.")
	}
	prov := provider.NewLocalStaticProvider(workers, sinks)
	prov.CreateEnvironments(deployment.Environments)
	prov.AllocateServices(deployment.Services)
	prov.AllocateSinks(deployment.Sinks)

	b := benchmark.Setup(deployment, prov.SvcMap, prov.SinkMap, config)
	b.StartBenchmark()
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
	serviceFile = viper.GetString("services")
	baseThroughput = viper.GetInt64("baselineTP")
	runtime = viper.GetInt64("runtime")
	resultDirPrefix = viper.GetString("resultDirPrefix")
	if s, castable := viper.Get("sinks").([]int); !castable {
		sinks = []int{}
	} else {
		sinks = s
	}
	if s, castable := viper.Get("workers").([]int); !castable {
		workers = []int{}
	} else {
		workers = s
	}
}
