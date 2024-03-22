---
title: "Custom Controller for Agones Game Servers"
linkTitle: "Custom Controller"
date:
publishDate:
description: >
  This example shows how to deploy and run the Custom Controller example on Agones, a tool to monitor the running dedicated game servers on Kubernetes.
---

## Deploy the Custom Controller

To get started, update the [deployment.yaml](https://github.com/googleforgames/agones/blob/main/examples/custom-controller/deployment.yaml) file to use your Docker image, then apply it to your cluster:

```bash
kubectl apply -f deployment.yaml
```

When you run the command, it quickly sets up your controller by doing four things: creating a service account for secure communication with Kubernetes, defining a role with the right permissions to handle game servers and events, linking this role to the account for broad access, and launching two controllers for reliability,

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






