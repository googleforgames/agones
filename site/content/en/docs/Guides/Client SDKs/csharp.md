---
title: "C# Game Server Client SDK"
linkTitle: "C#"
date: 2020-2-25
weight: 50
description: "This is the C# version of the Agones Game Server Client SDK."
publishDate: 2020-05-28
---

{{< alert title="Note" color="info" >}}
The C# SDK will also be available as a NuGet package in a future release.
{{< /alert >}}

Check the [Client SDK Documentation]({{< relref "_index.md" >}}) for more details on each of the SDK functions and how to run the SDK locally.

## Download

Download the source {{< ghlink href="sdks/csharp" >}}directly from GitHub{{< /ghlink >}}.

## Prerequisites

- .Net Standard 2.0 compliant framework.

## Usage

Reference the SDK in your project & create a new instance of the SDK wrapper:

```csharp
var agones = new AgonesSDK();
```

To connect to the SDK server, either locally or when running on Agones, run the `ConnectAsync()` method.
This will wait for up to 30 seconds if the SDK server has not yet started and the connection cannot be made,
and will return `false` if there was an issue connecting.

```csharp
bool ok = await agones.ConnectAsync();
```

To mark the game server as [ready to receive player connections]({{< relref "_index.md#ready" >}}), call the async method `ReadyAsync()`.

```csharp
async void SomeMethod()
{
    var status = await agones.ReadyAsync();
}
```

To send `Health` pings, call the async method `HealthAsync()`
```csharp
await agones.HealthAsync();
```

To get the details on the [backing `GameServer`]({{< relref "_index.md#gameserver" >}}) call `GetGameServerAsync()`.

Will return `null` if there is an error in retrieving the `GameServer` record.

```csharp
var gameserver = await agones.GetGameServerAsync();
```

To mark the GameServer as [Reserved]({{< relref "_index.md#reserveseconds" >}}) for a duration call 
`ReserveAsync(long duration)`.

```csharp
long duration = 30;
var status = await agones.ReserveAsync(duration);
```

To mark that the [game session is completed]({{< relref "_index.md#shutdown" >}}) and the game server should be shut down call `ShutdownAsync()`.

```csharp
var status = await agones.ShutdownAsync();
```

Similarly `SetAnnotation(string key, string value)` and `SetLabel(string key, string value)` are async methods that perform an action & return a `Status` object.

To watch when 
[the backing `GameServer` configuration changes]({{< relref "_index.md#watchgameserverfunctiongameserver" >}})
call `WatchGameServer(callback)`, where the delegate function `callback` of type `Action<GameServer>` will be executed every time the `GameServer` 
configuration changes.
This process is non-blocking internally.

```csharp
agonesSDK.WatchGameServer((gameServer) => { Console.WriteLine($"Server - Watch {gameServer}");});
```


## Remarks
- All requests other than `ConnectAsync` will wait for up to 15 seconds before giving up, can also be set in the constructor.
- Default host & port are `localhost:9357`
- Methods that do not return a data object such as `GameServer` will return a gRPC `Status` object. To check the state of the request, check `Status.StatusCode` & `Status.Detail`.
Ex:
```csharp
if(status.StatusCode == StatusCode.OK)
    //do stuff
```
