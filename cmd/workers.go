package cmd

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/dominik-/t-race/worker"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var workersCmd = &cobra.Command{
	Use:   "workers",
	Short: "Starts a set of t-race workers.",
	Long:  `Starts a set of t-race workers with the same parameters (but increasing port numbers)`,
	Run:   StartWorkers,
}

func init() {
	rootCmd.AddCommand(workersCmd)
	cobra.OnInitialize(initViperConfigWorkers)
	workersCmd.Flags().StringVar(&workerCfgFile, "workerConfig", "worker", "Config file name. Can lbe YAML, JSON or TOML format.")
	workersCmd.Flags().IntVarP(&workerCount, "workerCount", "c", 10, "Number of workers to start.")
	workersCmd.Flags().IntVarP(&benchmarkPort, "benchmarkPort", "b", 7000, "Port for the grpc server to receive benchmark configs.")
	workersCmd.Flags().IntVarP(&servicePort, "servicePort", "p", 8000, "Port for the grpc server to act within a service dependency graph.")
	workersCmd.Flags().StringVar(&samplingType, "samplingType", "probabilistic", "Sampling strategy type to implement at the worker. Depends on tracer. For Jaeger: const, remote, probabilistic, ratelimiting, lowerbound")
	workersCmd.Flags().Float64Var(&samplingParam, "samplingParam", 0.1, "Parameter for sampling type. Depends on type.")
	workersCmd.Flags().IntVarP(&metricsPort, "metricsPort", "m", 9000, "Port for the endpoint to scrape prometheus metrics from. The default /metrics path is used.")
	workersCmd.Flags().BoolVar(&exportMetrics, "exportMetrics", true, "Whether to collect prometheus metrics or not.")
	viper.SetEnvPrefix("workers")
	viper.AutomaticEnv()
	bindToViper("workerCount", workersCmd)
	bindToViper("benchmarkPort", workersCmd)
	bindToViper("servicePort", workersCmd)
	bindToViper("samplingType", workersCmd)
	bindToViper("samplingParam", workersCmd)
	bindToViper("metricsPort", workersCmd)
	bindToViper("exportMetrics", workersCmd)
}

var (
	workerCount int
)

func StartWorkers(cmd *cobra.Command, args []string) {
	sigTermRecv := make(chan os.Signal, 1)
	signal.Notify(sigTermRecv, syscall.SIGINT, syscall.SIGTERM)
	shutdownHooks := make([]chan bool, workerCount)
	fmt.Printf("Starting %d workers...\n", workerCount)
	for i := 0; i < workerCount; i++ {
		shutdownHooks[i] = worker.StartWorkerProcess(benchmarkPort+i, servicePort+i, metricsPort+i, exportMetrics, samplingType, samplingParam)
	}
	//wait for external signal to shut down
	fmt.Println("All workers running. Waiting for user interrupt.")
	<-sigTermRecv
	fmt.Println("Shutting down...")
	for _, hook := range shutdownHooks {
		hook <- true
	}
	<-time.NewTimer(time.Second * 3).C
	os.Exit(0)
}

func initViperConfigWorkers() {
	configFileDir, configFileName := filepath.Split(workerCfgFile)
	fileNameNoExt := configFileName[:len(configFileName)-len(filepath.Ext(configFileName))]
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
	workerCount = viper.GetInt("workerCount")
	benchmarkPort = viper.GetInt("benchmarkPort")
	servicePort = viper.GetInt("servicePort")
	samplingType = viper.GetString("samplingType")
	samplingParam = viper.GetFloat64("samplingParam")
	metricsPort = viper.GetInt("prometheusPort")
	exportMetrics = viper.GetBool("exportPrometheus")
}
