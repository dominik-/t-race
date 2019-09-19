package worker

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/opentracing/opentracing-go"
	"github.com/prometheus/client_golang/prometheus"
	"gitlab.tubit.tu-berlin.de/dominik-ernst/tracer-benchmarks/api"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
)

type Worker struct {
	Generator        SpanGenerator
	Reporters        []ResultReporter
	SpanDurationHist prometheus.Histogram
	Config           *api.WorkerConfiguration
	ServicePort      int
	SamplingStrategy string
	SamplingParams   []float64
	SetupDone        bool
}

func (w *Worker) GetTracer() opentracing.Tracer {
	return w.Generator.GetTracer()
}

func (w *Worker) StartWorker(config *api.WorkerConfiguration, stream api.BenchmarkWorker_StartWorkerServer) error {
	//need to do the run here, i.e. start Writer with generator and return results
	//hook to SIGINT/SIGTERM
	sigTermRecv := make(chan os.Signal, 1)
	signal.Notify(sigTermRecv, syscall.SIGINT, syscall.SIGTERM)
	//TODO differentiate between request generation and simply starting a listener/server for incoming requests.
	stopChan := make(chan bool, 1)
	defer close(stopChan)

	//Create sink (i.e. tracing backend) connection
	tracer, closer, err := InitTracer(config.SinkHostPort, config.OperationName, w.SamplingStrategy, w.SamplingParams[0])
	// we can't go on if this didnt work
	if err != nil {
		log.Fatalf("Couldn't create tracer with given config. Error was: %v", err)
	}
	defer closer.Close()
	//Setup for prometheus metrics
	if !w.SetupDone {
		w.SpanDurationHist = prometheus.NewHistogram(prometheus.HistogramOpts{
			Namespace: "worker",
			//Subsystem: config.OperationName,
			Name:    "span_duration",
			Help:    "A Histogram of Span durations",
			Buckets: []float64{10000.0, 20000.0, 50000.0, 100000.0, 200000.0},
		})
		prometheus.MustRegister(w.SpanDurationHist)
		w.SetupDone = true
	}

	doneChannel := make(chan bool, 1)
	w.Generator = NewOpenTracingSpanGenerator(tracer, config, w.SpanDurationHist)
	if config.Root {
		//We create a new done channel here if this is a root service
		//TODO: Ideally, the "default" case for the done channel link would be that the worker didn't receive "Call"-Requests for a few seconds,
		//indicating that the root-workers no longer send requests;
		doneChannel = w.Generator.WriteSpansUntilExitSignal(stopChan, calculateIntervalByThroughput(config.TargetThroughput), w.Reporters...)
	}

	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", w.ServicePort))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	server := grpc.NewServer(
		grpc.KeepaliveParams(keepalive.ServerParameters{
			MaxConnectionIdle: 30 * time.Minute,
		}),
	)
	api.RegisterBenchmarkWorkerServer(server, w)
	//start server in separate goroutine because we need to look at benchmark runtime here.
	go server.Serve(listener)
	//defer server.Stop()
	//TODO: make tolerance time (time after which worker is shut down forcefully after runtime) a parameter of the benchmark
	tolerance := 8 * time.Second
	log.Printf("Started worker. Config: %v\n", config)
	//create a timer with runtime
	timer := time.NewTimer(time.Duration(config.RuntimeSeconds) * time.Second)
	//intervals in which results are being pushed back to the coordinator
	//TODO: make ticker interval configurable for benchmark
	reportTicker := time.NewTicker(5 * time.Second)
WorkerLoop:
	for {
		select {
		case <-timer.C:
			stopChan <- true
			limit := time.NewTimer(tolerance)
			for {
				select {
				case <-doneChannel:
					log.Println("Benchmark ended regularly after runtime.")
					break WorkerLoop
				case <-limit.C:
					if config.Root {
						log.Printf("Benchmark shut down forcefully after runtime plus tolerance of %v.\n", tolerance)
					} else {
						log.Printf("Benchmark shut down after runtime plus tolerance of %v.\n", tolerance)
					}

					break WorkerLoop
				}
			}
		case <-sigTermRecv:
			stopChan <- true
			limit := time.NewTimer(5 * time.Second)
			for {
				select {
				case <-doneChannel:
					log.Println("Benchmark ended by manual interrupt from user.")
					break WorkerLoop
				case <-limit.C:
					log.Println("Benchmark shut down forcefully after manual interrupt from user.")
					break WorkerLoop
				}
			}
		case <-reportTicker.C:
			reportAllReporters(w.Reporters, stream)
		}
	}
	//important: do final reporting after benchmark ends. We give 5 seconds tolerance to make sure all results are reported.
	<-time.NewTimer(5 * time.Second).C
	reportAllReporters(w.Reporters, stream)
	server.GracefulStop()
	return nil
}

func reportAllReporters(reporters []ResultReporter, stream api.BenchmarkWorker_StartWorkerServer) {
	for _, reporter := range reporters {
		reporter.Report(stream)
	}
}

func calculateIntervalByThroughput(targetThroughput int64) time.Duration {
	if targetThroughput > 1000000 {
		targetThroughput = 0
	}
	if targetThroughput < 0 {
		targetThroughput = 0
	}
	if targetThroughput == 0 {
		log.Println("Target throughput of 0 or above maximum (1mio). Setting throughput to 1mio reqs/s.")
		return 1 * time.Microsecond
	}
	return time.Duration(1000000/targetThroughput) * time.Microsecond
}

func (w *Worker) Call(ctx context.Context, in *api.Empty) (*api.Empty, error) {
	//Delegate to to generator for all successor remote calls.
	w.Generator.DoUnitCalls(ctx, w.Reporters)
	return &api.Empty{}, nil
}
