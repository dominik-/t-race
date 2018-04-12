package model

import (
	"time"

	tracewriter "gitlab.tubit.tu-berlin.de/dominik-ernst/trace-writer-api"
)

type ConstantSpanGenerator struct {
	OperationName string
	Tags          map[string]interface{}
	Delay         time.Duration
	Counter       int64
}

type TraceGenerator interface {
	NextSpan() tracewriter.SpanModel
}

func (csg *ConstantSpanGenerator) NextSpan() tracewriter.SpanModel {
	csg.Counter++
	return *&tracewriter.SpanModel{
		Delay:         csg.Delay,
		Tags:          csg.Tags,
		OperationName: csg.OperationName,
	}
}
