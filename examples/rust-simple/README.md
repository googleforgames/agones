# Simple Rust Example

This is a very simple "server" that doesn't do much other than show how the SDK works in Rust.

It will
- Setup the Agones SDK
- Call `SDK::Ready()` to register that it is ready with Agones.
- Every 10 seconds, write a log saying "Hi! I'm a Game Server"
- After 60 seconds, call `SDK::Shutdown()` to shut the server down.

## Running by minikube

First of all, you have to configure Agones on minikude. Check out [these instructions](https://github.com/GoogleCloudPlatform/agones/blob/3b856a4b90862a3ea183643869f81801d4468220/install/README.md).

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
