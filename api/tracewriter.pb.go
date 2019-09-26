// Code generated by protoc-gen-go. DO NOT EDIT.
// source: tracewriter.proto

package api

import (
	context "context"
	fmt "fmt"
	proto "github.com/golang/protobuf/proto"
	timestamp "github.com/golang/protobuf/ptypes/timestamp"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
	math "math"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion3 // please upgrade the proto package

type RelationshipType int32

const (
	RelationshipType_CHILD     RelationshipType = 0
	RelationshipType_FOLLOWING RelationshipType = 1
)

var RelationshipType_name = map[int32]string{
	0: "CHILD",
	1: "FOLLOWING",
}

var RelationshipType_value = map[string]int32{
	"CHILD":     0,
	"FOLLOWING": 1,
}

func (x RelationshipType) String() string {
	return proto.EnumName(RelationshipType_name, int32(x))
}

func (RelationshipType) EnumDescriptor() ([]byte, []int) {
	return fileDescriptor_af9f2570be55f477, []int{0}
}

type WorkerConfiguration struct {
	WorkerId      string `protobuf:"bytes,1,opt,name=worker_id,json=workerId,proto3" json:"worker_id,omitempty"`
	OperationName string `protobuf:"bytes,2,opt,name=operation_name,json=operationName,proto3" json:"operation_name,omitempty"`
	//root indicates whether the worker is generating load directly and independently (root=true), or listens to requests (through the Call method).
	Root             bool  `protobuf:"varint,3,opt,name=root,proto3" json:"root,omitempty"`
	RuntimeSeconds   int64 `protobuf:"varint,4,opt,name=runtime_seconds,json=runtimeSeconds,proto3" json:"runtime_seconds,omitempty"`
	TargetThroughput int64 `protobuf:"varint,5,opt,name=target_throughput,json=targetThroughput,proto3" json:"target_throughput,omitempty"`
	//the sink is the backend address to send traces to, i.e. an endpoint of an opentracing-compatible tracer
	SinkHostPort string           `protobuf:"bytes,6,opt,name=sink_host_port,json=sinkHostPort,proto3" json:"sink_host_port,omitempty"`
	Context      *ContextTemplate `protobuf:"bytes,7,opt,name=context,proto3" json:"context,omitempty"`
	//a final unit of local "work", also used if no subsequent calls are done by this worker
	WorkFinal            *Work    `protobuf:"bytes,8,opt,name=work_final,json=workFinal,proto3" json:"work_final,omitempty"`
	Units                []*Unit  `protobuf:"bytes,9,rep,name=units,proto3" json:"units,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *WorkerConfiguration) Reset()         { *m = WorkerConfiguration{} }
func (m *WorkerConfiguration) String() string { return proto.CompactTextString(m) }
func (*WorkerConfiguration) ProtoMessage()    {}
func (*WorkerConfiguration) Descriptor() ([]byte, []int) {
	return fileDescriptor_af9f2570be55f477, []int{0}
}

func (m *WorkerConfiguration) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_WorkerConfiguration.Unmarshal(m, b)
}
func (m *WorkerConfiguration) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_WorkerConfiguration.Marshal(b, m, deterministic)
}
func (m *WorkerConfiguration) XXX_Merge(src proto.Message) {
	xxx_messageInfo_WorkerConfiguration.Merge(m, src)
}
func (m *WorkerConfiguration) XXX_Size() int {
	return xxx_messageInfo_WorkerConfiguration.Size(m)
}
func (m *WorkerConfiguration) XXX_DiscardUnknown() {
	xxx_messageInfo_WorkerConfiguration.DiscardUnknown(m)
}

var xxx_messageInfo_WorkerConfiguration proto.InternalMessageInfo

func (m *WorkerConfiguration) GetWorkerId() string {
	if m != nil {
		return m.WorkerId
	}
	return ""
}

func (m *WorkerConfiguration) GetOperationName() string {
	if m != nil {
		return m.OperationName
	}
	return ""
}

func (m *WorkerConfiguration) GetRoot() bool {
	if m != nil {
		return m.Root
	}
	return false
}

func (m *WorkerConfiguration) GetRuntimeSeconds() int64 {
	if m != nil {
		return m.RuntimeSeconds
	}
	return 0
}

func (m *WorkerConfiguration) GetTargetThroughput() int64 {
	if m != nil {
		return m.TargetThroughput
	}
	return 0
}

func (m *WorkerConfiguration) GetSinkHostPort() string {
	if m != nil {
		return m.SinkHostPort
	}
	return ""
}

func (m *WorkerConfiguration) GetContext() *ContextTemplate {
	if m != nil {
		return m.Context
	}
	return nil
}

func (m *WorkerConfiguration) GetWorkFinal() *Work {
	if m != nil {
		return m.WorkFinal
	}
	return nil
}

func (m *WorkerConfiguration) GetUnits() []*Unit {
	if m != nil {
		return m.Units
	}
	return nil
}

//Unit captures a request-response interaction with another emulated service.
type Unit struct {
	RelType RelationshipType `protobuf:"varint,1,opt,name=rel_type,json=relType,proto3,enum=api.RelationshipType" json:"rel_type,omitempty"`
	//This is sampled and waited for before the call is dispatched to the next worker listening at invoked_host_port.
	WorkBefore           *Work    `protobuf:"bytes,2,opt,name=work_before,json=workBefore,proto3" json:"work_before,omitempty"`
	InvokedHostPort      string   `protobuf:"bytes,3,opt,name=invoked_host_port,json=invokedHostPort,proto3" json:"invoked_host_port,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *Unit) Reset()         { *m = Unit{} }
func (m *Unit) String() string { return proto.CompactTextString(m) }
func (*Unit) ProtoMessage()    {}
func (*Unit) Descriptor() ([]byte, []int) {
	return fileDescriptor_af9f2570be55f477, []int{1}
}

func (m *Unit) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Unit.Unmarshal(m, b)
}
func (m *Unit) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Unit.Marshal(b, m, deterministic)
}
func (m *Unit) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Unit.Merge(m, src)
}
func (m *Unit) XXX_Size() int {
	return xxx_messageInfo_Unit.Size(m)
}
func (m *Unit) XXX_DiscardUnknown() {
	xxx_messageInfo_Unit.DiscardUnknown(m)
}

var xxx_messageInfo_Unit proto.InternalMessageInfo

func (m *Unit) GetRelType() RelationshipType {
	if m != nil {
		return m.RelType
	}
	return RelationshipType_CHILD
}

func (m *Unit) GetWorkBefore() *Work {
	if m != nil {
		return m.WorkBefore
	}
	return nil
}

func (m *Unit) GetInvokedHostPort() string {
	if m != nil {
		return m.InvokedHostPort
	}
	return ""
}

type Work struct {
	DistType             string             `protobuf:"bytes,1,opt,name=dist_type,json=distType,proto3" json:"dist_type,omitempty"`
	Parameters           map[string]float64 `protobuf:"bytes,2,rep,name=parameters,proto3" json:"parameters,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"fixed64,2,opt,name=value,proto3"`
	XXX_NoUnkeyedLiteral struct{}           `json:"-"`
	XXX_unrecognized     []byte             `json:"-"`
	XXX_sizecache        int32              `json:"-"`
}

func (m *Work) Reset()         { *m = Work{} }
func (m *Work) String() string { return proto.CompactTextString(m) }
func (*Work) ProtoMessage()    {}
func (*Work) Descriptor() ([]byte, []int) {
	return fileDescriptor_af9f2570be55f477, []int{2}
}

func (m *Work) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Work.Unmarshal(m, b)
}
func (m *Work) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Work.Marshal(b, m, deterministic)
}
func (m *Work) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Work.Merge(m, src)
}
func (m *Work) XXX_Size() int {
	return xxx_messageInfo_Work.Size(m)
}
func (m *Work) XXX_DiscardUnknown() {
	xxx_messageInfo_Work.DiscardUnknown(m)
}

var xxx_messageInfo_Work proto.InternalMessageInfo

func (m *Work) GetDistType() string {
	if m != nil {
		return m.DistType
	}
	return ""
}

func (m *Work) GetParameters() map[string]float64 {
	if m != nil {
		return m.Parameters
	}
	return nil
}

type TagTemplate struct {
	KeyByteLength        int64    `protobuf:"varint,1,opt,name=key_byte_length,json=keyByteLength,proto3" json:"key_byte_length,omitempty"`
	ValueByteLength      int64    `protobuf:"varint,2,opt,name=value_byte_length,json=valueByteLength,proto3" json:"value_byte_length,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *TagTemplate) Reset()         { *m = TagTemplate{} }
func (m *TagTemplate) String() string { return proto.CompactTextString(m) }
func (*TagTemplate) ProtoMessage()    {}
func (*TagTemplate) Descriptor() ([]byte, []int) {
	return fileDescriptor_af9f2570be55f477, []int{3}
}

func (m *TagTemplate) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_TagTemplate.Unmarshal(m, b)
}
func (m *TagTemplate) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_TagTemplate.Marshal(b, m, deterministic)
}
func (m *TagTemplate) XXX_Merge(src proto.Message) {
	xxx_messageInfo_TagTemplate.Merge(m, src)
}
func (m *TagTemplate) XXX_Size() int {
	return xxx_messageInfo_TagTemplate.Size(m)
}
func (m *TagTemplate) XXX_DiscardUnknown() {
	xxx_messageInfo_TagTemplate.DiscardUnknown(m)
}

var xxx_messageInfo_TagTemplate proto.InternalMessageInfo

func (m *TagTemplate) GetKeyByteLength() int64 {
	if m != nil {
		return m.KeyByteLength
	}
	return 0
}

func (m *TagTemplate) GetValueByteLength() int64 {
	if m != nil {
		return m.ValueByteLength
	}
	return 0
}

type Result struct {
	TraceId              []byte               `protobuf:"bytes,1,opt,name=trace_id,json=traceId,proto3" json:"trace_id,omitempty"`
	TraceNum             int64                `protobuf:"varint,2,opt,name=trace_num,json=traceNum,proto3" json:"trace_num,omitempty"`
	SpanId               []byte               `protobuf:"bytes,3,opt,name=span_id,json=spanId,proto3" json:"span_id,omitempty"`
	SpanNum              int64                `protobuf:"varint,4,opt,name=span_num,json=spanNum,proto3" json:"span_num,omitempty"`
	StartTime            *timestamp.Timestamp `protobuf:"bytes,5,opt,name=start_time,json=startTime,proto3" json:"start_time,omitempty"`
	FinishTime           *timestamp.Timestamp `protobuf:"bytes,6,opt,name=finish_time,json=finishTime,proto3" json:"finish_time,omitempty"`
	Sampled              bool                 `protobuf:"varint,7,opt,name=sampled,proto3" json:"sampled,omitempty"`
	XXX_NoUnkeyedLiteral struct{}             `json:"-"`
	XXX_unrecognized     []byte               `json:"-"`
	XXX_sizecache        int32                `json:"-"`
}

func (m *Result) Reset()         { *m = Result{} }
func (m *Result) String() string { return proto.CompactTextString(m) }
func (*Result) ProtoMessage()    {}
func (*Result) Descriptor() ([]byte, []int) {
	return fileDescriptor_af9f2570be55f477, []int{4}
}

func (m *Result) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Result.Unmarshal(m, b)
}
func (m *Result) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Result.Marshal(b, m, deterministic)
}
func (m *Result) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Result.Merge(m, src)
}
func (m *Result) XXX_Size() int {
	return xxx_messageInfo_Result.Size(m)
}
func (m *Result) XXX_DiscardUnknown() {
	xxx_messageInfo_Result.DiscardUnknown(m)
}

var xxx_messageInfo_Result proto.InternalMessageInfo

func (m *Result) GetTraceId() []byte {
	if m != nil {
		return m.TraceId
	}
	return nil
}

func (m *Result) GetTraceNum() int64 {
	if m != nil {
		return m.TraceNum
	}
	return 0
}

func (m *Result) GetSpanId() []byte {
	if m != nil {
		return m.SpanId
	}
	return nil
}

func (m *Result) GetSpanNum() int64 {
	if m != nil {
		return m.SpanNum
	}
	return 0
}

func (m *Result) GetStartTime() *timestamp.Timestamp {
	if m != nil {
		return m.StartTime
	}
	return nil
}

func (m *Result) GetFinishTime() *timestamp.Timestamp {
	if m != nil {
		return m.FinishTime
	}
	return nil
}

func (m *Result) GetSampled() bool {
	if m != nil {
		return m.Sampled
	}
	return false
}

type ContextTemplate struct {
	Tags                 []*TagTemplate `protobuf:"bytes,1,rep,name=tags,proto3" json:"tags,omitempty"`
	Baggage              []*TagTemplate `protobuf:"bytes,2,rep,name=baggage,proto3" json:"baggage,omitempty"`
	XXX_NoUnkeyedLiteral struct{}       `json:"-"`
	XXX_unrecognized     []byte         `json:"-"`
	XXX_sizecache        int32          `json:"-"`
}

func (m *ContextTemplate) Reset()         { *m = ContextTemplate{} }
func (m *ContextTemplate) String() string { return proto.CompactTextString(m) }
func (*ContextTemplate) ProtoMessage()    {}
func (*ContextTemplate) Descriptor() ([]byte, []int) {
	return fileDescriptor_af9f2570be55f477, []int{5}
}

func (m *ContextTemplate) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_ContextTemplate.Unmarshal(m, b)
}
func (m *ContextTemplate) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_ContextTemplate.Marshal(b, m, deterministic)
}
func (m *ContextTemplate) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ContextTemplate.Merge(m, src)
}
func (m *ContextTemplate) XXX_Size() int {
	return xxx_messageInfo_ContextTemplate.Size(m)
}
func (m *ContextTemplate) XXX_DiscardUnknown() {
	xxx_messageInfo_ContextTemplate.DiscardUnknown(m)
}

var xxx_messageInfo_ContextTemplate proto.InternalMessageInfo

func (m *ContextTemplate) GetTags() []*TagTemplate {
	if m != nil {
		return m.Tags
	}
	return nil
}

func (m *ContextTemplate) GetBaggage() []*TagTemplate {
	if m != nil {
		return m.Baggage
	}
	return nil
}

type ResultPackage struct {
	WorkerId             string    `protobuf:"bytes,1,opt,name=worker_id,json=workerId,proto3" json:"worker_id,omitempty"`
	EnvironmentId        string    `protobuf:"bytes,2,opt,name=environment_id,json=environmentId,proto3" json:"environment_id,omitempty"`
	Results              []*Result `protobuf:"bytes,3,rep,name=results,proto3" json:"results,omitempty"`
	XXX_NoUnkeyedLiteral struct{}  `json:"-"`
	XXX_unrecognized     []byte    `json:"-"`
	XXX_sizecache        int32     `json:"-"`
}

func (m *ResultPackage) Reset()         { *m = ResultPackage{} }
func (m *ResultPackage) String() string { return proto.CompactTextString(m) }
func (*ResultPackage) ProtoMessage()    {}
func (*ResultPackage) Descriptor() ([]byte, []int) {
	return fileDescriptor_af9f2570be55f477, []int{6}
}

func (m *ResultPackage) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_ResultPackage.Unmarshal(m, b)
}
func (m *ResultPackage) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_ResultPackage.Marshal(b, m, deterministic)
}
func (m *ResultPackage) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ResultPackage.Merge(m, src)
}
func (m *ResultPackage) XXX_Size() int {
	return xxx_messageInfo_ResultPackage.Size(m)
}
func (m *ResultPackage) XXX_DiscardUnknown() {
	xxx_messageInfo_ResultPackage.DiscardUnknown(m)
}

var xxx_messageInfo_ResultPackage proto.InternalMessageInfo

func (m *ResultPackage) GetWorkerId() string {
	if m != nil {
		return m.WorkerId
	}
	return ""
}

func (m *ResultPackage) GetEnvironmentId() string {
	if m != nil {
		return m.EnvironmentId
	}
	return ""
}

func (m *ResultPackage) GetResults() []*Result {
	if m != nil {
		return m.Results
	}
	return nil
}

type Empty struct {
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *Empty) Reset()         { *m = Empty{} }
func (m *Empty) String() string { return proto.CompactTextString(m) }
func (*Empty) ProtoMessage()    {}
func (*Empty) Descriptor() ([]byte, []int) {
	return fileDescriptor_af9f2570be55f477, []int{7}
}

func (m *Empty) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Empty.Unmarshal(m, b)
}
func (m *Empty) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Empty.Marshal(b, m, deterministic)
}
func (m *Empty) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Empty.Merge(m, src)
}
func (m *Empty) XXX_Size() int {
	return xxx_messageInfo_Empty.Size(m)
}
func (m *Empty) XXX_DiscardUnknown() {
	xxx_messageInfo_Empty.DiscardUnknown(m)
}

var xxx_messageInfo_Empty proto.InternalMessageInfo

func init() {
	proto.RegisterEnum("api.RelationshipType", RelationshipType_name, RelationshipType_value)
	proto.RegisterType((*WorkerConfiguration)(nil), "api.WorkerConfiguration")
	proto.RegisterType((*Unit)(nil), "api.Unit")
	proto.RegisterType((*Work)(nil), "api.Work")
	proto.RegisterMapType((map[string]float64)(nil), "api.Work.ParametersEntry")
	proto.RegisterType((*TagTemplate)(nil), "api.TagTemplate")
	proto.RegisterType((*Result)(nil), "api.Result")
	proto.RegisterType((*ContextTemplate)(nil), "api.ContextTemplate")
	proto.RegisterType((*ResultPackage)(nil), "api.ResultPackage")
	proto.RegisterType((*Empty)(nil), "api.Empty")
}

func init() { proto.RegisterFile("tracewriter.proto", fileDescriptor_af9f2570be55f477) }

var fileDescriptor_af9f2570be55f477 = []byte{
	// 809 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x84, 0x54, 0xcd, 0x8e, 0xdb, 0x36,
	0x10, 0x8e, 0xfc, 0x27, 0x7b, 0x9c, 0x5d, 0x7b, 0xd9, 0x14, 0x55, 0xdc, 0x43, 0x0c, 0x21, 0x6d,
	0x8d, 0x6d, 0xe1, 0x04, 0xdb, 0x4b, 0xd3, 0xa2, 0x28, 0xb0, 0xdb, 0xa4, 0x31, 0xb0, 0xd8, 0x2c,
	0x18, 0x17, 0x39, 0x0a, 0xb4, 0x3d, 0x96, 0x05, 0x4b, 0xa4, 0x40, 0x51, 0x9b, 0xaa, 0x0f, 0xd0,
	0x73, 0x5f, 0xa0, 0x6f, 0xd7, 0x07, 0x29, 0x38, 0x94, 0x1c, 0x67, 0x7b, 0xc8, 0x8d, 0xf3, 0x7d,
	0xdf, 0x0c, 0x87, 0xf3, 0x43, 0x38, 0x33, 0x5a, 0xac, 0xf1, 0xbd, 0x4e, 0x0c, 0xea, 0x79, 0xae,
	0x95, 0x51, 0xac, 0x2d, 0xf2, 0x64, 0xf2, 0x24, 0x56, 0x2a, 0x4e, 0xf1, 0x19, 0x41, 0xab, 0x72,
	0xfb, 0xcc, 0x24, 0x19, 0x16, 0x46, 0x64, 0xb9, 0x53, 0x85, 0xff, 0xb6, 0xe0, 0xb3, 0x77, 0x4a,
	0xef, 0x51, 0x5f, 0x29, 0xb9, 0x4d, 0xe2, 0x52, 0x0b, 0x93, 0x28, 0xc9, 0xbe, 0x84, 0xc1, 0x7b,
	0x82, 0xa3, 0x64, 0x13, 0x78, 0x53, 0x6f, 0x36, 0xe0, 0x7d, 0x07, 0x2c, 0x36, 0xec, 0x2b, 0x38,
	0x55, 0x39, 0x3a, 0x65, 0x24, 0x45, 0x86, 0x41, 0x8b, 0x14, 0x27, 0x07, 0xf4, 0x46, 0x64, 0xc8,
	0x18, 0x74, 0xb4, 0x52, 0x26, 0x68, 0x4f, 0xbd, 0x59, 0x9f, 0xd3, 0x99, 0x7d, 0x03, 0x23, 0x5d,
	0x4a, 0x9b, 0x45, 0x54, 0xe0, 0x5a, 0xc9, 0x4d, 0x11, 0x74, 0xa6, 0xde, 0xac, 0xcd, 0x4f, 0x6b,
	0xf8, 0xad, 0x43, 0xd9, 0xb7, 0x70, 0x66, 0x84, 0x8e, 0xd1, 0x44, 0x66, 0xa7, 0x55, 0x19, 0xef,
	0xf2, 0xd2, 0x04, 0x5d, 0x92, 0x8e, 0x1d, 0xb1, 0x3c, 0xe0, 0xec, 0x29, 0x9c, 0x16, 0x89, 0xdc,
	0x47, 0x3b, 0x55, 0x98, 0x28, 0x57, 0xda, 0x04, 0x3d, 0x4a, 0xe8, 0xa1, 0x45, 0x5f, 0xab, 0xc2,
	0xdc, 0x2a, 0x6d, 0xd8, 0x1c, 0xfc, 0xb5, 0x92, 0x06, 0xff, 0x30, 0x81, 0x3f, 0xf5, 0x66, 0xc3,
	0x8b, 0x47, 0x73, 0x91, 0x27, 0xf3, 0x2b, 0x87, 0x2d, 0x31, 0xcb, 0x53, 0x61, 0x90, 0x37, 0x22,
	0x36, 0x03, 0xb0, 0x4f, 0x8e, 0xb6, 0x89, 0x14, 0x69, 0xd0, 0x27, 0x97, 0x01, 0xb9, 0xd8, 0x8a,
	0x71, 0x2a, 0xd0, 0x2b, 0xcb, 0xb1, 0x27, 0xd0, 0x2d, 0x65, 0x62, 0x8a, 0x60, 0x30, 0x6d, 0x1f,
	0x44, 0xbf, 0xcb, 0xc4, 0x70, 0x87, 0x87, 0x7f, 0x7b, 0xd0, 0xb1, 0x36, 0x7b, 0x0e, 0x7d, 0x8d,
	0x69, 0x64, 0xaa, 0x1c, 0xa9, 0xac, 0xa7, 0x17, 0x9f, 0x93, 0x98, 0x63, 0x4a, 0x85, 0x2b, 0x76,
	0x49, 0xbe, 0xac, 0x72, 0xe4, 0xbe, 0xc6, 0xd4, 0x1e, 0xd8, 0x39, 0x0c, 0x29, 0x8b, 0x15, 0x6e,
	0x95, 0x76, 0x95, 0xfe, 0x28, 0x0d, 0xca, 0xf1, 0x92, 0x48, 0x76, 0x0e, 0x67, 0x89, 0xbc, 0x53,
	0x7b, 0xdc, 0x1c, 0x95, 0xa2, 0x4d, 0xa5, 0x18, 0xd5, 0x44, 0x53, 0x8d, 0xf0, 0x1f, 0x0f, 0x3a,
	0x36, 0x80, 0x6d, 0xf5, 0x26, 0x29, 0xcc, 0x87, 0x9c, 0x06, 0xbc, 0x6f, 0x01, 0xba, 0xfd, 0x05,
	0x40, 0x2e, 0xb4, 0xc8, 0xd0, 0xa0, 0x2e, 0x82, 0x16, 0x3d, 0xef, 0xf1, 0xe1, 0xf2, 0xf9, 0xed,
	0x81, 0x7b, 0x29, 0x8d, 0xae, 0xf8, 0x91, 0x78, 0xf2, 0x33, 0x8c, 0xee, 0xd1, 0x6c, 0x0c, 0xed,
	0x3d, 0x56, 0xf5, 0x25, 0xf6, 0xc8, 0x1e, 0x41, 0xf7, 0x4e, 0xa4, 0xa5, 0x7b, 0x97, 0xc7, 0x9d,
	0xf1, 0x63, 0xeb, 0x07, 0x2f, 0x14, 0x30, 0x5c, 0x8a, 0xb8, 0xe9, 0x0a, 0xfb, 0x1a, 0x46, 0x7b,
	0xac, 0xa2, 0x55, 0x65, 0x30, 0x4a, 0x51, 0xc6, 0x66, 0x47, 0x61, 0xda, 0xfc, 0x64, 0x8f, 0xd5,
	0x65, 0x65, 0xf0, 0x9a, 0x40, 0x5b, 0x02, 0x8a, 0xf1, 0x91, 0xb2, 0x45, 0xca, 0x11, 0x11, 0x1f,
	0xb4, 0xe1, 0x5f, 0x2d, 0xe8, 0x71, 0x2c, 0xca, 0xd4, 0xb0, 0xc7, 0xd0, 0xa7, 0x15, 0x6a, 0xc6,
	0xfd, 0x21, 0xf7, 0xc9, 0x5e, 0x6c, 0x6c, 0x7d, 0x1c, 0x25, 0xcb, 0xac, 0x8e, 0xe4, 0xb4, 0x37,
	0x65, 0xc6, 0xbe, 0x00, 0xbf, 0xc8, 0x85, 0xb4, 0x6e, 0x6d, 0x72, 0xeb, 0x59, 0x73, 0xb1, 0xb1,
	0x01, 0x89, 0xb0, 0x4e, 0x6e, 0xc2, 0x49, 0x68, 0x7d, 0x5e, 0x00, 0x14, 0x46, 0x68, 0x13, 0xd9,
	0x79, 0xa7, 0x99, 0x1e, 0x5e, 0x4c, 0xe6, 0x6e, 0x53, 0xe7, 0xcd, 0xa6, 0xce, 0x97, 0xcd, 0xa6,
	0xf2, 0x01, 0xa9, 0xad, 0xcd, 0x7e, 0x82, 0xe1, 0x36, 0x91, 0x49, 0xb1, 0x73, 0xbe, 0xbd, 0x4f,
	0xfa, 0x82, 0x93, 0x93, 0x73, 0x00, 0x7e, 0x21, 0xb2, 0x3c, 0xc5, 0x0d, 0xcd, 0x7f, 0x9f, 0x37,
	0x66, 0xb8, 0x86, 0xd1, 0xbd, 0x2d, 0x60, 0x4f, 0xa1, 0x63, 0x44, 0x5c, 0x04, 0x1e, 0xb5, 0x7c,
	0x4c, 0x2d, 0x3f, 0xea, 0x07, 0x27, 0x96, 0x9d, 0x83, 0xbf, 0x12, 0x71, 0x2c, 0x62, 0xac, 0x67,
	0xe3, 0xff, 0xc2, 0x46, 0x10, 0xfe, 0x09, 0x27, 0xae, 0xd8, 0xb7, 0x62, 0xbd, 0x17, 0x31, 0x7e,
	0xf2, 0x8f, 0x41, 0x79, 0x97, 0x68, 0x25, 0x33, 0x94, 0xc6, 0x2a, 0xea, 0x3f, 0xe6, 0x08, 0x25,
	0x99, 0xaf, 0x29, 0x68, 0x11, 0xb4, 0x29, 0x81, 0x61, 0xbd, 0x4e, 0x16, 0xe3, 0x0d, 0x17, 0xfa,
	0xd0, 0x7d, 0x99, 0xe5, 0xa6, 0x3a, 0xff, 0x0e, 0xc6, 0xf7, 0x57, 0x8d, 0x0d, 0xa0, 0x7b, 0xf5,
	0x7a, 0x71, 0xfd, 0xeb, 0xf8, 0x01, 0x3b, 0x81, 0xc1, 0xab, 0x37, 0xd7, 0xd7, 0x6f, 0xde, 0x2d,
	0x6e, 0x7e, 0x1b, 0x7b, 0x17, 0x06, 0x46, 0x97, 0x28, 0xd7, 0xbb, 0x4c, 0xe8, 0xbd, 0xfb, 0x25,
	0xd9, 0x2f, 0x30, 0x7c, 0x6b, 0xdb, 0x51, 0x9b, 0xc1, 0x61, 0x17, 0xee, 0xfd, 0xa0, 0x13, 0x76,
	0x94, 0x48, 0xfd, 0xe2, 0xf0, 0xc1, 0x73, 0x8f, 0x4d, 0xa1, 0x73, 0x25, 0xd2, 0x94, 0x01, 0xf1,
	0x94, 0xd5, 0xe4, 0xe8, 0x1c, 0x3e, 0x58, 0xf5, 0xa8, 0x8f, 0xdf, 0xff, 0x17, 0x00, 0x00, 0xff,
	0xff, 0x5b, 0x66, 0x28, 0x8c, 0xd5, 0x05, 0x00, 0x00,
}

// Reference imports to suppress errors if they are not otherwise used.
var _ context.Context
var _ grpc.ClientConn

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion4

// BenchmarkWorkerClient is the client API for BenchmarkWorker service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://godoc.org/google.golang.org/grpc#ClientConn.NewStream.
type BenchmarkWorkerClient interface {
	StartWorker(ctx context.Context, in *WorkerConfiguration, opts ...grpc.CallOption) (BenchmarkWorker_StartWorkerClient, error)
	Call(ctx context.Context, in *Empty, opts ...grpc.CallOption) (*Empty, error)
}

type benchmarkWorkerClient struct {
	cc *grpc.ClientConn
}

func NewBenchmarkWorkerClient(cc *grpc.ClientConn) BenchmarkWorkerClient {
	return &benchmarkWorkerClient{cc}
}

func (c *benchmarkWorkerClient) StartWorker(ctx context.Context, in *WorkerConfiguration, opts ...grpc.CallOption) (BenchmarkWorker_StartWorkerClient, error) {
	stream, err := c.cc.NewStream(ctx, &_BenchmarkWorker_serviceDesc.Streams[0], "/api.BenchmarkWorker/StartWorker", opts...)
	if err != nil {
		return nil, err
	}
	x := &benchmarkWorkerStartWorkerClient{stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

type BenchmarkWorker_StartWorkerClient interface {
	Recv() (*ResultPackage, error)
	grpc.ClientStream
}

type benchmarkWorkerStartWorkerClient struct {
	grpc.ClientStream
}

func (x *benchmarkWorkerStartWorkerClient) Recv() (*ResultPackage, error) {
	m := new(ResultPackage)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func (c *benchmarkWorkerClient) Call(ctx context.Context, in *Empty, opts ...grpc.CallOption) (*Empty, error) {
	out := new(Empty)
	err := c.cc.Invoke(ctx, "/api.BenchmarkWorker/Call", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// BenchmarkWorkerServer is the server API for BenchmarkWorker service.
type BenchmarkWorkerServer interface {
	StartWorker(*WorkerConfiguration, BenchmarkWorker_StartWorkerServer) error
	Call(context.Context, *Empty) (*Empty, error)
}

// UnimplementedBenchmarkWorkerServer can be embedded to have forward compatible implementations.
type UnimplementedBenchmarkWorkerServer struct {
}

func (*UnimplementedBenchmarkWorkerServer) StartWorker(req *WorkerConfiguration, srv BenchmarkWorker_StartWorkerServer) error {
	return status.Errorf(codes.Unimplemented, "method StartWorker not implemented")
}
func (*UnimplementedBenchmarkWorkerServer) Call(ctx context.Context, req *Empty) (*Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Call not implemented")
}

func RegisterBenchmarkWorkerServer(s *grpc.Server, srv BenchmarkWorkerServer) {
	s.RegisterService(&_BenchmarkWorker_serviceDesc, srv)
}

func _BenchmarkWorker_StartWorker_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(WorkerConfiguration)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(BenchmarkWorkerServer).StartWorker(m, &benchmarkWorkerStartWorkerServer{stream})
}

type BenchmarkWorker_StartWorkerServer interface {
	Send(*ResultPackage) error
	grpc.ServerStream
}

type benchmarkWorkerStartWorkerServer struct {
	grpc.ServerStream
}

func (x *benchmarkWorkerStartWorkerServer) Send(m *ResultPackage) error {
	return x.ServerStream.SendMsg(m)
}

func _BenchmarkWorker_Call_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Empty)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(BenchmarkWorkerServer).Call(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/api.BenchmarkWorker/Call",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(BenchmarkWorkerServer).Call(ctx, req.(*Empty))
	}
	return interceptor(ctx, in, info, handler)
}

var _BenchmarkWorker_serviceDesc = grpc.ServiceDesc{
	ServiceName: "api.BenchmarkWorker",
	HandlerType: (*BenchmarkWorkerServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Call",
			Handler:    _BenchmarkWorker_Call_Handler,
		},
	},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "StartWorker",
			Handler:       _BenchmarkWorker_StartWorker_Handler,
			ServerStreams: true,
		},
	},
	Metadata: "tracewriter.proto",
}
