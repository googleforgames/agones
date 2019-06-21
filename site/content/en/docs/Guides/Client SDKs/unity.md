---
title: "Unity Game Server Client SDK"
linkTitle: "Unity"
date: 2019-06-13T10:17:50Z
weight: 15
description: "This is the Unity version of the Agones Game Server Client SDK."
---

Check the [Client SDK Documentation]({{< relref "_index.md" >}}) for more details on each of the SDK functions and how to run the SDK locally.

## Download

Download the source {{< ghlink href="sdks/unity" >}}directly from Github{{< /ghlink >}}.

## Prerequisites

- Unity >= 2018.x (.NET 4.x)

## Usage

Import this script to your unity project and attach it to GameObject.

To begin working with the SDK, get an instance of it.

```csharp
var agones = agonesGameObject.GetComponent<Agones.AgonesSdk>();
```

To mark the game server as [ready to receive player connections]({{< relref "_index.md#ready" >}}), call the async method `Ready()`.

```csharp
async void SomeMethod()
{
    bool ok = await agones.Ready();
}
```

To mark that the [game session is completed]({{< relref "_index.md#shutdown" >}}) and the game server should be shut down call `Shutdown()`. 

```csharp
bool ok = await agones.Shutdown();
```

Similarly `SetAnnotation(string key, string value)` and `SetLabel(string key, string value)` are async methods that perform an action.

And there is no need to call `Health()`, it is automatically called.

> Note: The following code causes deadlock. Do not use a `Wait` method with the returned Task.
```csharp
void Deadlock()
{
    Task<bool> t = agones.Shutdown();
    t.Wait(); // deadlock!!!
}
```

## Settings

The properties for the Unity Agones SDK can be found in the Inspector.

- Health Interval Second
  - Interval of the server sending a health ping to the Agones sidecar. (default: `5`)
- Health Enabled
  - Whether the server sends a health ping to the Agones sidecar. (default: `true`)
- Log Enabled
  - Debug Logging Enabled. Debug logging for development of this Plugin. (default: `false`)