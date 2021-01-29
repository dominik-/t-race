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

	"github.com/dominik-/t-race/api"
	"github.com/opentracing/opentracing-go"
	"github.com/prometheus/client_golang/prometheus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
)

type Worker struct {
	Generator        SpanGenerator
	Reporter         ResultReporter
	SpanDurationHist prometheus.Histogram
	Config           *api.WorkerConfiguration
	ServicePort      int
	SamplingStrategy string
	SamplingParams   []float64
	SetupDone        bool
	MetricsRegistry  prometheus.Registerer
	UnitExecutorMap  map[string]Unit
	api.UnimplementedBenchmarkWorkerServer
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
	tracer, closer, err := InitTracer(config.SinkHostPort, config.ServiceName, w.SamplingStrategy, w.SamplingParams[0])
	// we can't go on if this didnt work
	if err != nil {
		log.Fatalf("Couldn't create tracer with given config. Error was: %v", err)
	}
	defer closer.Close()
	//Setup for prometheus metrics
	if !w.SetupDone {
		w.MetricsRegistry = prometheus.NewRegistry()
		w.SpanDurationHist = prometheus.NewHistogram(prometheus.HistogramOpts{
			Namespace: "worker",
			//Subsystem: config.OperationName,
			Name:    "span_duration",
			Help:    "A Histogram of Span durations",
			Buckets: []float64{10000.0, 20000.0, 50000.0, 100000.0, 200000.0},
		})
		w.MetricsRegistry.MustRegister(w.SpanDurationHist)
		w.SetupDone = true
	}
	w.Reporter = NewBufferingReporter(stream, 500)
	w.Config = config
	doneChannel := make(chan bool, 1)
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
	//start server in separate goroutine so we don't block here
	go server.Serve(listener)
	log.Printf("Started worker. Config: %v\n", config)
	w.UnitExecutorMap = make(map[string]Unit)
	for _, unit := range config.Units {
		w.UnitExecutorMap[unit.Identifier], err = CreateUnitExecutorFromConfig(unit, w)
	}
	w.Generator = NewOpenTracingSpanGenerator(tracer, w)
	//We create a new done channel here if this is a root service
	//TODO: Ideally, the "default" case for the done channel link would be that the worker didn't receive "Call"-Requests for a few seconds,
	//indicating that the root-workers no longer send requests;
	doneChannel = w.Generator.WriteSpansUntilExitSignal(stopChan, calculateIntervalByThroughput(config.TargetThroughput), w.Reporter)
	// go func() {
	// 	//TODO make report interval configurable; together with channel buffer size, this limits the maximum throughput!
	// 	reporterCollectionTicker := time.NewTicker(5 * time.Second)
	// ReporterLoop:
	// 	for {
	// 		select {
	// 		case <-doneChannel:
	// 			//log.Println("Received shutdown signal at write Span loop.")
	// 			break ReporterLoop
	// 		case <-reporterCollectionTicker.C:
	// 			go w.Reporter.Report()
	// 		}
	// 	}
	// }()

	//create a timer with runtime plus tolerance
	//TODO: make tolerance time (time after which worker is shut down forcefully after runtime) a parameter of the benchmark
	tolerance := 8 * time.Second
	timer := time.NewTimer(time.Duration(config.RuntimeSeconds) * time.Second)
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
					log.Printf("Benchmark shut down after runtime plus tolerance of %v.\n", tolerance)
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
		}
	}
	//important: do final reporting after benchmark ends. We give 5 seconds tolerance to make sure all results are reported.
	<-time.NewTimer(5 * time.Second).C
	w.Reporter.Report()
	server.GracefulStop()
	return nil
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

func (w *Worker) Call(ctx context.Context, id *api.DispatchId) (*api.Empty, error) {
	w.UnitExecutorMap[id.UnitReference].Invoke(ctx, w.GetTracer())
	return &api.Empty{}, nil
}
