# Fleet Specification

A `Fleet` is a set of warm GameServers that are available to be allocated from.

To allocate a `GameServer` from a `Fleet`, use a `FleetAllocation`.

Like any other Kubernetes resource you describe a `Fleet`'s desired state via a specification written in YAML or JSON to the Kubernetes API. The Agones controller will then change the actual state to the desired state.

A full `Fleet` specification is available below and in the [example folder](../examples/fleet.yaml) for reference :

```yaml
apiVersion: "stable.agones.dev/v1alpha1"
kind: Fleet
metadata:
  name: fleet-example
spec:
  replicas: 2
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
- `strategy` is the `GameServer` replacement strategy for when the `GameServer` template is edited.
  - `type` is replacement strategy for when the GameServer template is changed. Default option is "RollingUpdate", but "Recreate" is also available.
    - `RollingUpdate` will increment by `maxSurge` value on each iteration, while decrementing by `maxUnavailable` on each iteration, until all GameServers have been switched from one version to another.   
    - `Recreate` terminates all non-allocated `GameServers`, and starts up a new set with the new `GameServer` configuration to replace them.
  - `rollingUpdate` is only relevant when `type: RollingUpdate`
    - `maxSurge` is the amount to increment the new GameServers by. Defaults to 25%
    - `maxUnavailable` is the amount to decrements GameServers by. Defaults to 25%
- `template` a full `GameServer` configuration template.
   See the [GameServer](./gameserver_spec.md) reference for all available fields.

# Fleet Allocation Specification

A `FleetAllocation` is used to allocate a `GameServer` out of an existing `Fleet`

A full `Fleet` specification is available below and in the 
[example folder](../examples/fleetallocation.yaml) for reference :

```yaml
apiVersion: "stable.agones.dev/v1alpha1"
kind: FleetAllocation
metadata:
  generateName: fleet-allocation-example-
spec:
  # The name of the fleet to allocate from. Must be an existing Fleet in the same namespace
  # as this FleetAllocation
  fleetName: fleet-example
  # Custom metadata that is added to game server status in the moment of allocation
  # You can use this to tell the server necessary session data
  metadata:
    labels:
      mode: deathmatch
    annotations:
      map:  garden22
```

We recommend using `metadata > generateName`, to declare to Kubernetes that a unique
name for the `FleetAllocation` is generated when the `FleetAllocation` is created.

The `spec` field is the actual `FleetAllocation` specification and it is composed as follow:

- `fleetName` is the name of an existing Fleet. If this doesn't exist, and error will be returned
  when the `FleetAllocation` is created
- `metadata` is an optional list of custom labels and/or annotations that will be used to patch 
  the game server's metadata in the moment of allocation.
