package worker

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/dominik-/t-race/api"
	"github.com/opentracing/opentracing-go"
	"github.com/prometheus/client_golang/prometheus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
)

type Worker struct {
	Tracer           opentracing.Tracer
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

func (w *Worker) StartWorker(config *api.WorkerConfiguration, stream api.BenchmarkWorker_StartWorkerServer) error {
	//need to do the run here, i.e. start Writer with generator and return results
	//hook to SIGINT/SIGTERM
	sigTermRecv := make(chan os.Signal, 1)
	signal.Notify(sigTermRecv, syscall.SIGINT, syscall.SIGTERM)

	//Create sink (i.e. tracing backend) connection
	tracer, closer, err := InitTracer(config.SinkHostPort, config.ServiceName, w.SamplingStrategy, w.SamplingParams[0])
	// we can't go on if this didnt work
	if err != nil {
		log.Fatalf("Couldn't create tracer with given config. Error was: %v", err)
	}
	w.Tracer = tracer
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
	var generatorWG sync.WaitGroup
	generators := make([]UnitContextGenerator, 0)
	stopSignals := make([]chan bool, 0)
	//stopChan := make(chan bool, 1)

	for _, unit := range config.Units {
		unitExec, err := CreateUnitExecutorFromConfig(unit, w)
		if err != nil {
			log.Fatalf("Couldn't create executor for unit %s!", unit.Identifier)
		}
		if unit.ThroughputRatio > 0.000001 {
			generatorWG.Add(1)
			stopSignals = append(stopSignals, make(chan bool, 1))
			generators = append(generators, NewOpenTracingUnitSpanGenerator(unitExec, w.Config.ServiceName, tracer, w.Config.TargetThroughput, w.SpanDurationHist))
		}
		w.UnitExecutorMap[unit.Identifier] = unitExec
	}
	for i, generator := range generators {
		//TODO: do we need individual stop channels for each generator? to signal them to halt load generation?
		go generator.GenerateUntilExitSignal(stopSignals[i], w.Reporter, &generatorWG)
	}
	//w.Generator = NewOpenTracingSpanGenerator(tracer, w)
	//We create a new done channel here if this is a root service
	//TODO: Ideally, the "default" case for the done channel link would be that the worker didn't receive "Call"-Requests for a few seconds,
	//indicating that the root-workers no longer send requests;
	//doneChannel = w.Generator.WriteSpansUntilExitSignal(stopChan, config.TargetThroughput, w.Reporter)
	doneChannelLoadGenerators := make(chan bool, 1)
	doneChannelReporters := make(chan bool, 1)
	go func() {
		//TODO make report interval configurable; together with channel buffer size, this limits the maximum throughput!
		reporterCollectionTicker := time.NewTicker(5 * time.Second)
	ReporterLoop:
		for {
			select {
			case <-doneChannelReporters:
				//log.Println("Received shutdown signal at write Span loop.")
				break ReporterLoop
			case <-reporterCollectionTicker.C:
				go w.Reporter.Report()
			}
		}
	}()
	//create a timer with runtime plus tolerance
	//TODO: make tolerance time (time after which worker is shut down forcefully after runtime) a parameter of the benchmark
	tolerance := 8 * time.Second
	timer := time.NewTimer(time.Duration(config.RuntimeSeconds) * time.Second)
WorkerLoop:
	for {
		select {
		case <-timer.C:
			//Stop all running generators
			for _, ch := range stopSignals {
				ch <- true
			}
			limit := time.NewTimer(tolerance)
			go func(doneChannels ...chan bool) {
				generatorWG.Wait()
				for _, ch := range doneChannels {
					ch <- true
				}
			}(doneChannelLoadGenerators, doneChannelReporters)

			for {
				select {
				case <-doneChannelLoadGenerators:
					log.Println("Benchmark ended regularly after runtime.")
					break WorkerLoop
				case <-limit.C:
					log.Printf("Benchmark shut down after runtime plus tolerance of %v.\n", tolerance)
					break WorkerLoop
				}
			}
		case <-sigTermRecv:
			//Stop all running generators
			for _, ch := range stopSignals {
				ch <- true
			}
			limit := time.NewTimer(5 * time.Second)
			go func(doneChannels ...chan bool) {
				generatorWG.Wait()
				for _, ch := range doneChannels {
					ch <- true
				}
			}(doneChannelLoadGenerators, doneChannelReporters)
			for {
				select {
				case <-doneChannelLoadGenerators:
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

func (w *Worker) Call(ctx context.Context, id *api.DispatchId) (*api.Empty, error) {
	w.UnitExecutorMap[id.UnitReference].Invoke(ctx, w.Tracer)
	return &api.Empty{}, nil
}
