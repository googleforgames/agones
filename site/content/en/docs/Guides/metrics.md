---
title: "Metrics"
date: 2019-01-03T03:58:19Z
weight: 50
---

Agones controller exposes metrics via [OpenCensus](https://opencensus.io/). OpenCensus is a single distribution of libraries that collect metrics and distributed traces from your services, we only use it for metrics but it will allow us to support multiple exporters in the future.

We choose to start with [Prometheus](https://prometheus.io/) as this is the most popular with Kubernetes but it is also compatible with Stackdriver.
If you need another exporter, check the [list of supported](https://opencensus.io/exporters/supported-exporters/go/) exporters. It should be pretty straightforward to register a new one.(Github PR are more than welcomed)

We plan to support multiple exporters in the future via environement variables and helm flags.

## Backend integrations

### Prometheus

If you are running a [Prometheus](https://prometheus.io/) instance you just need to ensure that metrics and kubernetes service discovery are enabled. (helm chart values {{% feature expiryVersion="0.8.0" %}}`agones.metrics.enabled`{{% /feature %}}{{% feature publishVersion="0.8.0" %}}`agones.metrics.prometheusEnabled`{{% /feature %}} and `agones.metrics.prometheusServiceDiscovery`). This will automatically add annotations required by Prometheus to discover Agones metrics and start collecting them. (see [example](https://github.com/prometheus/prometheus/tree/master/documentation/examples/kubernetes-rabbitmq))

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
        stable.agones.dev/role: controller
  endpoints:
  - port: web
```

Finally include that `ServiceMonitor` in your [Prometheus instance CRD](https://github.com/coreos/prometheus-operator/blob/v0.17.0/Documentation/user-guides/getting-started.md#include-servicemonitors), this is usually done by adding a label to the `ServiceMonitor` above that is matched by the prometheus instance of your choice.

### Stackdriver

{{% feature expiryVersion="0.8.0" %}}
We don't yet support the [OpenCensus Stackdriver exporter](https://opencensus.io/exporters/supported-exporters/go/stackdriver/)
but you can still use the Prometheus Stackdriver integration by following these [instructions](https://cloud.google.com/monitoring/kubernetes-engine/prometheus).
Annotations required by this integration can be activated by setting the `agones.metrics.prometheusServiceDiscovery` 
to true (default) via the [helm chart value]({{< relref "../Installation/helm.md" >}}).
{{% /feature %}}
{{% feature publishVersion="0.8.0" %}}
We support the [OpenCensus Stackdriver exporter](https://opencensus.io/exporters/supported-exporters/go/stackdriver/). 
In order to use it you should enable [Stackdriver Monitoring API](https://cloud.google.com/monitoring/api/enable-api) in Google Cloud Console.
Follow the [Stackdriver Installation steps](#stackdriver-installation) to see your metrics on Stackdriver Monitoring website.
{{% /feature %}}

## Metrics available

| Name                                            | Description                                                         | Type    |
|-------------------------------------------------|---------------------------------------------------------------------|---------|
| agones_gameservers_count                        | The number of gameservers per fleet and status                      | gauge   |
| agones_fleet_allocations_count                  | The number of fleet allocations per fleet                           | gauge   |
| agones_gameservers_total                        | The total of gameservers per fleet and status                       | counter |
| agones_fleet_allocations_total                  | The total of fleet allocations per fleet                            | counter |
| agones_fleets_replicas_count                    | The number of replicas per fleet (total, desired, ready, allocated) | gauge   |
| agones_fleet_autoscalers_able_to_scale          | The fleet autoscaler can access the fleet to scale                  | gauge   |
| agones_fleet_autoscalers_buffer_limits          | he limits of buffer based fleet autoscalers (min, max)              | gauge   |
| agones_fleet_autoscalers_buffer_size            | The buffer size of fleet autoscalers (count or percentage)          | gauge   |
| agones_fleet_autoscalers_current_replicas_count | The current replicas count as seen by autoscalers                   | gauge   |
| agones_fleet_autoscalers_desired_replicas_count | The desired replicas count as seen by autoscalers                   | gauge   |
| agones_fleet_autoscalers_limited                | The fleet autoscaler is capped (1)                                  | gauge   |

## Dashboard

### Grafana Dashboards

We provide a set of useful [Grafana](https://grafana.com/) dashboards to monitor Agones workload, they are located under the {{< ghlink href="/build/grafana" branch="master" >}}grafana folder{{< /ghlink >}}:

- {{< ghlink href="/build/grafana/dashboard-autoscalers.yaml" branch="master" >}}Agones Autoscalers{{< /ghlink >}} allows you to monitor your current autoscalers replicas request as well as fleet replicas allocation and readyness statuses. You can only select one autoscaler at the time using the provided dropdown.

- {{< ghlink href="/build/grafana/dashboard-gameservers.yaml" branch="master" >}}Agones GameServers{{< /ghlink >}} displays your current game servers workload status (allocations , game servers statuses, fleets replicas) with optional fleet name filtering.

- {{< ghlink href="/build/grafana/dashboard-status.yaml" branch="master" >}}Agones Status{{< /ghlink >}} displays Agones controller health status.

- {{< ghlink href="/build/grafana/dashboard-controller-usage.yaml" branch="master" >}}Agones Controller Resource Usage{{< /ghlink >}} displays Agones Controller CPU and memory usage and also some Golang runtime metrics.

{{% feature publishVersion="0.8.0" %}}
- {{< ghlink href="/build/grafana/dashboard-goclient-requests.yaml" branch="master" >}}Agones Controller go-client requests{{< /ghlink >}} displays Agones Controller Kubernetes API consumption.

- {{< ghlink href="/build/grafana/dashboard-goclient-caches.yaml" branch="master" >}}Agones Controller go-client caches{{< /ghlink >}} displays Agones Controller Kubernetes Watches/Lists operations used.

- {{< ghlink href="/build/grafana/dashboard-goclient-workqueues.yaml" branch="master" >}}Agones Controller go-client workqueues{{< /ghlink >}} displays Agones Controller workqueue processing time and rates.
{{% /feature %}}

Dashboard screenshots :

![grafana dashboard autoscalers](../../../images/grafana-dashboard-autoscalers.png)

![grafana dashboard controller](../../../images/grafana-dashboard-controller.png)

> You can import our dashboards by copying the json content from {{< ghlink href="/build/grafana" branch="master" >}}each config map{{< /ghlink >}} into your own instance of Grafana (+ > Create > Import > Or paste json) or follow the [installation](#installation) guide.

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

> You can also run our {{< ghlink href="/build/Makefile" branch="master" branch="master" >}}Makefile{{< /ghlink >}} target `make setup-prometheus`
or `make kind-setup-prometheus` and `make minikube-setup-prometheus` for {{< ghlink href="/build/README.md#running-a-test-kind-cluster" branch="master" >}}Kind{{< /ghlink >}}
and {{< ghlink href="/build/README.md#running-a-test-minikube-cluster" branch="master" >}}Minikube{{< /ghlink >}}.

For resiliency it is recommended to run Prometheus on a dedicated node which is separate from nodes where Game Servers
are scheduled. If you use the above command, with our {{< ghlink href="/build/prometheus.yaml" branch="master" >}}prometheus.yaml{{< /ghlink >}} to set up Prometheus, it will schedule Prometheus pods on nodes
tainted with `stable.agones.dev/agones-metrics=true:NoExecute` and labeled with `stable.agones.dev/agones-metrics=true` if available.

As an example, to set up dedicated node pool for Prometheus on GKE, run the following command before installing Prometheus. Alternatively you can taint and label nodes manually.

```
gcloud container node-pools create agones-metrics --cluster=... --zone=... \
  --node-taints stable.agones.dev/agones-metrics=true:NoExecute \
  --node-labels stable.agones.dev/agones-metrics=true \
  --num-nodes=1
```

By default we will disable the push gateway (we don't need it for Agones) and other exporters.

The helm [chart](https://github.com/helm/charts/tree/master/stable/prometheus) support [nodeSelector](https://kubernetes.io/docs/concepts/configuration/assign-pod-node/#nodeselector), [affinity](https://kubernetes.io/docs/concepts/configuration/assign-pod-node/#affinity-and-anti-affinity) and [toleration](https://kubernetes.io/docs/concepts/configuration/taint-and-toleration/), you can use them to schedule prometheus deployments on an isolated node(s) to have an homogeneous game servers workload.

This will install a Prometheus Server in your current cluster with [Persistent Volume Claim](https://kubernetes.io/docs/concepts/storage/persistent-volumes/) (Deactivated for Minikube and Kind) for storing and querying time series, it will automatically start collecting metrics from Agones Controller.

Finally to access Prometheus metrics, rules and alerts explorer use

```bash
kubectl port-forward deployments/prom-prometheus-server 9090 -n metrics
```

> Again you can use our Makefile {{< ghlink href="/build/README.md#prometheus-portforward" branch="master" >}}`make prometheus-portforward`{{< /ghlink >}}.
  (For {{< ghlink href="/build/README.md#running-a-test-kind-cluster" branch="master" >}}Kind{{< /ghlink >}} and 
  {{< ghlink href="/build/README.md#running-a-test-minikube-cluster" branch="master" >}}Minikube{{< /ghlink >}} use their specific targets `make kind-prometheus-portforward` and `make minikube-prometheus-portforward`)

Now you can access the prometheus dashboard [http://localhost:9090](http://localhost:9090).

On the landing page you can start exploring metrics by creating [queries](https://prometheus.io/docs/prometheus/latest/querying/basics/). You can also verify what [targets](http://localhost:9090/targets) Prometheus currently monitors (Header Status > Targets), you should see Agones controller pod in the `kubernetes-pods` section.

> Metrics will be first registered when you will start using Agones.

Now let's install some Grafana dashboards.

### Grafana installation

Grafana is a open source time series analytics platform which supports Prometheus data source. We can also install easily import pre-built dashboards.

First we will install [Agones dashboard](#grafana-dashboards) as [config maps](https://kubernetes.io/docs/tasks/configure-pod-container/configure-pod-configmap/) in our cluster.

```bash
kubectl apply -f ../build/grafana/
```

Now we can install [grafana chart](https://github.com/helm/charts/tree/master/stable/grafana) from stable repository. (Replace `<your-admin-password>` with the admin password of your choice)

```bash
helm install --wait --name grafana stable/grafana --namespace metrics \
  --set adminPassword=<your-admin-password> -f ../build/grafana.yaml
```

This will install Grafana with our prepopulated dashboards and prometheus datasource [previously installed](#prometheus-installation)

> You can also use our {{< ghlink href="/build/Makefile" branch="master" >}}Makefile{{< /ghlink >}} targets (`setup-grafana`,`minikube-setup-grafana` and `kind-setup-grafana`).

Finally to access dashboards run

```bash
kubectl port-forward deployments/grafana 3000 -n metrics
```

Open a web browser to [http://127.0.0.1:3000](http://127.0.0.1:3000), you should see Agones [dashboards](#grafana-dashboards) after login as admin.

> Makefile targets `make grafana-portforward`,`make kind-grafana-portforward` and `make minikube-grafana-portforward`.

{{% feature publishVersion="0.8.0" %}}
### Stackdriver installation

In order to use [Stackdriver monitoring](https://app.google.stackdriver.com) you should [enable Stackdriver Monitoring API](https://cloud.google.com/monitoring/api/enable-api) on Google Cloud Console. You need to grant all the necessary permissions to the users (see [Access Control Guide](https://cloud.google.com/monitoring/access-control)). Stackdriver exporter uses a strategy called Application Default Credentials (ADC) to find your application's credentials. Details could be found here [Setting Up Authentication for Server to Server Production Applications](https://cloud.google.com/docs/authentication/production).

Note that Stackdriver monitoring is enabled by default on GKE clusters, however you can follow this [guide](https://cloud.google.com/kubernetes-engine/docs/how-to/monitoring#enabling_stackdriver_monitoring) if it was disabled on your GKE cluster.

Default metrics exporter is Prometheus. If you are using the [Helm installation]({{< ref "/docs/Installation/helm.md" >}}), you can install or upgrade Agones to use Stackdriver, using the following chart parameters:
```
helm upgrade --install --wait --set agones.metrics.stackdriverEnabled=true --set agones.metrics.prometheusEnabled=false --set agones.metrics.prometheusServiceDiscovery=false my-release-name agones/agones
```

With this configuration only Stackdriver exporter would be used instead of Prometheus exporter.

Create a Fleet or a Gameserver in order to check that connection with stackdriver API is configured properly and so that you will be able to see the metrics data.

Visit [Stackdriver monitoring](https://app.google.stackdriver.com) website, select your project, or choose `Create a new Workspace` and select GCP project where your cluster resides. In [Stackdriver metrics explorer](https://cloud.google.com/monitoring/charts/metrics-explorer) you should be able to find new metrics with prefix `agones/` (resource type is `Global`) after a couple of minutes. Choose the metrics you are interested in and add to a single or separate graphs. You can create multiple graphs, save them into your dashboard and use various aggregation parameters and reducers for each graph.

Example of the dashboard appearance is provided below:

![stackdriver monitoring dashboard](../../../images/stackdriver-metrics-dashboard.png)

Currently there exists only manual way of configuring Stackdriver Dashboard. So it is up to you to set an Alignment Period (minimal is 1 minute), GroupBy, Filter parameters and other graph settings.

#### Troubleshooting
If you can't see Agones metrics you should have a look at the controller logs for connection errors. Also ensure that your cluster has the necessary credentials to interact with Stackdriver Monitoring. You can configure `stackdriverProjectID` manually, if the automatic discovery is not working.

Permissions problem example from controller logs:
```
Failed to export to Stackdriver: rpc error: code = PermissionDenied desc = Permission monitoring.metricDescriptors.create denied (or the resource may not exist).
```
{{% /feature %}}
