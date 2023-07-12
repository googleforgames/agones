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

To do this, we can run the same binary (the SDK Server) that runs inside Agones, but pass in a flag
to run it in "local mode". Local mode means that the sidecar binary
will not try to connect to anything, and will just send log messages to stdout and persist local state in memory so
that you can see exactly what the SDK in your game server is doing, and can
confirm everything works.

For local development with integration into your cluster, the SDK Server can also be run in "out of cluster mode", discussed more [below](#running-locally-using-out-of-cluster-mode).

## Running the SDK Server

To run the SDK Server, you will need a copy of the binary.
This can either be done by downloading a prebuilt binary or running from source code.

### Getting the prebuilt binary

To run the prebuilt binary, for the latest release, you will need to download {{% ghrelease file_prefix="agonessdk-server" %}}, and unzip it.
You will find the executables for the SDK server, for each type of operating system.
`sdk-server.darwin.amd64` and `sdk-server.darwin.arm64` are for <u>MacOS</u>, `sdk-server.linux.amd64` and `sdk-server.linux.arm64` are for <u>Linux</u>, and `sdk-server.windows.amd64.exe` is for <u>Windows</u>.

### Running from source code

Instead of downloading and running executable binaries from the internet, you can simply run from source code.
First clone the [Agones GitHub repo](https://github.com/googleforgames/agones).
You can switch to a specific release's branch/tag or run from main.
Running from source code will require having golang installed, which can be done by following the instructions [here](https://go.dev/doc/install).

With golang installed and the Agones repository cloned, the SDK Server can easily be run with the following command (from the agones clone directory):
```bash
go run cmd/sdk-server/main.go
```
Note: This command does not contain any of the necessary command line flags used in the following sections.
It simply serves as an example of how to run from source code instead of running the prebuilt binary.
Passing commandline flags (e.g. `--local`) will work in the same way.

### Running In "Local Mode"

To run in local mode, pass the flag `--local` to the executable.

For example:

```bash
./sdk-server.linux.amd64 --local
```
or
```bash
go run cmd/sdk-server/main.go --local
```
You should see output similar to the following:
```
{"ctlConf":{"Address":"localhost","IsLocal":true,"LocalFile":"","Delay":0,"Timeout":0,"Test":"","GRPCPort":9357,"HTTPPort":9358},"message":"Starting sdk sidecar","severity":"info","source":"main","time":"2019-10-30T21:44:37.973139+03:00","version":"1.1.0"}
{"grpcEndpoint":"localhost:9357","message":"Starting SDKServer grpc service...","severity":"info","source":"main","time":"2019-10-30T21:44:37.974585+03:00"}
{"httpEndpoint":"localhost:9358","message":"Starting SDKServer grpc-gateway...","severity":"info","source":"main","time":"2019-10-30T21:44:37.975086+03:00"}
{"message":"Ready request has been received!","severity":"info","time":"2019-10-30T21:45:47.031989+03:00"}
{"message":"gameserver update received","severity":"info","time":"2019-10-30T21:45:47.03225+03:00"}
{"message":"Shutdown request has been received!","severity":"info","time":"2019-10-30T21:46:18.179341+03:00"}
{"message":"gameserver update received","severity":"info","time":"2019-10-30T21:46:18.179459+03:00"}
```

An alternative to "local mode" ("out of cluster mode", which uses an agones cluster) is discussed [below](#running-locally-using-out-of-cluster-mode).

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
docker run --network=host --rm us-docker.pkg.dev/agones-images/release/agones-sdk:{{< release-version >}} --local
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
docker run --network=host --rm -v $(pwd)/gameserver.yaml:/tmp/gameserver.yaml us-docker.pkg.dev/agones-images/release/agones-sdk:{{<release-version>}} --local -f /tmp/gameserver.yaml
```

## Running locally using "out of cluster" mode

An alternative to running completely isolated from a cluster is to run in "out of cluster" mode.
This allows you to run locally but interact with the controllers within a cluster.
This workflow works well when running a [Local Game Server](https://agones.dev/site/docs/guides/local-game-server/) in your cluster and want to run the server locally.
This means being able to allocate your game in its normal flow (much more prod-like environment) and be able to debug (e.g. breakpoint) your server code.
This can also be done with [running Minikube locally](https://agones.dev/site/docs/installation/creating-cluster/minikube/), which is great for early prototyping.

The name "out of cluster" mode is to contrast [InClusterConfig](https://pkg.go.dev/k8s.io/client-go/tools/clientcmd#InClusterConfig) which is used in the internal golang kubeconfig API.

To run in "out of cluster" mode, instead of passing `--local` or `-f ./gameserver.yaml`, you'd use `--kubeconfig` to connect into your cluster.
However, there are a number of commands that are necessary/useful to run when running in "out of cluster" mode.
Here is a sample with each piece discussed after.
```bash
go run cmd/sdk-server/main.go \
  --gameserver-name my-local-server \
  --pod-namespace default \
  --kubeconfig "$HOME/.kube/config" \
  --address 0.0.0.0 \
  --graceful-termination false
```

* `--gameserver-name` is a necessary arg, passed instead of the `GAMESERVER_NAME` enviroment variable.
  * It is set to the name of the dev `GameServer` k8s resource.
  * It tells the SDK Sever which resource to read/write to on the k8s cluster.
  * This example value of `my-local-server` matches to the instructions for setting up a [Local Game Server](https://agones.dev/site/docs/guides/local-game-server/).
* `--pod-namespacee` is a necessary arg, passed instead of the `POD_NAMESPACE` enviroment variable.
  * It is set set to the namespace which your `GameServer` resides in.
  * It tells the SDK Sever which namespace to look under for the `GameServer` to read/write to on the k8s cluster.
  * This example value of `default` is used as most instructions in this documentation assumes `GameServers` to be created in the `default` namespace.
* `--kubeconfig` tells the SDK Server how to connect to your cluster.
  * This actually does not trigger any special flow.
  * The SDK Server will run just as it would when created as a Sidecar in a k8s cluster.
  * Passing this argument simply provides where to connect along with the credentials to do so.
  * This example value of `"$HOME/.kube/config"` is the default location for your k8s authentication information. This requires you be logged in via `kubectl` and have the desired cluster selected via [`kubectl config use-context`](https://jamesdefabia.github.io/docs/user-guide/kubectl/kubectl_config_use-context/).
* `--address` specifies the binding IP address for the SDK Server.
  * By default, the binding address is `localhost`. This may be difficult for some development setups.
  * Overriding this value changes which IP address(es) the server will bind to for receiving gRPC/REST SDK API calls.
  * This example value of `0.0.0.0` sets the SDK Server to receive API calls that are sent to any IP address (that reach your machine).
* `--graceful-termination` set to false will disable some smooth state transitions when exiting.
  * By default, the SDK Server will wait until the `GameServer` has reached the `Shutdown` state before exiting ("graceful termination").
  * This will cause the SDK Server to hang (waiting on state update) when attempting to terminate (e.g. with `^C`).
  * When running binaries in a development context, quickly exiting and restarting the SDK Server is handy.

Now that you have the SDK Server running locally with k8s credentials, you can run your game server binary in an integrated fashion.
Your game server's SDK calls will reach the local SDK Server, which will then interact with the `GameServer` resource on your cluster.
You can allocate this `GameServer` just as you would for a normal `GameServer` running completely on k8s.
The state update to `Allocated` will show in your local game server binary.
