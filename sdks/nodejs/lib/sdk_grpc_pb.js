// Copyright 2024 Google LLC All Rights Reserved.
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
// GENERATED CODE -- DO NOT EDIT!

// Original file comments:
// Copyright 2017 Google LLC All Rights Reserved.
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
//
'use strict';
var sdk_pb = require('./sdk_pb.js');
var google_api_annotations_pb = require('./google/api/annotations_pb.js');
var protoc$gen$openapiv2_options_annotations_pb = require('./protoc-gen-openapiv2/options/annotations_pb.js');

function serialize_agones_dev_sdk_Duration(arg) {
  if (!(arg instanceof sdk_pb.Duration)) {
    throw new Error('Expected argument of type agones.dev.sdk.Duration');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_agones_dev_sdk_Duration(buffer_arg) {
  return sdk_pb.Duration.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_agones_dev_sdk_Empty(arg) {
  if (!(arg instanceof sdk_pb.Empty)) {
    throw new Error('Expected argument of type agones.dev.sdk.Empty');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_agones_dev_sdk_Empty(buffer_arg) {
  return sdk_pb.Empty.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_agones_dev_sdk_GameServer(arg) {
  if (!(arg instanceof sdk_pb.GameServer)) {
    throw new Error('Expected argument of type agones.dev.sdk.GameServer');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_agones_dev_sdk_GameServer(buffer_arg) {
  return sdk_pb.GameServer.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_agones_dev_sdk_KeyValue(arg) {
  if (!(arg instanceof sdk_pb.KeyValue)) {
    throw new Error('Expected argument of type agones.dev.sdk.KeyValue');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_agones_dev_sdk_KeyValue(buffer_arg) {
  return sdk_pb.KeyValue.deserializeBinary(new Uint8Array(buffer_arg));
}


// SDK service to be used in the GameServer SDK to the Pod Sidecar
var SDKService = exports['agones.dev.sdk.SDK'] = {
  // Call when the GameServer is ready
ready: {
    path: '/agones.dev.sdk.SDK/Ready',
    requestStream: false,
    responseStream: false,
    requestType: sdk_pb.Empty,
    responseType: sdk_pb.Empty,
    requestSerialize: serialize_agones_dev_sdk_Empty,
    requestDeserialize: deserialize_agones_dev_sdk_Empty,
    responseSerialize: serialize_agones_dev_sdk_Empty,
    responseDeserialize: deserialize_agones_dev_sdk_Empty,
  },
  // Call to self Allocation the GameServer
allocate: {
    path: '/agones.dev.sdk.SDK/Allocate',
    requestStream: false,
    responseStream: false,
    requestType: sdk_pb.Empty,
    responseType: sdk_pb.Empty,
    requestSerialize: serialize_agones_dev_sdk_Empty,
    requestDeserialize: deserialize_agones_dev_sdk_Empty,
    responseSerialize: serialize_agones_dev_sdk_Empty,
    responseDeserialize: deserialize_agones_dev_sdk_Empty,
  },
  // Call when the GameServer is shutting down
shutdown: {
    path: '/agones.dev.sdk.SDK/Shutdown',
    requestStream: false,
    responseStream: false,
    requestType: sdk_pb.Empty,
    responseType: sdk_pb.Empty,
    requestSerialize: serialize_agones_dev_sdk_Empty,
    requestDeserialize: deserialize_agones_dev_sdk_Empty,
    responseSerialize: serialize_agones_dev_sdk_Empty,
    responseDeserialize: deserialize_agones_dev_sdk_Empty,
  },
  // Send a Empty every d Duration to declare that this GameSever is healthy
health: {
    path: '/agones.dev.sdk.SDK/Health',
    requestStream: true,
    responseStream: false,
    requestType: sdk_pb.Empty,
    responseType: sdk_pb.Empty,
    requestSerialize: serialize_agones_dev_sdk_Empty,
    requestDeserialize: deserialize_agones_dev_sdk_Empty,
    responseSerialize: serialize_agones_dev_sdk_Empty,
    responseDeserialize: deserialize_agones_dev_sdk_Empty,
  },
  // Retrieve the current GameServer data
getGameServer: {
    path: '/agones.dev.sdk.SDK/GetGameServer',
    requestStream: false,
    responseStream: false,
    requestType: sdk_pb.Empty,
    responseType: sdk_pb.GameServer,
    requestSerialize: serialize_agones_dev_sdk_Empty,
    requestDeserialize: deserialize_agones_dev_sdk_Empty,
    responseSerialize: serialize_agones_dev_sdk_GameServer,
    responseDeserialize: deserialize_agones_dev_sdk_GameServer,
  },
  // Send GameServer details whenever the GameServer is updated
watchGameServer: {
    path: '/agones.dev.sdk.SDK/WatchGameServer',
    requestStream: false,
    responseStream: true,
    requestType: sdk_pb.Empty,
    responseType: sdk_pb.GameServer,
    requestSerialize: serialize_agones_dev_sdk_Empty,
    requestDeserialize: deserialize_agones_dev_sdk_Empty,
    responseSerialize: serialize_agones_dev_sdk_GameServer,
    responseDeserialize: deserialize_agones_dev_sdk_GameServer,
  },
  // Apply a Label to the backing GameServer metadata
setLabel: {
    path: '/agones.dev.sdk.SDK/SetLabel',
    requestStream: false,
    responseStream: false,
    requestType: sdk_pb.KeyValue,
    responseType: sdk_pb.Empty,
    requestSerialize: serialize_agones_dev_sdk_KeyValue,
    requestDeserialize: deserialize_agones_dev_sdk_KeyValue,
    responseSerialize: serialize_agones_dev_sdk_Empty,
    responseDeserialize: deserialize_agones_dev_sdk_Empty,
  },
  // Apply a Annotation to the backing GameServer metadata
setAnnotation: {
    path: '/agones.dev.sdk.SDK/SetAnnotation',
    requestStream: false,
    responseStream: false,
    requestType: sdk_pb.KeyValue,
    responseType: sdk_pb.Empty,
    requestSerialize: serialize_agones_dev_sdk_KeyValue,
    requestDeserialize: deserialize_agones_dev_sdk_KeyValue,
    responseSerialize: serialize_agones_dev_sdk_Empty,
    responseDeserialize: deserialize_agones_dev_sdk_Empty,
  },
  // Marks the GameServer as the Reserved state for Duration
reserve: {
    path: '/agones.dev.sdk.SDK/Reserve',
    requestStream: false,
    responseStream: false,
    requestType: sdk_pb.Duration,
    responseType: sdk_pb.Empty,
    requestSerialize: serialize_agones_dev_sdk_Duration,
    requestDeserialize: deserialize_agones_dev_sdk_Duration,
    responseSerialize: serialize_agones_dev_sdk_Empty,
    responseDeserialize: deserialize_agones_dev_sdk_Empty,
  },
};

