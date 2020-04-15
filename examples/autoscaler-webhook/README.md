# Example Webhook Autoscaler Service

This service provides an example of the webhook fleetautoscaler service which is used to control the number of GameServers in a Fleet (`Replica` count).

## Autoscaler Service
The service exposes an endpoint which allows client calls to custom scaling logic.

When this endpoint is called, target Replica count gets calculated. If a fleet does not need to scale we return `Scale` equals `false`. Endpoint receives and returns the JSON encoded [`FleetAutoscaleReview`](../../docs/fleetautoscaler_spec.md#webhook-endpoint-specification) every SyncPeriod which is 30 seconds.

Note that scaling up logic is based on the percentage of allocated gameservers in a fleet. If this fraction is more than threshold (i. e. 0.7) then `Scale` parameter in `FleetAutoscaleResponse` is set to `true` and `Replica` value is returned increased by the `scaleFactor` (in this example twice) which results in creating more `Ready` GameServers. If the fraction below the threshold (i. e. 0.3) we decrease the count of gameservers in a fleet. There is a `minReplicasCount` parameters which defined the lower limit of the gameservers number in a Fleet.

To learn how to deploy the fleet to GKE, please see the tutorial [Create a Fleet (Go)](https://agones.dev/site/docs/getting-started/create-fleet/).

## Example flow

1. Fleet with 100 Replicas (gameservers) was created.
2. 70 gameservers got allocated -> No scaling for now.
3. One more server allocated, we got 71 allocated gameservers, fraction in a fleet is over 0.7 `AllocatedPart > 0.7` so we are scaling by `scaleFactor`. Which results in doubling fleet size.
4. Fleet now has 200 Replicas.
5. `AllocatedPart = 71/200 = 0.355` so no downscaling for now.
6. 22 gameservers were shutdown and now this count of gameservers is not in Allocated state.
7. `AllocatedPart = 59/200 = 0.295` Thus `AllocatedPart < 0.3` and fleet got scaled down.
Fleet now return to 100 gameservers size.
