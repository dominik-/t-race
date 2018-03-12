package jaeger

import (
	"fmt"
	"io"
	"log"
	"time"

	"github.com/opentracing/opentracing-go"
	jaeger "github.com/uber/jaeger-client-go"
)

type TraceWriter interface {
	WriteSpan(int)
	WriteSpansUntilFinished()
}

type RuntimeTraceWriter struct {
	RuntimeInMinutes     time.Duration
	DelayInMicroseconds  time.Duration
	IntervalMilliseconds time.Duration
	TerminateSignal      chan bool
	tracer               opentracing.Tracer
	closer               io.Closer
	Identifier           string
	TraceCounter         int64
}

func NewRuntimeTraceWriter(runtimeMinutes, delayMicroseconds, intervalMilliseconds int64, id string) *RuntimeTraceWriter {

	transport, err := jaeger.NewUDPTransport("localhost:6831", 1500)

	if err != nil {
		log.Fatalf("Couldnt initialize connection: %s", err)
	}

	reporter := jaeger.NewRemoteReporter(transport)

	sampler, err := jaeger.NewProbabilisticSampler(1.0)

	if err != nil {
		log.Fatalf("Couldnt initialize sampler: %s", err)
	}
	jaegerTracer, jaegerCloser := jaeger.NewTracer("benchmarker", sampler, reporter)

	return &RuntimeTraceWriter{
		RuntimeInMinutes:     time.Duration(runtimeMinutes),
		DelayInMicroseconds:  time.Duration(delayMicroseconds),
		IntervalMilliseconds: time.Duration(intervalMilliseconds),
		TerminateSignal:      make(chan bool, 1),
		tracer:               jaegerTracer,
		closer:               jaegerCloser,
		Identifier:           id,
	}
}

func (t *RuntimeTraceWriter) WriteSpan(counter int64) {
	span := t.tracer.StartSpan(t.Identifier)
	span.SetTag("iteration", counter)
	<-time.After(t.DelayInMicroseconds * time.Microsecond)
	span.Finish()
}

func (t *RuntimeTraceWriter) WriteSpansUntilFinished() {
	ticker := time.NewTicker(t.IntervalMilliseconds * time.Millisecond)
	timer := time.After(t.RuntimeInMinutes * time.Minute)
GenerateLoop:
	for {
		select {
		case <-t.TerminateSignal:
			break GenerateLoop
		case <-timer:
			break GenerateLoop
		case <-ticker.C:
			t.TraceCounter++
			t.WriteSpan(t.TraceCounter)
			break
		}
	}
	t.closer.Close()
	fmt.Printf("TraceWriter %s wrote %d traces.\n", t.Identifier, t.TraceCounter)
}
