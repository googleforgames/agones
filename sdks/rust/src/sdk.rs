// Copyright 2018 Google LLC All Rights Reserved.
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

use std::env;
use std::sync::{Arc, Mutex};
use std::thread::sleep;
use std::time::Duration;

use futures::{Future, Sink, Stream};
use grpcio;
use protobuf::Message;

use alpha::*;
use errors::*;
use grpc::sdk;
use grpc::sdk_grpc;
use types::*;

/// SDK is an instance of the Agones SDK
pub struct Sdk {
    client: Arc<sdk_grpc::SdkClient>,
    health: Arc<Mutex<Option<grpcio::ClientCStreamSender<sdk::Empty>>>>,
    alpha: Arc<Alpha>,
}

impl Sdk {
    /// Starts a new SDK instance, and connects to localhost on port 9357.
    /// Blocks until connection and handshake are made.
    /// Times out after ~30 seconds.
    pub fn new() -> Result<Sdk> {
        let port = env::var("AGONES_SDK_GRPC_PORT").unwrap_or("9357".to_string());
        let addr = format!("localhost:{}", port);
        let env = Arc::new(grpcio::EnvBuilder::new().build());
        let ch = grpcio::ChannelBuilder::new(env)
            .keepalive_timeout(Duration::new(30, 0))
            .connect(&addr);
        let cli = sdk_grpc::SdkClient::new(ch.clone());
        let alpha = Alpha::new(ch);
        let req = sdk::Empty::new();

        // Unfortunately there isn't a native way to block until connected
        // so we had to roll our own.
        let mut counter = 0;
        loop {
            counter += 1;
            match cli.get_game_server(&req) {
                Ok(_) => break,
                Err(e) => {
                    if counter > 30 {
                        return Err(ErrorKind::Grpc(e).into());
                    }
                    sleep(Duration::from_secs(1));
                    continue;
                }
            }
        }

        let (sender, _) = cli.health()?;
        Ok(Sdk {
            client: Arc::new(cli),
            health: Arc::new(Mutex::new(Some(sender))),
            alpha: Arc::new(alpha),
        })
    }

    /// Alpha returns the Alpha SDK
    pub fn alpha(&self) -> &Arc<Alpha> {
        &self.alpha
    }

    /// Marks the Game Server as ready to receive connections
    pub fn ready(&self) -> Result<()> {
        let req = sdk::Empty::default_instance();
        let res = self.client.ready(req).map(|_| ())?;
        Ok(res)
    }

    /// Allocate the Game Server
    pub fn allocate(&self) -> Result<()> {
        let req = sdk::Empty::default_instance();
        let res = self.client.allocate(req).map(|_| ())?;
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
            return (
                self,
                Err(ErrorKind::HealthPingConnectionFailure(
                    "failed to hold client stream for health ping".to_string(),
                )
                .into()),
            );
        }
        let h: grpcio::ClientCStreamSender<sdk::Empty> = h.unwrap().into();

        let req = sdk::Empty::new();
        match h.send((req, grpcio::WriteFlags::default())).wait() {
            Ok(h) => {
                self.health = Arc::new(Mutex::new(Some(h)));
                (self, Ok(()))
            }
            Err(e) => (self, Err(ErrorKind::Grpc(e).into())),
        }
    }

    /// Set a Label value on the backing GameServer record that is stored in Kubernetes
    pub fn set_label<S>(&self, key: S, value: S) -> Result<()>
    where
        S: Into<String>,
    {
        let mut kv = sdk::KeyValue::new();
        kv.set_key(key.into());
        kv.set_value(value.into());
        let res = self.client.set_label(&kv).map(|_| ())?;
        Ok(res)
    }

    /// Set a Annotation value on the backing Gameserver record that is stored in Kubernetes
    pub fn set_annotation<S>(&self, key: S, value: S) -> Result<()>
    where
        S: Into<String>,
    {
        let mut kv = sdk::KeyValue::new();
        kv.set_key(key.into());
        kv.set_value(value.into());
        let res = self.client.set_annotation(&kv).map(|_| ())?;
        Ok(res)
    }

    /// Returns most of the backing GameServer configuration and Status
    pub fn get_gameserver(&self) -> Result<GameServer> {
        let req = sdk::Empty::new();
        let res = self
            .client
            .get_game_server(&req)
            .map(|e| GameServer::from_message(e))?;
        Ok(res)
    }

    /// Reserve marks the Game Server as Reserved for a given duration, at which point
    /// it will return the GameServer to a Ready state.
    /// Do note, the smallest unit available in the time.Duration argument is a second.
    pub fn reserve(&self, duration: Duration) -> Result<()> {
        let mut d = sdk::Duration::new();
        d.set_seconds(duration.as_secs() as i64);

        let res = self.client.reserve(&d).map(|_| ())?;
        Ok(res)
    }

    /// Watch the backing GameServer configuration on updated
    pub fn watch_gameserver<F>(&self, mut watcher: F) -> Result<()>
    where
        F: FnMut(GameServer) -> (),
    {
        let req = sdk::Empty::new();
        let mut receiver = self
            .client
            .watch_game_server(&req)?
            .map(|e| GameServer::from_message(e));
        loop {
            match receiver.into_future().wait() {
                Ok((Some(game_server), r)) => {
                    watcher(game_server);
                    receiver = r;
                }
                Ok((None, _)) => break,
                Err((e, _)) => return Err(e.into()),
            }
        }
        Ok(())
    }
}

impl Clone for Sdk {
    fn clone(&self) -> Self {
        Self {
            client: Arc::clone(&self.client),
            health: self.health.clone(),
            alpha: Arc::clone(&self.alpha),
        }
    }
}
