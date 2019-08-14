package cmd

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gitlab.tubit.tu-berlin.de/dominik-ernst/tracer-benchmarks/api"
	"gitlab.tubit.tu-berlin.de/dominik-ernst/tracer-benchmarks/worker"
	"google.golang.org/grpc"
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
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", benchmarkPort))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	if exportPrometheus {
		//TODO this listener is never closed so far
		listenerHTTPPrometheus, err := net.Listen("tcp", fmt.Sprintf(":%d", prometheusPort))
		if err != nil {
			log.Fatalf("failed to listen: %v", err)
		}
		http.Handle("/metrics", promhttp.Handler())
		go http.Serve(listenerHTTPPrometheus, nil)
	}
	/* // Read cert and key file
	backendCert, err := ioutil.ReadFile("/certs/tls.crt")
	if err != nil {
		log.Fatalf("Couldn't read or parse certfile: %v", err)
	}
	backendKey, err := ioutil.ReadFile("/certs/tls.key")
	if err != nil {
		log.Fatalf("Couldn't read or parse keyfile: %v", err)
	}
	// Generate Certificate struct
	cert, err := tls.X509KeyPair(backendCert, backendKey)
	if err != nil {
		log.Fatalf("failed to parse certificate: %v", err)
	}
	// Create credentials
	creds := credentials.NewServerTLSFromCert(&cert)
	// Use Credentials in gRPC server options
	serverOption := grpc.Creds(creds) */
	server := grpc.NewServer()
	reporters := make([]worker.ResultReporter, 1)
	reporters[0] = worker.NewBufferingReporter(50)
	//we add an empty worker, except for reporters; everything else is configured once the worker receives a benchmark configuration
	api.RegisterBenchmarkWorkerServer(server, &worker.Worker{
		Reporters:        reporters,
		ServicePort:      servicePort,
		SamplingStrategy: samplingType,
		SamplingParams:   []float64{samplingParam},
	})
	sigTermRecv := make(chan os.Signal, 1)
	signal.Notify(sigTermRecv, syscall.SIGINT, syscall.SIGTERM)
	go server.Serve(listener)
	//wait for external signal to shut down
	<-sigTermRecv
	server.Stop()
	listener.Close()
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
