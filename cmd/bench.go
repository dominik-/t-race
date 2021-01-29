package cmd

import (
	"encoding/json"
	"log"
	"path/filepath"

	"github.com/dominik-/t-race/benchmark"
	"github.com/dominik-/t-race/executionmodel"
	"github.com/dominik-/t-race/provider"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var benchCmd = &cobra.Command{
	Use:   "bench",
	Short: "Starts a t-race benchmark run.",
	Long:  `Starts a t-race benchmark run with given parameters. Requires existing deployment and service files.`,
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
	benchCmd.Flags().StringVar(&cfgFile, "config", "t-race", "Config file name. Can be YAML, JSON or TOML format.")
	benchCmd.Flags().StringP("services", "s", "services.yaml", "Service descriptor file name. Must be a YAML file.")
	benchCmd.Flags().Int64P("runtime", "r", 60, "The runtime of the benchmark in seconds.")
	benchCmd.Flags().Int64P("baselineTP", "t", 10, "The target throughput per second, that arrives at the root component.")
	benchCmd.Flags().String("resultDirPrefix", "results-", "Prefix for the directory, to which results are written. Defaults to \"results-\". The start time is always appended.")
	benchCmd.Flags().StringP("deploymentFile", "d", "deployment.json", "File that contains a static deployment of workers and sinks.")
	bindToViper("services", benchCmd)
	bindToViper("runtime", benchCmd)
	bindToViper("baselineTP", benchCmd)
	bindToViper("resultDirPrefix", benchCmd)
	bindToViper("deploymentFile", benchCmd)
}

func ExecuteBenchmark(cmd *cobra.Command, args []string) {
	config := &executionmodel.BenchmarkConfig{
		Throughput:      baseThroughput,
		Runtime:         runtime,
		ResultDirPrefix: resultDirPrefix,
	}
	architecture, err := executionmodel.ParseArchitectureDescription(serviceFile)
	if err != nil {
		log.Fatalf("Parsing of service descriptor file failed: %v", err)
	}
	log.Println("Parsed service descriptions successfully.")
	log.Printf("Architecture description is: %+v\n", architecture)
	s, _ := json.MarshalIndent(architecture, "", "\t")
	log.Println(string(s))
	prov, err := provider.NewStaticProvider(deploymentFile)
	if err != nil {
		log.Fatalf("Error parsing the deployment file: %v", err)
	}
	log.Println("Parsed static deployment successfully.")
	prov.CreateEnvironments(architecture.Environments)
	prov.AllocateServices(architecture.Services)
	prov.AllocateSinks(architecture.Sinks)

	log.Printf("Architecture with allocated provider resources: %v\n", architecture)
	s, _ = json.MarshalIndent(architecture, "", "\t")
	log.Println(string(s))
	b := benchmark.Setup(architecture, prov.SvcMap, prov.WorkerMap, prov.SinkMap, config)
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
