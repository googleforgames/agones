---
title: "Fleet Specification"
linkTitle: "Fleet"
date: 2019-01-03T03:58:52Z
description: "A `Fleet` is a set of warm GameServers that are available to be allocated from."
weight: 20
---

To allocate a `GameServer` from a `Fleet`, use a `FleetAllocation`.

Like any other Kubernetes resource you describe a `Fleet`'s desired state via a specification written in YAML or JSON to the Kubernetes API. The Agones controller will then change the actual state to the desired state.

A full `Fleet` specification is available below and in the {{< ghlink href="examples/fleet.yaml" >}}example folder{{< /ghlink >}} for reference :

```yaml
apiVersion: "stable.agones.dev/v1alpha1"
kind: Fleet
metadata:
  name: fleet-example
spec:
  replicas: 2
  scheduling: Packed
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxSurge: 25%
      maxUnavailable: 25%  
  template:
    metadata:
      labels:
        foo: bar
    spec:
      ports:
      - name: default
        portPolicy: "dynamic"
        containerPort: 26000
      health:
        initialDelaySeconds: 30
        periodSeconds: 60
      template:
        spec:
          containers:
          - name: example-server
            image: gcr.io/agones/test-server:0.1
```

Since Agones defines a new 
[Custom Resources Definition (CRD)](https://kubernetes.io/docs/concepts/api-extension/custom-resources/) 
we can define a new resource using the kind `Fleet` with the custom group `stable.agones.dev` and API 
version `v1alpha1`.

You can use the metadata field to target a specific 
[namespaces](https://kubernetes.io/docs/concepts/overview/working-with-objects/namespaces/) but also 
attach specific [annotations](https://kubernetes.io/docs/concepts/overview/working-with-objects/annotations/) 
and [labels](https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/) to your resource. 
This is a very common pattern in the Kubernetes ecosystem.

The `spec` field is the actual `Fleet` specification and it is composed as follow:

- `replicas` is the number of `GameServers` to keep Ready or Allocated in this Fleet
- `scheduling` defines how GameServers are organised across the cluster. Affects backing Pod scheduling, as well as scale
                 down mechanics.
                 "Packed" (default) is aimed at dynamic Kubernetes clusters, such as cloud providers, wherein we want to bin pack
                 resources. "Distributed" is aimed at static Kubernetes clusters, wherein we want to distribute resources across the entire
                 cluster. See [Scheduling and Autoscaling]({{< relref "../Advanced/scheduling-and-autoscaling.md" >}}) for more details.
- `strategy` is the `GameServer` replacement strategy for when the `GameServer` template is edited.
  - `type` is replacement strategy for when the GameServer template is changed. Default option is "RollingUpdate", but "Recreate" is also available.
    - `RollingUpdate` will increment by `maxSurge` value on each iteration, while decrementing by `maxUnavailable` on each iteration, until all GameServers have been switched from one version to another.   
    - `Recreate` terminates all non-allocated `GameServers`, and starts up a new set with the new `GameServer` configuration to replace them.
  - `rollingUpdate` is only relevant when `type: RollingUpdate`
    - `maxSurge` is the amount to increment the new GameServers by. Defaults to 25%
    - `maxUnavailable` is the amount to decrements GameServers by. Defaults to 25%
- `template` a full `GameServer` configuration template.
   See the [GameServer]({{< relref "gameserver.md" >}}) reference for all available fields.

{{% feature publishVersion="0.8.0" %}}
## GameServer Allocation Specification

A `GameServerAllocation` is used to atomically allocate a GameServer out of a set of GameServers. 
This could be a single Fleet, multiple Fleets, or a self managed group of GameServers.

A full `GameServerAllocation` specification is available below and in the 
{{< ghlink href="/examples/gameserverallocation.yaml" >}}example folder{{< /ghlink >}} for reference:


```yaml
apiVersion: "stable.agones.dev/v1alpha1"
kind: GameServerAllocation
metadata:
  generateName: simple-udp-
spec:
  required:
    matchLabels:
      game: my-game
    matchExpressions:
      - {key: tier, operator: In, values: [cache]}
  # ordered list of preferred allocations 
  # This also support `matchExpressions`
  preferred:
    - matchLabels:
        stable.agones.dev/fleet: green-fleet
    - matchLabels:
        stable.agones.dev/fleet: blue-fleet
  # defines how GameServers are organised across the cluster.
  # Options include:
  # "Packed" (default) is aimed at dynamic Kubernetes clusters, such as cloud providers, wherein we want to bin pack
  # resources
  # "Distributed" is aimed at static Kubernetes clusters, wherein we want to distribute resources across the entire
  # cluster
  scheduling: Packed
  # Optional custom metadata that is added to the game server at allocation
  # You can use this to tell the server necessary session data
  metadata:
    labels:
      mode: deathmatch
    annotations:
      map:  garden22
```

We recommend using `metadata > generateName`, to declare to Kubernetes that a unique
name for the `GameServerAllocation` is generated when the `GameServerAllocation` is created.

The `spec` field is the actual `GameServerAllocation` specification and it is composed as follow:

- `required` is a [label selector](https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/) 
   (matchLabels and/or matchExpressions) from which to choose GameServers from.
   GameServers still have the hard requirement to be `Ready` to be allocated from
- `preferred` is an order list of [label selectors](https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/)
   out of the `required` set.
   If the first selector is not matched, the selection attempts the second selector, and so on.
   This is useful for things like smoke testing of new game servers. 
- `scheduling` defines how GameServers are organised across the cluster, in this case specifically when allocating
  `GameServers` for usage.
   "Packed" (default) is aimed at dynamic Kubernetes clusters, such as cloud providers, wherein we want to bin pack
   resources. "Distributed" is aimed at static Kubernetes clusters, wherein we want to distribute resources across the entire
   cluster. See [Scheduling and Autoscaling]({{< ref "/docs/Advanced/scheduling-and-autoscaling.md" >}}) for more details.
 
- `metadata` is an optional list of custom labels and/or annotations that will be used to patch 
  the game server's metadata in the moment of allocation. This can be used to tell the server necessary session data
{{% /feature %}}

# Fleet Allocation Specification

{{% feature publishVersion="0.8.0" %}}
> Fleet Allocation is **deprecated** in version 0.8.0, and will be removed in the 0.10.0 release.
  Migrate to using GameServer Allocation instead.
{{% /feature %}}

A `FleetAllocation` is used to allocate a `GameServer` out of an existing `Fleet`

A full `FleetAllocation` specification is available below and in the 
{{< ghlink href="examples/fleetallocation.yaml" >}}example folder{{< /ghlink >}} for reference:

```yaml
apiVersion: "stable.agones.dev/v1alpha1"
kind: FleetAllocation
metadata:
  generateName: fleet-allocation-example-
spec:
  fleetName: fleet-example
  metadata:
    labels:
      mode: deathmatch
    annotations:
      map:  garden22
```

We recommend using `metadata > generateName`, to declare to Kubernetes that a unique
name for the `FleetAllocation` is generated when the `FleetAllocation` is created.

The `spec` field is the actual `FleetAllocation` specification and it is composed as follow:

- `fleetName` is the name of an existing Fleet. If this doesn't exist, an error will be returned
  when the `FleetAllocation` is created
- `metadata` is an optional list of custom labels and/or annotations that will be used to patch 
  the game server's metadata in the moment of allocation. This can be used to tell the server necessary session data
