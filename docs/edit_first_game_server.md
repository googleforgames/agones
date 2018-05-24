# Getting Started
The following guide is for developers without Docker or Kubernetes experience, that want to use the simple-udp example as a starting point for a custom game server. Since this guide is for Google Kubernetes Engine, we welcome a Pull Request to expand this to include Minikube as well.

## Prerequisites

1. Downland and install Golang from https://golang.org/dl/.
2. Install Docker from https://www.docker.com/get-docker.
3. Follow the install instructions (if you haven't already) at [Install and configure Agones on Kubernetes](../install/README.md). At least, you should have "Setting up a Google Kubernetes Engine (GKE) cluster", "Enabling creation of RBAC resources" and "Installing Agones" done.

## Modify the code and push another new image

### Modify the source code
Modify the main.go file. For example:

Change main.go line 92:

From:
```go
ack := "ACK: " + txt + "\n"
```

To:
```go
ack := "ACK: Hi," + txt + "\n"
```


### Build Server
Since Docker image is using Alpine Linux, the "go build" command has to include few more environment variables.

```bash
go get agones.dev/agones/pkg/sdk
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o bin/server -a -v main.go
```

## Using Docker File without using Minikube

### Create a new docker image
```bash
docker build -t gcr.io/[PROJECT_ID]/agones-udp-server:0.2 .
```
Note that: you can change the image name "agones-udp-server" to something else.

### Push the image to GCP Registry
```bash
docker push gcr.io/[PROJECT_ID]/agones-udp-server:0.2
```

### Modify gameserver.yaml
Modify the following line from gameserver.yaml to use the new configuration.

```
    spec:
      containers:
      - name: agones-simple-udp
        image: gcr.io/[PROJECT_ID]/agones-udp-server:0.2
```

### Deploy Server
Apply the latest settings to kubernetes container.

```bash
>> gcloud config set container/cluster [CLUSTER_NAME]
>> gcloud container clusters get-credentials [CLUSTER_NAME]
>> kubectl apply -f gameserver.yaml
```
### Check the GameServer Status
```bash
>> kubectl describe gameserver
```

### Verify
Follow the instruction from this link: https://github.com/GoogleCloudPlatform/agones/blob/master/docs/create_gameserver.md.

Let's retrieve the IP address and the allocated port of your Game Server :

```
kubectl get gs simple-udp -o jsonpath='{.status.address}:{.status.port}'
```

You can now communicate with the Game Server :

> NOTE: if you do not have netcat installed
  (i.e. you get a response of `nc: command not found`),
  you can install netcat by running `sudo apt install netcat`.

```
nc -u {IP} {PORT}
Hello World !
ACK: Hello World !
EXIT
```

You can finally type `EXIT` which tells the SDK to run the [Shutdown command](../sdks#shutdown), and therefore shuts down the `GameServer`.  

If you run `kubectl describe gameserver` again - either the GameServer will be gone completely, or it will be in `Shutdown` state, on the way to being deleted.
