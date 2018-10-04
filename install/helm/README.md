# Install Agones using Helm

This chart install the Agones application and defines deployment on a [Kubernetes](http://kubernetes.io) cluster using the [Helm](https://helm.sh) package manager.

## Prerequisites

- [Helm](https://docs.helm.sh/helm/) package manager 2.8.0+
- Kubernetes 1.9+
- Role-based access controls (RBAC) activated
- MutatingAdmissionWebhook admission controller activated, see [recommendation](https://kubernetes.io/docs/admin/admission-controllers/#is-there-a-recommended-set-of-admission-controllers-to-use)

## Installing the Chart

> If you don't have `Helm` installed locally, or `Tiller` installed in your Kubernetes cluster, read the [Using Helm](https://docs.helm.sh/using_helm/) documentation to get started.

To install the chart with the release name `my-release` using our stable helm repository:

```bash
$ helm repo add agones https://agones.dev/chart/stable
$ helm install --name my-release --namespace agones-system agones/agones
```

_We recommend to install Agones in its own namespaces (like `agones-system` as shown above)
you can use the helm `--namespace` parameter to specify a different namespace._

The command deploys Agones on the Kubernetes cluster with the default configuration. The [configuration](#configuration) section lists the parameters that can be configured during installation.

> **Tip**: List all releases using `helm list`


> If you are installing a development build of Agones (i.e. not the 0.4.0 release), you will need to install Agones the following way:

```bash
$ cd install/helm/
$ helm install --name my-release --namespace agones-system agones --set agones.image.tag=0.4.0-481970d
```

The full list of available tags is [here](https://console.cloud.google.com/gcr/images/agones-images/)

---

## Namespaces

By default Agones is configured to work with game servers deployed in the `default` namespace. If you are planning to use other namespace you can configure Agones via the parameter `gameservers.namespaces`.

For example to use `default` **and** `xbox` namespaces:

```bash
$ kubectl create namespace xbox
$ helm install --set "gameservers.namespaces={default,xbox}" --namespace agones-system --name my-release agones/agones
```

> You need to create your namespaces before installing Agones.

If you want to add a new namespace afterward simply upgrade your release:

```bash
$ kubectl create namespace ps4
$ helm upgrade --set "gameservers.namespaces={default,xbox,ps4}" my-release agones/agones
```

## RBAC

By default, `agones.rbacEnabled` is set to true. This enable RBAC support in Agones and must be true if RBAC is enabled in your cluster.

The chart will take care of creating the required service accounts and roles for Agones.

If you have RBAC disabled, or to put it another way, ABAC enabled, you should set this value to `false`.

## Uninstalling the Chart

To uninstall/delete the `my-release` deployment:

```bash
$ helm delete my-release
```

The command removes all the Kubernetes components associated with the chart and deletes the release.

## Configuration

The following tables lists the configurable parameters of the Agones chart and their default values.

| Parameter                                           | Description                                                                                     | Default                |
| --------------------------------------------------- | ----------------------------------------------------------------------------------------------- | ---------------------- |
| `agones.rbacEnabled`                                | Creates RBAC resources. Must be set for any cluster configured with RBAC                        | `true`                 |
| `agones.serviceaccount.controller`                  | Service account name for the controller                                                         | `agones-controller`    |
| `agones.serviceaccount.sdk`                         | Service account name for the sdk                                                                | `agones-sdk`           |
| `agones.image.registry`                             | Global image registry for all images                                                            | `gcr.io/agones-images` |
| `agones.image.tag`                                  | Global image tag for all images                                                                 | `0.4.0`                |
| `agones.image.controller.name`                      | Image name for the controller                                                                   | `agones-controller`    |
| `agones.image.controller.pullPolicy`                | Image pull policy for the controller                                                            | `IfNotPresent`         |
| `agones.image.controller.pullSecret`                | Image pull secret for the controller                                                            | ``                     |
| `agones.image.sdk.name`                             | Image name for the sdk                                                                          | `agones-sdk`           |
| `agones.image.sdk.alwaysPull`                       | Tells if the sdk image should always be pulled                                                  | `false`                |
| `agones.controller.healthCheck.http.port`           | Port to use for liveness probe service                                                          | `8080`                 |
| `agones.controller.healthCheck.initialDelaySeconds` | Initial delay before performing the first probe (in seconds)                                    | `3`                    |
| `agones.controller.healthCheck.periodSeconds`       | Seconds between every liveness probe (in seconds)                                               | `3`                    |
| `agones.controller.healthCheck.failureThreshold`    | Number of times before giving up (in seconds)                                                   | `3`                    |
| `agones.controller.healthCheck.timeoutSeconds`      | Number of seconds after which the probe times out (in seconds)                                  | `1`                    |
| `agones.controller.resources`                       | Controller resource requests/limit                                                              | `{}`                   |
| `agones.controller.generateTLS`                     | Set to true to generate TLS certificates or false to provide your own certificates in `certs/*` | `true`                 |
| `gameservers.namespaces`                            | a list of namespaces you are planning to use to deploy game servers                             | `["default"]`          |
| `gameservers.minPort`                               | Minimum port to use for dynamic port allocation                                                 | `7000`                 |
| `gameservers.maxPort`                               | Maximum port to use for dynamic port allocation                                                 | `8000`                 |

Specify each parameter using the `--set key=value[,key=value]` argument to `helm install`. For example,

```bash
$ helm install --name my-release --namespace agones-system \
  --set agones.namespace=mynamespace,gameservers.minPort=1000,gameservers.maxPort=5000 agones
```

The above command sets the namespace where Agones is deployed to `mynamespace`. Additionally Agones will use a dynamic port allocation range of 1000-5000.

Alternatively, a YAML file that specifies the values for the parameters can be provided while installing the chart. For example,

```bash
$ helm install --name my-release --namespace agones-system -f values.yaml agones/agones
```

> **Tip**: You can use the default [values.yaml](agones/values.yaml)

## TLS Certificates

By default agones chart generates tls certificates used by the adminission controller, while this is handy, it requires the agones controller to restart on each `helm upgrade` command. 
For most used cases the controller would have required a restart anyway (eg: controller image updated). However if you really need to avoid restarts we suggest that you turn off tls automatic generation (`agones.controller.generateTLS` to `false`) and provide your own certificates (`certs/server.crt`,`certs/server.key`).

> **Tip**: You can use our script located at `cert/cert.sh` to generates them.

## Confirm Agones is running

To confirm Agones is up and running, [go to the next section](../README.md#confirming-agones-started-successfully)
