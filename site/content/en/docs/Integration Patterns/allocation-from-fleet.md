---
title: "Matchmaker requests a GameServer from a Fleet"
linkTitle: "Allocation from a Fleet"
date: 2021-07-27
weight: 10
description: >
  This is the preferred workflow for a GameServer, in which an external matchmaker requests an allocation from one or 
  more `Fleets` using a `GameServerAllocation`.
---


![Allocated Lifecycle Sequence Diagram](../../../diagrams/gameserver-lifecycle.puml.png)

## Sample `GameServerAllocation`

Since Agones will automatically add the label `agones.dev/fleet` to a `GameServer` of a given `Fleet`, we can use that 
label selector to target a specific `Fleet` by name. In this instance, we are targeting the `Fleet` `xonotic`.

```yaml
apiVersion: "allocation.agones.dev/v1"
kind: GameServerAllocation
spec:
  required:
    matchLabels:
      agones.dev/fleet: xonotic
```

## Next Steps:

- Read the various references, including the
  [GameServer]({{< ref "/docs/Reference/gameserver.md" >}}) and [Fleet]({{< ref "/docs/Reference/fleet.md" >}}) 
  reference materials.
- Review the specifics of [Health Checking]({{< ref "/docs/Guides/health-checking.md" >}}).
- See all the commands the [Client SDK]({{< ref "/docs/Guides/Client SDKs/_index.md" >}}) provides - we only show a 
  few here!
- If you aren't familiar with the term [Pod](https://kubernetes.io/docs/concepts/workloads/pods/pod/), this should
  provide a reference.
