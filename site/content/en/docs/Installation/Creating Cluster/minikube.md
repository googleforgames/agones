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

Let's use a minikube profile for `agones`, to make sure we don't overlap any
existing Minikube clusters you may be running.

```bash
minikube profile agones
```

## Starting Minikube

The following command starts a local minikube cluster via virtualbox - but this can be
replaced by a [vm-driver](https://github.com/kubernetes/minikube#requirements) of your choice.

{{% feature expiryVersion="1.9.0" %}}
```bash
minikube start --kubernetes-version v1.15.10 --vm-driver virtualbox
```
{{% /feature %}}
{{% feature publishVersion="1.9.0" %}}
```bash
minikube start --kubernetes-version v1.16.13 --vm-driver virtualbox
```
{{% /feature %}}

## Next Steps

- Continue to [Install Agones]({{< relref "../Install Agones/_index.md" >}}).