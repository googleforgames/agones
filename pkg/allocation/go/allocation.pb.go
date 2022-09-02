// Copyright 2022 Google LLC All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// This code was autogenerated. Do not edit directly.
// Code generated by protoc-gen-go. DO NOT EDIT.
// source: proto/allocation/allocation.proto

package allocation

import (
	fmt "fmt"

	proto "github.com/golang/protobuf/proto"

	math "math"

	_ "google.golang.org/genproto/googleapis/api/annotations"

	context "golang.org/x/net/context"

	grpc "google.golang.org/grpc"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion2 // please upgrade the proto package

type AllocationRequest_SchedulingStrategy int32

const (
	AllocationRequest_Packed      AllocationRequest_SchedulingStrategy = 0
	AllocationRequest_Distributed AllocationRequest_SchedulingStrategy = 1
)

var AllocationRequest_SchedulingStrategy_name = map[int32]string{
	0: "Packed",
	1: "Distributed",
}
var AllocationRequest_SchedulingStrategy_value = map[string]int32{
	"Packed":      0,
	"Distributed": 1,
}

func (x AllocationRequest_SchedulingStrategy) String() string {
	return proto.EnumName(AllocationRequest_SchedulingStrategy_name, int32(x))
}
func (AllocationRequest_SchedulingStrategy) EnumDescriptor() ([]byte, []int) {
	return fileDescriptor_allocation_df43bf861ceb174e, []int{0, 0}
}

type GameServerSelector_GameServerState int32

const (
	GameServerSelector_READY     GameServerSelector_GameServerState = 0
	GameServerSelector_ALLOCATED GameServerSelector_GameServerState = 1
)

var GameServerSelector_GameServerState_name = map[int32]string{
	0: "READY",
	1: "ALLOCATED",
}
var GameServerSelector_GameServerState_value = map[string]int32{
	"READY":     0,
	"ALLOCATED": 1,
}

func (x GameServerSelector_GameServerState) String() string {
	return proto.EnumName(GameServerSelector_GameServerState_name, int32(x))
}
func (GameServerSelector_GameServerState) EnumDescriptor() ([]byte, []int) {
	return fileDescriptor_allocation_df43bf861ceb174e, []int{5, 0}
}

type AllocationRequest struct {
	// The k8s namespace that is hosting the targeted fleet of gameservers to be allocated
	Namespace string `protobuf:"bytes,1,opt,name=namespace,proto3" json:"namespace,omitempty"`
	// If specified, multi-cluster policies are applied. Otherwise, allocation will happen locally.
	MultiClusterSetting *MultiClusterSetting `protobuf:"bytes,2,opt,name=multiClusterSetting,proto3" json:"multiClusterSetting,omitempty"`
	// Deprecated: Please use gameServerSelectors instead. This field is ignored if the
	// gameServerSelectors field is set
	// The required allocation. Defaults to all GameServers.
	RequiredGameServerSelector *GameServerSelector `protobuf:"bytes,3,opt,name=requiredGameServerSelector,proto3" json:"requiredGameServerSelector,omitempty"` // Deprecated: Do not use.
	// Deprecated: Please use gameServerSelectors instead. This field is ignored if the
	// gameServerSelectors field is set
	// The ordered list of preferred allocations out of the `required` set.
	// If the first selector is not matched, the selection attempts the second selector, and so on.
	PreferredGameServerSelectors []*GameServerSelector `protobuf:"bytes,4,rep,name=preferredGameServerSelectors,proto3" json:"preferredGameServerSelectors,omitempty"` // Deprecated: Do not use.
	// Scheduling strategy. Defaults to "Packed".
	Scheduling AllocationRequest_SchedulingStrategy `protobuf:"varint,5,opt,name=scheduling,proto3,enum=allocation.AllocationRequest_SchedulingStrategy" json:"scheduling,omitempty"`
	// Deprecated: Please use metadata instead. This field is ignored if the
	// metadata field is set
	MetaPatch *MetaPatch `protobuf:"bytes,6,opt,name=metaPatch,proto3" json:"metaPatch,omitempty"`
	// Metadata is optional custom metadata that is added to the game server at
	// allocation. You can use this to tell the server necessary session data
	Metadata *MetaPatch `protobuf:"bytes,7,opt,name=metadata,proto3" json:"metadata,omitempty"`
	// Ordered list of GameServer label selectors.
	// If the first selector is not matched, the selection attempts the second selector, and so on.
	// This is useful for things like smoke testing of new game servers.
	// Note: This field can only be set if neither Required or Preferred is set.
	GameServerSelectors  []*GameServerSelector `protobuf:"bytes,8,rep,name=gameServerSelectors,proto3" json:"gameServerSelectors,omitempty"`
	XXX_NoUnkeyedLiteral struct{}              `json:"-"`
	XXX_unrecognized     []byte                `json:"-"`
	XXX_sizecache        int32                 `json:"-"`
}

func (m *AllocationRequest) Reset()         { *m = AllocationRequest{} }
func (m *AllocationRequest) String() string { return proto.CompactTextString(m) }
func (*AllocationRequest) ProtoMessage()    {}
func (*AllocationRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_allocation_df43bf861ceb174e, []int{0}
}
func (m *AllocationRequest) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_AllocationRequest.Unmarshal(m, b)
}
func (m *AllocationRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_AllocationRequest.Marshal(b, m, deterministic)
}
func (dst *AllocationRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_AllocationRequest.Merge(dst, src)
}
func (m *AllocationRequest) XXX_Size() int {
	return xxx_messageInfo_AllocationRequest.Size(m)
}
func (m *AllocationRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_AllocationRequest.DiscardUnknown(m)
}

var xxx_messageInfo_AllocationRequest proto.InternalMessageInfo

func (m *AllocationRequest) GetNamespace() string {
	if m != nil {
		return m.Namespace
	}
	return ""
}

func (m *AllocationRequest) GetMultiClusterSetting() *MultiClusterSetting {
	if m != nil {
		return m.MultiClusterSetting
	}
	return nil
}

// Deprecated: Do not use.
func (m *AllocationRequest) GetRequiredGameServerSelector() *GameServerSelector {
	if m != nil {
		return m.RequiredGameServerSelector
	}
	return nil
}

// Deprecated: Do not use.
func (m *AllocationRequest) GetPreferredGameServerSelectors() []*GameServerSelector {
	if m != nil {
		return m.PreferredGameServerSelectors
	}
	return nil
}

func (m *AllocationRequest) GetScheduling() AllocationRequest_SchedulingStrategy {
	if m != nil {
		return m.Scheduling
	}
	return AllocationRequest_Packed
}

func (m *AllocationRequest) GetMetaPatch() *MetaPatch {
	if m != nil {
		return m.MetaPatch
	}
	return nil
}

func (m *AllocationRequest) GetMetadata() *MetaPatch {
	if m != nil {
		return m.Metadata
	}
	return nil
}

func (m *AllocationRequest) GetGameServerSelectors() []*GameServerSelector {
	if m != nil {
		return m.GameServerSelectors
	}
	return nil
}

type AllocationResponse struct {
	GameServerName       string                                     `protobuf:"bytes,2,opt,name=gameServerName,proto3" json:"gameServerName,omitempty"`
	Ports                []*AllocationResponse_GameServerStatusPort `protobuf:"bytes,3,rep,name=ports,proto3" json:"ports,omitempty"`
	Address              string                                     `protobuf:"bytes,4,opt,name=address,proto3" json:"address,omitempty"`
	NodeName             string                                     `protobuf:"bytes,5,opt,name=nodeName,proto3" json:"nodeName,omitempty"`
	XXX_NoUnkeyedLiteral struct{}                                   `json:"-"`
	XXX_unrecognized     []byte                                     `json:"-"`
	XXX_sizecache        int32                                      `json:"-"`
}

func (m *AllocationResponse) Reset()         { *m = AllocationResponse{} }
func (m *AllocationResponse) String() string { return proto.CompactTextString(m) }
func (*AllocationResponse) ProtoMessage()    {}
func (*AllocationResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_allocation_df43bf861ceb174e, []int{1}
}
func (m *AllocationResponse) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_AllocationResponse.Unmarshal(m, b)
}
func (m *AllocationResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_AllocationResponse.Marshal(b, m, deterministic)
}
func (dst *AllocationResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_AllocationResponse.Merge(dst, src)
}
func (m *AllocationResponse) XXX_Size() int {
	return xxx_messageInfo_AllocationResponse.Size(m)
}
func (m *AllocationResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_AllocationResponse.DiscardUnknown(m)
}

var xxx_messageInfo_AllocationResponse proto.InternalMessageInfo

func (m *AllocationResponse) GetGameServerName() string {
	if m != nil {
		return m.GameServerName
	}
	return ""
}

func (m *AllocationResponse) GetPorts() []*AllocationResponse_GameServerStatusPort {
	if m != nil {
		return m.Ports
	}
	return nil
}

func (m *AllocationResponse) GetAddress() string {
	if m != nil {
		return m.Address
	}
	return ""
}

func (m *AllocationResponse) GetNodeName() string {
	if m != nil {
		return m.NodeName
	}
	return ""
}

// The gameserver port info that is allocated.
type AllocationResponse_GameServerStatusPort struct {
	Name                 string   `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`
	Port                 int32    `protobuf:"varint,2,opt,name=port,proto3" json:"port,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *AllocationResponse_GameServerStatusPort) Reset() {
	*m = AllocationResponse_GameServerStatusPort{}
}
func (m *AllocationResponse_GameServerStatusPort) String() string { return proto.CompactTextString(m) }
func (*AllocationResponse_GameServerStatusPort) ProtoMessage()    {}
func (*AllocationResponse_GameServerStatusPort) Descriptor() ([]byte, []int) {
	return fileDescriptor_allocation_df43bf861ceb174e, []int{1, 0}
}
func (m *AllocationResponse_GameServerStatusPort) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_AllocationResponse_GameServerStatusPort.Unmarshal(m, b)
}
func (m *AllocationResponse_GameServerStatusPort) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_AllocationResponse_GameServerStatusPort.Marshal(b, m, deterministic)
}
func (dst *AllocationResponse_GameServerStatusPort) XXX_Merge(src proto.Message) {
	xxx_messageInfo_AllocationResponse_GameServerStatusPort.Merge(dst, src)
}
func (m *AllocationResponse_GameServerStatusPort) XXX_Size() int {
	return xxx_messageInfo_AllocationResponse_GameServerStatusPort.Size(m)
}
func (m *AllocationResponse_GameServerStatusPort) XXX_DiscardUnknown() {
	xxx_messageInfo_AllocationResponse_GameServerStatusPort.DiscardUnknown(m)
}

var xxx_messageInfo_AllocationResponse_GameServerStatusPort proto.InternalMessageInfo

func (m *AllocationResponse_GameServerStatusPort) GetName() string {
	if m != nil {
		return m.Name
	}
	return ""
}

func (m *AllocationResponse_GameServerStatusPort) GetPort() int32 {
	if m != nil {
		return m.Port
	}
	return 0
}

// Specifies settings for multi-cluster allocation.
type MultiClusterSetting struct {
	// If set to true, multi-cluster allocation is enabled.
	Enabled bool `protobuf:"varint,1,opt,name=enabled,proto3" json:"enabled,omitempty"`
	// Selects multi-cluster allocation policies to apply. If not specified, all multi-cluster allocation policies are to be applied.
	PolicySelector       *LabelSelector `protobuf:"bytes,2,opt,name=policySelector,proto3" json:"policySelector,omitempty"`
	XXX_NoUnkeyedLiteral struct{}       `json:"-"`
	XXX_unrecognized     []byte         `json:"-"`
	XXX_sizecache        int32          `json:"-"`
}

func (m *MultiClusterSetting) Reset()         { *m = MultiClusterSetting{} }
func (m *MultiClusterSetting) String() string { return proto.CompactTextString(m) }
func (*MultiClusterSetting) ProtoMessage()    {}
func (*MultiClusterSetting) Descriptor() ([]byte, []int) {
	return fileDescriptor_allocation_df43bf861ceb174e, []int{2}
}
func (m *MultiClusterSetting) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_MultiClusterSetting.Unmarshal(m, b)
}
func (m *MultiClusterSetting) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_MultiClusterSetting.Marshal(b, m, deterministic)
}
func (dst *MultiClusterSetting) XXX_Merge(src proto.Message) {
	xxx_messageInfo_MultiClusterSetting.Merge(dst, src)
}
func (m *MultiClusterSetting) XXX_Size() int {
	return xxx_messageInfo_MultiClusterSetting.Size(m)
}
func (m *MultiClusterSetting) XXX_DiscardUnknown() {
	xxx_messageInfo_MultiClusterSetting.DiscardUnknown(m)
}

var xxx_messageInfo_MultiClusterSetting proto.InternalMessageInfo

func (m *MultiClusterSetting) GetEnabled() bool {
	if m != nil {
		return m.Enabled
	}
	return false
}

func (m *MultiClusterSetting) GetPolicySelector() *LabelSelector {
	if m != nil {
		return m.PolicySelector
	}
	return nil
}

// MetaPatch is the metadata used to patch the GameServer metadata on allocation
type MetaPatch struct {
	Labels               map[string]string `protobuf:"bytes,1,rep,name=labels,proto3" json:"labels,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
	Annotations          map[string]string `protobuf:"bytes,2,rep,name=annotations,proto3" json:"annotations,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
	XXX_NoUnkeyedLiteral struct{}          `json:"-"`
	XXX_unrecognized     []byte            `json:"-"`
	XXX_sizecache        int32             `json:"-"`
}

func (m *MetaPatch) Reset()         { *m = MetaPatch{} }
func (m *MetaPatch) String() string { return proto.CompactTextString(m) }
func (*MetaPatch) ProtoMessage()    {}
func (*MetaPatch) Descriptor() ([]byte, []int) {
	return fileDescriptor_allocation_df43bf861ceb174e, []int{3}
}
func (m *MetaPatch) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_MetaPatch.Unmarshal(m, b)
}
func (m *MetaPatch) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_MetaPatch.Marshal(b, m, deterministic)
}
func (dst *MetaPatch) XXX_Merge(src proto.Message) {
	xxx_messageInfo_MetaPatch.Merge(dst, src)
}
func (m *MetaPatch) XXX_Size() int {
	return xxx_messageInfo_MetaPatch.Size(m)
}
func (m *MetaPatch) XXX_DiscardUnknown() {
	xxx_messageInfo_MetaPatch.DiscardUnknown(m)
}

var xxx_messageInfo_MetaPatch proto.InternalMessageInfo

func (m *MetaPatch) GetLabels() map[string]string {
	if m != nil {
		return m.Labels
	}
	return nil
}

func (m *MetaPatch) GetAnnotations() map[string]string {
	if m != nil {
		return m.Annotations
	}
	return nil
}

// LabelSelector used for finding a GameServer with matching labels.
type LabelSelector struct {
	// Labels to match.
	MatchLabels          map[string]string `protobuf:"bytes,1,rep,name=matchLabels,proto3" json:"matchLabels,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
	XXX_NoUnkeyedLiteral struct{}          `json:"-"`
	XXX_unrecognized     []byte            `json:"-"`
	XXX_sizecache        int32             `json:"-"`
}

func (m *LabelSelector) Reset()         { *m = LabelSelector{} }
func (m *LabelSelector) String() string { return proto.CompactTextString(m) }
func (*LabelSelector) ProtoMessage()    {}
func (*LabelSelector) Descriptor() ([]byte, []int) {
	return fileDescriptor_allocation_df43bf861ceb174e, []int{4}
}
func (m *LabelSelector) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_LabelSelector.Unmarshal(m, b)
}
func (m *LabelSelector) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_LabelSelector.Marshal(b, m, deterministic)
}
func (dst *LabelSelector) XXX_Merge(src proto.Message) {
	xxx_messageInfo_LabelSelector.Merge(dst, src)
}
func (m *LabelSelector) XXX_Size() int {
	return xxx_messageInfo_LabelSelector.Size(m)
}
func (m *LabelSelector) XXX_DiscardUnknown() {
	xxx_messageInfo_LabelSelector.DiscardUnknown(m)
}

var xxx_messageInfo_LabelSelector proto.InternalMessageInfo

func (m *LabelSelector) GetMatchLabels() map[string]string {
	if m != nil {
		return m.MatchLabels
	}
	return nil
}

// GameServerSelector used for finding a GameServer with matching filters.
type GameServerSelector struct {
	// Labels to match.
	MatchLabels          map[string]string                  `protobuf:"bytes,1,rep,name=matchLabels,proto3" json:"matchLabels,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
	GameServerState      GameServerSelector_GameServerState `protobuf:"varint,2,opt,name=gameServerState,proto3,enum=allocation.GameServerSelector_GameServerState" json:"gameServerState,omitempty"`
	Players              *PlayerSelector                    `protobuf:"bytes,3,opt,name=players,proto3" json:"players,omitempty"`
	XXX_NoUnkeyedLiteral struct{}                           `json:"-"`
	XXX_unrecognized     []byte                             `json:"-"`
	XXX_sizecache        int32                              `json:"-"`
}

func (m *GameServerSelector) Reset()         { *m = GameServerSelector{} }
func (m *GameServerSelector) String() string { return proto.CompactTextString(m) }
func (*GameServerSelector) ProtoMessage()    {}
func (*GameServerSelector) Descriptor() ([]byte, []int) {
	return fileDescriptor_allocation_df43bf861ceb174e, []int{5}
}
func (m *GameServerSelector) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_GameServerSelector.Unmarshal(m, b)
}
func (m *GameServerSelector) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_GameServerSelector.Marshal(b, m, deterministic)
}
func (dst *GameServerSelector) XXX_Merge(src proto.Message) {
	xxx_messageInfo_GameServerSelector.Merge(dst, src)
}
func (m *GameServerSelector) XXX_Size() int {
	return xxx_messageInfo_GameServerSelector.Size(m)
}
func (m *GameServerSelector) XXX_DiscardUnknown() {
	xxx_messageInfo_GameServerSelector.DiscardUnknown(m)
}

var xxx_messageInfo_GameServerSelector proto.InternalMessageInfo

func (m *GameServerSelector) GetMatchLabels() map[string]string {
	if m != nil {
		return m.MatchLabels
	}
	return nil
}

func (m *GameServerSelector) GetGameServerState() GameServerSelector_GameServerState {
	if m != nil {
		return m.GameServerState
	}
	return GameServerSelector_READY
}

func (m *GameServerSelector) GetPlayers() *PlayerSelector {
	if m != nil {
		return m.Players
	}
	return nil
}

// PlayerSelector is filter for player capacity values.
// minAvailable should always be less or equal to maxAvailable.
type PlayerSelector struct {
	MinAvailable         uint64   `protobuf:"varint,1,opt,name=minAvailable,proto3" json:"minAvailable,omitempty"`
	MaxAvailable         uint64   `protobuf:"varint,2,opt,name=maxAvailable,proto3" json:"maxAvailable,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *PlayerSelector) Reset()         { *m = PlayerSelector{} }
func (m *PlayerSelector) String() string { return proto.CompactTextString(m) }
func (*PlayerSelector) ProtoMessage()    {}
func (*PlayerSelector) Descriptor() ([]byte, []int) {
	return fileDescriptor_allocation_df43bf861ceb174e, []int{6}
}
func (m *PlayerSelector) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_PlayerSelector.Unmarshal(m, b)
}
func (m *PlayerSelector) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_PlayerSelector.Marshal(b, m, deterministic)
}
func (dst *PlayerSelector) XXX_Merge(src proto.Message) {
	xxx_messageInfo_PlayerSelector.Merge(dst, src)
}
func (m *PlayerSelector) XXX_Size() int {
	return xxx_messageInfo_PlayerSelector.Size(m)
}
func (m *PlayerSelector) XXX_DiscardUnknown() {
	xxx_messageInfo_PlayerSelector.DiscardUnknown(m)
}

var xxx_messageInfo_PlayerSelector proto.InternalMessageInfo

func (m *PlayerSelector) GetMinAvailable() uint64 {
	if m != nil {
		return m.MinAvailable
	}
	return 0
}

func (m *PlayerSelector) GetMaxAvailable() uint64 {
	if m != nil {
		return m.MaxAvailable
	}
	return 0
}

func init() {
	proto.RegisterType((*AllocationRequest)(nil), "allocation.AllocationRequest")
	proto.RegisterType((*AllocationResponse)(nil), "allocation.AllocationResponse")
	proto.RegisterType((*AllocationResponse_GameServerStatusPort)(nil), "allocation.AllocationResponse.GameServerStatusPort")
	proto.RegisterType((*MultiClusterSetting)(nil), "allocation.MultiClusterSetting")
	proto.RegisterType((*MetaPatch)(nil), "allocation.MetaPatch")
	proto.RegisterMapType((map[string]string)(nil), "allocation.MetaPatch.AnnotationsEntry")
	proto.RegisterMapType((map[string]string)(nil), "allocation.MetaPatch.LabelsEntry")
	proto.RegisterType((*LabelSelector)(nil), "allocation.LabelSelector")
	proto.RegisterMapType((map[string]string)(nil), "allocation.LabelSelector.MatchLabelsEntry")
	proto.RegisterType((*GameServerSelector)(nil), "allocation.GameServerSelector")
	proto.RegisterMapType((map[string]string)(nil), "allocation.GameServerSelector.MatchLabelsEntry")
	proto.RegisterType((*PlayerSelector)(nil), "allocation.PlayerSelector")
	proto.RegisterEnum("allocation.AllocationRequest_SchedulingStrategy", AllocationRequest_SchedulingStrategy_name, AllocationRequest_SchedulingStrategy_value)
	proto.RegisterEnum("allocation.GameServerSelector_GameServerState", GameServerSelector_GameServerState_name, GameServerSelector_GameServerState_value)
}

// Reference imports to suppress errors if they are not otherwise used.
var _ context.Context
var _ grpc.ClientConn

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion4

// AllocationServiceClient is the client API for AllocationService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://godoc.org/google.golang.org/grpc#ClientConn.NewStream.
type AllocationServiceClient interface {
	Allocate(ctx context.Context, in *AllocationRequest, opts ...grpc.CallOption) (*AllocationResponse, error)
}

type allocationServiceClient struct {
	cc *grpc.ClientConn
}

func NewAllocationServiceClient(cc *grpc.ClientConn) AllocationServiceClient {
	return &allocationServiceClient{cc}
}

func (c *allocationServiceClient) Allocate(ctx context.Context, in *AllocationRequest, opts ...grpc.CallOption) (*AllocationResponse, error) {
	out := new(AllocationResponse)
	err := c.cc.Invoke(ctx, "/allocation.AllocationService/Allocate", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// AllocationServiceServer is the server API for AllocationService service.
type AllocationServiceServer interface {
	Allocate(context.Context, *AllocationRequest) (*AllocationResponse, error)
}

func RegisterAllocationServiceServer(s *grpc.Server, srv AllocationServiceServer) {
	s.RegisterService(&_AllocationService_serviceDesc, srv)
}

func _AllocationService_Allocate_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(AllocationRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(AllocationServiceServer).Allocate(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/allocation.AllocationService/Allocate",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(AllocationServiceServer).Allocate(ctx, req.(*AllocationRequest))
	}
	return interceptor(ctx, in, info, handler)
}

var _AllocationService_serviceDesc = grpc.ServiceDesc{
	ServiceName: "allocation.AllocationService",
	HandlerType: (*AllocationServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Allocate",
			Handler:    _AllocationService_Allocate_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "proto/allocation/allocation.proto",
}

func init() {
	proto.RegisterFile("proto/allocation/allocation.proto", fileDescriptor_allocation_df43bf861ceb174e)
}

var fileDescriptor_allocation_df43bf861ceb174e = []byte{
	// 776 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xa4, 0x55, 0x41, 0x6f, 0xe3, 0x44,
	0x14, 0x5e, 0x3b, 0x4d, 0x9a, 0xbc, 0xb0, 0x69, 0x78, 0xdd, 0x95, 0x8c, 0x55, 0x96, 0xac, 0x0f,
	0xab, 0x6a, 0x91, 0x12, 0x36, 0xe5, 0xc0, 0xee, 0xa1, 0x52, 0x68, 0x2b, 0x40, 0x4a, 0x21, 0x75,
	0x38, 0x94, 0xe3, 0xc4, 0x9e, 0xa6, 0x56, 0x1d, 0xdb, 0x9d, 0x19, 0x57, 0xe4, 0x86, 0xb8, 0x72,
	0xe0, 0xc0, 0x99, 0x5f, 0xc5, 0x5f, 0xe8, 0xdf, 0x40, 0x42, 0x33, 0x4e, 0xec, 0x49, 0xe2, 0x86,
	0x56, 0x7b, 0x9b, 0x79, 0xf3, 0xbd, 0x6f, 0xbe, 0xf7, 0xf9, 0xbd, 0x31, 0xbc, 0x4e, 0x58, 0x2c,
	0xe2, 0x1e, 0x09, 0xc3, 0xd8, 0x23, 0x22, 0x88, 0x23, 0x6d, 0xd9, 0x55, 0x67, 0x08, 0x45, 0xc4,
	0x3e, 0x98, 0xc6, 0xf1, 0x34, 0xa4, 0x3d, 0x92, 0x04, 0x3d, 0x12, 0x45, 0xb1, 0x50, 0x61, 0x9e,
	0x21, 0x9d, 0x3f, 0xab, 0xf0, 0xe9, 0x20, 0x07, 0xbb, 0xf4, 0x36, 0xa5, 0x5c, 0xe0, 0x01, 0x34,
	0x22, 0x32, 0xa3, 0x3c, 0x21, 0x1e, 0xb5, 0x8c, 0x8e, 0x71, 0xd8, 0x70, 0x8b, 0x00, 0x5e, 0xc0,
	0xfe, 0x2c, 0x0d, 0x45, 0x70, 0x12, 0xa6, 0x5c, 0x50, 0x36, 0xa6, 0x42, 0x04, 0xd1, 0xd4, 0x32,
	0x3b, 0xc6, 0x61, 0xb3, 0xff, 0x45, 0x57, 0x53, 0x73, 0xbe, 0x09, 0x73, 0xcb, 0x72, 0x71, 0x02,
	0x36, 0xa3, 0xb7, 0x69, 0xc0, 0xa8, 0xff, 0x1d, 0x99, 0xd1, 0x31, 0x65, 0x77, 0xf2, 0x30, 0xa4,
	0x9e, 0x88, 0x99, 0x55, 0x51, 0xcc, 0xaf, 0x74, 0xe6, 0x4d, 0xd4, 0xb7, 0xa6, 0x65, 0xb8, 0x5b,
	0x58, 0xf0, 0x0a, 0x0e, 0x12, 0x46, 0xaf, 0x28, 0x2b, 0x3d, 0xe6, 0xd6, 0x4e, 0xa7, 0xf2, 0xc8,
	0x5b, 0xb6, 0xf2, 0xe0, 0x08, 0x80, 0x7b, 0xd7, 0xd4, 0x4f, 0x43, 0xe9, 0x4a, 0xb5, 0x63, 0x1c,
	0xb6, 0xfa, 0x5f, 0xe9, 0xac, 0x1b, 0x7e, 0x77, 0xc7, 0x39, 0x7e, 0x2c, 0x18, 0x11, 0x74, 0x3a,
	0x77, 0x35, 0x0e, 0x3c, 0x82, 0xc6, 0x8c, 0x0a, 0x32, 0x22, 0xc2, 0xbb, 0xb6, 0x6a, 0xca, 0x8c,
	0x97, 0x2b, 0x36, 0x2f, 0x0f, 0xdd, 0x02, 0x87, 0xef, 0xa0, 0x2e, 0x37, 0x3e, 0x11, 0xc4, 0xda,
	0xdd, 0x96, 0x93, 0xc3, 0x70, 0x04, 0xfb, 0xd3, 0x12, 0x63, 0xea, 0x8f, 0x31, 0xc6, 0x2d, 0x4b,
	0x75, 0xde, 0x01, 0x6e, 0xd6, 0x86, 0x00, 0xb5, 0x11, 0xf1, 0x6e, 0xa8, 0xdf, 0x7e, 0x86, 0x7b,
	0xd0, 0x3c, 0x0d, 0xb8, 0x60, 0xc1, 0x24, 0x15, 0xd4, 0x6f, 0x1b, 0xce, 0xbf, 0x06, 0xa0, 0xee,
	0x10, 0x4f, 0xe2, 0x88, 0x53, 0x7c, 0x03, 0xad, 0xe2, 0x82, 0x1f, 0xc9, 0x8c, 0xaa, 0x7e, 0x6b,
	0xb8, 0x6b, 0x51, 0xfc, 0x01, 0xaa, 0x49, 0xcc, 0x04, 0xb7, 0x2a, 0x4a, 0xf5, 0xd1, 0x43, 0xc6,
	0x67, 0xb4, 0x7a, 0x21, 0x82, 0x88, 0x94, 0x8f, 0x62, 0x26, 0xdc, 0x8c, 0x01, 0x2d, 0xd8, 0x25,
	0xbe, 0xcf, 0x28, 0x97, 0xbd, 0x21, 0xef, 0x5a, 0x6e, 0xd1, 0x86, 0x7a, 0x14, 0xfb, 0x54, 0xc9,
	0xa8, 0xaa, 0xa3, 0x7c, 0x6f, 0x1f, 0xc3, 0x8b, 0x32, 0x52, 0x44, 0xd8, 0x91, 0x23, 0xb4, 0x18,
	0x27, 0xb5, 0x96, 0x31, 0x79, 0x95, 0x2a, 0xa5, 0xea, 0xaa, 0xb5, 0xc3, 0x60, 0xbf, 0x64, 0x6c,
	0xa4, 0x18, 0x1a, 0x91, 0x49, 0x48, 0x7d, 0xc5, 0x50, 0x77, 0x97, 0x5b, 0x1c, 0x40, 0x2b, 0x89,
	0xc3, 0xc0, 0x9b, 0xe7, 0xf3, 0x92, 0x4d, 0xe2, 0x67, 0x7a, 0xe9, 0x43, 0x32, 0xa1, 0x61, 0xfe,
	0xad, 0xd6, 0x12, 0x9c, 0x3f, 0x4c, 0x68, 0xe4, 0x0d, 0x81, 0xef, 0xa1, 0x16, 0x4a, 0x38, 0xb7,
	0x0c, 0xe5, 0xe1, 0xeb, 0xd2, 0xbe, 0xc9, 0x28, 0xf9, 0x59, 0x24, 0xd8, 0xdc, 0x5d, 0x24, 0xe0,
	0xf7, 0xd0, 0xd4, 0xde, 0x18, 0xcb, 0x54, 0xf9, 0x6f, 0xca, 0xf3, 0x07, 0x05, 0x30, 0x23, 0xd1,
	0x53, 0xed, 0xf7, 0xd0, 0xd4, 0x2e, 0xc0, 0x36, 0x54, 0x6e, 0xe8, 0x7c, 0x61, 0x9e, 0x5c, 0xe2,
	0x0b, 0xa8, 0xde, 0x91, 0x30, 0x5d, 0xf6, 0x41, 0xb6, 0xf9, 0x60, 0x7e, 0x63, 0xd8, 0xc7, 0xd0,
	0x5e, 0xe7, 0x7e, 0x4a, 0xbe, 0xf3, 0xb7, 0x01, 0xcf, 0x57, 0xfc, 0xc2, 0x21, 0x34, 0x67, 0x52,
	0xf3, 0x50, 0xb7, 0xe5, 0xed, 0x83, 0xfe, 0x76, 0xcf, 0x0b, 0xf0, 0xa2, 0x34, 0x2d, 0x5d, 0xea,
	0x5b, 0x07, 0x3c, 0x49, 0xdf, 0xbd, 0x09, 0x58, 0xf2, 0xbe, 0x5d, 0x94, 0x89, 0xec, 0x6d, 0x9f,
	0xda, 0xed, 0x4a, 0xf1, 0x12, 0xf6, 0xa6, 0x2b, 0xbd, 0x9c, 0xa9, 0x69, 0xf5, 0xbb, 0xff, 0x43,
	0xbb, 0x3a, 0x01, 0xd4, 0x5d, 0xa7, 0xc1, 0xaf, 0x61, 0x37, 0x09, 0xc9, 0x9c, 0x32, 0xbe, 0x78,
	0xdd, 0x6d, 0x9d, 0x71, 0xa4, 0x8e, 0xf2, 0x76, 0x5d, 0x42, 0x3f, 0xda, 0xb9, 0x2f, 0x61, 0x6f,
	0x4d, 0x19, 0x36, 0xa0, 0xea, 0x9e, 0x0d, 0x4e, 0x7f, 0x69, 0x3f, 0xc3, 0xe7, 0xd0, 0x18, 0x0c,
	0x87, 0x3f, 0x9d, 0x0c, 0x7e, 0x3e, 0x3b, 0x6d, 0x1b, 0xce, 0x25, 0xb4, 0x56, 0x75, 0xa0, 0x03,
	0x9f, 0xcc, 0x82, 0x68, 0x70, 0x47, 0x82, 0x50, 0x8e, 0x9e, 0xba, 0x73, 0xc7, 0x5d, 0x89, 0x29,
	0x0c, 0xf9, 0xb5, 0xc0, 0x98, 0x0b, 0x8c, 0x16, 0xeb, 0xff, 0x66, 0xe8, 0x3f, 0x5d, 0xa9, 0x26,
	0xf0, 0x28, 0xde, 0x40, 0x7d, 0x11, 0xa4, 0xf8, 0xf9, 0xd6, 0xff, 0x85, 0xfd, 0x6a, 0xfb, 0xab,
	0xe6, 0x74, 0x7e, 0xff, 0xe7, 0xfe, 0x2f, 0xd3, 0x76, 0x5e, 0xf6, 0xa4, 0xef, 0x5c, 0x95, 0x5b,
	0x64, 0x7c, 0x30, 0xde, 0x4e, 0x6a, 0xea, 0xf7, 0x7f, 0xf4, 0x5f, 0x00, 0x00, 0x00, 0xff, 0xff,
	0x44, 0x23, 0x5c, 0x13, 0x4d, 0x08, 0x00, 0x00,
}
