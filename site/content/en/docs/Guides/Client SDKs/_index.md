---
title: "Agones Game Server Client SDKs" 
linkTitle: "Client SDKs"
date: 2019-01-02T10:16:30Z
weight: 10
description: "The SDKs are integration points for game servers with Agones itself."
---

## Overview

The client SDKs are required for a game server to work with Agones.

The current supported SDKs are:

- [Unreal Engine]({{< relref "unreal.md" >}})
- [Unity]({{< relref "unity.md" >}})
- [C++]({{< relref "cpp.md" >}})
- [Node.js]({{< relref "nodejs.md" >}})
- [Go]({{< relref "go.md" >}})
- [Rust]({{< relref "rust.md" >}})
- [REST]({{< relref "rest.md" >}})

The SDKs are relatively thin wrappers around [gRPC](https://grpc.io) generated clients,
or an implementation of the REST API (exposed via [grpc-gateway](https://github.com/grpc-ecosystem/grpc-gateway)), 
where gRPC client generation and compilation isn't well supported.

They connect to a small process that Agones coordinates to run alongside the Game Server
in a Kubernetes [`Pod`](https://kubernetes.io/docs/concepts/workloads/pods/pod-overview/).
This means that more languages can be supported in the future with minimal effort
(but pull requests are welcome! 😊 ).

There is also [local development tooling]({{< relref "local.md" >}}) for working against the SDK locally,
without having to spin up an entire Kubernetes infrastructure.

## Function Reference

While each of the SDKs are canonical to their languages, they all have the following
functions that implement the core responsibilities of the SDK.

For language specific documentation, have a look at the respective source (linked above), 
and the {{< ghlink href="examples" >}}examples{{< /ghlink >}}.

### Ready()
This tells Agones that the Game Server is ready to take player connections.
Once a Game Server has specified that it is `Ready`, then the Kubernetes
GameServer record will be moved to the `Ready` state, and the details
for its public address and connection port will be populated.

While Agones prefers that `Shutdown()` is run once a game has completed to delete the `GameServer` instance,
if you want or need to move an `Allocated` `GameServer` back to `Ready` to be reused, you can call this SDK method again to do
this.

### Health()
This sends a single ping to designate that the Game Server is alive and
healthy. Failure to send pings within the configured thresholds will result
in the GameServer being marked as `Unhealthy`. 

See the {{< ghlink href="examples/gameserver.yaml" >}}gameserver.yaml{{< /ghlink >}} for all health checking
configurations.

### Shutdown()
This tells Agones to shut down the currently running game server.
The GameServer state will be set `Shutdown` and the 
backing Pod will be deleted, if they have not shut themselves down already. 

### SetLabel(key, value)

This will set a [Label](https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/) value on the backing `GameServer`
record that is stored in Kubernetes. To maintain isolation, the `key` value is automatically prefixed with "stable.agones.dev/sdk-"

> Note: There are limits on the characters that be used for label keys and values. Details are [here](https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/#syntax-and-character-set).

This can be useful if you want to information from your running game server process to be observable or searchable through the Kubernetes API.  

### SetAnnotation(key, value)

This will set a [Annotation](https://kubernetes.io/docs/concepts/overview/working-with-objects/annotations/) value on the backing
`Gameserver` record that is stored in Kubernetes. To maintain isolation, the `key` value is automatically prefixed with "stable.agones.dev/sdk-"

This can be useful if you want to information from your running game server process to be observable through the Kubernetes API.

### GameServer()

This returns most of the backing GameServer configuration and Status. This can be useful
for instances where you may want to know Health check configuration, or the IP and Port
the GameServer is currently allocated to.

Since the GameServer contains an entire [PodTemplate](https://kubernetes.io/docs/concepts/workloads/pods/pod-overview/#pod-templates)
the returned object is limited to that configuration that was deemed useful. If there are
areas that you feel are missing, please [file an issue](https://github.com/googleforgames/agones/issues) or pull request.

The easiest way to see what is exposed, is to check the  {{< ghlink href="sdk.proto" >}}`sdk.proto`{{< /ghlink >}},
specifically at the `message GameServer`.

For language specific documentation, have a look at the respective source (linked above), 
and the {{< ghlink href="examples" >}}examples{{< /ghlink >}}.

### WatchGameServer(function(gameserver){...})

This executes the passed in callback with the current `GameServer` details whenever the underlying `GameServer` configuration is updated.
This can be useful to track `GameServer > Status > State` changes, `metadata` changes, such as labels and annotations, and more.

In combination with this SDK, manipulating [Annotations](https://kubernetes.io/docs/concepts/overview/working-with-objects/annotations/) and
[Labels](https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/) can also be a useful way to communicate information through to running game server processes from outside those processes.
This is especially useful when combined with `GameServerAllocation` [applied metadata]({{< ref "/docs/Reference/gameserverallocation.md" >}}).

Since the GameServer contains an entire [PodTemplate](https://kubernetes.io/docs/concepts/workloads/pods/pod-overview/#pod-templates)
the returned object is limited to that configuration that was deemed useful. If there are
areas that you feel are missing, please [file an issue](https://github.com/googleforgames/agones/issues) or pull request.

The easiest way to see what is exposed, is to check the {{< ghlink href="sdk.proto" >}}`sdk.proto`{{< /ghlink >}},
specifically at the `message GameServer`.

For language specific documentation, have a look at the respective source (linked above), 
and the {{< ghlink href="examples" >}}examples{{< /ghlink >}}.

### Allocate()

With some matchmakers and game matching strategies, it can be important for game servers to mark themselves as `Allocated`.
For those scenarios, this SDK functionality exists. 

> Note: Using a [GameServerAllocation]({{< ref "/docs/Reference/gameserverallocation.md" >}}) is preferred in all other scenarios, 
as it gives Agones control over how packed `GameServers` are scheduled within a cluster, whereas with `Allocate()` you
relinquish control to an external service which likely doesn't have as much information as Agones.

{{% feature publishVersion="0.12.0" %}}
### Reserve(seconds)

With some matchmaking scenarios and systems it is important to be able to ensure that a `GameServer` is unable to be deleted,
but doesn't trigger a FleetAutoscaler scale up. This is where `Reserve(seconds)` is useful.

`Reserve(seconds)` will move the `GameServer` into the Reserved state for the specified number of seconds (0 is forever), and then it will be
moved back to `Ready` state. While in `Reserved` state, the `GameServer` will not be deleted on scale down or `Fleet` update,
and also will not be Allocated.

This is often used when a game server process must register itself with an external system, such as a matchmaker,
that requires it to designate itself as available for a game session for a certain period. Once a game session has started,
it should call `SDK.Allocate()` to designate that players are currently active on it.

Calling other state changing SDK commands such as `Ready` or `Allocate` will turn off the timer to reset the `GameServer` back
to the `Ready` state.

{{% /feature %}}

## Writing your own SDK

If there isn't an SDK for the language and platform you are looking for, you have several options:

### gRPC Client Generation

If client generation is well supported by [gRPC](https://grpc.io/docs/), then generate a client from the
{{< ghlink href="sdk.proto" >}}`sdk.proto`{{< /ghlink >}}, and look at the current {{< ghlink href="sdks" >}}sdks{{< /ghlink >}} to see how the wrappers are implemented to make interaction
with the SDK server simpler for the user.

### REST API Implementation

If client generation is not well supported by gRPC, or if there are other complicating factors, implement the SDK through
the [REST]({{< relref "rest.md" >}}) HTTP+JSON interface. This could be written by hand, or potentially generated from
the {{< ghlink href="sdk.swagger.json" >}}Swagger/OpenAPI Spec{{< /ghlink >}}.

Finally, if you build something that would be usable by the community, please submit a pull request!

## SDK Conformance Test

There is a tool `SDK server Conformance` checker which will run Local SDK server and record all requests your client is performing.

In order to check that SDK is working properly you should write simple SDK test client which would use all methods of your SDK.

Also to test that SDK cliet is receiving valid Gameserver data, your binary should set the same `Label` value as creation timestamp which you will receive as a result of GameServer() call and `Annotation` value same as gameserver UID received by Watch gameserver callback.

Complete list of endpoints which should be called by your test client is the following:
```
ready,allocate,setlabel,setannotation,gameserver,health,shutdown,watch
```

In order to run this test SDK server locally use:
```
SECONDS=30 make run-sdk-conformance-local
```

Docker container would timeout in 30 seconds and give your the comparison of received requests and expected requests

For instance you could run go sdk conformance test and see how the process goes: 
```
SDK_FOLDER=go make run-sdk-conformance-test
```

In order to add test client for your SDK, write `jstest.sh` and `Dockerfile`. Refer to {{< ghlink href="build/build-sdk-images/go/Dockerfile" >}}Golang SDK testing directory structure{{< /ghlink >}}.

## Building the Tools

If you wish to build the binaries from source
the `make` target `build-agones-sdk-binary` will compile the necessary binaries
for all supported operating systems (64 bit windows, linux and osx).

You can find the binaries in the `bin` folder in {{< ghlink href="cmd/sdk-server" >}}`cmd/sdk-server`{{< /ghlink >}}
once compilation is complete.

See {{< ghlink href="build" branch="master" >}}Developing, Testing and Building Agones{{< /ghlink >}} for more details.
