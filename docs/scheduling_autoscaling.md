# Scheduling and Autoscaling

⚠️⚠️⚠️ **This is currently a development feature and has not been released** ⚠️⚠️⚠️

> Autoscaling is currently ongoing work within Agones. The work you see here is just the beginning.


Table of Contents
=================

* [Fleet Autoscaling](#fleet-autoscaling)
* [Autoscalng Concepts](#autoscalng-concepts)
   * [Allocation Scheduling](#allocation-scheduling)
* [Fleet Scheduling](#fleet-scheduling)
   * [Packed](#packed)
      * [Allocation Scheduling Strategy](#allocation-scheduling-strategy)
   * [Distributed](#distributed)
      * [Allocation Scheduling Stategy](#allocation-scheduling-stategy)

Scheduling and autoscaling go hand in hand, as where in the cluster `GameServers` are provisioned
impacts how to autoscale fleets up and down (or if you would even want to)

## Fleet Autoscaling

Fleet autoscaling is currently the only type of autoscaling that exists in Agones. It is also only available as a simple
buffer autoscaling strategy. Have a look at the [Create a Fleet Autoscaler](create_fleetautoscaler.md) quickstart,
and the [Fleet Autoscaler Specification](fleetautoscaler_spec.md) for details.

Node scaling, and more sophisticated fleet autoscaling will be coming in future releases ([design](https://github.com/GoogleCloudPlatform/agones/issues/368))

## Autoscaling Concepts

To facilitate autoscaling, we need to combine several piece of concepts and functionality, described below.

### Allocation Scheduling

Allocation scheduling refers to the order in which `GameServers`, and specifically their backing `Pods` are chosen
from across the Kubernetes cluster within a given `Fleet` when [allocation](./create_fleet.md#4-allocate-a-game-server-from-the-fleet) occurs.

## Fleet Scheduling

There are two scheduling strategies for Fleets - each designed for different types of Kubernetes Environments.

### Packed

```yaml
apiVersion: "stable.agones.dev/v1alpha1"
kind: Fleet
metadata:
  name: simple-udp
spec:
  replicas: 100
  scheduling: Packed
  template:
    spec:
      ports:
      - containerPort: 7654
      template:
        spec:
          containers:
          - name: simple-udp
            image: gcr.io/agones-images/udp-server:0.4
```

This is the *default* Fleet scheduling strategy. It is designed for dynamic Kubernetes environments, wherein you wish 
to scale up and down as load increases or decreases, such as in a Cloud environment where you are paying
for the infrastructure you use.

It attempts to _pack_ as much as possible into the smallest set of nodes, to make
scaling infrastructure down as easy as possible.

Currently, Allocation scheduling is the only aspect this strategy affects, but in future releases it will
also affect `GameServer` `Pod` scheduling, and `Fleet` scale down scheduling as well.

#### Allocation Scheduling Strategy

Under the "Packed" strategy, allocation will prioritise allocating `GameServers` to nodes that are running on 
Nodes that already have allocated `GameServers` running on them.

### Distributed

```yaml
apiVersion: "stable.agones.dev/v1alpha1"
kind: Fleet
metadata:
  name: simple-udp
spec:
  replicas: 100
  scheduling: Distributed
  template:
    spec:
      ports:
      - containerPort: 7654
      template:
        spec:
          containers:
          - name: simple-udp
            image: gcr.io/agones-images/udp-server:0.4
```

This Fleet scheduling strategy is designed for static Kubernetes environments, such as when you are running Kubernetes
on bare metal, and the cluster size rarely changes, if at all.

This attempts to distribute the load across the entire cluster as much as possible, to take advantage of the static
size of the cluster.

Currently, the only thing the scheduling strategy affects is Allocation scheduling, but in future releases it will
also affect `GameServer` `Pod` scheduling, and `Fleet` scaledown scheduling as well.

#### Allocation Scheduling Strategy

Under the "Distributed" strategy, allocation will prioritise allocating `GameSerers` to nodes that have the least
number of allocated `GameServers` on them.
