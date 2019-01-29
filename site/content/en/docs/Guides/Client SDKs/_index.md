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

- [C++]({{< relref "cpp.md" >}})
- [Go](https://godoc.org/agones.dev/agones/sdks/go)
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
areas that you feel are missing, please [file an issue](https://github.com/GoogleCloudPlatform/agones/issues) or pull request.

The easiest way to see what is exposed, is to check the  {{< ghlink href="sdk.proto" >}}`sdk.proto`{{< /ghlink >}},
specifically at the `message GameServer`.

For language specific documentation, have a look at the respective source (linked above), 
and the {{< ghlink href="examples" >}}examples{{< /ghlink >}}.

### WatchGameServer(function(gameserver){...})

This executes the passed in callback with the current `GameServer` details whenever the underlying `GameServer` configuration is updated.
This can be useful to track `GameServer > Status > State` changes, `metadata` changes, such as labels and annotations, and more.

In combination with this SDK, manipulating [Annotations](https://kubernetes.io/docs/concepts/overview/working-with-objects/annotations/) and
[Labels](https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/) can also be a useful way to communicate information through to running game server processes from outside those processes.
This is especially useful when combined with `FleetAllocation` [applied metadata]({{< ref "/docs/Reference/fleet.md#fleet-allocation-specification" >}}).  

Since the GameServer contains an entire [PodTemplate](https://kubernetes.io/docs/concepts/workloads/pods/pod-overview/#pod-templates)
the returned object is limited to that configuration that was deemed useful. If there are
areas that you feel are missing, please [file an issue](https://github.com/GoogleCloudPlatform/agones/issues) or pull request.

The easiest way to see what is exposed, is to check the {{< ghlink href="sdk.proto" >}}`sdk.proto`{{< /ghlink >}},
specifically at the `message GameServer`.

For language specific documentation, have a look at the respective source (linked above), 
and the {{< ghlink href="examples" >}}examples{{< /ghlink >}}.

## Writing your own SDK

If there isn't a SDK for the language and platform you are looking for, you have several options:

### gRPC Client Generation

If client generation is well supported by [gRPC](https://grpc.io/docs/), then generate a client from the
{{< ghlink href="sdk.proto" >}}`sdk.proto`{{< /ghlink >}}, and look at the current {{< ghlink href="sdks" >}}sdks{{< /ghlink >}} to see how the wrappers are implemented to make interaction
with the SDK server simpler for the user.

### REST API Implementation

If client generation is not well supported by gRPC, or if there are other complicating factors, implement the SDK through
the [REST]({{< relref "rest.md" >}}) HTTP+JSON interface. This could be written by hand, or potentially generated from
the {{< ghlink href="sdk.swagger.json" >}}Swagger/OpenAPI Spec{{< /ghlink >}}.

Finally, if you build something that would be usable by the community, please submit a pull request!

## Building the Tools

If you wish to build the binaries from source
the `make` target `build-agones-sdk-binary` will compile the necessary binaries
for all supported operating systems (64 bit windows, linux and osx).

You can find the binaries in the `bin` folder in {{< ghlink href="cmd/sdk-server" >}}`cmd/sdk-server`{{< /ghlink >}}
once compilation is complete.

See {{< ghlink href="build" branch="master" >}}Developing, Testing and Building Agones{{< /ghlink >}} for more details.
