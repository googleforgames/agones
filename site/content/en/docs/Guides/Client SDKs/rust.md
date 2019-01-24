---
title: "Rust Game Server Client SDK"
linkTitle: "Rust"
date: 2019-01-02T10:17:57Z
weight: 50
description: "This is the Rust version of the Agones Game Server Client SDK."
---

Check the [Client SDK Documentation]({{< relref "_index.md" >}}) for more details on each of the SDK functions and how to run the SDK locally.

{{< alert color="warning">}}
The Rust SDK has not been actively maintained, and doesn't have all the SDK functionality, although it _should_ still work.
  Pull Requests and updates are welcome.
{{< /alert >}}

## Download

Download the source {{< ghlink href="sdks/rust" >}}directly from Github{{< /ghlink >}}.

## Prerequisites

- CMake >= 3.8.0
- Rust >= 1.19.0
- Go (>=1.7)

The SDK needs the above for building to [gRPC-rs](https://github.com/pingcap/grpc-rs).

## Usage

Add this crate to `dependencies` section in your Cargo.toml.
Specify a directory where this README.md is located to the `path`.

```toml
[dependencies]
agones = { path = "../agones/sdks/rust" }
```

Add `extern crate agones` to your crate root.

To begin working with the SDK, create an instance of it. This function blocks until connection and handshake are made.

```rust
let sdk = agones::Sdk::new()?;
```

To send a [health check]({{< relref "_index.md#health" >}}) ping call `sdk.health()`.

```rust
if sdk.health().is_ok() {
    println!("Health ping sent");
}
```

To mark that the [game session is completed]({{< relref "_index.md#shutdown" >}}) and the game server should be shut down call `sdk.shutdown()`. 

```rust
if sdk.shutdown().is_err() {
    println!("Could not run Shutdown");
}
```
