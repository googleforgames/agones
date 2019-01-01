---
title: "C++ Game Server Client SDK"
linkTitle: "C++"
date: 2019-01-02T10:17:50Z
weight: 10
description: "This is the C++ version of the Agones Game Server Client SDK. "
---

Check the [Client SDK Documentation]({{< relref "_index.md" >}}) for more details on each of the SDK functions and how to run the SDK locally.

## Download

Download the source from the [Releases Page](https://github.com/GoogleCloudPlatform/agones/releases) 
or {{< ghlink href="sdks/rust" >}}directly from Github{{< /ghlink >}}.

## Usage

The C++ SDK is specifically designed to be as simple as possible, and deliberately doesn't include any kind
of singleton management, or threading/asynchronous processing to allow developers to manage these aspects as they deem
appropriate for their system.  

We may consider these types of features in the future, depending on demand. 

To begin working with the SDK, create an instance of it.
```cpp
agones::SDK *sdk = new agones::SDK();
```

To connect to the SDK server, either local or when running on Agones, run the `sdk->Connect()` method.
This will block for up to 30 seconds if the SDK server has not yet started and the connection cannot be made,
and will return `false` if there was an issue connecting.

```cpp
bool ok = sdk->Connect();
```

To send a [health check]({{< relref "_index.md#health" >}}) ping call `sdk->Health()`. This is a synchronous request that will
return `false` if it has failed in any way. Read [GameServer Health Checking]({{< relref "../health-checking.md" >}}) for more
details on the game server health checking strategy.

```cpp
bool ok = sdk->Health();
```

To mark the game server as [ready to receive player connections]({{< relref "_index.md#ready" >}}), call `sdk->Ready()`.
This will return a grpc::Status object, from which we can call `status.ok()` to determine
if the function completed successfully.

For more information you can also look at the [gRPC Status reference](https://grpc.io/grpc/cpp/classgrpc_1_1_status.html)

```cpp
grpc::Status status = sdk->Ready();
if (!status.ok()) { ... }
```

To mark that the [game session is completed]({{< relref "_index.md#shutdown" >}}) and the game server should be shut down call `sdk->Shutdown()`. 

This will return a grpc::Status object, from which we can call `status.ok()` to determine
if the function completed successfully.

For more information you can also look at the [gRPC Status reference](https://grpc.io/grpc/cpp/classgrpc_1_1_status.html)

```cpp
grpc::Status status = sdk->Shutdown();
if (!status.ok()) { ... }
```

To [set a Label]({{< relref "_index.md#setlabelkey-value" >}}) on the backing `GameServer` call
`sdk->SetLabel(key, value)`.

This will return a grpc::Status object, from which we can call `status.ok()` to determine
if the function completed successfully.

For more information you can also look at the [gRPC Status reference](https://grpc.io/grpc/cpp/classgrpc_1_1_status.html)

```cpp
grpc::Status status = sdk->SetLabel("test-label", "test-value");
if (!status.ok()) { ... }
```

To [set an Annotation]({{< relref "_index.md#setannotationkey-value" >}}) on the backing `GameServer` call
`sdk->SetAnnotation(key, value)`.

This will return a grpc::Status object, from which we can call `status.ok()` to determine
if the function completed successfully.

For more information you can also look at the [gRPC Status reference](https://grpc.io/grpc/cpp/classgrpc_1_1_status.html)

```cpp
status = sdk->SetAnnotation("test-annotation", "test value");
if (!status.ok()) { ... }
```


To get the details on the [backing `GameServer`]({{< relref "_index.md#gameserver" >}}) call `sdk->GameServer(&gameserver)`,
passing in a `stable::agones::dev::sdk::GameServer*` to push the results of the `GameServer` configuration into.

This function will return a grpc::Status object, from which we can call `status.ok()` to determine
if the function completed successfully.

```cpp
stable::agones::dev::sdk::GameServer gameserver;
grpc::Status status = sdk->GameServer(&gameserver);
if (!status.ok()) {...}
```

To get [updates on the backing `GameServer`]({{< relref "_index.md#watchgameserverfunctiongameserver" >}}) as they happen, 
call `sdk->WatchGameServer([](stable::agones::dev::sdk::GameServer gameserver){...})`.

This will call the passed in `std::function`
synchronously (this is a blocking function, so you may want to run it in its own thread) whenever the backing `GameServer`
is updated.

```cpp
sdk->WatchGameServer([](stable::agones::dev::sdk::GameServer gameserver){
    std::cout << "GameServer Update, name: " << gameserver.object_meta().name() << std::endl;
    std::cout << "GameServer Update, state: " << gameserver.status().state() << std::endl;
});
```

For more information, you can also read the [SDK Overview]({{< relref "_index.md" >}}), check out 
{{< ghlink href="sdks/cpp/sdk.h" >}}sdk.h{{< /ghlink >}} and also look at the
{{< ghlink href="examples/cpp-simple" >}}C++ example{{< / >}}.

### Failure
When running on Agones, the above functions should only fail under exceptional circumstances, so please 
file a bug if it occurs.

## Dynamic and Static Libraries

In the [releases](https://github.com/googlecloudplatform/agones/releases) folder
you will find two archives for download that contain both static and dynamic libraries for building your
game server on Linux:

- `argonsdk-$(VERSION)-dev-linux-arch_64.tar.gz`: This includes all the 
headers as well as dynamic and static libraries that are needed for development and runtime.
- `argonsdk-$(VERSION)-runtime-linux-arch_64.tar.gz`: This includes just the dynamic libraries that 
are needed to run a binary compiled against the SDK and its dependencies.

### Building the Libraries

If you want to build the libraries from Agones source, 
the `make` target `build-sdk-cpp` will compile both static and dynamic libraries for Debian/Linux
for your usage, to be found in the `bin` directory inside this one.

## Building From Source
If you wish to compile from source, you will need to compile and install the following dependencies:

- [gRPC](https://grpc.io), v1.16.1 - [C++ compilation guide](https://github.com/grpc/grpc/tree/v1.16.x/src/cpp)

## Windows and macOS

If you are building a server on Windows or macOS, and need a development build to run on
that platform, at this time you will need to compile from source. Windows and macOS libraries
for the C++ SDK for easier cross platform development are planned and will be provided in the near future.
