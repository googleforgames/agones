---
title: "Deploying and Running Xonotic Server Using Agones"
linkTitle: "Xonotic"
date:
publishDate:
description: >
  This Xonotic example shows setting up, deploying, and managing a Xonotic game server on a Kubernetes cluster with Agones. It uses a simple Go wrapper to connect existing game servers with Agones, making it straightforward to run games in the cloud.
---

## Prerequisite

 To get started, ensure the following prerequisites are met:

  - You have a running Kubernetes cluster.

  - Agones is installed on your cluster. Refer to the [Agones guide](https://agones.dev/site/docs/installation/install-agones/).

  - The Xonotic client downloaded for gameplay. Download it from [Xonotic](http://www.xonotic.org)

## Deploy the Agones Fleet

Deploy the fleet configuration for your Xonotic servers using the command:

```bash
kubectl apply -f fleet.yaml
```

## Verify Fleet Creation

After applying the fleet configuration, you can check to ensure that the fleet has been successfully created:

```bash
kubectl get fleets
```

Verify that the Xonotic fleet status is `Ready`, confirming that the servers are correctly set up. Further, check the status of individual pods:

```bash
kubectl get pods
```

The Xonotic server pods should be `Running`, indicating they're active and ready for connections.

## Allocate a GameServer

Allocate a GameServer from the fleet using:

```bash
kubectl apply -f <your-gameserver.yaml>
```

After applying, you can monitor the newly created GameServer resource, its status, and IP:

```bash
kubectl get gameservers
```

## Viewing GameServer Logs

For troubleshooting or to check how your game servers are running, you can look at the logs of a specific pod using:

```bash
kubectl logs -f <xonotic-game-server-pod-name>
```

## Connect to the Game Server

After allocating a GameServer from the fleet and obtaining its status and IP, you're ready to connect and play. Here’s how to use the server IP and port to join the game with the Xonotic server:

**Launch Xonotic**: Start the Xonotic client you previously downloaded.

**Multiplayer Mode**: From the main menu, select "Multiplayer".

**Server Connection**: Choose to join a server manually and input the IP and port number you obtained from the `kubectl get gameservers` command.

**Join the Game**: After entering the server details, proceed to join the server. You should now be connected to your Agones-managed Xonotic game server and ready to play.

## Cleaning Up

Post-gameplay, consider cleaning up resources:

### Remove the Fleet

To delete the Agones fleet you deployed, execute the following command. This will remove the fleet along with all the game server instances it manages:

```bash
kubectl delete -f leet.yaml
```

### Remove Allocated GameServers

To delete the allocated game server, execute the following command.

```bash
kubectl delete -f <your-gameserver.yaml>
```
