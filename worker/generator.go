package worker

import (
	"context"
	"fmt"
	"io"
	"log"
	"time"

	"github.com/opentracing/opentracing-go"
	"gitlab.tubit.tu-berlin.de/dominik-ernst/tracer-benchmarks/api"

	jaegercfg "github.com/uber/jaeger-client-go/config"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

type OpenTracingSpanGenerator struct {
	Counter        int64
	Tracer         opentracing.Tracer
	Closer         io.Closer
	ResultsChannel chan *api.Result
	Units          []*GeneratedUnit
	Tags           map[string]string
	Baggage        map[string]string
	WorkFinalDist  DistributionSampler
	ServiceName    string
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
	DoUnitCalls(context.Context, []ResultReporter) bool
	GetTracer() opentracing.Tracer
	GetResultsChannel() chan *api.Result
	WriteSpansUntilExitSignal(<-chan bool, time.Duration, ...ResultReporter) chan bool
}

func (sg *OpenTracingSpanGenerator) GetTracer() opentracing.Tracer {
	return sg.Tracer
}

func (sg *OpenTracingSpanGenerator) GetResultsChannel() chan *api.Result {
	return sg.ResultsChannel
}

// InitTracer returns an instance of a Tracer that samples 100% of traces and logs all spans to stdout.
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

func NewOpenTracingSpanGenerator(tracer opentracing.Tracer, closer io.Closer, config *api.WorkerConfiguration) SpanGenerator {
	// initialize empty generator
	generator := &OpenTracingSpanGenerator{
		Units:  make([]*GeneratedUnit, 0),
		Tracer: tracer,
		Closer: closer,
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
		//TODO: is this done right to create traces?
		/* conn, err := grpc.Dial(unit.InvokedHostPort, securityOption,
			grpc.WithUnaryInterceptor(otgrpc.OpenTracingClientInterceptor(generator.Tracer, otgrpc.SpanDecorator(generator.GetClientDecorator))),
		) */
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
		generator.Tags = make(map[string]string, len(config.Context.Tags))
		generator.Baggage = make(map[string]string, len(config.Context.Baggage))
		for _, tagTemplate := range config.Context.Tags {
			generator.Tags[RandStringWithLength(tagTemplate.KeyByteLength)] = RandStringWithLength(tagTemplate.ValueByteLength)
		}
		for _, tagTemplate := range config.Context.Baggage {
			generator.Baggage[RandStringWithLength(tagTemplate.KeyByteLength)] = RandStringWithLength(tagTemplate.ValueByteLength)
		}
	}
	generator.ResultsChannel = make(chan *api.Result)

	return generator
}

//Deprecated: we don't use the generator anymore, spans, tags and baggage are created directly.
func (sg *OpenTracingSpanGenerator) GetClientDecorator(span opentracing.Span, method string, req, resp interface{}, grpcError error) {
	for tagKey, tagValue := range sg.Tags {
		span.SetTag(tagKey, tagValue)
	}
	for tagKey, tagValue := range sg.Baggage {
		span.SetBaggageItem(tagKey, tagValue)
	}
}

func (sg *OpenTracingSpanGenerator) DoUnitCalls(parent context.Context, reporters []ResultReporter) bool {
	//get SpanContext from golang Context
	//remoteClientSpan := opentracing.SpanFromContext(parent)
	md, ok := metadata.FromIncomingContext(parent)
	if !ok {
		md = metadata.New(nil)
	}
	//TODO create result and append result data over the call
	remoteContext, err := sg.Tracer.Extract(opentracing.HTTPHeaders, metadataReaderWriter{md})
	//create child relationship to client span - TODO: does that always make sense?
	var serverSpan opentracing.Span
	var ctx context.Context
	if err != nil && err == opentracing.ErrSpanContextNotFound {
		//start local "parent" span
		serverSpan = sg.Tracer.StartSpan(sg.ServiceName + "-parent")
		ctx = opentracing.ContextWithSpan(context.Background(), serverSpan)
	} else if err != nil {
		log.Fatalf("Couldn't parse Span Context! Error was: %v", err)
	} else {
		//start local span with child relationship to parent from remote context
		option := opentracing.ChildOf(remoteContext)
		serverSpan = sg.Tracer.StartSpan(sg.ServiceName+"-parent", option)
		ctx = opentracing.ContextWithSpan(parent, serverSpan)
	}
	//we add this services tags and baggage once
	for tagKey, tagValue := range sg.Tags {
		serverSpan.SetTag(tagKey, tagValue)
	}
	for tagKey, tagValue := range sg.Baggage {
		serverSpan.SetBaggageItem(tagKey, tagValue)
	}

	for i, unit := range sg.Units {
		//Step 1: wait for internal work emulation before each call; TODO: how long does this type check take? probably bad to do it every call.

		//Step 2: call successor
		//TODO: stuff is added automatically by the unary interceptor specified upon creation of the GRPC client; maybe we need to do our own injection at some point.
		//Step 1: create client-side span for calling the successor service, including relationship to current context.
		var relOption opentracing.SpanReference
		switch unit.Unit.GetRelType() {
		case api.RelationshipType_CHILD:
			relOption = opentracing.ChildOf(serverSpan.Context())
		case api.RelationshipType_FOLLOWING:
			relOption = opentracing.FollowsFrom(serverSpan.Context())
		}
		clientSpanName := fmt.Sprintf("%s-call-%d", sg.ServiceName, i)
		localClientSpan := sg.Tracer.StartSpan(clientSpanName, relOption)
		//Update context with reference to new client Span
		ctx = opentracing.ContextWithSpan(ctx, localClientSpan)
		//Step 2: wait for internal work emulation before the actual call; TODO: how long does this type check take? bad to do it every call?
		if _, parsed := unit.WorkSampler.(*NoDistribution); !parsed {
			//if the Distribution cannot be parsed to the NoDistribution, wait for the sampled amount of time.
			<-time.NewTimer(unit.WorkSampler.GetNextValue()).C
		}
		//Step 3a: Use context (incoming = outgoing here) and create a metadata writer;
		md, ok := metadata.FromOutgoingContext(ctx)
		if !ok {
			md = metadata.New(nil)
		} else {
			md = md.Copy()
		}
		mdWriter := metadataReaderWriter{md}
		//Step 3b: Inject the local span context with HTTP-Header-Format into the metadatawriter.
		err := sg.Tracer.Inject(localClientSpan.Context(), opentracing.HTTPHeaders, mdWriter)
		if err != nil {
			log.Printf("Tracer.Inject() failed: %v", err)
		}
		//Step 4: Call the successor service and create a new Context including the metadata in GRPC wire format.
		_, err = unit.UnitClient.Call(metadata.NewOutgoingContext(ctx, md), &api.Empty{})
		//TODO currently we just return false for each failed request
		if err != nil {
			return false
		}
		//Step 5: Finish local client Span
		localClientSpan.Finish()
	}
	//Do final work locally.
	if _, parsed := sg.WorkFinalDist.(*NoDistribution); !parsed {
		<-time.NewTimer(sg.WorkFinalDist.GetNextValue()).C
	}
	//Finish parent span
	serverSpan.Finish()
	//TODO return a result! Report the result to all reporters? Currently reports are not used here.
	return true
}

//WriteSpansUntilExitSignal takes a SpanGenerator, an exitSignalChannel and reporters.
//It loops writing spans to the IntervalTraceWriter until the exitSignalChannel receives a value.
//All reporters are invoked periodically every second. The returned channel can be used to listen for the termination of the write-loop.
func (sg *OpenTracingSpanGenerator) WriteSpansUntilExitSignal(exitSignalChannel <-chan bool, interval time.Duration,
	periodicReporters ...ResultReporter) chan bool {
	finishedIndicator := make(chan bool, 1)
	go func() {
		ticker := time.NewTicker(interval)
		//TODO make report interval configurable
		reportTicker := time.NewTicker(1 * time.Second)
	GenerateLoop:
		for {
			select {
			case <-exitSignalChannel:
				//log.Println("Received shutdown signal at write Span loop.")
				break GenerateLoop
			case <-ticker.C:
				//write span (async) to the writer, generate new parent context
				go sg.DoUnitCalls(context.Background(), periodicReporters)
				break
			case <-reportTicker.C:
				for _, reporter := range periodicReporters {
					go reporter.Collect(sg)
				}

			}
		}
		for _, reporter := range periodicReporters {
			go reporter.Collect(sg)
		}
		//by sending to this channel, we indicate a regular shutdown.
		//log.Println("Sending finished signal from write Span loop to worker loop.")
		finishedIndicator <- true
		close(finishedIndicator)
	}()

	return finishedIndicator
}
