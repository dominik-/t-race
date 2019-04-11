package worker

import (
	"gitlab.tubit.tu-berlin.de/dominik-ernst/tracer-benchmarks/api"
)

//ResultReporter is a wrapper around the collection and sending of results to the benchmark coordinator.
type ResultReporter interface {
	//Collect collects results from the results channel of a provided TraceWriter instance.
	Collect(SpanGenerator)
	//Report sends packages of results to the provided protobuf server stream
	Report(api.BenchmarkWorker_StartWorkerServer)
}

type BufferingReporter struct {
	resultBuffer []*api.Result
	size         int
}

func NewBufferingReporter(bufferSize int) *BufferingReporter {
	//we dont use buffer size yet, thus set all sizes to zero
	return &BufferingReporter{
		resultBuffer: make([]*api.Result, 0),
		size:         0,
	}
}

func (r *BufferingReporter) Collect(sg SpanGenerator) {
	for result := range sg.GetResultsChannel() {
		r.resultBuffer = append(r.resultBuffer, result)
	}
}

func (r *BufferingReporter) Report(stream api.BenchmarkWorker_StartWorkerServer) {
	stream.Send(&api.ResultPackage{
		Results: r.resultBuffer,
	})
	r.resultBuffer = make([]*api.Result, r.size)
}
