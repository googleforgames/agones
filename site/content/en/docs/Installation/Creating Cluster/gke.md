---
title: "Google Kubernetes Engine"
linkTitle: "Google Cloud"
weight: 10
description: >
    Follow these steps to create a [Google Kubernetes Engine (GKE)](https://cloud.google.com/kubernetes-engine/)
    cluster for your Agones install.
---

## Before you begin

Take the following steps to enable the Kubernetes Engine API:

1. Visit the [Kubernetes Engine][kubernetes] page in the Google Cloud Platform Console.
1. Create or select a project.
1. Wait for the API and related services to be enabled. This can take several minutes.
1. [Enable billing][billing] for your project.
  * If you are not an existing GCP user, you may be able to enroll for a $300 US [Free Trial][trial] credit.

[kubernetes]: https://console.cloud.google.com/kubernetes/list
[billing]: https://support.google.com/cloud/answer/6293499#enable-billing
[trial]: https://cloud.google.com/free/

### Choosing a shell

To complete this quickstart, we can use either [Google Cloud Shell][cloud-shell] or a local shell.

Google Cloud Shell is a shell environment for managing resources hosted on Google Cloud Platform (GCP). Cloud Shell comes preinstalled with the [`gcloud`][gcloud] and [`kubectl`][kubectl] command-line tools. `gcloud` provides the primary command-line interface for GCP, and `kubectl` provides the command-line interface for running commands against Kubernetes clusters.

If you prefer using your local shell, you must install the `gcloud` and `kubectl` command-line tools in your environment.

[cloud-shell]: https://cloud.google.com/shell/
[gcloud]: https://cloud.google.com/sdk/gcloud/
[kubectl]: https://kubernetes.io/docs/reference/kubectl/

#### Cloud shell

To launch Cloud Shell, perform the following steps:

1. Go to [Google Cloud Platform Console][cloud]
1. From the top-right corner of the console, click the
   **Activate Google Cloud Shell** button: ![cloud shell](../../../../images/cloud-shell.png)
1. A Cloud Shell session opens inside a frame at the bottom of the console. Use this shell to run `gcloud` and `kubectl` commands.
1. Set a compute zone in your geographical region with the following command. An example compute zone is `us-west1-a`. A full list can be found at [Available regions and zones][zones].
   ```bash
   gcloud config set compute/zone [COMPUTE_ZONE]
   ```

[cloud]: https://console.cloud.google.com/home/dashboard
[zones]: https://cloud.google.com/compute/docs/regions-zones/#available

#### Local shell

To install `gcloud` and `kubectl`, perform the following steps:

1. [Install the Google Cloud SDK][gcloud-install], which includes the `gcloud` command-line tool.
1. Initialize some default configuration by running the following command.
   * When asked `Do you want to configure a default Compute Region and Zone? (Y/n)?`, enter `Y` and choose a zone in your geographical region of choice.
   ```bash
   gcloud init
   ```
1. Install the `kubectl` command-line tool by running the following command:
   ```bash
   gcloud components install kubectl
   ```

[gcloud-install]: https://cloud.google.com/sdk/docs/quickstarts

### Choosing a GKE cluster mode

A [cluster][cluster] consists of at least one *control plane* machine and multiple worker machines called *nodes*. In Google Kubernetes Engine, nodes are [Compute Engine virtual machine][vms] instances that run the Kubernetes processes necessary to make them part of the cluster.

Agones supports GKE Standard mode and has alpha support for GKE Autopilot mode.
Agones `GameServer` and `Fleet` manifests that work on Standard are compatible
on Autopilot with some constraints, described in the following section. You
can't convert existing Standard clusters to Autopilot; create new Autopilot
clusters instead.

#### Agones on GKE Autopilot

Autopilot is GKE's fully-managed mode. GKE configures, maintains, scales, and
upgrades nodes for you, which can reduce your maintenance and operating
overhead. You only pay for the resources requested by your running Pods, and
you don't pay for unused node capacity or Kubernetes system workloads.

This section describes the Agones-specific considerations in Autopilot
clusters. For a general comparison between Autopilot and Standard, refer to
[Choose a GKE mode of operation](https://cloud.google.com/kubernetes-engine/docs/concepts/choose-cluster-mode).

Autopilot nodes are, by default, optimized for most workloads. If some of your
workloads have broad compute requirements such as Arm architecture or a minimum
CPU platform, you can also choose a
[compute class](https://cloud.google.com/kubernetes-engine/docs/concepts/autopilot-compute-classes)
that meets that requirement. However, if you have specialized hardware needs
that require fine-grained control over machine configuration, consider using
GKE Standard.

Agones on Autopilot has pre-configured opinionated constraints. Evaluate
whether these constraints impact your workloads:

*  **Operating system:** No Windows containers.
*  **Resource requests:** Autopilot has pre-determined
   [minimum Pod resource requests](https://cloud.google.com/kubernetes-engine/docs/concepts/autopilot-resource-requests#min-max-requests).
   If your game servers require less than those minimums, use GKE Standard.
*  **[Scheduling strategy]({{<ref "/docs/Advanced/scheduling-and-autoscaling#fleet-scheduling">}}):**
   `Packed` is supported, which is the Agones default. `Distributed` is not
   supported.
*  **[Host port policy]({{<ref "/docs/reference/agones_crd_api_reference#agones.dev/v1.GameServerPort">}}):** `Dynamic` is supported, which is the Agones default.
   `Static` and `Passthrough` are not supported.
*  **Seccomp profile:** Agones sets the seccomp profile to `Unconfined` to
   avoid unexpected container creation delays that might occur because
   Autopilot enables the
   [`RuntimeDefault` seccomp profile](https://cloud.google.com/kubernetes-engine/docs/concepts/seccomp-in-gke).
*  **[Pod disruption policy]({{<ref "/docs/Advanced/controlling-disruption#eviction-api">}}):**
   `eviction.safe: Never` is supported, which is the Agones
   default. `eviction.safe: Always` is supported. `eviction.safe: OnUpgrade` is
   not supported. If your game sessions exceed one hour, refer to
   [Considerations for long sessions]({{<ref "/docs/Advanced/controlling-disruption#considerations-for-long-sessions">}}).

## Creating the firewall

We need a firewall to allow UDP traffic to nodes tagged as `game-server` via ports 7000-8000. These firewall rules apply to cluster nodes you will create in the
next section.

```bash
gcloud compute firewall-rules create game-server-firewall \
  --allow udp:7000-8000 \
  --target-tags game-server \
  --description "Firewall to allow game server udp traffic"
```

## Creating the cluster

Create a GKE cluster in which you'll install Agones. You can use
[GKE Standard mode](#create-a-standard-mode-cluster-for-agones)
or [GKE Autopilot mode](#create-an-autopilot-mode-cluster-for-agones)
(Alpha).

### Create a Standard mode cluster for Agones

```bash
gcloud container clusters create [CLUSTER_NAME] --cluster-version={{% gke-example-cluster-version %}} \
  --tags=game-server \
  --scopes=gke-default \
  --num-nodes=4 \
  --no-enable-autoupgrade \
  --enable-image-streaming \
  --machine-type=e2-standard-4
```

{{< alert title="Note" color="info">}}
If you're creating a cluster to run Windows game servers, you need to add the `--enable-ip-alias` flag to create the cluster with [Alias IP ranges](https://cloud.google.com/vpc/docs/alias-ip) instead of routes-based networking.
{{< /alert >}}

Flag explanations:

* cluster-version: Agones requires a
  [supported Kubernetes version]({{<ref "/docs/Installation#usage-requirements">}}).
* tags: Defines the tags that will be attached to new nodes in the cluster. This is to grant access through ports via the firewall created in the next step.
* scopes: Defines the Oauth scopes required by the nodes.
* num-nodes: The number of nodes to be created in each of the cluster's zones. Default: 4. Depending on the needs of your game, this parameter should be adjusted.
* no-enable-autoupgrade: Disable automatic upgrades for nodes to reduce the likelihood of in-use games being disrupted.
* enable-image-streaming: Use [Image streaming](https://cloud.google.com/kubernetes-engine/docs/how-to/image-streaming) to pull container images, which leads to significant improvements in initialization times. [Limitations](https://cloud.google.com/kubernetes-engine/docs/how-to/image-streaming#limitations) apply to enable this feature.
* machine-type: The type of machine to use for nodes. Default: e2-standard-4. Depending on the needs of your game, you may wish to [have smaller or larger machines](https://cloud.google.com/compute/docs/machine-types).

#### (Optional) Creating a dedicated node pool

Create a [dedicated node pool](https://cloud.google.com/kubernetes-engine/docs/concepts/node-pools)
for the Agones resources to be installed in. If you skip this step, the Agones controllers will
share the default node pool with your game servers, which is fine for experimentation but not
recommended for a production deployment.

```bash
gcloud container node-pools create agones-system \
  --cluster=[CLUSTER_NAME] \
  --no-enable-autoupgrade \
  --node-taints agones.dev/agones-system=true:NoExecute \
  --node-labels agones.dev/agones-system=true \
  --num-nodes=1
```
where [CLUSTER_NAME] is the name of the cluster you created.

#### (Optional) Creating a metrics node pool

Create a node pool for [Metrics]({{< relref "../../Guides/metrics.md" >}}) if you want to monitor the
 Agones system using Prometheus with Grafana or Cloud Logging and Monitoring.

```bash
gcloud container node-pools create agones-metrics \
  --cluster=[CLUSTER_NAME] \
  --no-enable-autoupgrade \
  --node-taints agones.dev/agones-metrics=true:NoExecute \
  --node-labels agones.dev/agones-metrics=true \
  --num-nodes=1
```

Flag explanations:

* cluster: The name of your existing cluster in which the node pool is created.
* no-enable-autoupgrade: Disable automatic upgrades for nodes to reduce the likelihood of in-use games being disrupted.
* node-taints: The Kubernetes taints to automatically apply to nodes in this node pool.
* node-labels: The Kubernetes labels to automatically apply to nodes in this node pool.
* num-nodes: The Agones system controllers only require a single node of capacity to run. For faster recovery time in the event of a node failure, you can increase the size to 2.

#### (Optional) Creating a node pool for Windows

If you run game servers on Windows, you
need to create a dedicated node pool for those servers. Windows Server 2019 (`WINDOWS_LTSC_CONTAINERD`) is the recommended image for Windows
game servers.

{{< alert title="Warning" color="warning">}}
Running `GameServers` on Windows nodes is currently Alpha. Feel free to file feedback
through [Github issues](https://github.com/googleforgames/agones/issues).
{{< /alert >}}

```bash
gcloud container node-pools create windows \
  --cluster=[CLUSTER_NAME] \
  --no-enable-autoupgrade \
  --image-type WINDOWS_LTSC_CONTAINERD \
  --machine-type e2-standard-4 \
  --num-nodes=4
```

where [CLUSTER_NAME] is the name of the cluster you created.

### Create an Autopilot mode cluster for Agones

Agones supports Autopilot clusters in [alpha]({{<ref "/docs/Guides/feature-stages#alpha">}}).

1. Find a release channel that has a supported Kubernetes version for Agones:

   ```
   gcloud container get-server-config \
     --region=[COMPUTE_REGION] \
     --flatten="channels" \
     --format="yaml(channels) \
     --filter="(channels.channel~[RELEASE_CHANNEL]"
   ```
   Replace the following:

   * **COMPUTE_REGION**: the
   [Google Cloud region](https://cloud.google.com/compute/docs/regions-zones#available)
   for the cluster.
   * **RELEASE_CHANNEL**: the name of the release channel. Can be `RAPID`,
     `REGULAR`, or `STABLE`. Omit the `--filter` flag to view all channels.
  
   The output shows the default and available versions in the specified
   channel.

1. Create the cluster:

    ```bash
    gcloud container clusters create-auto [CLUSTER_NAME] \
      --region=[COMPUTE_REGION] \
      --release-channel=[RELEASE_CHANNEL] \
      --autoprovisioning-network-tags=game-server \
    ```
    Replace the following:

* **CLUSTER_NAME**: the name of your cluster.
* **RELEASE_CHANNEL**: one of
  `rapid`, `regular`, or `stable`. The default is `regular`. To set a specific
  available version in the release channel, use the
  `--cluster-version=[VERSION]` flag.

## Setting up cluster credentials

Finally, let's tell `gcloud` that we are speaking with this cluster, and get auth credentials for `kubectl` to use.

```bash
gcloud config set container/cluster [CLUSTER_NAME]
gcloud container clusters get-credentials [CLUSTER_NAME]
```

[cluster]: https://cloud.google.com/kubernetes-engine/docs/concepts/cluster-architecture
[vms]: https://cloud.google.com/compute/docs/instances/


{{< alert title="Note" color="info">}}
Before planning your production GKE infrastructure, it is worth reviewing the
[different types of GKE clusters that can be created](https://cloud.google.com/kubernetes-engine/docs/concepts/types-of-clusters),
such as Zonal or Regional, as each has different reliability and cost values, and ensuring this aligns with your
Service Level Objectives or Agreements.

This is particularly true for a zonal GKE control plane, which can go down temporarily to adjust for cluster resizing,
[automatic upgrades](https://cloud.google.com/kubernetes-engine/docs/concepts/cluster-upgrades) and
[repairs](https://cloud.google.com/kubernetes-engine/docs/concepts/maintenance-windows-and-exclusions#repairs).
{{< /alert >}}

## Next Steps

### GKE Standard

If you created a GKE Standard cluster, continue to [Install Agones]({{< relref "../Install Agones/_index.md" >}}).

### GKE Autopilot

{{<alert title="Note" color="info">}}
These installation instructions apply to Agones 1.30.
{{</alert>}}

If you created a GKE Autopilot cluster:

1.  [Install Helm](https://helm.sh/docs/intro/quickstart/)
1.  Install Agones:

    ```bash
    helm repo add agones https://agones.dev/chart/stable
    helm repo update
    helm install my-release \
      --namespace agones-system \
      --create-namespace agones/agones \
      --set agones.featureGates="SplitControllerAndExtensions=true"
    ```
    The `SplitControllerAndExtensions=true` feature gate is in the Alpha stage.
    You must specify this feature gate to enable Agones on an Autopilot cluster.

To verify the installation,
[create a GameServer]({{<ref "/docs/Getting Started/create-gameserver.md" >}}).
