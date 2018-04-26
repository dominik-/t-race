package model

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"gitlab.tubit.tu-berlin.de/dominik-ernst/tracer-benchmarks/proto"
)

type Benchmark struct {
	Reporters []ResultReporter
	Writers   []TraceWriter
	TraceGen  TraceGenerator
	Config    *BenchmarkConfig
}

type BenchmarkConfig struct {
	Interval        time.Duration
	Workers         int
	WorkerPrefix    string
	ResultDirPrefix string
	Runtime         time.Duration
}

func NewBenchmark(config *BenchmarkConfig, traceGen TraceGenerator, reporters ...ResultReporter) *Benchmark {

	writers := make([]TraceWriter, config.Workers)
	for i := 0; i < config.Workers; i++ {
		writers[i] = NewIntervalTraceWriter(config.Interval)
	}
	return &Benchmark{
		TraceGen:  traceGen,
		Reporters: reporters,
		Config:    config,
		Writers:   writers,
	}
}

type Worker struct {
	Reporters []ResultReporter
	Writer    TraceWriter
	Generator TraceGenerator
}

func (w *Worker) StartWorker(config *proto.WorkerConfiguration, stream proto.BenchmarkWorker_StartWorkerServer) error {
	//need to do the run here, i.e. start Writer with generator and return results
	//hook to SIGINT/SIGTERM
	sigTermRecv := make(chan os.Signal, 1)
	signal.Notify(sigTermRecv, syscall.SIGINT, syscall.SIGTERM)
	stopChan := make(chan bool)
	w.Writer.Initialize(config.WorkerId)
	doneChannel := w.Writer.WriteSpansUntilExitSignal(w.Generator, stopChan, w.Reporters...)
	log.Printf("Started worker. Config: %v\n", config)
	//create a timer with runtime
	timer := time.NewTimer(time.Duration(config.RuntimeSeconds) * time.Second)
	ticker := time.NewTicker(10 * time.Second)
	for {
		select {
		case <-timer.C:
			stopChan <- true
			limit := time.NewTimer(5 * time.Second)
			for {
				select {
				case <-doneChannel:
					log.Println("Benchmark ended after runtime.")
					os.Exit(0)
					break
				case <-limit.C:
					log.Fatalln("Benchmark shut down forcefully after runtime.")
					break
				}
			}
			break
		case <-sigTermRecv:
			stopChan <- true
			limit := time.NewTimer(5 * time.Second)
			for {
				select {
				case <-doneChannel:
					log.Println("Benchmark ended by manual interrupt from user.")
					os.Exit(0)
					break
				case <-limit.C:
					log.Fatalln("Benchmark shut down forcefully after manual interrupt from user.")
					break
				}
			}
			break
		case <-ticker.C:
			for _, reporter := range w.Reporters {
				//TODO refactor reporter interface and make it create a ResultPackage for grpc return-send
				reporter.WriteResult()
				//stream.Send()
			}
		}
	}
}

func (b *Benchmark) RunBenchmark() {
	//hook to SIGINT/SIGTERM
	sigTermRecv := make(chan os.Signal, 1)
	signal.Notify(sigTermRecv, syscall.SIGINT, syscall.SIGTERM)

	doneChannels := make([]chan bool, len(b.Writers))
	stopChannels := make([]chan bool, len(b.Writers))

	for chanNum := 0; chanNum < len(b.Writers); chanNum++ {
		stopChannels[chanNum] = make(chan bool, 1)
	}

	for i, writer := range b.Writers {
		writer.Initialize(fmt.Sprintf("%s-%d", b.Config.WorkerPrefix, i))
		doneChannels[i] = writer.WriteSpansUntilExitSignal(b.TraceGen, stopChannels[i])
	}
	fmt.Printf("Started benchmark. Config: %v\n", b.Config)
	//create a timer with runtime
	timer := time.NewTimer(b.Config.Runtime)

	//we end if either SIGINT/SIGTERM is received or the time finishes
	for {
		select {
		case <-timer.C:
			for _, stopChan := range stopChannels {
				stopChan <- true
			}
			<-time.NewTimer(500 * time.Millisecond).C
			os.Exit(0)
			break
		case <-sigTermRecv:
			for _, stopChan := range stopChannels {
				stopChan <- true
			}
			//after signalling to shutdown to all writers, wait half a second, then exit.
			<-time.NewTimer(500 * time.Millisecond).C
			log.Fatal("User arborted.")
			break
		}
	}
}
