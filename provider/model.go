package provider

import (
	"strconv"

	"gitlab.tubit.tu-berlin.de/dominik-ernst/tracer-benchmarks/benchmark"
)

type Provider interface {
	CreateEnvironments([]string)
	AllocateSinks([]*benchmark.Sink)
	AllocateServices([]*benchmark.Service)
}

type LocalStaticProvider struct {
	EnvMap              map[string]string
	SvcMap              map[string]string
	SinkMap             map[string]string
	WorkerPorts         []int
	SinkPorts           []int
	allocateWorkerPorts bool
	allocateSinkPorts   bool
	nextPort            int
}

func NewLocalStaticProvider(workerPorts, sinkPorts []int) *LocalStaticProvider {
	prov := &LocalStaticProvider{}
	if workerPorts == nil || len(workerPorts) < 1 {
		prov.allocateWorkerPorts = true
		prov.WorkerPorts = make([]int, 0)
	}
	if sinkPorts == nil || len(sinkPorts) < 1 {
		prov.allocateSinkPorts = true
		prov.SinkPorts = make([]int, 0)
	}
	prov.nextPort = 9001
	return prov
}

func (p *LocalStaticProvider) CreateEnvironments(envRefs []string) {
	p.EnvMap = make(map[string]string, len(envRefs))
	for _, e := range envRefs {
		p.EnvMap[e] = "localhost"
	}
}

func (p *LocalStaticProvider) AllocateServices(svcs []*benchmark.Service) {
	p.SvcMap = make(map[string]string, len(svcs))
	for i, s := range svcs {
		var port int
		if p.allocateWorkerPorts {
			port = p.nextPort
			p.nextPort++
		} else {
			port = p.WorkerPorts[i]
		}
		p.SvcMap[s.Identifier] = p.EnvMap[s.EnvironmentRef] + ":" + strconv.Itoa(port)
	}
}

func (p *LocalStaticProvider) AllocateSinks(sinks []*benchmark.Sink) {
	p.SinkMap = make(map[string]string, len(sinks))
	for i, s := range sinks {
		var port int
		if p.allocateSinkPorts {
			port = p.nextPort
			p.nextPort++
		} else {
			port = p.SinkPorts[i]
		}
		p.SinkMap[s.Identifier] = p.EnvMap[s.EnvironmentRef] + ":" + strconv.Itoa(port)
	}
}
