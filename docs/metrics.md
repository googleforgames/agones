# Metrics

Agones controller exposes metrics via [OpenCensus](https://opencensus.io/). OpenCensus is a single distribution of libraries that collect metrics and distributed traces from your services, we only use it for metrics but it will allow us to support multiple exporters in the future.

We choose to start with [Prometheus](https://prometheus.io/) as this is the most popular with Kubernetes but it is also compatible with Stackdriver.
If you need another exporter, check the [list of supported](https://opencensus.io/exporters/supported-exporters/go/) exporters. It should be pretty straightforward to register a new one.(Github PR are more than welcomed)

We plan to support multiple exporters in the future via environement variables and helm flags.

Table of Contents
=================
  - [Backend integrations](#backend-integrations)
    - [Prometheus](#prometheus)
    - [Prometheus Operator](#prometheus-operator)
    - [Stackdriver](#stackdriver)
  - [Metrics available](#metrics-available)
  - [Dashboard](#dashboard)
    - [Grafana Dashboards](#grafana-dashboards)
  - [Installation](#installation)
    - [Prometheus installation](#prometheus-installation)
    - [Grafana installation](#grafana-installation)
  - [Adding more metrics](#adding-more-metrics)
  
## Backend integrations

### Prometheus

If you are running a [Prometheus](https://prometheus.io/) intance you just need to ensure that metrics and kubernetes service discovery are enabled. (helm chart values `agones.metrics.enabled` and `agones.metrics.prometheusServiceDiscovery`). This will automatically add annotations required by Prometheus to discover Agones metrics and start collecting them. (see [example](https://github.com/prometheus/prometheus/tree/master/documentation/examples/kubernetes-rabbitmq))

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

We don't yet support the [OpenCensus Stackdriver exporter](https://opencensus.io/exporters/supported-exporters/go/stackdriver/) but you can still use the Prometheus Stackdriver integration by following these [instructions](https://cloud.google.com/monitoring/kubernetes-engine/prometheus).
Annotations required by this integration can be activated by setting the `agones.metrics.prometheusServiceDiscovery` to true (default) via the [helm chart value](../install/helm/agones/README.md#configuration).

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

### [Grafana](https://grafana.com/) Dashboards

We provide a set of useful [Grafana](https://grafana.com/) dashboards to monitor Agones workload, they are located under the [grafana folder](../build/grafana):

- [Agones Autoscalers](../build/grafana/dashboard-autoscalers.yaml) allows you to monitor your current autoscalers replicas request as well as fleet replicas allocation and readyness statuses. You can only select one autoscaler at the time using the provided dropdown.

- [Agones GameServers](../build/grafana/dashboard-gameservers.yaml) displays your current game servers workload status (allocations , game servers statuses, fleets replicas) with optional fleet name filtering.

- [Agones Status](../build/grafana/dashboard-status.yaml) displays Agones controller health status.

- [Agones Controller Resource Usage](../build/grafana/dashboard-controller-usage.yaml) displays Agones Controller CPU and memory usage and also some Golang runtime metrics.

Dashboard screenshots :

![](grafana-dashboard-autoscalers.png)

![](grafana-dashboard-controller.png)

> You can import our dashboards by copying the json content from [each config map](../build/grafana) into your own instance of Grafana (+ > Create > Import > Or paste json) or follow the [installation](#installation) guide.

## Installation

When operating a live multiplayer game you will need to observe performances, resource usage and availability to learn more about your system. This guide will explain how you can setup Prometheus and Grafana into your own Kubernetes cluster to monitor your Agones workload.

Before attemping this guide you should make sure you have [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/) and [helm](https://docs.helm.sh/using_helm/) installed and configured to reach your kubernetes cluster.

### Prometheus installation

Prometheus is an open source monitoring solution, we will use it to store Agones controller metrics and query back the data.

Let's install Prometheus using the [helm stable](https://github.com/helm/charts/tree/master/stable/prometheus) repository.

```bash
helm install --wait --name prom stable/prometheus --namespace metrics \
  --set pushgateway.enabled=false \
  --set kubeStateMetrics.enabled=false,nodeExporter.enabled=false
```

> You can also run our [Makefile](../build/Makefile) target `make setup-prometheus` or `make kind-setup-prometheus` and `make minikube-setup-prometheus` for [Kind](../build/README.md#running-a-test-kind-cluster) and [Minikube](../build/README.md#running-a-test-minikube-cluster).

By default we will disable the push gateway (we don't need it for Agones) and other exporters.

The helm [chart](https://github.com/helm/charts/tree/master/stable/prometheus) support [nodeSelector](https://kubernetes.io/docs/concepts/configuration/assign-pod-node/#nodeselector), [affinity](https://kubernetes.io/docs/concepts/configuration/assign-pod-node/#affinity-and-anti-affinity) and [toleration](https://kubernetes.io/docs/concepts/configuration/taint-and-toleration/), you can use them to schedule prometheus deployments on an isolated node(s) to have an homogeneous game servers workload.

This will install a Prometheus Server in your current cluster with [Persistent Volume Claim](https://kubernetes.io/docs/concepts/storage/persistent-volumes/) (Deactivated for Minikube and Kind) for storing and querying time series, it will automatically start collecting metrics from Agones Controller.

Finally to access Prometheus metrics, rules and alerts explorer use

```bash
kubectl port-forward deployments/prom-prometheus-server 9090 -n metrics
```

> Again you can use our Makefile [`make  prometheus-portforward`](../build/Makefile/README.md#prometheus-portforward).(For [Kind](../build/README.md#running-a-test-kind-cluster) and [Minikube](../build/README.md#running-a-test-minikube-cluster) use their specific targets `make kind-prometheus-portforward` and `make minikube-prometheus-portforward`)

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

> You can also use our [Makefile](../build/Makefile) targets (`setup-grafana`,`minikube-setup-grafana` and `kind-setup-grafana`).

Finally to access dashboards run

```bash
kubectl port-forward deployments/grafana 3000 -n metrics
```

Open a web browser to [http://127.0.0.1:3000](http://127.0.0.1:3000), you should see Agones [dashboards](#grafana-dashboards) after login as admin.

> Makefile targets `make grafana-portforward`,`make kind-grafana-portforward` and `make minikube-grafana-portforward`.

## Adding more metrics

If you want to contribute and add more metrics we recommend to use shared informers (cache) as it is currently implemented in the [metrics controller](../pkg/metrics/controller.go). Using shared informers allows to keep metrics code in one place and doesn't overload the Kubernetes API.

However there is some cases where you will have to add code inside your ressource controller (eg. latency metrics), you should minize metrics code in your controller by adding specific functions in the metrics packages as shown below.

```golang
package metrics

import "go.opencensus.io/stats"

...

func RecordSomeLatency(latency int64,ressourceName string) {
    stats.RecordWithTags(....)
}
```
