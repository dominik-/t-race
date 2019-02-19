package benchmark

import "time"

type BenchmarkConfig struct {
	Throughput      int64
	ResultDirPrefix string
	Runtime         int64
}

type Record struct {
	TraceID    []byte
	SpanID     []byte
	StartTime  time.Time
	FinishTime time.Time
}
