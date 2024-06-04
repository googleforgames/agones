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


## Upgrading Kubernetes

The following are strategies for safely upgrading the underlying Kubernetes cluster from one version to another.
They may require adjustment to your particular game architecture but should provide a solid foundation for updating your cluster safely.

The recommended approach is to use [multiple clusters](#multiple-clusters), such that the upgrade can be tested
gradually with production load and easily rolled back if the need arises.

Agones has [multiple supported Kubernetes versions]({{< relref "_index.md#agones-and-kubernetes-supported-versions" >}}) for each version. You can stick with a minor Kubernetes version until it is not supported by Agones, but it is recommended to do supported minor (e.g. 1.12.1 ➡ 1.13.2) Kubernetes version upgrades at the same time as a matching Agones upgrades.

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

## SDK Compatibility Guarantees

The SDK compatibility contract aims to ensure smooth upgrades and reduce the need for frequent
binary redeployments for game server users due to SDK changes. We implement SDK feature versioning
that matches our [Feature Gates]({{% ref "/docs/Guides/feature-stages.md#feature-gates" %}}) to
provide clear documentation of API maturity levels and history (Stable, Beta, Alpha) with release
information.

**Our SDK Server compatibility contract as of Agones v1.41.0**: A
game server binary using Beta and Stable SDKs will remain compatible with a _newer_ Agones Release,
within possible deprecation windows:
- If your game server uses a non-deprecated Stable API, your binary will be compatible for 10
releases (~1y) or more starting from the SDK version packaged.
  - For example, if the game server uses non-deprecated stable APIs in the 1.40 SDK, it will be
  compatible through the 1.50 SDK.
  - Stable APIs will almost certainly be compatible beyond 10 releases, as deprecation of stable
  APIs is rare, but 10 releases is guaranteed.
- If your game server uses a non-deprecated Beta API, your binary will be guaranteed compatible for
  5 releases (~6mo), but may be compatible past that point.
- Alpha SDK APIs are subject to change between releases.
  - A game server binary using Alpha SDKs may not be compatible with a newer Agones release if
  breaking changes have been made between releases.
  - When we make incompatible Alpha changes, we will document the APIs involved.

## SDK Deprecation Policies as of Agones 1.41

- Client SDK updates are not mandatory for game server binaries except for SDK when the underlying
proto format has deprecations or breaking Alpha API changes.

- Breaking changes will be called out in upgrade documentation to allow admins to plan their upgrades.

- Expect to check if there are breaking changes to Stable APIs you use yearly or Beta APIs semi-annually.

### Stable Deprecation Policies

A Stable API may be marked as deprecated in release X and removed from Stable in release X+10.

### Beta Deprecation Policies

When a SDK API feature graduates from Beta to Stable at release X, the API will be present in both Beta and
Stable surfaces from release X to release X+5. The Beta API is marked as deprecated in release X and
removed from Beta in release X+5.
A Beta API may be marked as deprecated in release X and removed from Beta in release X+5 without the
API graduating to Stable if it's determined that changes need to be made to the Beta API.

### Alpha Deprecation Policies

There is no guaranteed compatibility between releases for Alpha SDKs. When an Alpha API
graduates to Beta the API will be deleted from the Alpha SDK with no overlapping release.
An API may be removed from the Alpha SDK during any release without graduating to Beta.

## SDK APIs and Stability Levels

"Legacy" indicates that this API has been in the SDK Server in a release before we began tracking
SDK compatibility with release 1.41.0. \
The Actions may differ from the [Client SDK]({{< relref "Client SDKs">}}) depending on how each
Client SDK is implemented.

| Area                | Action                | Stable | Beta | Alpha  |
|---------------------|-----------------------|--------|------|--------|
| Lifecycle           | Ready                 | Legacy |      |        |
| Lifecycle           | Health                | Legacy |      |        |
| Lifecycle           | Reserve               | Legacy |      |        |
| Lifecycle           | Allocate              | Legacy |      |        |
| Lifecycle           | Shutdown              | Legacy |      |        |
| Configuration       | GetGameServer         | Legacy |      |        |
| Configuration       | WatchGameServer       | Legacy |      |        |
| Metadata            | SetAnnotation         | Legacy |      |        |
| Metadata            | SetLabel              | Legacy |      |        |
| Counters            | GetCounter            |        |      | 1.37.0 |
| Counters            | UpdateCounter         |        |      | 1.37.0 |
| Lists               | GetList               |        |      | 1.37.0 |
| Lists               | UpdateList            |        |      | 1.37.0 |
| Lists               | AddListValue          |        |      | 1.37.0 |
| Lists               | RemoveListValue       |        |      | 1.37.0 |
| Player Tracking     | GetPlayerCapacity     |        |      | Legacy |
| Player Tracking     | SetPlayerCapacity     |        |      | Legacy |
| Player Tracking     | PlayerConnect         |        |      | Legacy |
| Player Tracking     | GetConnectedPlayers   |        |      | Legacy |
| Player Tracking     | IsPlayerConnected     |        |      | Legacy |
| Player Tracking     | GetPlayerCount        |        |      | Legacy |
| Player Tracking     | PlayerDisconnect      |        |      | Legacy |

