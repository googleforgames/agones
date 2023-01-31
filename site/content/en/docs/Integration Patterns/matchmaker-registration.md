---
title: "Matchmaker requires game server process registration"
linkTitle: "Matchmaker registration"
date: 2021-07-27
weight: 20
description: >
    A scenario in which a Matchmaker requires a game server process to register themselves with the matchmaker, and the
    matchmaker decides which `GameServer` players are sent to.
---

In this scenario, the `GameServer` process will need to self Allocate when informed by the matchmaker that players 
are being sent to them.

![Reserved Lifecycle Sequence Diagram](../../../diagrams/gameserver-reserved.puml.png)

{{< alert title="Warning" color="warning">}}
This does relinquish control over how `GameServers` are packed across the cluster to the external matchmaker. It is likely
it will not do as good a job at packing and scaling as Agones.
{{< /alert >}}

## Next Steps:

- Read the various references, including the
  [GameServer]({{< ref "/docs/Reference/gameserver.md" >}}) and [Fleet]({{< ref "/docs/Reference/fleet.md" >}})
  reference materials.
- Review the specifics of [Health Checking]({{< ref "/docs/Guides/health-checking.md" >}}).
- See all the commands the [Client SDK]({{< ref "/docs/Guides/Client SDKs/_index.md" >}}) provides - we only show a
  few here!
- If you aren't familiar with the term [Pod](https://kubernetes.io/docs/concepts/workloads/pods/pod/), this should
  provide a reference.
