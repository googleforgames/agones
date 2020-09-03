---
title: "GameServer Specification"
linkTitle: "Gameserver"
date: 2019-01-03T03:58:48Z
weight: 10
description: >
  Like any other Kubernetes resource you describe a GameServer's desired state via a specification written in YAML or JSON to the Kubernetes API. The Agones controller will then change the actual state to the desired state.
---

A full GameServer specification is available below and in the {{< ghlink href="examples/gameserver.yaml" >}}example folder{{< /ghlink >}} for reference :

```yaml
apiVersion: "agones.dev/v1"
kind: GameServer
# GameServer Metadata
# https://v1-16.docs.kubernetes.io/docs/reference/generated/kubernetes-api/v1.16/#objectmeta-v1-meta
metadata:
  # generateName: "gds-example" # generate a unique name, with the given prefix
  name: "gds-example" # set a fixed name
spec:
  # if there is more than one container, specify which one is the game server
  container: example-server
  # Array of ports that can be exposed as direct connections to the game server container
  ports:
    # name is a descriptive name for the port
  - name: default
    # portPolicy has three options:
    # - "Dynamic" (default) the system allocates a free hostPort for the gameserver, for game clients to connect to
    # - "Static", user defines the hostPort that the game client will connect to. Then onus is on the user to ensure that the
    # port is available. When static is the policy specified, `hostPort` is required to be populated
    # - "Passthrough" dynamically sets the `containerPort` to the same value as the dynamically selected hostPort.
    #      This will mean that users will need to lookup what port has been opened through the server side SDK.
    portPolicy: Static
    # [Stage:Beta]
    # [FeatureFlag:ContainerPortAllocation]
    # The name of the container to open the port on. Defaults to the game server container if omitted or empty.
    container: simple-udp
    # the port that is being opened on the game server process
    containerPort: 7654
    # the port exposed on the host, only required when `portPolicy` is "Static". Overwritten when portPolicy is "Dynamic".
    hostPort: 7777
    # protocol being used. Defaults to UDP. TCP is the only other option
    protocol: UDP
  # Health checking for the running game server
  health:
    # Disable health checking. defaults to false, but can be set to true
    disabled: false
    # Number of seconds after the container has started before health check is initiated. Defaults to 5 seconds
    initialDelaySeconds: 5
    # If the `Health()` function doesn't get called at least once every period (seconds), then
    # the game server is not healthy. Defaults to 5
    periodSeconds: 5
    # Minimum consecutive failures for the health probe to be considered failed after having succeeded.
    # Defaults to 3. Minimum value is 1
    failureThreshold: 3
  # Parameters for game server sidecar
  sdkServer:
    # sdkServer log level parameter has three options:
    #  - "Info" (default) The SDK server will output all messages except for debug messages
    #  - "Debug" The SDK server will output all messages including debug messages
    #  - "Error" The SDK server will only output error messages
    logLevel: Info
    # grpcPort and httpPort control what ports the sdkserver listens on.
    # Starting with Agones 1.2 the default grpcPort is 9357 and the default
    # httpPort is 9358. In earlier releases, the defaults were 59357 and 59358
    # respectively but as these were in the ephemeral port range they could
    # conflict with other TCP connections.
    grpcPort: 9357
    httpPort: 9358
  # [Stage:Alpha]
  # [FeatureFlag:PlayerTracking]
  # Players provides the configuration for player tracking features.
  # Commented out since Alpha, and disabled by default
  # players:
  #   # set this GameServer's initial player capacity
  #   initialCapacity: 10
  # Pod template configuration
  # https://v1-16.docs.kubernetes.io/docs/reference/generated/kubernetes-api/v1.16/#podtemplate-v1-core
  template:
    # pod metadata. Name & Namespace is overwritten
    metadata:
      labels:
        myspeciallabel: myspecialvalue
    # Pod Specification
    spec:
      containers:
      - name: simple-udp
        image:  gcr.io/agones-images/udp-server:0.21
        imagePullPolicy: Always
```

Since Agones defines a new [Custom Resources Definition (CRD)](https://kubernetes.io/docs/concepts/api-extension/custom-resources/) we can define a new resource using the kind `GameServer` with the custom group `agones.dev` and API version `v1`.

You can use the metadata field to target a specific [namespaces](https://kubernetes.io/docs/concepts/overview/working-with-objects/namespaces/) 
but also attach specific [annotations](https://kubernetes.io/docs/concepts/overview/working-with-objects/annotations/) and [labels](https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/) to your resource. This is a very common pattern in the Kubernetes ecosystem.

The length of the `name` field of the Gameserver should not exceed 63 characters.

The `spec` field is the actual GameServer specification and it is composed as follow:

- `container` is the name of container running the GameServer in case you have more than one container defined in the [pod](https://kubernetes.io/docs/concepts/workloads/pods/pod-overview/). If you do,  this is a mandatory field. For instance this is useful if you want to run a sidecar to ship logs.
- `ports` are an array of ports that can be exposed as direct connections to the game server container
  - `name` is an optional descriptive name for a port
  - `portPolicy` has three options:
        - `Dynamic` (default) the system allocates a random free hostPort for the gameserver, for game clients to connect to.
        - `Static`, user defines the hostPort that the game client will connect to. Then onus is on the user to ensure that the port is available. When static is the policy specified, `hostPort` is required to be populated.
        - `Passthrough` dynamically sets the `containerPort`  to the same value as the dynamically selected hostPort. This will mean that users will need to lookup what port to open through the server side SDK before starting communications.
  - `container` (Alpha) the name of the container to open the port on. Defaults to the game server container if omitted or empty.
  - `containerPort` the port that is being opened on the game server process, this is a required field for `Dynamic` and `Static` port policies, and should not be included in <code>Passthrough</code> configuration.
  - `protocol` the protocol being used. Defaults to UDP. TCP is the only other option.
- `health` to track the overall healthy state of the GameServer, more information available in the [health check documentation]({{< relref "../Guides/health-checking.md" >}}).
- `sdkServer` defines parameters for the game server sidecar
  - `logging` field defines log level for SDK server. Defaults to "Info". It has three options:
    - "Info" (default) The SDK server will output all messages except for debug messages
    - "Debug" The SDK server will output all messages including debug messages
    - "Error" The SDK server will only output error messages
  - `grpcPort` the port that the SDK Server binds to for gRPC connections
  - `httpPort` the port that the SDK Server binds to for HTTP gRPC gateway connections
- `players` (Alpha, behind "PlayerTracking" feature gate), sets this GameServer's initial player capacity
- `template` the [pod spec template](https://v1-16.docs.kubernetes.io/docs/reference/generated/kubernetes-api/v1.16/#podtemplatespec-v1-core) to run your GameServer containers, [see](https://kubernetes.io/docs/concepts/workloads/pods/pod-overview/#pod-templates) for more information.

## GameServer State Diagram

The following diagram shows the lifecycle of a `GameServer`. 

Game Servers are created through Kubernetes API (either directly or through a [Fleet]({{< ref "fleet.md" >}})) and their state transitions are orchestrated by:

- GameServer controller, which allocates ports, launches Pods backing game servers and manages their lifetime
- Allocation controller, which marks game servers as `Allocated` to handle a game session
- SDK, which manages health checking and shutdown of a game server session

![GameServer State Diagram](../../../diagrams/gameserver-states.dot.png)
