package benchmark

import api "gitlab.tubit.tu-berlin.de/dominik-ernst/trace-writer-api"

func MapDeploymentToWorkerConfigs(d Model, b BenchmarkConfig, sinks, services map[string]string) map[string]*api.WorkerConfiguration {
	workers := make(map[string]*api.WorkerConfiguration, len(d.Services))
	for _, svc := range d.Services {
		var finalWork *api.Work
		if svc.FinalWork != nil {
			finalWork = toWorkUnit(svc.FinalWork)
		}
		var context *api.ContextTemplate
		if svc.Context != nil {
			context = toSpanContext(svc.Context)
		}
		conf := &api.WorkerConfiguration{
			OperationName:    svc.Identifier,
			SinkHostPort:     sinks[svc.SinkRef],
			Context:          context,
			Root:             svc.IsRoot,
			Units:            toUnits(svc.Units, services),
			WorkFinal:        finalWork,
			TargetThroughput: b.Throughput,
			RuntimeSeconds:   b.Runtime,
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
	if units != nil && len(units) > 0 {
		unitsAPI := make([]*api.Unit, len(units))
		for i, u := range units {
			var workBefore *api.Work
			if u.WorkUnit != nil {
				workBefore = toWorkUnit(u.WorkUnit)
			}
			unitsAPI[i] = &api.Unit{
				InvokedHostPort: services[u.SuccessorRef],
				RelType:         u.Rel,
				WorkBefore:      workBefore,
			}
		}
		return unitsAPI
	}
	return nil
}

func toWorkUnit(wu *WorkUnit) *api.Work {
	return &api.Work{
		DistType:   wu.Type,
		Parameters: wu.Params,
	}
}
