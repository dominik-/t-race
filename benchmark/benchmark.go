package benchmark

type BenchmarkConfig struct {
	Throughput      int64
	WorkerPrefix    string
	ResultDirPrefix string
	Runtime         int64
}
