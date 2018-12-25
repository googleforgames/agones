# Metrics

Agones controller exposes metrics via [OpenCensus](https://opencensus.io/). OpenCensus is a single distribution of libraries that collect metrics and distributed traces from your services, we only use it for metrics but it will allow us to support multiple exporters in the future.

We choose to start with Prometheus as this is the most popular with Kubernetes but it is also compatible with Stackdriver.
If you need another exporter, check the [list of supported](https://opencensus.io/exporters/supported-exporters/go/) exporters. It should be pretty straightforward to register a new one.(Github PR are more than welcomed)

We plan to support multiple exporters in the future via environement variables and helm flags.

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
Annotations required by this integration can be activated by setting the `agones.metrics.prometheusServiceDiscovery` to true (default) via the [helm chart value](../install/helm/README.md#configuration).

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

Grafana and Stackdriver - Coming Soon

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
