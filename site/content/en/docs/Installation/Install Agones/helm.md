---
title: "Install Agones using Helm"
linkTitle: "Helm"
weight: 20
description: >
  Install Agones on a [Kubernetes](http://kubernetes.io) cluster using the [Helm](https://helm.sh) package manager.
---

## Prerequisites

- [Helm](https://helm.sh/) package manager 3.2.3+
- [Supported Kubernetes Cluster]({{< relref "../_index.md#usage-requirements" >}})

## Helm 3

### Installing the Chart

To install the chart with the release name `my-release` using our stable helm repository:

```bash
helm repo add agones https://agones.dev/chart/stable
helm repo update
helm install my-release --namespace agones-system --create-namespace agones/agones
```

_We recommend installing Agones in its own namespaces, such as `agones-system` as shown above.
 If you want to use a different namespace, you can use the helm `--namespace` parameter to specify._

When running in production, Agones should be scheduled on a dedicated pool of nodes, distinct from where Game Servers are scheduled for better isolation and resiliency. By default Agones prefers to be scheduled on nodes labeled with `agones.dev/agones-system=true` and tolerates node taint `agones.dev/agones-system=true:NoExecute`. If no dedicated nodes are available, Agones will
run on regular nodes, but that's not recommended for production use. For instructions on setting up a dedicated node
pool for Agones, see the [Agones installation instructions]({{< relref "../_index.md" >}}) for your preferred environment.

The command deploys Agones on the Kubernetes cluster with the default configuration. The [configuration](#configuration) section lists the parameters that can be configured during installation.

{{< alert title="Tip" color="info">}}
List all releases using `helm list --all-namespaces`
{{< /alert >}}

### Namespaces

By default Agones is configured to work with game servers deployed in the `default` namespace. If you are planning to use another namespace you can configure Agones via the parameter `gameservers.namespaces`.

For example to use `default` **and** `xbox` namespaces:

```bash
kubectl create namespace xbox
helm install my-release agones/agones --set "gameservers.namespaces={default,xbox}" --namespace agones-system
```

{{< alert title="Note" color="info">}}
You need to create your namespaces before installing Agones.
{{< /alert >}}

If you want to add a new namespace afterward upgrade your release:

```bash
kubectl create namespace ps4
helm upgrade my-release agones/agones --reuse-values --set "gameservers.namespaces={default,xbox,ps4}" --namespace agones-system
```

### Uninstalling the Chart

To uninstall/delete the `my-release` deployment:

```bash
helm uninstall my-release --namespace=agones-system
```

## RBAC

By default, `agones.rbacEnabled` is set to true. This enables RBAC support in Agones and must be true if RBAC is enabled in your cluster.

The chart will take care of creating the required service accounts and roles for Agones.

If you have RBAC disabled, or to put it another way, ABAC enabled, you should set this value to `false`.

## Configuration

The following tables lists the configurable parameters of the Agones chart and their default values.

| Parameter                                           | Description                                                                                     | Default                |
| --------------------------------------------------- | ----------------------------------------------------------------------------------------------- | ---------------------- |
| `agones.rbacEnabled`                                | Creates RBAC resources. Must be set for any cluster configured with RBAC                        | `true`                 |
| `agones.registerWebhooks`                           | Registers the webhooks used for the admission controller                                        | `true`                 |
| `agones.registerApiService`                         | Registers the apiservice(s) used for the Kubernetes API extension                               | `true`                 |
| `agones.registerServiceAccounts`                    | Attempts to create service accounts for the controllers                                         | `true`                 |
| `agones.createPriorityClass`                        | Attempts to create priority classes for the controllers                                         | `true`                 |
| `agones.priorityClassName`                          | Name of the priority classes to create                                                          | `agones-system`        |
| `agones.featureGates`                               | A URL query encoded string of Flags to enable/disable e.g. `Example=true&OtherThing=false`. Any value accepted by [strconv.ParseBool(string)](https://golang.org/pkg/strconv/#ParseBool) can be used as a boolean value | \`\` |
| `agones.crds.install`                               | Install the CRDs with this chart. Useful to disable if you want to subchart (since crd-install hook is broken), so you can copy the CRDs into your own chart. | `true` |
| `agones.crds.cleanupOnDelete`                       | Run the pre-delete hook to delete all GameServers and their backing Pods when deleting the helm chart, so that all CRDs can be removed on chart deletion | `true`          |
| `agones.metrics.prometheusServiceDiscovery`         | Adds annotations for Prometheus ServiceDiscovery (and also Strackdriver)                        | `true`                 |
| `agones.metrics.prometheusEnabled`                  | Enables controller metrics on port `8080` and path `/metrics`                                   | `true`                 |
| `agones.metrics.stackdriverEnabled`                 | Enables Stackdriver exporter of controller metrics                                              | `false`                |
| `agones.metrics.stackdriverProjectID`               | This overrides the default gcp project id for use with stackdriver                              | \`\`                   |
| `agones.metrics.stackdriverLabels`                  | A set of default labels to add to all stackdriver metrics generated in form of key value pair (`key=value,key2=value2`). By default metadata are automatically added using Kubernetes API and GCP metadata enpoint.                              | \`\` |
| `agones.serviceaccount.controller`                  | Service account name for the controller. **Note**: Will be replaced with `agones.serviceaccount.controller.name` in Agones 1.16 | `agones-controller`    |
| `agones.serviceaccount.sdk`                         | Service account name for the sdk. **Note**: Will be replaced with `agones.serviceaccount.sdk.name` in Agones 1.16        | `agones-sdk`           |
| `agones.image.registry`                             | Global image registry for all images                                                            | `gcr.io/agones-images` |
| `agones.image.tag`                                  | Global image tag for all images                                                                 | `{{< release-version >}}` |
| `agones.image.controller.name`                      | Image name for the controller                                                                   | `agones-controller`    |
| `agones.image.controller.pullPolicy`                | Image pull policy for the controller                                                            | `IfNotPresent`         |
| `agones.image.controller.pullSecret`                | Image pull secret for the controller, allocator, sdk and ping image. Should be created both in `agones-system` and `default` namespaces | \`\`                     |
| `agones.image.sdk.name`                             | Image name for the sdk                                                                          | `agones-sdk`           |
| `agones.image.sdk.tag`                              | Image tag for the sdk                                                                           | value of `agones.image.tag`  |
| `agones.image.sdk.cpuRequest`                       | The [cpu request][cpu-constraints] for sdk server container                                     | `30m`                  |
| `agones.image.sdk.cpuLimit`                         | The [cpu limit][cpu-constraints] for the sdk server container                                   | `0` (none)             |
| `agones.image.sdk.memoryRequest`                    | The [memory request][memory-constraints] for sdk server container                               | `0` (none)             |
| `agones.image.sdk.memoryLimit`                      | The [memory limit][memory-constraints] for the sdk server container                             | `0` (none)             |
| `agones.image.sdk.alwaysPull`                       | Tells if the sdk image should always be pulled                                                  | `false`                |
| `agones.image.ping.name`                            | Image name for the ping service                                                                 | `agones-ping`          |
| `agones.image.ping.tag`                             | Image tag for the ping service                                                                  | value of `agones.image.tag` |
| `agones.image.ping.pullPolicy`                      | Image pull policy for the ping service                                                          | `IfNotPresent`         |
| `agones.controller.http.port`                       | Port to use for liveness probe service and metrics                                              | `8080`                 |
| `agones.controller.healthCheck.initialDelaySeconds` | Initial delay before performing the first probe (in seconds)                                    | `3`                    |
| `agones.controller.healthCheck.periodSeconds`       | Seconds between every liveness probe (in seconds)                                               | `3`                    |
| `agones.controller.healthCheck.failureThreshold`    | Number of times before giving up (in seconds)                                                   | `3`                    |
| `agones.controller.healthCheck.timeoutSeconds`      | Number of seconds after which the probe times out (in seconds)                                  | `1`                    |
| `agones.controller.resources`                       | Controller [resource requests/limit][resources]                                                 | `{}`                   |
| `agones.controller.generateTLS`                     | Set to true to generate TLS certificates or false to provide your own certificates in `certs/*` | `true`                 |
| `agones.controller.nodeSelector`                    | Controller [node labels][nodeSelector] for pod assignment                                       | `{}`                   |
| `agones.controller.tolerations`                     | Controller [toleration][toleration] labels for pod assignment                                   | `[]`                   |
| `agones.controller.affinity`                        | Controller [affinity][affinity] settings for pod assignment                                     | `{}`                   |
| `agones.controller.annotations`                     | [Annotations][annotations] added to the Agones controller pods                                  | `{}`                   |
| `agones.controller.numWorkers`                      | Number of workers to spin per resource type                                                     | `100`                  |
| `agones.controller.apiServerQPS`                    | Maximum sustained queries per second that controller should be making against API Server        | `400`                  |
| `agones.controller.apiServerQPSBurst`               | Maximum burst queries per second that controller should be making against API Server            | `500`                  |
| `agones.controller.logLevel`                        | Agones Controller Log level. Log only entries with that severity and above                      | `info`                 |
| `agones.controller.persistentLogs`                  | Store Agones controller logs in a temporary volume attached to a container for debugging        | `true`                 |
| `agones.controller.persistentLogsSizeLimitMB`       | Maximum total size of all Agones container logs in MB                                           | `10000`                |
| `agones.ping.install`                               | Whether to install the [ping service][ping]                                                     | `true`                 |
| `agones.ping.replicas`                              | The number of replicas to run in the deployment                                                 | `2`                    |
| `agones.ping.http.expose`                           | Expose the http ping service via a Service                                                      | `true`                 |
| `agones.ping.http.response`                         | The string response returned from the http service                                              | `ok`                   |
| `agones.ping.http.port`                             | The port to expose on the service                                                               | `80`                   |
| `agones.ping.http.serviceType`                      | The [Service Type][service] of the HTTP Service                                                 | `LoadBalancer`         |
| `agones.ping.http.loadBalancerIP`                   | The [Load Balancer IP][loadBalancer] of the HTTP Service load balancer. Only works if the Kubernetes provider supports this option.               | \`\`           |
| `agones.ping.http.loadBalancerSourceRanges`         | The [Load Balancer SourceRanges][loadBalancer] of the HTTP Service load balancer. Only works if the Kubernetes provider supports this option.     | `[]`         |
| `agones.ping.http.annotations`                      | [Annotations][annotations] added to the Agones ping http service                                | `{}`                   |
| `agones.ping.udp.expose`                            | Expose the udp ping service via a Service                                                       | `true`                 |
| `agones.ping.udp.rateLimit`                         | Number of UDP packets the ping service handles per instance, per second, per sender             | `20`                   |
| `agones.ping.udp.port`                              | The port to expose on the service                                                               | `80`                   |
| `agones.ping.udp.serviceType`                       | The [Service Type][service] of the UDP Service                                                  | `LoadBalancer`         |
| `agones.ping.udp.loadBalancerIP`                    | The [Load Balancer IP][loadBalancer] of the UDP Service load balancer. Only works if the Kubernetes provider supports this option.                | \`\`           |
| `agones.ping.udp.loadBalancerSourceRanges`          | The [Load Balancer SourceRanges][loadBalancer] of the UDP Service load balancer. Only works if the Kubernetes provider supports this option.      | `[]`         |
| `agones.ping.udp.annotations`                       | [Annotations][annotations] added to the Agones ping udp service                                 | `{}`                   |
| `agones.ping.healthCheck.initialDelaySeconds`       | Initial delay before performing the first probe (in seconds)                                    | `3`                    |
| `agones.ping.healthCheck.periodSeconds`             | Seconds between every liveness probe (in seconds)                                               | `3`                    |
| `agones.ping.healthCheck.failureThreshold`          | Number of times before giving up (in seconds)                                                   | `3`                    |
| `agones.ping.healthCheck.timeoutSeconds`            | Number of seconds after which the probe times out (in seconds)                                  | `1`                    |
| `agones.ping.resources`                             | Ping pods [resource requests/limit][resources]                                                  | `{}`                   |
| `agones.ping.nodeSelector`                          | Ping [node labels][nodeSelector] for pod assignment                                             | `{}`                   |
| `agones.ping.tolerations`                           | Ping [toleration][toleration] labels for pod assignment                                         | `[]`                   |
| `agones.ping.affinity`                              | Ping [affinity][affinity] settings for pod assignment                                           | `{}`                   |
| `agones.ping.annotations`                           | [Annotations][annotations] added to the Agones ping pods                                        | `{}`                   |
| `agones.allocator.apiServerQPS`                     | Maximum sustained queries per second that an allocator should be making against API Server      | `400`                  |
| `agones.allocator.apiServerQPSBurst`                | Maximum burst queries per second that an allocator should be making against API Server          | `500`                  |
| `agones.allocator.allocationTimeout`                | Remote allocation call timeout.                                                                 | `10s`                  |
| `agones.allocator.remoteAllocationTimeout`          | Total remote allocation timeout including retries.                                              | `30s`                  |
| `agones.allocator.logLevel`                         | Agones Allocator Log level. Log only entries with that severity and above                       | `info`                 |
| `agones.allocator.install`                          | Whether to install the [allocator service][allocator]                                           | `true`                 |
| `agones.allocator.replicas`                         | The number of replicas to run in the deployment                                                 | `3`                    |
| `agones.allocator.http.port`                        | The port to expose on the service                                                               | `443`                  |
| `agones.allocator.http.serviceType`                 | The [Service Type][service] of the HTTP Service                                                 | `LoadBalancer`         |
| `agones.allocator.http.loadBalancerIP`              | The [Load Balancer IP][loadBalancer] of the Agones allocator load balancer. Only works if the Kubernetes provider supports this option. | \`\`                     |
| `agones.allocator.http.loadBalancerSourceRanges`    | The [Load Balancer SourceRanges][loadBalancer] of the Agones allocator load balancer. Only works if the Kubernetes provider supports this option. | `[]`         |
| `agones.allocator.generateClientTLS`                | Set to true to generate client TLS certificates or false to provide certificates in `certs/allocator/allocator-client.default/*` | `true`                 |
| `agones.allocator.generateTLS`                      | Set to true to generate TLS certificates or false to provide certificates in `certs/allocator/*`| `true`                 |
| `agones.allocator.disableMTLS`                      | Turns off client cert authentication for incoming connections to the allocator.            | `false`                |
| `agones.allocator.disableTLS`                       | Turns off TLS security for incoming connections to the allocator. Only applicable to the REST API. It currently does not work for the gRPC API. ([issue](https://github.com/googleforgames/agones/issues/1945)) | `false`                |
| `agones.allocator.disableSecretCreation`            | Disables the creation of any allocator secrets. If true, you MUST provide the `allocator-tls`, `allocator-tls-ca`, and `allocator-client-ca` secrets before installation. | `false` |
| `agones.allocator.tolerations`                      | Allocator [toleration][toleration] labels for pod assignment                                    | `[]`                   |
| `agones.allocator.affinity`                         | Allocator [affinity][affinity] settings for pod assignment                                      | `{}`                   |
| `agones.allocator.annotations`                      | [Annotations][annotations] added to the Agones allocator pods                                   | `{}`                   |
| `agones.allocator.resources`                        | Allocator pods [resource requests/limit][resources]                                             | `{}`                   |
| `agones.allocator.nodeSelector`                     | Allocator [node labels][nodeSelector] for pod assignment                                        | `{}`                   |
| `agones.serviceaccount.controller.name`             | Service account name for the controller                                                         | `agones-controller`    |
| `agones.serviceaccount.sdk.name`                    | Service account name for the sdk                                                                | `agones-sdk`           |
| `agones.serviceaccount.allocator.name`              | Service account name for the allocator                                                          | `agones-allocator`    |
| `agones.serviceaccount.allocator.annotations`       | [Annotations][annotations] added to the Agones allocator service account                        | `{}`                   |
| `agones.serviceaccount.controller.annotations`      | [Annotations][annotations] added to the Agones controller service account                       | `{}`                   |
| `gameservers.namespaces`                            | a list of namespaces you are planning to use to deploy game servers                             | `["default"]`          |
| `gameservers.minPort`                               | Minimum port to use for dynamic port allocation                                                 | `7000`                 |
| `gameservers.maxPort`                               | Maximum port to use for dynamic port allocation                                                 | `8000`                 |
| `gameservers.podPreserveUnknownFields`              | Disable [field pruning][pruning] and schema validation on the Pod template for a [GameServer][gameserver] definition | `false`                |
| `helm.installTests`                                 | Add an ability to run `helm test agones` to verify the installation                             | `8000`                 |

{{% feature publishVersion="1.17.0" %}}
**New Configuration Features:**

| Parameter                                           | Description                                                                                     | Default                |
| --------------------------------------------------- | ----------------------------------------------------------------------------------------------- | ---------------------- |
|                                                     |                                                                                                 |         |
|                       |                           |                            |
{{% /feature %}}

[toleration]: https://kubernetes.io/docs/concepts/configuration/taint-and-toleration/
[nodeSelector]: https://kubernetes.io/docs/concepts/configuration/assign-pod-node/#nodeselector
[affinity]: https://kubernetes.io/docs/concepts/configuration/assign-pod-node/#affinity-and-anti-affinity
[cpu-constraints]: https://kubernetes.io/docs/tasks/administer-cluster/manage-resources/cpu-constraint-namespace/
[memory-constraints]: https://kubernetes.io/docs/tasks/administer-cluster/manage-resources/memory-constraint-namespace/
[ping]: {{< ref "/docs/Guides/ping-service.md" >}}
[service]: https://kubernetes.io/docs/concepts/services-networking/service/
[allocator]: {{< ref "/docs/advanced/allocator-service.md" >}}
[loadBalancer]: https://kubernetes.io/docs/concepts/services-networking/service/#loadbalancer
[annotations]: https://kubernetes.io/docs/concepts/overview/working-with-objects/annotations/
[resources]: https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/
[pruning]: https://kubernetes.io/docs/tasks/extend-kubernetes/custom-resources/custom-resource-definitions/#field-pruning
[gameserver]: {{< ref "/docs/Reference/gameserver.md" >}}

Specify each parameter using the `--set key=value[,key=value]` argument to `helm install`. For example,

```bash
helm install my-release --namespace agones-system \
  --set gameservers.minPort=1000,gameservers.maxPort=5000 agones
```

The above command will deploy Agones controllers to `agones-system` namespace. Additionally Agones will use a dynamic GameServers' port allocation range of 1000-5000.

Alternatively, a YAML file that specifies the values for the parameters can be provided while installing the chart. For example,

```bash
helm install my-release --namespace agones-system -f values.yaml agones/agones
```

{{< alert title="Tip" color="info">}}
You can use the default {{< ghlink href="install/helm/agones/values.yaml" >}}values.yaml{{< /ghlink >}}
{{< /alert >}}

## Helm test

{{< alert title="Tip" color="info">}}
In order to use `helm test` command described in this section you need to set `helm.installTests` helm parameter to `true`.
{{< /alert >}}

Check the Agones installation by running the following command:
```bash
helm test my-release --cleanup
```
```
RUNNING: agones-test
PASSED: agones-test
```

This test would create a `GameServer` resource and delete it afterwards.

{{< alert title="Tip" color="info">}}
If you receive the following error:
```
RUNNING: agones-test
ERROR: pods "agones-test" already exists
Error: 1 test(s) failed
```
That means that you skipped the `--cleanup` flag and you should either delete the `agones-test` pod manually or run with the same test `helm test my-release --cleanup` two more times.
{{< /alert >}}

## Controller TLS Certificates

By default agones chart generates tls certificates used by the admission controller, while this is handy, it requires the agones controller to restart on each `helm upgrade` command.
For most use cases the controller would have required a restart anyway (eg: controller image updated). However if you really need to avoid restarts we suggest that you turn off tls automatic generation (`agones.controller.generateTLS` to `false`) and provide your own certificates (`certs/server.crt`,`certs/server.key`).

{{< alert title="Tip" color="info">}}
You can use our script located at {{< ghlink href="install/helm/agones/certs/cert.sh" >}}cert.sh{{< /ghlink >}} to generate them.
{{< /alert >}}

## Reserved Allocator Load Balancer IP

In order to reuse the existing load balancer IP on upgrade or install the `agones-allocator` service as a `LoadBalancer` using a reserved static IP, a user can specify the load balancer's IP with the `agones.allocator.http.loadBalancerIP` helm configuration parameter value. By setting the `loadBalancerIP` value:

1. The `LoadBalancer` is created with the specified IP, if supported by the cloud provider.
2. A self-signed server TLS certificate is generated for the IP, used by the `agones-allocator` service.

## Next Steps

- [Confirm Agones is up and running]({{< relref "../confirm.md" >}})
