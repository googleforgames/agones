---
title: "Minikube"
linkTitle: "Minikube"
weight: 100
description: >
  Follow these steps to create a [Minikube](https://github.com/kubernetes/minikube) cluster
  for your Agones install.
---

## Installing Minikube

First, [install Minikube][minikube], which may also require you to install
a virtualisation solution, such as [VirtualBox][vb] as well.

[minikube]: https://minikube.sigs.k8s.io/docs/start/
[vb]: https://www.virtualbox.org

## Creating an `agones` profile

Create a minikube profile for `agones` so you don't overlap any of the minikube clusters you are already running.

```bash
minikube start -p agones
```
Set the minkube profile to `agones`.

```bash
minikube profile agones
```

## Starting Minikube

The following command starts a local minikube cluster via virtualbox - but this can be
replaced by a [vm-driver](https://github.com/kubernetes/minikube#requirements) of your choice.

```bash
minikube start --kubernetes-version v{{% k8s-version %}}.{{% minikube-k8s-minor-version %}} --vm-driver virtualbox
```

## Next Steps

- Continue to [Install Agones]({{< relref "../Install Agones/_index.md" >}}).
