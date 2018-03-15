package model

import (
	"time"

	"github.com/opentracing/opentracing-go"
)

type ConstantSpanGenerator struct {
	ServiceName string
	Tags        map[string]interface{}
	Duration    time.Duration
	Counter     int64
}

type TraceGenerator interface {
	NextSpan(opentracing.Tracer) *Result
}

func (csg *ConstantSpanGenerator) NextSpan(t opentracing.Tracer) *Result {
	t1 := time.Now()
	span := t.StartSpan(csg.ServiceName)
	t2 := time.Since(t1)
	for tagKey, tagValue := range csg.Tags {
		span.SetTag(tagKey, tagValue)
	}
	<-time.After(csg.Duration)
	t3 := time.Now()
	span.Finish()
	t4 := time.Since(t3)
	csg.Counter++
	return &Result{
		Latency: int64(t2) + int64(t4),
		Counter: csg.Counter,
	}
}
