package cmd

import (
	"io"
	"log"

	"github.com/opentracing/opentracing-go"

	"github.com/spf13/cobra"
	jaeger "github.com/uber/jaeger-client-go"
)

var (
	address      string
	samplingRate float64
)

func init() {
	rootCmd.AddCommand(jaegerCmd)
	jaegerCmd.Flags().StringVarP(&address, "address", "a", "localhost:6831", "The address of the jaeger agent to send traces to.")
	jaegerCmd.Flags().Float64VarP(&samplingRate, "samplingRate", "s", 1.0, "Sampling rate for jaeger's probabilistic sampler.")
}

var jaegerCmd = &cobra.Command{
	Use:   "jaeger",
	Short: "Benchmark jaeger as SUT",
	Long:  `Runs the tracing benchmark against jaeger as SUT, using jaeger-specific configuration`,
	Run:   RunBenchmarkWithJaeger,
}

//JaegerConnectionFactory implements the OpenTracingConnectionFactory interface.
type JaegerConnectionFactory struct {
	probabilisticSamplingRate float64
	targetAddress             string
	jaegerClosers             []io.Closer
}

//RunBenchmarkWithJaeger simple wrapper around the root command, handing over the specific connection factory.
func RunBenchmarkWithJaeger(cmd *cobra.Command, args []string) {
	ExecuteBenchmark(NewJaegerConnection())
}

func (conf *JaegerConnectionFactory) CreateConnection(identifier string) opentracing.Tracer {
	//passing 0 makes jaeger use the max packet size, which seems to be recommended
	transport, err := jaeger.NewUDPTransport(conf.targetAddress, 0)

	if err != nil {
		log.Fatalf("Couldnt initialize connection: %s", err)
	}

	reporter := jaeger.NewRemoteReporter(transport)

	sampler, err := jaeger.NewProbabilisticSampler(conf.probabilisticSamplingRate)

	if err != nil {
		log.Fatalf("Couldnt initialize sampler: %s", err)
	}
	jaegerTracer, jaegerCloser := jaeger.NewTracer(identifier, sampler, reporter)
	conf.jaegerClosers = append(conf.jaegerClosers, jaegerCloser)
	return jaegerTracer
}

func (conf *JaegerConnectionFactory) CloseConnections() {
	for _, closer := range conf.jaegerClosers {
		closer.Close()
	}
}

func NewJaegerConnection() *JaegerConnectionFactory {
	return &JaegerConnectionFactory{
		probabilisticSamplingRate: samplingRate,
		targetAddress:             address,
		jaegerClosers:             make([]io.Closer, 0),
	}

}
