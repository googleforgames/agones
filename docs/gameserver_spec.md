# GameServer Specification

Like any other Kubernetes resource you describe a GameServer's desired state via a specification written in YAML or JSON to the Kubernetes API. The Agones controller will then change the actual state to the desired state.

A full GameServer specification is available below and in the [example folder](https://github.com/GoogleCloudPlatform/agones/blob/release-0.5.0/examples/gameserver.yaml) for reference :

```
apiVersion: "stable.agones.dev/v1alpha1"
kind: GameServer
metadata:
  name: "gds-example"
spec:
  ports:
  - name: default
    portPolicy: "static"
    containerPort: 7654
    hostPort: 7777
    protocol: UDP
  health:
    disabled: false
    initialDelaySeconds: 5
    periodSeconds: 5
    failureThreshold: 3
  template:
    metadata:
      labels:
        myspeciallabel: myspecialvalue
    spec:
      containers:
      - name: example-server
        image: gcr.io/agones/test-server:0.1
        imagePullPolicy: Always
```

Since Agones defines a new [Custom Resources Definition (CRD)](https://kubernetes.io/docs/concepts/api-extension/custom-resources/) we can define a new resource using the kind `GameServer` with the custom group `stable.agones.dev` and API version `v1alpha1`.

You can use the metadata field to target a specific [namespaces](https://kubernetes.io/docs/concepts/overview/working-with-objects/namespaces/) but also attach specific [annotations](https://kubernetes.io/docs/concepts/overview/working-with-objects/annotations/) and [labels](https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/) to your ressource. This is a very common pattern in the Kubernetes ecosystem.

The `spec` field is the actual GameServer specification and it is composed as follow:

- `container` is the name of container running the GameServer in case you have more than one container defined in the [pod](https://kubernetes.io/docs/concepts/workloads/pods/pod-overview/). If you do,  this is a mandatory field. For instance this is useful if you want to run a sidecar to ship logs.
- `ports` are an array of ports that can be exposed as direct connections to the game server container
  - `name` is an optional descriptive name for a port
  - `portPolicy` has two options `dynamic` (default) the system allocates a free hostPort for the gameserver, for game clients to connect to. And `static`, user defines the hostPort that the game client will connect to. Then onus is on the user to ensure that the port is available. When static is the policy specified, `hostPort` is required to be populated.
  - `containerPort` the port that is being opened on the game server process, this is a required field.
  - `protocol` the protocol being used. Defaults to UDP. TCP is the only other option.
- `health` to track the overall healthy state of the GameServer, more information available in the [health check documentation](./health_checking.md).
- `template` the [pod spec template](https://v1-10.docs.kubernetes.io/docs/reference/generated/kubernetes-api/v1.10/#podtemplatespec-v1-core) to run your GameServer containers, [see](https://kubernetes.io/docs/concepts/workloads/pods/pod-overview/#pod-templates) for more information.
