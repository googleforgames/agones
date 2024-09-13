---
title: "GameServerAllocation Specification"
linkTitle: "GameServerAllocation"
date: 2019-07-07T03:58:52Z
description: "A `GameServerAllocation` is used to atomically allocate a GameServer out of a set of GameServers.
              This could be a single Fleet, multiple Fleets, or a self managed group of GameServers."
weight: 30
---


Allocation is the process of selecting the optimal `GameServer` that matches the filters defined in the `GameServerAllocation` specification below, and returning its details.

A successful Alloction moves the `GameServer` to the `Allocated` state, which indicates that it is currently active, likely with players on it, and should not be removed until SDK.Shutdown() is called, or it is explicitly manually deleted.

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
      # Specifies which State is the filter to be used when attempting to retrieve a GameServer
      # via Allocation. Defaults to "Ready". The only other option is "Allocated", which can be used in conjunction with
      # label/annotation/player selectors to retrieve an already Allocated GameServer.
      gameServerState: Ready
      # [Stage:Beta]
      # [FeatureFlag:CountsAndLists]
      counters: # selector for counter current values of a GameServer count
        rooms:
          minCount: 1 # minimum value. Defaults to 0.
          maxCount: 5 # maximum value. Defaults to max(int64)
          minAvailable: 1 # minimum available (current capacity - current count). Defaults to 0.
          maxAvailable: 10 # maximum available (current capacity - current count) Defaults to max(int64)
      lists:
        players:
          containsValue: "x6k8z" # only match GameServers who has this value in the list. Defaults to "", which is all.
          minAvailable: 1 # minimum available (current capacity - current count). Defaults to 0.
          maxAvailable: 10 # maximum available (current capacity - current count) Defaults to 0, which translates to max(int64)
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
  # [Stage: Beta]
  # [FeatureFlag:CountsAndLists]
  # `Priorities` configuration alters the order in which `GameServers` are searched for matches to the configured `selectors`.
  #
  # Priority of sorting is in descending importance. I.e. The position 0 `priority` entry is checked first.
  #
  # For `Packed` strategy sorting, this priority list will be the tie-breaker within the least utilised infrastructure, to ensure optimal
  # infrastructure usage while also allowing some custom prioritisation of `GameServers`.
  #
  # For `Distributed` strategy sorting, the entire selection of `GameServers` will be sorted by this priority list to provide the
  # order that `GameServers` will be allocated by.
  # Optional.
  priorities:
  - type: Counter  # Whether a Counter or a List.
    key: rooms  # The name of the Counter or List.
    order: Ascending  # "Ascending" lists smaller available capacity first.
  # [Stage: Beta]
  # [FeatureFlag:CountsAndLists]
  # Counter actions to perform during allocation. Optional.
  counters:
    rooms:
      action: Increment # Either "Increment" or "Decrement" the Counter’s Count.
      amount: 1 # Amount is the amount to increment or decrement the Count. Must be a positive integer.
      capacity: 5 # Amount to update the maximum capacity of the Counter to this number. Min 0, Max int64.
  # List actions to perform during allocation. Optional.
  lists:
    players:
      addValues: # appends values to a List’s Values array. Any duplicate values will be ignored
        - x7un
        - 8inz
      capacity: 40 # Updates the maximum capacity of the Counter to this number. Min 0, Max 1000.
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
- `matchLabels` is a map of {key,value} pairs. A single {key,value} in the matchLabels map is equivalent to an element
  of matchExpressions, whose key field is "key", the operator is "In", and the values array contains only "value".
  The requirements are ANDed. Optional.
- `matchExpressions` is a list of label selector requirements. The requirements are ANDed. Optional.
- `gameServerState` GameServerState specifies which State is the filter to be used when attempting to retrieve a
  GameServer via Allocation. Defaults to "Ready". The only other option is "Allocated", which can be used in
  conjunction with label/annotation/player selectors to retrieve an already Allocated GameServer.
- `counters` (Beta, "CountsAndLists" feature flag) enables filtering based on game server Counter status, such as
  the minimum and maximum number of active rooms. This helps in selecting game servers based on their current activity
  or capacity. Optional.
- `lists` (Beta, "CountsAndLists" feature flag) enables filtering based on game server List status, such as allowing
    for inclusion or exclusion of specific players. Optional.
- `scheduling` defines how GameServers are organised across the cluster, in this case specifically when allocating
  `GameServers` for usage.
  "Packed" (default) is aimed at dynamic Kubernetes clusters, such as cloud providers, wherein we want to bin pack
  resources. "Distributed" is aimed at static Kubernetes clusters, wherein we want to distribute resources across the entire
  cluster. See [Scheduling and Autoscaling]({{< ref "/docs/Advanced/scheduling-and-autoscaling.md" >}}) for more details.
- `metadata` is an optional list of custom labels and/or annotations that will be used to patch
  the game server's metadata in the moment of allocation. This can be used to tell the server necessary session data
- `priorities` (Beta, requires `CountsAndLists` feature flag) alters the priority by which game `GameServers` are allocated by available capacity.
- `counters` (Beta, "CountsAndLists" feature flag) Counter actions to perform during allocation.
- `lists` (Beta, "CountsAndLists" feature flag) List actions to perform during allocation.

Once created the `GameServerAllocation` will have a `status` field consisting of the following:

- `State` is the current state of a GameServerAllocation, e.g. `Allocated`, or `UnAllocated`
- `GameServerName` is the name of the game server attached to this allocation, once the `state` is `Allocated`
- `Ports` is a list of the ports that the game server makes available. See [the GameServer Reference]({{< ref "/docs/Reference/gameserver.md" >}}) for more details.
- `Address` is the primary network address where the game server can be reached.
- `Addresses` is an array of all network addresses where the game server can be reached. It is a copy of the [`Node.Status.addresses`][addresses] field for the node the `GameServer` is scheduled on.
- `NodeName` is the name of the node that the gameserver is running on.
- `Source` is "local" unless this allocation is from a remote cluster, in which case `Source` is the endpoint of the remote agones-allocator. See [Multi-cluster Allocation]({{< ref "/docs/Advanced/multi-cluster-allocation.md" >}}) for more details.
- `Metadata` conststs of:
  - `Labels` containing the labels of the game server at allocation time.
  - `Annotations` containing the annotations of the underlying game server at allocation time.
- `Counters` (Beta, "CountsAndLists" feature flag) is a map of [CounterStatus][counterstatus] of the game server at allocation time.
- `Lists` (Beta, "CountsAndLists" feature flag) is a map of [ListStatus][liststatus] of the game server at allocation time.
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
[addresses]: {{% k8s-api-version href="#nodeaddress-v1-core" %}}
[counterstatus]: {{% ref "/docs/Reference/agones_crd_api_reference.html#agones.dev/v1.CounterStatus" %}}
[liststatus]: {{% ref "/docs/Reference/agones_crd_api_reference.html#agones.dev/v1.ListStatus" %}}

## Next Steps:

- Check out the [Allocator Service]({{< ref "/docs/Advanced/allocator-service.md" >}}) as a richer alternative to `GameServerAllocation`.
