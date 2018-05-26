# Quickstart Create a Game Server

This guide covers how you can quickly get started using Agones to create GameServers.

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

>NOTE: Agones required Kubernetes versions 1.9 with role-based access controls (RBAC) and MutatingAdmissionWebhook features activated. To check your version, enter `kubectl version`.

If you don't have a Kubernetes cluster you can follow [these instructions](../install/README.md) to create a cluster on Google Kubernetes Engine (GKE) or Minikube, and install Agones.

For the purpose of this guide we're going to use the [simple-udp](../examples/simple-udp/) example as the GameServer container. This example is very simple UDP server written in Go. Don't hesitate to look at the code of this example for more information.

### 1. Create a GameServer

Let's create a GameServer using the following command :

```
kubectl apply -f https://raw.githubusercontent.com/GoogleCloudPlatform/agones/master/examples/simple-udp/server/gameserver.yaml
```

You should see a successful ouput similar to this :

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
NAME         AGE
simple-udp   5m
```

You can also see the [Pod](https://kubernetes.io/docs/concepts/workloads/pods/pod/) that got created by running `kubectl get pods`, the Pod will be prefixed by `simple-udp`.

```
NAME                                     READY     STATUS    RESTARTS   AGE
simple-udp-vwxpt                         2/2       Running   0          5m
```

As you can see above it says `READY: 2/2` this means there are two containers running in this Pod, this is because Agones injected the SDK sidecar for readiness and health checking of your Game Server.


For the full details of the YAML file head to the [GameServer Specification Guide](./gameserver_spec.md)

### 2. Fetch the GameServer Status

Let's wait for the GameServer state to become `Ready`:

```
watch kubectl describe gameserver
```

```
Name:         simple-udp
Namespace:    default
Labels:       <none>
Annotations:  kubectl.kubernetes.io/last-applied-configuration={"apiVersion":"stable.agones.dev/v1alpha1","kind":"GameServer","metadata":{"annotations":{},"name":"simple-udp","namespace":"default"},"spec":{"contain...
API Version:  stable.agones.dev/v1alpha1
Kind:         GameServer
Metadata:
  Cluster Name:
  Creation Timestamp:  2018-03-01T20:43:36Z
  Finalizers:
    stable.agones.dev
  Generation:        0
  Resource Version:  6238
  Self Link:         /apis/stable.agones.dev/v1alpha1/namespaces/default/gameservers/simple-udp
  UID:               372468b6-1d91-11e8-926e-fa163e0980b0
Spec:
  Port Policy:     dynamic
  Container:       simple-udp
  Container Port:  7654
  Health:
    Failure Threshold:      3
    Initial Delay Seconds:  5
    Period Seconds:         5
  Host Port:                7211
  Protocol:                 UDP
  Template:
    Metadata:
      Creation Timestamp:  <nil>
    Spec:
      Containers:
        Image:  gcr.io/agones-images/udp-server:0.1
        Name:   simple-udp
        Resources:
Status:
  Address:    10.130.65.212
  Node Name:  dev-worker-03
  Port:       7211
  State:      Ready
Events:
  Type    Reason          Age   From                   Message
  ----    ------          ----  ----                   -------
  Normal  PortAllocation  11s   gameserver-controller  Port allocated
  Normal  Creating        11s   gameserver-controller  Pod simple-udp-vwxpt created
  Normal  Starting        11s   gameserver-controller  Synced
  Normal  Ready           4s    gameserver-controller  Address and Port populated
```

If you look towards the bottom, you can see there is a `Status > State` value. We are waiting for it to move to `Ready`, which means that the game server is ready to accept connections.

You might also be interested to see the `Events` section, which outlines when various lifecycle events of the `GameSever` occur. We can also see when the `GameServer` is ready on the event stream as well - at which time the `Status > Address` and `Status > Port` have also been populated, letting us know what IP and port our client can now connect to!


Let's retrieve the IP address and the allocated port of your Game Server :

```
kubectl get gs simple-udp -o jsonpath='{.status.address}:{.status.port}'
```

This should ouput your Game Server IP address and port. (eg `10.130.65.208:7936`)

### 3. Connect to the GameServer

> NOTE: if you have Agones installed on Google Kubernetes Engine, and are using
  Cloud Shell for your terminal, UDP is blocked. For this step, we recommend
  SSH'ing into a running VM in your project, such as a Kubernetes node.
  You can click the 'SSH' button on the [Google Compute Engine Instances](https://console.cloud.google.com/compute/instances)
  page to do this. 

You can now communicate with the Game Server :

> NOTE: if you do not have netcat installed 
  (i.e. you get a response of `nc: command not found`), 
  you can install netcat by running `sudo apt install netcat`.

```
nc -u {IP} {PORT}
Hello World !
ACK: Hello World !
EXIT
```

You can finally type `EXIT` which tells the SDK to run the [Shutdown command](../sdks#shutdown), and therefore shuts down the `GameServer`.  

If you run `kubectl describe gameserver` again - either the GameServer will be gone completely, or it will be in `Shutdown` state, on the way to being deleted.


## Next Step

If you want to use your own GameServer container make sure you have properly integrated the [Agones SDK](../sdks/).
