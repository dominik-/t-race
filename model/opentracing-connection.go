package model

import (
	"github.com/opentracing/opentracing-go"
)

type OpentracingConnectionFactory interface {
	CreateConnection(identifier string) opentracing.Tracer
	CloseConnections()
}
