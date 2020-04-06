package provider

import (
	"encoding/json"
	"os"

	"github.com/dominik-/t-race/benchmark"
)

//Provider is a simple abstraction to integrate provisioning for deployment of t-race components.
type Provider interface {
	CreateEnvironments([]string)
	AllocateSinks([]*benchmark.Sink)
	AllocateServices([]*benchmark.Service)
}

//StaticProvider is the configuration-file-based basic provisioning, using "localhost" for deployment.
type StaticProvider struct {
	EnvMap     map[string]string
	SvcMap     map[string]string
	WorkerMap  map[string]string
	SinkMap    map[string]string
	deployment *Deployment
}

//WorkerAddress is a helper struct wrapping ip:port values for a Worker. JSON-tagged.
type WorkerAddress struct {
	BenchmarkAddress string `json:"benchmark"`
	ServiceAddress   string `json:"service"`
}

//Deployment wraps multiple workers and sinks. JSON-tagged.
type Deployment struct {
	WorkerAddresses []*WorkerAddress `json:"workers"`
	Sinks           []string         `json:"sinks"`
}

//NewStaticProvider creates a new StaticProvider from the given JSON file.
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

//CreateEnvironments for the static provider just uses localhost as the environment.
func (p *StaticProvider) CreateEnvironments(envRefs []string) {
	//we don't actually create environments here; usually we would create the instances and manage co-deployment here
	p.EnvMap = make(map[string]string, len(envRefs))
	for _, e := range envRefs {
		p.EnvMap[e] = "localhost"
	}
}

//AllocateServices maps services/worker addresses from the StaticProvider to the benchmark config.
func (p *StaticProvider) AllocateServices(svcs []*benchmark.Service) {
	p.SvcMap = make(map[string]string, len(svcs))
	p.WorkerMap = make(map[string]string, len(svcs))
	for i, s := range svcs {
		//We ignore environments here;
		p.SvcMap[s.Identifier] = p.deployment.WorkerAddresses[i].ServiceAddress
		p.WorkerMap[s.Identifier] = p.deployment.WorkerAddresses[i].BenchmarkAddress
	}
}

//AllocateSinks maps SUT addresses from the StaticProvider to the benchmark config.
func (p *StaticProvider) AllocateSinks(sinks []*benchmark.Sink) {
	p.SinkMap = make(map[string]string, len(sinks))
	for i, s := range sinks {
		p.SinkMap[s.Identifier] = p.deployment.Sinks[i]
	}
}
