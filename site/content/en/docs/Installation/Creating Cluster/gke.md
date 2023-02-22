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

1. Visit the [Kubernetes Engine][kubernetes] page in the Google Cloud console.
1. Create or select a project.
1. Wait for the API and related services to be enabled. This can take several
   minutes.
1. [Enable billing][billing] for your project.

   If you are not an existing Google Cloud user, you may be able to enroll for a $300 US [Free Trial][trial] credit.

[kubernetes]: https://console.cloud.google.com/kubernetes/list
[billing]: https://support.google.com/cloud/answer/6293499#enable-billing
[trial]: https://cloud.google.com/free/

### Choosing a shell

To complete this quickstart, use either [Cloud Shell][cloud-shell] or a local
shell.

Cloud Shell is a shell environment for managing resources hosted on Google
Cloud. Cloud Shell comes preinstalled with the [`gcloud`][gcloud] and
[`kubectl`][kubectl] command-line tools. You use `gcloud` to run commands
against the Google Cloud APIs, and you use `kubectl` to run commands against
Kubernetes APIs in clusters.

If you prefer using your local shell, you must install the `gcloud` and
`kubectl` command-line tools in your environment.

[cloud-shell]: https://cloud.google.com/shell/
[gcloud]: https://cloud.google.com/sdk/gcloud/
[kubectl]: https://kubernetes.io/docs/user-guide/kubectl-overview/

#### Cloud Shell

To launch Cloud Shell, perform the following steps:

1.  Go to the [Google Cloud console][cloud].
1.  From the top-right corner of the console, click
    **Activate Cloud Shell**: ![cloud shell](../../../../images/cloud-shell.png)

    A Cloud Shell session opens inside a frame at the bottom of the console. Use this shell to run `gcloud` and `kubectl` commands.
1.  Set a compute region in your geographical region with the following
    command. An example compute region is `us-west1`. For a full list, refer to
    [Available regions and zones][zones].

    ```bash
    gcloud config set compute/region [COMPUTE_REGION]
    ```

[cloud]: https://console.cloud.google.com/home/dashboard
[zones]: https://cloud.google.com/compute/docs/regions-zones/#available

#### Local shell

To install `gcloud` and `kubectl`, perform the following steps:

1. [Install the Google Cloud SDK][gcloud-install], which includes the `gcloud`
   command-line tool.
1. Initialize some default configuration by running the following command.
   
   ```bash
   gcloud init
   ```
   When asked `Do you want to configure a default Compute Region and Zone? (Y/n)?`, enter `Y` and choose a zone in your region of choice.

Installing `gcloud` also installs `kubectl` for you. You can also manually
install `kubectl` by running the following command:

```bash
gcloud components install kubectl
```

[gcloud-install]: https://cloud.google.com/sdk/docs/quickstarts

## Creating the firewall

We need a firewall to allow UDP traffic to nodes tagged as `game-server` via ports 7000-8000. These firewall rules apply to cluster nodes you will create in the
next section.

```bash
gcloud compute firewall-rules create game-server-firewall \
  --network default \
  --allow udp:7000-8000,tcp:7000-8000 \
  --source-ranges 0.0.0.0/0 \
  --direction INGRESS \
  --priority 500 \
  --target-tags game-server \
  --description "Firewall to allow game server udp traffic"
```

## Creating the cluster

A [cluster][cluster] consists of *control plane* machines and worker machines
called *nodes*. In Google Kubernetes Engine, nodes are
[Compute Engine virtual machine][vms] instances that run the Kubernetes
processes necessary to make them part of the cluster.

```bash
gcloud container clusters create-auto [CLUSTER_NAME] \
  --region=[COMPUTE_REGION] \
  --release-channel=[RELEASE_CHANNEL] \
  --autoprovisioning-network-tags=game-server \
  --tags=game-server
```
Replace the following:

*  COMPUTE_REGION: the Google Cloud region for the cluster, such as
  `us-central1`.
*  RELEASE_CHANNEL: the GKE release channel for the cluster, which determines
   how quickly new Kubernetes versions become available to use.

Agones requires a cluster running Kubernetes version {{% k8s-version %}} or
later. To see which version is the default in each GKE release channel, check
the
[release notes](https://cloud.google.com/kubernetes-engine/docs/release-notes#current_versions).

### (Optional) Run GameServers on Windows

If you run game servers on Windows, create a GKE Standard cluster with a
dedicated Windows node pool. Windows Server 2019 (`WINDOWS_LTSC`) is the
recommended image for Windows game servers.

{{< alert title="Warning" color="warning">}}
Running `GameServers` on Windows nodes is currently Alpha. File feedback
through [Github issues](https://github.com/googleforgames/agones/issues).
{{< /alert >}}

1.  Create a GKE Standard cluster:

    ```bash
    gcloud container clusters create [CLUSTER_NAME] \
      --cluster-version={{% k8s-version %}} \
      --no-enable-autoupgrade \
      --machine-type e2-standard-4 \
      --num-nodes=4 \
      --enable-ip-alias \
      --enable-image-streaming
    ```

1.  Create a Windows node pool:

    ```bash
    gcloud container node-pools create windows \
      --cluster=[CLUSTER_NAME] \
      --image-type=WINDOWS_LTSC \
      --machine-type=e2-standard-4 \
      --num-nodes=4 \
      --no-enable-autoupgrade
    ```

## Setting up cluster credentials

After creating the cluster, get authentication credentials for `kubectl` to
use against the Kubernetes API in that cluster.

```bash
gcloud config set container/cluster [CLUSTER_NAME]
gcloud container clusters get-credentials [CLUSTER_NAME]
```

[cluster]: https://cloud.google.com/kubernetes-engine/docs/concepts/cluster-architecture
[vms]: https://cloud.google.com/compute/docs/instances/


{{< alert title="Note" color="info">}}
Before planning your production GKE infrastructure, it is worth reviewing the
[different modes of GKE clusters that can be created](https://cloud.google.com/kubernetes-engine/docs/concepts/choose-cluster-mode),
such as Zonal or Regional, as each has different reliability and cost values, and ensuring this aligns with your
Service Level Objectives or Agreements.
{{< /alert >}}

## Next Steps

- Continue to [Install Agones]({{< relref "../Install Agones/_index.md" >}}).
