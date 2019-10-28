---
title: "Allocator Service"
date: 2019-10-25T05:45:05Z
version: "v1alpha1"
description: >
  Agones provides an mTLS based allocator service that is accessible from outside the cluster using a load balancer. The service is deployed and scales independent to Agones controller.
---

{{< alert title="Alpha" color="warning">}}
This feature is in a pre-release state and might change.
{{< /alert >}}

To allocate a game server, Agones in addition to [GameServerAllocations](https://github.com/googleforgames/agones/blob/master/pkg/apis/allocation/v1/gameserverallocation.go), provides a REST API service with mTLS authentication, called agones-allocator, which is on [v1alpha1 version](https://github.com/googleforgames/agones/blob/master/proto/allocation/v1alpha1), starting on agones v1.1.

The REST API service is accessible through a Kubernetes service that is externalized using a load balancer. For the http request to succeed, a client certificate must be provided that is in the authorization list of the allocator service.

The remainder of this article describes how to manually make a successful allocation request using the REST API.

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

Store the IP in a variable to use as the server endpoint in the next sections:

```bash
EXTERNAL_IP=`kubectl get services agones-allocator -n agones-system -o jsonpath='{.status.loadBalancer.ingress[0].ip}'`
```

## Server TLS certificate

Replace the default server TLS certificate with a certificate with CN and subjectAltName. There are multiple approaches to generate a certificate, including using CA. The following provides an example of generating a self-signed certificate using openssl.

```bash
#!/bin/bash

TLS_KEY_FILE=tls.key
TLS_CERT_FILE=tls.crt

cat /etc/ssl/openssl.cnf <(printf "\n[SAN]\nsubjectAltName=IP:${EXTERNAL_IP}") > openssl.cnf

openssl req -nodes -new -newkey rsa:2048 \
    -keyout ${TLS_KEY_FILE} \
    -out tls.csr \
    -subj "/CN=${EXTERNAL_IP}/O=${EXTERNAL_IP}" \
    -reqexts SAN \
    -config openssl.cnf

openssl x509 -req -days 365 -in tls.csr \
    -signkey ${TLS_KEY_FILE} \
    -out ${TLS_CERT_FILE} \
    -extensions SAN \
    -extfile openssl.cnf
```

After having the TLS certificates ready, run the following command to store the certificate as a Kubernetes TLS secret.

```bash
kubectl create secret tls allocator-tls -n agones-system --key=${TLS_KEY_FILE} --cert=${TLS_CERT_FILE} --dry-run -o yaml | kubectl apply -f -
```

## Client Certificate

Because agones-allocator uses an mTLS authentication mechanism, client must provide a certificate that is accepted by the server. Here is an example of generating a client certificate:

```bash
#!/bin/bash

KEY_FILE=client.key
CERT_FILE=client.crt

openssl req -x509 -nodes -days 365 -newkey rsa:2048 -keyout ${KEY_FILE} -out ${CERT_FILE}

CERT_FILE_VALUE=`cat ${CERT_FILE} | base64 -w 0`
```

### White-list client certificate

For the agones-allocator service to accept the newly generate client certificate, the generated client certificate CA or public portion of the certificate must be added to a kubernetes secret called `allocator-client-ca`.

```bash
kubectl get secret allocator-client-ca -o json -n agones-system | jq '.data["client_trial.crt"]="'${CERT_FILE_VALUE}'"' | kubectl apply -f -
```

This command creates a new entry in the secret data map called `client_trial.crt` for `allocator-client-ca` and stores it. You can also achieve this by `kubectl edit secret allocator-client-ca -n agones-system`, and then add the entry.

## Restart pods

Restart pods to get the new TLS certificate loaded to the agones-allocator service.

```bash
kubectl get pods -n agones-system -o=name | grep agones-allocator | xargs kubectl delete -n agones-system
```

## Send allocation request

Now the service is ready to accept requests from the client with the generated certificates. Create a [fleet](https://agones.dev/site/docs/getting-started/create-fleet/#1-create-a-fleet) and send an HTTP request to agones-allocator by providing fleet's name and the namespace to which it is deployed, set in the JSON [body](https://github.com/googleforgames/agones/blob/master/proto/allocation/v1alpha1/allocation.proto).

```bash
#!/bin/bash

NAMESPACE=<namespace>
FLEET_NAME=<fleet name>

curl https://${EXTERNAL_IP}:443/v1alpha1/gameserverallocation \
    --header "Content-Type: application/json" \
    -d '{"namespace": "'${NAMESPACE}'", "requiredGameServerSelector": {"matchLabels": {"agones.dev/fleet": "'${FLEET_NAME}'"}}}' \
    --key ${KEY_FILE} \
    --cert ${CERT_FILE} \
    --cacert ${TLS_CERT_FILE} -v
```

If your matchmaker is external to the cluster on which your game servers are hosted, agones-allocator provides the HTTP API (and gRPC in future) to allocate game services using mTLS authentication, which can scale independent to agones controller.
