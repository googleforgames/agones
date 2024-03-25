---
title: "Supertuxkart Example"
linkTitle: "Supertuxkart"
date:
publishDate:
description: >
  This example shows deploying a SuperTuxKart server on Kubernetes with Agones, highlighting a scalable, 
  efficient multiplayer gaming platform.
---

## Build and Push the Supertuxkart
To get started, replace the `agones-images` projectID with your GCP projectID in the `Makefile` and all `.yaml` files where it appears. For example, change `us-docker.pkg.dev/agones-images/...` to `us-docker.pkg.dev/your-project-id/...`.

To build and push the Supertuxkart image to your artifact registry, run:

```bash
# For an automated build and push
make cloud-build
```
For manual build and push:
`make build`
`make push`


## Deploy the Agones Fleet

To apply the fleet configuration to your cluster, use the following command:

```bash
kubectl apply -f fleet.yaml
```

## Monitor the fleet status

Monitor the Fleet's status until the GameServers are Ready:

```bash
kubectl get fleets
```

## Allocate a GameServer

Allocate a GameServer from the Fleet:

```bash
kubectl apply -f gameserverallocation.yaml
```

Then, check the allocated GameServer's status and IP

```bash
kubectl get gameservers
```

## Manage Servers

Agones Fleet will automatically manage the lifecycle of your game servers, including auto-scaling instances
based on demand and player load.
