---
title: "GameServerAllocation Specification"
linkTitle: "GameServerAllocation"
date: 2019-07-07T03:58:52Z
description: "A `GameServerAllocation` is used to atomically allocate a GameServer out of a set of GameServers. 
              This could be a single Fleet, multiple Fleets, or a self managed group of GameServers."
weight: 30
---

A full `GameServerAllocation` specification is available below and in the 
{{< ghlink href="/examples/gameserverallocation.yaml" >}}example folder{{< /ghlink >}} for reference:

```yaml
apiVersion: "allocation.agones.dev/v1"
kind: GameServerAllocation
spec:
  # GameServer selector from which to choose GameServers from.
  # GameServers still have the hard requirement to be `Ready` to be allocated from
  # however we can also make available `matchExpressions` for even greater
  # flexibility.
  # Below is an example of a GameServer allocated against a given fleet.
  # See: https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/ for more details
  required:
    matchLabels:
      game: my-game
    matchExpressions:
      - {key: tier, operator: In, values: [cache]}
  # An ordered list of preferred GameServer label selectors
  # that are optional to be fulfilled, but will be searched before the `required` selector.
  # If the first selector is not matched, the selection attempts the second selector, and so on.
  # If any of the preferred selectors are matched, the required selector is not considered.
  # This is useful for things like smoke testing of new game servers.
  # This also support `matchExpressions`
  preferred:
    - matchLabels:
        agones.dev/fleet: green-fleet
    - matchLabels:
        agones.dev/fleet: blue-fleet
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
- `preferred` is an ordered list of preferred
  [label selectors](https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/)
  that are _optional_ to be fulfilled, but will be searched before the `required` selector.
  If the first selector is not matched, the selection attempts the second selector, and so on.
  If any of the `preferred` selectors are matched, the `required` selector is not considered.
  This is useful for things like smoke testing of new game servers.
- `scheduling` defines how GameServers are organised across the cluster, in this case specifically when allocating
  `GameServers` for usage.
   "Packed" (default) is aimed at dynamic Kubernetes clusters, such as cloud providers, wherein we want to bin pack
   resources. "Distributed" is aimed at static Kubernetes clusters, wherein we want to distribute resources across the entire
   cluster. See [Scheduling and Autoscaling]({{< ref "/docs/Advanced/scheduling-and-autoscaling.md" >}}) for more details.
 
- `metadata` is an optional list of custom labels and/or annotations that will be used to patch 
  the game server's metadata in the moment of allocation. This can be used to tell the server necessary session data