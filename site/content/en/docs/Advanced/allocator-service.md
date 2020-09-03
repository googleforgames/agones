---
title: "Allocator Service"
date: 2020-05-19T05:45:05Z
publishDate: 2019-10-25T05:45:05Z
description: >
  Agones provides an mTLS based allocator service that is accessible from outside the cluster using a load balancer. The service is deployed and scales independent to Agones controller.
---

To allocate a game server, Agones in addition to {{< ghlink href="pkg/apis/allocation/v1/gameserverallocation.go" >}}GameServerAllocations{{< /ghlink >}}, provides a gRPC service with mTLS authentication, called agones-allocator, which is on {{< ghlink href="proto/allocation" >}}stable version{{< /ghlink >}}, starting on agones v1.6.

The gRPC service is accessible through a Kubernetes service that is externalized using a load balancer. For the gRPC request to succeed, a client certificate must be provided that is in the authorization list of the allocator service.

The remainder of this article describes how to manually make a successful allocation request using the gRPC API.

## Find the external IP

The service is hosted under the same namespace as the Agones controller. To find the external IP of your allocator service, replace agones-system namespace with the namespace to which Agones is deployed and execute the following command:

```bash
kubectl get service agones-allocator -n agones-system
```

The output of the command should look like:

<pre>
NAME                        TYPE           CLUSTER-IP      <b>EXTERNAL-IP</b>     PORT(S)            AGE
agones-allocator            LoadBalancer   10.55.251.73    <b>34.82.195.204</b>   443:30250/TCP      7d22h
</pre>

## Server TLS certificate

{{% feature publishVersion="1.9.0" %}}
If the `agones-allocator` service is installed as a `LoadBalancer` [using a static IP]({{< relref "/docs/Installation/Install Agones/helm.md#reserved-allocator-load-balancer-ip" >}}), a valid self-signed server TLS certificate is generated using the IP provided. Otherwise, the server TLS certificate should be replaced.
{{% /feature %}}

Replace the default server TLS certificate with a certificate with CN and subjectAltName. There are multiple approaches to generate a certificate. Agones recommends using [cert-manager.io](https://cert-manager.io/) solution for cluster level certificate management.

In order to use cert-manager solution, first, [install cert-manager](https://cert-manager.io/docs/installation/kubernetes/) on the cluster. Then, [configure](https://cert-manager.io/docs/configuration/) an `Issuer`/`ClusterIssuer` resource and last configure a `Certificate` resource to manage allocator-tls `Secret`.

Here is an example of using a self-signed `ClusterIssuer` for configuring allocator-tls `Secret`:

```bash
#!/bin/bash
# Create a self-signed ClusterIssuer
cat <<EOF | kubectl apply -f -
apiVersion: cert-manager.io/v1alpha2
kind: ClusterIssuer
metadata:
  name: selfsigned
spec:
  selfSigned: {}
EOF

EXTERNAL_IP=`kubectl get services agones-allocator -n agones-system -o jsonpath='{.status.loadBalancer.ingress[0].ip}'`

# Create a Certificate with IP for the allocator-tls secret
cat <<EOF | kubectl apply -f -
apiVersion: cert-manager.io/v1alpha2
kind: Certificate
metadata:
  name: allocator-selfsigned-cert
  namespace: agones-system
spec:
  commonName: ${EXTERNAL_IP}
  ipAddresses:
    - ${EXTERNAL_IP}
  secretName: allocator-tls
  issuerRef:
    name: selfsigned
    kind: ClusterIssuer
EOF

# Optional: Store the secret ca.crt in a file to be used by the client for the server authentication
TLS_CA_FILE=ca.crt
TLS_CA_VALUE=`kubectl get secret allocator-tls -n agones-system -ojsonpath='{.data.ca\.crt}'`
echo ${TLS_CA_VALUE} | base64 -d > ${TLS_CA_FILE}

# In case of MacOS
# echo ${TLS_CA_VALUE} | base64 -D > ${TLS_CA_FILE}

# Add ca.crt to the allocator-tls-ca Secret
kubectl get secret allocator-tls-ca -o json -n agones-system | jq '.data["tls-ca.crt"]="'${TLS_CA_VALUE}'"' | kubectl apply -f -
```

## Client Certificate

Because agones-allocator uses an mTLS authentication mechanism, client must provide a certificate that is accepted by the server. Here is an example of generating a client certificate. For the agones-allocator service to accept the newly generate client certificate, the generated client certificate CA or public portion of the certificate must be added to a kubernetes secret called `allocator-client-ca`.

```bash
#!/bin/bash

KEY_FILE=client.key
CERT_FILE=client.crt

openssl req -x509 -nodes -days 365 -newkey rsa:2048 -keyout ${KEY_FILE} -out ${CERT_FILE}

CERT_FILE_VALUE=`cat ${CERT_FILE} | base64 -w 0`

# In case of MacOS
# CERT_FILE_VALUE=`cat ${CERT_FILE} | base64`

# white-list client certificate
kubectl get secret allocator-client-ca -o json -n agones-system | jq '.data["client_trial.crt"]="'${CERT_FILE_VALUE}'"' | kubectl apply -f -
```

The last command creates a new entry in the secret data map called `client_trial.crt` for `allocator-client-ca` and stores it. You can also achieve this by `kubectl edit secret allocator-client-ca -n agones-system`, and then add the entry.

## Send allocation request

Now the service is ready to accept requests from the client with the generated certificates. Create a [fleet]({{< ref "/docs/Getting Started/create-fleet.md" >}}) and send a gRPC request to agones-allocator. To start, take a look at the allocation gRPC client examples in {{< ghlink href="examples/allocator-client/main.go" >}}golang{{< /ghlink >}} and {{< ghlink href="examples/allocator-client-csharp/Program.cs" >}}C#{{< /ghlink >}} languages. In the following, the {{< ghlink href="examples/allocator-client/main.go" >}}golang gRPC client example{{< /ghlink >}} is used to allocate a Game Server in the default namespace.

```bash
#!/bin/bash

NAMESPACE=default # replace with any namespace
EXTERNAL_IP=`kubectl get services agones-allocator -n agones-system -o jsonpath='{.status.loadBalancer.ingress[0].ip}'`
KEY_FILE=client.key
CERT_FILE=client.crt
TLS_CA_FILE=ca.crt

go run examples/allocator-client/main.go --ip ${EXTERNAL_IP} \
    --port 443 \
    --namespace ${NAMESPACE} \
    --key ${KEY_FILE} \
    --cert ${CERT_FILE} \
    --cacert ${TLS_CA_FILE}
```

If your matchmaker is external to the cluster on which your game servers are hosted, the `agones-allocator` provides the gRPC API to allocate game services using mTLS authentication, which can scale independently to the Agones controller.
