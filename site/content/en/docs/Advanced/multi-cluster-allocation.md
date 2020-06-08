---
title: "Multi-cluster Allocation"
date: 2019-10-25T05:45:05Z
description: >
  In order to allow allocation from multiple clusters, Agones provides a mechanism to set redirect rules for allocation requests to the right cluster.
---

{{< alert title="Beta" color="warning">}}
This feature is in a pre-release state and might change.
{{< /alert >}}

There may be different types of clusters, such as on-premise, and Google Kubernetes Engine (GKE), used by a game to help with the cost-saving and availability. 
For this purpose, Agones provides a mechanism to define priorities on the clusters. Priorities are defined on {{< ghlink href="pkg/apis/multicluster/v1/gameserverallocationpolicy.go" >}}GameServerAllocationPolicy{{< /ghlink >}} agones CRD. A matchmaker can enable the multi-cluster rules on a request and target [agones-allocator]({{< relref "allocator-service.md">}}) endpoint in any of the clusters and get resources allocated on the cluster with the highest priority. If the cluster with the highest priority is overloaded, the allocation request is redirected to the cluster with the next highest priority.

The remainder of this article describes how to enable multi-cluster allocation.

## Define Cluster Priority

{{< ghlink href="pkg/apis/multicluster/v1/gameserverallocationpolicy.go" >}}GameServerAllocationPolicy{{< /ghlink >}} is the CRD defined by Agones for setting multi-cluster allocation rules. In addition to cluster priority, it describes the connection information for the target cluster, including the game server namespace, agones-allocator endpoint and client K8s secrets name for redirecting the allocation request. Here is an example of setting the priority for a cluster and it's connection rules. One such resource should be defined per cluster. For clusters with the same priority, the cluster is chosen with a probability relative to its weight.

In the following example the policy is defined for cluster B in cluster A.

```bash
cat <<EOF | kubectl apply -f -
apiVersion: multicluster.agones.dev/v1
kind: GameServerAllocationPolicy
metadata:
  name: allocator-cluster-B
  namespace: cluster-A-ns
spec:
  connectionInfo:
    allocationEndpoints:
    - 34.82.195.204
    clusterName: "clusterB"
    namespace: cluster-B-ns
    secretName: allocator-client-to-cluster-B
    serverCa: c2VydmVyQ0E=
  priority: 1
  weight: 100
EOF
```

To define the local cluster priority, similarly, an allocation rule should be defined, while leaving allocationEndpoints unset. If the local cluster priority is not defined, the allocation from the local cluster happens only if allocation from other clusters with the existing allocation rules is unsuccessful.

`serverCa` is the server TLS CA public certificate, set only if the remote server certificate is not signed by a public CA (e.g. self-signed). If this field is not specified, the certificate can also be specified in the `ca.crt` field of the client secret (i.e. the secret referred to in the `secretName` field).

## Establish trust

To accept allocation requests from other clusters, agones-allocator for cluster B should be configured to accept the client's certificate from cluster A and the cluster A's client should be configured to accept the server TLS certificate, if it is not signed by a public Certificate Authority (CA).

Follow the steps to configure the [agones allocator gRPC service]({{< relref "allocator-service.md">}}). The client certificate pair in the mentioned document is stored as a K8s secret. Here are the secrets to set:

1.Client certificate to talk to other clusters:

```bash
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: Secret
metadata:
  name: allocator-client-to-cluster-B
  namespace: cluster-A-ns
type: Opaque
data:
  tls.crt: <REDACTED>
  tls.key: <REDACTED>
  ca.crt: <REDACTED>
EOF
```

The certificates are base 64 string of the certificate file e.g. `cat ${CERT_FILE} | base64 -w 0`

Agones recommends using [cert-manager.io](https://cert-manager.io/) solution for generating client certificates. 

2.Add client CA to the list of authorized client certificates by agones-allocator in the targeted cluster.

```bash
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: Secret
metadata:
  name: allocator-client-ca
  namespace: agones-system
type: Opaque
data:
  client1.crt: <REDACTED>
  client2.crt: <REDACTED>
  …
  clientN.crt: <REDACTED>
EOF
```

## Allocate multi-cluster

To enable multi-cluster allocation, set `multiClusterSetting.enabled` to `true` in {{< ghlink href="proto/allocation/allocation.proto" >}}allocation.proto{{< /ghlink >}} and send allocation requests. For more information visit [agones-allocator]({{< relref "allocator-service.md">}}). In the following, using {{< ghlink href="examples/allocator-client/main.go" >}}allocator-client sample{{< /ghlink >}}, a multi-cluster allocation request is sent to the agones-allocator service.

```bash
#!/bin/bash
EXTERNAL_IP=`kubectl get services agones-allocator -n agones-system -o jsonpath='{.status.loadBalancer.ingress[0].ip}'`

NAMESPACE=default # replace with any namespace

go run examples/allocator-client/main.go --ip ${EXTERNAL_IP} \
    --namespace ${NAMESPACE} \
    --key ${KEY_FILE} \
    --cert ${CERT_FILE} \
    --cacert ${TLS_CA_FILE} \
    --multicluster true
```
