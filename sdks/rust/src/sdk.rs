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


use std::sync::{Arc, Mutex};
use std::time::Duration;
use grpcio;
use futures::{Future, Sink};

use errors::*;
use grpc::sdk as sdk;
use grpc::sdk_grpc as sdk_grpc;
use protobuf::Message;

const PORT: i32 = 59357;

/// SDK is an instance of the Agones SDK
pub struct Sdk {
    client : Arc<sdk_grpc::SdkClient>,
    health : Arc<Mutex<Option<grpcio::ClientCStreamSender<sdk::Empty>>>>,
}

impl Sdk {

    /// Starts a new SDK instance, and connects to localhost on port 59357.
    /// Blocks until connection and handshake are made.
    /// Times out after 30 seconds.
    pub fn new() -> Result<Sdk> {
        let addr = format!("localhost:{}", PORT);
        let env = Arc::new(grpcio::EnvBuilder::new().build());
        let ch = grpcio::ChannelBuilder::new(env).keepalive_timeout(Duration::new(30, 0)).connect(&addr);
        let cli = sdk_grpc::SdkClient::new(ch);
        let req = sdk::Empty::new();
        let _ = cli.ready(&req).map(Box::new)?;
        let (sender, _) = cli.health()?;
        Ok(Sdk{client: Arc::new(cli), health: Arc::new(Mutex::new(Some(sender)))})
    }

    /// Marks the Game Server as ready to receive connections
    pub fn ready(&self) -> Result<()> {
        let req = sdk::Empty::default_instance();
        let res = self.client.ready(req).map(|_| ())?;
        Ok(res)
    }

    /// Marks the Game Server as ready to shutdown
    pub fn shutdown(&self) -> Result<()> {
        let req = sdk::Empty::default_instance();
        let res = self.client.shutdown(req).map(|_| ())?;
        Ok(res)
    }

    /// Sends a ping to the health check to indicate that this server is healthy
    pub fn health(mut self) -> (Self, Result<()>) {
        // Avoid `cannot move out of borrowed content` compile error for self.health
        let h = self.health.lock().unwrap().take();
        if h.is_none() {
            return (self, Err(ErrorKind::HealthPingConnectionFailure("failed to hold client stream for health ping".to_string()).into()));
        }
        let h : grpcio::ClientCStreamSender<sdk::Empty> = h.unwrap().into();

        let req = sdk::Empty::new();
        match h.send((req, grpcio::WriteFlags::default())).wait() {
            Ok(h) => {
                self.health = Arc::new(Mutex::new(Some(h)));
                (self, Ok(()))
            },
            Err(e) => {
                (self, Err(ErrorKind::Grpc(e).into()))
            },
        }
    }
}

impl Clone for Sdk {
    fn clone(&self) -> Self {
        Self {
            client: Arc::clone(&self.client),
            health: self.health.clone(),
        }
    }
}
