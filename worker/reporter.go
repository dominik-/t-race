package worker

import (
	"sync"

	"github.com/dominik-/t-race/api"
)

//ResultReporter is a wrapper around the collection and sending of results to the benchmark coordinator.
type ResultReporter interface {
	//Collect collects results from the results channel of a provided TraceWriter instance.
	Collect(*api.Result)
	//Report sends packages of results to the provided protobuf server stream
	Report()
}

type BufferingReporter struct {
	resultBuffer []*api.Result
	lock         sync.Mutex
	size         int
	target       api.BenchmarkWorker_StartWorkerServer
}

func NewBufferingReporter(target api.BenchmarkWorker_StartWorkerServer, bufferSize int) *BufferingReporter {
	if bufferSize == 0 || bufferSize < 0 {
		bufferSize = 1000
	}
	return &BufferingReporter{
		resultBuffer: make([]*api.Result, 0),
		size:         bufferSize,
		target:       target,
		lock:         sync.Mutex{},
	}
}

func (r *BufferingReporter) Collect(result *api.Result) {
	r.lock.Lock()
	defer r.lock.Unlock()
	r.resultBuffer = append(r.resultBuffer, result)
	if len(r.resultBuffer) > r.size {
		r.Report()
	}
}

func (r *BufferingReporter) Report() {
	r.lock.Lock()
	defer r.lock.Unlock()
	r.target.Send(&api.ResultPackage{
		Results: r.resultBuffer,
	})
	r.resultBuffer = make([]*api.Result, 0)
}
