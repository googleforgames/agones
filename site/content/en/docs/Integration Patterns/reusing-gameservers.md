---
title: "Reusing Allocated GameServers for more than one game session"
linkTitle: "Reusing Gameservers"
date: 2021-09-01
weight: 50
description: >
  After a `GameServer` has completed a player session, return it back to the pool of Ready `GameServers` for reuse. 
---

Having a `GameServer` terminate after a single player session is better for packing and optimisation of 
infrastructure usage, as well as safety to ensure the process returns to an absolute zero state.

However, depending on the `GameServer` startup time, or other factors there may be reasons you wish to reuse a 
`GameServer` for _n_ number of sessions before finally shutting it down.

The "magic trick" to this is knowing that the `GameServer` process can call 
[`SDK.Ready()`]({{% ref "/docs/Guides/Client SDKs/_index.md#ready" %}}) to return to a `Ready` 
state after the `GameServer` has been allocated. 

It is then up to the game developer to ensure that the game server process returns to a zero state once a game 
session has been completed. 

<a href="../../../diagrams/reusing-gameservers.puml.png" target="_blank">
<img src="../../../diagrams/reusing-gameservers.puml.png" alt="Reserved Lifecycle Sequence Diagram" />
</a>

## Next Steps

- Have a look at all commands the [Client SDK]({{< ref "/docs/Guides/Client SDKs/_index.md" >}}) provides.
- If you aren't familiar with the term [Pod](https://kubernetes.io/docs/concepts/workloads/pods/pod/), this shouldw
  provide a reference.
