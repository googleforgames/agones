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
// source: alpha.proto

package alpha

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

// I am Empty
type Empty struct {
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *Empty) Reset()         { *m = Empty{} }
func (m *Empty) String() string { return proto.CompactTextString(m) }
func (*Empty) ProtoMessage()    {}
func (*Empty) Descriptor() ([]byte, []int) {
	return fileDescriptor_alpha_adf85771d71a9075, []int{0}
}
func (m *Empty) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Empty.Unmarshal(m, b)
}
func (m *Empty) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Empty.Marshal(b, m, deterministic)
}
func (dst *Empty) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Empty.Merge(dst, src)
}
func (m *Empty) XXX_Size() int {
	return xxx_messageInfo_Empty.Size(m)
}
func (m *Empty) XXX_DiscardUnknown() {
	xxx_messageInfo_Empty.DiscardUnknown(m)
}

var xxx_messageInfo_Empty proto.InternalMessageInfo

// Store a count variable.
type Count struct {
	Count                int64    `protobuf:"varint,1,opt,name=count,proto3" json:"count,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *Count) Reset()         { *m = Count{} }
func (m *Count) String() string { return proto.CompactTextString(m) }
func (*Count) ProtoMessage()    {}
func (*Count) Descriptor() ([]byte, []int) {
	return fileDescriptor_alpha_adf85771d71a9075, []int{1}
}
func (m *Count) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Count.Unmarshal(m, b)
}
func (m *Count) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Count.Marshal(b, m, deterministic)
}
func (dst *Count) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Count.Merge(dst, src)
}
func (m *Count) XXX_Size() int {
	return xxx_messageInfo_Count.Size(m)
}
func (m *Count) XXX_DiscardUnknown() {
	xxx_messageInfo_Count.DiscardUnknown(m)
}

var xxx_messageInfo_Count proto.InternalMessageInfo

func (m *Count) GetCount() int64 {
	if m != nil {
		return m.Count
	}
	return 0
}

// Store a boolean result
type Bool struct {
	Bool                 bool     `protobuf:"varint,1,opt,name=bool,proto3" json:"bool,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *Bool) Reset()         { *m = Bool{} }
func (m *Bool) String() string { return proto.CompactTextString(m) }
func (*Bool) ProtoMessage()    {}
func (*Bool) Descriptor() ([]byte, []int) {
	return fileDescriptor_alpha_adf85771d71a9075, []int{2}
}
func (m *Bool) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Bool.Unmarshal(m, b)
}
func (m *Bool) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Bool.Marshal(b, m, deterministic)
}
func (dst *Bool) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Bool.Merge(dst, src)
}
func (m *Bool) XXX_Size() int {
	return xxx_messageInfo_Bool.Size(m)
}
func (m *Bool) XXX_DiscardUnknown() {
	xxx_messageInfo_Bool.DiscardUnknown(m)
}

var xxx_messageInfo_Bool proto.InternalMessageInfo

func (m *Bool) GetBool() bool {
	if m != nil {
		return m.Bool
	}
	return false
}

// The unique identifier for a given player.
type PlayerID struct {
	PlayerID             string   `protobuf:"bytes,1,opt,name=playerID,proto3" json:"playerID,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *PlayerID) Reset()         { *m = PlayerID{} }
func (m *PlayerID) String() string { return proto.CompactTextString(m) }
func (*PlayerID) ProtoMessage()    {}
func (*PlayerID) Descriptor() ([]byte, []int) {
	return fileDescriptor_alpha_adf85771d71a9075, []int{3}
}
func (m *PlayerID) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_PlayerID.Unmarshal(m, b)
}
func (m *PlayerID) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_PlayerID.Marshal(b, m, deterministic)
}
func (dst *PlayerID) XXX_Merge(src proto.Message) {
	xxx_messageInfo_PlayerID.Merge(dst, src)
}
func (m *PlayerID) XXX_Size() int {
	return xxx_messageInfo_PlayerID.Size(m)
}
func (m *PlayerID) XXX_DiscardUnknown() {
	xxx_messageInfo_PlayerID.DiscardUnknown(m)
}

var xxx_messageInfo_PlayerID proto.InternalMessageInfo

func (m *PlayerID) GetPlayerID() string {
	if m != nil {
		return m.PlayerID
	}
	return ""
}

// List of Player IDs
type PlayerIDList struct {
	List                 []string `protobuf:"bytes,1,rep,name=list,proto3" json:"list,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *PlayerIDList) Reset()         { *m = PlayerIDList{} }
func (m *PlayerIDList) String() string { return proto.CompactTextString(m) }
func (*PlayerIDList) ProtoMessage()    {}
func (*PlayerIDList) Descriptor() ([]byte, []int) {
	return fileDescriptor_alpha_adf85771d71a9075, []int{4}
}
func (m *PlayerIDList) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_PlayerIDList.Unmarshal(m, b)
}
func (m *PlayerIDList) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_PlayerIDList.Marshal(b, m, deterministic)
}
func (dst *PlayerIDList) XXX_Merge(src proto.Message) {
	xxx_messageInfo_PlayerIDList.Merge(dst, src)
}
func (m *PlayerIDList) XXX_Size() int {
	return xxx_messageInfo_PlayerIDList.Size(m)
}
func (m *PlayerIDList) XXX_DiscardUnknown() {
	xxx_messageInfo_PlayerIDList.DiscardUnknown(m)
}

var xxx_messageInfo_PlayerIDList proto.InternalMessageInfo

func (m *PlayerIDList) GetList() []string {
	if m != nil {
		return m.List
	}
	return nil
}

func init() {
	proto.RegisterType((*Empty)(nil), "agones.dev.sdk.alpha.Empty")
	proto.RegisterType((*Count)(nil), "agones.dev.sdk.alpha.Count")
	proto.RegisterType((*Bool)(nil), "agones.dev.sdk.alpha.Bool")
	proto.RegisterType((*PlayerID)(nil), "agones.dev.sdk.alpha.PlayerID")
	proto.RegisterType((*PlayerIDList)(nil), "agones.dev.sdk.alpha.PlayerIDList")
}

// Reference imports to suppress errors if they are not otherwise used.
var _ context.Context
var _ grpc.ClientConn

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion4

// SDKClient is the client API for SDK service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://godoc.org/google.golang.org/grpc#ClientConn.NewStream.
type SDKClient interface {
	// PlayerConnect increases the SDK’s stored player count by one, and appends this playerID to GameServer.Status.Players.IDs.
	//
	// GameServer.Status.Players.Count and GameServer.Status.Players.IDs are then set to update the player count and id list a second from now,
	// unless there is already an update pending, in which case the update joins that batch operation.
	//
	// PlayerConnect returns true and adds the playerID to the list of playerIDs if this playerID was not already in the
	// list of connected playerIDs.
	//
	// If the playerID exists within the list of connected playerIDs, PlayerConnect will return false, and the list of
	// connected playerIDs will be left unchanged.
	//
	// An error will be returned if the playerID was not already in the list of connected playerIDs but the player capacity for
	// the server has been reached. The playerID will not be added to the list of playerIDs.
	//
	// Warning: Do not use this method if you are manually managing GameServer.Status.Players.IDs and GameServer.Status.Players.Count
	// through the Kubernetes API, as indeterminate results will occur.
	PlayerConnect(ctx context.Context, in *PlayerID, opts ...grpc.CallOption) (*Bool, error)
	// Decreases the SDK’s stored player count by one, and removes the playerID from GameServer.Status.Players.IDs.
	//
	// GameServer.Status.Players.Count and GameServer.Status.Players.IDs are then set to update the player count and id list a second from now,
	// unless there is already an update pending, in which case the update joins that batch operation.
	//
	// PlayerDisconnect will return true and remove the supplied playerID from the list of connected playerIDs if the
	// playerID value exists within the list.
	//
	// If the playerID was not in the list of connected playerIDs, the call will return false, and the connected playerID list
	// will be left unchanged.
	//
	// Warning: Do not use this method if you are manually managing GameServer.status.players.IDs and GameServer.status.players.Count
	// through the Kubernetes API, as indeterminate results will occur.
	PlayerDisconnect(ctx context.Context, in *PlayerID, opts ...grpc.CallOption) (*Bool, error)
	// Update the GameServer.Status.Players.Capacity value with a new capacity.
	SetPlayerCapacity(ctx context.Context, in *Count, opts ...grpc.CallOption) (*Empty, error)
	// Retrieves the current player capacity. This is always accurate from what has been set through this SDK,
	// even if the value has yet to be updated on the GameServer status resource.
	//
	// If GameServer.Status.Players.Capacity is set manually through the Kubernetes API, use SDK.GameServer() or SDK.WatchGameServer() instead to view this value.
	GetPlayerCapacity(ctx context.Context, in *Empty, opts ...grpc.CallOption) (*Count, error)
	// Retrieves the current player count. This is always accurate from what has been set through this SDK,
	// even if the value has yet to be updated on the GameServer status resource.
	//
	// If GameServer.Status.Players.Count is set manually through the Kubernetes API, use SDK.GameServer() or SDK.WatchGameServer() instead to view this value.
	GetPlayerCount(ctx context.Context, in *Empty, opts ...grpc.CallOption) (*Count, error)
	// Returns if the playerID is currently connected to the GameServer. This is always accurate from what has been set through this SDK,
	// even if the value has yet to be updated on the GameServer status resource.
	//
	// If GameServer.Status.Players.IDs is set manually through the Kubernetes API, use SDK.GameServer() or SDK.WatchGameServer() instead to determine connected status.
	IsPlayerConnected(ctx context.Context, in *PlayerID, opts ...grpc.CallOption) (*Bool, error)
	// Returns the list of the currently connected player ids. This is always accurate from what has been set through this SDK,
	// even if the value has yet to be updated on the GameServer status resource.
	//
	// If GameServer.Status.Players.IDs is set manually through the Kubernetes API, use SDK.GameServer() or SDK.WatchGameServer() instead to view this value.
	GetConnectedPlayers(ctx context.Context, in *Empty, opts ...grpc.CallOption) (*PlayerIDList, error)
}

type sDKClient struct {
	cc *grpc.ClientConn
}

func NewSDKClient(cc *grpc.ClientConn) SDKClient {
	return &sDKClient{cc}
}

func (c *sDKClient) PlayerConnect(ctx context.Context, in *PlayerID, opts ...grpc.CallOption) (*Bool, error) {
	out := new(Bool)
	err := c.cc.Invoke(ctx, "/agones.dev.sdk.alpha.SDK/PlayerConnect", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *sDKClient) PlayerDisconnect(ctx context.Context, in *PlayerID, opts ...grpc.CallOption) (*Bool, error) {
	out := new(Bool)
	err := c.cc.Invoke(ctx, "/agones.dev.sdk.alpha.SDK/PlayerDisconnect", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *sDKClient) SetPlayerCapacity(ctx context.Context, in *Count, opts ...grpc.CallOption) (*Empty, error) {
	out := new(Empty)
	err := c.cc.Invoke(ctx, "/agones.dev.sdk.alpha.SDK/SetPlayerCapacity", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *sDKClient) GetPlayerCapacity(ctx context.Context, in *Empty, opts ...grpc.CallOption) (*Count, error) {
	out := new(Count)
	err := c.cc.Invoke(ctx, "/agones.dev.sdk.alpha.SDK/GetPlayerCapacity", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *sDKClient) GetPlayerCount(ctx context.Context, in *Empty, opts ...grpc.CallOption) (*Count, error) {
	out := new(Count)
	err := c.cc.Invoke(ctx, "/agones.dev.sdk.alpha.SDK/GetPlayerCount", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *sDKClient) IsPlayerConnected(ctx context.Context, in *PlayerID, opts ...grpc.CallOption) (*Bool, error) {
	out := new(Bool)
	err := c.cc.Invoke(ctx, "/agones.dev.sdk.alpha.SDK/IsPlayerConnected", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *sDKClient) GetConnectedPlayers(ctx context.Context, in *Empty, opts ...grpc.CallOption) (*PlayerIDList, error) {
	out := new(PlayerIDList)
	err := c.cc.Invoke(ctx, "/agones.dev.sdk.alpha.SDK/GetConnectedPlayers", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// SDKServer is the server API for SDK service.
type SDKServer interface {
	// PlayerConnect increases the SDK’s stored player count by one, and appends this playerID to GameServer.Status.Players.IDs.
	//
	// GameServer.Status.Players.Count and GameServer.Status.Players.IDs are then set to update the player count and id list a second from now,
	// unless there is already an update pending, in which case the update joins that batch operation.
	//
	// PlayerConnect returns true and adds the playerID to the list of playerIDs if this playerID was not already in the
	// list of connected playerIDs.
	//
	// If the playerID exists within the list of connected playerIDs, PlayerConnect will return false, and the list of
	// connected playerIDs will be left unchanged.
	//
	// An error will be returned if the playerID was not already in the list of connected playerIDs but the player capacity for
	// the server has been reached. The playerID will not be added to the list of playerIDs.
	//
	// Warning: Do not use this method if you are manually managing GameServer.Status.Players.IDs and GameServer.Status.Players.Count
	// through the Kubernetes API, as indeterminate results will occur.
	PlayerConnect(context.Context, *PlayerID) (*Bool, error)
	// Decreases the SDK’s stored player count by one, and removes the playerID from GameServer.Status.Players.IDs.
	//
	// GameServer.Status.Players.Count and GameServer.Status.Players.IDs are then set to update the player count and id list a second from now,
	// unless there is already an update pending, in which case the update joins that batch operation.
	//
	// PlayerDisconnect will return true and remove the supplied playerID from the list of connected playerIDs if the
	// playerID value exists within the list.
	//
	// If the playerID was not in the list of connected playerIDs, the call will return false, and the connected playerID list
	// will be left unchanged.
	//
	// Warning: Do not use this method if you are manually managing GameServer.status.players.IDs and GameServer.status.players.Count
	// through the Kubernetes API, as indeterminate results will occur.
	PlayerDisconnect(context.Context, *PlayerID) (*Bool, error)
	// Update the GameServer.Status.Players.Capacity value with a new capacity.
	SetPlayerCapacity(context.Context, *Count) (*Empty, error)
	// Retrieves the current player capacity. This is always accurate from what has been set through this SDK,
	// even if the value has yet to be updated on the GameServer status resource.
	//
	// If GameServer.Status.Players.Capacity is set manually through the Kubernetes API, use SDK.GameServer() or SDK.WatchGameServer() instead to view this value.
	GetPlayerCapacity(context.Context, *Empty) (*Count, error)
	// Retrieves the current player count. This is always accurate from what has been set through this SDK,
	// even if the value has yet to be updated on the GameServer status resource.
	//
	// If GameServer.Status.Players.Count is set manually through the Kubernetes API, use SDK.GameServer() or SDK.WatchGameServer() instead to view this value.
	GetPlayerCount(context.Context, *Empty) (*Count, error)
	// Returns if the playerID is currently connected to the GameServer. This is always accurate from what has been set through this SDK,
	// even if the value has yet to be updated on the GameServer status resource.
	//
	// If GameServer.Status.Players.IDs is set manually through the Kubernetes API, use SDK.GameServer() or SDK.WatchGameServer() instead to determine connected status.
	IsPlayerConnected(context.Context, *PlayerID) (*Bool, error)
	// Returns the list of the currently connected player ids. This is always accurate from what has been set through this SDK,
	// even if the value has yet to be updated on the GameServer status resource.
	//
	// If GameServer.Status.Players.IDs is set manually through the Kubernetes API, use SDK.GameServer() or SDK.WatchGameServer() instead to view this value.
	GetConnectedPlayers(context.Context, *Empty) (*PlayerIDList, error)
}

func RegisterSDKServer(s *grpc.Server, srv SDKServer) {
	s.RegisterService(&_SDK_serviceDesc, srv)
}

func _SDK_PlayerConnect_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(PlayerID)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(SDKServer).PlayerConnect(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/agones.dev.sdk.alpha.SDK/PlayerConnect",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(SDKServer).PlayerConnect(ctx, req.(*PlayerID))
	}
	return interceptor(ctx, in, info, handler)
}

func _SDK_PlayerDisconnect_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(PlayerID)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(SDKServer).PlayerDisconnect(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/agones.dev.sdk.alpha.SDK/PlayerDisconnect",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(SDKServer).PlayerDisconnect(ctx, req.(*PlayerID))
	}
	return interceptor(ctx, in, info, handler)
}

func _SDK_SetPlayerCapacity_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Count)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(SDKServer).SetPlayerCapacity(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/agones.dev.sdk.alpha.SDK/SetPlayerCapacity",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(SDKServer).SetPlayerCapacity(ctx, req.(*Count))
	}
	return interceptor(ctx, in, info, handler)
}

func _SDK_GetPlayerCapacity_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Empty)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(SDKServer).GetPlayerCapacity(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/agones.dev.sdk.alpha.SDK/GetPlayerCapacity",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(SDKServer).GetPlayerCapacity(ctx, req.(*Empty))
	}
	return interceptor(ctx, in, info, handler)
}

func _SDK_GetPlayerCount_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Empty)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(SDKServer).GetPlayerCount(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/agones.dev.sdk.alpha.SDK/GetPlayerCount",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(SDKServer).GetPlayerCount(ctx, req.(*Empty))
	}
	return interceptor(ctx, in, info, handler)
}

func _SDK_IsPlayerConnected_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(PlayerID)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(SDKServer).IsPlayerConnected(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/agones.dev.sdk.alpha.SDK/IsPlayerConnected",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(SDKServer).IsPlayerConnected(ctx, req.(*PlayerID))
	}
	return interceptor(ctx, in, info, handler)
}

func _SDK_GetConnectedPlayers_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Empty)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(SDKServer).GetConnectedPlayers(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/agones.dev.sdk.alpha.SDK/GetConnectedPlayers",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(SDKServer).GetConnectedPlayers(ctx, req.(*Empty))
	}
	return interceptor(ctx, in, info, handler)
}

var _SDK_serviceDesc = grpc.ServiceDesc{
	ServiceName: "agones.dev.sdk.alpha.SDK",
	HandlerType: (*SDKServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "PlayerConnect",
			Handler:    _SDK_PlayerConnect_Handler,
		},
		{
			MethodName: "PlayerDisconnect",
			Handler:    _SDK_PlayerDisconnect_Handler,
		},
		{
			MethodName: "SetPlayerCapacity",
			Handler:    _SDK_SetPlayerCapacity_Handler,
		},
		{
			MethodName: "GetPlayerCapacity",
			Handler:    _SDK_GetPlayerCapacity_Handler,
		},
		{
			MethodName: "GetPlayerCount",
			Handler:    _SDK_GetPlayerCount_Handler,
		},
		{
			MethodName: "IsPlayerConnected",
			Handler:    _SDK_IsPlayerConnected_Handler,
		},
		{
			MethodName: "GetConnectedPlayers",
			Handler:    _SDK_GetConnectedPlayers_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "alpha.proto",
}

func init() { proto.RegisterFile("alpha.proto", fileDescriptor_alpha_adf85771d71a9075) }

var fileDescriptor_alpha_adf85771d71a9075 = []byte{
	// 413 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xac, 0x93, 0xcf, 0xae, 0xd2, 0x40,
	0x14, 0xc6, 0x83, 0x50, 0xfe, 0x1c, 0xff, 0x44, 0x06, 0x50, 0x1c, 0x04, 0x71, 0x34, 0x86, 0xb0,
	0x68, 0x13, 0xdd, 0xb9, 0x04, 0x0c, 0x21, 0xba, 0x30, 0x65, 0xe7, 0x6e, 0x68, 0x27, 0xb5, 0xb1,
	0x74, 0x1a, 0x66, 0xd4, 0x10, 0xe2, 0xc6, 0x57, 0xf0, 0x29, 0x7c, 0x9e, 0xfb, 0x0a, 0xf7, 0x41,
	0x6e, 0xe6, 0x0c, 0x70, 0x03, 0x29, 0xe4, 0xe6, 0xde, 0xbb, 0x3b, 0xd3, 0x6f, 0xfa, 0xfd, 0xbe,
	0x93, 0x73, 0x06, 0x1e, 0xf2, 0x24, 0xfb, 0xce, 0xdd, 0x6c, 0x25, 0xb5, 0x24, 0x4d, 0x1e, 0xc9,
	0x54, 0x28, 0x37, 0x14, 0xbf, 0x5c, 0x15, 0xfe, 0x70, 0x51, 0xa3, 0x2f, 0x23, 0x29, 0xa3, 0x44,
	0x78, 0x3c, 0x8b, 0x3d, 0x9e, 0xa6, 0x52, 0x73, 0x1d, 0xcb, 0x54, 0xd9, 0x7f, 0x58, 0x05, 0x9c,
	0x4f, 0xcb, 0x4c, 0xaf, 0x59, 0x17, 0x9c, 0xb1, 0xfc, 0x99, 0x6a, 0xd2, 0x04, 0x27, 0x30, 0x45,
	0xbb, 0xd0, 0x2f, 0x0c, 0x8a, 0xbe, 0x3d, 0x30, 0x0a, 0xa5, 0x91, 0x94, 0x09, 0x21, 0x50, 0x5a,
	0x48, 0x99, 0xa0, 0x58, 0xf5, 0xb1, 0x66, 0xef, 0xa0, 0xfa, 0x35, 0xe1, 0x6b, 0xb1, 0x9a, 0x4d,
	0x08, 0x85, 0x6a, 0xb6, 0xad, 0xf1, 0x4e, 0xcd, 0xdf, 0x9f, 0x19, 0x83, 0x47, 0xbb, 0x7b, 0x5f,
	0x62, 0xa5, 0x8d, 0x57, 0x12, 0x2b, 0x03, 0x2a, 0x0e, 0x6a, 0x3e, 0xd6, 0xef, 0xff, 0x97, 0xa1,
	0x38, 0x9f, 0x7c, 0x26, 0x4b, 0x78, 0x6c, 0xef, 0x8e, 0x65, 0x9a, 0x8a, 0x40, 0x93, 0x9e, 0x9b,
	0xd7, 0x9d, 0xbb, 0x33, 0xa4, 0x34, 0x5f, 0x37, 0xa1, 0x59, 0xff, 0xef, 0xc5, 0xe5, 0xbf, 0x07,
	0x94, 0xb5, 0x3c, 0xfc, 0xe8, 0xd9, 0x44, 0x5e, 0x60, 0xad, 0x3f, 0x16, 0x86, 0x44, 0xc1, 0x53,
	0xeb, 0x34, 0x89, 0x55, 0x70, 0x0f, 0xc4, 0x37, 0x48, 0xec, 0xb2, 0xf6, 0x21, 0x31, 0xdc, 0xbb,
	0x1b, 0x68, 0x06, 0xf5, 0xb9, 0xd0, 0xdb, 0x36, 0x79, 0xc6, 0x83, 0x58, 0xaf, 0x49, 0x27, 0xdf,
	0x15, 0x67, 0x43, 0x4f, 0x88, 0x76, 0x82, 0xaf, 0x91, 0xd9, 0xa1, 0xcf, 0x8e, 0xba, 0xdc, 0x3a,
	0x1b, 0xe2, 0x12, 0xea, 0xd3, 0x9b, 0x12, 0xd1, 0x94, 0x9e, 0x8b, 0xc3, 0x7a, 0x48, 0x6c, 0x93,
	0x13, 0x44, 0x12, 0xc1, 0x93, 0x6b, 0x1c, 0x2e, 0xd7, 0xed, 0x59, 0x1d, 0x64, 0xb5, 0x48, 0xe3,
	0x78, 0x86, 0xc6, 0x76, 0x03, 0xf5, 0x99, 0x3a, 0xd8, 0x17, 0x11, 0xde, 0x69, 0x7e, 0x43, 0xa4,
	0xbd, 0x25, 0x2c, 0x77, 0x63, 0x44, 0xe8, 0x6d, 0x76, 0x5b, 0xfd, 0x87, 0xfc, 0x86, 0xc6, 0x54,
	0xe8, 0x3d, 0xd7, 0xfa, 0xab, 0xf3, 0xad, 0xb2, 0xf3, 0xd9, 0xcc, 0xf3, 0x60, 0xaf, 0x30, 0xc3,
	0x0b, 0xf2, 0xfc, 0x44, 0x86, 0x51, 0xe5, 0x9b, 0x83, 0xca, 0xa2, 0x8c, 0x6f, 0xf9, 0xc3, 0x55,
	0x00, 0x00, 0x00, 0xff, 0xff, 0x26, 0xa2, 0xb8, 0x96, 0x0e, 0x04, 0x00, 0x00,
}
