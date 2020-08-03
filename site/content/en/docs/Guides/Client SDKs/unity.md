---
title: "Unity Game Server Client SDK"
linkTitle: "Unity"
date: 2019-06-13T10:17:50Z
weight: 15
description: "This is the Unity version of the Agones Game Server Client SDK."
---

{{< alert title="Note" color="info" >}}
The Unity SDK is not feature complete in 1.2.0, but will be feature complete with the 1.3.0 release.
{{< /alert >}}

Check the [Client SDK Documentation]({{< relref "_index.md" >}}) for more details on each of the SDK functions and how to run the SDK locally.

## Download

Download the source {{< ghlink href="sdks/unity" >}}directly from GitHub{{< /ghlink >}}.

## Prerequisites

- Unity >= 2018.x (.NET 4.x)

## Usage

Import this script to your unity project and attach it to GameObject.

To begin working with the SDK, get an instance of it.

```csharp
var agones = agonesGameObject.GetComponent<Agones.AgonesSdk>();
```

To connect to the SDK server, either local or when running on Agones, run the async `Connect()` method.
This will wait for up to 30 seconds if the SDK server has not yet started and the connection cannot be made,
and will return `false` if there was an issue connecting.

```csharp
bool ok = await agones.Connect();
```

To mark the game server as [ready to receive player connections]({{< relref "_index.md#ready" >}}), call the async method `Ready()`.

```csharp
async void SomeMethod()
{
    bool ok = await agones.Ready();
}
```

To get the details on the [backing `GameServer`]({{< relref "_index.md#gameserver" >}}) call `GameServer()`.

Will return `null` if there is an error in retrieving the `GameServer` record.

```csharp
var gameserver = await agones.GameServer();
```

To mark the GameServer as [Reserved]({{< relref "_index.md#reserveseconds" >}}) for a duration call 
`Reserve(TimeSpan duration)`.

```csharp
ok = await agones.Reserve(duration);
```

To mark that the [game session is completed]({{< relref "_index.md#shutdown" >}}) and the game server should be shut down call `Shutdown()`.

```csharp
bool ok = await agones.Shutdown();
```

Similarly `SetAnnotation(string key, string value)` and `SetLabel(string key, string value)` are async methods that perform an action.

And there is no need to call `Health()`, it is automatically called.

To watch when 
[the backing `GameServer` configuration changes]({{< relref "_index.md#watchgameserverfunctiongameserver" >}})
call `WatchGameServer(callback)`, where the delegate function `callback` will be executed every time the `GameServer` 
configuration changes.

```csharp
agones.WatchGameServer(gameServer => Debug.Log($"Server - Watch {gameServer}"));
```

{{% alert title="Warning" color="warning"%}}
The following code causes deadlock. Do not use a `Wait` method with the returned Task.
```csharp
void Deadlock()
{
    Task<bool> t = agones.Shutdown();
    t.Wait(); // deadlock!!!
}
```
{{% /alert %}}

## Settings

The properties for the Unity Agones SDK can be found in the Inspector.

- Health Interval Second
  - Interval of the server sending a health ping to the Agones sidecar. (default: `5`)
- Health Enabled
  - Whether the server sends a health ping to the Agones sidecar. (default: `true`)
- Log Enabled
  - Debug Logging Enabled. Debug logging for development of this Plugin. (default: `false`)
