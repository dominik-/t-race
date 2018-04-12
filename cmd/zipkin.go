package cmd

import (
	"github.com/opentracing/opentracing-go"

	"log"

	zipkin "github.com/openzipkin/zipkin-go-opentracing"
	"github.com/spf13/cobra"
)

var (
	zipkinAddress string
)

func init() {
	//rootCmd.AddCommand(zipkinCmd)
	//TODO
	zipkinCmd.Flags().StringVarP(&zipkinAddress, "address", "a", "http://localhost:9411/api/v1/spans", "The address of the jaeger agent to send traces to.")
}

var zipkinCmd = &cobra.Command{
	Use:   "zipkin",
	Short: "Benchmark zipkin as SUT",
	Long:  `Runs the tracing benchmark against zipkin as SUT, using zipkin-specific configuration`,
	Run:   RunBenchmarkWithZipkin,
}

//ZipkinConnectionFactory implements the OpenTracingConnectionFactory interface.
type ZipkinConnectionFactory struct {
	targetAddress string
	zipkinCloser  []zipkin.Collector
}

//RunBenchmarkWithZipkin simple wrapper around the root command, handing over the specific connection factory.
func RunBenchmarkWithZipkin(cmd *cobra.Command, args []string) {
	ExecuteBenchmark()
}

func NewZipkinConnection() *ZipkinConnectionFactory {
	return &ZipkinConnectionFactory{
		targetAddress: zipkinAddress,
		zipkinCloser:  make([]zipkin.Collector, 0),
	}

}

func (conf *ZipkinConnectionFactory) CreateConnection(identifier string) opentracing.Tracer {
	// Create our HTTP collector.
	collector, err := zipkin.NewHTTPCollector(conf.targetAddress)
	conf.zipkinCloser = append(conf.zipkinCloser, collector)

	if err != nil {
		log.Fatalf("Couldnt initialize connection: %s", err)
		return nil
	}

	// Create our recorder. (addr)
	recorder := zipkin.NewRecorder(collector, false, "", "traceBench")

	tracer, err := zipkin.NewTracer(recorder,
		zipkin.WithLogger(zipkin.LoggerFunc(func(kv ...interface{}) error {
			log.Printf("%+v\n", kv)
			return nil
		})),
		zipkin.DebugMode(false),
		zipkin.DebugAssertUseAfterFinish(false),
		zipkin.DebugAssertUseAfterFinish(false),
	)

	if err != nil {
		log.Fatalf("unable to create Zipkin tracer: %+v\n", err)
		return nil
	}

	return tracer
}

func (conf *ZipkinConnectionFactory) CloseConnections() {
	for _, closer := range conf.zipkinCloser {
		closer.Close()
	}
}
