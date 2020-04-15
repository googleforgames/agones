---
title: "Quickstart: Create a Game Server"
linkTitle: "Create a Game Server"
date: 2019-01-02T06:35:31Z
weight: 10
description: >
  This guide covers how you can quickly get started using Agones to create GameServers.  
---

## Objectives

- Create a GameServer in Kubernetes using Agones custom resource.
- Get information about the GameServer such as IP address, port and state.
- Connect to the GameServer.

## Prerequisites

The following prerequisites are required to create a GameServer :

1. A Kubernetes cluster with the UDP port range 7000-8000 open on each node.
2. Agones controller installed in the targeted cluster
3. kubectl properly configured
4. Netcat which is already installed on most Linux/macOS distributions, for windows you can use [WSL](https://docs.microsoft.com/en-us/windows/wsl/install-win10).

If you don't have a Kubernetes cluster you can follow [these instructions]({{< ref "/docs/Installation/_index.md" >}}) to create a cluster on Google Kubernetes Engine (GKE), Minikube or Azure Kubernetes Service (AKS), and install Agones.

For the purpose of this guide we're going to use the {{< ghlink href="examples/simple-udp/" >}}simple-udp{{< /ghlink >}} example as the GameServer container. This example is a very simple UDP server written in Go. Don't hesitate to look at the code of this example for more information.

### 1. Create a GameServer

Let's create a GameServer using the following command :

```
kubectl create -f https://raw.githubusercontent.com/googleforgames/agones/{{< release-branch >}}/examples/simple-udp/gameserver.yaml
```

You should see a successful output similar to this :

```
gameserver "simple-udp" created
```

This has created a GameServer record inside Kubernetes, which has also created a backing [Pod](https://kubernetes.io/docs/concepts/workloads/pods/pod/) to run our simple udp game server code in.
If you want to see all your running GameServers you can run:

```
kubectl get gameservers
```
It should look something like this:

```
NAME               STATE     ADDRESS       PORT   NODE     AGE
simple-udp-7pjrq   Ready   35.233.183.43   7190   agones   3m
```

You can also see the [Pod](https://kubernetes.io/docs/concepts/workloads/pods/pod/) that got created by running `kubectl get pods`, the Pod will be prefixed by `simple-udp`.

```
NAME                READY     STATUS    RESTARTS   AGE
simple-udp-7pjrq    2/2       Running   0          5m
```

As you can see above it says `READY: 2/2` this means there are two containers running in this Pod, this is because Agones injected the SDK sidecar for readiness and health checking of your Game Server.


For the full details of the YAML file head to the [GameServer Specification Guide]({{< ref "/docs/Reference/gameserver.md" >}})

### 2. Fetch the GameServer Status

Let's wait for the GameServer state to become `Ready`:

```
watch kubectl describe gameserver
```

```
Name:         simple-udp-7pjrq
Namespace:    default
Labels:       <none>
Annotations:  agones.dev/sdk-version: 0.9.0-764fa53
API Version:  agones.dev/v1
Kind:         GameServer
Metadata:
  Creation Timestamp:  2019-02-27T15:06:20Z
  Finalizers:
    agones.dev
  Generate Name:     simple-udp-
  Generation:        1
  Resource Version:  30377
  Self Link:         /apis/agones.dev/v1/namespaces/default/gameservers/simple-udp-7pjrq
  UID:               3d7ac3e1-3aa1-11e9-a4f5-42010a8a0019
Spec:
  Container:  simple-udp
  Health:
    Failure Threshold:      3
    Initial Delay Seconds:  5
    Period Seconds:         5
  Ports:
    Container Port:  7654
    Host Port:       7190
    Name:            default
    Port Policy:     Dynamic
    Protocol:        UDP
  Scheduling:        Packed
  Template:
    Metadata:
      Creation Timestamp:  <nil>
    Spec:
      Containers:
        Image:  {{< example-image >}}
        Name:   simple-udp
        Resources:
          Limits:
            Cpu:     20m
            Memory:  32Mi
          Requests:
            Cpu:     20m
            Memory:  32Mi
Status:
  Address:    35.233.183.43
  Node Name:  agones
  Ports:
    Name:  default
    Port:  7190
  State:   Ready
Events:
  Type    Reason          Age   From                   Message
  ----    ------          ----  ----                   -------
  Normal  PortAllocation  34s   gameserver-controller  Port allocated
  Normal  Creating        34s   gameserver-controller  Pod simple-udp-7pjrq created
  Normal  Scheduled       34s   gameserver-controller  Address and port populated
  Normal  Ready           27s   gameserver-controller  SDK.Ready() executed
```

If you look towards the bottom, you can see there is a `Status > State` value. We are waiting for it to move to `Ready`, which means that the game server is ready to accept connections.

You might also be interested to see the `Events` section, which outlines when various lifecycle events of the `GameServer` occur. We can also see when the `GameServer` is ready on the event stream as well - at which time the `Status > Address` and `Status > Ports > Port` have also been populated, letting us know what IP and port our client can now connect to!


Let's retrieve the IP address and the allocated port of your Game Server :

```
kubectl get gs
```

This should output your Game Server IP address and ports, eg:

```
NAME               STATE   ADDRESS         PORT   NODE     AGE
simple-udp-7pjrq   Ready   35.233.183.43   7190   agones   4m
```

{{< alert title="Note" color="info">}}
 If you have Agones installed on minikube the address printed will not be
  reachable from the host machine. Instead, use the output of `minikube ip` for
  the following section.
{{< /alert >}}

### 3. Connect to the GameServer

{{< alert title="Note" color="info">}}
If you have Agones installed on Google Kubernetes Engine, and are using
  Cloud Shell for your terminal, UDP is blocked. For this step, we recommend
  SSH'ing into a running VM in your project, such as a Kubernetes node.
  You can click the 'SSH' button on the [Google Compute Engine Instances](https://console.cloud.google.com/compute/instances)
  page to do this.
  Run `toolbox` on GKE Node to run docker container with tools and then `nc` command would be available.
{{< /alert >}}

You can now communicate with the Game Server :

{{< alert title="Note" color="info">}}
If you do not have netcat installed
  (i.e. you get a response of `nc: command not found`),
  you can install netcat by running `sudo apt install netcat`.
{{< /alert >}}

```
nc -u {IP} {PORT}
Hello World !
ACK: Hello World !
EXIT
```

You can finally type `EXIT` which tells the SDK to run the [Shutdown command]({{< ref "/docs/Guides/Client SDKs/_index.md#shutdown" >}}), and therefore shuts down the `GameServer`.

If you run `kubectl describe gameserver` again - either the GameServer will be gone completely, or it will be in `Shutdown` state, on the way to being deleted.


## Next Step

If you want to use your own GameServer container make sure you have properly integrated the [Agones SDK]({{< ref "/docs/Guides/Client SDKs/_index.md" >}}).


