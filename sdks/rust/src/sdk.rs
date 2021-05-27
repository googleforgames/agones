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

use std::{env, time::Duration};
use tonic::transport::Channel;

mod api {
    tonic::include_proto!("agones.dev.sdk");
}

use api::sdk_client::SdkClient;
pub use api::GameServer;

pub type WatchStream = tonic::Streaming<GameServer>;

use crate::{
    alpha::Alpha,
    errors::{Error, Result},
};

#[inline]
fn empty() -> api::Empty {
    api::Empty {}
}

/// SDK is an instance of the Agones SDK
#[derive(Clone)]
pub struct Sdk {
    client: SdkClient<Channel>,
    health: tokio::sync::mpsc::Sender<api::Empty>,
    alpha: Alpha,
}

impl Sdk {
    /// Starts a new SDK instance, and connects to localhost on the port specified
    /// or else falls back to the `AGONES_SDK_GRPC_PORT` environment variable,
    /// or defaults to 9357.
    pub async fn new(port: Option<u16>, keep_alive: Option<Duration>) -> Result<Self> {
        // TODO: Add TLS? For some reason the original Cargo.toml was enabling
        // grpcio's openssl features, but AFAICT the SDK and sidecar only ever
        // communicate via a non-TLS connection, so seems like we could just
        // use the simpler client setup code if TLS is absolutely never needed
        let addr: http::Uri = format!(
            "http://localhost:{}",
            port.unwrap_or_else(|| {
                env::var("AGONES_SDK_GRPC_PORT")
                    .ok()
                    .and_then(|s| s.parse().ok())
                    .unwrap_or(9357)
            })
        )
        .parse()?;

        let builder = tonic::transport::channel::Channel::builder(addr)
            .keep_alive_timeout(keep_alive.unwrap_or_else(|| Duration::from_secs(30)));

        // let mut root_store = rustls::RootCertStore::empty();
        // root_store.add_server_trust_anchors(&webpki_roots::TLS_SERVER_ROOTS);
        // let mut rusttls_config = rustls::ClientConfig::new();
        // rusttls_config.alpn_protocols = vec![b"h2".to_vec(), b"http/1.1".to_vec()];
        // rusttls_config.root_store = root_store;
        // let tls_config =
        //     tonic::transport::ClientTlsConfig::new().rustls_client_config(rusttls_config);
        // builder = builder.tls_config(tls_config)?;

        let channel = builder.connect().await?;
        let mut client = SdkClient::new(channel.clone());
        let alpha = Alpha::new(channel);

        // Loop until we connect. The original implementation just looped once
        // every second up to a maximum of 30 seconds, but it's better for the
        // external caller to wrap this in their own timeout, eg
        // https://docs.rs/tokio/1.6.0/tokio/time/fn.timeout.html
        let mut connect_interval = tokio::time::interval(Duration::from_millis(100));

        loop {
            connect_interval.tick().await;

            match client.get_game_server(empty()).await {
                Err(e) => {
                    println!("unable to retrieve game server: {:#?}", e);
                }
                Ok(_) => break,
            }
        }

        // Keep both sender and receiver as RPC is canceled when sender or receiver is dropped
        let (sender, mut receiver) = tokio::sync::mpsc::channel(10);

        let health_stream = async_stream::stream! {
            while let Some(item) = receiver.recv().await {
                yield item;
            }
        };

        client.health(health_stream).await?;

        Ok(Self {
            client,
            health: sender,
            alpha,
        })
    }

    /// Alpha returns the Alpha SDK
    #[inline]
    pub fn alpha(&self) -> &Alpha {
        &self.alpha
    }

    /// Marks the Game Server as ready to receive connections
    #[inline]
    pub async fn ready(&mut self) -> Result<()> {
        Ok(self.client.ready(empty()).await.map(|_| ())?)
    }

    /// Allocate the Game Server
    #[inline]
    pub async fn allocate(&mut self) -> Result<()> {
        Ok(self.client.allocate(empty()).await.map(|_| ())?)
    }

    /// Marks the Game Server as ready to shutdown
    #[inline]
    pub async fn shutdown(&mut self) -> Result<()> {
        Ok(self.client.shutdown(empty()).await.map(|_| ())?)
    }

    /// Sends a ping to the health check to indicate that this server is healthy
    #[inline]
    pub async fn health(&self) -> Result<()> {
        self.health.send(empty()).await.map(|_| ()).map_err(|_| {
            Error::HealthPingConnectionFailure("tonic receiver was dropped".to_string())
        })
    }

    /// Set a Label value on the backing GameServer record that is stored in Kubernetes
    #[inline]
    pub async fn set_label(
        &mut self,
        key: impl Into<String>,
        value: impl Into<String>,
    ) -> Result<()> {
        Ok(self
            .client
            .set_label(api::KeyValue {
                key: key.into(),
                value: value.into(),
            })
            .await
            .map(|_| ())?)
    }

    /// Set a Annotation value on the backing Gameserver record that is stored in Kubernetes
    #[inline]
    pub async fn set_annotation(
        &mut self,
        key: impl Into<String>,
        value: impl Into<String>,
    ) -> Result<()> {
        Ok(self
            .client
            .set_annotation(api::KeyValue {
                key: key.into(),
                value: value.into(),
            })
            .await
            .map(|_| ())?)
    }

    /// Returns most of the backing GameServer configuration and Status
    #[inline]
    pub async fn get_gameserver(&mut self) -> Result<GameServer> {
        Ok(self
            .client
            .get_game_server(empty())
            .await
            .map(|res| res.into_inner())?)
    }

    /// Reserve marks the Game Server as Reserved for a given duration, at which point
    /// it will return the GameServer to a Ready state.
    /// Do note, the smallest unit available in the time.Duration argument is one second.
    #[inline]
    pub async fn reserve(&mut self, duration: Duration) -> Result<()> {
        Ok(self
            .client
            .reserve(api::Duration {
                seconds: std::cmp::max(duration.as_secs() as i64, 1),
            })
            .await
            .map(|_| ())?)
    }

    /// Watch the backing GameServer configuration on updated
    #[inline]
    pub async fn watch_gameserver<F>(&mut self) -> Result<WatchStream> {
        Ok(self
            .client
            .watch_game_server(empty())
            .await
            .map(|stream| stream.into_inner())?)
    }
}
