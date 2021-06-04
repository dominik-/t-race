package worker

import (
	"context"
	"log"
	"time"

	"github.com/dominik-/t-race/api"
	"github.com/golang/protobuf/ptypes"
	"github.com/opentracing/opentracing-go"
	otlog "github.com/opentracing/opentracing-go/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

type Unit interface {
	Invoke(context.Context, opentracing.Tracer)
	ExtractIncomingMetadata(context.Context, opentracing.Tracer) (opentracing.SpanContext, error)
	StartContext(opentracing.Tracer, opentracing.SpanContext, context.Context) (opentracing.Span, context.Context)
	EmulateWork()
	AddContextMetadata(opentracing.Span)
	Next(context.Context, opentracing.Span, opentracing.Tracer)
	CloseContext(opentracing.Span)
	GetLoadPercentage() float64
	SetWeight(int64)
	GetWeight() int64
}

type UnitExecutor struct {
	data             *api.Unit
	syncCount        int
	syncSet          map[string]int
	WorkSampler      DistributionSampler
	SuccessorClients map[string]*grpc.ClientConn
	Tags             map[string]string
	Baggage          map[string]string
	Logs             map[string]string
	Worker           *Worker
	Weight           int64
}

func CreateUnitExecutorFromConfig(unitConfig *api.Unit, workerConfig *Worker) (*UnitExecutor, error) {
	dist, err := LookupDistribution(unitConfig.WorkBefore)
	if err != nil {
		//surface error from parsing the distribution
		return nil, err
	}
	// Create TLS credentials for grpc clients that skip root CA verification
	/* 	creds := credentials.NewTLS(&tls.Config{
	   		InsecureSkipVerify: true,
	   	})
		option := grpc.WithTransportCredentials(creds) */
	clientConnections := make(map[string]*grpc.ClientConn)
	for _, successor := range unitConfig.Successors {
		if successor.IsRemote {
			conn, err := grpc.Dial(successor.HostPort, grpc.WithInsecure())
			if err != nil {
				return nil, err
			}
			clientConnections[successor.ServiceId] = conn
		}
	}
	var tags map[string]string
	var baggage map[string]string
	var logs map[string]string
	if unitConfig.Context != nil {
		if unitConfig.Context.Tags != nil {
			tags = generateStringMap(unitConfig.Context.Tags)
		}
		if unitConfig.Context.Baggage != nil {
			baggage = generateStringMap(unitConfig.Context.Baggage)
		}
		if unitConfig.Context.Logs != nil {
			logs = generateStringMap(unitConfig.Context.Logs)
		}
	}

	return &UnitExecutor{
		data:             unitConfig,
		syncCount:        len(unitConfig.Inputs),
		syncSet:          make(map[string]int),
		WorkSampler:      dist,
		SuccessorClients: clientConnections,
		Tags:             tags,
		Baggage:          baggage,
		Logs:             logs,
		Worker:           workerConfig,
	}, nil
}

func (executor *UnitExecutor) Invoke(ctx context.Context, tracer opentracing.Tracer) {
	//Assumption: at this point we always have a context
	spanCtx, err := executor.ExtractIncomingMetadata(ctx, tracer)
	if err != nil {
		log.Fatalf("Couldn't extract metadata, please check format. Data was: %v", ctx)
	}
	spanStart := time.Now()
	span, ctxNew := executor.StartContext(tracer, spanCtx, ctx)
	executor.AddContextMetadata(span)
	executor.EmulateWork()
	executor.Next(ctxNew, span, tracer)
	sampled := getSampledFlag(span.Context())
	traceID := getTraceIdAsBytes(span.Context())
	spanID := getSpanID(span.Context())
	executor.CloseContext(span)
	finishTimeDelta := time.Since(spanStart)
	executor.Worker.SpanDurationHist.Observe(float64(finishTimeDelta.Nanoseconds() / 1000.0))
	started, err := ptypes.TimestampProto(spanStart)
	finished, err := ptypes.TimestampProto(spanStart.Add(finishTimeDelta))
	if err != nil {
		log.Fatalf("Couldn't convert timestamps to proto format.")
	}
	go executor.Worker.Reporter.Collect(&api.Result{
		TraceId:    traceID,
		SpanId:     spanID,
		StartTime:  started,
		FinishTime: finished,
		Sampled:    sampled,
	})
}

func (executor *UnitExecutor) ExtractIncomingMetadata(ctx context.Context, tracer opentracing.Tracer) (opentracing.SpanContext, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		md = metadata.New(nil)
	}
	remoteContext, err := tracer.Extract(opentracing.HTTPHeaders, metadataReaderWriter{md})
	//if there is no span context to be found in headers, were fine actually, because it means that this is a root span. Could do an additional check for that here.
	if err != nil && err != opentracing.ErrSpanContextNotFound {
		return nil, err
	}
	return remoteContext, nil
}

func (executor *UnitExecutor) StartContext(tracer opentracing.Tracer, spancontext opentracing.SpanContext, ctx context.Context) (opentracing.Span, context.Context) {
	return opentracing.StartSpanFromContextWithTracer(ctx, tracer, executor.data.Identifier, mapOpenTracingRelationshipType(executor.data.RelType, spancontext))
	//var span opentracing.Span
	//spanStart := time.Now() TODO: do local measurements
	//if executor.data.ThroughputRatio != 0.0 {
	//start local "parent" span as root span
	//span = tracer.StartSpan(executor.data.Identifier)
	//} else {
	//start local span with relationship indicator
	//	relationshipTypeOption := mapOpenTracingRelationshipType(executor.data.RelType, spanContext)
	//	span = tracer.StartSpan(executor.data.Identifier, relationshipTypeOption)
	//}
	//return span
}

func mapOpenTracingRelationshipType(relType api.RelationshipType, spanContext opentracing.SpanContext) opentracing.StartSpanOption {
	switch relType {
	case api.RelationshipType_FOLLOWS:
		return opentracing.FollowsFrom(spanContext)
	default:
		return opentracing.ChildOf(spanContext)
	}
}

func (executor *UnitExecutor) EmulateWork() {
	<-time.NewTimer(executor.WorkSampler.GetNextValue()).C
}

func (executor *UnitExecutor) AddContextMetadata(span opentracing.Span) {
	for k, v := range executor.Tags {
		span.SetTag(k, v)
	}
	for k, v := range executor.Baggage {
		span.SetBaggageItem(k, v)
	}
	for k, v := range executor.Logs {
		//we chose the LogFields method here over logKV
		span.LogFields(
			otlog.String(k, v),
		)
	}
}

func (executor *UnitExecutor) Next(ctx context.Context, span opentracing.Span, tracer opentracing.Tracer) {
	//for each successor we have 4 different cases: remote or local, req-resp or fire and forget
	//var ctxNew context.Context
	for _, successor := range executor.data.Successors {
		localClientSpan, ctxNew := opentracing.StartSpanFromContextWithTracer(ctx, tracer, "invoke-"+successor.UnitId, mapOpenTracingRelationshipType(executor.data.RelType, span.Context()))
		//localClientSpan := tracer.StartSpan("invoke-"+successor.UnitId, mapOpenTracingRelationshipType(api.RelationshipType_CHILD, span.Context()))
		//ctxNew = opentracing.ContextWithSpan(ctx, localClientSpan)
		if successor.IsRemote {
			md, ok := metadata.FromOutgoingContext(ctxNew)
			if !ok {
				md = metadata.New(nil)
			} else {
				md = md.Copy()
			}
			mdWriter := metadataReaderWriter{md}
			//Step 3b: Inject the local span context with HTTP-Header-Format into the metadatawriter.
			err := tracer.Inject(localClientSpan.Context(), opentracing.HTTPHeaders, mdWriter)
			if err != nil {
				log.Printf("Tracer.Inject() failed: %v", err)
			}
			if successor.Sync {
				//Step 3a: Use context ("outgoing" is from the perspective of the calling service!) and create a metadata writer;
				api.NewBenchmarkWorkerClient(executor.SuccessorClients[successor.ServiceId]).Call(metadata.NewOutgoingContext(ctxNew, md), &api.DispatchId{UnitReference: successor.UnitId})
			} else {
				go api.NewBenchmarkWorkerClient(executor.SuccessorClients[successor.ServiceId]).Call(metadata.NewOutgoingContext(ctxNew, md), &api.DispatchId{UnitReference: successor.UnitId})
			}
		} else {
			if successor.Sync {
				executor.Worker.UnitExecutorMap[successor.UnitId].Invoke(ctxNew, tracer)
			} else {
				go executor.Worker.UnitExecutorMap[successor.UnitId].Invoke(ctxNew, tracer)
			}
		}
		localClientSpan.Finish()
	}
}

func (executor *UnitExecutor) CloseContext(span opentracing.Span) {
	span.Finish()
}

//Turns templates for tags and baggage into a map of strings to strings.
func generateStringMap(templates []*api.KeyValueTemplate) map[string]string {
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

func (executor *UnitExecutor) GetLoadPercentage() float64 {
	return executor.data.ThroughputRatio
}

func (executor *UnitExecutor) SetWeight(w int64) {
	executor.Weight = w
}

func (executor *UnitExecutor) GetWeight() int64 {
	return executor.Weight
}
