---
title: "Access Agones via the Kubernetes API"
linkTitle: "Access Agones via the K8s API"
date: 2019-01-03T01:20:41Z
weight: 50
description: >
  It's likely that we will want to programmatically interact with Agones. Everything that can be done
  via the `kubectl` and yaml configurations can also be done via
  the [Kubernetes API](https://kubernetes.io/docs/concepts/overview/kubernetes-api/).
---

Installing Agones creates several [Custom Resource Definitions (CRD)](https://kubernetes.io/docs/concepts/api-extension/custom-resources),
which can be accessed and manipulated through the Kubernetes API. 

The detailed list of Agones CRDs with their parameters could be found here - [Agones CRD API Reference](../../reference/agones_crd_api_reference/).

Kubernetes has multiple [client libraries](https://kubernetes.io/docs/reference/using-api/client-libraries/), however,
at time of writing, only
the [Go](https://github.com/kubernetes/client-go) and
[Python](https://github.com/kubernetes-client/python/) clients are documented to support accessing CRDs.

This can be found in the [Accessing a custom resource](https://kubernetes.io/docs/concepts/api-extension/custom-resources/#accessing-a-custom-resource)
section of the Kubernetes documentation.

At this time, we recommend interacting with Agones through the Go client that has been generated in this repository,
but other methods may also work as well.

## Go Client

Kubernetes Go Client tooling generates a Client for Agones that we can use to interact with the Agones
installation on our Kubernetes cluster.

- [Godoc for the Agones Client](https://pkg.go.dev/agones.dev/agones/pkg/client/clientset/versioned)
- [Godoc for the standard Kubernetes Client](https://pkg.go.dev/k8s.io/client-go/kubernetes)

### Authentication

This client uses the same authentication mechanisms as the Kubernetes API.

If you plan to run your code in the same cluster as the Agones install, have a look at the
[in cluster configuration](https://github.com/kubernetes/client-go/tree/master/examples/in-cluster-client-configuration)
example from the Kubernetes Client.

If you plan to run your code outside the Kubernetes cluster as your Agones install,
look at the [out of cluster configuration](https://github.com/kubernetes/client-go/tree/master/examples/out-of-cluster-client-configuration)
example from the Kubernetes client.

### Example

The following is an example of a in-cluster configuration, that creates a `Clientset` for Agones
and then creates a `GameServer`.

A full example code is available in the {{< ghlink href="examples/crd-client/main.go" >}} example folder{{< /ghlink >}}.

```go
package main

import (
	"fmt"

	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	"agones.dev/agones/pkg/client/clientset/versioned"
	"agones.dev/agones/pkg/util/runtime"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func main() {
	config, err := rest.InClusterConfig()
	logger := runtime.NewLoggerWithSource("main")
	if err != nil {
		logger.WithError(err).Fatal("Could not create in cluster config")
	}

	// Access to standard Kubernetes resources through the Kubernetes Clientset
	// We don't actually need this for this example, but it's just here for
	// illustrative purposes
	kubeClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		logger.WithError(err).Fatal("Could not create the kubernetes clientset")
	}

	// Access to the Agones resources through the Agones Clientset
	// Note that we reuse the same config as we used for the Kubernetes Clientset
	agonesClient, err := versioned.NewForConfig(config)
	if err != nil {
		logger.WithError(err).Fatal("Could not create the agones api clientset")
	}

	// Create a GameServer
	gs := &agonesv1.GameServer{ObjectMeta: metav1.ObjectMeta{GenerateName: "udp-server", Namespace: "default"},
		Spec: agonesv1.GameServerSpec{
			Container: "udp-server",
			Ports: []agonesv1.GameServerPort{{
				ContainerPort: 7654,
				HostPort:      7654,
				Name:          "gameport",
				PortPolicy:    agonesv1.Static,
				Protocol:      corev1.ProtocolUDP,
			}},
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{Name: "udp-server", Image: "{{% example-image %}}"}},
				},
			},
		},
	}
	newGS, err := agonesClient.AgonesV1().GameServers("default").Create(gs)
	if err != nil {
		panic(err)
	}

	fmt.Printf("New game servers' name is: %s", newGS.ObjectMeta.Name)
}
```
In order to create GS using provided example, you can run it as a Kubernetes Job:
```bash
$ kubectl create -f https://raw.githubusercontent.com/googleforgames/agones/{{< release-branch >}}/examples/crd-client/create-gs.yaml --namespace agones-system
$ kubectl get pods --namespace agones-system
NAME                                 READY   STATUS      RESTARTS   AGE
create-gs-6wz86-7qsm5                0/1     Completed   0          6s
$ kubectl logs create-gs-6wz86-7qsm5  --namespace agones-system
{"message":"\u0026{0xc0001dde00 default}","severity":"info","source":"main","time":"2020-04-21T11:14:00.477576428Z"}
{"message":"New GameServer name is: helm-test-server-fxfgg","severity":"info","time":"2020-04-21T11:14:00.516024697Z"}
```
You have just created a GameServer using Kubernetes Go Client.

## Direct Access to the REST API via Kubectl

If there isn't a client written in your preferred language, it is always possible to communicate
directly with Kubernetes API to interact with Agones.

The Kubernetes API can be authenticated and exposed locally through the
[`kubectl proxy`](https://kubernetes.io/docs/tasks/extend-kubernetes/http-proxy-access-api/)


For example:

```bash
$ kubectl proxy &
Starting to serve on 127.0.0.1:8001

# list all Agones endpoints
$ curl http://localhost:8001/apis | grep agones -A 5 -B 5
...
    {
      "name": "agones.dev",
      "versions": [
        {
          "groupVersion": "agones.dev/v1",
          "version": "v1"
        }
      ],
      "preferredVersion": {
        "groupVersion": "agones.dev/v1",
        "version": "v1"
      },
      "serverAddressByClientCIDRs": null
    }
...

# List Agones resources
$ curl http://localhost:8001/apis/agones.dev/v1
{
  "kind": "APIResourceList",
  "apiVersion": "v1",
  "groupVersion": "agones.dev/v1",
  "resources": [
    {
      "name": "gameservers",
      "singularName": "gameserver",
      "namespaced": true,
      "kind": "GameServer",
      "verbs": [
        "delete",
        "deletecollection",
        "get",
        "list",
        "patch",
        "create",
        "update",
        "watch"
      ],
      "shortNames": [
        "gs"
      ]
    }
  ]
}

# list all gameservers in the default namespace
$ curl http://localhost:8001/apis/agones.dev/v1/namespaces/default/gameservers
{
    "apiVersion": "agones.dev/v1",
    "items": [
        {
            "apiVersion": "agones.dev/v1",
            "kind": "GameServer",
            "metadata": {
                "annotations": {
                    "kubectl.kubernetes.io/last-applied-configuration": "{\"apiVersion\":\"agones.dev/v1\",\"kind\":\"GameServer\",\"metadata\":{\"annotations\":{},\"name\":\"simple-udp\",\"namespace\":\"default\"},\"spec\":{\"containerPort\":7654,\"hostPort\":7777,\"portPolicy\":\"static\",\"template\":{\"spec\":{\"containers\":[{\"image\":\"{{% example-image %}}\",\"name\":\"simple-udp\"}]}}}}\n"
                },
                "clusterName": "",
                "creationTimestamp": "2018-03-02T21:41:05Z",
                "finalizers": [
                    "agones.dev"
                ],
                "generation": 0,
                "name": "simple-udp",
                "namespace": "default",
                "resourceVersion": "760",
                "selfLink": "/apis/agones.dev/v1/namespaces/default/gameservers/simple-udp",
                "uid": "692beea6-1e62-11e8-beb2-080027637781"
            },
            "spec": {
                "PortPolicy": "Static",
                "container": "simple-udp",
                "containerPort": 7654,
                "health": {
                    "failureThreshold": 3,
                    "initialDelaySeconds": 5,
                    "periodSeconds": 5
                },
                "hostPort": 7777,
                "protocol": "UDP",
                "template": {
                    "metadata": {
                        "creationTimestamp": null
                    },
                    "spec": {
                        "containers": [
                            {
                                "image": "{{% example-image %}}",
                                "name": "simple-udp",
                                "resources": {}
                            }
                        ]
                    }
                }
            },
            "status": {
                "address": "192.168.99.100",
                "nodeName": "agones",
                "port": 7777,
                "state": "Ready"
            }
        }
    ],
    "kind": "GameServerList",
    "metadata": {
        "continue": "",
        "resourceVersion": "1062",
        "selfLink": "/apis/agones.dev/v1/namespaces/default/gameservers"
    }
}

# allocate a gameserver from a fleet named 'simple-udp', with GameServerAllocation

$ curl -d '{"apiVersion":"allocation.agones.dev/v1","kind":"GameServerAllocation","spec":{"required":{"matchLabels":{"agones.dev/fleet":"simple-udp"}}}}' -H "Content-Type: application/json" -X POST http://localhost:8001/apis/allocation.agones.dev/v1/namespaces/default/gameserverallocations

{
    "kind": "GameServerAllocation",
    "apiVersion": "allocation.agones.dev/v1",
    "metadata": {
        "name": "simple-udp-v6jwb-cmdcv",
        "namespace": "default",
        "creationTimestamp": "2019-07-03T17:19:47Z"
    },
    "spec": {
        "multiClusterSetting": {
            "policySelector": {}
        },
        "required": {
            "matchLabels": {
                "agones.dev/fleet": "simple-udp"
            }
        },
        "scheduling": "Packed",
        "metadata": {}
    },
    "status": {
        "state": "Allocated",
        "gameServerName": "simple-udp-v6jwb-cmdcv",
        "ports": [
            {
                "name": "default",
                "port": 7445
            }
        ],
        "address": "34.94.118.237",
        "nodeName": "gke-test-cluster-default-f11755a7-5km3"
    }
}
```

You may wish to review the [Agones Kubernetes API]({{< ref "/docs/Reference/agones_crd_api_reference.html" >}}) for the full data structure reference.

The [Kubernetes API Concepts](https://kubernetes.io/docs/reference/using-api/api-concepts/)
section may also provide the more details on the API conventions that are used in the Kubernetes API.

## Next Steps

- Learn how to use [Allocator Service]({{< relref "../Advanced/allocator-service.md" >}}) for single and multi-cluster Allocation.
