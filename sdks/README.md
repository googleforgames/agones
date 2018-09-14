# Agones Game Server Client SDKs

The SDKs are integration points for game servers with Agones itself.

They are required for a game server to work with Agones.

The current supported SDKs are:
- [C++](cpp)
- [Go](https://godoc.org/agones.dev/agones/sdks/go)
- [Rust](rust)
- [REST](../docs/sdk_rest_api.md)

The SDKs are relatively thin wrappers around [gRPC](https://grpc.io) generated clients,
or an implementation of the REST API (exposed via [grpc-gateway](https://github.com/grpc-ecosystem/grpc-gateway)), 
where gRPC client generation and compilation isn't well supported.

They connect to a small process that Agones coordinates to run alongside the Game Server
in a Kubernetes [`Pod`](https://kubernetes.io/docs/concepts/workloads/pods/pod-overview/).
This means that more languages can be supported in the future with minimal effort
(but pull requests are welcome! 😊 ).

## Function Reference

While each of the SDKs are canonical to their languages, they all have the following
functions that implement the core responsibilities of the SDK.

For language specific documentation, have a look at the respective source (linked above), 
and the [examples](../examples).

### Ready()
This tells Agones that the Game Server is ready to take player connections.
Once a Game Server has specified that it is `Ready`, then the Kubernetes
GameServer record will be moved to the `Ready` state, and the details
for its public address and connection port will be populated.

### Health()
This sends a single ping to designate that the Game Server is alive and
healthy. Failure to send pings within the configured thresholds will result
in the GameServer being marked as `Unhealthy`. 

See the [gameserver.yaml](../examples/gameserver.yaml) for all health checking
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

The easiest way to see what is exposed, is to check the [`sdk.proto`](https://github.com/GoogleCloudPlatform/agones/blob/master/sdk.proto),
specifically at the `message GameServer`.

For language specific documentation, have a look at the respective source (linked above), 
and the [examples](../examples).

### WatchGameServer(function(gameserver){...})

This executes the passed in callback with the current `GameServer` details whenever the underlying `GameServer` configuration is updated.
This can be useful to track `GameServer > Status > State` changes, `metadata` changes, such as labels and annotations, and more.

In combination with this SDK, manipulating [Annotations](https://kubernetes.io/docs/concepts/overview/working-with-objects/annotations/) and
[Labels](https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/) can also be a useful way to communicate information through to running game server processes from outside those processes.
This is especially useful when combined with `FleetAllocation` [applied metadata](../docs/fleet_spec.md#fleet-allocation-specification).  

Since the GameServer contains an entire [PodTemplate](https://kubernetes.io/docs/concepts/workloads/pods/pod-overview/#pod-templates)
the returned object is limited to that configuration that was deemed useful. If there are
areas that you feel are missing, please [file an issue](https://github.com/GoogleCloudPlatform/agones/issues) or pull request.

The easiest way to see what is exposed, is to check the [`sdk.proto`](https://github.com/GoogleCloudPlatform/agones/blob/master/sdk.proto),
specifically at the `message GameServer`.

For language specific documentation, have a look at the respective source (linked above), 
and the [examples](../examples).

## Local Development

When the game server is running on Agones, the SDK communicates over TCP to a small
gRPC server that Agones coordinated to run in a container in the same network 
namespace as it - usually referred to in Kubernetes terms as a "sidecar".

Therefore, when developing locally, we also need a process for the SDK to connect to!

To do this, we can run the same binary that runs inside Agones, but pass in a flag
to run it in "local mode". Local mode means that the sidecar binary
won't try and connect to anything, and will just send logs messages to stdout so 
that you can see exactly what the SDK in your game server is doing, and can
confirm everything works.

To do this you will need to download the latest agonessdk-server-{VERSION}.zip from 
[releases](https://github.com/googlecloudplatform/agones/releases), and unzip it.
You will find the executables for the SDK server, one for each type of operating system.

- `sdk-server.windows.amd64.exe` - Windows
- `sdk-server.darwin.amd64` - macOS  
-  `sdk-server.linux.amd64` - Linux

To run in local mode, pass the flag `--local` to the executable.

For example:

```console
$ ./sidecar.linux.amd64 --local
{"ctlConf":{"Address":"localhost","IsLocal":true,"LocalFile":""},"grpcPort":59357,"httpPort":59358,"level":"info","msg":"Starting sdk sidecar","source":"main","time":"2018-08-25T18:01:58-07:00","version":"0.4.0-b44960a8"}
{"level":"info","msg":"Starting SDKServer grpc service...","source":"main","time":"2018-08-25T18:01:58-07:00"}
{"level":"info","msg":"Starting SDKServer grpc-gateway...","source":"main","time":"2018-08-25T18:01:58-07:00"}
{"level":"info","msg":"Ready request has been received!","time":"2017-12-22T16:09:19-08:00"}
{"level":"info","msg":"Shutdown request has been received!","time":"2017-12-22T16:10:19-08:00"}
```

### Providing your own `GameServer` configuration for local development

By default, the local sdk-server will create a dummy `GameServer` configuration that is used for `GameServer()`
and `WatchGameServer()` SDK calls. If you wish to provide your own configuration, as either yaml or json, this
can be passed through as either `--file` or `-f` along with the `--local` flag.

For example:

```console
$ ./sdk-server.linux.amd64 --local -f ../../../examples/simple-udp/gameserver.yaml
{"ctlConf":{"Address":"localhost","IsLocal":true,"LocalFile":"../../../examples/simple-udp/gameserver.yaml"},"grpcPort":59357,"httpPort":59358,"level":"info","msg":"Starting sdk sidecar","source":"main","time":"2018-08-25T17:56:39-07:00","version":"0.4.0-b44960a8"}
{"level":"info","msg":"Reading GameServer configuration","path":"/home/user/workspace/agones/src/agones.dev/agones/examples/simple-udp/gameserver.yaml","source":"main","time":"2018-08-25T17:56:39-07:00"}
{"level":"info","msg":"Starting SDKServer grpc service...","source":"main","time":"2018-08-25T17:56:39-07:00"}
{"level":"info","msg":"Starting SDKServer grpc-gateway...","source":"main","time":"2018-08-25T17:56:39-07:00"}
```

### Writing your own SDK

If there isn't a SDK for the language and platform you are looking for, you have several options:

#### gRPC Client Generation

If client generation is well supported by [gRPC](https://grpc.io/docs/), then generate a client from the
[sdk.proto](../sdk.proto), and look at the current [sdks](.) to see how the wrappers are implemented to make interaction
with the SDK server simpler for the user.

#### REST API Implementation

If client generation is not well supported by gRPC, or if there are other complicating factors, implement the SDK through
the [REST](../docs/sdk_rest_api.md) HTTP+JSON interface. This could be written by hand, or potentially generated from
the [Swagger/OpenAPI Spec](../sdk.swagger.json).  

Finally, if you build something that would be usable by the community, please submit a pull request!

### Building the Local Tools

If you wish to build the binaries for local development from source
the `make` target `build-agones-sdk-binary` will compile the necessary binaries
for all supported operating systems (64 bit windows, linux and osx).

You can find the binaries in the `bin` folder in [`cmd/sdk-server`](../cmd/sdk-server)
once compilation is complete.

See [Developing, Testing and Building Agones](../build) for more details.
