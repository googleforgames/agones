// Copyright 2018 Google Inc. All Rights Reserved.
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
// source: sdk.proto

package sdk

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"

import (
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

type Empty struct {
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *Empty) Reset()         { *m = Empty{} }
func (m *Empty) String() string { return proto.CompactTextString(m) }
func (*Empty) ProtoMessage()    {}
func (*Empty) Descriptor() ([]byte, []int) {
	return fileDescriptor_sdk_5516a740f4bef6e9, []int{0}
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

func init() {
	proto.RegisterType((*Empty)(nil), "stable.agones.dev.sdk.Empty")
}

// Reference imports to suppress errors if they are not otherwise used.
var _ context.Context
var _ grpc.ClientConn

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion4

// Client API for SDK service

type SDKClient interface {
	// Call when the GameServer is ready
	Ready(ctx context.Context, in *Empty, opts ...grpc.CallOption) (*Empty, error)
	// Call when the GmaeServer is shutting down
	Shutdown(ctx context.Context, in *Empty, opts ...grpc.CallOption) (*Empty, error)
	// Send a Empty every d Duration to declare that this GameSever is healthy
	Health(ctx context.Context, opts ...grpc.CallOption) (SDK_HealthClient, error)
}

type sDKClient struct {
	cc *grpc.ClientConn
}

func NewSDKClient(cc *grpc.ClientConn) SDKClient {
	return &sDKClient{cc}
}

func (c *sDKClient) Ready(ctx context.Context, in *Empty, opts ...grpc.CallOption) (*Empty, error) {
	out := new(Empty)
	err := grpc.Invoke(ctx, "/stable.agones.dev.sdk.SDK/Ready", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *sDKClient) Shutdown(ctx context.Context, in *Empty, opts ...grpc.CallOption) (*Empty, error) {
	out := new(Empty)
	err := grpc.Invoke(ctx, "/stable.agones.dev.sdk.SDK/Shutdown", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *sDKClient) Health(ctx context.Context, opts ...grpc.CallOption) (SDK_HealthClient, error) {
	stream, err := grpc.NewClientStream(ctx, &_SDK_serviceDesc.Streams[0], c.cc, "/stable.agones.dev.sdk.SDK/Health", opts...)
	if err != nil {
		return nil, err
	}
	x := &sDKHealthClient{stream}
	return x, nil
}

type SDK_HealthClient interface {
	Send(*Empty) error
	CloseAndRecv() (*Empty, error)
	grpc.ClientStream
}

type sDKHealthClient struct {
	grpc.ClientStream
}

func (x *sDKHealthClient) Send(m *Empty) error {
	return x.ClientStream.SendMsg(m)
}

func (x *sDKHealthClient) CloseAndRecv() (*Empty, error) {
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	m := new(Empty)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

// Server API for SDK service

type SDKServer interface {
	// Call when the GameServer is ready
	Ready(context.Context, *Empty) (*Empty, error)
	// Call when the GmaeServer is shutting down
	Shutdown(context.Context, *Empty) (*Empty, error)
	// Send a Empty every d Duration to declare that this GameSever is healthy
	Health(SDK_HealthServer) error
}

func RegisterSDKServer(s *grpc.Server, srv SDKServer) {
	s.RegisterService(&_SDK_serviceDesc, srv)
}

func _SDK_Ready_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Empty)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(SDKServer).Ready(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/stable.agones.dev.sdk.SDK/Ready",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(SDKServer).Ready(ctx, req.(*Empty))
	}
	return interceptor(ctx, in, info, handler)
}

func _SDK_Shutdown_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Empty)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(SDKServer).Shutdown(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/stable.agones.dev.sdk.SDK/Shutdown",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(SDKServer).Shutdown(ctx, req.(*Empty))
	}
	return interceptor(ctx, in, info, handler)
}

func _SDK_Health_Handler(srv interface{}, stream grpc.ServerStream) error {
	return srv.(SDKServer).Health(&sDKHealthServer{stream})
}

type SDK_HealthServer interface {
	SendAndClose(*Empty) error
	Recv() (*Empty, error)
	grpc.ServerStream
}

type sDKHealthServer struct {
	grpc.ServerStream
}

func (x *sDKHealthServer) SendAndClose(m *Empty) error {
	return x.ServerStream.SendMsg(m)
}

func (x *sDKHealthServer) Recv() (*Empty, error) {
	m := new(Empty)
	if err := x.ServerStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

var _SDK_serviceDesc = grpc.ServiceDesc{
	ServiceName: "stable.agones.dev.sdk.SDK",
	HandlerType: (*SDKServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Ready",
			Handler:    _SDK_Ready_Handler,
		},
		{
			MethodName: "Shutdown",
			Handler:    _SDK_Shutdown_Handler,
		},
	},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "Health",
			Handler:       _SDK_Health_Handler,
			ClientStreams: true,
		},
	},
	Metadata: "sdk.proto",
}

func init() { proto.RegisterFile("sdk.proto", fileDescriptor_sdk_5516a740f4bef6e9) }

var fileDescriptor_sdk_5516a740f4bef6e9 = []byte{
	// 136 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xe2, 0xe2, 0x2c, 0x4e, 0xc9, 0xd6,
	0x2b, 0x28, 0xca, 0x2f, 0xc9, 0x17, 0x12, 0x2d, 0x2e, 0x49, 0x4c, 0xca, 0x49, 0xd5, 0x4b, 0x4c,
	0xcf, 0xcf, 0x4b, 0x2d, 0xd6, 0x4b, 0x49, 0x2d, 0xd3, 0x2b, 0x4e, 0xc9, 0x56, 0x62, 0xe7, 0x62,
	0x75, 0xcd, 0x2d, 0x28, 0xa9, 0x34, 0x7a, 0xc0, 0xc8, 0xc5, 0x1c, 0xec, 0xe2, 0x2d, 0xe4, 0xca,
	0xc5, 0x1a, 0x94, 0x9a, 0x98, 0x52, 0x29, 0x24, 0xa3, 0x87, 0x55, 0x87, 0x1e, 0x58, 0xb9, 0x14,
	0x5e, 0x59, 0x25, 0x06, 0x21, 0x0f, 0x2e, 0x8e, 0xe0, 0x8c, 0xd2, 0x92, 0x94, 0xfc, 0xf2, 0x3c,
	0x8a, 0x4d, 0x62, 0xf3, 0x48, 0x4d, 0xcc, 0x29, 0xc9, 0xa0, 0xcc, 0x1c, 0x0d, 0x46, 0x27, 0xd6,
	0x28, 0xe6, 0xe2, 0x94, 0xec, 0x24, 0x36, 0x70, 0x80, 0x18, 0x03, 0x02, 0x00, 0x00, 0xff, 0xff,
	0x67, 0x99, 0xbb, 0xc6, 0x1d, 0x01, 0x00, 0x00,
}
