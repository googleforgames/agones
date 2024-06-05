---
title: "Allocating based on GameServer Player Capacity"
linkTitle: "Player Capacity"
date: 2021-08-31
weight: 90
description: >
  Find a `GameServer` that has room for a specific number of players.
---

{{% pageinfo color="info" %}}
[Counters and Lists]({{< ref "/docs/Guides/counters-and-lists.md" >}}) will eventually replace the Beta functionality
of Player Tracking, which will subsequently be removed from Agones.

If you are currently using this Beta feature, we would love for you to test (and ideally migrate to!) this new
functionality to Counters and Lists to ensure it meet all your needs.

This document will be updated to utilise Counters and Lists in the near future.
{{% /pageinfo %}}

{{< alpha
title="Player Tracking and Allocation Player Filter"
gate="PlayerTracking,PlayerAllocationFilter" >}}

Using this approach, we are able to be able to make a request that is akin to: "Find me a `GameServer` that is already
allocated, with room for _n_ number of players, and if one is not available, allocate me a `Ready` `GameServer`".

Common applications of this type of allocation are Lobby servers where players await matchmaking, or a 
persistent world server where players connect and disconnect from a large map.

<a href="../../../diagrams/allocation-player-capacity.puml.png" target="_blank">
<img src="../../../diagrams/allocation-player-capacity.puml.png" alt="Player Capacity Allocation Diagram" />
</a>

## Example `GameServerAllocation`

The below allocation will attempt to find an already Allocated `GameServer` from the `Fleet` "lobby" with room for 10 
to 15 players, and if it cannot find one, will allocate a Ready one from the same `Fleet`. 

```yaml
apiVersion: "allocation.agones.dev/v1"
kind: GameServerAllocation
spec:
  selectors:
    - matchLabels:
        agones.dev/fleet: lobby
      gameServerState: Allocated
      players:
        minAvailable: 10
        maxAvailable: 15
    - matchLabels:
        agones.dev/fleet: lobby
```

{{< alert title="Note" color="info">}}
We recommend doing an extra check when players connect to a `GameServer` that there is the expected player capacity
on the `GameServer` as there can be a small delay between a player connecting and it being reported
to Agones.
{{< /alert >}}

## Next Steps

- Have a look at all commands the [Client SDK]({{< ref "/docs/Guides/Client SDKs/_index.md" >}}) provides.
- Check all the options available on [`GameServerAllocation`]({{% ref "/docs/Reference/gameserverallocation.md" %}}).
- If you aren't familiar with the term [Pod](https://kubernetes.io/docs/concepts/workloads/pods/pod/), this should
  provide a reference.
