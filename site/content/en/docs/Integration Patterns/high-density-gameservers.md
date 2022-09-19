---
title: "High Density GameServers"
linkTitle: "High Density GameServers"
date: 2021-08-31
weight: 70
description: >
  How to run multiple concurrent game sessions in a single GameServer process.
---
{{< beta title="Allocation State Filter" gate="StateAllocationFilter" >}}

Depending on the setup and resource requirements of your game server process, sometimes it can be a more economical 
use of resources to run multiple concurrent game sessions from within a single `GameServer` instance.

The tradeoff here is that this requires more management on behalf of the integrated game server process and external 
systems, since it works around the common Kubernetes and/or Agones container lifecycle.

Utilising the new allocation `gameServerState` filter as well as the existing ability to edit the 
`GameServer` labels at both [allocation time]({{% ref "/docs/Reference/gameserverallocation.md" %}}), and from 
within the game server process, [via the SDK][sdk], 
means Agones is able to atomically remove a `GameServer` from the list of potentially allocatable 
`GameServers` at allocation time, and then return it back into the pool of allocatable `GameServers` if and when the 
game server process deems that is has room to host another game session. 

<a href="../../../diagrams/high-density.puml.png" target="_blank">
<img src="../../../diagrams/high-density.puml.png" alt="High Density Allocation Diagram" />
</a>

{{< alert title="Info" color="info">}}
To watch for Allocation events, there is the initial `GameServer.status.state` change from `Ready` to `Allocated`,
but it is also useful to know that the value of `GameServer.metadata.annotations["agones.dev/last-allocated"]` will
change as it is set by Agones with each allocation with the current timestamp, regardless of if there 
is a state change or not.
{{< /alert >}}

## Example `GameServerAllocation`

The below `Allocation` will first attempt to find a `GameServer` from the `Fleet` `simple-udp` that is already 
Allocated and also has the label `agones.dev/sdk-gs-session-ready` with the value of `true`.

The above condition indicates that the matching game server process behind the matched `GameServer` record is able to 
accept another game session at this time.

If an Allocated `GameServer` does not exist with the desired labels, then use the next selector to allocate a Ready 
`GameServer` from the `simple-udp` `Fleet`.

Whichever condition is met, once allocation is made against a `GameServer`, its label of `agones.dev/sdk-gs-session-ready` 
will be set to the value of `false` and it will no longer match the first selector, thereby removing it from any 
future allocations with the below schema.

It will then be up to the game server process to decide on if and when it is appropriate to set the 
`agones.dev/sdk-gs-session-ready` value back to `true`, thereby indicating that it can accept another concurrent 
gameplay session.

```yaml
apiVersion: "allocation.agones.dev/v1"
kind: GameServerAllocation
spec:
  selectors:
    - matchLabels:
        agones.dev/fleet: simple-udp
        agones.dev/sdk-gs-session-ready: "true" # this is important
      gameServerState: Allocated # new state filter: allocate from Allocated servers
    - matchLabels:
        agones.dev/fleet: simple-udp
      gameServerState: Ready # Allocate out of the Ready Pool (which would be default, so backward compatible)
  metadata:
    labels:
      agones.dev/sdk-gs-session-ready: "false" # this removes it from the pool
```

{{< alert title="Info" color="info">}}
It's important to note that the labels that the `GameServer` process use to add itself back into the pool of 
allocatable instances, must start with the prefix `agones.dev/sdk-`, since only labels that have this prefix are 
available to be [updated from the SDK][sdk].
{{< /alert >}}

## Consistency

Agones, and Kubernetes itself are built as eventually consistent, self-healing systems. To that end, it is worth 
noting that there may be minor delays between each of the operations in the above flow.  For example, depending on the 
cluster load, it may take up to a second for an [SDK driven label change][sdk] on a `GameServer` record to be 
visible to the Agones allocation system. We recommend building your integrations with Agones with this in mind.

## Next Steps

* View the details about [using the SDK][sdk] to set 
  labels on the `GameServer`.
* Check all the options available on [`GameServerAllocation`]({{% ref "/docs/Reference/gameserverallocation.md" %}}).

[sdk]: {{% ref "/docs/Guides/Client SDKs/_index.md#setlabelkey-value" %}}