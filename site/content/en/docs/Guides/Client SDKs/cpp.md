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
or {{< ghlink href="sdks/cpp" >}}directly from Github{{< /ghlink >}}.

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

To [set a Label]({{< relref "_index.md#setlabel-key-value" >}}) on the backing `GameServer` call
`sdk->SetLabel(key, value)`.

This will return a grpc::Status object, from which we can call `status.ok()` to determine
if the function completed successfully.

For more information you can also look at the [gRPC Status reference](https://grpc.io/grpc/cpp/classgrpc_1_1_status.html)

```cpp
grpc::Status status = sdk->SetLabel("test-label", "test-value");
if (!status.ok()) { ... }
```

To [set an Annotation]({{< relref "_index.md#setannotation-key-value" >}}) on the backing `GameServer` call
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

To get [updates on the backing `GameServer`]({{< relref "_index.md#watchgameserver-function-gameserver" >}}) as they happen, 
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
{{< ghlink href="sdks/cpp/include/agones/sdk.h" >}}sdk.h{{< /ghlink >}} and also look at the
{{< ghlink href="examples/cpp-simple" >}}C++ example{{< / >}}.

### Failure
When running on Agones, the above functions should only fail under exceptional circumstances, so please 
file a bug if it occurs.

### Building the Libraries from source
CMake is used to build SDK for all platforms. It is possible to build SDK as a static or dynamic library.

## Prerequisites
* CMake >= 3.13.0
* Git
* C++14 compiler

## Options
Following options are available
- **AGONES_BUILD_SHARED** (default is OFF) - build sdk as a shared library.
- **AGONES_CREATE_PACKAGE** (default is ON) - create an "install" step, to create a cmake package.

## Windows
Building with Visual Studio:
```
md .build
cd .build
cmake .. -G "Visual Studio 15 2017 Win64" -Wno-dev
cmake --build . --config Release
```
Building with NMake
```
md .build
cd .build
cmake .. -G "NMake Makefiles" -DCMAKE_BUILD_TYPE=Release -Wno-dev
cmake --build .
```

## Linux / MacOS
```
mkdir -p .build
cd .build
cmake .. -DCMAKE_BUILD_TYPE=Release -G "Unix Makefiles" -Wno-dev
cmake --build .
```

## Remarks
CMake option `-Wno-dev` is specified to suppress [CMP0048](https://cmake.org/cmake/help/v3.13/policy/CMP0048.html) deprecation warning for gRPC depencency.

### Using SDK
In CMake-based projects it's enough to specify a folder where SDK is installed with `CMAKE_PREFIX_PATH` and use `find_package(agones CONFIG REQUIRED)` command. For example: {{< ghlink href="examples/cpp-simple" >}}cpp-simple{{< / >}}.
If **AGONES_CREATE_PACKAGE** option is off, then `CMAKE_PREFIX_PATH` should be set to a path where SDK is built (usualy `agones/sdks/cpp/.build`).
It maybe useful to disable some [protobuf warnings](https://github.com/protocolbuffers/protobuf/blob/master/cmake/README.md#notes-on-compiler-warnings) in your project.
