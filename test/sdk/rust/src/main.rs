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

extern crate agones;

use std::env;
use std::result::Result;
use std::thread;
use std::time::Duration;

use async_std::task;

macro_rules! enclose {
    ( ($( $x:ident ),*) $y:expr ) => {
        {
            $(let mut $x = $x.clone();)*
            $y
        }
    };
}

fn main() {
    println!("Rust Game Server has started!");

    ::std::process::exit(match run() {
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

fn run() -> Result<(), String> {
    let env_run_async = env::var("RUN_ASYNC").unwrap_or("false".to_string());
    if env_run_async.contains("true") {
        println!("RUN_ASYNC is set to true, so run test for async functions");
        run_async()
    } else {
        run_sync()
    }
}

fn run_sync() -> Result<(), String> {
    println!("Creating SDK instance");
    let sdk = agones::Sdk::new().map_err(|_| "Could not connect to the sidecar. Exiting!")?;

    let _health = thread::spawn(enclose! {(sdk) move || {
        loop {
            match sdk.health() {
                (s, Ok(_)) => {
                    println!("Health ping sent");
                    sdk = s;
                },
                (s, Err(e)) => {
                    println!("Health ping failed : {:?}", e);
                    sdk = s;
                }
            }
            thread::sleep(Duration::from_secs(2));
        }
    }});

    #[allow(unused_mut)]
    let _watch = thread::spawn(enclose! {(sdk) move || {
        println!("Starting to watch GameServer updates...");
        let mut once = true;
        let _ = sdk.watch_gameserver(|gameserver| {
            println!("GameServer Update, name: {}", gameserver.object_meta.clone().unwrap().name);
            println!("GameServer Update, state: {}", gameserver.status.clone().unwrap().state);
            if once {
                println!("Setting an annotation");
                let uid = gameserver.object_meta.clone().unwrap().uid.clone();
                let _ = sdk.set_annotation("test-annotation", &uid.to_string());
                once = false;
            }
        });
    }});

    // Waiting for a thread to spawn
    thread::sleep(Duration::from_secs(2));

    println!("Marking server as ready...");
    sdk.ready()
        .map_err(|e| format!("Could not run Ready(): {}. Exiting!", e))?;

    println!("...marked Ready");

    println!("Reserving for 5 seconds");
    sdk.reserve(Duration::new(5, 0))
        .map_err(|e| format!("Could not run Reserve(): {}. Exiting!", e))?;
    println!("...Reserved");

    println!("Allocate game server ...");
    sdk.allocate()
        .map_err(|e| format!("Could not run Allocate(): {}. Exiting!", e))?;

    println!("...marked Allocated");

    println!("Getting GameServer details...");
    let gameserver = sdk
        .get_gameserver()
        .map_err(|e| format!("Could not run GameServer(): {}. Exiting!", e))?;

    println!(
        "GameServer name: {}",
        gameserver.object_meta.clone().unwrap().name
    );

    println!("Setting a label");
    let creation_ts = gameserver.object_meta.clone().unwrap().creation_timestamp;
    sdk.set_label("test-label", &creation_ts.to_string())
        .map_err(|e| format!("Could not run SetLabel(): {}. Exiting!", e))?;

    let feature_gates = env::var("FEATURE_GATES").unwrap_or("".to_string());
    if feature_gates.contains("PlayerTracking=true") {
        run_player_tracking_features(&sdk)?;
    }

    for i in 0..1 {
        let time = i * 5;
        println!("Running for {} seconds", time);

        thread::sleep(Duration::from_secs(5));
    }

    println!("Shutting down...");
    sdk.shutdown()
        .map_err(|e| format!("Could not run Shutdown: {}. Exiting!", e))?;
    println!("...marked for Shutdown");
    Ok(())
}

fn run_player_tracking_features(sdk: &agones::Sdk) -> Result<(), String> {
    println!("Setting player capacity...");
    sdk.alpha()
        .set_player_capacity(10)
        .map_err(|e| format!("Could not run SetPlayerCapacity(): {}. Exiting!", e))?;

    println!("Getting player capacity...");
    let capacity = sdk
        .alpha()
        .get_player_capacity()
        .map_err(|e| format!("Could not run GetPlayerCapacity(): {}. Exiting!", e))?;
    println!("Player capacity: {}", capacity);

    println!("Increasing the player count...");
    let player_id = "1234".to_string();
    let added = sdk
        .alpha()
        .player_connect(&player_id)
        .map_err(|e| format!("Could not run PlayerConnect(): {}. Exiting!", e))?;
    if added {
        println!("Added player");
    } else {
        panic!("Failed to add player. Exiting!");
    }

    let connected = sdk
        .alpha()
        .is_player_connected(&player_id)
        .map_err(|e| format!("Could not run IsPlayerConnected(): {}. Exiting!", e))?;
    if connected {
        println!("{} is connected", player_id);
    } else {
        panic!("{} is not connected. Exiting!", player_id);
    }

    let player_ids = sdk
        .alpha()
        .get_connected_players()
        .map_err(|e| format!("Could not run GetConnectedPlayers(): {}. Exiting!", e))?;
    println!("Connected players: {:?}", player_ids);

    let player_count = sdk
        .alpha()
        .get_player_count()
        .map_err(|e| format!("Could not run GetConnectedPlayers(): {}. Exiting!", e))?;
    println!("Current player count: {}", player_count);

    println!("Decreasing the player count...");
    let removed = sdk
        .alpha()
        .player_disconnect(&player_id)
        .map_err(|e| format!("Could not run PlayerDisconnect(): {}. Exiting!", e))?;
    if removed {
        println!("Removed player");
    } else {
        panic!("Failed to remove player. Exiting!");
    }

    let player_count = sdk
        .alpha()
        .get_player_count()
        .map_err(|e| format!("Could not GetPlayerCount(): {}. Exiting!", e))?;
    println!("Current player count: {}", player_count);

    Ok(())
}

fn run_async() -> Result<(), String> {
    let sdk = task::block_on(async { agones::Sdk::new_async().await })
        .map_err(|_| "Could not connect to the sidecar. Exiting!")?;

    task::spawn(enclose! {(sdk) async move {
        loop {
            match sdk.health_async().await {
                Ok(_) => {
                    println!("Health ping sent");
                }
                Err(e) => {
                    println!("Health ping failed : {:?}", e);
                }
            }
            task::sleep(Duration::from_secs(2)).await;
        }
    }});

    #[allow(unused_mut)]
    task::spawn(enclose! {(sdk) async move {
        println!("Starting to watch GameServer updates...");
        let mut once = true;
        let _ = sdk.watch_gameserver_async(|gameserver| {
            println!(
                "GameServer Update, name: {}",
                gameserver.object_meta.clone().unwrap().name
            );
            println!(
                "GameServer Update, state: {}",
                gameserver.status.clone().unwrap().state
            );
            if once {
                println!("Setting an annotation");
                let uid = gameserver.object_meta.clone().unwrap().uid.clone();
                #[allow(unused_mut)]
                task::spawn(enclose! {(sdk) async move {
                    let _ = sdk.set_annotation_async("test-annotation", &uid.to_string())
                        .await;
                }});
                once = false;
            }
        })
        .await;
    }});

    task::block_on(task::sleep(Duration::from_secs(2)));

    #[allow(unused_mut)]
    let handle: task::JoinHandle<Result<(), String>> = task::spawn(enclose! {(sdk) async move {
        println!("Marking server as ready...");
        sdk.ready_async()
            .await
            .map_err(|e| format!("Could not run Ready(): {}. Exiting!", e))?;

        println!("...marked Ready");

        println!("Reserving for 5 seconds");
        sdk.reserve_async(Duration::new(5, 0))
            .await
            .map_err(|e| format!("Could not run Reserve(): {}. Exiting!", e))?;
        println!("...Reserved");

        println!("Allocate game server ...");
        sdk.allocate_async()
            .await
            .map_err(|e| format!("Could not run Allocate(): {}. Exiting!", e))?;

        println!("...marked Allocated");

        println!("Getting GameServer details...");
        let gameserver = sdk
            .get_gameserver_async()
            .await
            .map_err(|e| format!("Could not run GameServer(): {}. Exiting!", e))?;

        println!(
            "GameServer name: {}",
            gameserver.object_meta.clone().unwrap().name
        );

        println!("Setting a label");
        let creation_ts = gameserver.object_meta.clone().unwrap().creation_timestamp;
        sdk.set_label_async("test-label", &creation_ts.to_string())
            .await
            .map_err(|e| format!("Could not run SetLabel(): {}. Exiting!", e))?;

        let feature_gates = env::var("FEATURE_GATES").unwrap_or("".to_string());
        if feature_gates.contains("PlayerTracking=true") {
            run_player_tracking_features_async(&sdk).await?;
        }

        for i in 0..1 {
            let time = i * 5;
            println!("Running for {} seconds", time);

            task::sleep(Duration::from_secs(5)).await;
        }

        println!("Shutting down...");
        sdk.shutdown_async()
            .await
            .map_err(|e| format!("Could not run Shutdown: {}. Exiting!", e))?;
        println!("...marked for Shutdown");
        Ok(())
    }});
    task::block_on(handle).map_err(|e| format!("{}", e))?;

    Ok(())
}

async fn run_player_tracking_features_async(sdk: &agones::Sdk) -> Result<(), String> {
    println!("Setting player capacity...");
    sdk.alpha()
        .set_player_capacity_async(10)
        .await
        .map_err(|e| format!("Could not run SetPlayerCapacity(): {}. Exiting!", e))?;

    println!("Getting player capacity...");
    let capacity = sdk
        .alpha()
        .get_player_capacity_async()
        .await
        .map_err(|e| format!("Could not run GetPlayerCapacity(): {}. Exiting!", e))?;
    println!("Player capacity: {}", capacity);

    println!("Increasing the player count...");
    let player_id = "1234".to_string();
    let added = sdk
        .alpha()
        .player_connect_async(&player_id)
        .await
        .map_err(|e| format!("Could not run PlayerConnect(): {}. Exiting!", e))?;
    if added {
        println!("Added player");
    } else {
        panic!("Failed to add player. Exiting!");
    }

    let connected = sdk
        .alpha()
        .is_player_connected_async(&player_id)
        .await
        .map_err(|e| format!("Could not run IsPlayerConnected(): {}. Exiting!", e))?;
    if connected {
        println!("{} is connected", player_id);
    } else {
        panic!("{} is not connected. Exiting!", player_id);
    }

    let player_ids = sdk
        .alpha()
        .get_connected_players_async()
        .await
        .map_err(|e| format!("Could not run GetConnectedPlayers(): {}. Exiting!", e))?;
    println!("Connected players: {:?}", player_ids);

    let player_count = sdk
        .alpha()
        .get_player_count_async()
        .await
        .map_err(|e| format!("Could not run GetConnectedPlayers(): {}. Exiting!", e))?;
    println!("Current player count: {}", player_count);

    println!("Decreasing the player count...");
    let removed = sdk
        .alpha()
        .player_disconnect_async(&player_id)
        .await
        .map_err(|e| format!("Could not run PlayerDisconnect(): {}. Exiting!", e))?;
    if removed {
        println!("Removed player");
    } else {
        panic!("Failed to remove player. Exiting!");
    }

    let player_count = sdk
        .alpha()
        .get_player_count_async()
        .await
        .map_err(|e| format!("Could not GetPlayerCount(): {}. Exiting!", e))?;
    println!("Current player count: {}", player_count);

    Ok(())
}
