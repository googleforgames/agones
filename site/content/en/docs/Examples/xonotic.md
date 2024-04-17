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

  - Agones is installed on your cluster. Refer to the [Agones guide](https://agones.dev/site/docs/installation/install-agones/) for the instructions.

  - The Xonotic client downloaded for gameplay. Download it from [Xonotic](http://www.xonotic.org)

  - (Optional) Review {{< ghlink href="examples/xonotic" >}}Xonotic code{{< /ghlink >}} to see the details of this example.


## Create a Fleet

Let's create a Fleet using the following command:

```bash
kubectl apply -f https://raw.githubusercontent.com/googleforgames/agones/{{< release-branch >}}/examples/xonotic/fleet.yaml
```

You should see a successful output similar to this :

```
fleet.agones.dev/xonotic created
```

This has created a Fleet record inside Kubernetes, which in turn creates two ready [GameServers]({{< ref "/docs/Reference/gameserver.md" >}})
that are available to be allocated for a game session.

```bash
kubectl get fleet
```
It should look something like this:

```
NAME           SCHEDULING   DESIRED   CURRENT   ALLOCATED   READY   AGE
xonotic        Packed       2         2         0           2       55s
```

You can also see the GameServers that have been created by the Fleet by running `kubectl get gameservers`,
the GameServer will be prefixed by `xonotic`.

```
NAME                       STATE   ADDRESS          PORT   NODE                                        AGE
xonotic-7lk8x-hgfrg        Ready   34.71.168.92     7206   gk3-genai-quickstart-pool-3-ba2a705f-wpmc   103s
xonotic-7lk8x-rwhst        Ready   34.71.168.92     7330   gk3-genai-quickstart-pool-3-ba2a705f-wpmc   103s
```

For the full details of the YAML file head to the [Fleet Specification Guide]({{< ref "/docs/Reference/fleet.md" >}})


## Connect to the Game Server

After allocating a GameServer from the fleet and obtaining its status and IP, you're ready to connect and play. Here’s how to use the server IP and port to join the game with the Xonotic server:

**Launch Xonotic**: Start the Xonotic client you previously downloaded by running the executable for your operating system ([documentation](https://xonotic.org/faq/#install)).

**Multiplayer Mode**: From the main menu, select "Multiplayer".

**Server Connection**: Choose to join a server manually and input the IP and port number you obtained from the `kubectl get gameservers` command.

![Enter IP and Port](../../../images/xonotic-ip-port.png)


**Join the Game**: After entering the server details, proceed to join the server. You should now be connected to your Agones-managed Xonotic game server and ready to play.

![Join the Game](../../../images/xonotic-join-game.png)


## Cleaning Up

After you're done playing, it's a good idea to clean up. To remove the Agones fleet you deployed, execute the following command. This will remove the fleet along with all the game server instances it manages:

```bash
kubectl delete -f https://raw.githubusercontent.com/googleforgames/agones/{{< release-branch >}}/examples/xonotic/fleet.yaml
```
