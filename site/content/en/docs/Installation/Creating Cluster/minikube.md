---
title: "Minikube"
linkTitle: "Minikube"
weight: 100
description: >
  Follow these steps to create a [Minikube](https://github.com/kubernetes/minikube) cluster
  for your Agones install.
---

## Installing Minikube

First, [install Minikube][minikube], which may also require you to install
a virtualisation solution, such as [VirtualBox][vb] as well.

[minikube]: https://minikube.sigs.k8s.io/docs/start/
[vb]: https://www.virtualbox.org

## Starting Minikube

Minikube will need to be started with the supported version of Kubernetes that is supported with Agones, via the 
`--kubernetes-version` command line flag.

Optionally, we also recommend starting with an `agones` profile, using `-p` to keep this cluster separate from any other 
clusters you may have running with Minikube.

```bash
minikube start --kubernetes-version v{{% k8s-version %}}.{{% minikube-k8s-minor-version %}} -p agones
```

Check the official [minikube start](https://minikube.sigs.k8s.io/docs/commands/start/) reference for more options that 
may be required for your platform of choice.

{{< alert title="Note" color="info">}}
You may need to increase the `--cpu` or `--memory` values for your minikube instance, depending on what resources are 
available on the host and/or how many GameServers you wish to run locally. 
{{< /alert >}}

## Local connection workarounds

Depending on your operating system and virtualization platform that you are using with Minikube, it may not be 
possible to connect directly to a `GameServer` hosted on Agones as you would on a cloud hosted Kubernetes cluster.

If you are unable to do so, the following workarounds are available, and may work on your platform:

### minikube ip

Rather than using the published IP of a `GameServer` to connect, run `minikube ip` to get the local IP for the 
minikube node, and connect to that address.

### Create a service

This would only be for local development, but if none of the other workarounds work, creating a Service for the 
`GameServer` you wish to connect to is a valid solution, to tunnel traffic to the appropriate GameServer container.

Use the following yaml:

```yaml
apiVersion: v1
kind: Service
metadata:
  name: agones-gameserver
spec:
  type: LoadBalancer
  selector:
    agones.dev/gameserver: ${GAMESERVER_NAME}
  ports:
  - protocol: UDP
    port: 7000 # local port
    targetPort: ${GAMESERVER_CONTAINER_PORT}
```

Where `${GAMESERVER_NAME}` is replaced with the GameServer you wish to connect to, and `${GAMESERVER_CONTAINER_PORT}`
is replaced with the container port GameServer exposes for connection.

Running `minikube service list -p agones` will show you the IP and port to connect to locally in the `URL` field.

To connect to a different `GameServer`, run `kubectl edit service agones-gameserver` and edit the `${GAMESERVER_NAME}` 
value to point to the new `GameServer` instance and/or the `${GAMESERVER_CONTAINER_PORT}` value as appropriate.

## Next Steps

- Continue to [Install Agones]({{< relref "../Install Agones/_index.md" >}}).
