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
// Copyright 2020 Google LLC All Rights Reserved.
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

// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.28.1
// 	protoc        v3.21.6
// source: alpha.proto

package alpha

import (
	reflect "reflect"
	sync "sync"

	_ "github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2/options"
	_ "google.golang.org/genproto/googleapis/api/annotations"
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

// I am Empty
type Empty struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields
}

func (x *Empty) Reset() {
	*x = Empty{}
	if protoimpl.UnsafeEnabled {
		mi := &file_alpha_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Empty) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Empty) ProtoMessage() {}

func (x *Empty) ProtoReflect() protoreflect.Message {
	mi := &file_alpha_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Empty.ProtoReflect.Descriptor instead.
func (*Empty) Descriptor() ([]byte, []int) {
	return file_alpha_proto_rawDescGZIP(), []int{0}
}

// Store a count variable.
type Count struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Count int64 `protobuf:"varint,1,opt,name=count,proto3" json:"count,omitempty"`
}

func (x *Count) Reset() {
	*x = Count{}
	if protoimpl.UnsafeEnabled {
		mi := &file_alpha_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Count) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Count) ProtoMessage() {}

func (x *Count) ProtoReflect() protoreflect.Message {
	mi := &file_alpha_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Count.ProtoReflect.Descriptor instead.
func (*Count) Descriptor() ([]byte, []int) {
	return file_alpha_proto_rawDescGZIP(), []int{1}
}

func (x *Count) GetCount() int64 {
	if x != nil {
		return x.Count
	}
	return 0
}

// Store a boolean result
type Bool struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Bool bool `protobuf:"varint,1,opt,name=bool,proto3" json:"bool,omitempty"`
}

func (x *Bool) Reset() {
	*x = Bool{}
	if protoimpl.UnsafeEnabled {
		mi := &file_alpha_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Bool) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Bool) ProtoMessage() {}

func (x *Bool) ProtoReflect() protoreflect.Message {
	mi := &file_alpha_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Bool.ProtoReflect.Descriptor instead.
func (*Bool) Descriptor() ([]byte, []int) {
	return file_alpha_proto_rawDescGZIP(), []int{2}
}

func (x *Bool) GetBool() bool {
	if x != nil {
		return x.Bool
	}
	return false
}

// The unique identifier for a given player.
type PlayerID struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	PlayerID string `protobuf:"bytes,1,opt,name=playerID,proto3" json:"playerID,omitempty"`
}

func (x *PlayerID) Reset() {
	*x = PlayerID{}
	if protoimpl.UnsafeEnabled {
		mi := &file_alpha_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *PlayerID) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*PlayerID) ProtoMessage() {}

func (x *PlayerID) ProtoReflect() protoreflect.Message {
	mi := &file_alpha_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use PlayerID.ProtoReflect.Descriptor instead.
func (*PlayerID) Descriptor() ([]byte, []int) {
	return file_alpha_proto_rawDescGZIP(), []int{3}
}

func (x *PlayerID) GetPlayerID() string {
	if x != nil {
		return x.PlayerID
	}
	return ""
}

// List of Player IDs
type PlayerIDList struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	List []string `protobuf:"bytes,1,rep,name=list,proto3" json:"list,omitempty"`
}

func (x *PlayerIDList) Reset() {
	*x = PlayerIDList{}
	if protoimpl.UnsafeEnabled {
		mi := &file_alpha_proto_msgTypes[4]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *PlayerIDList) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*PlayerIDList) ProtoMessage() {}

func (x *PlayerIDList) ProtoReflect() protoreflect.Message {
	mi := &file_alpha_proto_msgTypes[4]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use PlayerIDList.ProtoReflect.Descriptor instead.
func (*PlayerIDList) Descriptor() ([]byte, []int) {
	return file_alpha_proto_rawDescGZIP(), []int{4}
}

func (x *PlayerIDList) GetList() []string {
	if x != nil {
		return x.List
	}
	return nil
}

var File_alpha_proto protoreflect.FileDescriptor

var file_alpha_proto_rawDesc = []byte{
	0x0a, 0x0b, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x14, 0x61,
	0x67, 0x6f, 0x6e, 0x65, 0x73, 0x2e, 0x64, 0x65, 0x76, 0x2e, 0x73, 0x64, 0x6b, 0x2e, 0x61, 0x6c,
	0x70, 0x68, 0x61, 0x1a, 0x1c, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2f, 0x61, 0x70, 0x69, 0x2f,
	0x61, 0x6e, 0x6e, 0x6f, 0x74, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x1a, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x63, 0x2d, 0x67, 0x65, 0x6e, 0x2d, 0x6f, 0x70,
	0x65, 0x6e, 0x61, 0x70, 0x69, 0x76, 0x32, 0x2f, 0x6f, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x2f,
	0x61, 0x6e, 0x6e, 0x6f, 0x74, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x22, 0x07, 0x0a, 0x05, 0x45, 0x6d, 0x70, 0x74, 0x79, 0x22, 0x1d, 0x0a, 0x05, 0x43, 0x6f,
	0x75, 0x6e, 0x74, 0x12, 0x14, 0x0a, 0x05, 0x63, 0x6f, 0x75, 0x6e, 0x74, 0x18, 0x01, 0x20, 0x01,
	0x28, 0x03, 0x52, 0x05, 0x63, 0x6f, 0x75, 0x6e, 0x74, 0x22, 0x29, 0x0a, 0x04, 0x42, 0x6f, 0x6f,
	0x6c, 0x12, 0x21, 0x0a, 0x04, 0x62, 0x6f, 0x6f, 0x6c, 0x18, 0x01, 0x20, 0x01, 0x28, 0x08, 0x42,
	0x0d, 0x92, 0x41, 0x0a, 0xa2, 0x02, 0x07, 0x62, 0x6f, 0x6f, 0x6c, 0x65, 0x61, 0x6e, 0x52, 0x04,
	0x62, 0x6f, 0x6f, 0x6c, 0x22, 0x26, 0x0a, 0x08, 0x50, 0x6c, 0x61, 0x79, 0x65, 0x72, 0x49, 0x44,
	0x12, 0x1a, 0x0a, 0x08, 0x70, 0x6c, 0x61, 0x79, 0x65, 0x72, 0x49, 0x44, 0x18, 0x01, 0x20, 0x01,
	0x28, 0x09, 0x52, 0x08, 0x70, 0x6c, 0x61, 0x79, 0x65, 0x72, 0x49, 0x44, 0x22, 0x22, 0x0a, 0x0c,
	0x50, 0x6c, 0x61, 0x79, 0x65, 0x72, 0x49, 0x44, 0x4c, 0x69, 0x73, 0x74, 0x12, 0x12, 0x0a, 0x04,
	0x6c, 0x69, 0x73, 0x74, 0x18, 0x01, 0x20, 0x03, 0x28, 0x09, 0x52, 0x04, 0x6c, 0x69, 0x73, 0x74,
	0x32, 0xa9, 0x06, 0x0a, 0x03, 0x53, 0x44, 0x4b, 0x12, 0x6d, 0x0a, 0x0d, 0x50, 0x6c, 0x61, 0x79,
	0x65, 0x72, 0x43, 0x6f, 0x6e, 0x6e, 0x65, 0x63, 0x74, 0x12, 0x1e, 0x2e, 0x61, 0x67, 0x6f, 0x6e,
	0x65, 0x73, 0x2e, 0x64, 0x65, 0x76, 0x2e, 0x73, 0x64, 0x6b, 0x2e, 0x61, 0x6c, 0x70, 0x68, 0x61,
	0x2e, 0x50, 0x6c, 0x61, 0x79, 0x65, 0x72, 0x49, 0x44, 0x1a, 0x1a, 0x2e, 0x61, 0x67, 0x6f, 0x6e,
	0x65, 0x73, 0x2e, 0x64, 0x65, 0x76, 0x2e, 0x73, 0x64, 0x6b, 0x2e, 0x61, 0x6c, 0x70, 0x68, 0x61,
	0x2e, 0x42, 0x6f, 0x6f, 0x6c, 0x22, 0x20, 0x82, 0xd3, 0xe4, 0x93, 0x02, 0x1a, 0x22, 0x15, 0x2f,
	0x61, 0x6c, 0x70, 0x68, 0x61, 0x2f, 0x70, 0x6c, 0x61, 0x79, 0x65, 0x72, 0x2f, 0x63, 0x6f, 0x6e,
	0x6e, 0x65, 0x63, 0x74, 0x3a, 0x01, 0x2a, 0x12, 0x73, 0x0a, 0x10, 0x50, 0x6c, 0x61, 0x79, 0x65,
	0x72, 0x44, 0x69, 0x73, 0x63, 0x6f, 0x6e, 0x6e, 0x65, 0x63, 0x74, 0x12, 0x1e, 0x2e, 0x61, 0x67,
	0x6f, 0x6e, 0x65, 0x73, 0x2e, 0x64, 0x65, 0x76, 0x2e, 0x73, 0x64, 0x6b, 0x2e, 0x61, 0x6c, 0x70,
	0x68, 0x61, 0x2e, 0x50, 0x6c, 0x61, 0x79, 0x65, 0x72, 0x49, 0x44, 0x1a, 0x1a, 0x2e, 0x61, 0x67,
	0x6f, 0x6e, 0x65, 0x73, 0x2e, 0x64, 0x65, 0x76, 0x2e, 0x73, 0x64, 0x6b, 0x2e, 0x61, 0x6c, 0x70,
	0x68, 0x61, 0x2e, 0x42, 0x6f, 0x6f, 0x6c, 0x22, 0x23, 0x82, 0xd3, 0xe4, 0x93, 0x02, 0x1d, 0x22,
	0x18, 0x2f, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x2f, 0x70, 0x6c, 0x61, 0x79, 0x65, 0x72, 0x2f, 0x64,
	0x69, 0x73, 0x63, 0x6f, 0x6e, 0x6e, 0x65, 0x63, 0x74, 0x3a, 0x01, 0x2a, 0x12, 0x70, 0x0a, 0x11,
	0x53, 0x65, 0x74, 0x50, 0x6c, 0x61, 0x79, 0x65, 0x72, 0x43, 0x61, 0x70, 0x61, 0x63, 0x69, 0x74,
	0x79, 0x12, 0x1b, 0x2e, 0x61, 0x67, 0x6f, 0x6e, 0x65, 0x73, 0x2e, 0x64, 0x65, 0x76, 0x2e, 0x73,
	0x64, 0x6b, 0x2e, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x2e, 0x43, 0x6f, 0x75, 0x6e, 0x74, 0x1a, 0x1b,
	0x2e, 0x61, 0x67, 0x6f, 0x6e, 0x65, 0x73, 0x2e, 0x64, 0x65, 0x76, 0x2e, 0x73, 0x64, 0x6b, 0x2e,
	0x61, 0x6c, 0x70, 0x68, 0x61, 0x2e, 0x45, 0x6d, 0x70, 0x74, 0x79, 0x22, 0x21, 0x82, 0xd3, 0xe4,
	0x93, 0x02, 0x1b, 0x1a, 0x16, 0x2f, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x2f, 0x70, 0x6c, 0x61, 0x79,
	0x65, 0x72, 0x2f, 0x63, 0x61, 0x70, 0x61, 0x63, 0x69, 0x74, 0x79, 0x3a, 0x01, 0x2a, 0x12, 0x6d,
	0x0a, 0x11, 0x47, 0x65, 0x74, 0x50, 0x6c, 0x61, 0x79, 0x65, 0x72, 0x43, 0x61, 0x70, 0x61, 0x63,
	0x69, 0x74, 0x79, 0x12, 0x1b, 0x2e, 0x61, 0x67, 0x6f, 0x6e, 0x65, 0x73, 0x2e, 0x64, 0x65, 0x76,
	0x2e, 0x73, 0x64, 0x6b, 0x2e, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x2e, 0x45, 0x6d, 0x70, 0x74, 0x79,
	0x1a, 0x1b, 0x2e, 0x61, 0x67, 0x6f, 0x6e, 0x65, 0x73, 0x2e, 0x64, 0x65, 0x76, 0x2e, 0x73, 0x64,
	0x6b, 0x2e, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x2e, 0x43, 0x6f, 0x75, 0x6e, 0x74, 0x22, 0x1e, 0x82,
	0xd3, 0xe4, 0x93, 0x02, 0x18, 0x12, 0x16, 0x2f, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x2f, 0x70, 0x6c,
	0x61, 0x79, 0x65, 0x72, 0x2f, 0x63, 0x61, 0x70, 0x61, 0x63, 0x69, 0x74, 0x79, 0x12, 0x67, 0x0a,
	0x0e, 0x47, 0x65, 0x74, 0x50, 0x6c, 0x61, 0x79, 0x65, 0x72, 0x43, 0x6f, 0x75, 0x6e, 0x74, 0x12,
	0x1b, 0x2e, 0x61, 0x67, 0x6f, 0x6e, 0x65, 0x73, 0x2e, 0x64, 0x65, 0x76, 0x2e, 0x73, 0x64, 0x6b,
	0x2e, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x2e, 0x45, 0x6d, 0x70, 0x74, 0x79, 0x1a, 0x1b, 0x2e, 0x61,
	0x67, 0x6f, 0x6e, 0x65, 0x73, 0x2e, 0x64, 0x65, 0x76, 0x2e, 0x73, 0x64, 0x6b, 0x2e, 0x61, 0x6c,
	0x70, 0x68, 0x61, 0x2e, 0x43, 0x6f, 0x75, 0x6e, 0x74, 0x22, 0x1b, 0x82, 0xd3, 0xe4, 0x93, 0x02,
	0x15, 0x12, 0x13, 0x2f, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x2f, 0x70, 0x6c, 0x61, 0x79, 0x65, 0x72,
	0x2f, 0x63, 0x6f, 0x75, 0x6e, 0x74, 0x12, 0x7b, 0x0a, 0x11, 0x49, 0x73, 0x50, 0x6c, 0x61, 0x79,
	0x65, 0x72, 0x43, 0x6f, 0x6e, 0x6e, 0x65, 0x63, 0x74, 0x65, 0x64, 0x12, 0x1e, 0x2e, 0x61, 0x67,
	0x6f, 0x6e, 0x65, 0x73, 0x2e, 0x64, 0x65, 0x76, 0x2e, 0x73, 0x64, 0x6b, 0x2e, 0x61, 0x6c, 0x70,
	0x68, 0x61, 0x2e, 0x50, 0x6c, 0x61, 0x79, 0x65, 0x72, 0x49, 0x44, 0x1a, 0x1a, 0x2e, 0x61, 0x67,
	0x6f, 0x6e, 0x65, 0x73, 0x2e, 0x64, 0x65, 0x76, 0x2e, 0x73, 0x64, 0x6b, 0x2e, 0x61, 0x6c, 0x70,
	0x68, 0x61, 0x2e, 0x42, 0x6f, 0x6f, 0x6c, 0x22, 0x2a, 0x82, 0xd3, 0xe4, 0x93, 0x02, 0x24, 0x12,
	0x22, 0x2f, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x2f, 0x70, 0x6c, 0x61, 0x79, 0x65, 0x72, 0x2f, 0x63,
	0x6f, 0x6e, 0x6e, 0x65, 0x63, 0x74, 0x65, 0x64, 0x2f, 0x7b, 0x70, 0x6c, 0x61, 0x79, 0x65, 0x72,
	0x49, 0x44, 0x7d, 0x12, 0x77, 0x0a, 0x13, 0x47, 0x65, 0x74, 0x43, 0x6f, 0x6e, 0x6e, 0x65, 0x63,
	0x74, 0x65, 0x64, 0x50, 0x6c, 0x61, 0x79, 0x65, 0x72, 0x73, 0x12, 0x1b, 0x2e, 0x61, 0x67, 0x6f,
	0x6e, 0x65, 0x73, 0x2e, 0x64, 0x65, 0x76, 0x2e, 0x73, 0x64, 0x6b, 0x2e, 0x61, 0x6c, 0x70, 0x68,
	0x61, 0x2e, 0x45, 0x6d, 0x70, 0x74, 0x79, 0x1a, 0x22, 0x2e, 0x61, 0x67, 0x6f, 0x6e, 0x65, 0x73,
	0x2e, 0x64, 0x65, 0x76, 0x2e, 0x73, 0x64, 0x6b, 0x2e, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x2e, 0x50,
	0x6c, 0x61, 0x79, 0x65, 0x72, 0x49, 0x44, 0x4c, 0x69, 0x73, 0x74, 0x22, 0x1f, 0x82, 0xd3, 0xe4,
	0x93, 0x02, 0x19, 0x12, 0x17, 0x2f, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x2f, 0x70, 0x6c, 0x61, 0x79,
	0x65, 0x72, 0x2f, 0x63, 0x6f, 0x6e, 0x6e, 0x65, 0x63, 0x74, 0x65, 0x64, 0x42, 0x53, 0x5a, 0x07,
	0x2e, 0x2f, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x92, 0x41, 0x47, 0x12, 0x1e, 0x0a, 0x0b, 0x61, 0x6c,
	0x70, 0x68, 0x61, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x32, 0x0f, 0x76, 0x65, 0x72, 0x73, 0x69,
	0x6f, 0x6e, 0x20, 0x6e, 0x6f, 0x74, 0x20, 0x73, 0x65, 0x74, 0x2a, 0x01, 0x01, 0x32, 0x10, 0x61,
	0x70, 0x70, 0x6c, 0x69, 0x63, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x2f, 0x6a, 0x73, 0x6f, 0x6e, 0x3a,
	0x10, 0x61, 0x70, 0x70, 0x6c, 0x69, 0x63, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x2f, 0x6a, 0x73, 0x6f,
	0x6e, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_alpha_proto_rawDescOnce sync.Once
	file_alpha_proto_rawDescData = file_alpha_proto_rawDesc
)

func file_alpha_proto_rawDescGZIP() []byte {
	file_alpha_proto_rawDescOnce.Do(func() {
		file_alpha_proto_rawDescData = protoimpl.X.CompressGZIP(file_alpha_proto_rawDescData)
	})
	return file_alpha_proto_rawDescData
}

var file_alpha_proto_msgTypes = make([]protoimpl.MessageInfo, 5)
var file_alpha_proto_goTypes = []interface{}{
	(*Empty)(nil),        // 0: agones.dev.sdk.alpha.Empty
	(*Count)(nil),        // 1: agones.dev.sdk.alpha.Count
	(*Bool)(nil),         // 2: agones.dev.sdk.alpha.Bool
	(*PlayerID)(nil),     // 3: agones.dev.sdk.alpha.PlayerID
	(*PlayerIDList)(nil), // 4: agones.dev.sdk.alpha.PlayerIDList
}
var file_alpha_proto_depIdxs = []int32{
	3, // 0: agones.dev.sdk.alpha.SDK.PlayerConnect:input_type -> agones.dev.sdk.alpha.PlayerID
	3, // 1: agones.dev.sdk.alpha.SDK.PlayerDisconnect:input_type -> agones.dev.sdk.alpha.PlayerID
	1, // 2: agones.dev.sdk.alpha.SDK.SetPlayerCapacity:input_type -> agones.dev.sdk.alpha.Count
	0, // 3: agones.dev.sdk.alpha.SDK.GetPlayerCapacity:input_type -> agones.dev.sdk.alpha.Empty
	0, // 4: agones.dev.sdk.alpha.SDK.GetPlayerCount:input_type -> agones.dev.sdk.alpha.Empty
	3, // 5: agones.dev.sdk.alpha.SDK.IsPlayerConnected:input_type -> agones.dev.sdk.alpha.PlayerID
	0, // 6: agones.dev.sdk.alpha.SDK.GetConnectedPlayers:input_type -> agones.dev.sdk.alpha.Empty
	2, // 7: agones.dev.sdk.alpha.SDK.PlayerConnect:output_type -> agones.dev.sdk.alpha.Bool
	2, // 8: agones.dev.sdk.alpha.SDK.PlayerDisconnect:output_type -> agones.dev.sdk.alpha.Bool
	0, // 9: agones.dev.sdk.alpha.SDK.SetPlayerCapacity:output_type -> agones.dev.sdk.alpha.Empty
	1, // 10: agones.dev.sdk.alpha.SDK.GetPlayerCapacity:output_type -> agones.dev.sdk.alpha.Count
	1, // 11: agones.dev.sdk.alpha.SDK.GetPlayerCount:output_type -> agones.dev.sdk.alpha.Count
	2, // 12: agones.dev.sdk.alpha.SDK.IsPlayerConnected:output_type -> agones.dev.sdk.alpha.Bool
	4, // 13: agones.dev.sdk.alpha.SDK.GetConnectedPlayers:output_type -> agones.dev.sdk.alpha.PlayerIDList
	7, // [7:14] is the sub-list for method output_type
	0, // [0:7] is the sub-list for method input_type
	0, // [0:0] is the sub-list for extension type_name
	0, // [0:0] is the sub-list for extension extendee
	0, // [0:0] is the sub-list for field type_name
}

func init() { file_alpha_proto_init() }
func file_alpha_proto_init() {
	if File_alpha_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_alpha_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Empty); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_alpha_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Count); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_alpha_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Bool); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_alpha_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*PlayerID); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_alpha_proto_msgTypes[4].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*PlayerIDList); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_alpha_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   5,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_alpha_proto_goTypes,
		DependencyIndexes: file_alpha_proto_depIdxs,
		MessageInfos:      file_alpha_proto_msgTypes,
	}.Build()
	File_alpha_proto = out.File
	file_alpha_proto_rawDesc = nil
	file_alpha_proto_goTypes = nil
	file_alpha_proto_depIdxs = nil
}
