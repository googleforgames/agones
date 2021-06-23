---
title: "Confirming Agones Installation"
linkTitle: "Confirm Installation"
weight: 90
description: >
  Verify Agones is installed and has started successfully.
---

To confirm Agones is up and running, run the following command:

```bash
kubectl describe --namespace agones-system pods
```

It should describe six pods created in the `agones-system` namespace, with no error messages or status. All `Conditions` sections should look like this:

```
Conditions:
  Type              Status
  Initialized       True
  Ready             True
  ContainersReady   True
  PodScheduled      True
```

All this pods should be in a `RUNNING` state:


```bash
kubectl get pods --namespace agones-system
```
```
NAME                                 READY   STATUS    RESTARTS   AGE
agones-allocator-5c988b7b8d-cgtbs    1/1     Running   0          8m47s
agones-allocator-5c988b7b8d-hhhr5    1/1     Running   0          8m47s
agones-allocator-5c988b7b8d-pv577    1/1     Running   0          8m47s
agones-controller-7db45966db-56l66   1/1     Running   0          8m44s
agones-ping-84c64f6c9d-bdlzh         1/1     Running   0          8m37s
agones-ping-84c64f6c9d-sjgzz         1/1     Running   0          8m47s
```

That's it! 

Now with Agones installed, you can utilise its [Custom Resource Definitions][crds] to create
 resources of type `GameServer`, `Fleet` and more!

[crds]: https://kubernetes.io/docs/concepts/api-extension/custom-resources/

## What's next

* Go through the [Create a Game Server Quickstart][quickstart]

[quickstart]: {{< ref "/docs/Getting Started/create-gameserver.md" >}}