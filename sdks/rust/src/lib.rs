//! the Rust game server SDK
#[macro_use]
extern crate error_chain;
extern crate grpcio;
extern crate grpcio_proto;
extern crate protobuf;
extern crate futures;

mod grpc;
mod sdk;
pub mod errors;

pub use sdk::Sdk;
