# Agones Game Server Client SDKs

The SDKs are integration points for game servers with Agones itself.

They are required for a game server to work with Agones.

There are currently two support SDKs:
- [C++](cpp)
- [Go](https://godoc.org/agones.dev/agones/sdks/go)

The SDKs are relatively thin wrappers around [gRPC](https://grpc.io), generated clients,
which connects to a small process that Agones coordinates to run alongside the Game Server
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

```bash
$ ./sidecar.linux.amd64 --local
{"level":"info","local":true,"msg":"Starting sdk sidecar","port":59357,"time":"2017-12-22T16:09:03-08:00","version":"0.1-5217b21"}
{"level":"info","msg":"Ready request has been received!","time":"2017-12-22T16:09:19-08:00"}
{"level":"info","msg":"Shutdown request has been received!","time":"2017-12-22T16:10:19-08:00"}
```

### Building the Local Tools

If you wish to build the binaries for local development from source
the `make` target `build-agones-sdk-binary` will compile the necessary binaries
for all supported operating systems (64 bit windows, linux and osx).

You can find the binaries in the `bin` folder in [`cmd/sdk-server`](../cmd/sdk-server)
once compilation is complete.

See [Developing, Testing and Building Agones](../build) for more details.
