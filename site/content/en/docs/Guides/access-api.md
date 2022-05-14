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
	"context"

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
	gs := &agonesv1.GameServer{ObjectMeta: metav1.ObjectMeta{GenerateName: "simple-game-server", Namespace: "default"},
		Spec: agonesv1.GameServerSpec{
			Container: "simple-game-server",
			Ports: []agonesv1.GameServerPort{{
				ContainerPort: 7654,
				HostPort:      7654,
				Name:          "gameport",
				PortPolicy:    agonesv1.Static,
				Protocol:      corev1.ProtocolUDP,
			}},
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{Name: "simple-game-server", Image: "{{% example-image %}}"}},
				},
			},
		},
	}
	newGS, err := agonesClient.AgonesV1().GameServers("default").Create(context.TODO(), gs, metav1.CreateOptions{})
	if err != nil {
		panic(err)
	}

	fmt.Printf("New game servers' name is: %s", newGS.ObjectMeta.Name)
}
```
In order to create GS using provided example, you can run it as a Kubernetes Job:
```bash
kubectl create -f https://raw.githubusercontent.com/googleforgames/agones/{{< release-branch >}}/examples/crd-client/create-gs.yaml --namespace agones-system
kubectl get pods --namespace agones-system
```
```
NAME                                 READY   STATUS      RESTARTS   AGE
create-gs-6wz86-7qsm5                0/1     Completed   0          6s
```
```bash
kubectl logs create-gs-6wz86-7qsm5  --namespace agones-system
```
```
{"message":"\u0026{0xc0001dde00 default}","severity":"info","source":"main","time":"2020-04-21T11:14:00.477576428Z"}
{"message":"New GameServer name is: helm-test-server-fxfgg","severity":"info","time":"2020-04-21T11:14:00.516024697Z"}
```
You have just created a GameServer using Kubernetes Go Client.

## Best Practice: Using Informers and Listers

Almost all  Kubernetes' controllers and custom controllers utilise `Informers` and `Listers`
to reduce the load on the Kubernetes's control plane.

Repetitive, direct access of the Kubernetes control plane API can significantly
reduce the performance of the cluster -- and Informers and Listers help resolving that issue.

Informers and Listers reduce the load on the Kubernetes control plane
by creating, using and maintaining an eventually consistent an in-memory cache.
This can be watched and also queried with zero cost, since it will only read against
its in-memory model of the Kubernetes resources.

Informer's role and Lister's role are different.

An Informer is the mechanism for watching a Kubernetes object's event,
such that when a Kubernetes object changes(e.g. CREATE,UPDATE,DELETE), the Informer is informed,
and can execute a callback with the relevant object as an argument.

This can be very useful for building event based systems against the Kubernetes API.

A Lister is the mechanism for querying Kubernetes object's against the client side in-memory cache.
Since the Lister stores objects in an in-memory cache, queries against a come at practically no cost.

Of course, Agones itself also uses Informers and Listers in its codebase.

### Example

The following is an example of Informers and Listers,
that show the GameServer's name & status & IPs in the Kubernetes cluster.

```go
package main

import (
	"context"
	"time"

	"agones.dev/agones/pkg/client/clientset/versioned"
	"agones.dev/agones/pkg/client/informers/externalversions"
	"agones.dev/agones/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

func main() {
	config, err := rest.InClusterConfig()
	logger := runtime.NewLoggerWithSource("main")
	if err != nil {
		logger.WithError(err).Fatal("Could not create in cluster config")
	}
	kubeClient, err := kubernetes.NewForConfig(config)
	agonesClient, err := versioned.NewForConfig(config)
	if err != nil {
		logger.WithError(err).Fatal("Could not create the agones api clientset")
	}

	// Create InformerFactory which create the informer
	informerFactory := informers.NewSharedInformerFactory(kubeClient, time.Second*30)
	agonesInformerFactory := externalversions.NewSharedInformerFactory(agonesClient, time.Second*30)

	// Create Pod informer by informerFactory
	podInformer := informerFactory.Core().V1().Pods()

	// Create GameServer informer by informerFactory
	gameServers := agonesInformerFactory.Agones().V1().GameServers()
	gsInformer := gameServers.Informer()

	// Add EventHandler to informer
	// When the object's event happens, the function will be called
	// For example, when the pod is added, 'AddFunc' will be called and put out the "Pod Added"
	podInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    func(new interface{}) { logger.Infof("Pod Added") },
		UpdateFunc: func(old, new interface{}) { logger.Infof("Pod Updated") },
		DeleteFunc: func(old interface{}) { logger.Infof("Pod Deleted") },
	})
	gsInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    func(new interface{}) { logger.Infof("GameServer Added") },
		UpdateFunc: func(old, new interface{}) { logger.Infof("GameServer Updated") },
		DeleteFunc: func(old interface{}) { logger.Infof("GameServer Deleted") },
	})

	ctx := context.Background()

	// Start Go routines for informer
	informerFactory.Start(ctx.Done())
	agonesInformerFactory.Start(ctx.Done())
	// Wait until finish caching with List API
	informerFactory.WaitForCacheSync(ctx.Done())
	agonesInformerFactory.WaitForCacheSync(ctx.Done())

	// Create Lister which can list objects from the in-memory-cache
	podLister := podInformer.Lister()
	gsLister := gameServers.Lister()

	for {
		// Get List objects of Pods from Pod Lister
		p := podLister.Pods("default")
		// Get List objects of GameServers from GameServer Lister
		gs, err := gsLister.List(labels.Everything())
		if err != nil {
			panic(err)
		}
		// Show GameServer's name & status & IPs
		for _, g := range gs {
			a, err := p.Get(g.GetName())
			if err != nil {
				panic(err)
			}
			logger.Infof("------------------------------")
			logger.Infof("Name: %s", g.GetName())
			logger.Infof("Status: %s", g.Status.State)
			logger.Infof("External IP: %s", g.Status.Address)
			logger.Infof("Internal IP: %s", a.Status.PodIP)
		}
		time.Sleep(time.Second * 25)
	}
}
```

You can list GameServer's name and status and IPs using Kubernetes Informers and Listers.

## Direct Access to the REST API via Kubectl

If there isn't a client written in your preferred language, it is always possible to communicate
directly with Kubernetes API to interact with Agones.

The Kubernetes API can be authenticated and exposed locally through the
[`kubectl proxy`](https://kubernetes.io/docs/tasks/extend-kubernetes/http-proxy-access-api/)


For example:

```bash
kubectl proxy &
```
```
Starting to serve on 127.0.0.1:8001
```

### List all Agones endpoints
```bash
curl http://localhost:8001/apis | grep agones -A 5 -B 5
```
```
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
```

### List Agones resources
```bash
curl http://localhost:8001/apis/agones.dev/v1
```
```
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
```

### List all gameservers in the default namespace
```bash
curl http://localhost:8001/apis/agones.dev/v1/namespaces/default/gameservers
```
```
{
    "apiVersion": "agones.dev/v1",
    "items": [
        {
            "apiVersion": "agones.dev/v1",
            "kind": "GameServer",
            "metadata": {
                "annotations": {
                    "kubectl.kubernetes.io/last-applied-configuration": "{\"apiVersion\":\"agones.dev/v1\",\"kind\":\"GameServer\",\"metadata\":{\"annotations\":{},\"name\":\"simple-game-server\",\"namespace\":\"default\"},\"spec\":{\"containerPort\":7654,\"hostPort\":7777,\"portPolicy\":\"static\",\"template\":{\"spec\":{\"containers\":[{\"image\":\"{{% example-image %}}\",\"name\":\"simple-game-server\"}]}}}}\n"
                },
                "clusterName": "",
                "creationTimestamp": "2018-03-02T21:41:05Z",
                "finalizers": [
                    "agones.dev"
                ],
                "generation": 0,
                "name": "simple-game-server",
                "namespace": "default",
                "resourceVersion": "760",
                "selfLink": "/apis/agones.dev/v1/namespaces/default/gameservers/simple-game-server",
                "uid": "692beea6-1e62-11e8-beb2-080027637781"
            },
            "spec": {
                "PortPolicy": "Static",
                "container": "simple-game-server",
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
                                "name": "simple-game-server",
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
```

### Allocate a gameserver from a fleet named 'simple-game-server', with GameServerAllocation
```bash
curl -d '{"apiVersion":"allocation.agones.dev/v1","kind":"GameServerAllocation","spec":{"required":{"matchLabels":{"agones.dev/fleet":"simple-game-server"}}}}' -H "Content-Type: application/json" -X POST http://localhost:8001/apis/allocation.agones.dev/v1/namespaces/default/gameserverallocations
```
```
{
    "kind": "GameServerAllocation",
    "apiVersion": "allocation.agones.dev/v1",
    "metadata": {
        "name": "simple-game-server-v6jwb-cmdcv",
        "namespace": "default",
        "creationTimestamp": "2019-07-03T17:19:47Z"
    },
    "spec": {
        "multiClusterSetting": {
            "policySelector": {}
        },
        "required": {
            "matchLabels": {
                "agones.dev/fleet": "simple-game-server"
            }
        },
        "scheduling": "Packed",
        "metadata": {}
    },
    "status": {
        "state": "Allocated",
        "gameServerName": "simple-game-server-v6jwb-cmdcv",
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
