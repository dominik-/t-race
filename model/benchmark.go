package model

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
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

func (b *Benchmark) RunBenchmark(conn OpentracingConnectionFactory) {
	//hook to SIGINT/SIGTERM
	sigTermRecv := make(chan os.Signal, 1)
	signal.Notify(sigTermRecv, syscall.SIGINT, syscall.SIGTERM)

	doneChannels := make([]chan bool, len(b.Writers))
	stopChannels := make([]chan bool, len(b.Writers))

	for chanNum := 0; chanNum < len(b.Writers); chanNum++ {
		stopChannels[chanNum] = make(chan bool, 1)
	}

	for i, writer := range b.Writers {
		writer.Initialize(conn, fmt.Sprintf("%s-%d", b.Config.WorkerPrefix, i))
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
