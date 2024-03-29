---
title: "Custom Controller for Agones Game Servers"
linkTitle: "Custom Controller"
date:
publishDate:
description: >
  This Custom Controller example shows how to create, deploy and run a Custom Kubernetes Controller for Agones that logs changes to GameServers and modifies their labels.
---

## Prerequisite

 To get started, ensure the following prerequisites are met:

  - You have a running Kubernetes cluster.
  
  - Agones is installed on your cluster. See [Agones guide](https://agones.dev/site/docs/installation/install-agones/).

## Deploy the Custom Controller

For a quick deployment of the custom controller on your cluster, execute:

```bash
kubectl apply -f deployment.yaml
```

When you run this command, it quickly sets up your controller by doing four things: 
 - Creating a service account for secure communication with Kubernetes
 - Defining a role with the right permissions to handle game servers and events
 - Linking this role to the account for broad access
 - Launching two controllers for reliability.

## Verify the Controller

To ensure the custom controller is operational, execute the following command. You should see two instances of the controller actively running:

```bash
kubectl get pods -n agones-system
```

## Monitor the log events for the custom controller pod

To monitor the log events of the custom controller pod during the creation, modification, and deletion of game servers, use the following command:

```bash
kubectl logs -f <custom-controller-pod> -n agones-system
```

**Note**: If a custom controller runs into trouble with logging events, the backup controller will automatically assume the leadership role, ensuring uninterrupted logging of event details.

## Deploy the Agones Fleet

To apply the fleet configuration to your cluster, use the following command:

```bash
kubectl apply -f examples/simple-game-server/fleet.yaml
```

When you run this command, it will:
- Specify that there should be 2 replicas of the Fleet.
- Specify that the Pods should have a container port named "default" with a value of 7654.
- Set the resource requests and limits for the container named `simple-game-server`.

## Create a GameServer Instance

Create a gameserver using below command. After this, you'll be able to see logs about the server being created.

```bash
kubectl create -f <your-gameserver.yaml>
```

## Edit the GameServer

To edit settings of your game server, use:

```bash
kubectl edit gameserver <simple-game-server-name> 
```

This will open an editor for you to make changes, and the modification will be reflected in log events.

## Delete the GameServer

To remove your game server and track its deletion in the log events, run the following command:

```bash
kubectl delete gameserver <simple-game-server-name>
```

## Cleaning Up

When you're done with the Agones fleet and the custom controller, it's a good practice to clean up the resources to prevent unnecessary resource consumption. Follow these steps to remove them:

### Remove the Fleet

To delete the Agones fleet you deployed, execute the following command. This will remove the fleet along with all the game server instances it manages:

```bash
kubectl delete -f examples/simple-game-server/fleet.yaml
```

### Remove the Custom Controller

To remove the custom controller from your cluster, execute the following command. This will delete the deployment that you created earlier.

```bash
kubectl delete -f deployment.yaml
```
