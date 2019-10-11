package benchmark

import (
	"encoding/binary"
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
	TraceID     string
	SpanID      uint64
	StartTime   int64
	FinishTime  int64
	Sampled     bool
}

type TraceID []byte

func (t TraceID) String() string {
	high := binary.BigEndian.Uint64(t[:8])
	low := binary.BigEndian.Uint64(t[8:])
	if high == 0 {
		return fmt.Sprintf("0x%032x", low)
	}
	return fmt.Sprintf("%x%016x", high, low)
}

func resultsToRecords(results *api.ResultPackage, worker *api.WorkerConfiguration) []*Record {
	resultSlice := results.GetResults()
	records := make([]*Record, len(resultSlice))
	for i := range resultSlice {
		traceID := TraceID{}
		traceID = resultSlice[i].TraceId
		spanID := binary.BigEndian.Uint64(resultSlice[i].SpanId)
		startTime, err := ptypes.Timestamp(resultSlice[i].StartTime)
		endTime, err := ptypes.Timestamp(resultSlice[i].FinishTime)
		if err != nil {
			fmt.Printf("Couldn't convert timestamp %v to time, error was: %s", resultSlice[i].StartTime, err)
		}
		records[i] = &Record{
			Service:     worker.GetOperationName(),
			TraceNumber: resultSlice[i].TraceNum,
			SpanNumber:  resultSlice[i].SpanNum,
			TraceID:     traceID.String(),
			SpanID:      spanID,
			StartTime:   startTime.UnixNano(),
			FinishTime:  endTime.UnixNano(),
			Sampled:     resultSlice[i].Sampled,
		}
	}
	return records
}
