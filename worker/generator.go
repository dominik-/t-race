package worker

import (
	"context"
	"encoding/binary"
	"io"
	"log"
	"reflect"
	"time"

	"github.com/dominik-/t-race/api"
	"github.com/golang/protobuf/ptypes"

	"github.com/opentracing/opentracing-go"
	"github.com/prometheus/client_golang/prometheus"

	jaegerclient "github.com/uber/jaeger-client-go"
	jaegercfg "github.com/uber/jaeger-client-go/config"
)

//TODO this flag is implementation-specific for jaeger-client-go impl of opentracing SpanContext. Should remove this.
const flagSampled = byte(1)

type OpenTracingSpanGenerator struct {
	TraceCounter     int64
	SpanDurationHist prometheus.Histogram
	Tracer           opentracing.Tracer
	Closer           io.Closer
	Units            map[string]Unit
	Tags             map[string]string
	Baggage          map[string]string
	WorkFinalDist    DistributionSampler
	ServiceName      string
}

type GeneratedUnit struct {
	Unit        *api.Unit
	WorkSampler DistributionSampler
	UnitClient  api.BenchmarkWorkerClient
}

func CreateGeneratedUnit(u *api.Unit, client api.BenchmarkWorkerClient) (*GeneratedUnit, error) {
	dist, err := LookupDistribution(u.WorkBefore)
	if err != nil {
		//surface error
		return nil, err
	}
	return &GeneratedUnit{
		Unit:        u,
		WorkSampler: dist,
		UnitClient:  client,
	}, nil
}

type SpanGenerator interface {
	DoUnitCalls(ResultReporter) bool
	GetTracer() opentracing.Tracer
	WriteSpansUntilExitSignal(<-chan bool, time.Duration, ResultReporter) chan bool
}

func (sg *OpenTracingSpanGenerator) GetTracer() opentracing.Tracer {
	return sg.Tracer
}

// InitTracer returns an instance of a Tracer that logs sampled Spans to stdout the given sinkAddress.
func InitTracer(sinkAddress, serviceName, samplingstrategy string, samplingParam float64) (opentracing.Tracer, io.Closer, error) {
	tracerConfig := jaegercfg.Configuration{
		ServiceName: serviceName,
		Sampler: &jaegercfg.SamplerConfig{
			Type:  samplingstrategy,
			Param: samplingParam,
		},
		Reporter: &jaegercfg.ReporterConfig{
			LocalAgentHostPort: sinkAddress,
			LogSpans:           true,
		},
	}
	return tracerConfig.NewTracer()
}

func NewOpenTracingSpanGenerator(tracer opentracing.Tracer, worker *Worker) SpanGenerator {
	return &OpenTracingSpanGenerator{
		TraceCounter:     0,
		Units:            worker.UnitExecutorMap,
		Tracer:           tracer,
		SpanDurationHist: worker.SpanDurationHist,
		ServiceName:      worker.Config.ServiceName,
	}
}

//this function is actually jaeger-specific, as the OpentTacing API doesn't include any operations for accessing content of the context.
func getSampledFlag(ctx opentracing.SpanContext) bool {
	if converted, ok := ctx.(jaegerclient.SpanContext); ok {
		return converted.IsSampled()
	}
	panic("Couldn't convert opentracing context to jaeger context!")
	// if ctx == nil {
	// 	//we can panic here, because we will panic further down anyways if the context is nil
	// 	panic("Context is nil!!")
	// }
	// v := reflect.ValueOf(ctx)
	// f := v.FieldByName("flags")

	// if !f.IsValid() || f.Kind() != reflect.Uint8 {
	// 	panic("Couldn't convert to byte!!")
	// }

	// byteFlags := byte(f.Uint())

	// return (byteFlags & flagSampled) == flagSampled
}

//TODO: remove this after conversion of opentracing spancontext to jaeger-client-go spancontext (see getSampledFlag() method)
type TraceID struct {
	High, Low uint64
}

//TODO: refactor this to use a conversion of opentracing spancontext to jaeger-client-go spancontext (see getSampledFlag() method)
func getTraceIdAsBytes(ctx opentracing.SpanContext) []byte {
	v := reflect.ValueOf(ctx)
	f := v.FieldByName("traceID")

	if !f.IsValid() || f.Kind() != reflect.Struct {
		log.Printf("Couldn't convert to struct!!")
	}
	//One level deeper...

	b1, b2 := make([]byte, 8), make([]byte, 8)
	binary.BigEndian.PutUint64(b1, f.FieldByName("High").Uint())
	binary.BigEndian.PutUint64(b2, f.FieldByName("Low").Uint())
	return append(b1, b2...)
}

//TODO: refactor this to use a conversion of opentracing spancontext to jaeger-client-go spancontext (see getSampledFlag() method)
func getSpanID(ctx opentracing.SpanContext) []byte {
	v := reflect.ValueOf(ctx)
	f := v.FieldByName("spanID")

	if !f.IsValid() || f.Kind() != reflect.Uint64 {
		log.Printf("Couldn't convert to uint!!")
	}
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, f.Uint())
	return b
}

func (sg *OpenTracingSpanGenerator) finishAndReportTimedOTSpan(startTime time.Time, span opentracing.Span, childSpanCount int64, reporter ResultReporter) error {
	sampled := getSampledFlag(span.Context())
	traceID := getTraceIdAsBytes(span.Context())
	spanID := getSpanID(span.Context())
	span.Finish()
	finishTimeDelta := time.Since(startTime)
	sg.SpanDurationHist.Observe(float64(finishTimeDelta.Nanoseconds() / 1000.0))
	started, err := ptypes.TimestampProto(startTime)
	finished, err := ptypes.TimestampProto(startTime.Add(finishTimeDelta))
	if err != nil {
		log.Printf("Couldn't convert timestamps to proto format.")
		return err
	}
	reporter.Collect(&api.Result{
		TraceNum:   sg.TraceCounter,
		SpanNum:    childSpanCount,
		TraceId:    traceID,
		SpanId:     spanID,
		StartTime:  started,
		FinishTime: finished,
		Sampled:    sampled,
	})
	return nil
}

func (sg *OpenTracingSpanGenerator) DoUnitCalls(reporter ResultReporter) bool {
	//This function emulates load applied to "root" execution units, thus traces start here. We create a new, empty context.
	ctx := context.Background()
	//TODO: should we create a "root" trace here already?

	//TODO create result of trace generation and append result data to reporter
	//extract trace context from grpc metadata
	//create child relationship to client span - TODO: does that always make sense?
	//var serverSpan opentracing.Span
	//spanStart := time.Now()
	//we add this service's tags and baggage before doing further calls
	//Generate spans for subsequent calls.
	childrenCounter := 0
	for _, unit := range sg.Units {
		//TODO: respect the weighting here and implement counters accordingly!!
		//TODO: currently, precision is limited to 5e-10, but this is unlikely an issue
		if unit.GetLoadPercentage() > 0.00001 {
			go unit.Invoke(ctx, sg.GetTracer())
		}
		childrenCounter++
	}
	//Finish parent span and do reporting etc.
	//TODO return a result!?
	return true
}

//WriteSpansUntilExitSignal takes a SpanGenerator, an exitSignalChannel and reporters.
//It loops writing spans to the IntervalTraceWriter until the exitSignalChannel receives a value.
//All reporters are invoked periodically every second. The returned channel can be used to listen for the termination of the write-loop.
func (sg *OpenTracingSpanGenerator) WriteSpansUntilExitSignal(exitSignalChannel <-chan bool, interval time.Duration,
	reporter ResultReporter) chan bool {
	finishedIndicator := make(chan bool, 1)
	go func() {
		ticker := time.NewTicker(interval)
		//TODO make report interval configurable; together with channel buffer size, this limits the maximum throughput!
	GenerateLoop:
		for {
			select {
			case <-exitSignalChannel:
				//log.Println("Received shutdown signal at write Span loop.")
				break GenerateLoop
			case <-ticker.C:
				//write span (async) to the writer, generate new parent context
				go sg.DoUnitCalls(reporter)
				break
			}
		}
		//by sending to this channel, we indicate a regular shutdown.
		//log.Println("Sending finished signal from write Span loop to worker loop.")
		finishedIndicator <- true
		close(finishedIndicator)
	}()

	return finishedIndicator
}
