---
title: "Godot Game Server Client SDK"
linkTitle: "Godot"
date: 2021-04-27T10:17:50Z
weight: 15
description: "This is the Godot version of the Agones Game Server Client SDK."
---

Check the [Client SDK Documentation]({{< relref "_index.md" >}}) for more details on each of the SDK functions and how to run the SDK locally.

## Download

Download the source {{< ghlink href="sdks/godot" >}}directly from GitHub{{< /ghlink >}}.

## Prerequisites

- Godot >= 3.x

## Usage

Copy the `addons` directory into your Godot project then go to your Project Settings and enable the `Agones SDK` plugin.

To begin working with the SDK, get an instance of it.

```gdscript
var agones : AgonesSdk

func _ready():
    agones = preload("res://addons/com.google.agones/AgonesSdk.gd").new()
    add_child(agones)
```

To mark the game server as [ready to receive player connections]({{< relref "_index.md#ready" >}}), call the async method `Ready()`.

```gdscript
func SomeMethod():
    var result = yield(agones.Ready(), "completed")
```

To get the details on the [backing `GameServer`]({{< relref "_index.md#gameserver" >}}) call `GetGameServer()`.

Will return an error object if there is an error in retrieving the `GameServer` record.

```gdscript
var gameserver = yield(agones.GetGameServer(), "completed")
```

To mark the GameServer as [Reserved]({{< relref "_index.md#reserveseconds" >}}) for a duration call 
`Reserve(TimeSpan duration)`.

```gdscript
var result = yield(agones.Reserve(duration), "completed")
```

To mark that the [game session is completed]({{< relref "_index.md#shutdown" >}}) and the game server should be shut down call `Shutdown()`.

```gdscript
var result = yield(agones.Shutdown(), "completed")
```

Methods like `SetAnnotation("{\"key\": \"foo\", \"value\": \"bar\"}")` and `SetLabel("{\"key\": \"foo\", \"value\": \"bar\"}")` are async methods that require a JSON string.

There is a `on_request(path, params, method)` signal on each client object that gives. This can be used for debugging or logging purposes.

```gdscript
func _ready():
    agones.connect("on_request", self, "log_request")

func log_request(path, params, method):
    print(path)
```

If the request results in an error, the SDK will return an AgonesError object.

```gdscript
    var result = yield(agones.Ready(), "completed")
    if result is AgonesError:
        print(result.message)
        return
```
