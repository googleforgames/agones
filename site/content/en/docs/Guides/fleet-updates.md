---
title: "Fleet Updates"
date: 2019-08-27T03:58:19Z
weight: 20
description: >
  Common patterns and approaches for updating Fleets with newer and/or different versions of your `GameServer` configuration. 
---

## Rolling Update Strategy

When Fleets are edited and updated, the default strategy of Agones is to roll the new version of the `GameServer`
out to the entire `Fleet`, in a step by step increment and decrement by adding a chunk of the new version and removing
a chunk of the current set of `GameServers`. 

This is done while ensuring that `Allocated` `GameServers` are not deleted
until they are specifically shutdown through the game servers SDK, as they are expected to have players on them. 

You can see this in the `Fleet.Spec.Strategy` [reference]({{< ref "/docs/Reference/fleet.md" >}}), with controls for how
much of the `Fleet` is  incremented and decremented at one time:

```yaml
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxSurge: 25%
      maxUnavailable: 25%
```

So when a Fleet is edited, either through `kubectl` `edit`/`apply` or via the Kubernetes API, this performs the following operations:

1. Adds the `maxSurge` number of `GameServers` to the Fleet.
1. Shutdown the `maxUnavailable` number of `GameServers` in the Fleet, skipping `Allocated` `GameServers`.
1. Repeat above steps until all the previous `GameServer` configurations have been `Shutdown` and deleted.

## Recreate Strategy

This is an optimal `Fleet` update strategy if you want to replace all `GameServers` that are not `Allocated`
with a new version as quickly as possible.

You can see this in the `Fleet.Spec.Strategy` [reference]({{< ref "/docs/Reference/fleet.md" >}}):

```yaml
  strategy:
    type: Recreate
```

So when a Fleet is edited, either through `kubectl` `edit`/`apply` or via the Kubernetes API, this performs the following operations:

1. `Shutdown` all `GameServers` in the Fleet that are not currently `Allocated`.
1. Create the same number of the new version of the `GameServers` that were previously deleted.
1. Repeat above steps until all the previous `GameServer` configurations have been `Shutdown` and deleted.

## Two (or more) Fleets Strategy

If you want very fine-grained control over the rate that new versions of a `GameServer` configuration is rolled out, or 
if you want to do some version of A/B testing or smoke test between different versions, running two (or more) `Fleets` at the same time is a
good solution for this. 

To do this, create a second `Fleet` inside your cluster, starting with zero replicas. From there you can scale this newer `Fleet`
up and the older `Fleet` down as required by your specific rollout strategy.

This also allows you to rollback if issues arise with the newer version, as you can delete the newer `Fleet`
and scale up the old Fleet to its previous levels, resulting in minimal impact to the players. 

> For GameServerAllocation, you will need to have at least a single shared label between the `GameServers` in each
> Fleet.

### GameServerAllocation Across Fleets

Since `GameServerAllocation` is powered by label selectors, it is possible to allocate across multiple fleets, and/or
give preference to particular sets of `GameServers` over others. You can see details of this in 
the `GameServerAllocation` [reference]({{< ref "/docs/Reference/gameserverallocation.md" >}}).

In a scenario where a new `v2` version of a `Fleet` is being slowly scaled up in a separate Fleet from the previous `v1`
Fleet, we can specify that we `prefer` allocation to occur from the `v2` fleet, and if none are available, fallback to
the `v1` fleet, like so:

```yaml
apiVersion: "allocation.agones.dev/v1"
kind: GameServerAllocation
spec:
  required:
    matchLabels:
      game: my-awesome-game
  preferred:
    - matchLabels:
        agones.dev/fleet: v2
```

In this example, all `GameServers` have the label `game: my-awesome-game`, so the Allocation will search across both
Fleets through that mechanism. The `preferred` label matching selector tells the allocation system to first search
all `GameServers` with the `v2` `Fleet` label, and if not found, search through the rest of the set.

The above `GameServerAllocation` can then be used while you scale up the `v2` Fleet and scale down the original Fleet at
the rate that you deem fit for your specific rollout. 