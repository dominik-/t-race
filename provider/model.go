package provider

import (
	"encoding/json"
	"os"

	"gitlab.tubit.tu-berlin.de/dominik-ernst/tracer-benchmarks/benchmark"
)

type Provider interface {
	CreateEnvironments([]string)
	AllocateSinks([]*benchmark.Sink)
	AllocateServices([]*benchmark.Service)
}

type StaticProvider struct {
	EnvMap     map[string]string
	SvcMap     map[string]string
	SinkMap    map[string]string
	deployment *Deployment
}

type WorkerAddress struct {
	BenchmarkAddress string `json:"benchmark"`
	ServiceAddress   string `json:"service"`
}

type Deployment struct {
	WorkerAddresses []*WorkerAddress `json:"workers"`
	Sinks           []string         `json:"sinks"`
}

func NewStaticProvider(filename string) (*StaticProvider, error) {
	fileHandle, err := os.Open(filename)
	if err != nil {
		return nil, err
	}

	decoder := json.NewDecoder(fileHandle)
	var d Deployment
	err = decoder.Decode(&d)
	if err != nil {
		return nil, err
	}
	return &StaticProvider{
		deployment: &d,
	}, nil
}

func (p *StaticProvider) CreateEnvironments(envRefs []string) {
	//we don't actually create environments here; usually we would create the instances and manage co-deployment here
	p.EnvMap = make(map[string]string, len(envRefs))
	for _, e := range envRefs {
		p.EnvMap[e] = "localhost"
	}
}

func (p *StaticProvider) AllocateServices(svcs []*benchmark.Service) {
	p.SvcMap = make(map[string]string, len(svcs))
	for i, s := range svcs {
		//We ignore environments here;
		p.SvcMap[s.Identifier] = p.deployment.WorkerAddresses[i].ServiceAddress
	}
}

func (p *StaticProvider) AllocateSinks(sinks []*benchmark.Sink) {
	p.SinkMap = make(map[string]string, len(sinks))
	for i, s := range sinks {
		p.SinkMap[s.Identifier] = p.deployment.Sinks[i]
	}
}
