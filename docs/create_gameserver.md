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

If you don't have a Kubernetes cluster you can follow [these instructions](../install/README.md) to create a cluster on Google Kubernetes Engine (GKE), Minikube or Azure Kubernetes Service (AKS), and install Agones.

For the purpose of this guide we're going to use the [simple-udp](../examples/simple-udp/) example as the GameServer container. This example is very simple UDP server written in Go. Don't hesitate to look at the code of this example for more information.

### 1. Create a GameServer

Let's create a GameServer using the following command :

```
kubectl apply -f https://raw.githubusercontent.com/GoogleCloudPlatform/agones/master/examples/simple-udp/gameserver.yaml
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
Name:         simple-udp-jq8kd-q8dzg
Namespace:    default
Labels:       stable.agones.dev/gameserverset=simple-udp-jq8kd
Annotations:  <none>
API Version:  stable.agones.dev/v1alpha1
Kind:         GameServer
Metadata:
  Cluster Name:
  Creation Timestamp:  2018-06-30T14:15:43Z
  Finalizers:
    stable.agones.dev
  Generate Name:  simple-udp-jq8kd-
  Generation:     1
  Resource Version:        11978
  Self Link:               /apis/stable.agones.dev/v1alpha1/namespaces/default/gameservers/simple-udp-jq8kd-q8dzg
  UID:                     132bb210-7c70-11e8-b9be-08002703ef08
Spec:
  Container:  simple-udp
  Health:
    Failure Threshold:      3
    Initial Delay Seconds:  5
    Period Seconds:         5
  Ports:
    Container Port:  7654
    Host Port:       7614
    Name:            default
    Port Policy:     dynamic
    Protocol:        UDP
  Template:
    Metadata:
      Creation Timestamp:  <nil>
    Spec:
      Containers:
        Image:  gcr.io/agones-images/udp-server:0.4
        Name:   simple-udp
        Resources:
Status:
  Address:    192.168.99.100
  Node Name:  agones
  Ports:
    Name:  default
    Port:  7614
  State:   Ready
Events:
  Type    Reason          Age   From                   Message
  ----    ------          ----  ----                   -------
  Normal  PortAllocation  23s   gameserver-controller  Port allocated
  Normal  Creating        23s   gameserver-controller  Pod simple-udp-jq8kd-q8dzg-9kww8 created
  Normal  Starting        23s   gameserver-controller  Synced
  Normal  Ready           20s   gameserver-controller  Address and Port populated
```

If you look towards the bottom, you can see there is a `Status > State` value. We are waiting for it to move to `Ready`, which means that the game server is ready to accept connections.

You might also be interested to see the `Events` section, which outlines when various lifecycle events of the `GameSever` occur. We can also see when the `GameServer` is ready on the event stream as well - at which time the `Status > Address` and `Status > Port` have also been populated, letting us know what IP and port our client can now connect to!


Let's retrieve the IP address and the allocated port of your Game Server :

```
kubectl get gs -o=custom-columns=NAME:.metadata.name,STATUS:.status.state,IP:.status.address,PORT:.status.ports
```

This should ouput your Game Server IP address and ports, eg:

```
NAME         STATUS    IP               PORT
simple-udp   Ready     192.168.99.100   [map[name:default port:7614]]
```

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
