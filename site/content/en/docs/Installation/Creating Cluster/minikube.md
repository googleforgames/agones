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

Depending on your Operating System, you may also need to change the `--driver`
([driver list](https://minikube.sigs.k8s.io/docs/drivers/)) to enable `GameServer` connectivity with or without
some workarounds listed below. 
{{< /alert >}}

### Known working drivers

Other operating systems and drivers may work, but at this stage have not been verified to work with UDP connections
via Agones exposed ports.

**Linux (amd64)**
* Docker (default)
* kvm2

**Mac (amd64)**
* Docker (default)
* Hyperkit

**Windows (amd64)**
* hyper-v (might need
  <a href="https://blog.thepolyglotprogrammer.com/setting-up-kubernetes-on-wsl-to-work-with-minikube-on-windows-10-90dac3c72fa1" data-proofer-ignore>this blog post</a>
  and/or [this comment](https://github.com/microsoft/WSL/issues/4288#issuecomment-652259640) for WSL support)

_If you have successfully tested with other platforms and drivers, please click "edit this page" in the top right hand
side and submit a pull request to let us know._

## Local connection workarounds

Depending on your operating system and virtualization platform that you are using with Minikube, it may not be
possible to connect directly to a `GameServer` hosted on Agones as you would on a cloud hosted Kubernetes cluster.

If you are unable to do so, the following workarounds are available, and may work on your platform:

### minikube ip

Rather than using the published IP of a `GameServer` to connect, run `minikube ip -p agones` to get the local IP for
the minikube node, and connect to that address.

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

{{< alert title="Warning" color="warning">}}
`minikube tunnel` ([docs](https://minikube.sigs.k8s.io/docs/handbook/accessing/))
does not support UDP ([Github Issue](https://github.com/kubernetes/minikube/issues/12362)) on some combination of
operating system, platforms and drivers, but is required when using the `Service` workaround.
{{< /alert >}}

### Use a different driver

If you cannot connect through the `Service`or use other workarounds, you may want to try a different
[minikube driver](https://minikube.sigs.k8s.io/docs/drivers/), and if that doesn't work, connection via UDP may not
be possible with minikube, and you may want to try either a
[different local Kubernetes tool](https://kubernetes.io/docs/tasks/tools/) or use a cloud hosted Kubernetes cluster.

## Next Steps

- Continue to [Install Agones]({{< relref "../Install Agones/_index.md" >}}).
