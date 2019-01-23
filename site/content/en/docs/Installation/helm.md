---
title: "Install Agones using Helm"
linkTitle: "Install with Helm"
weight: 4
description: >
  This chart install the Agones application and defines deployment on a [Kubernetes](http://kubernetes.io) cluster using the [Helm](https://helm.sh) package manager.

---

## Prerequisites

- [Helm](https://docs.helm.sh/helm/) package manager 2.10.0+
- Kubernetes 1.11+
- Role-based access controls (RBAC) activated
- MutatingAdmissionWebhook and ValidatingAdmissionWebhook admission controllers activated, see [recommendation](https://kubernetes.io/docs/admin/admission-controllers/#is-there-a-recommended-set-of-admission-controllers-to-use)

## Installing the Chart

> If you don't have `Helm` installed locally, or `Tiller` installed in your Kubernetes cluster, read the [Using Helm](https://docs.helm.sh/using_helm/) documentation to get started.

To install the chart with the release name `my-release` using our stable helm repository:

```bash
$ helm repo add agones https://agones.dev/chart/stable
$ helm install --name my-release --namespace agones-system agones/agones
```

_We recommend to install Agones in its own namespaces (like `agones-system` as shown above)
you can use the helm `--namespace` parameter to specify a different namespace._

{{% feature publishVersion="0.8.0" %}}

When running in production, Agones should be scheduled on a dedicated pool of nodes, distinct from where Game Servers are scheduled for better isolation and resiliency. By default Agones prefers to be scheduled on nodes labeled with `stable.agones.dev/agones-system=true` and tolerates node taint `stable.agones.dev/agones-system=true:NoExecute`. If no dedicated nodes are available, Agones will
run on regular nodes, but that's not recommended for production use.

As an example, to set up dedicated node pool for Agones on GKE, run the following command before installing Agones. Alternatively you can taint and label nodes manually.

 ```
gcloud container node-pools create agones-system --cluster=... --zone=... \
  --node-taints stable.agones.dev/agones-system=true:NoExecute \
  --node-labels stable.agones.dev/agones-system=true \
  --num-nodes=1
```

{{% /feature %}}

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
| `agones.crds.install`                               | Install the CRDs with this chart. Useful to disable if you want to subchart (since crd-install hook is broken), so you can copy the CRDs into your own chart. | `true` |
| `agones.crds.cleanupOnDelete`                       | Run the pre-delete hook to delete all GameServers and their backing Pods when deleting the helm chart, so that all CRDs can be removed on chart deletion | `true`          |  
| `agones.metrics.enabled`                            | Enables controller metrics on port `8080` and path `/metrics`                                   | `true`                 |
| `agones.metrics.prometheusServiceDiscovery`         | Adds annotations for Prometheus ServiceDiscovery (and also Strackdriver)                        | `true`                 |
| `agones.serviceaccount.controller`                  | Service account name for the controller                                                         | `agones-controller`    |
| `agones.serviceaccount.sdk`                         | Service account name for the sdk                                                                | `agones-sdk`           |
| `agones.image.registry`                             | Global image registry for all images                                                            | `gcr.io/agones-images` |
| `agones.image.tag`                                  | Global image tag for all images                                                                 | `0.4.0`                |
| `agones.image.controller.name`                      | Image name for the controller                                                                   | `agones-controller`    |
| `agones.image.controller.pullPolicy`                | Image pull policy for the controller                                                            | `IfNotPresent`         |
| `agones.image.controller.pullSecret`                | Image pull secret for the controller                                                            | ``                     |
| `agones.image.sdk.name`                             | Image name for the sdk                                                                          | `agones-sdk`           |
| `agones.image.sdk.cpuRequest`                       | The [cpu request][constraints] for sdk server container                                         | `30m`                  |
| `agones.image.sdk.cpuLimit`                         | The [cpu limit][constraints] for the sdk server container                                       | `0` (none)             |
| `agones.image.sdk.alwaysPull`                       | Tells if the sdk image should always be pulled                                                  | `false`                |
| `agones.image.ping.name`                            | Image name for the ping service                                                                 | `agones-ping`          |
| `agones.image.ping.pullPolicy`                      | Image pull policy for the ping service                                                          | `IfNotPresent`         |
| `agones.controller.http.port`                       | Port to use for liveness probe service and metrics                                              | `8080`                 |
| `agones.controller.healthCheck.initialDelaySeconds` | Initial delay before performing the first probe (in seconds)                                    | `3`                    |
| `agones.controller.healthCheck.periodSeconds`       | Seconds between every liveness probe (in seconds)                                               | `3`                    |
| `agones.controller.healthCheck.failureThreshold`    | Number of times before giving up (in seconds)                                                   | `3`                    |
| `agones.controller.healthCheck.timeoutSeconds`      | Number of seconds after which the probe times out (in seconds)                                  | `1`                    |
| `agones.controller.resources`                       | Controller resource requests/limit                                                              | `{}`                   |
| `agones.controller.generateTLS`                     | Set to true to generate TLS certificates or false to provide your own certificates in `certs/*` | `true`                 |
| `agones.ping.install`                               | Whether to install the [ping service][ping]                                                     | `true`                 |
| `agones.ping.replicas`                              | The number of replicas to run in the deployment                                                 | `2`                    | 
| `agones.ping.http.expose`                           | Expose the http ping service via a Service                                                      | `true`                 | 
| `agones.ping.http.response`                         | The string response returned from the http service                                              | `ok`                   | 
| `agones.ping.http.port`                             | The port to expose on the service                                                               | `80`                   |
| `agones.ping.http.serviceType`                      | The [Service Type][service] of the HTTP Service                                                 | `LoadBalancer`         |
| `agones.ping.udp.expose`                            | Expose the udp ping service via a Service                                                       | `true`                 | 
| `agones.ping.udp.rateLimit`                         | Number of UDP packets the ping service handles per instance, per second, per sender             | `20`                   | 
| `agones.ping.udp.port`                              | The port to expose on the service                                                               | `80`                   |
| `agones.ping.udp.serviceType`                       | The [Service Type][service] of the UDP Service                                                  | `LoadBalancer`         |
| `agones.ping.healthCheck.initialDelaySeconds`       | Initial delay before performing the first probe (in seconds)                                    | `3`                    |
| `agones.ping.healthCheck.periodSeconds`             | Seconds between every liveness probe (in seconds)                                               | `3`                    |
| `agones.ping.healthCheck.failureThreshold`          | Number of times before giving up (in seconds)                                                   | `3`                    |
| `agones.ping.healthCheck.timeoutSeconds`            | Number of seconds after which the probe times out (in seconds)                                  | `1`                    |
| `gameservers.namespaces`                            | a list of namespaces you are planning to use to deploy game servers                             | `["default"]`          |
| `gameservers.minPort`                               | Minimum port to use for dynamic port allocation                                                 | `7000`                 |
| `gameservers.maxPort`                               | Maximum port to use for dynamic port allocation                                                 | `8000`                 |

{{% feature publishVersion="0.8.0" %}}
**New Configuration Features:**
 
| Parameter                                           | Description                                                                                     | Default                |
| --------------------------------------------------- | ----------------------------------------------------------------------------------------------- | ---------------------- |
| `agones.controller.nodeSelector`                    | Controller [node labels](nodeSelector) for pod assignment                                       | `{}`                   |
| `agones.controller.tolerations`                     | Controller [toleration][toleration] labels for pod assignment                                   | `[]`                   |
| `agones.controller.affinity`                        | Controller [affinity](affinity) settings for pod assignment                                     | `{}`                   |
| `agones.ping.resources`                             | Ping pods resource requests/limit                                                               | `{}`                   |
| `agones.ping.nodeSelector`                          | Ping [node labels](nodeSelector) for pod assignment                                             | `{}`                   |
| `agones.ping.tolerations`                           | Ping [toleration][toleration] labels for pod assignment                                         | `[]`                   |
| `agones.ping.affinity`                              | Ping [affinity](affinity) settings for pod assignment                                           | `{}`                   |
[toleration]: https://kubernetes.io/docs/concepts/configuration/taint-and-toleration/
{{% /feature %}}

[constraints]: https://kubernetes.io/docs/tasks/administer-cluster/manage-resources/cpu-constraint-namespace/
[ping]: {{< ref "/docs/Guides/ping-service.md" >}}
[service]: https://kubernetes.io/docs/concepts/services-networking/service/
[nodeSelector]: https://kubernetes.io/docs/concepts/configuration/assign-pod-node/#nodeselector
[affinity]: https://kubernetes.io/docs/concepts/configuration/assign-pod-node/#affinity-and-anti-affinity

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

> **Tip**: You can use the default {{< ghlink href="install/helm/agones/values.yaml" >}}values.yaml{{< /ghlink >}}

## TLS Certificates

By default agones chart generates tls certificates used by the adminission controller, while this is handy, it requires the agones controller to restart on each `helm upgrade` command. 
For most used cases the controller would have required a restart anyway (eg: controller image updated). However if you really need to avoid restarts we suggest that you turn off tls automatic generation (`agones.controller.generateTLS` to `false`) and provide your own certificates (`certs/server.crt`,`certs/server.key`).

> **Tip**: You can use our script located at `cert/cert.sh` to generates them.

## Confirm Agones is running

To confirm Agones is up and running, [go to the next section]({{< relref "_index.md#confirming-agones-started-successfully" >}})
