# Quickstart Edit a Game Server
The following guide is for developers without Docker or Kubernetes experience, that want to use the simple-udp example as a starting point for a custom game server. This guide addresses Google Kubernetes Engine and Minikube.  We would welcome a Pull Request to expand this to include other platforms as well.

## Prerequisites

1. A [Go](https://golang.org/dl/) environment
2. [Docker](https://www.docker.com/get-started/)
3. Agones installed on GKE or Minikube
4. kubectl properly configured

To install on GKE, follow the install instructions (if you haven't already) at
[Setting up a Google Kubernetes Engine (GKE) cluster](../install/README.md#setting-up-a-google-kubernetes-engine-gke-cluster). Also complete the "Enabling creation of RBAC resources" and "Installing Agones" sets of instructions on the same page.

To install locally on Minikube, read [Setting up a Minikube cluster](../install/README.md#setting-up-a-minikube-cluster).  Also complete the "Enabling creation of RBAC resources" and "Installing Agones" sets of instructions on the same page.

## Modify the code and push another new image

### Modify the simple-udp example source code
Modify the main.go file. For example:

Change main.go line 92:

From:
```go
ack := "ACK: " + txt + "\n"
```

To:
```go
ack := "ACK: Echo says " + txt + "\n"
```

### Build Server
Since Docker image is using Alpine Linux, the "go build" command has to include few more environment variables.

```bash
>> go get agones.dev/agones/pkg/sdk
>> GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o bin/server -a -v main.go
```

## Using Docker File

### Create a new docker image
```bash
>> docker build -t gcr.io/[PROJECT_ID]/agones-udp-server:modified .
```

Note: you can change the image name "agones-udp-server" to something else.

### If using GKE, push the image to GCP Registry
```bash
>> docker push gcr.io/[PROJECT_ID]/agones-udp-server:modified
```

Note: Review [Authentication Methods](https://cloud.google.com/container-registry/docs/advanced-authentication)
for additional information regarding use of gcloud as a Docker credential helper
and advanced authentication methods to the Google Container Registry.

### If using Minikube, load the image into Minikube
```bash
>> docker save gcr.io/[PROJECT_ID]/agones-udp-server:modified | (eval $(minikube docker-env) && docker load)
```

### Modify gameserver.yaml
Modify the following line from gameserver.yaml to use the new configuration.

```
    spec:
      containers:
      - name: agones-simple-udp
        image: gcr.io/[PROJECT_ID]/agones-udp-server:modified
```

### If using GKE, deploy Server to GKE
Apply the latest settings to kubernetes container.

```bash
>> gcloud config set container/cluster [CLUSTER_NAME]
>> gcloud container clusters get-credentials [CLUSTER_NAME]
>> kubectl apply -f gameserver.yaml
```

### If using Minikube, deploy the Server to Minikube
```bash
>> kubectl apply -f gameserver.yaml
```


### Check the GameServer Status
```bash
>> kubectl describe gameserver
```

### Verify
Let's retrieve the IP address and the allocated port of your Game Server:

```
kubectl get gs simple-udp -o jsonpath='{.status.address}:{.status.ports[0].port}'
```

You can now communicate with the Game Server :

> NOTE: if you do not have netcat installed
  (i.e. you get a response of `nc: command not found`),
  you can install netcat by running `sudo apt install netcat`.

```
nc -u {IP} {PORT}
Hello World!
ACK:  Echo says  Hello World!
EXIT
```

You can finally type `EXIT` which tells the SDK to run the [Shutdown command](../sdks#shutdown), and therefore shuts down the `GameServer`.  

If you run `kubectl describe gameserver` again - either the GameServer will be gone completely, or it will be in `Shutdown` state, on the way to being deleted.

## Next Steps

If you want to perform rolling updates of modified game servers, see [Quickstart Create a Game Server Fleet](./create_fleet.md).
