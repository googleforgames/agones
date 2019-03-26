// Copyright 2019 Google Inc. All Rights Reserved.
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
use std::collections::HashMap;

use grpc::sdk;

#[derive(PartialEq, Clone, Default)]
pub struct GameServer {
    // message fields
    pub object_meta: Option<GameServerObjectMeta>,
    pub spec: Option<GameServerSpec>,
    pub status: Option<GameServerStatus>,
}

impl GameServer {
    pub fn from_message(msg: sdk::GameServer) -> GameServer {
        GameServer {
            object_meta: msg
                .object_meta
                .into_option()
                .map(|e| GameServerObjectMeta::from_message(e)),
            spec: msg
                .spec
                .into_option()
                .map(|e| GameServerSpec::from_message(e)),
            status: msg
                .status
                .into_option()
                .map(|e| GameServerStatus::from_message(e)),
        }
    }
}

#[derive(PartialEq, Clone, Default)]
pub struct GameServerObjectMeta {
    pub name: String,
    pub namespace: String,
    pub uid: String,
    pub resource_version: String,
    pub generation: i64,
    pub creation_timestamp: i64,
    pub deletion_timestamp: i64,
    pub annotations: HashMap<String, String>,
    pub labels: HashMap<String, String>,
}

impl GameServerObjectMeta {
    pub fn from_message(msg: sdk::GameServer_ObjectMeta) -> GameServerObjectMeta {
        GameServerObjectMeta {
            name: msg.name,
            namespace: msg.namespace,
            uid: msg.uid,
            resource_version: msg.resource_version,
            generation: msg.generation,
            creation_timestamp: msg.creation_timestamp,
            deletion_timestamp: msg.deletion_timestamp,
            annotations: msg.annotations,
            labels: msg.labels,
        }
    }
}

#[derive(PartialEq, Clone, Default)]
pub struct GameServerSpec {
    pub health: Option<GameServerSpecHealth>,
}

impl GameServerSpec {
    pub fn from_message(msg: sdk::GameServer_Spec) -> GameServerSpec {
        GameServerSpec {
            health: msg
                .health
                .into_option()
                .map(|e| GameServerSpecHealth::from_message(e)),
        }
    }
}

#[derive(PartialEq, Clone, Default)]
pub struct GameServerSpecHealth {
    // message fields
    pub disabled: bool,
    pub period_seconds: i32,
    pub failure_threshold: i32,
    pub initial_delay_seconds: i32,
}

impl GameServerSpecHealth {
    pub fn from_message(msg: sdk::GameServer_Spec_Health) -> GameServerSpecHealth {
        GameServerSpecHealth {
            disabled: msg.Disabled,
            period_seconds: msg.PeriodSeconds,
            failure_threshold: msg.FailureThreshold,
            initial_delay_seconds: msg.InitialDelaySeconds,
        }
    }
}

#[derive(PartialEq, Clone, Default)]
pub struct GameServerStatus {
    pub state: String,
    pub address: String,
    pub ports: Vec<GameServerStatusPort>,
}

impl GameServerStatus {
    pub fn from_message(msg: sdk::GameServer_Status) -> GameServerStatus {
        GameServerStatus {
            state: msg.state,
            address: msg.address,
            ports: msg
                .ports
                .into_iter()
                .map(|e| GameServerStatusPort::from_message(e))
                .collect(),
        }
    }
}

#[derive(PartialEq, Clone, Default)]
pub struct GameServerStatusPort {
    pub name: String,
    pub port: i32,
}

impl GameServerStatusPort {
    pub fn from_message(msg: sdk::GameServer_Status_Port) -> GameServerStatusPort {
        GameServerStatusPort {
            name: msg.name,
            port: msg.port,
        }
    }
}
