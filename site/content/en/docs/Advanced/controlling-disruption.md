---
title: "Controlling Disruption"
date: 2023-01-24T20:15:26Z
weight: 20
description: >
  Game servers running on Agones may be disrupted by Kubernetes; learn how to control disruption of your game servers.
---

## Disruption in Kubernetes

[A `Pod` in Kubernetes may be disrupted](https://kubernetes.io/docs/concepts/workloads/pods/disruptions/#voluntary-and-involuntary-disruptions) for involuntary reasons, e.g. hardware failure, or voluntary reasons, such as when nodes are drained for upgrades. 

By default, Agones assumes your game server should never be disrupted and configures the `Pod` appropriately - but this isn't always the ideal setting. Here we discuss how Agones allows you to control the two most significant sources of voluntary `Pod` disruptions (`Pod` evictions), node upgrades and Cluster Autoscaler, using the `eviction` API on the `GameServer` object. 

{{< alpha title="`eviction` API" gate="SafeToEvict" >}}

## Considerations

When discussing game server pod disruption, it's important to keep two factors in mind:

* **Session Length:** What is your game servers session length (the time from when the `GameServer` is allocated to when its shutdown)? In general, we bucket the session lengths into "less than 10 minutes", "10 minutes to an hour", and "greater than an hour". (See [below](#whats-special-about-ten-minutes-and-one-hour) if you are curious about session length considerations.)
* **Disruption Tolerance:** Is your game server tolerant of disruption in a timely fashion? If your game server can handle the `TERM` signal and has [terminationGracePeriodSeconds](https://kubernetes.io/docs/concepts/containers/container-lifecycle-hooks/#hook-handler-execution) configured for a period less than 10 minutes, meaning that the game server can either complete or checkpoint state within 10 minutes, we consider the game server tolerant of disruption in a timely fashion.

## Agones `eviction` API

Agones offers a simple way to control disruption of your game servers that should be applicable to most cloud products with default configurations: [`GameServerSpec.eviction`]({{< ref "/docs/Reference/agones_crd_api_reference.html#agones.dev/v1.GameServerSpec" >}})

### `safe: Never` (default)

By default Agones assumes your game server needs to run indefinitely and tries to prevent `Pod` disruption. This is equivalent to, and can be explicitly set with:

```
eviction:
  safe: Never
```

Note: If your session length is greater than an hour, Agones may not be able to protect your game server from disruption. See [below](#considerations-for-long-sessions).

### `safe: Always`

If your game server is tolerant of disruption in a timely fashion, you should set:

```
eviction:
  safe: Always
```

You further need to configure [terminationGracePeriodSeconds](https://kubernetes.io/docs/concepts/containers/container-lifecycle-hooks/#hook-handler-execution) as appropriate for your game server.

Note that to maintain backward compatibility with Agones prior to the introduction of the `SafeToEvict` feature gate, if your game server previously configured the `cluster-autoscaler.kubernetes.io/safe-to-evict: true` annotation, we assume `eviction.safe: Always` is intended.

### `safe: OnUpgrade`

If your game server has a session length between ten minutes and one hour, you can set:

```
eviction:
  safe: OnUpgrade
```

This will ensure the game server `Pod` can be evicted by node upgrades, but Cluster Autoscaler will not evict it.

If you do not wish your game server to be disrupted by upgrades, you can either set `Never`, or, with a 10m-1h session length, set [terminationGracePeriodSeconds](https://kubernetes.io/docs/concepts/containers/container-lifecycle-hooks/#hook-handler-execution) to one hour (or your session length), then intercept `SIGTERM` but do nothing with it. Either approach will allow your game server to run to completion.

{{< alert title="GKE Autopilot" color="info" >}}
GKE Autopilot supports only `Never` and `Always`, not `UpgradeOnly`.
{{< /alert >}}

## What's special about ten minutes and one hour?

* **Ten minutes:** Cluster Autoscaler respects [ten minutes of graceful termination](https://github.com/kubernetes/autoscaler/blob/master/cluster-autoscaler/FAQ.md#does-ca-respect-gracefultermination-in-scale-down) on scale-down. On some cloud products, you can configure `--max-graceful-termination-sec` to change this, but it is not advised: Cluster Autoscaler is currently only capable of scaling down one node at a time, and larger graceful termination windows slow this down farther (see [autoscaler#5079](https://github.com/kubernetes/autoscaler/issues/5079)). If the ten minute limit does not apply to you, generally you should choose between `safe: Always` (for sessions less than an hour), or see [below](#considerations-for-long-sessions).

* **One hour:** On many cloud products, `PodDisruptionBudget` can only block node upgrade evictions for a certain period of time - on GKE this is 1h. After that, the PDB is ignored, or the node upgrade fails with an error. Controlling `Pod` disruption for longer than one hour requires cluster configuration changes outside of Agones - see [below](#considerations-for-long-sessions).

## Considerations for long sessions

Outside of Cluster Autoscaler, the main source of disruption for long sessions is node upgrade. On some cloud products, such as GKE Standard, node upgrades are entirely within your control. On others, such as GKE Autopilot, node upgrade is automatic. Typical node upgrades use an eviction based, rolling recreate strategy, and may not honor `PodDisruptionBudget` for longer than an hour. Here we document strategies you can use for your cloud product to support long sessions.

### On GKE

On GKE, there are currently two possible approaches to manage disruption for session lengths longer than an hour:

* (GKE Standard/Autopilot) [Blue/green deployment](https://martinfowler.com/bliki/BlueGreenDeployment.html) at the cluster level: If you are using an automated deployment process, you can:
  * create a new, `green` cluster within a release channel e.g. every week,
  * use [maintenance exclusions](https://cloud.google.com/kubernetes-engine/docs/concepts/maintenance-windows-and-exclusions#exclusions) to prevent node upgrades for 30d, and
  * scale the `Fleet` on the old, `blue` cluster down to 0, and
  * use [multi-cluster allocation]({{< relref "multi-cluster-allocation.md" >}}) on Agones, which will then direct new allocations to the new `green` cluster (since `blue` has 0 desired), then
  * delete the old, `blue` cluster when the `Fleet` successfully scales down.

* (GKE Standard only) Use [node pool blue/green upgrades](https://cloud.google.com/kubernetes-engine/docs/concepts/node-pool-upgrade-strategies#blue-green-upgrade-strategy)

### Other cloud products

The blue/green cluster strategy described for GKE is likely applicable to your cloud product.

We welcome contributions to this section for other products!

## Implementation / Under the hood

Each option uses a slightly different permutation of:
* the `safe-to-evict` annotation to block [Cluster Autoscaler based eviction](https://github.com/kubernetes/autoscaler/blob/master/cluster-autoscaler/FAQ.md#what-types-of-pods-can-prevent-ca-from-removing-a-node)
* and the `agones.dev/safe-to-evict` label selector to select the `agones-gameserver-safe-to-evict-false` `PodDisruptionBudget`. This blocks [Cluster Autoscaler](https://github.com/kubernetes/autoscaler/blob/master/cluster-autoscaler/FAQ.md#what-types-of-pods-can-prevent-ca-from-removing-a-node) and (for a limited time) [disruption from node upgrades](https://kubernetes.io/docs/concepts/workloads/pods/disruptions/#pod-disruption-budgets).
  * Note that PDBs do influence pod preemption as well, but it's not guaranteed.

As a quick reference:

| evictions.safe setting  |  `safe-to-evict` pod annotation |  `agones.dev/safe-to-evict` label |
|-------------------------|---------------------------------|-----------------------------------|
| `Never` (default)       | `false`                         | `false` (matches PDB)             |
| `OnUpdate`              | `false`                         | `true` (does not match PDB)       |
| `Always`                | `true`                          | `true` (does not match PDB)       |

## Further Reading

* [`eviction` design](https://github.com/googleforgames/agones/issues/2794)