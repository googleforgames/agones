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
  
  - Agones is installed on your cluster. Refer [Agones guide](https://agones.dev/site/docs/installation/install-agones/).

  - (Optional) Review {{< ghlink href="examples/custom-controller" >}}Custom Controller code{{< /ghlink >}} to see the details of this example.

## Create a Custom Controller

Let's create a custom controller on your cluster using the following command:

```bash
kubectl apply -f https://raw.githubusercontent.com/googleforgames/agones/{{< release-branch >}}/examples/custom-controller/deployment.yaml
```

When you run this command, it quickly sets up your controller by doing four things: 
 - Sets up the appropriate [RBAC](https://kubernetes.io/docs/reference/access-authn-authz/rbac/) permissions for the custom controller
 - Launching two controllers for reliability, with leader election setup between them.

## Verify the Controller

To ensure the custom controller is operational, execute the following command. You should see two instances of the controller actively running with the prefix `custom-controller`:

```bash
kubectl get pods -n agones-system
```

You should see a successful output similar to this:

```
NAME                                 READY   STATUS    RESTARTS   AGE
custom-controller-74c798cfd8-ld6wk   1/1     Running   0          84s
custom-controller-74c798cfd8-whpp2   1/1     Running   0          84s
```

## Create a Fleet

Let's create a Fleet using the following command:

```bash
kubectl apply -f https://raw.githubusercontent.com/googleforgames/agones/{{< release-branch >}}/examples/simple-game-server/fleet.yaml
```

You should see a successful output similar to this :

```
fleet.agones.dev/simple-game-server created
```

This has created a Fleet record inside Kubernetes, which in turn creates two ready [GameServers]({{< ref "/docs/Reference/gameserver.md" >}})
that are available to be allocated for a game session.

```bash
kubectl get fleet
```
It should look something like this:

```
NAME                 SCHEDULING   DESIRED   CURRENT   ALLOCATED   READY     AGE
simple-game-server   Packed       2         3         0           2         9m
```

You can also see the GameServers that have been created by the Fleet by running `kubectl get gameservers`,
the GameServer will be prefixed by `simple-game-server`.

```
NAME                             STATE     ADDRESS            PORT   NODE      AGE
simple-game-server-llg4x-rx6rc   Ready     192.168.122.205    7752   minikube   9m
simple-game-server-llg4x-v6g2r   Ready     192.168.122.205    7623   minikube   9m
```

For the full details of the YAML file head to the [Fleet Specification Guide]({{< ref "/docs/Reference/fleet.md" >}})

{{< alert title="Note" color="info">}} The game servers deployed from a `Fleet` resource will be deployed in the same namespace. The above example omits specifying a namespace, which implies both the `Fleet` and the associated `GameServer` resources will be deployed to the `default` namespace. {{< /alert >}}

## Monitor the log events for the custom controller pod

To monitor the logs of the custom controller during the creation, modification, and deletion of game servers, use the following command:

```bash
kubectl logs -f deployments/custom-controller -n agones-system
```

**Note**: If this controller fails for any reason, we've also implemented leader election such that the backup controller will automatically assume the leadership role, ensuring uninterrupted logging of event details.

## Cleaning Up

When you're done with the Agones fleet and the custom controller, it's a good practice to clean up the resources to prevent unnecessary resource consumption. Follow these steps to remove them:

### Remove the Fleet

To delete the Agones fleet you deployed, execute the following command. This will remove the fleet along with all the game server instances it manages:

```bash
kubectl delete -f https://raw.githubusercontent.com/googleforgames/agones/{{< release-branch >}}/examples/simple-game-server/fleet.yaml
```

### Remove the Custom Controller

To remove the custom controller from your cluster, execute the following command. This will delete the deployment that you created earlier.

```bash
kubectl delete -f https://raw.githubusercontent.com/googleforgames/agones/{{< release-branch >}}/examples/custom-controller/deployment.yaml
```
