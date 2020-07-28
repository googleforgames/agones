---
title: "Rust Game Server Client SDK"
linkTitle: "Rust"
date: 2019-01-02T10:17:57Z
weight: 50
description: "This is the Rust version of the Agones Game Server Client SDK."
---

Check the [Client SDK Documentation]({{< relref "_index.md" >}}) for more details on each of the SDK functions and how to run the SDK locally.

## Download

Download the source {{< ghlink href="sdks/rust" >}}directly from GitHub{{< /ghlink >}}.

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

To mark the [game session as ready]({{< relref "_index.md#ready" >}}) call `sdk.ready()`.

```rust
sdk.ready()?;
```

To mark the game server as [reserved]({{< relref "_index.md#reserveseconds" >}}) for a period of time, call `sdk.reserve(duration)`.

```rust
sdk.reserve(Duration::new(5, 0))?;
```

To mark that the [game session is completed]({{< relref "_index.md#shutdown" >}}) and the game server should be shut down call `sdk.shutdown()`.

```rust
if sdk.shutdown().is_err() {
    println!("Could not run Shutdown");
}
```

To [set a Label]({{< relref "_index.md#setlabelkey-value" >}}) on the backing `GameServer` call `sdk.set_label(key, value)`.

```rust
sdk.set_label("test-label", "test-value")?;
```

To [set an Annotation]({{< relref "_index.md#setannotationkey-value" >}}) on the backing `GameServer` call `sdk.set_annotation(key, value)`.

```rust
sdk.set_annotation("test-annotation", "test value")?;
```

To get [details of the backing `GameServer`]({{< relref "_index.md#gameserver" >}}) call `sdk.get_gameserver()`.

The function will return an instance of `agones::types::GameServer` including `GameServer` configuration info.

```rust
let gameserver = sdk.get_gameserver()?;
```

To get [updates on the backing `GameServer`]({{< relref "_index.md#watchgameserverfunctiongameserver" >}}) as they happen, call `sdk.watch_gameserver(|gameserver| {...})`.

This will call the passed closure synchronously (this is a blocking function, so you may want to run it in its own thread) whenever the backing `GameServer` is updated.

```rust
sdk.watch_gameserver(|gameserver| {
    println!("GameServer Update, name: {}", gameserver.object_meta.unwrap().name);
    println!("GameServer Update, state: {}", gameserver.status.unwrap().state);
})?;
```

For more information, please read the [SDK Overview]({{< relref "_index.md" >}}), check out {{< ghlink href="sdks/rust/src/sdk.rs" >}}agones sdk implementation{{< /ghlink >}} and also look at the {{< ghlink href="examples/rust-simple" >}}Rust example{{< / >}}.
