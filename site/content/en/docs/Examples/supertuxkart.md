---
title: "Deploying and Running SuperTuxKart Server Using Agones"
linkTitle: "Supertuxkart"
date:
publishDate:
description: >
  This Supertuxkart example shows how to set up, deploy, and manage a Supertuxkart game server on a Kubernetes cluster using Agones. It highlights an approach to integrate with existing dedicated game servers.
---

## Prerequisite

 To get started, ensure the following prerequisites are met:

  - You have a running Kubernetes cluster.
  
  - Agones is installed on your cluster. See [Agones guide](https://agones.dev/site/docs/installation/install-agones/).

  - Supertuxkart client is downloaded separately to play. See [SuperTuxKart](https://supertuxkart.net/)
  - Example code for Supertuxkart on Agones is available {{< ghlink href="examples/supertuxkart" >}}here{{< /ghlink >}}

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
kubectl apply -f <your-gameserver.yaml>
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

## Connect to the Game Server

After allocating a GameServer from the fleet and obtaining its status and IP, you're ready to connect and play. Here’s how to use the server IP and port to join the game with the SuperTuxKart client:

**Launch SuperTuxKart**: Open the SuperTuxKart client you downloaded earlier.

**Navigate to Online Play**: From the main menu, select the "Online Play" option.

**Enter Server Details**: Look for an option to "Join" a server by IP or a similar option where you can manually enter server details. Input the IP address and port number you obtained from the `kubectl get gameservers` command.

**Join the Game**: After entering the server details, proceed to join the server. You should now be connected to your Agones-managed SuperTuxKart game server and ready to play.

## Cleaning Up

After playing SuperTuxKart, it's a good practice to clean up the resources to prevent unnecessary resource consumption. Follow these steps to remove them:

### Remove the Fleet

To delete the Agones fleet you deployed, execute the following command. This will remove the fleet along with all the game server instances it manages:

```bash
kubectl delete -f fleet.yaml
```
