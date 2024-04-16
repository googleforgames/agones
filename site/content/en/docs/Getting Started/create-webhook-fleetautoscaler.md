---
title: "Quickstart: Create a Fleet Autoscaler with Webhook Policy"
linkTitle: "Create a Webhook Fleetautoscaler"
date: 2019-01-02T06:42:44Z
weight: 40
description: >
  This guide covers how you can create a webhook fleet autoscaler policy.
---

In some cases, your game servers may need to use custom logic for scaling your fleet that is more complex than what
can be expressed using the Buffer policy in the fleetautoscaler. This guide shows how you can extend Agones
with an autoscaler webhook to implement a custom autoscaling policy.

When you use an autoscaler webhook the logic computing the number of target replicas is delegated to an external
HTTP/S endpoint, such as one provided by a Kubernetes deployment and service in the same cluster (as shown in the
examples below). The fleetautoscaler will send a request to the webhook autoscaler's `/scale` endpoint every sync
period (currently 30s) with a JSON body, and scale the target fleet based on the data that is returned.

## Chapter 1 Configuring HTTP fleetautoscaler webhook

### Prerequisites

It is assumed that you have completed the instructions to [Create a Game Server Fleet]({{< relref "./create-fleet.md" >}}) and have a running fleet of game servers.

### Objectives

- Run a fleet
- Deploy the Webhook Pod and service for autoscaling
- Create a Fleet Autoscaler with Webhook policy type in Kubernetes using Agones custom resource
- Watch the Fleet scales up when allocating GameServers
- Watch the Fleet scales down after GameServer shutdown

#### 1. Deploy the fleet

Run a fleet in a cluster:
```bash
kubectl apply -f https://raw.githubusercontent.com/googleforgames/agones/{{< release-branch >}}/examples/simple-game-server/fleet.yaml
```

#### 2. Deploy a Webhook service for autoscaling

In this step we would deploy an example webhook that will control the size of the fleet based on allocated gameservers
portion in a fleet. You can see the source code for this example webhook server {{< ghlink href="examples/autoscaler-webhook/main.go" >}}here{{< /ghlink >}}.
The fleetautoscaler would trigger this endpoint every 30 seconds. More details could be found {{< ghlink href="examples/autoscaler-webhook/" >}}also here{{< /ghlink >}}.
We need to create a pod which will handle HTTP requests with json payload
[`FleetAutoscaleReview`]({{< relref "../Reference/fleetautoscaler.md#webhook-endpoint-specification" >}}) and return back it
with [`FleetAutoscaleResponse`]({{< relref "../Reference/fleetautoscaler.md#webhook-endpoint-specification" >}}) populated.

The `Scale` flag and `Replicas` values returned in the `FleetAutoscaleResponse` tells the FleetAutoscaler what target size the backing Fleet should be scaled up or down to. If `Scale` is false - no scaling occurs.

Run next command to create a service and a Webhook pod in a cluster:
```bash
kubectl apply -f https://raw.githubusercontent.com/googleforgames/agones/{{< release-branch >}}/examples/autoscaler-webhook/autoscaler-service.yaml
```

To check that it is running and liveness probe is fine:
```bash
kubectl describe pod autoscaler-webhook
```

```
Name:           autoscaler-webhook-86944884c4-sdtqh
Namespace:      default
Node:           gke-test-cluster-default-1c5dec79-h0tq/10.138.0.2
...
Status:         Running
```

#### 3. Create a Fleet Autoscaler

Let's create a Fleet Autoscaler using the following command:

```bash
kubectl apply -f https://raw.githubusercontent.com/googleforgames/agones/{{< release-branch >}}/examples/webhookfleetautoscaler.yaml
```

You should see a successful output similar to this:

```
fleetautoscaler.autoscaling.agones.dev "webhook-fleet-autoscaler" created
```

This has created a FleetAutoscaler record inside Kubernetes.
It has the link to Webhook service we deployed above.

#### 4. See the fleet and autoscaler status.

In order to track the list of gameservers which run in your fleet you can run this command in a separate terminal tab:

```bash
 watch "kubectl get gs -n default"
```

In order to get autoscaler status use the following command:

```bash
kubectl describe fleetautoscaler webhook-fleet-autoscaler
```

It should look something like this:

```
Name:         webhook-fleet-autoscaler
Namespace:    default
Labels:       <none>
Annotations:  kubectl.kubernetes.io/last-applied-configuration={"apiVersion":
"autoscaling.agones.dev/v1","kind":"FleetAutoscaler","metadata":{"annotations"
:{},"name":"webhook-fleet-autoscaler","namespace":"default...
API Version:  autoscaling.agones.dev/v1
Kind:         FleetAutoscaler
etadata:
  Cluster Name:
  Creation Timestamp:  2018-12-22T12:52:23Z
  Generation:          1
  Resource Version:    2274579
  Self Link:           /apis/autoscaling.agones.dev/v1/namespaces/default/fleet
autoscalers/webhook-fleet-autoscaler
  UID:                 6d03eae4-05e8-11e9-84c2-42010a8a01c9
Spec:
  Fleet Name:  simple-game-server
  Policy:
    Type:  Webhook
    Webhook:
      Service:
        Name:       autoscaler-webhook-service
        Namespace:  default
        Path:       scale
      URL:
Status:
  Able To Scale:     true
  Current Replicas:  2
  Desired Replicas:  2
  Last Scale Time:   <nil>
  Scaling Limited:   false
Events:              <none>
```

You can see the status (able to scale, not limited), the last time the fleet was scaled (nil for never), current and desired fleet size.

The autoscaler makes a query to a webhoook service deployed on step 1 and on response changing the target Replica size, and the fleet creates/deletes game server instances
to achieve that number. The convergence is achieved in time, which is usually measured in seconds.

#### 5. Allocate Game Servers from the Fleet to trigger scale up

If you're interested in more details for game server allocation, you should consult the [Create a Game Server Fleet]({{< relref "../Getting Started/create-fleet.md" >}}) page.
Here we only interested in triggering allocations to see the autoscaler in action.

```bash
kubectl create -f https://raw.githubusercontent.com/googleforgames/agones/{{< release-branch >}}/examples/simple-game-server/gameserverallocation.yaml -o yaml
```

You should get in return the allocated game server details, which should end with something like:

```
status:
  address: 34.94.118.237
  gameServerName: simple-game-server-v6jwb-6bzkz
  nodeName: gke-test-cluster-default-f11755a7-5km3
  ports:
  - name: default
    port: 7832
  state: Allocated
```

Note the address and port, you might need them later to connect to the server.

Run the kubectl command one more time so that we have both servers allocated:
```bash
kubectl create -f https://raw.githubusercontent.com/googleforgames/agones/{{< release-branch >}}/examples/simple-game-server/gameserverallocation.yaml -o yaml
```

#### 6. Check new Autoscaler and Fleet status

Now let's wait a few seconds to allow the autoscaler to detect the change in the fleet and check again its status

```bash
kubectl describe fleetautoscaler webhook-fleet-autoscaler
```

The last part should look similar to this:

```
Spec:
  Fleet Name:  simple-game-server
  Policy:
    Type:  Webhook
    Webhook:
      Service:
        Name:       autoscaler-webhook-service
        Namespace:  default
        Path:       scale
      URL:
Status:
  Able To Scale:     true
  Current Replicas:  4
  Desired Replicas:  4
  Last Scale Time:   2018-12-22T12:53:47Z
  Scaling Limited:   false
Events:
  Type    Reason            Age   From                        Message
  ----    ------            ----  ----                        -------
  Normal  AutoScalingFleet  35s   fleetautoscaler-controller  Scaling fleet simple-game-server from 2 to 4
```

You can see that the fleet size has increased in particular case doubled to 4 gameservers (based on our custom logic in our webhook), the autoscaler having compensated for the two allocated instances.
Last Scale Time has been updated and a scaling event has been logged.

Double-check the actual number of game server instances and status by running:

```bash
 kubectl get gs -n default
```

This will get you a list of all the current `GameServers` and their `Status > State`.

```
NAME                     STATE       ADDRESS         PORT     NODE        AGE
simple-game-server-dmkp4-8pkk2   Ready       35.247.13.175   7386     minikube     5m
simple-game-server-dmkp4-b7x87   Allocated   35.247.13.175   7219     minikube     5m
simple-game-server-dmkp4-r4qtt   Allocated   35.247.13.175   7220     minikube     5m
simple-game-server-dmkp4-rsr6n   Ready       35.247.13.175   7297     minikube     5m
```

#### 7. Check downscaling using Webhook Autoscaler policy

Based on our custom webhook deployed earlier, if the fraction of allocated replicas in whole Replicas count would be less than threshold (0.3) then the fleet would scale down by scaleFactor, in our example by 2.

Note that the example webhook server has a limitation that it would not decrease fleet replica count under `minReplicasCount`, which is equal to 2.

We need to run EXIT command on one gameserver (Use IP address and port of the allocated gameserver from the previous step) in order to decrease the number of allocated gameservers in a fleet (<0.3).
```
nc -u 35.247.13.175 7220
EXIT
```

Server would be in shutdown state.
Wait about 30 seconds.
Then you should see scaling down event in the output of next command:
```bash
kubectl describe fleetautoscaler webhook-fleet-autoscaler
```

You should see these lines in events:
```
  Normal   AutoScalingFleet  11m                fleetautoscaler-controller  Scaling fleet simple-game-server from 2 to 4
  Normal   AutoScalingFleet  1m                 fleetautoscaler-controller  Scaling fleet simple-game-server from 4 to 2
```

And get gameservers command output:
```bash
kubectl get gs -n default
```

```
NAME                             STATUS      ADDRESS          PORT     NODE       AGE
simple-game-server-884fg-6q5sk   Ready       35.247.117.202   7373     minikube   5m
simple-game-server-884fg-b7l58   Allocated   35.247.117.202   7766     minikube   5m
```

#### 8. Cleanup
You can delete the autoscaler service and associated resources with the following commands.

```bash
kubectl delete -f https://raw.githubusercontent.com/googleforgames/agones/{{< release-branch >}}/examples/autoscaler-webhook/autoscaler-service.yaml
```


Removing the fleet:
```bash
kubectl delete -f https://raw.githubusercontent.com/googleforgames/agones/{{< release-branch >}}/examples/simple-game-server/fleet.yaml
```

## Chapter 2 Configuring HTTPS fleetautoscaler webhook with CA Bundle

### Objectives

Using TLS and a certificate authority (CA) bundle we can establish trusted communication between Fleetautoscaler and
an HTTPS server running the autoscaling webhook that controls the size of the fleet (Replicas count). The certificate of the
autoscaling webhook must be signed by the CA provided in fleetautoscaler yaml configuration file. Using TLS eliminates
the possibility of a man-in-the-middle attack between the fleetautoscaler and the autoscaling webhook.

#### 1. Deploy the fleet

Run a fleet in a cluster:
```bash
kubectl apply -f https://raw.githubusercontent.com/googleforgames/agones/{{< release-branch >}}/examples/simple-game-server/fleet.yaml
```

#### 2. Create X509 Root and Webhook certificates

The procedure of generating a Self-signed CA certificate is as follows:

The first step is to create the private root key:
```bash
openssl genrsa -out rootCA.key 2048
```

The next step is to self-sign this certificate:
```bash
openssl req -x509 -new -nodes -key rootCA.key -sha256 -days 1024 -out rootCA.pem
```

This will start an interactive script that will ask you for various bits of information. Fill it out as you see fit.

Every webhook that you wish to install a trusted certificate will need to go through this process. First, just like with the root CA step, you’ll need to create a private key (different from the root CA):
```bash
openssl genrsa -out webhook.key 2048
```

Next create configuration file `cert.conf` for the certificate signing request:
```none
[req]
distinguished_name = req_distinguished_name
req_extensions = v3_req
prompt = no
[req_distinguished_name]
CN = autoscaler-tls-service.default.svc
[v3_req]
keyUsage = digitalSignature
extendedKeyUsage = serverAuth
subjectAltName = @alt_names
[alt_names]
DNS.1 = autoscaler-tls-service.default.svc
```

Generate the certificate signing request, use valid hostname which in this case will be `autoscaler-tls-service.default.svc` as `Common Name (eg, fully qualified host name)` as well as `DNS.1` in the `alt_names` section of the config file.

Check the [Kubernetes documentation](https://kubernetes.io/docs/concepts/services-networking/dns-pod-service/#a-aaaa-records) to see how Services get assigned DNS entries.
```bash
openssl req -new -out webhook.csr -key webhook.key -config cert.conf
```

Once that’s done, you’ll sign the CSR, which requires the CA root key:
```bash
openssl x509 -req -in webhook.csr -CA rootCA.pem -CAkey rootCA.key -CAcreateserial -out webhook.crt -days 500 -sha256 -extfile cert.conf -extensions v3_req
```
This would generate webhook.crt certificate

Add secret which later would be mounted to autoscaler-webhook-tls pod.
```bash
kubectl create secret tls autoscalersecret --cert=webhook.crt --key=webhook.key
```

You need to put Base64-encoded string into caBundle field in your fleetautoscaler yaml configuration:
```bash
wget https://raw.githubusercontent.com/googleforgames/agones/{{< release-branch >}}/examples/webhookfleetautoscalertls.yaml
sed --in-place -e "s/\$CA_BUNDLE/$( cat ./rootCA.pem |  base64 -w0)/" webhookfleetautoscalertls.yaml
```

#### 3. Deploy a Webhook service for autoscaling

Run next command to create a service and a Webhook pod in a cluster:
```bash
kubectl apply -f https://raw.githubusercontent.com/googleforgames/agones/{{< release-branch >}}/examples/autoscaler-webhook/autoscaler-service-tls.yaml
```

To check that it is running and liveness probe is fine:
```bash
kubectl describe pod autoscaler-webhook-tls
```

Wait for the Running status results:
```
Name:               autoscaler-webhook-tls-f74c9bff7-ssrsc
Namespace:          default
...
Status:         Running
```

#### 4. Create a Fleet Autoscaler

Let's create a Fleet Autoscaler using the following command (caBundle should be set properly on Step 2):

```bash
kubectl apply -f ./webhookfleetautoscalertls.yaml
```

#### 5. See the fleet and autoscaler status.

In order to track the list of gameservers which run in your fleet you can run this command in a separate terminal tab:

```bash
 watch "kubectl get gs -n default"
```

#### 6. Allocate two Game Servers from the Fleet to trigger scale up

If you're interested in more details for game server allocation, you should consult the [Create a Game Server Fleet]({{< relref "create-fleet.md" >}}) page.
Here we only interested in triggering allocations to see the autoscaler in action.

```bash
for i in {0..1} ; do kubectl create -f https://raw.githubusercontent.com/googleforgames/agones/{{< release-branch >}}/examples/simple-game-server/gameserverallocation.yaml -o yaml ; done
```

#### 7. Check new Autoscaler and Fleet status

Now let's wait a few seconds to allow the autoscaler to detect the change in the fleet and check again its status

```bash
kubectl describe fleetautoscaler  webhook-fleetautoscaler-tls
```

The last part should look similar to this:

```
Spec:
  Fleet Name:  simple-game-server
  Policy:
    Type:  Webhook
    Webhook:
      Ca Bundle:  LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUN1RENDQWFBQ0NRQ29kcEFNbTlTd0pqQU5CZ2txaGtpRzl3MEJBUXNGQURBZU1Rc3dDUVlEVlFRR0V3SlYKVXpFUE1BMEdBMVVFQ3d3R1FXZHZibVZ6TUI0WERURTVNREV3TkRFeE5URTBORm9YRFRJeE1UQXlOREV4TlRFMApORm93SGpFTE1Ba0dBMVVFQmhNQ1ZWTXhEekFOQmdOVkJBc01Ca0ZuYjI1bGN6Q0NBU0l3RFFZSktvWklodmNOCkFRRUJCUUFEZ2dFUEFEQ0NBUW9DZ2dFQkFOQ0h5dndDOTZwZDlTdkFhMUIvRWg2ekcxeDBLS1dPaVhtNzhJcngKKzZ5WHd5YVpsMVo1cVExbUZoOThMSGVZUmQwWVgzRTJnelZ5bFpvUlUra1ZESzRUc0VzV0tNUFVpdVo0MUVrdApwbythbEN6alAyaXZzRGZaOGEvdnByL3dZZ2FrWGtWalBUaGpKUk9xTnFIdWROMjZVcUFJYnNOTVpoUkxkOVFFCnFLSjRPNmFHNVMxTVNqZFRGVHFlbHJiZitDcXNKaHltZEIzZmxGRUVvdXExSmoxS0RoQjRXWlNTbS9VSnpCNkcKNHUzY3BlQm1jTFVRR202ZlFHb2JFQSt5SlpMaEVXcXBrd3ZVZ2dCNmRzWE8xZFNIZXhhZmlDOUVUWGxVdFRhZwo1U2JOeTVoYWRWUVV3Z253U0J2djR2R0t1UUxXcWdXc0JyazB5Wll4Sk5Bb0V5RUNBd0VBQVRBTkJna3Foa2lHCjl3MEJBUXNGQUFPQ0FRRUFRMkgzaWJRcWYzQTNES2l1eGJISURkbll6TlZ2Z0dhRFpwaVZyM25ocm55dmxlNVgKR09hRm0rMjdRRjRWV29FMzZDTGhYZHpEWlM4bEpIY09YUW5KOU83Y2pPYzkxVmh1S2NmSHgwS09hU1oweVNrVAp2bEtXazlBNFdoNGE0QXFZSlc3Z3BUVHR1UFpydnc4VGsvbjFaWEZOYVdBeDd5RU5OdVdiODhoNGRBRDVaTzRzCkc5SHJIdlpuTTNXQzFBUXA0Q3laRjVyQ1I2dkVFOWRkUmlKb3IzM3pLZTRoRkJvN0JFTklZZXNzZVlxRStkcDMKK0g4TW5LODRXeDFUZ1N5Vkp5OHlMbXFpdTJ1aThjaDFIZnh0OFpjcHg3dXA2SEZLRlRsTjlBeXZUaXYxYTBYLwpEVTk1eTEwdi9oTlc0WHpuMDJHNGhrcjhzaUduSEcrUEprT3hBdz09Ci0tLS0tRU5EIENFUlRJRklDQVRFLS0tLS0K
      Service:    <nil>
      URL:        https://autoscaler-tls-service.default.svc:8000/scale
Events:
  Type    Reason            Age   From                        Message
  ----    ------            ----  ----                        -------
  Normal  AutoScalingFleet  5s   fleetautoscaler-controller  Scaling fleet simple-game-server from 2 to 4
```

You can see that the fleet size has increased in particular case doubled to 4 gameservers (based on our custom logic in our webhook), the autoscaler having compensated for the two allocated instances.
Last Scale Time has been updated and a scaling event has been logged.

Double-check the actual number of game server instances and status by running:

```bash
 kubectl get gs -n default
```

This will get you a list of all the current `GameServers` and their `Status > State`.

```
NAME                     STATE       ADDRESS         PORT      NODE      AGE
simple-game-server-njmr7-2t4nx   Ready       35.203.159.68   7330      minikube   1m
simple-game-server-njmr7-65rp6   Allocated   35.203.159.68   7294      minikube   4m
```

#### 8. Cleanup
You can delete the autoscaler service and associated resources with the following commands.

```bash
kubectl delete -f https://raw.githubusercontent.com/googleforgames/agones/{{< release-branch >}}/examples/autoscaler-webhook/autoscaler-service-tls.yaml
```

Removing x509 key secret:
```bash
kubectl delete secret autoscalersecret
```

Removing the fleet:
```bash
kubectl delete -f https://raw.githubusercontent.com/googleforgames/agones/{{< release-branch >}}/examples/simple-game-server/fleet.yaml
```

### Comments

Note that secure communication has been established and we can trust that communication between the fleetautoscaler and
the autoscaling webhook. If you need to run the autoscaling webhook outside of the Kubernetes cluster, you can use
another root certificate authority as long as you put it into the caBundle parameter in fleetautoscaler configuration
(in pem format, base64-encoded).

## Troubleshooting Guide

If you run into problems with the configuration of your fleetautoscaler and webhook service the easiest way to debug
them is to run:
```bash
kubectl describe fleetautoscaler <FleetAutoScalerName>
```
and inspect the events at the bottom of the output.

### Common error messages.

If you have configured the wrong service Path for the FleetAutoscaler you will see a message like
```
Error calculating desired fleet size on FleetAutoscaler simple-fleet-r7fdv-autoscaler. Error: bad status code 404 from the server: https://autoscaler-tls-service.default.svc:8000/scale
```

If you are using a hostname other than `autoscaler-tls-service.default.svc` as the
`Common Name (eg, fully qualified host name)` when creating a certificate using `openssl` tool you will see a
message like
```
Post https://autoscaler-tls-service.default.svc:8000/scale: x509: certificate is not valid for any names, but wanted to match autoscaler-tls-service.default.svc
```

If you see errors like the following in `autoscaler-webhook-tls` pod logs:
```
http: TLS handshake error from 10.48.3.125:33374: remote error: tls: bad certificate
```
Then there could be an issue with your `./rootCA.pem`.

You can repeat the process from step 2, in order to fix your certificates setup.

## Next Steps

Read the advanced [Scheduling and Autoscaling]({{< relref "../Advanced/scheduling-and-autoscaling.md" >}}) guide, for more details on autoscaling.

If you want to use your own GameServer container make sure you have properly integrated the [Agones SDK]({{< relref "../Guides/Client SDKs/_index.md" >}}).
