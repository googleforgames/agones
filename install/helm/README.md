**(This is currently a development feature)**

# Install Agones using Helm

This chart install the Agones application and defines deployment on a [Kubernetes](http://kubernetes.io) cluster using the [Helm](https://helm.sh) package manager.

## Prerequisites

- [Helm](https://docs.helm.sh/helm/) package manager 2.8.0+
- Kubernetes 1.9+
- Role-based access controls (RBAC) activated
- MutatingAdmissionWebhook admission controller activated, see [recommendation](https://kubernetes.io/docs/admin/admission-controllers/#is-there-a-recommended-set-of-admission-controllers-to-use)

## Installing the Chart

> If you don't have `Helm` installed locally, or `Tiller` installed in your Kubernetes cluster, read the [Using Helm](https://docs.helm.sh/using_helm/) documentation to get started.

To install the chart with the release name `my-release`:

```bash
$ cd install/helm/
$ helm install --name my-release agones
```

The command deploys Agones on the Kubernetes cluster with the default configuration. The [configuration](#configuration) section lists the parameters that can be configured during installation.


> **Tip**: List all releases using `helm list`

## Uninstalling the Chart

To uninstall/delete the `my-release` deployment:

```bash
$ helm delete my-release
```

The command removes all the Kubernetes components associated with the chart and deletes the release.

## Configuration

The following tables lists the configurable parameters of the Agones chart and their default values.

| Parameter                            | Description                                                     | Default                    |
| ------------------------------------ | ----------------------------------------------------------------| ---------------------------|
| `namespace`                          | Namespace to use for Agones                                     | `agones-system`            |
| `serviceaccount.controller`          | Service account name for the controller                         | `agones-controller`        |
| `serviceaccount.sdk`                 | Service account name for the sdk                                | `agones-sdk`               |
| `image.registry`                     | Global image registry for all images                            | `gcr.io/agones-images`     |
| `image.tag`                          | Global image tag for all images                                 | `0.2`                      |
| `image.controller.name`              | Image name for the controller                                   | `agones-controller`        |
| `image.controller.pullPolicy`        | Image pull policy for the controller                            | `IfNotPresent`             |
| `image.sdk.name`                     | Image name for the sdk                                          | `agones-sdk`               |
| `image.sdk.alwaysPull`               | Tells if the sdk image should always be pulled                  | `false`                    |
| `minPort`                            | Minimum port to use for dynamic port allocation                 | `7000`                     |
| `maxPort`                            | Maximum port to use for dynamic port allocation                 | `8000`                     |
| `healthCheck.http.port`              | Port to use for liveness probe service                          | `8080`                     |
| `healthCheck.initialDelaySeconds`    | Initial delay before performing the first probe (in seconds)    | `3`                        |
| `healthCheck.periodSeconds`          | Seconds between every liveness probe (in seconds)               | `3`                        |
| `healthCheck.failureThreshold`       | Number of times before giving up (in seconds)                   | `3`                        |
| `healthCheck.timeoutSeconds`         | Number of seconds after which the probe times out (in seconds)  | `1`                        |

Specify each parameter using the `--set key=value[,key=value]` argument to `helm install`. For example,

```bash
$ helm install --name my-release \
  --set namespace=mynamespace,minPort=1000,maxPort=5000 agones
```

The above command sets the namespace where Agones is deployed to `mynamespace`. Additionally Agones will use a dynamic port allocation range of 1000-5000.

Alternatively, a YAML file that specifies the values for the parameters can be provided while installing the chart. For example,

```bash
$ helm install --name my-release -f values.yaml agones
```

> **Tip**: You can use the default [values.yaml](agones/values.yaml)

## Confirm Agones is running

To confirm Agones is up and running, [go to the next section](../../docs/installing_agones.md#confirming-agones-started-successfully)