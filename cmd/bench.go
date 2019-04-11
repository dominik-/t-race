package cmd

import (
	"log"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gitlab.tubit.tu-berlin.de/dominik-ernst/tracer-benchmarks/benchmark"
	"gitlab.tubit.tu-berlin.de/dominik-ernst/tracer-benchmarks/provider"
)

var benchCmd = &cobra.Command{
	Use:   "bench",
	Short: "Starts a t-race benchmark run.",
	Long:  `Starts a t-race benchmark run with given parameters. Requires an existing deployment file.`,
	Run:   ExecuteBenchmark,
}

var (
	cfgFile         string
	serviceFile     string
	baseThroughput  int64
	runtime         int64
	resultDirPrefix string
	deploymentFile  string
)

func init() {
	rootCmd.AddCommand(benchCmd)
	cobra.OnInitialize(initBenchmarkConfig)
	benchCmd.PersistentFlags().StringVar(&cfgFile, "config", "t-race", "Config file name. Can be YAML, JSON or TOML format.")
	benchCmd.PersistentFlags().StringP("services", "s", "services.yaml", "Service descriptor file name. Must be a YAML file.")
	benchCmd.PersistentFlags().Int64P("runtime", "r", 60, "The runtime of the benchmark in seconds.")
	benchCmd.PersistentFlags().Int64P("baselineTP", "t", 10, "The target throughput per second, that arrives at the root component.")
	benchCmd.PersistentFlags().String("resultDirPrefix", "results-", "Prefix for the directory, to which results are written. Defaults to \"results-\". The start time is always appended.")
	benchCmd.PersistentFlags().String("deploymentFile", "deployment.json", "File that contains a static deployment of workers and sinks.")
	bindToViper("services", benchCmd)
	bindToViper("runtime", benchCmd)
	bindToViper("baselineTP", benchCmd)
	bindToViper("resultDirPrefix", benchCmd)
	bindToViper("deploymentFile", benchCmd)
}

func ExecuteBenchmark(cmd *cobra.Command, args []string) {
	config := &benchmark.BenchmarkConfig{
		Throughput:      baseThroughput,
		Runtime:         runtime,
		ResultDirPrefix: resultDirPrefix,
	}
	deployment, err := benchmark.ParseDeploymentDescription(serviceFile)
	if err != nil {
		log.Fatalf("Parsing of service descriptor file failed: %v", err)
	}
	log.Println("Parsed service descriptions successfully.")
	prov, err := provider.NewStaticProvider(deploymentFile)
	if err != nil {
		log.Fatalf("Error parsing the deployment file: %v", err)
	}
	log.Println("Parsed static deployment successfully.")
	prov.CreateEnvironments(deployment.Environments)
	prov.AllocateServices(deployment.Services)
	prov.AllocateSinks(deployment.Sinks)

	b := benchmark.Setup(deployment, prov.SvcMap, prov.WorkerMap, prov.SinkMap, config)
	b.StartBenchmark()
}

func initBenchmarkConfig() {
	configFileDir, configFileName := filepath.Split(cfgFile)
	fileNameNoExt := configFileName[:len(configFileName)-len(filepath.Ext(configFileName))]
	//log.Printf("filename: %s, dirname: %s, noext: %s", configFileName, configFileDir, fileNameNoExt)
	viper.SetConfigName(fileNameNoExt)
	viper.AddConfigPath(configFileDir)
	viper.AddConfigPath(".")
	err := viper.ReadInConfig()
	if err != nil {
		serr, ok := err.(*viper.ConfigFileNotFoundError)
		if !ok {
			log.Printf("No config file: %v. Using command line params or defaults.", err)
		} else {
			log.Fatalf("Configuration error: %v", serr)
		}
	}
	serviceFile = viper.GetString("services")
	baseThroughput = viper.GetInt64("baselineTP")
	runtime = viper.GetInt64("runtime")
	resultDirPrefix = viper.GetString("resultDirPrefix")
	deploymentFile = viper.GetString("deploymentFile")
}
