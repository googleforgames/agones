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
apiVersion: "autoscaling.agones.dev/v1"
kind: FleetAutoscaler
# FleetAutoscaler Metadata
# {{< k8s-api-version href="#objectmeta-v1-meta" >}}
metadata:
  name: fleet-autoscaler-example
spec:
  # The name of the fleet to attach to and control. Must be an existing Fleet in the same namespace
  # as this FleetAutoscaler
  fleetName: fleet-example
  # The autoscaling policy
  policy:
    # type of the policy. for now, only Buffer is available
    type: Buffer
    # parameters of the buffer policy
    buffer:
      # Size of a buffer of "ready" game server instances
      # The FleetAutoscaler will scale the fleet up and down trying to maintain this buffer, 
      # as instances are being allocated or terminated
      # it can be specified either in absolute (i.e. 5) or percentage format (i.e. 5%)
      bufferSize: 5
      # minimum fleet size to be set by this FleetAutoscaler. 
      # if not specified, the actual minimum fleet size will be bufferSize
      minReplicas: 10
      # maximum fleet size that can be set by this FleetAutoscaler
      # required
      maxReplicas: 20
  # The autoscaling sync strategy
  sync:
    # type of the sync. for now, only FixedInterval is available
    type: FixedInterval
    # parameters of the fixedInterval sync
    fixedInterval:
      # the time in seconds between each auto scaling
      seconds: 30
```

Or for Webhook FleetAutoscaler below and in {{< ghlink href="examples/webhookfleetautoscaler.yaml" >}}example folder{{< /ghlink >}}:

```yaml
apiVersion: "autoscaling.agones.dev/v1"
kind: FleetAutoscaler
metadata:
  name: webhook-fleet-autoscaler
spec:
  fleetName: simple-game-server
  policy:
    # type of the policy - this example is Webhook
    type: Webhook
    # parameters for the webhook policy - this is a WebhookClientConfig, as per other K8s webhooks
    webhook:
      # use a service, or URL
      service:
        name: autoscaler-webhook-service
        namespace: default
        path: scale
      # optional for URL defined webhooks
      # url: ""
      # caBundle:  optional, used for HTTPS webhook type
  # The autoscaling sync strategy
  sync:
    # type of the sync. for now, only FixedInterval is available
    type: FixedInterval
    # parameters of the fixedInterval sync
    fixedInterval:
      # the time in seconds between each auto scaling
      seconds: 30
```

{{% feature publishVersion="1.37.0" %}}
Counter-based `FleetAutoscaler` specification below and in the {{< ghlink href="examples/counterfleetautoscaler.yaml" >}}example folder{{< /ghlink >}}:

```yaml
apiVersion: autoscaling.agones.dev/v1
kind: FleetAutoscaler
metadata:
  name: fleet-autoscaler-counter
spec:
  fleetName: fleet-example
  policy:
    type: Counter  # Counter based autoscaling
    counter:
      # Key is the name of the Counter. Required field.
      key: players
      # BufferSize is the size of a buffer of counted items that are available in the Fleet (available capacity).
      # Value can be an absolute number (ex: 5) or a percentage of the Counter available capacity (ex: 5%).
      # An absolute number is calculated from percentage by rounding up. Must be bigger than 0. Required field.
      bufferSize: 5
      # MinCapacity is the minimum aggregate Counter total capacity across the fleet.
      # If BufferSize is specified as a percentage, MinCapacity is required and cannot be 0.
      # If non zero, MinCapacity must be smaller than MaxCapacity and must be greater than or equal to BufferSize.
      minCapacity: 10
      # MaxCapacity is the maximum aggregate Counter total capacity across the fleet.
      # MaxCapacity must be greater than or equal to both MinCapacity and BufferSize. Required field.
      maxCapacity: 100
```

List-based `FleetAutoscaler` specification below and in the {{< ghlink href="examples/listfleetautoscaler.yaml" >}}example folder{{< /ghlink >}}:

```yaml
apiVersion: autoscaling.agones.dev/v1
kind: FleetAutoscaler
metadata:
  name: fleet-autoscaler-list
spec:
  fleetName: fleet-example
  policy:
    type: List  # List based autoscaling.
    list:
      # Key is the name of the List. Required field.
      key: rooms
      # BufferSize is the size of a buffer based on the List capacity that is available over the current
      # aggregate List length in the Fleet (available capacity).
      # It can be specified either as an absolute value (i.e. 5) or percentage format (i.e. 5%).
      # Must be bigger than 0. Required field.
      bufferSize: 5
      # MinCapacity is the minimum aggregate List total capacity across the fleet.
      # If BufferSize is specified as a percentage, MinCapacity is required must be greater than 0.
      # If non zero, MinCapacity must be smaller than MaxCapacity and must be greater than or equal to BufferSize.
      minCapacity: 10
      # MaxCapacity is the maximum aggregate List total capacity across the fleet.
      # MaxCapacity must be greater than or equal to both MinCapacity and BufferSize. Required field.
      maxCapacity: 100
```
{{% /feature %}}

Since Agones defines a new 
[Custom Resources Definition (CRD)](https://kubernetes.io/docs/concepts/api-extension/custom-resources/) 
we can define a new resource using the kind `FleetAutoscaler` with the custom group `autoscaling.agones.dev` 
and API version `v1`

{{% feature expiryVersion="1.37.0" %}}
The `spec` field is the actual `FleetAutoscaler` specification and it is composed as follows:

- `fleetName` is name of the fleet to attach to and control. Must be an existing `Fleet` in the same namespace
   as this `FleetAutoscaler`.
- `policy` is the autoscaling policy
  - `type` is type of the policy. "Buffer" and "Webhook" are available
  - `buffer` parameters of the buffer policy type
    - `bufferSize`  is the size of a buffer of "ready" and "reserved" game server instances.
                    The FleetAutoscaler will scale the fleet up and down trying to maintain this buffer, 
                    as instances are being allocated or terminated.
                    Note that "reserved" game servers could not be scaled down.
                    It can be specified either in absolute (i.e. 5) or percentage format (i.e. 5%)
    - `minReplicas` is the minimum fleet size to be set by this FleetAutoscaler. 
                    if not specified, the minimum fleet size will be bufferSize if absolute value is used.
                    When `bufferSize` in percentage format is used, `minReplicas` should be more than 0.
    - `maxReplicas` is the maximum fleet size that can be set by this FleetAutoscaler. Required. 
  - `webhook` parameters of the webhook policy type
    - `service` is a reference to the service for this webhook. Either `service` or `url` must be specified. If the webhook is running within the cluster, then you should use `service`. Port 8000 will be used if it is open, otherwise it is an error.
      - `name`  is the service name bound to Deployment of autoscaler webhook. Required {{< ghlink href="examples/autoscaler-webhook/autoscaler-service.yaml" >}}(see example){{< /ghlink >}}
                      The FleetAutoscaler will scale the fleet up and down based on the response from this webhook server
      - `namespace` is the kubernetes namespace where webhook is deployed. Optional
                      If not specified, the "default" would be used
      - `path` is an optional URL path which will be sent in any request to this service. (i. e. /scale)
      - `port` is optional, it is the port for the service which is hosting the webhook. The default is 8000 for backward compatibility. If given, it should be a valid port number (1-65535, inclusive).
    - `url` gives the location of the webhook, in standard URL form (`[scheme://]host:port/path`). Exactly one of `url` or `service` must be specified. The `host` should not refer to a service running in the cluster; use the `service` field instead.  (optional, instead of service)
    - `caBundle` is a PEM encoded certificate authority bundle which is used to issue and then validate the webhook's server certificate. Base64 encoded PEM string. Required only for HTTPS. If not present HTTP client would be used.
  - Note: only one `buffer` or `webhook` could be defined for FleetAutoscaler which is based on the `type` field.
- `sync` is autoscaling sync strategy. It defines when to run the autoscaling
  - `type` is type of the sync. For now only "FixedInterval" is available
  - `fixedInterval` parameters of the fixedInterval sync
    - `seconds` is the time in seconds between each auto scaling
{{% /feature %}}

{{% feature publishVersion="1.37.0" %}}
The `spec` field is the actual `FleetAutoscaler` specification and it is composed as follows:

- `fleetName` is name of the fleet to attach to and control. Must be an existing `Fleet` in the same namespace
   as this `FleetAutoscaler`.
- `policy` is the autoscaling policy
  - `type` is type of the policy. "Buffer" and "Webhook" are available
  - `buffer` parameters of the buffer policy type
    - `bufferSize`  is the size of a buffer of "ready" and "reserved" game server instances.
                    The FleetAutoscaler will scale the fleet up and down trying to maintain this buffer, 
                    as instances are being allocated or terminated.
                    Note that "reserved" game servers could not be scaled down.
                    It can be specified either in absolute (i.e. 5) or percentage format (i.e. 5%)
    - `minReplicas` is the minimum fleet size to be set by this FleetAutoscaler. 
                    if not specified, the minimum fleet size will be bufferSize if absolute value is used.
                    When `bufferSize` in percentage format is used, `minReplicas` should be more than 0.
    - `maxReplicas` is the maximum fleet size that can be set by this FleetAutoscaler. Required. 
  - `webhook` parameters of the webhook policy type
    - `service` is a reference to the service for this webhook. Either `service` or `url` must be specified. If the webhook is running within the cluster, then you should use `service`. Port 8000 will be used if it is open, otherwise it is an error.
      - `name`  is the service name bound to Deployment of autoscaler webhook. Required {{< ghlink href="examples/autoscaler-webhook/autoscaler-service.yaml" >}}(see example){{< /ghlink >}}
                      The FleetAutoscaler will scale the fleet up and down based on the response from this webhook server
      - `namespace` is the kubernetes namespace where webhook is deployed. Optional
                      If not specified, the "default" would be used
      - `path` is an optional URL path which will be sent in any request to this service. (i. e. /scale)
      - `port` is optional, it is the port for the service which is hosting the webhook. The default is 8000 for backward compatibility. If given, it should be a valid port number (1-65535, inclusive).
    - `url` gives the location of the webhook, in standard URL form (`[scheme://]host:port/path`). Exactly one of `url` or `service` must be specified. The `host` should not refer to a service running in the cluster; use the `service` field instead.  (optional, instead of service)
    - `caBundle` is a PEM encoded certificate authority bundle which is used to issue and then validate the webhook's server certificate. Base64 encoded PEM string. Required only for HTTPS. If not present HTTP client would be used.
  - Note: only one `buffer` or `webhook` could be defined for FleetAutoscaler which is based on the `type` field.
  - `counter` parameters of the counter policy type
    - `counter` contains the settings for counter-based autoscaling:
      - `key` is the name of the counter to use for scaling decisions.
      - `bufferSize` is the size of a buffer of counted items that are available in the Fleet (available capacity). Value can be an absolute number or a percentage of desired game server instances. An absolute number is calculated from percentage by rounding up. Must be bigger than 0.
      - `minCapacity` is the minimum aggregate Counter total capacity across the fleet. If zero, MinCapacity is ignored. If non zero, MinCapacity must be smaller than MaxCapacity and bigger than BufferSize.
      - `maxCapacity` is the maximum aggregate Counter total capacity across the fleet. It must be bigger than both MinCapacity and BufferSize.
  - `list` parameters of the list policy type
    - `list` contains the settings for list-based autoscaling:
      - `key` is the name of the list to use for scaling decisions.
      - `bufferSize` is the size of a buffer based on the List capacity that is available over the current aggregate List length in the Fleet (available capacity). It can be specified either as an absolute value or percentage format.
      - `minCapacity` is the minimum aggregate List total capacity across the fleet. If zero, it is ignored. If non zero, it must be smaller than MaxCapacity and bigger than BufferSize.
      - `maxCapacity` is the maximum aggregate List total capacity across the fleet. It must be bigger than both MinCapacity and BufferSize. Required field.
- `sync` is autoscaling sync strategy. It defines when to run the autoscaling
  - `type` is type of the sync. For now only "FixedInterval" is available
  - `fixedInterval` parameters of the fixedInterval sync
    - `seconds` is the time in seconds between each auto scaling
{{% /feature %}}
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
	Status v1.FleetStatus `json:"status"`
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
	// ReservedReplicas are the total number of Reserved GameServer replicas in this fleet.
	// Reserved instances won't be deleted on scale down, but won't cause an autoscaler to scale up.
	ReservedReplicas int32 `json:"reservedReplicas"`
	// AllocatedReplicas are the number of Allocated GameServer replicas
	AllocatedReplicas int32 `json:"allocatedReplicas"`
}
```

For Webhook Fleetautoscaler Policy either HTTP or HTTPS could be used. Switching between them occurs depending on https presence in `URL` or by the presence of `caBundle`.
The example of the webhook written in Go could be found {{< ghlink href="examples/autoscaler-webhook/main.go" >}}here{{< /ghlink >}}.

It implements the {{< ghlink href="examples/autoscaler-webhook/" >}}scaling logic{{< /ghlink >}} based on the percentage of allocated gameservers in a fleet.
