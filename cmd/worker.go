package cmd

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gitlab.tubit.tu-berlin.de/dominik-ernst/tracer-benchmarks/worker"
)

var workerCmd = &cobra.Command{
	Use:   "worker",
	Short: "Starts a new t-race worker.",
	Long:  `Starts a new t-race worker with the given parameters.`,
	Run:   StartWorker,
}

func init() {
	rootCmd.AddCommand(workerCmd)
	cobra.OnInitialize(initViperConfigWorker)
	workerCmd.Flags().StringVar(&workerCfgFile, "worker", "tbench-worker", "Config file name. Can be YAML, JSON or TOML format.")
	workerCmd.Flags().IntVarP(&benchmarkPort, "benchmarkPort", "b", 7000, "Port for the grpc server to receive benchmark configs.")
	workerCmd.Flags().IntVarP(&servicePort, "servicePort", "p", 8000, "Port for the grpc server to act within a service dependency graph.")
	workerCmd.Flags().StringVar(&samplingType, "samplingType", "probabilistic", "Sampling strategy type to implement at the worker. Depends on tracer. For Jaeger: const, remote, probabilistic, ratelimiting, lowerbound")
	workerCmd.Flags().Float64Var(&samplingParam, "samplingParam", 0.1, "Parameter for sampling type. Depends on type.")
	workerCmd.Flags().IntVar(&prometheusPort, "prometheusPort", 9000, "Port for the endpoint to scrape prometheus metrics from. The default /metrics path is used.")
	workerCmd.Flags().BoolVar(&exportPrometheus, "exportPrometheus", true, "Whether to collect prometheus metrics or not.")
	viper.SetEnvPrefix("worker")
	viper.AutomaticEnv()
	bindToViper("benchmarkPort", workerCmd)
	bindToViper("servicePort", workerCmd)
	bindToViper("samplingType", workerCmd)
	bindToViper("samplingParam", workerCmd)
	bindToViper("prometheusPort", workerCmd)
	bindToViper("exportPrometheus", workerCmd)
}

var (
	workerCfgFile    string
	benchmarkPort    int
	servicePort      int
	samplingType     string
	samplingParam    float64
	prometheusPort   int
	exportPrometheus bool
)

func StartWorker(cmd *cobra.Command, args []string) {
	sigTermRecv := make(chan os.Signal, 1)
	signal.Notify(sigTermRecv, syscall.SIGINT, syscall.SIGTERM)
	shutdown := worker.StartWorkerProcess(benchmarkPort, servicePort, prometheusPort, exportPrometheus, samplingType, samplingParam)
	//wait for external signal to shut down
	<-sigTermRecv
	shutdown <- true
	fmt.Println("Shutting down...")
	<-time.NewTimer(time.Second * 2).C
	os.Exit(0)
}

func initViperConfigWorker() {
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
	benchmarkPort = viper.GetInt("benchmarkPort")
	servicePort = viper.GetInt("servicePort")
	samplingType = viper.GetString("samplingType")
	samplingParam = viper.GetFloat64("samplingParam")
	prometheusPort = viper.GetInt("prometheusPort")
	exportPrometheus = viper.GetBool("exportPrometheus")
}
