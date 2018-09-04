# Quickstart Create a Fleet Autoscaler

This guide covers how you can quickly get started using Agones to create a Fleet 
Autoscaler to manage your fleet size automatically, based on actual load.

## Prerequisites

It is assumed that you have followed the instructions to [Create a Game Server Fleet](./create_fleet.md)
and you have a running fleet of game servers. 

## Objectives

- Create a Fleet Autoscaler in Kubernetes using Agones custom resource.
- Watch the Fleet scale up when allocating GameServers
- Watch the Fleet scale down when shutting down allocated GameServers
- Edit the autoscaler specification to apply live changes

### 1. Create a Fleet Autoscaler

Let's create a Fleet Autoscaler using the following command : 

```
kubectl apply -f https://raw.githubusercontent.com/GoogleCloudPlatform/agones/master/examples/simple-udp/fleetautoscaler.yaml
```

You should see a successful ouput similar to this :

```
fleetautoscaler.stable.agones.sev "simple-udp-autoscaler" created
```

This has created a FleetAutoscaler record inside Kubernetes.

### 2. See the autoscaler status.

```
kubectl describe fleetautoscaler simple-udp-autoscaler
``` 

It should look something like this:

```
Name:         simple-udp-autoscaler
Namespace:    default
Labels:       <none>
Annotations:  kubectl.kubernetes.io/last-applied-configuration={"apiVersion":"st
able.agones.dev/v1alpha1","kind":"FleetAutoscaler","metadata":{"annotations":{},
"name":"simple-udp-autoscaler","namespace":"default"},...
API Version:  stable.agones.dev/v1alpha1
Kind:         FleetAutoscaler
Metadata:
  Cluster Name:
  Creation Timestamp:  2018-10-02T15:19:58Z
  Generation:          1
  Owner References:
    API Version:           stable.agones.dev/v1alpha1
    Block Owner Deletion:  true
    Controller:            true
    Kind:                  Fleet
    Name:                  simple-udp
    UID:                   9960762e-c656-11e8-933e-fa163e07a1d4
  Resource Version:        6123197
  Self Link:               /apis/stable.agones.dev/v1alpha1/namespaces/default/f
leetautoscalers/simple-udp-autoscaler
  UID:                     9fd0efa1-c656-11e8-933e-fa163e07a1d4
Spec:
  Fleet Name:  simple-udp
  Policy:
    Buffer:
      Buffer Size:   2
      Max Replicas:  10
      Min Replicas:  2
    Type:            Buffer
Status:
  Able To Scale:     true
  Current Replicas:  2
  Desired Replicas:  2
  Last Scale Time:   <nil>
  Scaling Limited:   false
Events:              <none>
```

You can see the status (able to scale, not limited), the last time the fleet was scaled (nil for never)
and the current and desired fleet size. 

The autoscaler works by changing the desired size, and the fleet creates/deletes game server instances
to achieve to that number. The convergence is achieved in time, which is usually measured in seconds.

### 3. Allocate a Game Server from the Fleet 

If you're interested in more details for game server allocation, you should consult the [Create a Game Server Fleet](./create_fleet.md) page.
In here we are only interested in triggering allocations to see the autoscaler in action.

```
kubectl create -f https://raw.githubusercontent.com/GoogleCloudPlatform/agones/master/examples/simple-udp/fleetallocation.yaml -o yaml
```

You should get in return the allocated game server details, which should end with something like:
```
    status:
      address: 10.30.64.99
      nodeName: universal3
      ports:
      - name: default
        port: 7131
      state: Allocated
```

Note the address and port, you might need them later to connect to the server.

### 4. See the autoscaler in action

Now let's wait a few seconds to allow the autoscaler to detect the change in the fleet and check again its status

```
kubectl describe fleetautoscaler simple-udp-autoscaler
``` 

The last part should look something like this:

```
Spec:
  Fleet Name:  simple-udp
  Policy:
    Buffer:
      Buffer Size:   2
      Max Replicas:  10
      Min Replicas:  2
    Type:            Buffer
Status:
  Able To Scale:     true
  Current Replicas:  3
  Desired Replicas:  3
  Last Scale Time:   2018-10-02T16:00:02Z
  Scaling Limited:   false
Events:
  Type    Reason            Age   From                        Message
  ----    ------            ----  ----                        -------
  Normal  AutoScalingFleet  2m    fleetautoscaler-controller  Scaling fleet simple-udp from 2 to 3
```

You can see that the fleet size has increased, the autoscaler having compensated for the allocated instance.
Last Scale Time has been updated, and a scaling event has been logged.

Double-check the actual number of game server instances and status by running

```
kubectl get gs -o=custom-columns=NAME:.metadata.name,STATUS:.status.state,IP:.status.address,PORT:.status.ports
``` 

This will get you a list of all the current `GameSevers` and their `Status > State`.

```
NAME                     STATUS      IP             PORT
simple-udp-mzhrl-hz8wk   Allocated   10.30.64.99    [map[name:default port:7131]]
simple-udp-mzhrl-k6jg5   Ready       10.30.64.100   [map[name:default port:7243]]
simple-udp-mzhrl-n2sk2   Ready       10.30.64.168   [map[name:default port:7658]]
``` 

### 5. Shut the allocated instance down

Since we've only got one allocation, we'll just grab the details of the IP and port of the
only allocated `GameServer`: 

```
kubectl get $(kubectl get fleetallocation -o name) -o jsonpath='{.status.GameServer.status.GameServer.status.ports[0].port}'
```

This should output your Game Server IP address and port. (eg `10.130.65.208:7936`)

You can now communicate with the `GameServer`:

```
nc -u {IP} {PORT}
Hello World !
ACK: Hello World !
EXIT
```

You can finally type `EXIT` which tells the SDK to run the [Shutdown command](../sdks/README.md#shutdown), and therefore shuts down the `GameServer`.  
 
### 6. See the fleet scaling down

Now let's wait a few seconds to allow the autoscaler to detect the change in the fleet and check again its status

```
kubectl describe fleetautoscaler simple-udp-autoscaler
``` 

It should look something like this:

```
Spec:
  Fleet Name:  simple-udp
  Policy:
    Buffer:
      Buffer Size:   2
      Max Replicas:  10
      Min Replicas:  2
    Type:            Buffer
Status:
  Able To Scale:     true
  Current Replicas:  3
  Desired Replicas:  2
  Last Scale Time:   2018-10-02T16:09:02Z
  Scaling Limited:   false
Events:
  Type    Reason            Age   From                        Message
  ----    ------            ----  ----                        -------
  Normal  AutoScalingFleet  9m    fleetautoscaler-controller  Scaling fleet simple-udp from 2 to 3
  Normal  AutoScalingFleet  45s   fleetautoscaler-controller  Scaling fleet simple-udp from 3 to 2
```

You can see that the fleet size has decreased, the autoscaler adjusting to game server instance being de-allocated,
the Last Scale Time and the events have been updated. Note that simple-udp game server instance you just closed earlier
might stay a bit in 'Unhealthy' state (and its pod in 'Terminating' until it gets removed.

Double-check the actual number of game server instances and status by running

```
kubectl get gs -o=custom-columns=NAME:.metadata.name,STATUS:.status.state,IP:.status.address,PORT:.status.ports
``` 

This will get you a list of all the current `GameSevers` and their `Status > State`.

```
NAME                     STATUS    IP             PORT
simple-udp-mzhrl-k6jg5   Ready     10.30.64.100   [map[name:default port:7243]]
simple-udp-mzhrl-t7944   Ready     10.30.64.168   [map[port:7561 name:default]]
``` 

### 7. Change autoscaling parameters

We can also change the configuration of the `FleetAutoscaler` of the running `Fleet`, and have the changes
applied live, without interruptions of service.

Run `kubectl edit fleetautoscaler simple-udp-autoscaler` and set the `bufferSize` field to `5`. 

Let's look at the list of game servers again. Run `watch kubectl get gs -o=custom-columns=NAME:.metadata.name,STATUS:.status.state,IP:.status.address,PORT:.status.ports`
until you can see that are 5 ready server instances:

```
NAME                     STATUS    IP             PORT
simple-udp-mzhrl-7jpkp   Ready     10.30.64.100   [map[name:default port:7019]]
simple-udp-mzhrl-czt8v   Ready     10.30.64.168   [map[name:default port:7556]]
simple-udp-mzhrl-k6jg5   Ready     10.30.64.100   [map[name:default port:7243]]
simple-udp-mzhrl-nb8h2   Ready     10.30.64.168   [map[name:default port:7357]]
simple-udp-mzhrl-qspb6   Ready     10.30.64.99    [map[name:default port:7859]]
simple-udp-mzhrl-zg9rq   Ready     10.30.64.99    [map[name:default port:7745]]
```

## Next Steps

If you want to use your own GameServer container make sure you have properly integrated the [Agones SDK](../sdks/). 