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

{{< tabpane >}}
  {{< tab header="selectors" lang="yaml" >}}
apiVersion: "allocation.agones.dev/v1"
kind: GameServerAllocation
spec:
  # GameServer selector from which to choose GameServers from.
  # Defaults to all GameServers.
  # matchLabels, matchExpressions, gameServerState and player filters can be used for filtering.
  # See: https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/ for more details on label selectors.
  # An ordered list of GameServer label selectors.
  # If the first selector is not matched, the selection attempts the second selector, and so on.
  # This is useful for things like smoke testing of new game servers.
  selectors:
    - matchLabels:
        agones.dev/fleet: green-fleet
        # [Stage:Alpha]
        # [FeatureFlag:PlayerAllocationFilter]    
      players:
        minAvailable: 0
        maxAvailable: 99
    - matchLabels:
        agones.dev/fleet: blue-fleet
    - matchLabels:
        game: my-game
      matchExpressions:
        - {key: tier, operator: In, values: [cache]}
      # [Stage:Beta]
      # [FeatureFlag:StateAllocationFilter]
      # Specifies which State is the filter to be used when attempting to retrieve a GameServer
      # via Allocation. Defaults to "Ready". The only other option is "Allocated", which can be used in conjunction with
      # label/annotation/player selectors to retrieve an already Allocated GameServer.    
      gameServerState: Ready
      # [Stage:Alpha]
      # [FeatureFlag:PlayerAllocationFilter]
      # Provides a filter on minimum and maximum values for player capacity when retrieving a GameServer
      # through Allocation. Defaults to no limits.
      players:
        minAvailable: 0
        maxAvailable: 99
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
  {{< /tab >}}
  {{< tab header="required & preferred (deprecated)" lang="yaml" >}}
apiVersion: "allocation.agones.dev/v1"
kind: GameServerAllocation
spec:
  # Deprecated, use field selectors instead.
  # GameServer selector from which to choose GameServers from.
  # Defaults to all GameServers.
  # matchLabels, matchExpressions, gameServerState and player filters can be used for filtering.
  # See: https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/ for more details on label selectors.
  # Deprecated, use field selectors instead.
  required:
    matchLabels:
      game: my-game
    matchExpressions:
      - {key: tier, operator: In, values: [cache]}
    # [Stage:Beta]
    # [FeatureFlag:StateAllocationFilter]
    # Specifies which State is the filter to be used when attempting to retrieve a GameServer
    # via Allocation. Defaults to "Ready". The only other option is "Allocated", which can be used in conjunction with
    # label/annotation/player selectors to retrieve an already Allocated GameServer.    
    gameServerState: Ready
    # [Stage:Alpha]
    # [FeatureFlag:PlayerAllocationFilter]
    # Provides a filter on minimum and maximum values for player capacity when retrieving a GameServer
    # through Allocation. Defaults to no limits.
    players:
      minAvailable: 0
      maxAvailable: 99
  # Deprecated, use field selectors instead.
  # An ordered list of preferred GameServer label selectors
  # that are optional to be fulfilled, but will be searched before the `required` selector.
  # If the first selector is not matched, the selection attempts the second selector, and so on.
  # If any of the preferred selectors are matched, the required selector is not considered.
  # This is useful for things like smoke testing of new game servers.
  # This also support matchExpressions, gameServerState and player filters.
  preferred:
    - matchLabels:
        agones.dev/fleet: green-fleet
      # [Stage:Alpha]
      # [FeatureFlag:PlayerAllocationFilter]    
      players:
        minAvailable: 0
        maxAvailable: 99
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
  {{< /tab >}}
{{< /tabpane >}}

The `spec` field is the actual `GameServerAllocation` specification, and it is composed as follows:

- Deprecated, use `selectors` instead. If `selectors` is set, this field will be ignored.
  `required` is a [GameServerSelector][gameserverselector]
  (matchLabels. matchExpressions, gameServerState and player filters) from which to choose GameServers from.
- Deprecated, use `selectors` instead. If `selectors` is set, this field will be ignored.
  `preferred` is an ordered list of preferred [GameServerSelector][gameserverselector]
  that are _optional_ to be fulfilled, but will be searched before the `required` selector.
  If the first selector is not matched, the selection attempts the second selector, and so on.
  If any of the `preferred` selectors are matched, the `required` selector is not considered.
  This is useful for things like smoke testing of new game servers.
- `selectors` is an ordered list of [GameServerSelector][gameserverselector].
  If the first selector is not matched, the selection attempts the second selector, and so on.
  This is useful for things like smoke testing of new game servers.
- `scheduling` defines how GameServers are organised across the cluster, in this case specifically when allocating
  `GameServers` for usage.
  "Packed" (default) is aimed at dynamic Kubernetes clusters, such as cloud providers, wherein we want to bin pack
  resources. "Distributed" is aimed at static Kubernetes clusters, wherein we want to distribute resources across the entire
  cluster. See [Scheduling and Autoscaling]({{< ref "/docs/Advanced/scheduling-and-autoscaling.md" >}}) for more details.
- `metadata` is an optional list of custom labels and/or annotations that will be used to patch
  the game server's metadata in the moment of allocation. This can be used to tell the server necessary session data

{{< alert title="Info" color="info" >}}

For performance reasons, the query cache for a `GameServerAllocation` is _eventually consistent_.

Usually, the cache is populated practically immediately on `GameServer` change, but under high load of the Kubernetes
control plane, it may take some time for updates to `GameServer` selectable features to be populated into the cache
(although this doesn't affect the atomicity of the Allocation operation).

While Agones will do a small series of retries when an allocatable `GameServer` is not available in its cache,
depending on your game requirements, it may be worth implementing your own more extend retry mechanism for
Allocation requests for high load scenarios.

{{< /alert >}}

Each `GameServerAllocation` will allocate from a single [namespace][namespace]. The namespace can be specified outside of
the spec, either with the `--namespace` flag when using the command line / `kubectl` or
[in the url]({{% ref "/docs/Guides/access-api.md#allocate-a-gameserver-from-a-fleet-named-simple-game-server-with-gameserverallocation"  %}})
when using an API call. If not specified when using the command line, the [namespace][namespace] will be automatically set to `default`.


[gameserverselector]: {{% ref "/docs/Reference/agones_crd_api_reference.html#allocation.agones.dev/v1.GameServerSelector"  %}}
[namespace]: https://kubernetes.io/docs/concepts/overview/working-with-objects/namespaces
