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
//
'use strict';
var alpha_pb = require('./alpha_pb.js');
var google_api_annotations_pb = require('./google/api/annotations_pb.js');
var protoc$gen$openapiv2_options_annotations_pb = require('./protoc-gen-openapiv2/options/annotations_pb.js');

function serialize_agones_dev_sdk_alpha_Bool(arg) {
  if (!(arg instanceof alpha_pb.Bool)) {
    throw new Error('Expected argument of type agones.dev.sdk.alpha.Bool');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_agones_dev_sdk_alpha_Bool(buffer_arg) {
  return alpha_pb.Bool.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_agones_dev_sdk_alpha_Count(arg) {
  if (!(arg instanceof alpha_pb.Count)) {
    throw new Error('Expected argument of type agones.dev.sdk.alpha.Count');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_agones_dev_sdk_alpha_Count(buffer_arg) {
  return alpha_pb.Count.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_agones_dev_sdk_alpha_Empty(arg) {
  if (!(arg instanceof alpha_pb.Empty)) {
    throw new Error('Expected argument of type agones.dev.sdk.alpha.Empty');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_agones_dev_sdk_alpha_Empty(buffer_arg) {
  return alpha_pb.Empty.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_agones_dev_sdk_alpha_PlayerID(arg) {
  if (!(arg instanceof alpha_pb.PlayerID)) {
    throw new Error('Expected argument of type agones.dev.sdk.alpha.PlayerID');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_agones_dev_sdk_alpha_PlayerID(buffer_arg) {
  return alpha_pb.PlayerID.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_agones_dev_sdk_alpha_PlayerIDList(arg) {
  if (!(arg instanceof alpha_pb.PlayerIDList)) {
    throw new Error('Expected argument of type agones.dev.sdk.alpha.PlayerIDList');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_agones_dev_sdk_alpha_PlayerIDList(buffer_arg) {
  return alpha_pb.PlayerIDList.deserializeBinary(new Uint8Array(buffer_arg));
}


// SDK service to be used in the GameServer SDK to the Pod Sidecar.
var SDKService = exports['agones.dev.sdk.alpha.SDK'] = {
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
playerConnect: {
    path: '/agones.dev.sdk.alpha.SDK/PlayerConnect',
    requestStream: false,
    responseStream: false,
    requestType: alpha_pb.PlayerID,
    responseType: alpha_pb.Bool,
    requestSerialize: serialize_agones_dev_sdk_alpha_PlayerID,
    requestDeserialize: deserialize_agones_dev_sdk_alpha_PlayerID,
    responseSerialize: serialize_agones_dev_sdk_alpha_Bool,
    responseDeserialize: deserialize_agones_dev_sdk_alpha_Bool,
  },
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
playerDisconnect: {
    path: '/agones.dev.sdk.alpha.SDK/PlayerDisconnect',
    requestStream: false,
    responseStream: false,
    requestType: alpha_pb.PlayerID,
    responseType: alpha_pb.Bool,
    requestSerialize: serialize_agones_dev_sdk_alpha_PlayerID,
    requestDeserialize: deserialize_agones_dev_sdk_alpha_PlayerID,
    responseSerialize: serialize_agones_dev_sdk_alpha_Bool,
    responseDeserialize: deserialize_agones_dev_sdk_alpha_Bool,
  },
  // Update the GameServer.Status.Players.Capacity value with a new capacity.
setPlayerCapacity: {
    path: '/agones.dev.sdk.alpha.SDK/SetPlayerCapacity',
    requestStream: false,
    responseStream: false,
    requestType: alpha_pb.Count,
    responseType: alpha_pb.Empty,
    requestSerialize: serialize_agones_dev_sdk_alpha_Count,
    requestDeserialize: deserialize_agones_dev_sdk_alpha_Count,
    responseSerialize: serialize_agones_dev_sdk_alpha_Empty,
    responseDeserialize: deserialize_agones_dev_sdk_alpha_Empty,
  },
  // Retrieves the current player capacity. This is always accurate from what has been set through this SDK,
// even if the value has yet to be updated on the GameServer status resource.
//
// If GameServer.Status.Players.Capacity is set manually through the Kubernetes API, use SDK.GameServer() or SDK.WatchGameServer() instead to view this value.
getPlayerCapacity: {
    path: '/agones.dev.sdk.alpha.SDK/GetPlayerCapacity',
    requestStream: false,
    responseStream: false,
    requestType: alpha_pb.Empty,
    responseType: alpha_pb.Count,
    requestSerialize: serialize_agones_dev_sdk_alpha_Empty,
    requestDeserialize: deserialize_agones_dev_sdk_alpha_Empty,
    responseSerialize: serialize_agones_dev_sdk_alpha_Count,
    responseDeserialize: deserialize_agones_dev_sdk_alpha_Count,
  },
  // Retrieves the current player count. This is always accurate from what has been set through this SDK,
// even if the value has yet to be updated on the GameServer status resource.
//
// If GameServer.Status.Players.Count is set manually through the Kubernetes API, use SDK.GameServer() or SDK.WatchGameServer() instead to view this value.
getPlayerCount: {
    path: '/agones.dev.sdk.alpha.SDK/GetPlayerCount',
    requestStream: false,
    responseStream: false,
    requestType: alpha_pb.Empty,
    responseType: alpha_pb.Count,
    requestSerialize: serialize_agones_dev_sdk_alpha_Empty,
    requestDeserialize: deserialize_agones_dev_sdk_alpha_Empty,
    responseSerialize: serialize_agones_dev_sdk_alpha_Count,
    responseDeserialize: deserialize_agones_dev_sdk_alpha_Count,
  },
  // Returns if the playerID is currently connected to the GameServer. This is always accurate from what has been set through this SDK,
// even if the value has yet to be updated on the GameServer status resource.
//
// If GameServer.Status.Players.IDs is set manually through the Kubernetes API, use SDK.GameServer() or SDK.WatchGameServer() instead to determine connected status.
isPlayerConnected: {
    path: '/agones.dev.sdk.alpha.SDK/IsPlayerConnected',
    requestStream: false,
    responseStream: false,
    requestType: alpha_pb.PlayerID,
    responseType: alpha_pb.Bool,
    requestSerialize: serialize_agones_dev_sdk_alpha_PlayerID,
    requestDeserialize: deserialize_agones_dev_sdk_alpha_PlayerID,
    responseSerialize: serialize_agones_dev_sdk_alpha_Bool,
    responseDeserialize: deserialize_agones_dev_sdk_alpha_Bool,
  },
  // Returns the list of the currently connected player ids. This is always accurate from what has been set through this SDK,
// even if the value has yet to be updated on the GameServer status resource.
//
// If GameServer.Status.Players.IDs is set manually through the Kubernetes API, use SDK.GameServer() or SDK.WatchGameServer() instead to view this value.
getConnectedPlayers: {
    path: '/agones.dev.sdk.alpha.SDK/GetConnectedPlayers',
    requestStream: false,
    responseStream: false,
    requestType: alpha_pb.Empty,
    responseType: alpha_pb.PlayerIDList,
    requestSerialize: serialize_agones_dev_sdk_alpha_Empty,
    requestDeserialize: deserialize_agones_dev_sdk_alpha_Empty,
    responseSerialize: serialize_agones_dev_sdk_alpha_PlayerIDList,
    responseDeserialize: deserialize_agones_dev_sdk_alpha_PlayerIDList,
  },
};

