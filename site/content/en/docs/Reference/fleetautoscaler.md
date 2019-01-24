---
title: "Fleet Autoscaler Specification"
linkTitle: "Fleet Autoscaler"
date: 2019-01-03T03:58:55Z
description: "A `FleetAutoscaler`'s job is to automatically scale up and down a `Fleet` in response to demand."
weight: 30
---

A full `FleetAutoscaler` specification is available below and in the 
{{< ghlink href="examples/fleetautoscaler.yaml" >}}example folder{{< /ghlink >}} for reference :

```yaml
apiVersion: "stable.agones.dev/v1alpha1"
kind: FleetAutoscaler
metadata:
  name: fleet-autoscaler-example
spec:
  fleetName: fleet-example
  policy:
    type: Buffer
    buffer:
      bufferSize: 5
      minReplicas: 10
      maxReplicas: 20
```

Or for Webhook FleetAutoscaler below and in {{< ghlink href="examples/webhookfleetautoscaler.yaml" >}}example folder{{< /ghlink >}}:

```yaml
apiVersion: "stable.agones.dev/v1alpha1"
kind: FleetAutoscaler
metadata:
  name: fleet-autoscaler-example
spec:
  fleetName: fleet-example
  policy:
    type: Webhook
    webhook:
      name: "fleet-autoscaler-webhook"
      namespace: "default"
      path: "/scale"
```

Since Agones defines a new 
[Custom Resources Definition (CRD)](https://kubernetes.io/docs/concepts/api-extension/custom-resources/) 
we can define a new resource using the kind `FleetAutoscaler` with the custom group `stable.agones.dev` and API 
version `v1alpha1`.

The `spec` field is the actual `FleetAutoscaler` specification and it is composed as follows:

- `fleetName` is name of the fleet to attach to and control. Must be an existing `Fleet` in the same namespace
   as this `FleetAutoscaler`.
- `policy` is the autoscaling policy
  - `type` is type of the policy. "Buffer" and "Webhook" are available
  - `buffer` parameters of the buffer policy type
    - `bufferSize`  is the size of a buffer of "ready" game server instances
                    The FleetAutoscaler will scale the fleet up and down trying to maintain this buffer, 
                    as instances are being allocated or terminated
                    it can be specified either in absolute (i.e. 5) or percentage format (i.e. 5%)
    - `minReplicas` is the minimum fleet size to be set by this FleetAutoscaler. 
                    if not specified, the minimum fleet size will be bufferSize
    - `maxReplicas` is the maximum fleet size that can be set by this FleetAutoscaler. Required. 
  - `webhook` parameters of the webhook policy type
    - `service` is a reference to the service for this webhook. Either `service` or `url` must be specified. If the webhook is running within the cluster, then you should use `service`. Port 8000 will be used if it is open, otherwise it is an error.
      - `name`  is the service name bound to Deployment of autoscaler webhook. Required {{< ghlink href="examples/autoscaler-webhook/autoscaler-service.yaml" >}}(see example){{< /ghlink >}}
                      The FleetAutoscaler will scale the fleet up and down based on the response from this webhook server
      - `namespace` is the kubernetes namespace where webhook is deployed. Optional
                      If not specified, the "default" would be used
      - `path` is an optional URL path which will be sent in any request to this service. (i. e. /scale)
    - `url` gives the location of the webhook, in standard URL form (`[scheme://]host:port/path`). Exactly one of `url` or `service` must be specified. The `host` should not refer to a service running in the cluster; use the `service` field instead.  (optional, instead of service)
{{% feature publishVersion="0.8.0" %}}
<li>`caBundle` is a PEM encoded certificate authority bundle which is used to issue and then validate the webhook's server certificate. Base64 encoded PEM string. Required only for HTTPS. If not present HTTP client would be used.</li>
{{% /feature %}}

Note: only one `buffer` or `webhook` could be defined for FleetAutoscaler which is based on the `type` field.

# Webhook Endpoint Specification

Webhook endpoint is used to delegate the scaling logic to a separate pod or server.

FleetAutoscaler would send a request to the webhook endpoint every sync period (which is currently 30s) with a JSON body, and scale the target fleet based on the data that is returned.
JSON payload with a FleetAutoscaleReview data structure would be sent to webhook endpoint and received from it with FleetAutoscaleResponse field populated. FleetAutoscaleResponse contains target Replica count which would trigger scaling of the fleet according to it.

In order to define the path to your Webhook you can use either `URL` or `service`. Note that `caBundle` parameter is required if you use HTTPS for webhook fleetautoscaler, `caBundle` should be omitted if you want to use HTTP webhook server.

The connection to this webhook endpoint should be defined in `FleetAutoscaler` using Webhook policy type.

```go
// FleetAutoscaleReview is passed to the webhook with a populated Request value,
// and then returned with a populated Response.
type FleetAutoscaleReview struct {
	Request  *FleetAutoscaleRequest  `json:"request"`
	Response *FleetAutoscaleResponse `json:"response"`
}

type FleetAutoscaleRequest struct {
	// UID is an identifier for the individual request/response. It allows us to distinguish instances of requests which are
	// otherwise identical (parallel requests, requests when earlier requests did not modify etc)
	// The UID is meant to track the round trip (request/response) between the Autoscaler and the WebHook, not the user request.
	// It is suitable for correlating log entries between the webhook and apiserver, for either auditing or debugging.
	UID types.UID `json:"uid""`
	// Name is the name of the Fleet being scaled
	Name string `json:"name"`
	// Namespace is the namespace associated with the request (if any).
	Namespace string `json:"namespace"`
	// The Fleet's status values
	Status v1alpha1.FleetStatus `json:"status"`
}

type FleetAutoscaleResponse struct {
	// UID is an identifier for the individual request/response.
	// This should be copied over from the corresponding FleetAutoscaleRequest.
	UID types.UID `json:"uid"`
	// Set to false if no scaling should occur to the Fleet
	Scale bool `json:"scale"`
	// The targeted replica count
	Replicas int32 `json:"replicas"`
}

// FleetStatus is the status of a Fleet
type FleetStatus struct {
	// Replicas the total number of current GameServer replicas
	Replicas int32 `json:"replicas"`
	// ReadyReplicas are the number of Ready GameServer replicas
	ReadyReplicas int32 `json:"readyReplicas"`
	// AllocatedReplicas are the number of Allocated GameServer replicas
	AllocatedReplicas int32 `json:"allocatedReplicas"`
}
```

For Webhook Fleetautoscaler Policy either HTTP or HTTPS could be used. Switching between them occurs depending on https presence in `URL` or by presense of `caBundle`.
The example of the webhook written in Go could be found {{< ghlink href="examples/autoscaler-webhook/main.go" >}}here{{< /ghlink >}}.

It implements the {{< ghlink href="examples/autoscaler-webhook/" >}}scaling logic{{< /ghlink >}} based on the percentage of allocated gameservers in a fleet.

