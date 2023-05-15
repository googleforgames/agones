---
title: "Cloud Best Practices" 
linkTitle: "Cloud Best Practices"
date: 2023-05-12T00:00:00Z
weight: 9
description: "Best practices for running Agones in production."
---

## Overview

Running Agones in production takes consideration, from planning your launch to figuring
out the best course of action for cluster and Agones upgrades. On this page, we've collected
some general best practices. We also have cloud specific pages for:

* [Google Kubernetes Engine (GKE)]({{< relref "gke.md" >}})

If you are interested in submitting best practices for your cloud prodiver / on-prem, [please contribute!]({{< relref "/Contribute" >}})

## Separate game servers from other workloads

We recommend using [taints and tolerations](https://kubernetes.io/docs/concepts/scheduling-eviction/taint-and-toleration/) to ensure that your game servers are running separate from Agones controllers. There are many ways to accomplish this, here we discuss one:

* Label a group of nodes `agones.dev/agones-system=true`
* Taint the same group of nodes with `agones.dev/agones-system=true:NoExecute`

If you are collecting [Metrics]({{< relref "metrics" >}}) using our standard Prometheus installation, consider isolating the `agones-metrics` namespace as well.

See [Creating a Cluster]({{< relref "Creating Cluster" >}}) for initial set up on your cloud provider.

## Redundant Clusters

### Allocate Across Clusters

Agones supports [Multi-cluster Allocation]({{< relref "multi-cluster-allocation" >}}), allowing you to allocate from a set of clusters, versus a single point of potential failure. There are several other options for multi-cluster allocation:
* [Anthos Service Mesh](https://cloud.google.com/anthos/service-mesh) can be used to route allocation traffic to different clusters based on arbitrary criteria. See [Global Multiplayer Demo](https://github.com/googleforgames/global-multiplayer-demo) for an example where the match maker influences which cluster the allocation is routed to.
* [Allocation Endpoint](https://github.com/googleforgames/agones/tree/main/examples/allocation-endpoint) can be used in Cloud Run to proxy allocation requests.
* Or peruse the [Third Party Examples]({{< relref "../../Third Party Content/libraries-tools.md/#allocation" >}})

### Spread

You should consider spreading your game servers in two ways:
* **Across geographic fault domains** ([GCP regions](https://cloud.google.com/compute/docs/regions-zones), [AWS availability zones](https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/using-regions-availability-zones.html), separate datacenters, etc.): This is desirable for geographic fault isolation, but also for optimizing client latency to the game server.
* **Within a fault domain**: Kubernetes Clusters are single points of failure. A single misconfigured RBAC rule, an overloaded Kubernetes Control Plane, etc. can prevent new game server allocations, or worse, disrupt existing sessions. Running multiple clusters within a fault domain also allows for [easier upgrades]({{< relref "Upgrading#upgrading-agones-multiple-clusters" >}}).
