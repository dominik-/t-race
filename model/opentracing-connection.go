package model

import (
	"github.com/opentracing/opentracing-go"
)

type OpentracingConnection interface {
	CreateConnection(identifier string) opentracing.Tracer
	CloseConnections()
}
