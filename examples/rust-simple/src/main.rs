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

use std::time::Duration;

#[tokio::main(flavor = "multi_thread", worker_threads = 4)]
async fn main() {
    println!("Rust Game Server has started!");

    ::std::process::exit(match run().await {
        Ok(_) => {
            println!("Rust Game Server finished.");
            0
        }
        Err(msg) => {
            println!("{}", msg);
            1
        }
    });
}

async fn run() -> Result<(), String> {
    println!("Creating SDK instance");
    let mut sdk = match tokio::time::timeout(
        Duration::from_secs(30),
        agones::Sdk::new(None /* default port */, None /* keep_alive */),
    )
    .await
    {
        Ok(sdk) => sdk.map_err(|e| format!("unable to create sdk client: {}", e))?,
        Err(_) => {
            return Err("timed out attempting to connect to the sidecar".to_owned());
        }
    };

    let _health = sdk.spawn_health_task(Duration::from_secs(2));

    let _watch = {
        let mut watch_client = sdk.clone();
        let (tx, mut rx) = tokio::sync::oneshot::channel::<()>();

        tokio::task::spawn(async move {
            println!("Starting to watch GameServer updates...");
            match watch_client.watch_gameserver().await {
                Err(e) => println!("Failed to watch for GameServer updates: {}", e),
                Ok(mut stream) => loop {
                    tokio::select! {
                        gs = stream.message() => {
                            match gs {
                                Ok(Some(gs)) => {
                                    println!("GameServer Update, name: {}", gs.object_meta.unwrap().name);
                                    println!("GameServer Update, state: {}", gs.status.unwrap().state);
                                }
                                Ok(None) => {
                                    println!("Server closed the GameServer watch stream");
                                    break;
                                }
                                Err(e) => {
                                    eprintln!("GameServer Update stream encountered an error: {}", e);
                                }
                            }

                        }
                        _ = &mut rx => {
                            println!("Shutting down GameServer watch loop");
                            break;
                        }
                    }
                },
            }
        });

        tx
    };

    println!("Setting a label");
    sdk.set_label("test-label", "test-value")
        .await
        .map_err(|e| format!("Could not run SetLabel(): {}. Exiting!", e))?;

    println!("Setting an annotation");
    sdk.set_annotation("test-annotation", "test value")
        .await
        .map_err(|e| format!("Could not run SetAnnotation(): {}. Exiting!", e))?;

    println!("Marking server as ready...");
    sdk.ready()
        .await
        .map_err(|e| format!("Could not run Ready(): {}. Exiting!", e))?;

    println!("...marked Ready");

    println!("Setting as Reserved for 5 seconds");
    sdk.reserve(Duration::from_secs(5))
        .await
        .map_err(|e| format!("Could not run Reserve(): {}. Exiting!", e))?;
    println!("...Reserved");

    tokio::time::sleep(Duration::from_secs(6)).await;

    println!("Getting GameServer details...");
    let gameserver = sdk
        .get_gameserver()
        .await
        .map_err(|e| format!("Could not run GameServer(): {}. Exiting!", e))?;

    println!("GameServer name: {}", gameserver.object_meta.unwrap().name);

    for i in 0..10 {
        let time = i * 10;
        println!("Running for {} seconds", time);

        tokio::time::sleep(Duration::from_secs(10)).await;

        if i == 5 {
            println!("Shutting down after 60 seconds...");
            sdk.shutdown()
                .await
                .map_err(|e| format!("Could not run Shutdown: {}. Exiting!", e))?;
            println!("...marked for Shutdown");
        }
    }

    Ok(())
}
