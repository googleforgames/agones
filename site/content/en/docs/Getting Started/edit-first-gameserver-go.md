---
title: "Quickstart: Edit a Game Server"
linkTitle: "Edit Your First Game Server (Go)"
date: 2019-01-02T06:42:56Z
description: >
  The following guide is for developers without Docker or Kubernetes experience, that want to use the simple-game-server example as a starting point for a custom game server. 
---

This guide addresses Google Kubernetes Engine and Minikube.  We would welcome a Pull Request to expand this to include other platforms as well.

## Prerequisites

1. A [Go](https://golang.org/dl/) environment
2. [Docker](https://www.docker.com/get-started/)
3. Agones installed on GKE or Minikube
4. kubectl properly configured

To install on GKE, follow the install instructions (if you haven't already) at
[Setting up a Google Kubernetes Engine (GKE) cluster]({{< ref "/docs/Installation/Creating Cluster/gke.md" >}}).
Also complete the "Enabling creation of RBAC resources" and "Installing Agones" sets of instructions on the same page.

To install locally on Minikube, read [Setting up a Minikube cluster]({{< ref "/docs/Installation/Creating Cluster/minikube.md" >}}).
Also complete the "Enabling creation of RBAC resources" and "Installing Agones" sets of instructions on the same page. 

## Modify the code and push another new image

### Modify the simple-game-server example source code
Modify the {{< ghlink href="examples/simple-game-server/main.go" >}}main.go{{< /ghlink >}} file. For example:

Change the following line in function `udpReadWriteLoop` in file `main.go`:

From:
```go
response = "ACK: " + response + "\n"
```

To:
```go
response = "ACK Echo Says: " + response + "\n"
```

### Build Server
Since Docker image is using Alpine Linux, the "go build" command has to include few more environment variables.

```bash
go get agones.dev/agones/pkg/sdk
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o bin/server -a -v main.go
```

## Using Docker File

### Create a new docker image and push the image to GCP Registry
```bash
make WITH_WINDOWS=0 WITH_ARM64=0 REPOSITORY={$REGISTRY} push
```

Note: Review [Authentication Methods](https://cloud.google.com/container-registry/docs/advanced-authentication)
for additional information regarding use of gcloud as a Docker credential helper
and advanced authentication methods to the Google Container Registry.

### If using Minikube, load the image into Minikube
```bash
minikube cache add gcr.io/[PROJECT_ID]/agones-agones-simple-game-server:modified
```

### Modify gameserver.yaml
Modify the following line from gameserver.yaml to use the new configuration.

```yaml
spec:
  containers:
  - name: simple-game-server
    image: ${REGISTRY}/simple-game-server:${TAG}
```

### If using GKE, deploy Server to GKE
Apply the latest settings to the Kubernetes container.

```bash
gcloud config set container/cluster [CLUSTER_NAME]
gcloud container clusters get-credentials [CLUSTER_NAME]
kubectl create -f gameserver.yaml
```

### If using Minikube, deploy the Server to Minikube
```bash
kubectl apply -f gameserver.yaml
```

{{< alert title="Note" color="info">}}
If you changed `main.go` again and want to apply the changes to the new game servers, then you also need 
to modify the `gamerserver.yaml` file's
[`imagePullPolicy`](https://kubernetes.io/docs/concepts/containers/images/#updating-images) to be `Always`,
or the node may use a cached copy of the image, which doesn't have the new changes.

  ```yaml
  spec:
  containers:
  - name: simple-game-server
    image: ${REGISTRY}/simple-game-server:${TAG}
    imagePullPolicy: Always
  ```

Alternatively, you can also manually increment the `version` field in the `Makefile` file and change the `TAG`
variable accordingly.
{{< /alert >}}

### Check the GameServer Status
```bash
kubectl describe gameserver
```

### Verify
Let's retrieve the IP address and the allocated port of your Game Server:

```bash
kubectl get gs -o jsonpath='{.items[0].status.address}:{.items[0].status.ports[0].port}'
```

You can now communicate with the Game Server :

{{< alert title="Note" color="info">}}
If you do not have netcat installed
  (i.e. you get a response of `nc: command not found`),
  you can install netcat by running `sudo apt install netcat`.
{{< /alert >}}

```
nc -u {IP} {PORT}
Hello World!
ACK Echo Says:  Hello World!
EXIT
```

You can finally type `EXIT` which tells the SDK to run the [Shutdown command]({{< ref "/docs/Guides/Client SDKs/_index.md#shutdown" >}}), and therefore shuts down the `GameServer`.  

If you run `kubectl describe gameserver` again - either the GameServer will be gone completely, or it will be in `Shutdown` state, on the way to being deleted.

## Next Steps

If you want to perform rolling updates of modified game servers, see [Quickstart Create a Game Server Fleet]({{< relref "./create-fleet.md" >}}).
