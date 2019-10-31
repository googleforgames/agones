---
title: "Local Development"
linkTitle: "Local Development"
date: 2019-01-29T10:18:08Z
weight: 1000
description: >
  Working against the SDK without having to run a full kubernetes stack
---

## Local Development

When the game server is running on Agones, the SDK communicates over TCP to a small
gRPC server that Agones coordinated to run in a container in the same network 
namespace as it - usually referred to in Kubernetes terms as a "sidecar".

Therefore, when developing locally, we also need a process for the SDK to connect to!

To do this, we can run the same binary that runs inside Agones, but pass in a flag
to run it in "local mode". Local mode means that the sidecar binary
will not try to connect to anything, and will just send log messages to stdout and persist local state in memory so 
that you can see exactly what the SDK in your game server is doing, and can
confirm everything works.

To do this you will need to download the latest agonessdk-server-{VERSION}.zip from 
[releases](https://github.com/googleforgames/agones/releases), and unzip it.
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

```console
$ export AGONES_PATH=/home/go/src/agones.dev/agones
$ ./sdk-server.linux.amd64 --local -f ${AGONES_PATH}/examples/simple-udp/gameserver.yaml
{"ctlConf":{"Address":"localhost","IsLocal":true,"LocalFile":"/home/go/src/agones.dev/agones/examples/simple-udp/gameserver.yaml"},"grpcPort":59357,"httpPort":59358,"level":"info","msg":"Starting sdk sidecar","source":"main","time":"2018-08-25T17:56:39-07:00","version":"0.4.0-b44960a8"}
{"level":"info","msg":"Reading GameServer configuration","path":"/home/user/workspace/agones/src/agones.dev/agones/examples/simple-udp/gameserver.yaml","source":"main","time":"2018-08-25T17:56:39-07:00"}
{"level":"info","msg":"Starting SDKServer grpc service...","source":"main","time":"2018-08-25T17:56:39-07:00"}
{"level":"info","msg":"Starting SDKServer grpc-gateway...","source":"main","time":"2018-08-25T17:56:39-07:00"}
```

### Changing State of a Local GameServer

Some SDK calls would change the GameServer state according to [GameServer State Diagram]({{< ref "../../Reference/gameserver.md#gameserver-state-diagram" >}}). Also local SDK server would persist labels and annotations updates.

Here is a complete list of these commands: ready, allocate, setlabel, setannotation, shutdown, reserve.

For example call to Reserve() for 30 seconds would change the GameServer state to Reserve and if no call to Allocate() occurs it would return back to Ready state after this period.

{{< alert title="Note" color="info">}}
All state transitions are supported for local SDK server, however not all of them are valid in the real scenario. For instance, you cannot make a transition of a GameServer from Shutdown to a Ready state, but can do using local SDK server.
{{< /alert >}}

All changes to the GameServer state could be observed and retrieved using Watch() or GameServer() methods using GameServer SDK.

Example of using HTTP gateway locally:
```console
$ curl -X POST "http://localhost:59358/ready" -H "accept: application/json" -H "Content-Type: application/json" -d "{}"
{}
$ curl -GET "http://localhost:59358/gameserver" -H "accept: application/json"
{"object_meta":{"creation_timestamp":"-62135596800"},"spec":{"health":{}},"status":{"state":"Ready"}}
$ curl -X PUT "http://localhost:59358/metadata/label" -H "accept: application/json" -H "Content-Type: application/json" -d "{ \"key\": \"foo\", \"value\": \"bar\"}"
$ curl -GET "http://localhost:59358/gameserver" -H "accept: application/json"}"
{"object_meta":{"creation_timestamp":"-62135596800","labels":{"agones.dev/sdk-foo":"bar"}},"spec":{"health":{}},"status":{"state":"Ready"}}
```
