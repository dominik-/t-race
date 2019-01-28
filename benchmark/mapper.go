package benchmark

import api "gitlab.tubit.tu-berlin.de/dominik-ernst/trace-writer-api"

func MapDeploymentToWorkerConfigs(d Deployment, sinks, services map[string]string) map[string]*api.WorkerConfiguration {
	workers := make(map[string]*api.WorkerConfiguration, len(d.Services))
	for _, svc := range d.Services {
		conf := &api.WorkerConfiguration{
			OperationName: svc.Identifier,
			SinkHostPort:  sinks[svc.SinkRef],
			Context:       toSpanContext(svc.Context),
			Root:          svc.IsRoot,
			Units:         toUnits(svc.Units, services),
			WorkFinal:     toWorkUnit(svc.FinalWork),
		}
		workers[svc.Identifier] = conf
	}
	//TODO: add errorhandling
	return workers
}

func toSpanContext(c *SpanContext) *api.ContextTemplate {
	return &api.ContextTemplate{
		Baggage: toTagTemplate(c.Baggage),
		Tags:    toTagTemplate(c.Tags),
	}
}

func toTagTemplate(template map[int]int) []*api.TagTemplate {
	templates := make([]*api.TagTemplate, 0)
	for keysize, valsize := range template {
		templates = append(templates, &api.TagTemplate{
			KeyByteLength:   int64(keysize),
			ValueByteLength: int64(valsize),
		})
	}
	return templates
}

func toUnits(units []*Unit, services map[string]string) []*api.Unit {
	unitsAPI := make([]*api.Unit, len(units))
	for i, u := range units {
		unitsAPI[i] = &api.Unit{
			InvokedHostPort: services[u.Successor.Identifier],
			RelType:         u.Rel,
			WorkBefore:      toWorkUnit(u.WorkUnit),
		}
	}
	return unitsAPI
}

func toWorkUnit(wu *WorkUnit) *api.Work {
	return &api.Work{
		DistType:   wu.Type,
		Parameters: wu.Params,
	}
}
