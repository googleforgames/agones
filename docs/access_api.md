# Accessing Agones via the Kubernetes API

It's likely that we will want to programmatically interact with Agones. Everything that can be done
via the `kubectl` and yaml configurations can also be done via
the [Kubernetes API](https://kubernetes.io/docs/concepts/overview/kubernetes-api/).

Installing Agones creates several [Custom Resource Definitions (CRD)](https://kubernetes.io/docs/concepts/api-extension/custom-resources),
which can be accessed and manipulated through the Kubernetes API.

Kubernetes has multiple [client libraries](https://kubernetes.io/docs/reference/client-libraries/), however,
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

- [Godoc for the Agones Client](https://godoc.org/agones.dev/agones/pkg/client/clientset/versioned)
- [Godoc for the standard Kubernetes Client](https://godoc.org/k8s.io/client-go)

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

```go
package main

import (
	"fmt"

	"agones.dev/agones/pkg/apis/stable/v1alpha1"
	"agones.dev/agones/pkg/client/clientset/versioned"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/kubernetes"
)

func main() {
	config, err := rest.InClusterConfig()
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
	gs := &v1alpha1.GameServer{ObjectMeta: metav1.ObjectMeta{GenerateName: "udp-server", Namespace: "default"},
		Spec: v1alpha1.GameServerSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{Name: "udp-server", Image: "gcr.io/agones-images/udp-server:0.4"}},
				},
			},
		},
	}
	newGS, err := agonesClient.StableV1alpha1().GameServers("default").Create(gs)
	if err != nil {
		panic(err)
	}

	fmt.Printf("New game servers' name is: %s", newGS.ObjectMeta.Name)
}

```

## Direct Access to the REST API via Kubectl

If there isn't a client written in your preferred language, it is always possible to communicate
directly with Kubernetes API to interact with Agones.

The Kubernetes API can be authenticated and exposed locally through the
[`kubectl proxy`](https://kubernetes.io/docs/tasks/access-kubernetes-api/http-proxy-access-api/)


For example:

```bash
$ kubectl proxy &
Starting to serve on 127.0.0.1:8001

# list all Agones endpoints
$ curl http://localhost:8001/apis | grep agones -A 5 -B 5
...
    {
      "name": "stable.agones.dev",
      "versions": [
        {
          "groupVersion": "stable.agones.dev/v1alpha1",
          "version": "v1alpha1"
        }
      ],
      "preferredVersion": {
        "groupVersion": "stable.agones.dev/v1alpha1",
        "version": "v1alpha1"
      },
      "serverAddressByClientCIDRs": null
    }
...

# List Agones resources
$ curl http://localhost:8001/apis/stable.agones.dev/v1alpha1
{
  "kind": "APIResourceList",
  "apiVersion": "v1",
  "groupVersion": "stable.agones.dev/v1alpha1",
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

# list all gameservers in the default namesace
$ curl http://localhost:8001/apis/stable.agones.dev/v1alpha1/namespaces/default/gameservers
{
    "apiVersion": "stable.agones.dev/v1alpha1",
    "items": [
        {
            "apiVersion": "stable.agones.dev/v1alpha1",
            "kind": "GameServer",
            "metadata": {
                "annotations": {
                    "kubectl.kubernetes.io/last-applied-configuration": "{\"apiVersion\":\"stable.agones.dev/v1alpha1\",\"kind\":\"GameServer\",\"metadata\":{\"annotations\":{},\"name\":\"simple-udp\",\"namespace\":\"default\"},\"spec\":{\"containerPort\":7654,\"hostPort\":7777,\"portPolicy\":\"static\",\"template\":{\"spec\":{\"containers\":[{\"image\":\"gcr.io/agones-images/udp-server:0.4\",\"name\":\"simple-udp\"}]}}}}\n"
                },
                "clusterName": "",
                "creationTimestamp": "2018-03-02T21:41:05Z",
                "finalizers": [
                    "stable.agones.dev"
                ],
                "generation": 0,
                "name": "simple-udp",
                "namespace": "default",
                "resourceVersion": "760",
                "selfLink": "/apis/stable.agones.dev/v1alpha1/namespaces/default/gameservers/simple-udp",
                "uid": "692beea6-1e62-11e8-beb2-080027637781"
            },
            "spec": {
                "PortPolicy": "static",
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
                                "image": "gcr.io/agones-images/udp-server:0.4",
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
        "selfLink": "/apis/stable.agones.dev/v1alpha1/namespaces/default/gameservers"
    }
}

# allocate a gameserver from a fleet named 'simple-udp'
# (in 0.4.0 you won't need to specify the namespace in the FleetAllocation metadata config)

$ curl -d '{"apiVersion":"stable.agones.dev/v1alpha1","kind":"FleetAllocation","metadata":{"generateName":"simple-udp-", "namespace": "default"},"spec":{"fleetName":"simple-udp"}}' -H "Content-Type: application/json" -X POST http://localhost:8001/apis/stable.agones.dev/v1alpha1/namespaces/default/fleetallocations

{
    "apiVersion": "stable.agones.dev/v1alpha1",
    "kind": "FleetAllocation",
    "metadata": {
        "clusterName": "",
        "creationTimestamp": "2018-08-22T17:08:30Z",
        "generateName": "simple-udp-",
        "generation": 1,
        "name": "simple-udp-4xsrl",
        "namespace": "default",
        "ownerReferences": [
            {
                "apiVersion": "stable.agones.dev/v1alpha1",
                "blockOwnerDeletion": true,
                "controller": true,
                "kind": "GameServer",
                "name": "simple-udp-296d5-4qcds",
                "uid": "99832e51-a62b-11e8-b7bb-bc2623b75dea"
            }
        ],
        "resourceVersion": "1228",
        "selfLink": "/apis/stable.agones.dev/v1alpha1/namespaces/default/fleetallocations/simple-udp-4xsrl",
        "uid": "fe8717ae-a62d-11e8-b7bb-bc2623b75dea"
    },
    "spec": {
        "fleetName": "simple-udp",
        "metadata": {}
    },
    "status": {
        "GameServer": {
            "metadata": {
                "creationTimestamp": "2018-08-22T16:51:22Z",
                "finalizers": [
                    "stable.agones.dev"
                ],
                "generateName": "simple-udp-296d5-",
                "generation": 1,
                "labels": {
                    "stable.agones.dev/gameserverset": "simple-udp-296d5"
                },
                "name": "simple-udp-296d5-4qcds",
                "namespace": "default",
                "ownerReferences": [
                    {
                        "apiVersion": "stable.agones.dev/v1alpha1",
                        "blockOwnerDeletion": true,
                        "controller": true,
                        "kind": "GameServerSet",
                        "name": "simple-udp-296d5",
                        "uid": "9980351b-a62b-11e8-b7bb-bc2623b75dea"
                    }
                ],
                "resourceVersion": "1225",
                "selfLink": "/apis/stable.agones.dev/v1alpha1/namespaces/default/gameservers/simple-udp-296d5-4qcds",
                "uid": "99832e51-a62b-11e8-b7bb-bc2623b75dea"
            },
            "spec": {
                "container": "simple-udp",
                "health": {
                    "failureThreshold": 3,
                    "initialDelaySeconds": 5,
                    "periodSeconds": 5
                },
                "ports": [
                    {
                        "containerPort": 7654,
                        "hostPort": 7968,
                        "name": "default",
                        "portPolicy": "dynamic",
                        "protocol": "UDP"
                    }
                ],
                "template": {
                    "metadata": {
                        "creationTimestamp": null
                    },
                    "spec": {
                        "containers": [
                            {
                                "image": "gcr.io/agones-images/udp-server:0.4",
                                "name": "simple-udp",
                                "resources": {}
                            }
                        ]
                    }
                }
            },
            "status": {
                "address": "192.168.39.184",
                "nodeName": "minikube",
                "ports": [
                    {
                        "name": "default",
                        "port": 7968
                    }
                ],
                "state": "Allocated"
            }
        }
    }
}
```

The [Verb Resources](https://github.com/kubernetes/community/blob/master/contributors/devel/api-conventions.md#verbs-on-resources)
section provide the more details on the API conventions that are used in the Kubernetes API.

It may also be useful to look at the [API patterns for standard Kubernetes resources](https://v1-10.docs.kubernetes.io/docs/reference/generated/kubernetes-api/v1.10/#-strong-write-operations-strong--54).


## Next Steps

Learn how to interact with Agones programmatically through the API while creating an [Allocator Service](./create_allocator_service.md).
