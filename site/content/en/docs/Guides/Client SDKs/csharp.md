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

## Install using NuGet

- Download the nuget package [directly](https://www.nuget.org/packages/AgonesSDK/)
- Install the latest version using the Package Manager: `Install-Package AgonesSDK`
- Install the latest version using the .NET CLI: `dotnet add package AgonesSDK`

To select a specific version, append `--version`, for example: `--version 1.8.0` to either commands.

## Prerequisites

- .Net Standard 2.0 compliant framework.

## Usage

Reference the SDK in your project & create a new instance of the SDK wrapper:

### Initialization

To use the AgonesSDK, you will need to import the namespace by adding `using Agones;` at the beginning of your relevant files.

```csharp
var agones = new AgonesSDK();
```

### Connection

To connect to the SDK server, either locally or when running on Agones, run the `ConnectAsync()` method.
This will wait for up to 30 seconds if the SDK server has not yet started and the connection cannot be made,
and will return `false` if there was an issue connecting.

```csharp
bool ok = await agones.ConnectAsync();
```

### Ready

To mark the game server as [ready to receive player connections]({{< relref "_index.md#ready" >}}), call the async method `ReadyAsync()`.

```csharp
async void SomeMethod()
{
    var status = await agones.ReadyAsync();
}
```

### Health

To send `Health` pings, call the async method `HealthAsync()`
```csharp
await agones.HealthAsync();
```

### GetGameServer

To get the details on the [backing `GameServer`]({{< relref "_index.md#gameserver" >}}) call `GetGameServerAsync()`.

Will return `null` if there is an error in retrieving the `GameServer` record.

```csharp
var gameserver = await agones.GetGameServerAsync();
```

### Reserve

To mark the GameServer as [Reserved]({{< relref "_index.md#reserveseconds" >}}) for a duration call 
`ReserveAsync(long duration)`.

```csharp
long duration = 30;
var status = await agones.ReserveAsync(duration);
```

### ShutDown

To mark that the [game session is completed]({{< relref "_index.md#shutdown" >}}) and the game server should be shut down call `ShutdownAsync()`.

```csharp
var status = await agones.ShutdownAsync();
```

### SetAnnotation &  SetLabel
Similarly `SetAnnotation(string key, string value)` and `SetLabel(string key, string value)` are async methods that perform an action & return a `Status` object.

### WatchGameServer

To watch when 
[the backing `GameServer` configuration changes]({{< relref "_index.md#watchgameserverfunctiongameserver" >}})
call `WatchGameServer(callback)`, where the delegate function `callback` of type `Action<GameServer>` will be executed every time the `GameServer` 
configuration changes.
This process is non-blocking internally.

```csharp
agonesSDK.WatchGameServer((gameServer) => { Console.WriteLine($"Server - Watch {gameServer}");});
```

## Player Tracking

{{< alpha title="Player Tracking" gate="PlayerTracking" >}}

### Alpha: PlayerConnect

This method increases the SDK’s stored player count by one, and appends this playerID to GameServer.Status.Players.IDs.
Returns true and adds the playerID to the list of playerIDs if the playerIDs was not already in the list of connected playerIDs.

```csharp
bool ok = await agones.Alpha().PlayerConnectAsync(playerId);
```

### Alpha: PlayerDisconnect

This function decreases the SDK’s stored player count by one, and removes the playerID from GameServer.Status.Players.IDs.
Will return true and remove the supplied playerID from the list of connected playerIDs if the playerID value exists within the list.

```csharp
bool ok = await agones.Alpha().PlayerDisconnectAsync(playerId);
```

### Alpha: SetPlayerCapacity

Update the `GameServer.Status.Players.Capacity` value with a new capacity.

```csharp
var capacity = 100;
var status = await agones.Alpha().SetPlayerCapacityAsync(capacity);
```

### Alpha: GetPlayerCapacity

This function retrieves the current player capacity `GameServer.Status.Players.Capacity`. 
This is always accurate from what has been set through this SDK, even if the value has yet to be updated on the GameServer status resource.

```csharp
long cap = await agones.Alpha().GetPlayerCapacityAsync();
```

### Alpha: GetPlayerCount

Returns the current player count

```csharp
long count = await agones.Alpha().GetPlayerCountAsync();
```

### Alpha: IsPlayerConnected

This returns if the playerID is currently connected to the GameServer.
This is always accurate, even if the value hasn’t been updated to the GameServer status yet.

```csharp
var playerId = "player1";
bool isConnected = await agones.Alpha().IsPlayerConnectedAsync(playerId);
```

## Remarks
- All requests other than `ConnectAsync` will wait for up to 15 seconds before giving up, time to wait can also be set in the constructor.
- Default host & port are `localhost:9357`
- Methods that do not return a data object such as `GameServer` will return a gRPC `Grpc.Core.Status` object. To check the state of the request, check `Status.StatusCode` & `Status.Detail`.
Ex:
```csharp
if(status.StatusCode == StatusCode.OK)
    //do stuff
```
