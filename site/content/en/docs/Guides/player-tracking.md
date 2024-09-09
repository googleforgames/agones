---
title: "Player Tracking"
linkTitle: "Player Tracking"
date: 2020-05-19
weight: 30
description: >
  Track player connections, disconnections, counts and capacities through the Agones SDK
---

{{% pageinfo color="info" %}}
[Counters and Lists]({{< ref "/docs/Guides/counters-and-lists.md" >}}) replaces the Alpha functionality of Player
Tracking, and Player Tracking will soon be removed from Agones.

If you are currently using this Alpha feature, please plan on migrating to use Counters and Lists to ensure it meets
all your needs.
{{% /pageinfo %}}

{{< alpha title="Player Tracking" gate="PlayerTracking" >}}

## Managing GameServer Capacities

To track your `GameServer` current player capacity, Agones gives you the ability to both set an initial capacity at
`GameServer` creation, as well be able to change it during the lifecycle of the `GameServer` through the Agones SDK.

To set the initial capacity, you can do so via `GameServer.Spec.Players.InitialCapacity` like so:

```yaml
apiVersion: "agones.dev/v1"
kind: GameServer
metadata:
  name: "gs-example"
spec:
  # ...
  players:
    # set this GameServer's initial player capacity to 10
    initialCapacity: 10
```

From there, if you need to change the capacity of the GameSever as gameplay is in progress, you can also do so via 
[`SDK.Alpha().SetPlayerCapacity(count)`]({{< ref "/docs/Guides/Client SDKs/_index.md#alphasetplayercapacitycount" >}}) 

The current player capacity is represented in `GameServer.Status.Players.Capacity` resource value.

We can see this in action, when we look at the Status section of a GameServer resource
, wherein the capacity has been set to 20:

```
...
Status:
  Address:    14.81.195.72
  Node Name:  gke-test-cluster-default-6cd0ba67-1mps
  Players:
    Capacity:  20
    Count:     0
    Ids:       <nil>
  Ports:
    Name:          gameport
    Port:          7983
  Reserved Until:  <nil>
  State:           Ready
```

From the SDK, the game server binary can also retrieve the current player capacity 
via [`SDK.Alpha().GetPlayerCapacity()`]({{< ref "/docs/Guides/Client SDKs/_index.md#alphagetplayercapacity" >}}).

{{% alert title="Note" color="info" %}}
Changing the capacity value here has no impact on players actually
connected to or trying to connect to your server, as that is not a responsibility of Agones.

This functionality is for tracking purposes only. 
{{% /alert %}}

## Connecting and Disconnecting Players

As players connect and disconnect from your game, the Player Tracking functions enable you to track which players 
are currently connected.

It assumed that each player that connects has a unique token that identifies them as a player.

When a player connects to the game server binary, 
calling [`SDK.Alpha().PlayerConnect(playerID)`]({{< ref "/docs/Guides/Client SDKs/_index.md#alphaplayerconnectplayerid" >}})
with the unique player token will register them as connected, and store their player id.
 
At disconnection time,
call [`SDK.Alpha().PlayerDisconnect(playerID)`]({{< ref "/docs/Guides/Client SDKs/_index.md#alphaplayerdisconnectplayerid" >}})
, which will deregister them and remove their player id from the list.

Each of these `playerIDs` is stored on `GameServer.Status.Players.IDs`, and the current count of connected players
can be seen in `GameServer.Status.Players.Count`. 

You can see this in action below in the `GameServer` Status section, where there are 4 players connected:

```
...
Status:
  Address:    39.82.196.74
  Node Name:  gke-test-cluster-default-6cd0ba77-1mps
  Players:
    Capacity:  10
    Count:     4
    Ids:
      xy8a
      m0ux
      71nj
      lpq5
  Ports:
    Name:          gameport
    Port:          7166
  Reserved Until:  <nil>
  State:           Ready
```

{{% alert title="Note" color="info" %}}
Calling `PlayerConnect` or `PlayerDisconnect` functions will not
connect or disconnect players, as that is not under the control of Agones.

This functionality is for tracking purposes only. 
{{% /alert %}}

## Checking Player Data

Not only is the connected player data stored on the `GameServer` resource, it is also stored in memory within the
SDK, so that it can be used from within the game server binary as a realtime, thread safe, registry of connected
players.

Therefore, if you want to:

* Get the current player count, call [`SDK.Alpha().GetPlayerCount()`]({{< ref "/docs/Guides/Client SDKs/_index.md#alphagetplayercount" >}})
* Check if a specific player is connected, call [`SDK.Alpha().IsPlayerConnected(playerID)`]({{< ref "/docs/Guides/Client SDKs/_index.md#alphaisplayerconnectedplayerid" >}})
* Retrieve the full list of connected players, call [`SDK.Alpha().GetConnectedPlayers()`]({{< ref "/docs/Guides/Client SDKs/_index.md#alphagetconnectedplayers" >}})

## Next Steps

* Review the [Player Tracking SDK Reference]({{< ref "/docs/Guides/Client SDKs/_index.md#player-tracking" >}})
