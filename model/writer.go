package model

import (
	"log"
	"net/rpc"
	"time"

	tracewriter "gitlab.tubit.tu-berlin.de/dominik-ernst/trace-writer-api"
)

//TraceWriter is a component which writes traces. It requires a connection to an OpenTracing compatible tracer and ResultReporters. It is expected to write traces until an external 'stop' signal is received.
type TraceWriter interface {
	//Set up the connection for this writer.
	Initialize(string)
	WriteSpan(TraceGenerator) *tracewriter.Result
	//Write traces until an exit signal is received on the channel give as parameter and report its own exit back through the returned channel.
	WriteSpansUntilExitSignal(TraceGenerator, chan bool, ...ResultReporter) chan bool
}

//ResultReporter is a wrapper around the calculation (e.g. average or quantiles) and writing of a result (e.g. to disk or std-out)
type ResultReporter interface {
	Report([]*tracewriter.Result)
	CalculateResult([]*tracewriter.Result) float64
	WriteResult() error
}

type IntervalTraceWriter struct {
	Interval   time.Duration
	rpcClient  *rpc.Client
	identifier string
	results    []*tracewriter.Result
}

func (w *IntervalTraceWriter) Initialize(identifier string) {
	client, err := rpc.DialHTTP("tcp", ":5656")
	if err != nil {
		log.Fatal("Could not connect to local RPC server:", err)
	}
	w.rpcClient = client
	w.identifier = identifier
	w.results = make([]*tracewriter.Result, 0)
}

func (w *IntervalTraceWriter) WriteSpansUntilExitSignal(tracegen TraceGenerator, receiveShutdownChannel chan bool, periodicReporters ...ResultReporter) chan bool {
	finishedIndicator := make(chan bool, 1)
	go func() {
		ticker := time.NewTicker(w.Interval)
		reportTicker := time.NewTicker(1 * time.Second)
	GenerateLoop:
		for {
			select {
			case <-receiveShutdownChannel:
				break GenerateLoop
			case <-ticker.C:
				w.results = append(w.results, w.WriteSpan(tracegen))
				break
			case <-reportTicker.C:
				for _, reporter := range periodicReporters {
					go reporter.Report(w.results)
					w.results = make([]*tracewriter.Result, 0)
				}

			}
		}
		for _, reporter := range periodicReporters {
			go reporter.Report(w.results)
		}
		finishedIndicator <- true
		close(finishedIndicator)
	}()

	return finishedIndicator
}

func (w *IntervalTraceWriter) WriteSpan(tracegen TraceGenerator) *tracewriter.Result {
	span := tracegen.NextSpan()
	result := new(tracewriter.Result)
	err := w.rpcClient.Call("OpentracingWriter.WriteSpan", span, &result)
	if err != nil {
		log.Printf("Failed to write span: %v", err)
		result.Success = false
	}
	return result
}

func NewIntervalTraceWriter(interval time.Duration) *IntervalTraceWriter {
	return &IntervalTraceWriter{
		Interval: interval,
	}
}
