---
title: "GameServer Creation, Allocation and Shutdown Lifecycle"
linkTitle: "GameServer Lifecycle"
date: 2019-02-01T02:15:18Z
weight: 15
description: >
  Common patterns and lifecycles of `GameServer` creation and integration with the SDK,
  when being started and match made.
---

## Matchmaker requests a GameServer from a Fleet

This is the preferred workflow for a GameServer, in which an external matchmaker requests an allocation from one or more
`Fleets` using a `GameServerAllocation`:

![Allocated Lifecycle Sequence Diagram](../../../diagrams/gameserver-lifecycle.puml.png)

## Matchmaker requires game server process registration

Scenarios in which a Matchmaker requires a game server process to register themselves with the matchmaker, and the
matchmaker decides which `GameServer` players are sent to, this flow is common:

![Reserved Lifecycle Sequence Diagram](../../../diagrams/gameserver-reserved.puml.png)

{{< alert title="Warning" color="warning">}}
This does relinquish control over how `GameServers` are packed across the cluster to the external matchmaker. It is likely
  it will not do as good a job at packing and scaling as Agones. 
{{< /alert >}}

## Next Steps:

- Read the various references, including the [GameServer]({{< ref "/docs/Reference/gameserver.md" >}}) and [Fleet]({{< ref "/docs/Reference/fleet.md" >}}) reference materials.
- Review the specifics of [Health Checking]({{< relref "health-checking.md" >}}).
- See all the commands the [Client SDK]({{< relref "Client SDKs/_index.md" >}}) provides - we only show a few here!
- If you aren't familiar with the term [Pod](https://kubernetes.io/docs/concepts/workloads/pods/pod/), this should provide a reference.