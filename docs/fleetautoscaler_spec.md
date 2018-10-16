# Fleet Autoscaler Specification

A `FleetAutoscaler`'s job is to automatically scale up and down a `Fleet` in response to demand.

A full `FleetAutoscaler` specification is available below and in the 
[example folder](../examples/fleetautoscaler.yaml) for reference :

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

Since Agones defines a new 
[Custom Resources Definition (CRD)](https://kubernetes.io/docs/concepts/api-extension/custom-resources/) 
we can define a new resource using the kind `FleetAutoscaler` with the custom group `stable.agones.dev` and API 
version `v1alpha1`.

The `spec` field is the actual `FleetAutoscaler` specification and it is composed as follows:

- `fleetName` is name of the fleet to attach to and control. Must be an existing `Fleet` in the same namespace
   as this `FleetAutoscaler`.
- `policy` is the autoscaling policy
  - `type` is type of the policy. For now, only "Buffer" is available
  - `buffer` parameters of the buffer policy
    - `bufferSize`  is the size of a buffer of "ready" game server instances
                    The FleetAutoscaler will scale the fleet up and down trying to maintain this buffer, 
                    as instances are being allocated or terminated
                    it can be specified either in absolute (i.e. 5) or percentage format (i.e. 5%)
    - `minReplicas` is the minimum fleet size to be set by this FleetAutoscaler. 
                    if not specified, the minimum fleet size will be bufferSize
    - `maxReplicas` is the maximum fleet size that can be set by this FleetAutoscaler. Required. 