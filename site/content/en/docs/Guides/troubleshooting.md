---
title: "Troubleshooting"
date: 2019-01-03T01:20:49Z
weight: 200
description: "Troubleshooting guides and steps."
---

## Something went wrong with my GameServer

If there is something going wrong with your GameServer, there are a few approaches to determining the cause:

Depending on what is happening, you may want to run `kubectl describe <gameserver name>` to view the events
that are associated with that particular `GameServer` resource. This can give you insight into the lifecycle of the
`GameServer` and if anything has gone wrong.

For example, here we can see where the simple-udp example has been moved to the `Unhealthy` state
due to a crash in the backing `GameServer` Pod container's binary.

```
root@6a71afd42291:/go/src/agones.dev/agones# kubectl describe gs simple-udp-zqppv
Name:         simple-udp-zqppv
Namespace:    default
Labels:       <none>
Annotations:  agones.dev/sdk-version: 1.0.0-dce1546
API Version:  agones.dev/v1
Kind:         GameServer
Metadata:
  Creation Timestamp:  2019-08-16T21:25:44Z
  Finalizers:
    agones.dev
  Generate Name:     simple-udp-
  Generation:        1
  Resource Version:  1378575
  Self Link:         /apis/agones.dev/v1/namespaces/default/gameservers/simple-udp-zqppv
  UID:               6818adc7-c06c-11e9-8dbd-42010a8a0109
Spec:
  Container:  simple-udp
  Health:
    Failure Threshold:      3
    Initial Delay Seconds:  5
    Period Seconds:         5
  Ports:
    Container Port:  7654
    Host Port:       7058
    Name:            default
    Port Policy:     Dynamic
    Protocol:        UDP
  Scheduling:        Packed
  Template:
    Metadata:
      Creation Timestamp:  <nil>
    Spec:
      Containers:
        Image:  gcr.io/agones-images/udp-server:0.18
        Name:   simple-udp
        Resources:
          Limits:
            Cpu:     20m
            Memory:  32Mi
          Requests:
            Cpu:     20m
            Memory:  32Mi
Status:
  Address:    35.230.59.117
  Node Name:  gke-test-cluster-default-590db5e4-4s6r
  Ports:
    Name:          default
    Port:          7058
  Reserved Until:  <nil>
  State:           Unhealthy
Events:
  Type     Reason          Age   From                   Message
  ----     ------          ----  ----                   -------
  Normal   PortAllocation  72s   gameserver-controller  Port allocated
  Normal   Creating        72s   gameserver-controller  Pod simple-udp-zqppv created
  Normal   Scheduled       72s   gameserver-controller  Address and port populated
  Normal   RequestReady    67s   gameserver-sidecar     SDK state change
  Normal   Ready           66s   gameserver-controller  SDK.Ready() complete
  Warning  Unhealthy       34s   health-controller      Issue with Gameserver pod
```

The backing Pod has the same name as the `GameServer` - so it's also worth looking at the
details and events for the Pod to see if there are any issues there, such as restarts due to binary crashes etc.

For example, you can see the restart count on the gcr.io/agones-images/udp-server:0.18 container
is set to `1`, due to the game server binary crash

```
root@6a71afd42291:/go/src/agones.dev/agones# kubectl describe pod simple-udp-zqppv
Name:               simple-udp-zqppv
Namespace:          default
Priority:           0
PriorityClassName:  <none>
Node:               gke-test-cluster-default-590db5e4-4s6r/10.138.0.23
Start Time:         Fri, 16 Aug 2019 21:25:44 +0000
Labels:             agones.dev/gameserver=simple-udp-zqppv
                    agones.dev/role=gameserver
Annotations:        agones.dev/container: simple-udp
                    agones.dev/sdk-version: 1.0.0-dce1546
                    cluster-autoscaler.kubernetes.io/safe-to-evict: false
Status:             Running
IP:                 10.48.1.80
Controlled By:      GameServer/simple-udp-zqppv
Containers:
  simple-udp:
    Container ID:   docker://69eacd03cc89b0636b78abe47926b02183ba84d18fa20649ca443f5232511661
    Image:          gcr.io/agones-images/udp-server:0.18
    Image ID:       docker-pullable://gcr.io/agones-images/udp-server@sha256:6a60eff5e68b88b5ce75ae98082d79cff36cda411a090f3495760e5c3b6c3575
    Port:           7654/UDP
    Host Port:      7058/UDP
    State:          Running
      Started:      Fri, 16 Aug 2019 21:26:22 +0000
    Last State:     Terminated
      Reason:       Completed
      Exit Code:    0
      Started:      Fri, 16 Aug 2019 21:25:45 +0000
      Finished:     Fri, 16 Aug 2019 21:26:22 +0000
    Ready:          True
    Restart Count:  1
    Limits:
      cpu:     20m
      memory:  32Mi
    Requests:
      cpu:        20m
      memory:     32Mi
    Liveness:     http-get http://:8080/gshealthz delay=5s timeout=1s period=5s #success=1 #failure=3
    Environment:  <none>
    Mounts:
      /var/run/secrets/kubernetes.io/serviceaccount from empty (ro)
  agones-gameserver-sidecar:
    Container ID:   docker://f3c475c34d26232e19b60be65b03bc6ce41931f4c37e00770d3ab4a36281d31c
    Image:          gcr.io/agones-mark/agones-sdk:1.0.0-dce1546
    Image ID:       docker-pullable://gcr.io/agones-mark/agones-sdk@sha256:4b5693e95ee3023a2b2e2099d102bb6bac58d4ce0ac472e58a09cee6d160cd19
    Port:           <none>
    Host Port:      <none>
    State:          Running
      Started:      Fri, 16 Aug 2019 21:25:48 +0000
    Ready:          True
    Restart Count:  0
    Requests:
      cpu:     30m
    Liveness:  http-get http://:8080/healthz delay=3s timeout=1s period=3s #success=1 #failure=3
    Environment:
      GAMESERVER_NAME:  simple-udp-zqppv
      POD_NAMESPACE:    default (v1:metadata.namespace)
    Mounts:
      /var/run/secrets/kubernetes.io/serviceaccount from agones-sdk-token-vr6qq (ro)
Conditions:
  Type              Status
  Initialized       True
  Ready             True
  ContainersReady   True
  PodScheduled      True
Volumes:
  empty:
    Type:    EmptyDir (a temporary directory that shares a pod's lifetime)
    Medium:
  agones-sdk-token-vr6qq:
    Type:        Secret (a volume populated by a Secret)
    SecretName:  agones-sdk-token-vr6qq
    Optional:    false
QoS Class:       Burstable
Node-Selectors:  <none>
Tolerations:     node.kubernetes.io/not-ready:NoExecute for 300s
                 node.kubernetes.io/unreachable:NoExecute for 300s
Events:
  Type    Reason     Age                   From                                             Message
  ----    ------     ----                  ----                                             -------
  Normal  Scheduled  2m32s                 default-scheduler                                Successfully assigned default/simple-udp-zqppv to gke-test-cluster-default-590db5e4-4s6r
  Normal  Pulling    2m31s                 kubelet, gke-test-cluster-default-590db5e4-4s6r  pulling image "gcr.io/agones-mark/agones-sdk:1.0.0-dce1546"
  Normal  Started    2m28s                 kubelet, gke-test-cluster-default-590db5e4-4s6r  Started container
  Normal  Pulled     2m28s                 kubelet, gke-test-cluster-default-590db5e4-4s6r  Successfully pulled image "gcr.io/agones-mark/agones-sdk:1.0.0-dce1546"
  Normal  Created    2m28s                 kubelet, gke-test-cluster-default-590db5e4-4s6r  Created container
  Normal  Created    114s (x2 over 2m31s)  kubelet, gke-test-cluster-default-590db5e4-4s6r  Created container
  Normal  Started    114s (x2 over 2m31s)  kubelet, gke-test-cluster-default-590db5e4-4s6r  Started container
  Normal  Pulled     114s (x2 over 2m31s)  kubelet, gke-test-cluster-default-590db5e4-4s6r  Container image "gcr.io/agones-images/udp-server:0.18" already present on machine
```

Finally, you can also get the logs of your `GameServer` `Pod` as well via `kubectl logs <pod name> -c <game server container name>`, for example:

```
root@6a71afd42291:/go/src/agones.dev/agones# kubectl logs simple-udp-zqppv -c simple-udp
2019/08/16 21:26:23 Creating SDK instance
2019/08/16 21:26:24 Starting Health Ping
2019/08/16 21:26:24 Starting UDP server, listening on port 7654
2019/08/16 21:26:24 Marking this server as ready
```

The above commands will only give the most recent container's logs (so we won't get the previous crash), but 
you can use `kubectl logs --previous=true simple-udp-zqppv -c simple-udp` to get the previous instance of the containers logs, or 
use your Kubernetes platform of choice's logging aggregation tools to view the crash details.

For more tips and tricks, the [Kubernetes Cheatsheet: Interactive with Pods](https://kubernetes.io/docs/reference/kubectl/cheatsheet/#interacting-with-running-pods)
 also provides more troubleshooting techniques.

## How do I see the logs for Agones?

If something is going wrong, and you want to see the logs for Agones, there are potentially two places you will want to
check:

1. The controller: assuming you installed Agones in the `agones-system` namespace, you will find that there
is a single pod called `agones-controller-<hash>` (where hash is the unique code that Kubernetes generates) 
that exists there, that you can get the logs from. This is the main
controller for Agones, and should be the first place to check when things go wrong.  
   1. To get the logs from this controller run:   
   `kubectl logs --namespace=agones-system agones-controller-<hash>`   
2. The SDK server sidecar: Agones runs a small [gRPC](https://grpc.io/) + http server for the SDK in a container in the
same network namespace as the game server container to connect to via the SDK.  
The logs from this SDK server are also useful for tracking down issues, especially if you are having trouble with a
particular `GameServer`.   
   1. To find the `Pod` for the `GameServer` look for the pod with a name that is prefixed with the name of the 
   owning `GameServer`. For example if you have a `GameServer` named `simple-udp`, it's pod could potentially be named
   `simple-udp-dnbwj`.
   2. To get the logs from that `Pod`, we need to specify that we want the logs from the `agones-gameserver-sidecar`
   container. To do that, run the following:   
   `kubectl logs simple-udp-dnbwj -c agones-gameserver-sidecar`

Agones uses JSON structured logging, therefore errors will be visible through the `"severity":"info"` key and value.       

## I uninstalled Agones before deleted all my `GameServers` and now they won't delete

Agones `GameServers` use [Finalizers](https://kubernetes.io/docs/tasks/access-kubernetes-api/custom-resources/custom-resource-definitions/#finalizers)
to manage garbage collection of the `GameServers`. This means that if the Agones controller 
doesn't remove the finalizer for you (i.e. if it has been uninstalled),  it can be tricky to remove them all.

Thankfully, if we create a patch to remove the finalizers from all GameServers, we can delete them with impunity.

A quick one liner to do this:

`kubectl get gameserver -o name | xargs -n1 -P1 -I{} kubectl patch {} --type=merge -p '{"metadata": {"finalizers": []}}'`

Once this is done, you can `kubectl delete gs --all` and clean everything up (if it's not gone already).


## I'm getting Forbidden errors when trying to install Agones

Ensure that you are running Kubernetes 1.12 or later, which does not require any special
clusterrolebindings to install Agones.

If you want to install Agones on an older version of Kubernetes, you need to create a
clusterrolebinding to add your identity as a cluster admin, e.g.

```bash
# Kubernetes Engine
kubectl create clusterrolebinding cluster-admin-binding \
  --clusterrole cluster-admin --user `gcloud config get-value account`
# Minikube
kubectl create clusterrolebinding cluster-admin-binding \
  --clusterrole=cluster-admin --serviceaccount=kube-system:default
```

On GKE, `gcloud config get-value accounts` will return a lowercase email address, so if
you are using a CamelCase email, you may need to type it in manually.
