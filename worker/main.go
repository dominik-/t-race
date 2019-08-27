package worker

import (
	"fmt"
	"io"
	"log"
	"net"
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"gitlab.tubit.tu-berlin.de/dominik-ernst/tracer-benchmarks/api"
	"google.golang.org/grpc"
)

func StartWorkerProcess(benchmarkPort, servicePort, prometheusPort int, exportPrometheus bool, samplingType string, samplingParam float64) chan bool {
	listenerBenchmark, err := net.Listen("tcp", fmt.Sprintf(":%d", benchmarkPort))
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
	reporters := make([]ResultReporter, 1)
	reporters[0] = NewBufferingReporter(50)
	//we add an empty worker, except for reporters; everything else is configured once the worker receives a benchmark configuration
	api.RegisterBenchmarkWorkerServer(server, &Worker{
		Reporters:        reporters,
		ServicePort:      servicePort,
		SamplingStrategy: samplingType,
		SamplingParams:   []float64{samplingParam},
	})
	go server.Serve(listenerBenchmark)
	//wait for external signal to shut down
	shutdownHook := make(chan bool, 1)
	go waitForShutdown(shutdownHook, server, listenerBenchmark)
	return shutdownHook
}

func waitForShutdown(hook <-chan bool, server *grpc.Server, closeables ...io.Closer) {
	<-hook
	server.Stop()
	for _, closer := range closeables {
		closer.Close()
	}
}
