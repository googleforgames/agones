# Quickstart Create a Game Server Fleet

This guide covers how you can quickly get started using Agones to create a Fleet
of warm GameServers ready for you to allocate out of and play on!

## Prerequisites

The following prerequisites are required to create a GameServer:

1. A Kubernetes cluster with the UDP port range 7000-8000 open on each node.
2. Agones controller installed in the targeted cluster
3. kubectl properly configured
4. Netcat which is already installed on most Linux/macOS distributions, for windows you can use [WSL](https://docs.microsoft.com/en-us/windows/wsl/install-win10).

>NOTE: Agones required Kubernetes versions 1.9+ to run. See the [cluster requirements](../README.md#requirements) for more details.

If you don't have a Kubernetes cluster you can follow [these instructions](../install/README.md) to create a cluster on Google Kubernetes Engine (GKE), Minikube or Azure Kubernetes Service (AKS), and install Agones.

For the purpose of this guide we're going to use the [simple-udp](../examples/simple-udp/) example as the GameServer container. This example is very simple UDP server written in Go. Don't hesitate to look at the code of this example for more information.

While not required, you may wish to go through the [Create a Game Server](create_gameserver.md) quickstart before this one.

## Objectives

- Create a Fleet in Kubernetes using Agones custom resource.
- Scale the Fleet up from it's initial configuration.
- Request a GameServer allocation from the Fleet to play on.
- Connect to the allocated GameServer.
- Deploy a new GameServer configuration to the Fleet

### 1. Create a Fleet

Let's create a Fleet using the following command :

```
kubectl apply -f https://raw.githubusercontent.com/GoogleCloudPlatform/agones/master/examples/simple-udp/fleet.yaml
```

You should see a successful ouput similar to this :

```
fleet "simple-udp" created
```

This has created a Fleet record inside Kubernetes, which in turn creates two warm [GameServers](gameserver_spec.md) to
be available to being allocated for usage for a game session.

```
kubectl get fleet
```
It should look something like this:

```
NAME         AGE
simple-udp   5m
```

You can also see the GameServers that have been created by the Fleet by running `kubectl get gameservers`,
the GameServer will be prefixed by `simple-udp`.

```
NAME                     AGE
simple-udp-xvp4n-jvhbm   36s
simple-udp-xvp4n-x6z5m   36s
```

For the full details of the YAML file head to the [Fleet Specification Guide](./fleet_spec.md#fleet-specification)

### 2. Fetch the Fleet status

Let's wait for the two `GameServers` to become ready.

```
watch kubectl describe fleet simple-udp
```

```
Name:         simple-udp
Namespace:    default
Labels:       <none>
Annotations:  kubectl.kubernetes.io/last-applied-configuration={"apiVersion":"stable.agones.dev/v1alpha1","kind":"Fleet","metadata":{"annotations":{},"name":"simple-udp","namespace":"default"},"spec":{"replicas":2,...
API Version:  stable.agones.dev/v1alpha1
Kind:         Fleet
Metadata:
  Cluster Name:
  Creation Timestamp:  2018-07-01T18:55:35Z
  Generation:          1
  Resource Version:    24685
  Self Link:           /apis/stable.agones.dev/v1alpha1/namespaces/default/fleets/simple-udp
  UID:                 56710a91-7d60-11e8-b2dd-08002703ef08
Spec:
  Replicas:  2
  Strategy:
    Rolling Update:
      Max Surge:        25%
      Max Unavailable:  25%
    Type:               RollingUpdate
  Template:
    Metadata:
      Creation Timestamp:  <nil>
    Spec:
      Health:
      Ports:
        Container Port:  7654
        Name:            default
        Port Policy:     dynamic
      Template:
        Metadata:
          Creation Timestamp:  <nil>
        Spec:
          Containers:
            Image:  gcr.io/agones-images/udp-server:0.4
            Name:   simple-udp
            Resources:
Status:
  Allocated Replicas:  0
  Ready Replicas:      2
  Replicas:            2
Events:
  Type    Reason                 Age   From              Message
  ----    ------                 ----  ----              -------
  Normal  CreatingGameServerSet  13s   fleet-controller  Created GameServerSet simple-udp-wlqnd
```

If you look towards the bottom, you can see there is a section of `Status > Ready Replicas` which will tell you
how many `GameServers` are currently in a Ready state. After a short period, there should be 2 `Ready Replicas`.

### 3. Scale up the Fleet

Let's scale up the `Fleet` from 2 `replicates` to 5.

Run `kubectl edit fleet simple-udp`, which will open an editor for you to edit the Fleet configuration.

Scroll down to the `spec > replicas` section, and change the values of `replicas: 2` to `replicas: 5`.

Save the file and exit - this will apply the changes.

If we now run `kubectl get gameservers` we should see 5 `GameServers` prefixed by `simple-udp`.

```
NAME                     AGE
simple-udp-xvp4n-jvhbm   11m
simple-udp-xvp4n-x6z5m   11m
simple-udp-xvp4n-z8znu   36s
simple-udp-xvp4n-a6z0e   36s
simple-udp-xvp4n-i6bnm   36s
```

### 4. Allocate a Game Server from the Fleet

Since we have a fleet of warm gameservers, we need a way to request one of them for usage!

We can do this through a `FleetAllocation`, which will both return to us a `GameServer` (assuming one is available)
and also move it to the `Allocated` state.

In production, you would likely do this through a [Kubernetes API call](./access_api.md), but we can also
do this through `kubectl` as well, and ask it to return the response in yaml so that we can see what has happened.

```
kubectl create -f https://raw.githubusercontent.com/GoogleCloudPlatform/agones/master/examples/simple-udp/fleetallocation.yaml -o yaml
```

For the full details of the YAML file head to the [Fleet Specification Guide](./fleet_spec.md#fleet-allocation-specification)

You should get back a response that looks like the following:

```
apiVersion: stable.agones.dev/v1alpha1
kind: FleetAllocation
metadata:
  clusterName: ""
  creationTimestamp: 2018-07-01T18:56:31Z
  generateName: simple-udp-
  generation: 1
  name: simple-udp-l7dn9
  namespace: default
  ownerReferences:
  - apiVersion: stable.agones.dev/v1alpha1
    blockOwnerDeletion: true
    controller: true
    kind: GameServer
    name: simple-udp-wlqnd-s2xr5
    uid: 5676a611-7d60-11e8-b2dd-08002703ef08
  resourceVersion: "24719"
  selfLink: /apis/stable.agones.dev/v1alpha1/namespaces/default/fleetallocations/simple-udp-l7dn9
  uid: 77c22f17-7d60-11e8-b2dd-08002703ef08
spec:
  fleetName: simple-udp
status:
  GameServer:
    metadata:
      creationTimestamp: 2018-07-01T18:55:35Z
      finalizers:
      - stable.agones.dev
      generateName: simple-udp-wlqnd-
      generation: 1
      labels:
        stable.agones.dev/gameserverset: simple-udp-wlqnd
      name: simple-udp-wlqnd-s2xr5
      namespace: default
      ownerReferences:
      - apiVersion: stable.agones.dev/v1alpha1
        blockOwnerDeletion: true
        controller: true
        kind: GameServerSet
        name: simple-udp-wlqnd
        uid: 56731f1a-7d60-11e8-b2dd-08002703ef08
      resourceVersion: "24716"
      selfLink: /apis/stable.agones.dev/v1alpha1/namespaces/default/gameservers/simple-udp-wlqnd-s2xr5
      uid: 5676a611-7d60-11e8-b2dd-08002703ef08
    spec:
      container: simple-udp
      health:
        failureThreshold: 3
        initialDelaySeconds: 5
        periodSeconds: 5
      ports:
      - containerPort: 7654
        hostPort: 7604
        name: default
        portPolicy: dynamic
        protocol: UDP
      template:
        metadata:
          creationTimestamp: null
        spec:
          containers:
          - image: gcr.io/agones-images/udp-server:0.4
            name: simple-udp
            resources: {}
    status:
      address: 192.168.99.100
      nodeName: agones
      ports:
      - name: default
        port: 7604
      state: Allocated
```

If you see the `status` section, you should see that there is a `GameServer`, and if you look at its
`status > state` value, you can also see that it has been moved to `Allocated`. This means you have been successfully
allocated a `GameServer` out of the fleet, and you can now connect your players to it!

A handy trick for checking to see how many `GameServers` you have `Allocated` vs `Ready`, run the following:

```
kubectl get gs -o=custom-columns=NAME:.metadata.name,STATUS:.status.state,IP:.status.address,PORT:.status.ports
```

This will get you a list of all the current `GameSevers` and their `Status > State`.

```
NAME                     STATUS      IP               PORT
simple-udp-tfqn7-c9tqz   Ready       192.168.39.150   [map[name:default port:7136]]
simple-udp-tfqn7-g8fhq   Allocated   192.168.39.150   [map[name:default port:7148]]
simple-udp-tfqn7-p8wnl   Ready       192.168.39.150   [map[name:default port:7453]]
simple-udp-tfqn7-t6bwp   Ready       192.168.39.150   [map[name:default port:7228]]
simple-udp-tfqn7-wkb7b   Ready       192.168.39.150   [map[name:default port:7226]]
```

### 5. Scale down the Fleet

Not only can we scale our fleet up, but we can scale it down as well.

The nice thing about Agones, is that it is smart enough to know when `GameServers` have been moved to `Allocated`
and will automatically leave them running on scale down -- as we assume that players are playing on this game server,
and we shouldn't disconnect them!

Let's scale down our Fleet to 0 (yep! you can do that!), and watch what happens.
Run `kubectl edit fleet simple-udp` and replace `replicas: 5` with `replicas 0`, save the file and exit your editor.

It may take a moment for all the `GameServers` to shut down, so let's watch them all and see what happens:
```
watch kubectl get gs -o=custom-columns=NAME:.metadata.name,STATUS:.status.state,IP:.status.address,PORT:.status.ports
```

Eventually, one by one they will be removed from the list, and you should simply see:

```
NAME                     STATUS      IP               PORT
simple-udp-tfqn7-g8fhq   Allocated   192.168.39.150   [map[name:default port:7148]]
```

That lone `Allocated` `GameServer` is left all alone, but still running!

If you would like, try editing the `Fleet` configuration `replicas` field and watch the list of `GameServers`
grow and shrink.

### 6. Connect to the GameServer

Since we've only got one allocation, we'll just grab the details of the IP and port of the
only allocated `GameServer`:

```
kubectl get $(kubectl get fleetallocation -o name) -o jsonpath='{.status.GameServer.staatus.GameServer.status.ports[0].port}'
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

If you run `kubectl describe gs | grep State` again - either the GameServer will be replaced with a new, `Ready` `GameServer`
, or it will be in `Shutdown` state, on the way to being deleted.

Since we are running a `Fleet`, Agones will always do it's best to ensure there are always the configured number
of `GameServers` in the pool in either a `Ready` or `Allocated` state.

### 7. Deploy a new version of the GameServer on the Fleet

We can also change the configuration of the `GameServer` of the running `Fleet`, and have the changes
roll out, without interrupting the currently `Allocated` `GameServers`.

Let's take this for a spin! Run `kubectl edit fleet simple-udp` and set the `replicas` field to back to `5`.

Let's also allocate ourselves a `GameServer`

```
kubectl create -f https://raw.githubusercontent.com/GoogleCloudPlatform/agones/master/examples/simple-udp/fleetallocation.yaml -o yaml
```

We should now have four `Ready` `GameServers` and one `Allocated`.

We can check this by running `kubectl get gs -o=custom-columns=NAME:.metadata.name,STATUS:.status.state,IP:.status.address,PORT:.status.ports`.

```
NAME                     STATUS      IP               PORT
simple-udp-tfqn7-c9tz7   Ready       192.168.39.150   [map[name:default port:7136]]
simple-udp-tfqn7-g8fhq   Allocated   192.168.39.150   [map[name:default port:7148]]
simple-udp-tfqn7-n0wnl   Ready       192.168.39.150   [map[name:default port:7453]]
simple-udp-tfqn7-hiiwp   Ready       192.168.39.150   [map[name:default port:7228]]
simple-udp-tfqn7-w8z7b   Ready       192.168.39.150   [map[name:default port:7226]]
```

In production, we'd likely be changing a `containers > image` configuration to update our `Fleet`
to run a new game server process, but to make this example simple, change `containerPort` from `7654`
to `6000`. Save the file and exit.

This will start the deployment of a new set of `GameServers` running
with a Container Port of `6000`.

> NOTE: This will make it such that you can no longer connect to the simple-udp game server.  

Run `watch kubectl get gs -o=custom-columns=NAME:.metadata.name,STATUS:.status.state,CONTAINERPORT:.spec.ports[0].containerPort`
until you can see that there are
one of `7654`, which is the `Allocated` `GameServer`, and four instances to `6000` which
is the new configuration.

You have now deployed a new version of your game!

## Next Steps

You can now create a fleet autoscaler to automatically resize your fleet based on the actual usage.
See [Create a Fleet Autoscaler](./create_fleetautoscaler.md).

Or if you want to try to use your own GameServer container make sure you have properly integrated the [Agones SDK](../sdks/).

If you would like to learn how to programmatically allocate a Game Server from the fleet using the Agones API, see the [Allocator Service](./create_allocator_service.md) tutorial.
