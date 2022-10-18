---
title: "Local Development"
linkTitle: "Local Development"
date: 2019-01-29T10:18:08Z
weight: 1000
description: >
  Working against the SDK without having to run a full kubernetes stack
---

When the game server is running on Agones, the SDK communicates over TCP to a small
gRPC server that Agones coordinated to run in a container in the same network
namespace as it - usually referred to in Kubernetes terms as a "sidecar".

Therefore, when developing locally, we also need a process for the SDK to connect to!

To do this, we can run the same binary that runs inside Agones, but pass in a flag
to run it in "local mode". Local mode means that the sidecar binary
will not try to connect to anything, and will just send log messages to stdout and persist local state in memory so
that you can see exactly what the SDK in your game server is doing, and can
confirm everything works.

To do this you will need to download {{% ghrelease file_prefix="agonessdk-server" %}}, and unzip it.
You will find the executables for the SDK server, for each type of operating system.

macOS
* sdk-server.darwin.amd64
* sdk-server.darwin.arm64

Linux
* sdk-server.linux.amd64
* sdk-server.linux.arm64

Windows
* sdk-server.windows.amd64.exe

To run in local mode, pass the flag `--local` to the executable.

For example:

```bash
./sdk-server.linux.amd64 --local
```
```
{"ctlConf":{"Address":"localhost","IsLocal":true,"LocalFile":"","Delay":0,"Timeout":0,"Test":"","GRPCPort":9357,"HTTPPort":9358},"message":"Starting sdk sidecar","severity":"info","source":"main","time":"2019-10-30T21:44:37.973139+03:00","version":"1.1.0"}
{"grpcEndpoint":"localhost:9357","message":"Starting SDKServer grpc service...","severity":"info","source":"main","time":"2019-10-30T21:44:37.974585+03:00"}
{"httpEndpoint":"localhost:9358","message":"Starting SDKServer grpc-gateway...","severity":"info","source":"main","time":"2019-10-30T21:44:37.975086+03:00"}
{"message":"Ready request has been received!","severity":"info","time":"2019-10-30T21:45:47.031989+03:00"}
{"message":"gameserver update received","severity":"info","time":"2019-10-30T21:45:47.03225+03:00"}
{"message":"Shutdown request has been received!","severity":"info","time":"2019-10-30T21:46:18.179341+03:00"}
{"message":"gameserver update received","severity":"info","time":"2019-10-30T21:46:18.179459+03:00"}
```

## Providing your own `GameServer` configuration for local development

By default, the local sdk-server will create a default `GameServer` configuration that is used for `GameServer()`
and `WatchGameServer()` SDK calls. If you wish to provide your own configuration, as either yaml or json, this
can be passed through as either `--file` or `-f` along with the `--local` flag.

If the `GamerServer` configuration file is changed while the local server is running,
this will be picked up by the local server, and will change the current active configuration, as well as sending out
events for `WatchGameServer()`. This is a useful way of testing functionality, such as changes of state from `Ready` to
`Allocated` in your game server code.

{{< alert title="Note" color="info">}}
File modification events can fire more than one for each save (for a variety of reasons),
but it's best practice to ensure handlers that implement `WatchGameServer()` be idempotent regardless, as repeats can
happen when live as well.
{{< /alert >}}

For example:

```bash
wget https://raw.githubusercontent.com/googleforgames/agones/{{< release-branch >}}/examples/simple-game-server/gameserver.yaml
./sdk-server.linux.amd64 --local -f ./gameserver.yaml
```
```
{"ctlConf":{"Address":"localhost","IsLocal":true,"LocalFile":"./gameserver.yaml","Delay":0,"Timeout":0,"Test":"","GRPCPort":9357,"HTTPPort":9358},"message":"Starting sdk sidecar","severity":"info","source":"main","time":"2019-10-30T21:47:45.742776+03:00","version":"1.1.0"}
{"filePath":"/Users/alexander.apalikov/Downloads/agonessdk-server-1.1.0/gameserver.yaml","message":"Reading GameServer configuration","severity":"info","time":"2019-10-30T21:47:45.743369+03:00"}
{"grpcEndpoint":"localhost:9357","message":"Starting SDKServer grpc service...","severity":"info","source":"main","time":"2019-10-30T21:47:45.759692+03:00"}
{"httpEndpoint":"localhost:9358","message":"Starting SDKServer grpc-gateway...","severity":"info","source":"main","time":"2019-10-30T21:47:45.760312+03:00"}
```

## Changing State of a Local GameServer

Some SDK calls would change the GameServer state according to [GameServer State Diagram]({{< ref "../../Reference/gameserver.md#gameserver-state-diagram" >}}). Also local SDK server would persist labels and annotations updates.

Here is a complete list of these commands: ready, allocate, setlabel, setannotation, shutdown, reserve.

For example call to Reserve() for 30 seconds would change the GameServer state to Reserve and if no call to Allocate() occurs it would return back to Ready state after this period.

{{< alert title="Note" color="info">}}
All state transitions are supported for local SDK server, however not all of them are valid in the real scenario. For instance, you cannot make a transition of a GameServer from Shutdown to a Ready state, but can do using local SDK server.
{{< /alert >}}

All changes to the GameServer state could be observed and retrieved using Watch() or GameServer() methods using GameServer SDK.

Example of using HTTP gateway locally:

```bash
curl -X POST "http://localhost:9358/ready" -H "accept: application/json" -H "Content-Type: application/json" -d "{}"
```
```
{}
```
```bash
curl -GET "http://localhost:9358/gameserver" -H "accept: application/json"
```
```
{"object_meta":{"creation_timestamp":"-62135596800"},"spec":{"health":{}},"status":{"state":"Ready"}}
```
```bash
curl -X PUT "http://localhost:9358/metadata/label" -H "accept: application/json" -H "Content-Type: application/json" -d "{ \"key\": \"foo\", \"value\": \"bar\"}"
curl -GET "http://localhost:9358/gameserver" -H "accept: application/json"
```
```
{"object_meta":{"creation_timestamp":"-62135596800","labels":{"agones.dev/sdk-foo":"bar"}},"spec":{"health":{}},"status":{"state":"Ready"}}
```

## Running Local Mode in a Container

Once you have your game server process in a container, you may also want to test the container build locally as well.

Since the production agones-sdk binary has the `--local` mode built in, you can also use the production container image
locally as well!

Since the SDK and your game server container need to share a port on `localhost`, one of the easiest ways to do that 
is to have them both run using the host network, like so:

In one shell run:

```shell
docker run --network=host --rm gcr.io/agones-images/agones-sdk:{{< release-version >}} --local
```

You should see a similar output to what you would if you were running the binary directly, i.e. outside a container.

Then in another shell, start your game server container:

```shell
docker run --network=host --rm <your image here>
```

If you want to [mount a custom `gameserver.yaml`](#providing-your-own-gameserver-configuration-for-local-development), 
this is also possible:  

```bash
wget https://raw.githubusercontent.com/googleforgames/agones/{{< release-branch >}}/examples/simple-game-server/gameserver.yaml
# required so that the `agones` user in the container can read the file
chmod o+r gameserver.yaml
docker run --network=host --rm -v $(pwd)/gameserver.yaml:/tmp/gameserver.yaml gcr.io/agones-images/agones-sdk:{{<release-version>}} --local -f /tmp/gameserver.yaml
```
