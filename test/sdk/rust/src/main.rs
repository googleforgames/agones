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

use std::{env, thread, time::Duration};

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
    let env_run_async = env::var("RUN_ASYNC").unwrap_or_default();
    if env_run_async.contains("true") {
        println!("RUN_ASYNC is set to true, so run test for async functions");
        run_async().await
    } else {
        tokio::task::block_in_place(run_sync)
    }
}

fn run_sync() -> Result<(), String> {
    use tokio::runtime::Handle;

    println!("Creating SDK instance");
    let mut sdk = Handle::current().block_on(async move {
        match tokio::time::timeout(
            Duration::from_secs(30),
            agones::Sdk::new(None /* default port */, None /* keep_alive */),
        )
        .await
        {
            Ok(sdk) => sdk.map_err(|e| format!("unable to create sdk client: {}", e)),
            Err(_) => Err("timed out attempting to connect to the sidecar".to_owned()),
        }
    })?;

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

    // Waiting for a thread to spawn
    thread::sleep(Duration::from_secs(2));

    println!("Marking server as ready...");
    Handle::current().block_on(async {
        sdk.ready()
            .await
            .map_err(|e| format!("Could not run Ready(): {}. Exiting!", e))
    })?;

    println!("...marked Ready");

    println!("Reserving for 5 seconds");
    Handle::current().block_on(async {
        sdk.reserve(Duration::from_secs(5))
            .await
            .map_err(|e| format!("Could not run Reserve(): {}. Exiting!", e))
    })?;
    println!("...Reserved");

    println!("Allocate game server ...");
    Handle::current().block_on(async {
        sdk.allocate()
            .await
            .map_err(|e| format!("Could not run Allocate(): {}. Exiting!", e))
    })?;

    println!("...marked Allocated");

    println!("Getting GameServer details...");
    let gameserver = Handle::current().block_on(async {
        sdk.get_gameserver()
            .await
            .map_err(|e| format!("Could not run GameServer(): {}. Exiting!", e))
    })?;

    println!(
        "GameServer name: {}",
        gameserver.object_meta.clone().unwrap().name
    );

    println!("Setting a label");
    let creation_ts = gameserver.object_meta.unwrap().creation_timestamp;
    Handle::current().block_on(async {
        sdk.set_label("test-label", &creation_ts.to_string())
            .await
            .map_err(|e| format!("Could not run SetLabel(): {}. Exiting!", e))
    })?;

    let feature_gates = env::var("FEATURE_GATES").unwrap_or_default();
    if feature_gates.contains("PlayerTracking=true") {
        run_player_tracking_features(sdk.alpha().clone())?;
    }

    for i in 0..1 {
        let time = i * 5;
        println!("Running for {} seconds", time);

        thread::sleep(Duration::from_secs(5));
    }

    println!("Shutting down...");
    Handle::current().block_on(async {
        sdk.shutdown()
            .await
            .map_err(|e| format!("Could not run Shutdown: {}. Exiting!", e))
    })?;
    println!("...marked for Shutdown");
    Ok(())
}

fn run_player_tracking_features(mut alpha: agones::alpha::Alpha) -> Result<(), String> {
    use tokio::runtime::Handle;

    println!("Setting player capacity...");
    Handle::current().block_on(async {
        alpha
            .set_player_capacity(10)
            .await
            .map_err(|e| format!("Could not run SetPlayerCapacity(): {:#?}. Exiting!", e))
    })?;

    println!("Getting player capacity...");
    let capacity = Handle::current().block_on(async {
        alpha
            .get_player_capacity()
            .await
            .map_err(|e| format!("Could not run GetPlayerCapacity(): {}. Exiting!", e))
    })?;
    println!("Player capacity: {}", capacity);

    println!("Increasing the player count...");
    let player_id = "1234".to_string();

    let added = Handle::current().block_on(async {
        alpha
            .player_connect(&player_id)
            .await
            .map_err(|e| format!("Could not run PlayerConnect(): {}. Exiting!", e))
    })?;
    if added {
        println!("Added player");
    } else {
        panic!("Failed to add player. Exiting!");
    }

    let connected = Handle::current().block_on(async {
        alpha
            .is_player_connected(&player_id)
            .await
            .map_err(|e| format!("Could not run IsPlayerConnected(): {}. Exiting!", e))
    })?;
    if connected {
        println!("{} is connected", player_id);
    } else {
        panic!("{} is not connected. Exiting!", player_id);
    }

    let player_ids = Handle::current().block_on(async {
        alpha
            .get_connected_players()
            .await
            .map_err(|e| format!("Could not run GetConnectedPlayers(): {}. Exiting!", e))
    })?;
    println!("Connected players: {:?}", player_ids);

    let player_count = Handle::current().block_on(async {
        alpha
            .get_player_count()
            .await
            .map_err(|e| format!("Could not run GetConnectedPlayers(): {}. Exiting!", e))
    })?;
    println!("Current player count: {}", player_count);

    println!("Decreasing the player count...");
    let removed = Handle::current().block_on(async {
        alpha
            .player_disconnect(&player_id)
            .await
            .map_err(|e| format!("Could not run PlayerDisconnect(): {}. Exiting!", e))
    })?;
    if removed {
        println!("Removed player");
    } else {
        panic!("Failed to remove player. Exiting!");
    }

    let player_count = Handle::current().block_on(async {
        alpha
            .get_player_count()
            .await
            .map_err(|e| format!("Could not GetPlayerCount(): {}. Exiting!", e))
    })?;
    println!("Current player count: {}", player_count);

    Ok(())
}

async fn run_async() -> Result<(), String> {
    let mut sdk = match tokio::time::timeout(
        Duration::from_secs(30),
        agones::Sdk::new(None /* default port */, None /* keep_alive */),
    )
    .await
    {
        Ok(sdk) => sdk.map_err(|e| format!("unable to create sdk client: {}", e))?,
        Err(_) => return Err("timed out attempting to connect to the sidecar".to_owned()),
    };

    let _health = sdk.spawn_health_task(Duration::from_secs(2));

    let _watch = {
        let mut watch_client = sdk.clone();
        let (tx, mut rx) = tokio::sync::oneshot::channel::<()>();

        tokio::task::spawn(async move {
            println!("Starting to watch GameServer updates...");
            let mut once = true;
            match watch_client.watch_gameserver().await {
                Err(e) => println!("Failed to watch for GameServer updates: {}", e),
                Ok(mut stream) => loop {
                    tokio::select! {
                        gs = stream.message() => {
                            match gs {
                                Ok(Some(gs)) => {
                                    let om = gs.object_meta.unwrap();
                                    println!("GameServer Update, name: {}", om.name);
                                    println!("GameServer Update, state: {}", gs.status.unwrap().state);

                                    if once {
                                        println!("Setting an annotation");
                                        let uid = om.uid.clone();

                                        if let Err(e) = watch_client.set_annotation("test-annotation", uid).await {
                                            eprintln!("Failed to set annotation from watch task: {}", e);
                                        }

                                        once = false;
                                    }
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

    tokio::time::sleep(Duration::from_secs(2)).await;

    println!("Marking server as ready...");
    sdk.ready()
        .await
        .map_err(|e| format!("Could not run Ready(): {}. Exiting!", e))?;
    println!("...marked Ready");

    println!("Reserving for 5 seconds");
    sdk.reserve(Duration::new(5, 0))
        .await
        .map_err(|e| format!("Could not run Reserve(): {}. Exiting!", e))?;
    println!("...Reserved");

    println!("Allocate game server ...");
    sdk.allocate()
        .await
        .map_err(|e| format!("Could not run Allocate(): {}. Exiting!", e))?;

    println!("...marked Allocated");

    println!("Getting GameServer details...");
    let gameserver = sdk
        .get_gameserver()
        .await
        .map_err(|e| format!("Could not run GameServer(): {}. Exiting!", e))?;

    println!(
        "GameServer name: {}",
        gameserver.object_meta.clone().unwrap().name
    );

    println!("Setting a label");
    let creation_ts = gameserver.object_meta.clone().unwrap().creation_timestamp;
    sdk.set_label("test-label", &creation_ts.to_string())
        .await
        .map_err(|e| format!("Could not run SetLabel(): {}. Exiting!", e))?;

    let feature_gates = env::var("FEATURE_GATES").unwrap_or_default();
    if feature_gates.contains("PlayerTracking=true") {
        run_player_tracking_features_async(sdk.alpha().clone()).await?;
    }

    for i in 0..1 {
        let time = i * 5;
        println!("Running for {} seconds", time);

        tokio::time::sleep(Duration::from_secs(5)).await;
    }

    println!("Shutting down...");
    sdk.shutdown()
        .await
        .map_err(|e| format!("Could not run Shutdown: {}. Exiting!", e))?;
    println!("...marked for Shutdown");
    Ok(())
}

async fn run_player_tracking_features_async(mut alpha: agones::alpha::Alpha) -> Result<(), String> {
    println!("Setting player capacity...");
    alpha
        .set_player_capacity(10)
        .await
        .map_err(|e| format!("Could not run SetPlayerCapacity(): {}. Exiting!", e))?;

    println!("Getting player capacity...");
    let capacity = alpha
        .get_player_capacity()
        .await
        .map_err(|e| format!("Could not run GetPlayerCapacity(): {}. Exiting!", e))?;
    println!("Player capacity: {}", capacity);

    println!("Increasing the player count...");
    let player_id = "1234".to_string();
    let added = alpha
        .player_connect(&player_id)
        .await
        .map_err(|e| format!("Could not run PlayerConnect(): {}. Exiting!", e))?;
    if added {
        println!("Added player");
    } else {
        panic!("Failed to add player. Exiting!");
    }

    let connected = alpha
        .is_player_connected(&player_id)
        .await
        .map_err(|e| format!("Could not run IsPlayerConnected(): {}. Exiting!", e))?;
    if connected {
        println!("{} is connected", player_id);
    } else {
        panic!("{} is not connected. Exiting!", player_id);
    }

    let player_ids = alpha
        .get_connected_players()
        .await
        .map_err(|e| format!("Could not run GetConnectedPlayers(): {}. Exiting!", e))?;
    println!("Connected players: {:?}", player_ids);

    let player_count = alpha
        .get_player_count()
        .await
        .map_err(|e| format!("Could not run GetConnectedPlayers(): {}. Exiting!", e))?;
    println!("Current player count: {}", player_count);

    println!("Decreasing the player count...");
    let removed = alpha
        .player_disconnect(&player_id)
        .await
        .map_err(|e| format!("Could not run PlayerDisconnect(): {}. Exiting!", e))?;
    if removed {
        println!("Removed player");
    } else {
        panic!("Failed to remove player. Exiting!");
    }

    let player_count = alpha
        .get_player_count()
        .await
        .map_err(|e| format!("Could not GetPlayerCount(): {}. Exiting!", e))?;
    println!("Current player count: {}", player_count);

    Ok(())
}
