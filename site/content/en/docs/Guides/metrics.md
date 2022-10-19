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

If you are running a [Prometheus](https://prometheus.io/) instance you just need to ensure that metrics and kubernetes service discovery are enabled. (helm chart values `agones.metrics.prometheusEnabled` and `agones.metrics.prometheusServiceDiscovery`). This will automatically add annotations required by Prometheus to discover Agones metrics and start collecting them. (see [example](https://github.com/prometheus/prometheus/tree/main/documentation/examples/kubernetes-rabbitmq))

If your Prometheus metrics collection agent requires that you scrape from the pods directly(such as with [Google Cloud Managed Prometheus](https://cloud.google.com/stackdriver/docs/managed-prometheus)), then the metrics ports for the controller and allocator will both be named `http` and exposed on `8080`. In the case of the allocator, the port name and number can be overriden with the `agones.allocator.serviceMetrics.http.portName` and `agones.allocator.serviceMetrics.http.port` helm chart values.

### Prometheus Operator

If you have [Prometheus operator](https://github.com/coreos/prometheus-operator) installed in your cluster, just enable ServiceMonitor installation in values:

```yaml
agones:
  metrics:
    serviceMonitor:
      enabled: true
```

### Stackdriver

We support the [OpenCensus Stackdriver exporter](https://opencensus.io/exporters/supported-exporters/go/stackdriver/).
In order to use it you should enable [Stackdriver Monitoring API](https://cloud.google.com/monitoring/api/enable-api) in Google Cloud Console.
Follow the [Stackdriver Installation steps](#stackdriver-installation) to see your metrics on Stackdriver Monitoring website.

## Metrics available

{{% feature expiryVersion="1.28.0" %}}
| Name                                                  | Description                                                                                                                                                                                 | Type      |
|-------------------------------------------------------|---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|-----------|
| agones_gameservers_count                              | The number of gameservers per fleet and status                                                                                                                                              | gauge     |
| agones_gameserver_allocations_duration_seconds        | The distribution of gameserver allocation requests latencies                                                                                                                                | histogram |
| agones_gameservers_total                              | The total of gameservers per fleet and status                                                                                                                                               | counter   |
| agones_fleets_replicas_count                          | The number of replicas per fleet (total, desired, ready, reserved, allocated)                                                                                                               | gauge     |
| agones_fleet_autoscalers_able_to_scale                | The fleet autoscaler can access the fleet to scale                                                                                                                                          | gauge     |
| agones_fleet_autoscalers_buffer_limits                | The limits of buffer based fleet autoscalers (min, max)                                                                                                                                     | gauge     |
| agones_fleet_autoscalers_buffer_size                  | The buffer size of fleet autoscalers (count or percentage)                                                                                                                                  | gauge     |
| agones_fleet_autoscalers_current_replicas_count       | The current replicas count as seen by autoscalers                                                                                                                                           | gauge     |
| agones_fleet_autoscalers_desired_replicas_count       | The desired replicas count as seen by autoscalers                                                                                                                                           | gauge     |
| agones_fleet_autoscalers_limited                      | The fleet autoscaler is capped (1)                                                                                                                                                          | gauge     |
| agones_gameservers_node_count                         | The distribution of gameservers per node                                                                                                                                                    | histogram |
| agones_nodes_count                                    | The count of nodes empty and with gameservers                                                                                                                                               | gauge     |
| agones_gameservers_state_duration                     | The distribution of gameserver state duration in seconds. Note: this metric could have some missing samples by design. Do not use the `_total` counter as the real value for state changes. | histogram |
| agones_k8s_client_http_request_total                  | The total of HTTP requests to the Kubernetes API by status code                                                                                                                             | counter   |
| agones_k8s_client_http_request_duration_seconds       | The distribution of HTTP requests latencies to the Kubernetes API by status code                                                                                                            | histogram |
| agones_k8s_client_cache_list_total                    | The total number of list operations for client-go caches                                                                                                                                    | counter   |
| agones_k8s_client_cache_list_duration_seconds         | Duration of a Kubernetes list API call in seconds                                                                                                                                           | histogram |
| agones_k8s_client_cache_list_items                    | Count of items in a list from the Kubernetes API                                                                                                                                            | histogram |
| agones_k8s_client_cache_watches_total                 | The total number of watch operations for client-go caches                                                                                                                                   | counter   |
| agones_k8s_client_cache_last_resource_version         | Last resource version from the Kubernetes API                                                                                                                                               | gauge     |
| agones_k8s_client_workqueue_depth                     | Current depth of the work queue                                                                                                                                                             | gauge     |
| agones_k8s_client_workqueue_latency_seconds           | How long an item stays in the work queue                                                                                                                                                    | histogram |
| agones_k8s_client_workqueue_items_total               | Total number of items added to the work queue                                                                                                                                               | counter   |
| agones_k8s_client_workqueue_work_duration_seconds     | How long processing an item from the work queue takes                                                                                                                                       | histogram |
| agones_k8s_client_workqueue_retries_total             | Total number of items retried to the work queue                                                                                                                                             | counter   |
| agones_k8s_client_workqueue_longest_running_processor | How long the longest running workqueue processor has been running in microseconds                                                                                                           | gauge     |
| agones_k8s_client_workqueue_unfinished_work_seconds   | How long unfinished work has been sitting in the workqueue in seconds                                                                                                                       | gauge     |
{{% /feature %}}

{{% feature publishVersion="1.28.0" %}}
| Name                                                  | Description                                                                                                                                                                                 | Type      |
|-------------------------------------------------------|---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|-----------|
| agones_gameservers_count                              | The number of gameservers per fleet and status                                                                                                                                              | gauge     |
| agones_gameserver_allocations_duration_seconds        | The distribution of gameserver allocation requests latencies                                                                                                                                | histogram |
| agones_gameservers_total                              | The total of gameservers per fleet and status                                                                                                                                               | counter   |
| agones_gameserver_player_connected_total              | The total number of players connected to gameservers (Only available when [player tracking]({{< relref "player-tracking.md" >}}) is enabled)                                                | gauge     |
| agones_gameserver_player_capacity_total               | The available capacity for players on gameservers (Only available when [player tracking]({{< relref "player-tracking.md" >}}) is enabled)                                                   | gauge     |
| agones_fleets_replicas_count                          | The number of replicas per fleet (total, desired, ready, reserved, allocated)                                                                                                               | gauge     |
| agones_fleet_autoscalers_able_to_scale                | The fleet autoscaler can access the fleet to scale                                                                                                                                          | gauge     |
| agones_fleet_autoscalers_buffer_limits                | The limits of buffer based fleet autoscalers (min, max)                                                                                                                                     | gauge     |
| agones_fleet_autoscalers_buffer_size                  | The buffer size of fleet autoscalers (count or percentage)                                                                                                                                  | gauge     |
| agones_fleet_autoscalers_current_replicas_count       | The current replicas count as seen by autoscalers                                                                                                                                           | gauge     |
| agones_fleet_autoscalers_desired_replicas_count       | The desired replicas count as seen by autoscalers                                                                                                                                           | gauge     |
| agones_fleet_autoscalers_limited                      | The fleet autoscaler is capped (1)                                                                                                                                                          | gauge     |
| agones_gameservers_node_count                         | The distribution of gameservers per node                                                                                                                                                    | histogram |
| agones_nodes_count                                    | The count of nodes empty and with gameservers                                                                                                                                               | gauge     |
| agones_gameservers_state_duration                     | The distribution of gameserver state duration in seconds. Note: this metric could have some missing samples by design. Do not use the `_total` counter as the real value for state changes. | histogram |
| agones_k8s_client_http_request_total                  | The total of HTTP requests to the Kubernetes API by status code                                                                                                                             | counter   |
| agones_k8s_client_http_request_duration_seconds       | The distribution of HTTP requests latencies to the Kubernetes API by status code                                                                                                            | histogram |
| agones_k8s_client_cache_list_total                    | The total number of list operations for client-go caches                                                                                                                                    | counter   |
| agones_k8s_client_cache_list_duration_seconds         | Duration of a Kubernetes list API call in seconds                                                                                                                                           | histogram |
| agones_k8s_client_cache_list_items                    | Count of items in a list from the Kubernetes API                                                                                                                                            | histogram |
| agones_k8s_client_cache_watches_total                 | The total number of watch operations for client-go caches                                                                                                                                   | counter   |
| agones_k8s_client_cache_last_resource_version         | Last resource version from the Kubernetes API                                                                                                                                               | gauge     |
| agones_k8s_client_workqueue_depth                     | Current depth of the work queue                                                                                                                                                             | gauge     |
| agones_k8s_client_workqueue_latency_seconds           | How long an item stays in the work queue                                                                                                                                                    | histogram |
| agones_k8s_client_workqueue_items_total               | Total number of items added to the work queue                                                                                                                                               | counter   |
| agones_k8s_client_workqueue_work_duration_seconds     | How long processing an item from the work queue takes                                                                                                                                       | histogram |
| agones_k8s_client_workqueue_retries_total             | Total number of items retried to the work queue                                                                                                                                             | counter   |
| agones_k8s_client_workqueue_longest_running_processor | How long the longest running workqueue processor has been running in microseconds                                                                                                           | gauge     |
| agones_k8s_client_workqueue_unfinished_work_seconds   | How long unfinished work has been sitting in the workqueue in seconds                                                                                                                       | gauge     |
{{% /feature %}}

### Dropping Metric Labels

{{% alpha title="Reset Metric Export on Fleet / Autoscaler deletion" gate="ResetMetricsOnDelete" %}}

When a Fleet or FleetAutoscaler is deleted from the system, Agones will automatically clear metrics that utilise 
their name as a label from the exported metrics, so the metrics exported do not continuously grow in size over the 
lifecycle of the Agones installation.


## Dashboard

### Grafana Dashboards

We provide a set of useful [Grafana](https://grafana.com/) dashboards to monitor Agones workload, they are located under the {{< ghlink href="/build/grafana" branch="main" >}}grafana folder{{< /ghlink >}}:

- {{< ghlink href="/build/grafana/dashboard-autoscalers.yaml" branch="main" >}}Agones Autoscalers{{< /ghlink >}} allows you to monitor your current autoscalers replicas request as well as fleet replicas allocation and readyness statuses. You can only select one autoscaler at the time using the provided dropdown.

- {{< ghlink href="/build/grafana/dashboard-gameservers.yaml" branch="main" >}}Agones GameServers{{< /ghlink >}} displays your current game servers workload status (allocations, game servers statuses, fleets replicas) with optional fleet name filtering.

- {{< ghlink href="/build/grafana/dashboard-allocations.yaml" branch="main" >}}Agones GameServer Allocations{{< /ghlink >}} displays Agones gameservers allocations rates and counts per fleet.

- {{< ghlink href="/build/grafana/dashboard-allocator-usage.yaml" branch="main" >}}Agones Allocator Resource{{< /ghlink >}} displays Agones Allocators CPU, memory usage and also some useful Golang runtime metrics.

- {{< ghlink href="/build/grafana/dashboard-status.yaml" branch="main" >}}Agones Status{{< /ghlink >}} displays Agones controller health status.

- {{< ghlink href="/build/grafana/dashboard-controller-usage.yaml" branch="main" >}}Agones Controller Resource Usage{{< /ghlink >}} displays Agones Controller CPU and memory usage and also some Golang runtime metrics.

- {{< ghlink href="/build/grafana/dashboard-goclient-requests.yaml" branch="main" >}}Agones Controller go-client requests{{< /ghlink >}} displays Agones Controller Kubernetes API consumption.

- {{< ghlink href="/build/grafana/dashboard-goclient-caches.yaml" branch="main" >}}Agones Controller go-client caches{{< /ghlink >}} displays Agones Controller Kubernetes Watches/Lists operations used.

- {{< ghlink href="/build/grafana/dashboard-goclient-workqueues.yaml" branch="main" >}}Agones Controller go-client workqueues{{< /ghlink >}} displays Agones Controller workqueue processing time and rates.

- {{< ghlink href="/build/grafana/dashboard-apiserver-requests.yaml" branch="main" >}}Agones Controller API Server requests{{< /ghlink >}} displays your current API server request rate, errors rate and request latencies with optional CustomResourceDefinition filtering by Types: fleets, gameserversets, gameservers, gameserverallocations.

Dashboard screenshots :

![grafana dashboard autoscalers](../../../images/grafana-dashboard-autoscalers.png)

![grafana dashboard controller](../../../images/grafana-dashboard-controller.png)

{{< alert title="Note" color="info">}}
You can import our dashboards by copying the json content from {{< ghlink href="/build/grafana" branch="main" >}}each config map{{< /ghlink >}} into your own instance of Grafana (+ > Create > Import > Or paste json) or follow the [installation](#installation) guide.
{{< /alert >}}

## Installation

When operating a live multiplayer game you will need to observe performances, resource usage and availability to learn more about your system. This guide will explain how you can setup Prometheus and Grafana into your own Kubernetes cluster to monitor your Agones workload.

Before attemping this guide you should make sure you have [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/) and [helm](https://docs.helm.sh/using_helm/) installed and configured to reach your kubernetes cluster.

### Prometheus installation

Prometheus is an open source monitoring solution, we will use it to store Agones controller metrics and query back the data.

Let's install Prometheus using the [Prometheus Community Kubernetes Helm Charts](https://prometheus-community.github.io/helm-charts/) repository.

```bash
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm repo update

helm upgrade --install --wait prom prometheus-community/prometheus --version 11.16.2 --namespace metrics \
    --set server.global.scrape_interval=30s \
    --set server.persistentVolume.enabled=true \
    --set server.persistentVolume.size=64Gi \
    -f ./build/prometheus.yaml
```

For resiliency it is recommended to run Prometheus on a dedicated node which is separate from nodes where Game Servers
are scheduled. If you use the above command, with our {{< ghlink href="/build/prometheus.yaml" branch="main" >}}prometheus.yaml{{< /ghlink >}} to set up Prometheus, it will schedule Prometheus pods on nodes
tainted with `agones.dev/agones-metrics=true:NoExecute` and labeled with `agones.dev/agones-metrics=true` if available.

As an example, to set up a dedicated node pool for Prometheus on GKE, run the following command before installing Prometheus. Alternatively you can taint and label nodes manually.

```bash
gcloud container node-pools create agones-metrics --cluster=... --zone=... \
  --node-taints agones.dev/agones-metrics=true:NoExecute \
  --node-labels agones.dev/agones-metrics=true \
  --num-nodes=1
```

By default we will disable the push gateway (we don't need it for Agones) and other exporters.

The helm chart supports
[nodeSelector](https://kubernetes.io/docs/concepts/configuration/assign-pod-node/#nodeselector),
[affinity](https://kubernetes.io/docs/concepts/configuration/assign-pod-node/#affinity-and-anti-affinity) and [toleration](https://kubernetes.io/docs/concepts/configuration/taint-and-toleration/), you can use them to schedule Prometheus deployments on an isolated node(s) to have an homogeneous game servers workload.

This will install a Prometheus Server in your current cluster with [Persistent Volume Claim](https://kubernetes.io/docs/concepts/storage/persistent-volumes/) (Deactivated for Minikube and Kind) for storing and querying time series, it will automatically start collecting metrics from Agones Controller.

Finally, to access Prometheus metrics, rules and alerts explorer use

```bash
kubectl port-forward deployments/prom-prometheus-server 9090 -n metrics
```

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

Now we can install the
[Grafana Community Kubernetes Helm Charts](https://grafana.github.io/helm-charts/) from
their repository. (Replace `<your-admin-password>` with the admin password of your choice)

```bash
helm repo add grafana https://grafana.github.io/helm-charts
helm repo update

helm upgrade --install --wait grafana grafana/grafana --version=5.7.10 --namespace metrics \
  --set adminPassword=<your-admin-password> -f ./build/grafana.yaml
```

This will install Grafana with our prepopulated dashboards and prometheus datasource [previously installed](#prometheus-installation)

Finally to access dashboards run

```bash
kubectl port-forward deployments/grafana 3000 -n metrics
```

Open a web browser to [http://localhost:3000](http://localhost:3000), you should see Agones [dashboards](#grafana-dashboards) after login as admin.

### Stackdriver installation

In order to use [Stackdriver monitoring](https://app.google.stackdriver.com) you must [enable the Monitoring API](https://cloud.google.com/monitoring/api/enable-api) in the Google Cloud Console. You need to grant all the necessary permissions to the users (see [Access Control Guide](https://cloud.google.com/monitoring/access-control)). Stackdriver exporter uses a strategy called Application Default Credentials (ADC) to find your application's credentials. Details can be found in [Setting Up Authentication for Server to Server Production Applications](https://cloud.google.com/docs/authentication/production).

{{< alert title="Note" color="info">}}
Cloud Operations for GKE (including stackdriver monitoring) is enabled by default on GKE clusters, however you can follow this [guide](https://cloud.google.com/stackdriver/docs/solutions/gke/installing#upgrade-instructions) if it is currently disabled in your GKE cluster.
{{< /alert >}}

The default metrics exporter installed with Agones is Prometheus. If you are using the [Helm installation]({{< ref "/docs/Installation/Install Agones/helm.md" >}}), you can install or upgrade Agones to use Stackdriver, using the following chart parameters:
```bash
helm upgrade --install --wait --set agones.metrics.stackdriverEnabled=true --set agones.metrics.prometheusEnabled=false --set agones.metrics.prometheusServiceDiscovery=false my-release-name agones/agones --namespace=agones-system
```

With this configuration only Stackdriver exporter would be used instead of Prometheus exporter.

#### Using Stackdriver with Workload Identity

If you would like to enable stackdriver in conjunction with [Workload Identity](https://cloud.google.com/kubernetes-engine/docs/how-to/workload-identity), there are a few extra steps you need to follow:

1. When setting up the Google service account following the instructions for [Authenticating to Google Cloud](https://cloud.google.com/kubernetes-engine/docs/how-to/workload-identity#authenticating_to), create two IAM policy bindings, one for `serviceAccount:PROJECT_ID.svc.id.goog[agones-system/agones-controller]` and one for `serviceAccount:PROJECT_ID.svc.id.goog[agones-system/agones-allocator]`.

1. Pass parameters to helm when installing Agones to add annotations to the `agones-controller` and `agones-allocator` Kubernetes service accounts:

```bash
helm install my-release --namespace agones-system --create-namespace agones/agones --set agones.metrics.stackdriverEnabled=true --set agones.metrics.prometheusEnabled=false --set agones.metrics.prometheusServiceDiscovery=false --set agones.serviceaccount.allocator.annotations."iam\.gke\.io/gcp-service-account"="GSA_NAME@PROJECT_ID\.iam\.gserviceaccount\.com" --set agones.serviceaccount.controller.annotations."iam\.gke\.io/gcp-service-account"="GSA_NAME@PROJECT_ID\.iam\.gserviceaccount\.com"
```

To verify that metrics are being sent to Stackdriver, create a Fleet or a Gameserver and look for the metrics to show up in the Stackdriver dashboard. Navigate to the [Metrics explorer](https://console.cloud.google.com/monitoring/metrics-explorer) and search for metrics with the prefix `agones/`. Select a metric and look for data to be plotted in the graph to the right.

An example of a custom dashboard is:

![stackdriver monitoring dashboard](../../../images/stackdriver-metrics-dashboard.png)

Currently there exists only manual way of configuring Stackdriver Dashboard. So it is up to you to set an Alignment Period (minimal is 1 minute), GroupBy, Filter parameters and other graph settings.

#### Troubleshooting
If you can't see Agones metrics you should have a look at the controller logs for connection errors. Also ensure that your cluster has the necessary credentials to interact with Stackdriver Monitoring. You can configure `stackdriverProjectID` manually, if the automatic discovery is not working.

Permissions problem example from controller logs:
```
Failed to export to Stackdriver: rpc error: code = PermissionDenied desc = Permission monitoring.metricDescriptors.create denied (or the resource may not exist).
```
