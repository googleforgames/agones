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
        },
        Err(msg) => {
            println!("{}", msg);
            1
        }
    });
}

fn run() -> Result<(), String>{

    println!("Creating SDK instance");
    let sdk = agones::Sdk::new().map_err(|_| "Could not connect to the sidecar. Exiting!")?;

    let _t = thread::spawn(enclose!{(sdk) move || {
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

    println!("Marking server as ready...");

    for i in 0..10 {
        let time = i * 10;
        println!("Running for {} seconds", time);

        thread::sleep(Duration::from_secs(10));

        if i == 5 {
            println!("Shutting down after 60 seconds...");
            sdk.shutdown().map_err(|e| format!("Could not run Shutdown: {}. Exiting!", e))?;
            println!("...marked for Shutdown");
        }
    }

    Ok(())
}
