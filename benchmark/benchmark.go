package benchmark

import (
	"fmt"

	"github.com/golang/protobuf/ptypes"
	"gitlab.tubit.tu-berlin.de/dominik-ernst/tracer-benchmarks/api"
)

type BenchmarkConfig struct {
	Throughput      int64
	ResultDirPrefix string
	Runtime         int64
}

type Record struct {
	Service     string
	TraceNumber int64
	SpanNumber  int64
	StartTime   int64
	FinishTime  int64
	Sampled     bool
}

func resultsToRecords(results *api.ResultPackage, worker *api.WorkerConfiguration) []*Record {
	resultSlice := results.GetResults()
	records := make([]*Record, len(resultSlice))
	for i := range resultSlice {
		startTime, err := ptypes.Timestamp(resultSlice[i].StartTime)
		endTime, err := ptypes.Timestamp(resultSlice[i].FinishTime)
		if err != nil {
			fmt.Printf("Couldn't convert timestamp %v to time, error was: %s", resultSlice[i].StartTime, err)
		}
		records[i] = &Record{
			Service:     worker.GetOperationName(),
			TraceNumber: resultSlice[i].TraceNum,
			SpanNumber:  resultSlice[i].SpanNum,
			StartTime:   startTime.UnixNano(),
			FinishTime:  endTime.UnixNano(),
			Sampled:     resultSlice[i].Sampled,
		}
	}
	return records
}
