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
- [C#]({{< relref "csharp.md" >}})
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

## Connecting to the SDK Server

Starting with Agones 1.1.0, the port that the SDK Server listens on for incoming gRPC or HTTP requests is
configurable. This provides flexibility in cases where the default port conflicts with a port that is needed
by the game server.

Agones will automatically set the following environment variables on all game server containers:

* `AGONES_SDK_GRPC_PORT`: The port where the gRPC server is listening (defaults to 9357)
* `AGONES_SDK_HTTP_PORT`: The port where the grpc-gateway is listening (defaults to 9358)

The SDKs will automatically discover and connect to the gRPC port specified in the environment variable.

If your game server requires using a REST client, it is advised to use the port from the environment variable,
otherwise your game server will not be able to contact the SDK server if it is configured to use a non-default port.

## Function Reference

While each of the SDKs are canonical to their languages, they all have the following
functions that implement the core responsibilities of the SDK.

For language specific documentation, have a look at the respective source (linked above), 
and the {{< ghlink href="examples" >}}examples{{< /ghlink >}}.

Calling any of state changing functions mentioned below does not guarantee that GameServer Custom Resource object would actually change its state right after the call. For instance, it could be moved to the `Shutdown` state elsewhere (for example, when a fleet scales down), which leads to no changes in `GameServer` object. You can verify the result of this call by waiting for the desired state in a callback to WatchGameServer() function.

Functions which changes GameServer state or settings are:

1. Ready()
1. Shutdown()
1. SetLabel()
1. SetAnnotation()
1. Allocate()
1. Reserve() 
1. Alpha().SetCapacity()
1. Alpha().PlayerConnect()
1. Alpha().PlayerDisconnect()

### Lifecycle Management

#### Ready()
This tells Agones that the Game Server is ready to take player connections.
Once a Game Server has specified that it is `Ready`, then the Kubernetes
GameServer record will be moved to the `Ready` state, and the details
for its public address and connection port will be populated.

While Agones prefers that `Shutdown()` is run once a game has completed to delete the `GameServer` instance,
if you want or need to move an `Allocated` `GameServer` back to `Ready` to be reused, you can call this SDK method again to do
this.

#### Health()
This sends a single ping to designate that the Game Server is alive and
healthy. Failure to send pings within the configured thresholds will result
in the GameServer being marked as `Unhealthy`. 

See the {{< ghlink href="examples/gameserver.yaml" >}}gameserver.yaml{{< /ghlink >}} for all health checking
configurations.

#### Reserve(seconds)

With some matchmaking scenarios and systems it is important to be able to ensure that a `GameServer` is unable to be deleted,
but doesn't trigger a FleetAutoscaler scale up. This is where `Reserve(seconds)` is useful.

`Reserve(seconds)` will move the `GameServer` into the Reserved state for the specified number of seconds (0 is forever), and then it will be
moved back to `Ready` state. While in `Reserved` state, the `GameServer` will not be deleted on scale down or `Fleet` update,
and also it could not be Allocated using [GameServerAllocation]({{< ref "/docs/Reference/gameserverallocation.md" >}}).

This is often used when a game server process must register itself with an external system, such as a matchmaker,
that requires it to designate itself as available for a game session for a certain period. Once a game session has started,
it should call `SDK.Allocate()` to designate that players are currently active on it.

Calling other state changing SDK commands such as `Ready` or `Allocate` will turn off the timer to reset the `GameServer` back
to the `Ready` state or to promote it to an `Allocated` state accordingly.

#### Allocate()

With some matchmakers and game matching strategies, it can be important for game servers to mark themselves as `Allocated`.
For those scenarios, this SDK functionality exists.

There is a chance that GameServer does not actually become `Allocated` after this call. Please refer to the general note in [Function Reference](#function-reference) above.

{{< alert title="Note" color="info">}}
Using a [GameServerAllocation]({{< ref "/docs/Reference/gameserverallocation.md" >}}) is preferred in all other scenarios, 
as it gives Agones control over how packed `GameServers` are scheduled within a cluster, whereas with `Allocate()` you
relinquish control to an external service which likely doesn't have as much information as Agones.
{{< /alert >}}

#### Shutdown()
This tells Agones to shut down the currently running game server. The GameServer state will be set `Shutdown` and the 
backing Pod will be Terminated.

It's worth reading the [Termination of Pods](https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle/#pod-termination)
Kubernetes documentation, to understand the termination process, and the related configuration options.

As a rule of thumb, implement a graceful shutdown in your game sever process when it receives the TERM signal
from Kubernetes when the backing Pod goes into Termination state.

Be aware that if you use a variation of `System.exit(0)` after calling SDK.Shutdown(), your game server container may
restart for a brief period, inline with our [Health Checking]({{% ref "/docs/Guides/health-checking.md#health-failure-strategy" %}}) policies. 

### Configuration Retrieval 

#### GameServer()

This returns most of the backing GameServer configuration and Status. This can be useful
for instances where you may want to know Health check configuration, or the IP and Port
the GameServer is currently allocated to.

Since the GameServer contains an entire [PodTemplate](https://kubernetes.io/docs/concepts/workloads/pods/pod-overview/#pod-templates)
the returned object is limited to that configuration that was deemed useful. If there are
areas that you feel are missing, please [file an issue](https://github.com/googleforgames/agones/issues) or pull request.

The easiest way to see what is exposed, is to check
the {{% ghlink href="proto/sdk/sdk.proto" %}}`sdk.proto`{{% /ghlink %}}, specifically at
the `message GameServer`.

For language specific documentation, have a look at the respective source (linked above), 
and the {{< ghlink href="examples" >}}examples{{< /ghlink >}}.

#### WatchGameServer(function(gameserver){...})

This executes the passed in callback with the current `GameServer` details whenever the underlying `GameServer` configuration is updated.
This can be useful to track `GameServer > Status > State` changes, `metadata` changes, such as labels and annotations, and more.

In combination with this SDK, manipulating [Annotations](https://kubernetes.io/docs/concepts/overview/working-with-objects/annotations/) and
[Labels](https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/) can also be a useful way to communicate information through to running game server processes from outside those processes.
This is especially useful when combined with `GameServerAllocation` [applied metadata]({{< ref "/docs/Reference/gameserverallocation.md" >}}).

Since the GameServer contains an entire [PodTemplate](https://kubernetes.io/docs/concepts/workloads/pods/pod-overview/#pod-templates)
the returned object is limited to that configuration that was deemed useful. If there are
areas that you feel are missing, please [file an issue](https://github.com/googleforgames/agones/issues) or pull request.

The easiest way to see what is exposed, is to check
the {{% ghlink href="proto/sdk/sdk.proto" %}}`sdk.proto`{{% /ghlink %}}, specifically at
the `message GameServer`.

For language specific documentation, have a look at the respective source (linked above), 
and the {{< ghlink href="examples" >}}examples{{< /ghlink >}}.

You can use `SDKWatchSendOnExecute` feature passed as a [feature gate]({{< ref "/docs/Guides/feature-stages.md#feature-gates" >}}) if you want a `GameServer` to be returned right after `SDK.WatchGameServer` is called.

### Metadata Management

#### SetLabel(key, value)

This will set a [Label](https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/) value on the backing `GameServer`
record that is stored in Kubernetes. To maintain isolation, the `key` value is automatically prefixed with "agones.dev/sdk-".

{{< alert title="Warning" color="warning">}}
There are limits on the characters that be used for label keys and values. Details are [here](https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/#syntax-and-character-set).
{{< /alert >}}

This can be useful if you want information from your running game server process to be observable or searchable through the Kubernetes API.  

#### SetAnnotation(key, value)

This will set a [Annotation](https://kubernetes.io/docs/concepts/overview/working-with-objects/annotations/) value on the backing
`GameServer` record that is stored in Kubernetes. To maintain isolation, the `key` value is automatically prefixed with "agones.dev/sdk-".

This can be useful if you want information from your running game server process to be observable through the Kubernetes API.


### Player Tracking

{{< alpha title="Player Tracking" gate="PlayerTracking" >}}

#### Alpha().PlayerConnect(playerID)

This function increases the SDK’s stored player count by one, and appends this playerID to 
`GameServer.Status.Players.IDs`.

[`GameServer.Status.Players.Count` and `GameServer.Status.Players.IDs`][playerstatus]
are then set to update the player count and id list a second from now,
unless there is already an update pending, in which case the update joins that batch operation.

`PlayerConnect()` returns true and adds the playerID to the list of playerIDs if this playerID was not already in the
list of connected playerIDs.

If the playerID exists within the list of connected playerIDs, `PlayerConnect()` will return false, and the list of
connected playerIDs will be left unchanged.

An error will be returned if the playerID was not already in the list of connected playerIDs but the player capacity for
the server has been reached. The playerID will not be added to the list of playerIDs.

{{< alert title="Note" color="info">}}
Do not use this method if you are manually managing `GameServer.Status.Players.IDs` and `GameServer.Status.Players.Count`
through the Kubernetes API, as indeterminate results will occur.  
{{< /alert >}}
    
#### Alpha().PlayerDisconnect(playerID)

This function decreases the SDK’s stored player count by one, and removes the playerID from 
[`GameServer.Status.Players.IDs`][playerstatus].

`GameServer.Status.Players.Count` and `GameServer.Status.Players.IDs` are then set to 
update the player count and id list a second from now,
unless there is already an update pending, in which case the update joins that batch operation.

`PlayerDisconnect()` will return true and remove the supplied playerID from the list of connected playerIDs if the
playerID value exists within the list.

If the playerID was not in the list of connected playerIDs, the call will return false, and the connected playerID list
will be left unchanged.

{{< alert title="Note" color="info">}}
Do not use this method if you are manually managing `GameServer.Status.Players.IDs` and `GameServer.Status.Players.Count`
through the Kubernetes API, as indeterminate results will occur.  
{{< /alert >}}

#### Alpha().SetPlayerCapacity(count)

Update the [`GameServer.Status.Players.Capacity`][playerstatus] value with a new capacity.

#### Alpha().GetPlayerCapacity()

This function retrieves the current player capacity. This is always accurate from what has been set through this SDK,
even if the value has yet to be updated on the GameServer status resource.

{{< alert title="Note" color="info">}}
If `GameServer.Status.Players.Capacity` is set manually through the Kubernetes API, use `SDK.GameServer()` or 
`SDK.WatchGameServer()` instead to view this value.
{{< /alert >}}

#### Alpha().GetPlayerCount()

This function returns if the playerID is currently connected to the GameServer. 
This is always accurate from what has been set through this SDK,
even if the value has yet to be updated on the GameServer status resource.

{{< alert title="Note" color="info">}}
If `GameServer.Status.Players.IDs` is set manually through the Kubernetes API, use SDK.GameServer() 
or SDK.WatchGameServer() instead to retrieve the current player count.
{{< /alert >}}

#### Alpha().IsPlayerConnected(playerID)

This function returns if the playerID is currently connected to the GameServer. This is always accurate from what has
been set through this SDK,
even if the value has yet to be updated on the GameServer status resource.

{{< alert title="Note" color="info">}}
If `GameServer.Status.Players.IDs` is set manually through the Kubernetes API, use SDK.GameServer() 
or SDK.WatchGameServer() instead to determine connected status.
{{< /alert >}}

#### Alpha().GetConnectedPlayers()

This function returns the list of the currently connected player ids. This is always accurate from what has been set
through this SDK, even if the value has yet to be updated on the GameServer status resource.

{{< alert title="Note" color="info">}}
If `GameServer.Status.Players.IDs` is set manually through the Kubernetes API, use SDK.GameServer() 
or SDK.WatchGameServer() instead to list the connected players.
{{< /alert >}}

[playerstatus]: {{< ref "/docs/Reference/agones_crd_api_reference.html#agones.dev/v1.PlayerStatus" >}}

## Writing your own SDK

If there isn't an SDK for the language and platform you are looking for, you have several options:

### gRPC Client Generation

If client generation is well supported by [gRPC](https://grpc.io/docs/), then generate client(s) from
the proto files found in the {{% ghlink href="proto/sdk" %}}`proto/sdk`{{% /ghlink %}},
directory and look at the current {{< ghlink href="sdks" >}}sdks{{< /ghlink >}} to see how the wrappers are
implemented to make interaction with the SDK server simpler for the user.

### REST API Implementation

If client generation is not well supported by gRPC, or if there are other complicating factors, implement the SDK through
the [REST]({{< relref "rest.md" >}}) HTTP+JSON interface. This could be written by hand, or potentially generated from
the {{< ghlink href="sdks/swagger" >}}Swagger/OpenAPI Specifications{{< /ghlink >}}.

Finally, if you build something that would be usable by the community, please submit a pull request!

## SDK Conformance Test

There is a tool `SDK server Conformance` checker which will run Local SDK server and record all requests your client is performing.

In order to check that SDK is working properly you should write simple SDK test client which would use all methods of your SDK.

Also to test that SDK client is receiving valid Gameserver data, your binary should set the same `Label` value as creation timestamp which you will receive as a result of GameServer() call and `Annotation` value same as gameserver UID received by Watch gameserver callback.

Complete list of endpoints which should be called by your test client is the following:
```
ready,allocate,setlabel,setannotation,gameserver,health,shutdown,watch
```

In order to run this test SDK server locally use:
```
SECONDS=30 make run-sdk-conformance-local
```

Docker container would timeout in 30 seconds and give your the comparison of received requests and expected requests

For instance you could run Go SDK conformance test and see how the process goes: 
```
SDK_FOLDER=go make run-sdk-conformance-test
```

In order to add test client for your SDK, write `sdktest.sh` and `Dockerfile`. Refer to {{< ghlink href="build/build-sdk-images/go" >}}Golang SDK Conformance testing directory structure{{< /ghlink >}}.

## Building the Tools

If you wish to build the binaries from source
the `make` target `build-agones-sdk-binary` will compile the necessary binaries
for all supported operating systems (64 bit windows, linux and osx).

You can find the binaries in the `bin` folder in {{< ghlink href="cmd/sdk-server" >}}`cmd/sdk-server`{{< /ghlink >}}
once compilation is complete.

See {{< ghlink href="build" branch="master" >}}Developing, Testing and Building Agones{{< /ghlink >}} for more details.
