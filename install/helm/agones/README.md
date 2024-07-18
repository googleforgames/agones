# Install Agones using Helm

This chart installs the Agones application and defines deployment on a [Kubernetes](http://kubernetes.io) cluster using the [Helm](https://helm.sh) package manager.

See [Install Agones using Helm](https://agones.dev/site/docs/installation/install-agones/helm/) for installation and configuration instructions.

## Development Work on Agones Custom Resource Definitions (CRDs)

Agones [extends the Kubernetes API with CustomResourceDefinitions](https://kubernetes.io/docs/tasks/extend-kubernetes/custom-resources/custom-resource-definitions/) for the kinds `Fleet`, `GameServerSet`, `GameServer`, `FleetAutoscaler`. (`GameServerAllocationPolicy` has been depcrecated in favor of `GameServerAllocation` which does not a have CRD.) Regardless of installation method, the definitions for the Agones Custom Resources are in the [agones/install/helm/agones/templates/crds](./templates/crds/) directory. The double braces `{{ }}` in the CRDs and elsewhere are [Helm chart template](https://helm.sh/docs/chart_template_guide/getting_started/) features.

### Adding a New Field to a CRD

> [!IMPORTANT]
>
> Any new *required* field in a CRD must be non-nullable **and** have a default. We define a field as required if the controller throws a panic or error when it encounters a `nil` value for that field. Most new fields should be `nullable: true`, and the controller should be able to handle a `nil` value without a panic or error.
>
> This ensures that after an Agones upgrade to a version that introduces a new field, the upgraded controller does not panic or error when encountering an older custom resource that was created before the Agones upgrade.

```yaml
      foo:
        title: Example required CRD field. Non-nullable with default.
        type: string
        enum:
        - foo1
        - foo2
        - foo3
        default: foo3
      bar:
        title: Example non-required CRD field. Nullable with optional default.
        type: object
        nullable: true
        properties:
          barCapacity:
            type: integer
            minimum: 0
            default: 99 # Default for a nullable field is optional
```

Once you have added a new field to a CRD, generate the values for the `install.yaml` file by running `~/agones/build$ make gen-install`. This ensure that the `yaml` installation methods has the same values as the preferred Helm installation method.

If the above example fields were added, for example, to the [_gameserverspecschema.yaml](./templates/crds/_gameserverspecschema.yaml), then the fields should also be added to the `GameServerSpec` struct in [gameserver.go](../../../pkg/apis/agones/v1/gameserver.go).

```go
type GameServerSpec struct {
...
	Foo     apis.Foos    `json:"foo"`
	Bar     *apis.Bars   `json:"bar,omitempty"`
...
}

const (
	// Foo1 enum value for testing CRD defaulting
	Foo1 Foos = "foo1"
	// Foo2 enum value for testing CRD defaulting
	Foo2 Foos = "foo2"
	// Foo3 enum value for testing CRD defaulting
	Foo3 Foos = "foo3"
)

// Foos enum values for testing CRD defaulting
type Foos string

// Bars tracks the initial player capacity
type Bars struct {
	BarCapacity int64 `json:"barCapacity,omitempty"`
}
```

### Removing an Existing Field From a CRD

Keep in mind that there can only be one definition of a `kind` in the `apiVersion: agones.dev/v1` in a Kubernetes cluster. This means that change to a CRD during an upgrade, downgrade, or Feature Gate change of Agones is immediately applied to custom resources in the cluster. Breaking changes to fields may be covered under our [SDK deprecation policy](../../../site/content/en/docs/Installation/upgrading.md).
