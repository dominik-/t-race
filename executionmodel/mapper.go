package executionmodel

import (
	"strings"

	"github.com/dominik-/t-race/api"
)

func MapArchitectureToWorkers(d Architecture, b BenchmarkConfig, sinkAddresses, serviceAddresses map[string]string) map[string]*api.WorkerConfiguration {
	workers := make(map[string]*api.WorkerConfiguration, len(d.Services))
	for _, svc := range d.Services {
		workers[svc.Identifier] = &api.WorkerConfiguration{
			WorkerId:         "worker-" + svc.Identifier,
			SinkHostPort:     sinkAddresses[svc.SinkRef],
			TargetThroughput: b.Throughput,
			RuntimeSeconds:   b.Runtime,
			ServiceName:      svc.Identifier,
			Units:            make([]*api.Unit, 0),
		}
		for _, unit := range svc.Units {
			inputs := make([]*api.UnitRef, 0)
			for _, input := range unit.InputRefs {
				inputs = append(inputs, &api.UnitRef{
					UnitId:    input.Unit,
					ServiceId: input.Service,
					Sync:      input.Sync,
				})
			}
			successors := make([]*api.UnitRef, len(unit.SuccessorRefs))
			for i, successor := range unit.SuccessorRefs {
				var isRemote bool
				remoteServiceAddress := ""
				if strings.Compare(successor.Service, svc.Identifier) == 0 {
					isRemote = false
				} else {
					isRemote = true
					remoteServiceAddress = serviceAddresses[successor.Service]
				}
				successors[i] = &api.UnitRef{
					ServiceId: successor.Service,
					UnitId:    successor.Unit,
					IsRemote:  isRemote,
					HostPort:  remoteServiceAddress,
					Sync:      successor.Sync,
				}
			}
			apiUnit := &api.Unit{
				Identifier:      unit.Identifier,
				RelType:         api.RelationshipType(unit.Rel),
				WorkBefore:      toWork(unit.WorkTemplate),
				Context:         toContext(unit.Context),
				Inputs:          inputs,
				ThroughputRatio: unit.ThroughputRatio,
				Successors:      successors,
			}
			workers[svc.Identifier].Units = append(workers[svc.Identifier].Units, apiUnit)
		}
	}
	//TODO: add errorhandling
	return workers
}

func toContext(c *Context) *api.ContextTemplate {
	if c != nil {
		return &api.ContextTemplate{
			Baggage: toKVTemplate(c.Baggage),
			Tags:    toKVTemplate(c.Tags),
		}
	}
	return nil
}

func toKVTemplate(template []*KeyValueTemplate) []*api.KeyValueTemplate {
	templates := make([]*api.KeyValueTemplate, 0)
	for _, tmpl := range template {
		var apiTemplate *api.KeyValueTemplate
		apiTemplate = &api.KeyValueTemplate{
			KeyStatic:   tmpl.KeyStatic,
			KeyLength:   int64(tmpl.KeyLength),
			ValueStatic: tmpl.ValueStatic,
			ValueLength: int64(tmpl.ValueLength),
		}
		templates = append(templates, apiTemplate)
	}
	return templates
}

func toWork(wu *Work) *api.Work {
	return &api.Work{
		DistType:   wu.Type,
		Parameters: wu.Params,
	}
}
