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

use std::result::Result;
use std::thread;
use std::time::Duration;

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

    let _watch = thread::spawn(enclose! {(sdk) move || {
        println!("Starting to watch GameServer updates...");
        let _ = sdk.watch_gameserver(|gameserver| {
            println!("GameServer Update, name: {}", gameserver.object_meta.unwrap().name);
            println!("GameServer Update, state: {}", gameserver.status.unwrap().state);
        });
    }});

    println!("Setting a label");
    sdk.set_label("test-label", "test-value")
        .map_err(|e| format!("Could not run SetLabel(): {}. Exiting!", e))?;

    println!("Setting an annotation");
    sdk.set_annotation("test-annotation", "test value")
        .map_err(|e| format!("Could not run SetAnnotation(): {}. Exiting!", e))?;

    println!("Marking server as ready...");
    sdk.ready()
        .map_err(|e| format!("Could not run Ready(): {}. Exiting!", e))?;

    println!("...marked Ready");

    println!("Setting as Reserved for 5 seconds");
    sdk.reserve(Duration::new(5, 0)).map_err(|e| format!("Could not run Reserve(): {}. Exiting!", e))?;
    println!("...Reserved");

    thread::sleep(Duration::new(6, 0));

    println!("Getting GameServer details...");
    let gameserver = sdk
        .get_gameserver()
        .map_err(|e| format!("Could not run GameServer(): {}. Exiting!", e))?;

    println!("GameServer name: {}", gameserver.object_meta.unwrap().name);

    for i in 0..10 {
        let time = i * 10;
        println!("Running for {} seconds", time);

        thread::sleep(Duration::from_secs(10));

        if i == 5 {
            println!("Shutting down after 60 seconds...");
            sdk.shutdown()
                .map_err(|e| format!("Could not run Shutdown: {}. Exiting!", e))?;
            println!("...marked for Shutdown");
        }
    }

    Ok(())
}
