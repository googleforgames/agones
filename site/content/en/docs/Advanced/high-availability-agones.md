---
title: "High Availability Agones"
date: 2023-02-10
weight: 20
description: >
  Learn how to configure your Agones services for high availability and resiliancy to disruptions.
publishDate: 2023-02-28
---

{{% feature expiryVersion="1.37.0" %}}
{{< beta title="split controller and extensions" gate="SplitControllerAndExtensions" >}}
{{% /feature %}}

## High Availability for Agones Controller

{{% feature expiryVersion="1.37.0" %}}
When `SplitControllerAndExtensions` is enabled, the `agones-controller` responsibility is split up into `agones-controller`, which enacts the Agones control loop, and `agones-extensions`, which acts as a service endpoint for webhooks and the allocation extension API. Splitting these responsibilities allows the `agones-extensions` pod to be **horizontally scaled**, making the Agones control plane **highly available** and more **resiliant to disruption**.

`SplitControllerAndExtensions` enables multiple `agones-controller` pods, with a primary controller selected via leader election. Having multiple `agones-controller` minimizes downtime of the service from pod disruptions such as deployment updates, autoscaler evictions, and crashes.

`SplitControllerAndExtensions` must be enabled for GKE Autopilot.
{{% /feature %}}

{{% feature publishVersion="1.37.0" %}}
The `agones-controller` responsibility is split up into `agones-controller`, which enacts the Agones control loop, and `agones-extensions`, which acts as a service endpoint for webhooks and the allocation extension API. Splitting these responsibilities allows the `agones-extensions` pod to be **horizontally scaled**, making the Agones control plane **highly available** and more **resiliant to disruption**.

Multiple `agones-controller` pods enabled, with a primary controller selected via leader election. Having multiple `agones-controller` minimizes downtime of the service from pod disruptions such as deployment updates, autoscaler evictions, and crashes.
{{% /feature %}}

## Extension Pod Configrations 

The `agones-extensions` binary has a similar `helm` configuration to `agones-controller`, see [here]({{< relref "/docs/Installation/Install Agones/helm.md" >}}). If you previously overrode `agones.controller.*` settings, you may need to override the same `agones.extensions.*` setting.

To change `controller.numWorkers` to 200 from 100 values and through the use of `helm --set`, add the follow to the `helm` command:

{{< alert color="warning" >}} Important: This will not have any effect on any `extensions` values! {{< /alert >}}
```
 ...
 --set agones.controller.numWorkers=200
 ...
```

An important configuration to note is the PodDisruptionBudget fields, `agones.extensions.pdb.minAvailable` and `agones.extensions.pdb.maxUnavailable`. Currently, the `agones.extensions.pdb.minAvailable` field is set to 1. 

## Deployment Considerations

{{% feature expiryVersion="1.37.0" %}}
Leader election will automatically be enabled when `SplitControllerAndExtensions` is enabled and `agones.controller.replicas` is > 1. [`agones.controller.replicas`]({{< relref "/docs/Installation/Install Agones/helm.md#configuration" >}}) defaults to 2.

When `SplitControllerAndExtensions` is enabled, what was previously a single `agones-controller` pod is deployed by default as 2 `agones-controller` and 2 `agones-extensions` pods. For example:
{{% /feature %}}

{{% feature publishVersion="1.37.0" %}}
Leader election will automatically be enabled and `agones.controller.replicas` is > 1. [`agones.controller.replicas`]({{< relref "/docs/Installation/Install Agones/helm.md#configuration" >}}) defaults to 2.

The default configuration now deploys 2 `agones-controller` pods and 2 `agones-extensions` pods, replacing the previous single `agones-controller` pod setup. For example:
{{% /feature %}}

```
NAME                                 READY   STATUS    RESTARTS   AGE
agones-allocator-78c6b8c79-h9nqc     1/1     Running   0          23h
agones-allocator-78c6b8c79-l2bzp     1/1     Running   0          23h
agones-allocator-78c6b8c79-rw75j     1/1     Running   0          23h
agones-controller-fbf944f4-vs9xx     1/1     Running   0          23h
agones-controller-fbf944f4-sjk3t     1/1     Running   0          23h
agones-extensions-5648fc7dcf-hm6lk   1/1     Running   0          23h
agones-extensions-5648fc7dcf-qbc6h   1/1     Running   0          23h
agones-ping-5b9647874-2rrl6          1/1     Running   0          27h
agones-ping-5b9647874-rksgg          1/1     Running   0          27h
```

The number of replicas for `agones-extensions` can be set using helm variable [`agones.extensions.replicas`]({{< relref "/docs/Installation/Install Agones/helm.md#configuration" >}}), but the default is `2`. 

We expect the aggregate memory consumption of the pods will be slightly higher than the previous singleton pod, but as the responsibilities are now split across the pods, the aggregate CPU consumption should also be similar.

## Feature Design

Please see [HA Agones](https://github.com/googleforgames/agones/issues/2797).