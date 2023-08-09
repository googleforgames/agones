---
title: "Quickstart: Create a Game Server Fleet"
linkTitle: "Create a Fleet"
date: 2019-01-02T06:42:20Z
weight: 20
description: >
  This guide covers how you can quickly get started using Agones to create a Fleet of warm GameServers ready for you to allocate out of and play on!
---

## Prerequisites

{{< gs-prerequisites >}}

While not required, you may wish to go through the [Create a Game Server]({{< relref "create-gameserver.md" >}}) quickstart before this one.

## Objectives

- Create a [Fleet](https://agones.dev/site/docs/reference/fleet/) in Kubernetes using an Agones custom resource.
- Scale the Fleet up from its initial configuration.
- Request a GameServer allocation from the Fleet to play on.
- Connect to the allocated GameServer.
- Deploy a new GameServer configuration to the Fleet.

### 1. Create a Fleet

Let's create a Fleet using the following command:

```bash
kubectl apply -f https://raw.githubusercontent.com/googleforgames/agones/{{< release-branch >}}/examples/simple-game-server/fleet.yaml
```

You should see a successful output similar to this :

```
fleet.agones.dev/simple-game-server created
```

This has created a Fleet record inside Kubernetes, which in turn creates two warm [GameServers]({{< ref "/docs/Reference/gameserver.md" >}})
that are available to be allocated for a game session.

```bash
kubectl get fleet
```
It should look something like this:

```
NAME                 SCHEDULING   DESIRED   CURRENT   ALLOCATED   READY     AGE
simple-game-server   Packed       2         3         0           2         9m
```

You can also see the GameServers that have been created by the Fleet by running `kubectl get gameservers`,
the GameServer will be prefixed by `simple-game-server`.

```
NAME                             STATE     ADDRESS            PORT   NODE      AGE
simple-game-server-llg4x-rx6rc   Ready     192.168.122.205    7752   minikube   9m
simple-game-server-llg4x-v6g2r   Ready     192.168.122.205    7623   minikube   9m
```

For the full details of the YAML file head to the [Fleet Specification Guide]({{< ref "/docs/Reference/fleet.md" >}})

{{< alert title="Note" color="info">}} The game servers deployed from a `Fleet` resource will be deployed in the same namespace. The above example omits specifying a namespace, which implies both the `Fleet` and the associated `GameServer` resources will be deployed to the `default` namespace. {{< /alert >}}

### 2. Fetch the Fleet status

Let's wait for the two `GameServers` to become ready.

```bash
watch kubectl describe fleet simple-game-server
```

```
Name:         simple-game-server
Namespace:    default
Labels:       <none>
Annotations:  kubectl.kubernetes.io/last-applied-configuration={"apiVersion":"agones.dev/v1","kind":"Fleet","metadata":{"annotations":{},"name":"simple-game-server","namespace":"default"},"spec":{"replicas":2,...
API Version:  agones.dev/v1
Kind:         Fleet
Metadata:
  Cluster Name:
  Creation Timestamp:  2018-07-01T18:55:35Z
  Generation:          1
  Resource Version:    24685
  Self Link:           /apis/agones.dev/v1/namespaces/default/fleets/simple-game-server
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
        Port Policy:     Dynamic
      Template:
        Metadata:
          Creation Timestamp:  <nil>
        Spec:
          Containers:
            Image:  {{< example-image >}}
            Name:   simple-game-server
            Resources:
Status:
  Allocated Replicas:  0
  Ready Replicas:      2
  Replicas:            2
Events:
  Type    Reason                 Age   From              Message
  ----    ------                 ----  ----              -------
  Normal  CreatingGameServerSet  13s   fleet-controller  Created GameServerSet simple-game-server-wlqnd
```

If you look towards the bottom, you can see there is a section of `Status > Ready Replicas` which will tell you
how many `GameServers` are currently in a Ready state. After a short period, there should be 2 `Ready Replicas`.

### 3. Scale up the Fleet

Let's scale up the `Fleet` from 2 `replicates` to 5.

Run `kubectl scale fleet simple-game-server --replicas=5` to change Replicas count from 2 to 5.

If we now run `kubectl get gameservers` we should see 5 `GameServers` prefixed by `simple-game-server`.

```
NAME                             STATE    ADDRESS           PORT    NODE       AGE
simple-game-server-sdhzn-kcmh6   Ready    192.168.122.205   7191    minikube   52m
simple-game-server-sdhzn-pdpk5   Ready    192.168.122.205   7752    minikube   53m
simple-game-server-sdhzn-r4d6x   Ready    192.168.122.205   7623    minikube   52m
simple-game-server-sdhzn-wng5k   Ready    192.168.122.205   7709    minikube   53m
simple-game-server-sdhzn-wnhsw   Ready    192.168.122.205   7478    minikube   52m
```

### 4. Allocate a Game Server from the Fleet

Since we have a fleet of warm gameservers, we need a way to request one of them for usage, and mark that it has
players accessing it (and therefore, it should not be deleted until they are finished with it).

{{< alert title="Note" color="info">}}
 In production, you would likely do the following through a [Kubernetes API call]({{< ref "/docs/Guides/access-api.md" >}}), but we can also
do this through `kubectl` as well, and ask it to return the response in yaml so that we can see what has happened.
{{< /alert >}}

We can do the allocation of a GameServer for usage through a `GameServerAllocation`, which will both
return to us the details of a `GameServer` (assuming one is available), and also move it to the `Allocated` state,
which demarcates that it has players on it, and should not be removed until `SDK.Shutdown()` is called, or it is manually deleted.

It is worth noting that there is nothing specific that ties a `GameServerAllocation` to a fleet.
A `GameServerAllocation` uses a [label selector](https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/)
to determine what group of `GameServers` it will attempt to allocate out of. That being said, a `Fleet` and `GameServerAllocation`
are often used in conjunction.

{{< ghlink href="/examples/simple-game-server/gameserverallocation.yaml" >}}This example{{< /ghlink >}} uses the label selector to specifically target the `simple-game-server` fleet that we just created.

```bash
kubectl create -f https://raw.githubusercontent.com/googleforgames/agones/{{< release-branch >}}/examples/simple-game-server/gameserverallocation.yaml -o yaml
```

For the full details of the YAML file head to the [GameServerAllocation Specification Guide]({{< ref "/docs/Reference/gameserverallocation.md" >}})

You should get back a response that looks like the following:

```yaml
apiVersion: allocation.agones.dev/v1
kind: GameServerAllocation
metadata:
  creationTimestamp: 2019-02-19T02:13:12Z
  name: simple-game-server-dph9b-hfk24
  namespace: default
spec:
  metadata: {}
  required:
    matchLabels:
      agones.dev/fleet: simple-game-server
  scheduling: Packed
status:
  address: 192.168.122.152
  gameServerName: simple-game-server-dph9b-hfk24
  nodeName: minikube
  ports:
  - name: default
    port: 7714
  state: Allocated
```

If you look at the `status` section, there are several things to take note of. The `state` value will tell if
a `GameServer` was allocated or not. If a `GameServer` could not be found, this will be set to `UnAllocated`.
If there are too many concurrent requests overwhelmed the system, `state` will be set to
`Contention` even though there are available `GameServers`.

However, we see that the `status.state` value was set to `Allocated`.
This means you have been successfully allocated a `GameServer` out of the fleet, and you can now connect your players to it!

You can see various immutable details of the `GameServer` in the status - the `address`, `ports` and the name
of the `GameServer`, in case you want to use it to retrieve more details.

We can also check to see how many `GameServers` you have `Allocated` vs `Ready` with the following command
("gs" is shorthand for "gameserver").

```bash
kubectl get gs
```

This will get you a list of all the current `GameServers` and their `Status.State`.

```
NAME                             STATE       ADDRESS           PORT   NODE      AGE
simple-game-server-sdhzn-kcmh6   Ready       192.168.122.205   7191   minikube  52m
simple-game-server-sdhzn-pdpk5   Ready       192.168.122.205   7752   minikube  53m
simple-game-server-sdhzn-r4d6x   Allocated   192.168.122.205   7623   minikube  52m
simple-game-server-sdhzn-wng5k   Ready       192.168.122.205   7709   minikube  53m
simple-game-server-sdhzn-wnhsw   Ready       192.168.122.205   7478   minikube  52m
```

{{< alert title="Note" color="info">}}
 `GameServerAllocations` are create only and not stored for performance reasons, so you won't be able to list
  them after they have been created - but you can see their effects on `GameServers`
{{< /alert >}}

A handy trick for checking to see how many `GameServers` you have `Allocated` vs `Ready`, run the following:

```bash
kubectl get gs
```

This will get you a list of all the current `GameServers` and their `Status > State`.

```
NAME                             STATE       ADDRESS          PORT   NODE        AGE
simple-game-server-tfqn7-c9tqz   Ready       192.168.39.150   7136   minikube    52m
simple-game-server-tfqn7-g8fhq   Allocated   192.168.39.150   7148   minikube    53m
simple-game-server-tfqn7-p8wnl   Ready       192.168.39.150   7453   minikube    52m
simple-game-server-tfqn7-t6bwp   Ready       192.168.39.150   7228   minikube    53m
simple-game-server-tfqn7-wkb7b   Ready       192.168.39.150   7226   minikube    52m
```

### 5. Scale down the Fleet

Not only can we scale our fleet up, but we can scale it down as well.

The nice thing about Agones is that it is smart enough to know when `GameServers` have been moved to `Allocated`
and will automatically leave them running on scale down -- as we assume that players are playing on this game server,
and we shouldn't disconnect them!

Let's scale down our Fleet to 0 (yep! you can do that!), and watch what happens.

Run `kubectl scale fleet simple-game-server --replicas=0` to change Replicas count from 5 to 0.

It may take a moment for all the `GameServers` to shut down, so let's watch them all and see what happens:
```bash
watch kubectl get gs
```

Eventually, one by one they will be removed from the list, and you should simply see:

```
NAME                             STATUS      ADDRESS          PORT    NODE       AGE
simple-game-server-tfqn7-g8fhq   Allocated   192.168.39.150   7148    minikube   55m
```

That lone `Allocated` `GameServer` is left all alone, but still running!

If you would like, try editing the `Fleet` configuration `replicas` field and watch the list of `GameServers`
grow and shrink.

### 6. Connect to the GameServer

Since we've only got one allocation, we'll just grab the details of the IP and port of the
only allocated `GameServer`:

```bash
kubectl get gameservers | grep Allocated | awk '{print $3":"$4 }'
```

This should output your Game Server IP address and port. (eg `10.130.65.208:7936`)

You can now communicate with the `GameServer`:

```
nc -u {IP} {PORT}
Hello World !
ACK: Hello World !
EXIT
```

You can finally type `EXIT` which tells the SDK to run the [Shutdown command]({{< ref "/docs/Guides/Client SDKs/_index.md#shutdown" >}}), and therefore shuts down the `GameServer`.

If you run `kubectl describe gs | grep State` again - either the GameServer will be replaced with a new, `Ready` `GameServer`
, or it will be in `Shutdown` state, on the way to being deleted.

Since we are running a `Fleet`, Agones will always do it's best to ensure there are always the configured number
of `GameServers` in the pool in either a `Ready` or `Allocated` state.

### 7. Deploy a new version of the GameServer on the Fleet

We can also change the configuration of the `GameServer` of the running `Fleet`, and have the changes
roll out, without interrupting the currently `Allocated` `GameServers`.

Let's take this for a spin! Run `kubectl scale fleet simple-game-server --replicas=5` to return Replicas count back to 5.

Let's also allocate ourselves a `GameServer`:

```bash
kubectl create -f https://raw.githubusercontent.com/googleforgames/agones/{{< release-branch >}}/examples/simple-game-server/gameserverallocation.yaml -o yaml
```

We should now have four `Ready` `GameServers` and one `Allocated`.

We can check this by running `kubectl get gs`.

```
NAME                             STATE       ADDRESS          PORT   NODE       AGE
simple-game-server-tfqn7-c9tz7   Ready       192.168.39.150   7136   minikube   5m
simple-game-server-tfqn7-g8fhq   Allocated   192.168.39.150   7148   minikube   5m
simple-game-server-tfqn7-n0wnl   Ready       192.168.39.150   7453   minikube   5m
simple-game-server-tfqn7-hiiwp   Ready       192.168.39.150   7228   minikube   5m
simple-game-server-tfqn7-w8z7b   Ready       192.168.39.150   7226   minikube   5m
```

In production, we'd likely be changing a `containers > image` configuration to update our `Fleet`
to run a new game server process, but to make this example simple, change `containerPort` from `7654`
to `6000`.

Run `kubectl edit fleet simple-game-server`, and make the necessary changes, and then save and exit your editor.

This will start the deployment of a new set of `GameServers` running
with a Container Port of `6000`.

{{< alert title="Warning" color="warning">}}
This will make it such that you can no longer connect to the simple-game-server game server.
{{< /alert >}}

Run `kubectl describe gs | grep "Container Port"`
until you can see that there is
one with a containerPort of `7654`, which is the `Allocated` `GameServer`, and four instances with a containerPort of `6000` which
is the new configuration. You can also run `kubectl get gs` and look at the **Age** column to see that one `GameServer` is much
older than the other four.

You have now deployed a new version of your game!

## Next Steps

- Have a look at the [GameServerAllocation specification]({{< ref "/docs/Reference/gameserverallocation.md" >}}), and see
    how the extra functionality can enable smoke testing, server information communication, and more.
- You can now create a fleet autoscaler to automatically resize your fleet based on the actual usage.
  See [Create a Fleet Autoscaler]({{< relref "create-fleetautoscaler.md" >}}).
- Have a look at the [GameServer Integration Patterns]({{< ref "/docs/Integration Patterns/_index.md" >}}),
    to give you a set of examples on how all the pieces fit together with your matchmaker and other systems.
- Or if you want to try to use your own GameServer container make sure you have properly integrated the [Agones SDK]({{< ref "/docs/Guides/Client SDKs/_index.md" >}}).
- If you would like to learn how to programmatically allocate a Game Server from the fleet, see how to [Access Agones via the Kubernetes API]({{< relref "../Guides/access-api.md" >}}) or alternatively use the [Allocator Service]({{< relref "../Advanced/allocator-service.md" >}}), depending on your needs.
