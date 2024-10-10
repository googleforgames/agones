---
title: "Quickstart: Create a Game Server"
linkTitle: "Create a Game Server"
date: 2019-01-02T06:35:31Z
weight: 10
description: >
  This guide covers how you can quickly get started using Agones to create GameServers.
---

## Prerequisites

{{< gs-prerequisites >}}

## Objectives

- Create a GameServer in Kubernetes using Agones custom resource.
- Get information about the GameServer such as IP address, port and state.
- Connect to the GameServer.


### 1. Create a GameServer

Let's create a GameServer using the following command :

```bash
kubectl create -f https://raw.githubusercontent.com/googleforgames/agones/{{< release-branch >}}/examples/simple-game-server/gameserver.yaml
```

You should see a successful output similar to this :

```
gameserver.agones.dev/simple-game-server-4ss4j created
```

This has created a GameServer record inside Kubernetes, which has also created a backing [Pod](https://kubernetes.io/docs/concepts/workloads/pods/pod/) to run our simple udp game server code in.
If you want to see all your running GameServers you can run:

```bash
kubectl get gameservers
```
It should look something like this:

```
NAME                       STATE     ADDRESS       PORT   NODE     AGE
simple-game-server-7pjrq   Ready   35.233.183.43   7190   agones   3m
```

You can also see the [Pod](https://kubernetes.io/docs/concepts/workloads/pods/pod/) that got created by running `kubectl get pods`, the Pod will be prefixed by `simple-game-server`.

```
NAME                        READY     STATUS    RESTARTS   AGE
simple-game-server-7pjrq    2/2       Running   0          5m
```

As you can see above it says `READY: 2/2` this means there are two containers running in this Pod, this is because Agones injected the [SDK sidecar](https://agones.dev/site/docs/guides/troubleshooting/#how-do-i-see-the-logs-for-agones) for readiness
and health checking of your Game Server.


For the full details of the YAML file head to the [GameServer Specification Guide]({{< ref "/docs/Reference/gameserver.md" >}})

### 2. Fetch the GameServer Status

Let's wait for the GameServer state to become `Ready`. You can use the `watch`
tool to see the state change. If your operating system does not have `watch`,
manually run `kubectl describe gameserver` until the state changes.

```bash
watch kubectl describe gameserver
```

```
Name:         simple-game-server-7pjrq
Namespace:    default
Labels:       <none>
Annotations:  agones.dev/sdk-version: 0.9.0-764fa53
API Version:  agones.dev/v1
Kind:         GameServer
Metadata:
  Creation Timestamp:  2019-02-27T15:06:20Z
  Finalizers:
    agones.dev/controller
  Generate Name:     simple-game-server-
  Generation:        1
  Resource Version:  30377
  Self Link:         /apis/agones.dev/v1/namespaces/default/gameservers/simple-game-server-7pjrq
  UID:               3d7ac3e1-3aa1-11e9-a4f5-42010a8a0019
Spec:
  Container:  simple-game-server
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
        Name:   simple-game-server
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
  Normal  Creating        34s   gameserver-controller  Pod simple-game-server-7pjrq created
  Normal  Scheduled       34s   gameserver-controller  Address and port populated
  Normal  Ready           27s   gameserver-controller  SDK.Ready() executed
```

If you look towards the bottom, you can see there is a `Status > State` value. We are waiting for it to move to `Ready`, which means that the game server is ready to accept connections.

You might also be interested to see the `Events` section, which outlines when various lifecycle events of the `GameServer` occur. We can also see when the `GameServer` is ready on the event stream as well - at which time the `Status > Address` and `Status > Ports > Port` have also been populated, letting us know what IP and port our client can now connect to!


Let's retrieve the IP address and the allocated port of your Game Server :

```bash
kubectl get gs
```

This should output your Game Server IP address and ports, eg:

```
NAME                       STATE   ADDRESS         PORT   NODE     AGE
simple-game-server-7pjrq   Ready   35.233.183.43   7190   agones   4m
```

{{< alert title="Note" color="info">}}
If you have Agones installed on minikube, or other local Kubernetes tooling, and you are having issues connecting
to the `GameServer`, please check the
[Minikube local connection workarounds]({{% ref "/docs/Installation/Creating Cluster/minikube.md#local-connection-workarounds" %}}).
{{< /alert >}}

### 3. Connect to the GameServer

You can now communicate with the Game Server, by running:
```shell
nc -u {IP} {PORT}
```

Now write any text you would like, and hit `<Enter>`. You should see your text echoed back, like so: 

```shell
nc -u 35.233.183.43 7190
Hello World !
ACK: Hello World !
```

You can finally type `EXIT` and hit `<Enter>`, which tells the SDK to run the 
[Shutdown command]({{< ref "/docs/Guides/Client SDKs/_index.md#shutdown" >}}), and therefore shuts down the `GameServer`.

To exit `nc` you can press `<ctrl>+c`.

If you run `kubectl describe gameserver` again - either the GameServer will be gone completely, or it will be in `Shutdown` state, on the way to being deleted.

{{< alert title="Note" color="info">}}
If you do not have netcat installed
(i.e. you get a response of `nc: command not found`),
you can install netcat by running `sudo apt install netcat`.

If you are on Windows, you can alternatively install netcat on
[WSL](https://docs.microsoft.com/en-us/windows/wsl/install-win10),
or download a version of netcat for Windows from [nmap.org](https://nmap.org/ncat/).
{{< /alert >}}

## Next Step

If you want to use your own GameServer container make sure you have properly integrated the [Agones SDK]({{< ref "/docs/Guides/Client SDKs/_index.md" >}}).


