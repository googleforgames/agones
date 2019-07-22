# Simple Rust Example

This is a very simple "server" that doesn't do much other than show how the SDK works in Rust.

It will
- Setup the Agones SDK
- Call `SDK::Ready()` to register that it is ready with Agones.
- Every 10 seconds, write a log saying "Hi! I'm a Game Server"
- After 60 seconds, call `SDK::Shutdown()` to shut the server down.

## Running locally (without Docker)

This will build example server which will run for 100 seconds:
```
make build
```

In a separate terminal run next command:
```
cd ../../build; make run-sdk-conformance-local TIMEOUT=120 TESTS=ready,watch,health,gameserver
```
This will start an SDK-server in a docker, which will be running for 120 seconds.

Run the Rust Simple Gameserver binary:
```
make run
```

You will see the following output:
```
Rust Game Server has started!
Creating SDK instance
Setting a label
Starting to watch GameServer updates...
Health ping sent
Setting an annotation
...
```

Clean the resulting `sdk` directory and `target` folder:
```
make clean
```


## Running locally with Docker

Build the container locally:
```
make build-image
```

In a separate terminal run next command:
```
cd ../../build; make run-sdk-conformance-local TIMEOUT=120 TESTS=ready,watch,health,gameserver
```
This will start an SDK-server in a docker, which will be running for 120 seconds.

Run next make targets:
```
make run-image
```

You will see the following output:
```
docker run --network=host rust-simple-server:0.2
Rust Game Server has started!
Creating SDK instance
Setting a label
Starting to watch GameServer updates...
Health ping sent
Setting an annotation
GameServer Update, name: local
GameServer Update, state: Ready
Marking server as ready...
...marked Ready
Getting GameServer details...
GameServer name: local
Running for 0 seconds
Health ping sent
Health ping sent
Health ping sent
Health ping sent
Running for 10 seconds
```
Clean the resulting `sdk` directory:
```
make clean-docker
```

## Running by minikube

First of all, you have to configure Agones on minikube. Check out [these instructions](https://agones.dev/site/docs/installation/#setting-up-a-minikube-cluster).

```
$ eval $(minikube docker-env)
$ make build-image
$ kubectl create -f gameserver.yaml
```

You can see output of the example by the following.

```
$ POD_NAME=`kubectl get pods -o go-template --template '{{range .items}}{{.metadata.name}}{{"\n"}}{{end}}'`
$ kubectl logs $POD_NAME -c rust-simple
Rust Game Server has started!
Creating SDK instance
Marking server as ready...
Running for 0 seconds
Health ping sent
Health ping sent
Health ping sent
Health ping sent
```
