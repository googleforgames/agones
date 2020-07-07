---
title: "Metrics"
date: 2019-01-03T03:58:19Z
weight: 50
---

Agones controller exposes metrics via [OpenCensus](https://opencensus.io/). OpenCensus is a single distribution of libraries that collect metrics and distributed traces from your services, we only use it for metrics but it will allow us to support multiple exporters in the future.

We choose to start with [Prometheus](https://prometheus.io/) as this is the most popular with Kubernetes but it is also compatible with Stackdriver.
If you need another exporter, check the [list of supported](https://opencensus.io/exporters/supported-exporters/go/) exporters. It should be pretty straightforward to register a new one. (GitHub PRs are more than welcome.)

We plan to support multiple exporters in the future via environment variables and helm flags.

## Backend integrations

### Prometheus

If you are running a [Prometheus](https://prometheus.io/) instance you just need to ensure that metrics and kubernetes service discovery are enabled. (helm chart values `agones.metrics.prometheusEnabled` and `agones.metrics.prometheusServiceDiscovery`). This will automatically add annotations required by Prometheus to discover Agones metrics and start collecting them. (see [example](https://github.com/prometheus/prometheus/tree/master/documentation/examples/kubernetes-rabbitmq))

### Prometheus Operator

If you have [Prometheus operator](https://github.com/coreos/prometheus-operator) installed in your cluster, make sure to add a [`ServiceMonitor`](https://github.com/coreos/prometheus-operator/blob/v0.17.0/Documentation/api.md#servicemonitorspec) to discover Agones metrics as shown below:

```yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: agones
  labels:
    app: agones
spec:
  selector:
    matchLabels:
        agones.dev/role: controller
  endpoints:
  - port: web
```

Finally include that `ServiceMonitor` in your [Prometheus instance CRD](https://github.com/coreos/prometheus-operator/blob/v0.17.0/Documentation/user-guides/getting-started.md#include-servicemonitors), this is usually done by adding a label to the `ServiceMonitor` above that is matched by the prometheus instance of your choice.

### Stackdriver

We support the [OpenCensus Stackdriver exporter](https://opencensus.io/exporters/supported-exporters/go/stackdriver/).
In order to use it you should enable [Stackdriver Monitoring API](https://cloud.google.com/monitoring/api/enable-api) in Google Cloud Console.
Follow the [Stackdriver Installation steps](#stackdriver-installation) to see your metrics on Stackdriver Monitoring website.

## Metrics available

| Name                                            | Description                                                         | Type      |
|-------------------------------------------------|---------------------------------------------------------------------|-----------|
| agones_gameservers_count                        | The number of gameservers per fleet and status                      | gauge     |
| agones_gameserver_allocations_duration_seconds  | The distribution of gameserver allocation requests latencies         | histogram     |
| agones_gameservers_total                        | The total of gameservers per fleet and status                       | counter   |
| agones_fleets_replicas_count                    | The number of replicas per fleet (total, desired, ready, allocated) | gauge     |
| agones_fleet_autoscalers_able_to_scale          | The fleet autoscaler can access the fleet to scale                  | gauge     |
| agones_fleet_autoscalers_buffer_limits          | The limits of buffer based fleet autoscalers (min, max)              | gauge     |
| agones_fleet_autoscalers_buffer_size            | The buffer size of fleet autoscalers (count or percentage)          | gauge     |
| agones_fleet_autoscalers_current_replicas_count | The current replicas count as seen by autoscalers                   | gauge     |
| agones_fleet_autoscalers_desired_replicas_count | The desired replicas count as seen by autoscalers                   | gauge     |
| agones_fleet_autoscalers_limited                | The fleet autoscaler is capped (1)                                  | gauge     |
| agones_gameservers_node_count                   | The distribution of gameservers per node                            | histogram |
| agones_nodes_count                              | The count of nodes empty and with gameservers                       | gauge     |
| agones_gameservers_state_duration  | The distribution of gameserver state duration in seconds. Note: this metric could have some missing samples by design. Do not use the `_total` counter as the real value for state changes.     | histogram     |
| agones_k8s_client_http_request_total            | The total of HTTP requests to the Kubernetes API by status code       | counter   |
| agones_k8s_client_http_request_duration_seconds | The distribution of HTTP requests latencies to the Kubernetes API by status code  | histogram   |
| agones_k8s_client_cache_list_total              | The total number of list operations for client-go caches                         | counter   |
| agones_k8s_client_cache_list_duration_seconds   | Duration of a Kubernetes list API call in seconds                        | histogram   |
| agones_k8s_client_cache_list_items              | Count of items in a list from the Kubernetes API                            | histogram   |
| agones_k8s_client_cache_watches_total           | The total number of watch operations for client-go caches                         | counter   |
| agones_k8s_client_cache_last_resource_version   | Last resource version from the Kubernetes API                            | gauge   |
| agones_k8s_client_workqueue_depth               | Current depth of the work queue                          | gauge   |
| agones_k8s_client_workqueue_latency_seconds     | How long an item stays in the work queue                          | histogram   |
| agones_k8s_client_workqueue_items_total         | Total number of items added to the work queue                          | counter   |
| agones_k8s_client_workqueue_work_duration_seconds | How long processing an item from the work queue takes                          | histogram   |
| agones_k8s_client_workqueue_retries_total         | Total number of items retried to the work queue                          | counter   |
| agones_k8s_client_workqueue_longest_running_processor         | How long the longest running workqueue processor has been running in microseconds  | gauge   |
| agones_k8s_client_workqueue_unfinished_work_seconds         | How long unfinished work has been sitting in the workqueue in seconds    | gauge   |

## Dashboard

### Grafana Dashboards

We provide a set of useful [Grafana](https://grafana.com/) dashboards to monitor Agones workload, they are located under the {{< ghlink href="/build/grafana" branch="master" >}}grafana folder{{< /ghlink >}}:

- {{< ghlink href="/build/grafana/dashboard-autoscalers.yaml" branch="master" >}}Agones Autoscalers{{< /ghlink >}} allows you to monitor your current autoscalers replicas request as well as fleet replicas allocation and readyness statuses. You can only select one autoscaler at the time using the provided dropdown.

- {{< ghlink href="/build/grafana/dashboard-gameservers.yaml" branch="master" >}}Agones GameServers{{< /ghlink >}} displays your current game servers workload status (allocations, game servers statuses, fleets replicas) with optional fleet name filtering.

- {{< ghlink href="/build/grafana/dashboard-allocations.yaml" branch="master" >}}Agones GameServer Allocations{{< /ghlink >}} displays Agones gameservers allocations rates and counts per fleet.

- {{< ghlink href="/build/grafana/dashboard-allocator-usage.yaml" branch="master" >}}Agones Allocator Resource{{< /ghlink >}} displays Agones Allocators CPU, memory usage and also some useful Golang runtime metrics.

- {{< ghlink href="/build/grafana/dashboard-status.yaml" branch="master" >}}Agones Status{{< /ghlink >}} displays Agones controller health status.

- {{< ghlink href="/build/grafana/dashboard-controller-usage.yaml" branch="master" >}}Agones Controller Resource Usage{{< /ghlink >}} displays Agones Controller CPU and memory usage and also some Golang runtime metrics.

- {{< ghlink href="/build/grafana/dashboard-goclient-requests.yaml" branch="master" >}}Agones Controller go-client requests{{< /ghlink >}} displays Agones Controller Kubernetes API consumption.

- {{< ghlink href="/build/grafana/dashboard-goclient-caches.yaml" branch="master" >}}Agones Controller go-client caches{{< /ghlink >}} displays Agones Controller Kubernetes Watches/Lists operations used.

- {{< ghlink href="/build/grafana/dashboard-goclient-workqueues.yaml" branch="master" >}}Agones Controller go-client workqueues{{< /ghlink >}} displays Agones Controller workqueue processing time and rates.

- {{< ghlink href="/build/grafana/dashboard-apiserver-requests.yaml" branch="master" >}}Agones Controller API Server requests{{< /ghlink >}} displays your current API server request rate, errors rate and request latencies with optional CustomResourceDefinition filtering by Types: fleets, gameserversets, gameservers, gameserverallocations.

Dashboard screenshots :

![grafana dashboard autoscalers](../../../images/grafana-dashboard-autoscalers.png)

![grafana dashboard controller](../../../images/grafana-dashboard-controller.png)

{{< alert title="Note" color="info">}}
You can import our dashboards by copying the json content from {{< ghlink href="/build/grafana" branch="master" >}}each config map{{< /ghlink >}} into your own instance of Grafana (+ > Create > Import > Or paste json) or follow the [installation](#installation) guide.
{{< /alert >}}

## Installation

When operating a live multiplayer game you will need to observe performances, resource usage and availability to learn more about your system. This guide will explain how you can setup Prometheus and Grafana into your own Kubernetes cluster to monitor your Agones workload.

Before attemping this guide you should make sure you have [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/) and [helm](https://docs.helm.sh/using_helm/) installed and configured to reach your kubernetes cluster.

### Prometheus installation

Prometheus is an open source monitoring solution, we will use it to store Agones controller metrics and query back the data.

Let's install Prometheus using the [helm stable](https://github.com/helm/charts/tree/master/stable/prometheus) repository.

```bash
helm upgrade --install --wait prom stable/prometheus --namespace metrics \
    --set server.global.scrape_interval=30s \
    --set server.persistentVolume.enabled=true \
    --set server.persistentVolume.size=64Gi \
    -f ./build/prometheus.yaml
```

{{% alert title="Note" color="info"%}}
You can also run our {{< ghlink href="/build/Makefile" branch="master" branch="master" >}}Makefile{{< /ghlink >}} target `make setup-prometheus`
or `make kind-setup-prometheus` and `make minikube-setup-prometheus` 
for {{< ghlink href="/build/README.md#running-a-test-kind-cluster" branch="master" >}}Kind{{< /ghlink >}} and {{< ghlink href="/build/README.md#running-a-test-minikube-cluster" branch="master" >}}Minikube{{< /ghlink >}}.
{{% /alert %}}

For resiliency it is recommended to run Prometheus on a dedicated node which is separate from nodes where Game Servers
are scheduled. If you use the above command, with our {{< ghlink href="/build/prometheus.yaml" branch="master" >}}prometheus.yaml{{< /ghlink >}} to set up Prometheus, it will schedule Prometheus pods on nodes
tainted with `agones.dev/agones-metrics=true:NoExecute` and labeled with `agones.dev/agones-metrics=true` if available.

As an example, to set up a dedicated node pool for Prometheus on GKE, run the following command before installing Prometheus. Alternatively you can taint and label nodes manually.

```
gcloud container node-pools create agones-metrics --cluster=... --zone=... \
  --node-taints agones.dev/agones-metrics=true:NoExecute \
  --node-labels agones.dev/agones-metrics=true \
  --num-nodes=1
```

By default we will disable the push gateway (we don't need it for Agones) and other exporters.

The helm [chart](https://github.com/helm/charts/tree/master/stable/prometheus) support [nodeSelector](https://kubernetes.io/docs/concepts/configuration/assign-pod-node/#nodeselector), [affinity](https://kubernetes.io/docs/concepts/configuration/assign-pod-node/#affinity-and-anti-affinity) and [toleration](https://kubernetes.io/docs/concepts/configuration/taint-and-toleration/), you can use them to schedule prometheus deployments on an isolated node(s) to have an homogeneous game servers workload.

This will install a Prometheus Server in your current cluster with [Persistent Volume Claim](https://kubernetes.io/docs/concepts/storage/persistent-volumes/) (Deactivated for Minikube and Kind) for storing and querying time series, it will automatically start collecting metrics from Agones Controller.

Finally to access Prometheus metrics, rules and alerts explorer use

```bash
kubectl port-forward deployments/prom-prometheus-server 9090 -n metrics
```

{{< alert title="Note" color="info">}}
 Again you can use our Makefile {{< ghlink href="/build/README.md#prometheus-portforward" branch="master" >}}`make prometheus-portforward`{{< /ghlink >}}.
  (For {{< ghlink href="/build/README.md#running-a-test-kind-cluster" branch="master" >}}Kind{{< /ghlink >}} and {{< ghlink href="/build/README.md#running-a-test-minikube-cluster" branch="master" >}}Minikube{{< /ghlink >}} use their specific targets `make kind-prometheus-portforward` and `make minikube-prometheus-portforward`)
{{< /alert >}}

Now you can access the prometheus dashboard [http://localhost:9090](http://localhost:9090).

On the landing page you can start exploring metrics by creating [queries](https://prometheus.io/docs/prometheus/latest/querying/basics/). You can also verify what [targets](http://localhost:9090/targets) Prometheus currently monitors (Header Status > Targets), you should see Agones controller pod in the `kubernetes-pods` section.

{{< alert title="Note" color="info">}}
Metrics will be first registered when you will start using Agones.
{{< /alert >}}

Now let's install some Grafana dashboards.

### Grafana installation

Grafana is a open source time series analytics platform which supports Prometheus data source. We can also easily import pre-built dashboards.

First we will install [Agones dashboard](#grafana-dashboards) as [config maps](https://kubernetes.io/docs/tasks/configure-pod-container/configure-pod-configmap/) in our cluster.

```bash
kubectl apply -f ./build/grafana/
```

Now we can install [grafana chart](https://github.com/helm/charts/tree/master/stable/grafana) from stable repository. (Replace `<your-admin-password>` with the admin password of your choice)

```bash
helm install --wait --name grafana stable/grafana --version=5.0.13 --namespace metrics \
  --set adminPassword=<your-admin-password> -f ./build/grafana.yaml
```

This will install Grafana with our prepopulated dashboards and prometheus datasource [previously installed](#prometheus-installation)

{{< alert title="Note" color="info">}}
You can also use our {{< ghlink href="/build/Makefile" branch="master" >}}Makefile{{< /ghlink >}} targets (`setup-grafana`, `minikube-setup-grafana` and `kind-setup-grafana`).
{{< /alert >}}

Finally to access dashboards run

```bash
kubectl port-forward deployments/grafana 3000 -n metrics
```

Open a web browser to [http://localhost:3000](http://localhost:3000), you should see Agones [dashboards](#grafana-dashboards) after login as admin.

{{< alert title="Note" color="info">}}
You can also use our `Makefile` targets `make grafana-portforward`, `make kind-grafana-portforward` and `make minikube-grafana-portforward`.
{{< /alert >}}

### Stackdriver installation

In order to use [Stackdriver monitoring](https://app.google.stackdriver.com) you should [enable Stackdriver Monitoring API](https://cloud.google.com/monitoring/api/enable-api) on Google Cloud Console. You need to grant all the necessary permissions to the users (see [Access Control Guide](https://cloud.google.com/monitoring/access-control)). Stackdriver exporter uses a strategy called Application Default Credentials (ADC) to find your application's credentials. Details could be found here [Setting Up Authentication for Server to Server Production Applications](https://cloud.google.com/docs/authentication/production).

Note that Stackdriver monitoring is enabled by default on GKE clusters, however you can follow this [guide](https://cloud.google.com/kubernetes-engine/docs/how-to/monitoring#enabling_stackdriver_monitoring) if it was disabled on your GKE cluster.

Default metrics exporter is Prometheus. If you are using the [Helm installation]({{< ref "/docs/Installation/Install Agones/helm.md" >}}), you can install or upgrade Agones to use Stackdriver, using the following chart parameters:
```
helm upgrade --install --wait --set agones.metrics.stackdriverEnabled=true --set agones.metrics.prometheusEnabled=false --set agones.metrics.prometheusServiceDiscovery=false my-release-name agones/agones --namespace=agones-system
```

With this configuration only Stackdriver exporter would be used instead of Prometheus exporter.

Create a Fleet or a Gameserver in order to check that connection with stackdriver API is configured properly and so that you will be able to see the metrics data.

Visit [Stackdriver monitoring](https://app.google.stackdriver.com) website, select your project, or choose `Create a new Workspace` and select GCP project where your cluster resides. In [Stackdriver metrics explorer](https://cloud.google.com/monitoring/charts/metrics-explorer) you should be able to find new metrics with prefix `agones/` after a couple of minutes. Choose the metrics you are interested in and add to a single or separate graphs. Select `Kubernetes Container` resource type for each of them. You can create multiple graphs, save them into your dashboard and use various aggregation parameters and reducers for each graph.

Example of the dashboard appearance is provided below:

![stackdriver monitoring dashboard](../../../images/stackdriver-metrics-dashboard.png)

Currently there exists only manual way of configuring Stackdriver Dashboard. So it is up to you to set an Alignment Period (minimal is 1 minute), GroupBy, Filter parameters and other graph settings.

#### Troubleshooting
If you can't see Agones metrics you should have a look at the controller logs for connection errors. Also ensure that your cluster has the necessary credentials to interact with Stackdriver Monitoring. You can configure `stackdriverProjectID` manually, if the automatic discovery is not working.

Permissions problem example from controller logs:
```
Failed to export to Stackdriver: rpc error: code = PermissionDenied desc = Permission monitoring.metricDescriptors.create denied (or the resource may not exist).
```
