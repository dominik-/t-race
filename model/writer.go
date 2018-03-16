package model

import (
	"time"

	"github.com/opentracing/opentracing-go"
)

//TraceWriter is a component which writes traces. It requires a connection to an OpenTracing compatible tracer and ResultReporters. It is expected to write traces until an external 'stop' signal is received.
type TraceWriter interface {
	//Set up the connection for this writer.
	Initialize(OpentracingConnectionFactory, string)
	WriteSpan(TraceGenerator) *Result
	//Write traces until an exit signal is received on the channel give as parameter and report its own exit back through the returned channel.
	WriteSpansUntilExitSignal(TraceGenerator, chan bool, ...ResultReporter) chan bool
}

//ResultReporter is a wrapper around the calculation (e.g. average or quantiles) and writing of a result (e.g. to disk or std-out)
type ResultReporter interface {
	Report([]Result)
	CalculateResult([]*Result) float64
	WriteResult() error
}

type Result struct {
	Latency int64
	Counter int64
}

type IntervalTraceWriter struct {
	IntervalMilliseconds time.Duration
	tracer               opentracing.Tracer
	identifier           string
	results              []Result
}

func (w *IntervalTraceWriter) Initialize(conn OpentracingConnectionFactory, identifier string) {
	w.tracer = conn.CreateConnection(identifier)
	w.identifier = identifier
	w.results = make([]Result, 0)
}

func (w *IntervalTraceWriter) WriteSpansUntilExitSignal(tracegen TraceGenerator, receiveShutdownChannel chan bool, periodicReporters ...ResultReporter) chan bool {
	finishedIndicator := make(chan bool, 1)
	go func() {
		ticker := time.NewTicker(w.IntervalMilliseconds)
		reportTicker := time.NewTicker(1 * time.Second)
	GenerateLoop:
		for {
			select {
			case <-receiveShutdownChannel:
				break GenerateLoop
			case <-ticker.C:
				w.results = append(w.results, *w.WriteSpan(tracegen))
				break
			case <-reportTicker.C:
				for _, reporter := range periodicReporters {
					go reporter.Report(w.results)
					w.results = make([]Result, 0)
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

func (w *IntervalTraceWriter) WriteSpan(tracegen TraceGenerator) *Result {
	return tracegen.NextSpan(w.tracer)
}

func NewIntervalTraceWriter(intervalMilliseconds time.Duration) *IntervalTraceWriter {
	return &IntervalTraceWriter{
		IntervalMilliseconds: intervalMilliseconds,
	}
}
