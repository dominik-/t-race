package provider

import "gitlab.tubit.tu-berlin.de/dominik-ernst/tracer-benchmarks/benchmark"

type Provider interface {
	CreateEnvironments([]*benchmark.Environments) map[string]string
	AllocateSinks([]*benchmark.Sink) map[string]string
	AllocateServices([]*benchmark.Service) map[string]string
}

type LocalStaticProvider struct {
	envMap              map[string]string
	svcMap              map[string]string
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

func (p *LocalStaticProvider) CreateEnvironments(envs []*benchmark.Environments) map[string]string {
	p.envMap = make(map[string]string, len(envs))
	for _, e := range envs {
		p.envMap[e.Identifier] = "localhost"
	}
	return
}

func (p *LocalStaticProvider) AllocateServices(svcs []*benchmark.Service) map[string]string {
	svcMap = make(map[string]string, len(svcs))
	for i, s := range svcs {
		var port int
		if p.allocateWorkerPorts {
			port = p.nextPort
			p.nextPort++
		} else {
			port = p.WorkerPorts[i]
		}
		svcMap[s.Identifier] = p.envMap[s.EnvironmentRef] + ":" + port
	}
	return
}

func (p *LocalStaticProvider) AllocateSinks(sinks []*benchmark.Sink) map[string]string {
	sinkMap = make(map[string]string, len(sinks))
	for i, s := range sinks {
		var port int
		if p.allocateSinkPorts {
			port = p.nextPort
			p.nextPort++
		} else {
			port = p.SinkPorts[i]
		}
		sinkMap[s.Identifier] = p.envMap[s.EnvironmentRef] + ":" + port
	}
	return
}
