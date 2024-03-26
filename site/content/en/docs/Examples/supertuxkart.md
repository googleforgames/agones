---
title: "Supertuxkart Example"
linkTitle: "Supertuxkart"
date:
publishDate:
description: >
  This Supertuxkart example shows how to deploy and run a Supertuxkart server using Agones on a Kubernetes cluster.
  Prior to beginning, ensure the following prerequisites are met:

  - You have a running Kubernetes cluster.

  - Agones is installed on your cluster.
---

## Deploy the Agones Fleet

To apply the fleet configuration to your cluster, use the following command:

```bash
kubectl apply -f fleet.yaml
```

When you run this command, it will:
- Initiate the creation of a Supertuxkart game server fleet as defined in your `fleet.yaml`.
- Set up the fleet with 2 replicas, meaning two instances of the game server will be launched.
- Configure each game server to expose the specified container port (8080).
- Apply health check settings, with an initial delay of 30 seconds and subsequent checks every 60 seconds.
- Ensure the deployment strategy is set to `Recreate`, meaning all existing replicas are removed before new ones are created if the fleet is updated.

## Verify Fleet Creation

After applying the fleet configuration, you can check to ensure that the fleet has been successfully created:

```bash
kubectl get fleets
```

Make sure your Supertuxkart servers are all set by checking that their status shows as `Ready`.
This means they're properly configured. Additionally, you can check the status of individual pods within the fleet:

```bash
kubectl get pods
```

Ensure Supertuxkart pods are active by confirming their statuses are listed as `Running`.

## Allocate a GameServer

Allocate a GameServer from the fleet using:

```bash
kubectl apply -f gameserver.yaml
```

After applying, you can monitor the newly created GameServer resource, its status, and IP:

```bash
kubectl get gameservers
```

## Viewing GameServer Logs

For troubleshooting or to check how your game servers are running, you can look at the logs of a specific pod using:

```bash
kubectl logs -f <supertuxkart-game-server-pod-name>
```
