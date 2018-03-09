package main

import (
	"io"
	"log"
	"time"

	"github.com/opentracing/opentracing-go"
	jaeger "github.com/uber/jaeger-client-go"
)

type TraceWriter interface {
	WriteSpan(int)
	WriteSpansUntilFinished() int
}

type RuntimeTraceWriter struct {
	RuntimeInMinutes    time.Duration
	DelayInMicroseconds time.Duration
	TerminateSignal     <-chan bool
	tracer              opentracing.Tracer
	closer              io.Closer
	Identifier          string
}

func NewRuntimeTraceWriter(runtimeMinutes, delayMicroseconds int64, shutdown <-chan bool, id string) *RuntimeTraceWriter {

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
		RuntimeInMinutes:    time.Duration(runtimeMinutes),
		DelayInMicroseconds: time.Duration(delayMicroseconds),
		TerminateSignal:     shutdown,
		tracer:              jaegerTracer,
		closer:              jaegerCloser,
		Identifier:          id,
	}
}

func (t *RuntimeTraceWriter) WriteSpan(counter int) {
	span := t.tracer.StartSpan(t.Identifier)
	span.SetTag("iteration", counter)
	<-time.After(t.DelayInMicroseconds * time.Microsecond)
	span.Finish()
}

func (t *RuntimeTraceWriter) WriteSpansUntilFinished() int {
	//TODO make frequency adjustable
	ticker := time.NewTicker(100 * time.Millisecond)
	timer := time.After(t.RuntimeInMinutes * time.Minute)
	counter := 0
GenerateLoop:
	for {
		select {
		case <-t.TerminateSignal:
			break GenerateLoop
		case <-timer:
			break GenerateLoop
		case <-ticker.C:
			counter++
			t.WriteSpan(counter)
			break
		}
	}
	t.closer.Close()
	return counter
}
