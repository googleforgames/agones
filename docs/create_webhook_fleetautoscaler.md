# Quickstart Create a Fleet Autoscaler with Webhook Policy

This guide covers how you can create webhook fleet autoscaler policy.
The main difference from the Buffer policy is that the logic on how many target replicas you need is delegated to a separate pod.
This type of Autoscaler would send an HTTP request to the webhook endpoint every sync period (which is currently 30s) with a JSON body, and scale the target fleet based on the data that is returned.

## Prerequisites

It is assumed that you have read the instructions to [Create a Game Server Fleet](./create_fleet.md)
and you have a running fleet of game servers or you could run command from Step #1.

## Objectives

- Run a fleet
- Deploy the Webhook Pod and service for autoscaling
- Create a Fleet Autoscaler with Webhook policy type in Kubernetes using Agones custom resource
- Watch the Fleet scales up when allocating GameServers
- Watch the Fleet scales down after GameServer shutdown

### 1. Deploy the fleet

Run a fleet in a cluster:
```
kubectl apply -f https://raw.githubusercontent.com/GoogleCloudPlatform/agones/master/examples/simple-udp/fleet.yaml
```

### 2. Deploy a Webhook service for autoscaling

In this step we would deploy an example webhook which will control the size of the fleet based on allocated gameservers portion in a fleet. You can see the source code for this example webhook server [here](../examples/autoscaler-webhook/main.go). The fleetautoscaler would trigger this endpoint every 30 seconds. More details could be found [also here](../examples/autoscaler-webhook/README.md).
We need to create a pod which will handle HTTP requests with json payload [`FleetAutoscaleReview`](./fleetautoscaler_spec.md#webhook-endpoint-specification) and return back it with [`FleetAutoscaleResponse`](./fleetautoscaler_spec.md#webhook-endpoint-specification) populated.

The `Scale` flag and `Replicas` values returned in the `FleetAutoscaleResponse` and `Replicas` value tells the FleetAutoscaler what target size the backing Fleet should be scaled up or down to. If `Scale` is false - no scalling occurs.

Run next command to create a service and a Webhook pod in a cluster:
```
kubectl apply -f https://raw.githubusercontent.com/GoogleCloudPlatform/agones/master/examples/autoscaler-webhook/autoscaler-service.yaml
```

To check that it is running and liveness probe is fine:
```
kubectl describe pod autoscaler-webhook
```

```
Name:           autoscaler-webhook-86944884c4-sdtqh
Namespace:      default
Node:           gke-test-cluster-default-1c5dec79-h0tq/10.138.0.2
...
Status:         Running
```

### 3. Create a Fleet Autoscaler

Let's create a Fleet Autoscaler using the following command:

```
kubectl apply -f https://raw.githubusercontent.com/GoogleCloudPlatform/agones/master/examples/webhookfleetautoscaler.yaml
```

You should see a successful ouput similar to this:

```
fleetautoscaler.stable.agones.sev "webhook-fleet-autoscaler" created
```

This has created a FleetAutoscaler record inside Kubernetes.
It has the link to Webhook service we deployed above.

### 4. See the fleet and autoscaler status.

In order to track the list of gameservers which run in your fleet you can run this command in a separate terminal tab:

```
 watch "kubectl get gs -n default"
```

In order to get autoscaler status use the following command:

```
kubectl describe fleetautoscaler webhook-fleet-autoscaler
```

It should look something like this:

```
Name:         webhook-fleet-autoscaler
Namespace:    default
Labels:       <none>
Annotations:  kubectl.kubernetes.io/last-applied-configuration={"apiVersion":"stable.agones.dev/v1alpha1","kind":"FleetAutoscaler","metadata":{"annotations":{},"name":"webhook-fleet-autoscaler","namespace":"default...
API Version:  stable.agones.dev/v1alpha1
Kind:         FleetAutoscaler
etadata:
  Cluster Name:
  Creation Timestamp:  2018-12-22T12:52:23Z
  Generation:          1
  Resource Version:    2274579
  Self Link:           /apis/stable.agones.dev/v1alpha1/namespaces/default/fleetautoscalers/webhook-fleet-autoscaler
  UID:                 6d03eae4-05e8-11e9-84c2-42010a8a01c9
Spec:
  Fleet Name:  simple-udp
  Policy:
    Type:  Webhook
    Webhook:
      Service:
        Name:       autoscaler-webhook-service
        Namespace:  default
        Path:       scale
      URL:
Status:
  Able To Scale:     true
  Current Replicas:  2
  Desired Replicas:  2
  Last Scale Time:   <nil>
  Scaling Limited:   false
Events:              <none>
```

You can see the status (able to scale, not limited), the last time the fleet was scaled (nil for never), current and desired fleet size.

The autoscaler make a query to a webhoook service deployed on step 1 and on response changing the target Replica size, and the fleet creates/deletes game server instances
to achieve that number. The convergence is achieved in time, which is usually measured in seconds.

### 5. Allocate Game Servers from the Fleet to trigger scale up

If you're interested in more details for game server allocation, you should consult the [Create a Game Server Fleet](./create_fleet.md) page.
Here we only interested in triggering allocations to see the autoscaler in action.

```
kubectl create -f https://raw.githubusercontent.com/GoogleCloudPlatform/agones/master/examples/simple-udp/fleetallocation.yaml -o yaml
```

You should get in return the allocated game server details, which should end with something like:
```
    status:
      address: 35.247.13.175
      nodeName: gke-test-cluster-default-1c5dec79-qrqv
      ports:
      - name: default
        port: 7047
      state: Allocated
```

Note the address and port, you might need them later to connect to the server.

Run the kubectl command one more time so that we have both servers allocated:
```
kubectl create -f https://raw.githubusercontent.com/GoogleCloudPlatform/agones/master/examples/simple-udp/fleetallocation.yaml -o yaml
```

### 6. Check new Autoscaler and Fleet status

Now let's wait a few seconds to allow the autoscaler to detect the change in the fleet and check again its status

```
kubectl describe fleetautoscaler webhook-fleet-autoscaler
```

The last part should look similar to this:

```
Spec:
  Fleet Name:  simple-udp
  Policy:
    Type:  Webhook
    Webhook:
      Service:
        Name:       autoscaler-webhook-service
        Namespace:  default
        Path:       scale
      URL:
Status:
  Able To Scale:     true
  Current Replicas:  4
  Desired Replicas:  4
  Last Scale Time:   2018-12-22T12:53:47Z
  Scaling Limited:   false
Events:
  Type    Reason            Age   From                        Message
  ----    ------            ----  ----                        -------
  Normal  AutoScalingFleet  35s   fleetautoscaler-controller  Scaling fleet simple-udp from 2 to 4
```

You can see that the fleet size has increased in particular case doubled to 4 gameservers (based on our custom logic in our webhook), the autoscaler having compensated for the two allocated instances.
Last Scale Time has been updated and a scaling event has been logged.

Double-check the actual number of game server instances and status by running:

```
 kubectl get gs -n default
```

This will get you a list of all the current `GameSevers` and their `Status > State`.

```
NAME                     STATUS      IP              PORT
simple-udp-dmkp4-8pkk2   Ready       35.247.13.175   [map[name:default port:7386]]
simple-udp-dmkp4-b7x87   Allocated   35.247.13.175   [map[name:default port:7219]]
simple-udp-dmkp4-r4qtt   Allocated   35.247.13.175   [map[name:default port:7220]]
simple-udp-dmkp4-rsr6n   Ready       35.247.13.175   [map[name:default port:7297]]
```

### 7. Check down scaling using Webhook Autoscaler policy

Based on our custom webhook deployed earlier, if the fraction of allocated replicas in whole Replicas count would be less that threshold (0.3) than fleet would scale down by scaleFactor, in our example by 2.

Note that example webhook server have a limitation that it would not decrease fleet replica count under `minReplicasCount`, which is equal to 2.

We need to run EXIT command on one gameserver (Use IP address and port of the allocated gameserver from the previous step) in order to decrease the number of allocated gameservers in a fleet (<0.3).
```
nc -u 35.247.13.175 7220
EXIT
```

Server would be in shutdown state.
Wait about 30 seconds.
Then you should see scaling down event in the output of next command:
```
kubectl describe fleetautoscaler webhook-fleet-autoscaler
```

You should see these lines in events:
```
  Normal   AutoScalingFleet  11m                fleetautoscaler-controller  Scaling fleet simple-udp from 2 to 4
  Normal   AutoScalingFleet  1m                 fleetautoscaler-controller  Scaling fleet simple-udp from 4 to 2
```

And get gameservers command output:
```
kubectl get gs -n default
```

```
NAME                     STATUS      IP               PORT
simple-udp-884fg-6q5sk   Ready       35.247.117.202   7373
simple-udp-884fg-b7l58   Allocated   35.247.117.202   7766
```

## Next Steps

Read the advanced [Scheduling and Autoscaling](scheduling_autoscaling.md) guide, for more details on autoscaling.

If you want to use your own GameServer container make sure you have properly integrated the [Agones SDK](../sdks/).
