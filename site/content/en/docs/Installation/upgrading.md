---
title: "Upgrading Agones and Kubernetes"
linkTitle: "Upgrading"
weight: 100
date: 2019-08-16T00:19:19Z
description: >
  Strategies and techniques for managing Agones and Kubernetes upgrades in a safe manner.
---

{{< alert color="info" title="Note" >}}
Whichever approach you take to upgrading Agones, make sure to test it in your development environment 
before applying it to production.
{{< /alert >}}

## Upgrading Agones

The following are strategies for safely upgrading Agones from one version to another. They may require adjustment to 
your particular game architecture but should provide a solid foundation for updating Agones safely.

The recommended approach is to use [multiple clusters](#upgrading-agones-multiple-clusters), such that the upgrade can be tested
gradually with production load and easily rolled back if the need arises.

{{< alert color="warning" title="Warning" >}}
Changing [Feature Gates]({{% ref "/docs/Guides/feature-stages.md#feature-gates" %}}) within your Agones install
can constitute an "upgrade" as it may create or remove functionality
in the Agones installation that may not be forward or backward compatible with installed resources in an existing 
installation.
{{< /alert >}}

### Upgrading Agones: Multiple Clusters

We essentially want to transition our GameServer allocations from a cluster with the old version of Agones,
to a cluster with the upgraded version of Agones while ensuring nothing surprising 
happens during this process.

This also allows easy rollback to the previous infrastructure that we already know to be working in production, with
minimal interruptions to player experience.

The following are steps to implement this:

1. Create a new cluster of the same size or smaller as the current cluster.
2. Install the new version of Agones on the new cluster.
3. Deploy the same set of Fleets, GameServers and FleetAutoscalers from the old cluster into the new cluster.
4. With your matchmaker, start sending a small percentage of your matched players' game sessions to the new cluster.
5. Assuming everything is working successfully on the new cluster, slowly increase the percentage of matched sessions to the new cluster, until you reach 100%.
6. Once you are comfortable with the stability of the new cluster with the new Agones version, shut down the old cluster.
7. Congratulations - you have now upgraded to a new version of Agones! 👍

### Upgrading Agones: Single Cluster

If you are upgrading a single cluster, we recommend creating a maintenance window, in which your game goes offline
for the period of your upgrade, as there will be a short period in which Agones will be non-responsive during the upgrade.

#### Installation with install.yaml

If you installed [Agones with install.yaml]({{< relref "./Install Agones/yaml.md" >}}), then you will need to delete
the previous installation of Agones before upgrading to the new version, as we need to remove all of Agones before installing
the new version.

1. Start your maintenance window.
1. Delete the current set of Fleets, GameServers and FleetAutoscalers in your cluster.
1. Make sure to delete the same version of Agones that was previously installed, for example:
   `kubectl delete -f https://raw.githubusercontent.com/googleforgames/agones/<old-release-version>/install/yaml/install.yaml`
1. Install Agones [with install.yaml]({{< relref "./Install Agones/yaml.md" >}}).
1. Deploy the same set of Fleets, GameServers and FleetAutoscalers back into the cluster.
1. Run any other tests to ensure the Agones installation is working as expected.
1. Close your maintenance window.
7. Congratulations - you have now upgraded to a new version of Agones! 👍

#### Installation with Helm

Helm features capabilities for upgrading to newer versions of Agones without having to uninstall Agones completely.

For details on how to use Helm for upgrades, see the [helm upgrade](https://v2.helm.sh/docs/helm/#helm-upgrade) documentation.

Given the above, the steps for upgrade are simpler:

1. Start your maintenance window.
2. Delete the current set of Fleets, GameServers and FleetAutoscalers in your cluster.
3. Run `helm upgrade` with the appropriate arguments, such a `--version`, for your specific upgrade
4. Deploy the same set of Fleets, GameServers and FleetAutoscalers back into the cluster.
5. Run any other tests to ensure the Agones installation is working as expected.
6. Close your maintenance window.
7. Congratulations - you have now upgraded to a new version of Agones! 👍

{{< alert color="warning" title="Warning" >}}
If you are getting an error of `cannot convert int64 to float64` on `helm upgrade` you will need to `helm uninstall` 
the original installation and do a full `helm install` to the new version to upgrade Agones. 

This is due to [helm/helm#5806](https://github.com/helm/helm/issues/5806) and will be resolved with Kubernetes 1.20
{{< /alert >}}

## Upgrading Kubernetes

The following are strategies for safely upgrading the underlying Kubernetes cluster from one version to another.
They may require adjustment to your particular game architecture but should provide a solid foundation for updating your cluster safely.

The recommended approach is to use [multiple clusters](#multiple-clusters), such that the upgrade can be tested
gradually with production load and easily rolled back if the need arises.

Agones currently has only a [single supported Kubernetes version]({{< relref "_index.md#usage-requirements" >}}) for each version,
so it is recommended to do supported minor (e.g. 1.12.1 ➡ 1.13.2) Kubernetes version upgrades at the same time as a matching Agones upgrades.

Patch upgrades (e.g. 1.12.1 ➡ 1.12.3) within the same minor version of Kubernetes can be done at any time. 

### Multiple Clusters

This process is very similar to the [Upgrading Agones: Multiple Clusters](#upgrading-agones-multiple-clusters) approach above.

We essentially want to transition our GameServer allocations from a cluster with the old version of Kubernetes,
to a cluster with the upgraded version of Kubernetes while ensuring nothing surprising 
happens during this process.

This also allows easy rollback to the previous infrastructure that we already know to be working in production, with
minimal interruptions to player experience.

The following are steps to implement this:

1. Create a new cluster of the same size or smaller as the current cluster, with the new version of Kubernetes
2. Install the same version of Agones on the new cluster, as you have on the previous cluster.
3. Deploy the same set of Fleets and/or GameServers from the old cluster into the new cluster.
4. With your matchmaker, start sending a small percentage of your matched players' game sessions to the new cluster.
5. Assuming everything is working successfully on the new cluster, slowly increase the percentage of matched sessions to the new cluster, until you reach 100%.
6. Once you are comfortable with the stability of the new cluster with the new Kubernetes version, shut down the old cluster.
7. Congratulations - you have now upgraded to a new version of Kubernetes! 👍

### Single Cluster

If you are upgrading a single cluster, we recommend creating a maintenance window, in which your game goes offline
for the period of your upgrade, as there will be a short period in which Agones will be non-responsive during the node
upgrades.

1. Start your maintenance window.
1. Scale your Fleets down to 0 and/or delete your GameServers. This is a good safety measure so there aren't race conditions
   between the Agones controller being recreated and GameServers being deleted doesn't occur, and GameServers can end up stuck in erroneous states.
1. Start and complete your control plane upgrade(s).
1. Start and complete your node upgrades.
1. Scale your Fleets back up and/or recreate your GameServers. 
1. Run any other tests to ensure the Agones installation is still working as expected.
1. Close your maintenance window.
7. Congratulations - you have now upgraded to a new version of Kubernetes! 👍
