---
title: "Google Kubernetes Engine Best Practices"
linkTitle: "Google Cloud"
date: 2023-05-12T00:00:00Z
description: "Best practices for running Agones on Google Kubernetes Engine (GKE)."
---

## Overview

On this page, we've collected several [Google Kubernetes Engine (GKE)](https://cloud.google.com/kubernetes-engine/) best practices.

## Release Channels

### Why?

We recommend using [Release Channels](https://cloud.google.com/kubernetes-engine/docs/concepts/release-channels) for all GKE clusters. Using Release Channels has several advantages:
* Google automatically manages the version and upgrade cadence for your Kubernetes Control Plane and its nodes.
* Clusters on a Release Channel are allowed to use the `No minor upgrades` and `No minor or node upgrades` [scope of maintenance exclusions](https://cloud.google.com/kubernetes-engine/docs/concepts/maintenance-windows-and-exclusions#limitations-maint-exclusions) - in other words, enrolling a cluster in a Release Channel gives you _more control_ over node upgrades.
* Clusters enrolled in `rapid` channel have access to the newest Kubernetes version first. Agones strives to [support the newest release in `rapid` channel]({{< relref "Installation#agones-and-kubernetes-supported-versions" >}}) to allow you to test the newest Kubernetes soon after it's available in GKE.

{{< alert title="Note" color="info" >}}
GKE Autopilot clusters must be on Release Channels.
{{< /alert >}}

### What channel should I use?

We recommend the `regular` channel, which offers a balance between stability and freshness. See [this guide](https://cloud.google.com/kubernetes-engine/docs/concepts/release-channels#what_channel_should_i_use) for more discussion.

If you need to disallow minor version upgrades for more than 6 months, consider choosing the freshest Kubernetes version possible: Choosing the freshest version on `rapid` or `regular` will extend the amount of time before your cluster reaches [end of life](https://cloud.google.com/kubernetes-engine/docs/release-schedule#schedule-for-release-channels).

### What versions are available on a given channel?

You can query the versions available across different channels using `gcloud`:

```
gcloud container get-server-config \
  --region=[COMPUTE_REGION] \
  --flatten="channels" \
  --format="yaml(channels)"
```
Replace the following:

* **COMPUTE_REGION**: the
[Google Cloud region](https://cloud.google.com/compute/docs/regions-zones#available)
where you will create the cluster.

## Managing Game Server Disruption on GKE

If your game session length is less than an hour, use the `eviction` API to configure your game servers appropriately - see [Controlling Disruption]({{< relref "controlling-disruption" >}}).

For sessions longer than an hour, there are currently two possible approaches to manage disruption:

* (GKE Standard/Autopilot) [Blue/green deployment](https://martinfowler.com/bliki/BlueGreenDeployment.html) at the cluster level: If you are using an automated deployment process, you can:
  * create a new, `green` cluster within a release channel e.g. every week,
  * use [maintenance exclusions](https://cloud.google.com/kubernetes-engine/docs/concepts/maintenance-windows-and-exclusions#exclusions) to prevent node upgrades for 30d, and
  * scale the `Fleet` on the old, `blue` cluster down to 0, and
  * use [multi-cluster allocation]({{< relref "multi-cluster-allocation.md" >}}) on Agones, which will then direct new allocations to the new `green` cluster (since `blue` has 0 desired), then
  * delete the old, `blue` cluster when the `Fleet` successfully scales down.

* (GKE Standard only) Use [node pool blue/green upgrades](https://cloud.google.com/kubernetes-engine/docs/concepts/node-pool-upgrade-strategies#blue-green-upgrade-strategy)
