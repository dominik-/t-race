package worker

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"reflect"
	"time"

	"github.com/golang/protobuf/ptypes"

	"github.com/opentracing/opentracing-go"
	"github.com/prometheus/client_golang/prometheus"
	"gitlab.tubit.tu-berlin.de/dominik-ernst/tracer-benchmarks/api"

	jaegerclient "github.com/uber/jaeger-client-go"
	jaegercfg "github.com/uber/jaeger-client-go/config"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

//TODO this flag is implementation-specific for jaeger-client-go impl of opentracing SpanContext. Should remove this.
const flagSampled = byte(1)

type OpenTracingSpanGenerator struct {
	TraceCounter     int64
	SpanDurationHist prometheus.Histogram
	Tracer           opentracing.Tracer
	Closer           io.Closer
	Units            []*GeneratedUnit
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
	DoUnitCalls(context.Context, ResultReporter) bool
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

func NewOpenTracingSpanGenerator(tracer opentracing.Tracer, config *api.WorkerConfiguration, hist prometheus.Histogram) SpanGenerator {
	// initialize empty generator
	generator := &OpenTracingSpanGenerator{
		TraceCounter:     0,
		Units:            make([]*GeneratedUnit, 0),
		Tracer:           tracer,
		SpanDurationHist: hist,
	}
	var err error
	// Create TLS credentials for grpc clients that skip root CA verification
	/* 	creds := credentials.NewTLS(&tls.Config{
	   		InsecureSkipVerify: true,
	   	})
		option := grpc.WithTransportCredentials(creds) */
	securityOption := grpc.WithInsecure()
	// Establish connections to all following workers.
	for _, unit := range config.Units {
		conn, err := grpc.Dial(unit.InvokedHostPort, securityOption)
		if err != nil {
			log.Printf("Couldnt connect to hostport of unit: %v, error was: %v\n", unit, err)
		}
		genUnit, err := CreateGeneratedUnit(unit, api.NewBenchmarkWorkerClient(conn))
		if err != nil {
			log.Printf("Couldnt create a unit, error was: %v\n", err)
		} else {
			generator.Units = append(generator.Units, genUnit)
		}
	}
	if len(generator.Units) != len(config.Units) {
		log.Printf("Only %d units of %d were successfully created/parsed. Please check logs for more details.", len(generator.Units), len(config.Units))
	}

	generator.WorkFinalDist, err = LookupDistribution(config.WorkFinal)
	if err != nil {
		log.Fatalf("Couldnt get the distribution for the WorkFinal part, error was: %v\n", err)
	}
	//Generate tags and baggage once
	if config.Context != nil {
		generator.Tags = toStringMap(config.Context.Tags)
		generator.Baggage = toStringMap(config.Context.Baggage)
	}
	generator.ServiceName = config.OperationName
	return generator
}

//Turns templates for tags and baggage into a map of strings to strings.
func toStringMap(templates []*api.KeyValueTemplate) map[string]string {
	data := make(map[string]string, len(templates))
	for _, tagTemplate := range templates {
		//Differentiation 0: key is static value, i.e. check for length to be 0 or less
		if tagTemplate.GetKeyLength() <= 0 {
			//Differentiation 1: value is a static value, i.e. check for length to be 0 or less
			if tagTemplate.GetValueLength() <= 0 {
				data[tagTemplate.GetKeyStatic()] = tagTemplate.GetValueStatic()
			} else {
				data[tagTemplate.GetKeyStatic()] = RandStringWithLength(tagTemplate.GetValueLength())
			}
		} else {
			//Differentiation 1 (again): value is a static value, i.e. check for length to be 0 or less
			if tagTemplate.GetValueLength() <= 0 {
				data[RandStringWithLength(tagTemplate.GetKeyLength())] = tagTemplate.GetValueStatic()
			} else {
				data[RandStringWithLength(tagTemplate.GetKeyLength())] = RandStringWithLength(tagTemplate.GetValueLength())
			}
		}
	}
	return data
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

func (sg *OpenTracingSpanGenerator) DoUnitCalls(parent context.Context, reporter ResultReporter) bool {
	//get grpc metadata from golang Context
	md, ok := metadata.FromIncomingContext(parent)
	if !ok {
		md = metadata.New(nil)
	}
	//TODO create result of trace generation and append result data to reporter
	//extract trace context from grpc metadata
	remoteContext, err := sg.Tracer.Extract(opentracing.HTTPHeaders, metadataReaderWriter{md})
	//create child relationship to client span - TODO: does that always make sense?
	var serverSpan opentracing.Span
	var ctx context.Context
	spanStart := time.Now()
	if err != nil && err == opentracing.ErrSpanContextNotFound {
		//start local "parent" span as root span
		serverSpan = sg.Tracer.StartSpan(sg.ServiceName + "-parent")
		ctx = opentracing.ContextWithSpan(context.Background(), serverSpan)
	} else if err != nil {
		log.Fatalf("Couldn't parse Span Context! Error was: %v", err)
	} else {
		//start local span with child relationship to parent from remote context; note that this is always a "child" reference, as the parent is a "client span" from the caller specific to this service,
		// which in turn has correctly mapped CHILD or FOLLOWS relationship to its parent.
		option := opentracing.ChildOf(remoteContext)
		serverSpan = sg.Tracer.StartSpan(sg.ServiceName+"-parent", option)
		ctx = opentracing.ContextWithSpan(parent, serverSpan)
	}
	//we add this service's tags and baggage before doing further calls
	for tagKey, tagValue := range sg.Tags {
		serverSpan.SetTag(tagKey, tagValue)
	}
	for tagKey, tagValue := range sg.Baggage {
		serverSpan.SetBaggageItem(tagKey, tagValue)
	}
	var childrenCounter int64 = 0
	//Generate spans for subsequent calls.
	for i, unit := range sg.Units {
		//Step 1: create client-side span for calling the successor service, including relationship to current context.
		var relOption opentracing.SpanReference
		switch unit.Unit.GetRelType() {
		case api.RelationshipType_CHILD:
			relOption = opentracing.ChildOf(serverSpan.Context())
		case api.RelationshipType_FOLLOWING:
			relOption = opentracing.FollowsFrom(serverSpan.Context())
		}
		clientSpanName := fmt.Sprintf("%s-call-%d", sg.ServiceName, i)
		clientSpanStart := time.Now()
		localClientSpan := sg.Tracer.StartSpan(clientSpanName, relOption)
		//Update context with reference to new client Span
		ctx = opentracing.ContextWithSpan(ctx, localClientSpan)
		//Step 2: wait for internal work emulation before the actual call;
		//if workBefore is nil, this check returns the NoDistribution, which means the call should be done in parallel to the previous one.
		//TODO: how long does this type check take? bad to do it every call?
		if _, parsed := unit.WorkSampler.(*NoDistribution); !parsed {
			//if the Distribution cannot be parsed to the NoDistribution, wait for the sampled amount of time and make a "synchronous" remote call
			<-time.NewTimer(unit.WorkSampler.GetNextValue()).C
			sg.DoRemoteCall(unit, ctx, localClientSpan, clientSpanStart, childrenCounter, reporter)
		} else {
			//if no workBefore is done, do "async" remote call
			go sg.DoRemoteCall(unit, ctx, localClientSpan, clientSpanStart, childrenCounter, reporter)
		}
		childrenCounter++
	}
	//Do final work locally.
	if _, parsed := sg.WorkFinalDist.(*NoDistribution); !parsed {
		<-time.NewTimer(sg.WorkFinalDist.GetNextValue()).C
	}

	//Finish parent span and do reporting etc.
	sg.finishAndReportTimedOTSpan(spanStart, serverSpan, -1, reporter)
	sg.TraceCounter++
	//TODO return a result!?
	return true
}

func (sg *OpenTracingSpanGenerator) DoRemoteCall(unit *GeneratedUnit, ctx context.Context, clientSpan opentracing.Span, spanStart time.Time, childSpanNum int64, reporter ResultReporter) {
	//Step 3a: Use context ("outgoing" is from the perspective of the calling service!) and create a metadata writer;
	md, ok := metadata.FromOutgoingContext(ctx)
	if !ok {
		md = metadata.New(nil)
	} else {
		md = md.Copy()
	}
	mdWriter := metadataReaderWriter{md}
	//Step 3b: Inject the local span context with HTTP-Header-Format into the metadatawriter.
	err := sg.Tracer.Inject(clientSpan.Context(), opentracing.HTTPHeaders, mdWriter)
	if err != nil {
		log.Printf("Tracer.Inject() failed: %v", err)
	}
	//Step 4: Call the successor service and create a new Context including the metadata in GRPC wire format.
	//We ignore the result, because it's just an Empty struct.
	_, err = unit.UnitClient.Call(metadata.NewOutgoingContext(ctx, md), &api.Empty{})
	if err != nil {
		log.Printf("Failed remote call to successor at %s, error was %v", unit.Unit.GetInvokedHostPort(), err)
	}
	//Step 5: Finish local client Span
	err = sg.finishAndReportTimedOTSpan(spanStart, clientSpan, childSpanNum, reporter)
	if err != nil {
		log.Printf("Error in finishing span: %v", clientSpan)
	}
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
				go sg.DoUnitCalls(context.Background(), reporter)
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
