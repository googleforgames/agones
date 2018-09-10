# Tutorial Create an Allocator Service

This tutorial describes how to interact programmatically with the [Agones API](https://godoc.org/agones.dev/agones/pkg/client/clientset/versioned/typed/stable/v1alpha1).  To do this, we will implement a [Service](https://kubernetes.io/docs/concepts/services-networking/service/) which allocates a Game Server on demand by calling the Create() method of the FleetAllocationInterface.  After creating the fleet allocation, we will return the JSON encoded GameServerStatus of the allocated GameServer.

The type of service we will be learning about could be used by a game client to connect directly to a dedicated Game Server, as part of a larger system, such as a matchmaker service, or in conjunction with a database of level transition data.  We will be using the service as a vehicle with which to execute the API calls found in our main.go file.

## Objectives
- Create a secure allocator service
- Deploy the service to [GKE](https://cloud.google.com/kubernetes-engine/)
- Allocate a Game Server from a Fleet using the Agones API

## Prerequisites
1. [Docker](https://www.docker.com/get-started/)
2. Agones installed on GKE, running a simple-udp fleet
3. kubectl properly configured
4. A local copy of the [allocator service](https://github.com/GoogleCloudPlatform/agones/tree/master/examples/allocator-service)
5. A repository for Docker images, such as [Docker Hub](https://hub.docker.com/) or [GC Container Registry](https://cloud.google.com/container-registry/)


>NOTE: Agones requires Kubernetes versions 1.9 with role-based access controls (RBAC) and MutatingAdmissionWebhook features activated. To check your version, enter `kubectl version`.

To install on GKE, follow the install instructions (if you haven't already) at
[Setting up a Google Kubernetes Engine (GKE) cluster](../install/README.md#setting-up-a-google-kubernetes-engine-gke-cluster). Also complete the "Enabling creation of RBAC resources" and "Installing Agones" sets of instructions on the same page.

While not required, you may wish to review the [Create a Game Server](./create_gameserver.md), [Create Game Server Fleet](./create_fleet.md), and/or [Edit a Game Server](./edit_first_game_server.md) quickstarts.


### 1. Build and Push the Service
Change directories to your local agones/examples/allocator-service directory and build a new docker image.  The multi-stage Dockerfile will pull down all the dependencies for you and build the executable.  For example, where USER is your username, REPO is your repository, and TAG is your tag:
```
docker build -t [USER]/allocator-service:[TAG] .
```

Push it to your repository:
```
docker push [USER]/allocator-service:[TAG]
```

Edit allocator-service.yaml to point to your new image:
```
containers:
- name: fleet-allocator
  image: [REPO]/[USER]/allocator-service:[TAG]
  imagePullPolicy: Always
```


### 2. Create Firewall Rules

Let's making some [firewall](https://kubernetes.io/docs/tasks/access-application-cluster/configure-cloud-provider-firewall/) rules that will be used by kubernetes health checks and the ingress which we will create shortly.

First, we will make one for the HTTPS health checks that will be sent to our service by running:
```
gcloud compute firewall-rules create fleet-allocator-healthcheck \
  --allow tcp \
  --source-ranges 130.211.0.0/22,35.191.0.0/16 \
  --target-tags fleet-allocator \
  --description "Firewall to allow health check of fleet allocator service"
```

The output should be something like:
```
Creating firewall...done.                                            
NAME                         NETWORK  DIRECTION  PRIORITY  ALLOW  DENY  DISABLED
fleet-allocator-healthcheck  default  INGRESS    1000      tcp          False
```

Create a firewall rule for nodePort traffic on the range of ports used by Ingress services of type nodePort.  We are using nobdePort becuase it supports TLS.
```
gcloud compute firewall-rules create nodeport-rule \
  --allow=tcp:30000-32767 \
  --target-tags fleet-allocator \
  --description "Firewall to allow nodePort traffic of fleet allocator service"
```

The output should be something like:
```
Creating firewall...done.                                            
NAME           NETWORK  DIRECTION  PRIORITY  ALLOW            DENY  DISABLED
nodeport-rule  default  INGRESS    1000      tcp:30000-32767        False
```


### 3. Make It Secure
Let's keep security in mind from the beginning by creating a certificate, key and secret for the allocator service, and another set for the web server.

Pick a more permanent location for the files if you like - /tmp may be purged depending on your operating system.

Create a public private key pair for the allocator service:
```
openssl req -x509 -nodes -days 365 -newkey rsa:2048 -keyout /tmp/allocsvc.key -out  /tmp/allocsvc.crt -subj "/CN=my-allocator/O=my-allocator"
```

The output should be something like:
```
Generating a 2048 bit RSA private key
....................................................+++
......................+++
writing new private key to '/tmp/allocsvc.key'
-----
```

Create a public private key pair that will be bound to the pod and used by the web server :
```
openssl req -x509 -nodes -days 365 -newkey rsa:2048 -keyout /tmp/tls.key -out  /tmp/tls.crt -subj "/CN=my-allocator-w3/O=my-allocator-w3"
```

The output should be something like:
```
Generating a 2048 bit RSA private key
....................................................+++
......................+++
writing new private key to '/tmp/tls.key'
-----
```


### 4. Create Kubernetes Secrets

The allocatorsecret will allow the service to use TLS for connections with workers.

Create the [secret](https://kubernetes.io/docs/concepts/configuration/secret/) by running this command:
```
kubectl create secret tls allocatorsecret --cert=/tmp/allocsvc.crt --key=/tmp/allocsvc.key
```

The output should be something like:
```
secret "allocatorsecret" created
```

The allocatorw3secret will let data be served by the webserver over https.

Create the secret by running this command:
```
kubectl create secret tls allocatorw3secret --cert=/tmp/tls.crt --key=/tmp/tls.key
```
The output should be something like:
```
secret "allocatorw3secret" created
```

See that the secrets exist by running:
```
kubectl get secrets
```

The output should contain the secrets:
```
NAME                     TYPE                                  DATA      AGE
...
allocatorsecret          kubernetes.io/tls                     2         29s
allocatorw3secret        kubernetes.io/tls                     2         15s
...
```


### 5. Create the Service Account
This service will interact with Agones via the Agones API by using a [service account](https://kubernetes.io/docs/tasks/configure-pod-container/configure-service-account/) named fleet-allocator.  Specifically, the fleet-allocator service account is granted permissions to perform create operations against FleetAllocation objects, and get operations against Fleet objects.

Create the service account by changing directories to your local agones/examples/allocator-service directory and running this command:
```
kubectl create -f service-account.yaml
```

The output should look like this:
```
role "fleet-allocator" created
serviceaccount "fleet-allocator" created
rolebinding "fleet-allocator" created
```


### 6. Define and Deploy the Service
The service definition defines a [nodePort](https://kubernetes.io/docs/concepts/services-networking/service/#nodeport) service which uses https, and sets up ports and names.  The deployment describes the number of replicas we would like, which account to use, which image to use, and defines a health check.

Define and Deploy the service by running this command:
```
kubectl create -f allocator-service.yaml
```

The output should look like this:
```
service "fleet-allocator-backend" created
deployment "fleet-allocator" created
```


### 7. Deploy the Ingress Resource
This [Ingress](https://kubernetes.io/docs/concepts/services-networking/ingress/) directs traffic to the allocator service using an ephemeral IP address.  The allocator service pod needs to exist and the readiness probe should be passing health checks before the ingress is created.

Deploy the Ingress with this command:
```
kubectl apply -f allocator-ingress.yaml
```

The output should look like this:
```
ingress "fleet-allocator-ingress" created
```


### 8. Retrieve the Ephemeral Public IP Address
After deployment, it will take about a minute for the IP address to be present, and up to 10 minutes before it can start returning data.

Run this command to get the IP address:
```
kubectl get ingress fleet-allocator-ingress
```

The output should look something like this:
```
NAME                      HOSTS     ADDRESS          PORTS     AGE
fleet-allocator-ingress   *         35.186.225.103   80, 443   1m
```

To learn more about the status of the ingress, run:
```
kubectl get ingress fleet-allocator-ingress -o yaml
```

When the output shows the ingress.kubernetes.io/backends as 'HEALTHY' rather than 'UNHEALTHY' or 'UNKOWN' it is probably ready.
```
apiVersion: extensions/v1beta1
kind: Ingress
metadata:
  annotations:
    ingress.kubernetes.io/backends: '{"k8s-be-30021--7e98a70481f48a13":"HEALTHY"}'
    ingress.kubernetes.io/https-forwarding-rule: k8s-fws-default-fleet-allocator-ingress--7e98a70481f48a13
    ingress.kubernetes.io/https-target-proxy: k8s-tps-default-fleet-allocator-ingress--7e98a70481f48a13
    ingress.kubernetes.io/ssl-cert: k8s-ssl-1ab99915a1f6b5f1-b2a9924cee73d20a--7e98a70481f48a13
    ingress.kubernetes.io/url-map: k8s-um-default-fleet-allocator-ingress--7e98a70481f48a13
    kubectl.kubernetes.io/last-applied-configuration: |
      {"apiVersion":"extensions/v1beta1","kind":"Ingress","metadata":{"annotations":{"kubernetes.io/ingress.allow-http":"false","kubernetes.io/ingress.class":"gce"},"labels":{"app":"fleet-allocator"},"name":"fleet-allocator-ingress","namespace":"default"},"spec":{"backend":{"serviceName":"fleet-allocator-backend","servicePort":8000},"tls":[{"secretName":"allocatorsecret"}]}}
    kubernetes.io/ingress.allow-http: "false"
    kubernetes.io/ingress.class: gce
  creationTimestamp: 2018-09-23T19:13:36Z
  generation: 1
  labels:
    app: fleet-allocator
  name: fleet-allocator-ingress
  namespace: default
  resourceVersion: "4086"
  selfLink: /apis/extensions/v1beta1/namespaces/default/ingresses/fleet-allocator-ingress
  uid: c5a149b3-bf64-11e8-8a6e-42010a8e013f
spec:
  backend:
    serviceName: fleet-allocator-backend
    servicePort: 8000
  tls:
  - secretName: allocatorsecret
status:
  loadBalancer:
    ingress:
    - ip: 35.186.225.103
```

### 9. Check Game Servers
Let's make sure that we have one or more Game Servers in a ready state by running this command:
```
kubectl get gs -o=custom-columns=NAME:.metadata.name,STATUS:.status.state,IP:.status.address,PORT:.status.ports
```

For a fleet of 2 replicas, you should see 2 Game Servers with a Status of Ready:
```
NAME                     STATUS    IP               PORT
simple-udp-s2snf-765bc   Ready     35.231.204.26    [map[name:default port:7260]]
simple-udp-s2snf-vf6l8   Ready     35.196.162.169   [map[name:default port:7591]]
```

If there is no fleet, please review [Create Game Server Fleet](../docs/create_fleet.md).


### 10. Allocate a Game Server
Now that the ingress has been created, let's allocate a Game Server by passing in our user and key to the /address endpoint.  This will call the allocate() function in main.go, which will return a JSON string of the GameServerStatus of an allocated GameServer, or an error.  The service uses Basic Auth to provide some security as to who can allocate GameServer resources, and the generated key is in main.go, in the function basicAuth().  Read the comments and code in main.go for a more detailed explanation of each function and method.

Allocate a Game Server by running this command:
```
curl -k -u v1GameClientKey:EAEC945C371B2EC361DE399C2F11E https://[the.ip.address]/address
```

The output should show the JSON of the GameServerStatus, similar to this:
```
{"status":{"state":"Allocated","ports":[{"name":"default","port":7260}],"address":"35.231.204.26","nodeName":"gke-agones-simple-udp-cluste-default-pool-e03a9bde-000f"}}
```

You may need to wait a few moments longer if the output has ssl errors like this:
```
curl: (35) error:14094410:SSL routines:ssl3_read_bytes:sslv3 alert handshake failure
```

Check the Game Servers again, and notice the Allocated Status.  You should see something like this:
```
NAME                     STATUS      IP               PORT
simple-udp-s2snf-765bc   Allocated   35.231.204.26    [map[name:default port:7260]]
simple-udp-s2snf-vf6l8   Ready       35.196.162.169   [map[name:default port:7591]]
```

Congratulations, your call to the API has allocated a Game Server from your simple-udp Fleet!


### 11. Cleanup
You can delete the allocator service and associated resources with the following commands.

Delete the Ingress
```
kubectl delete ingress fleet-allocator-ingress
```

Delete the Service
```
kubectl delete -f allocator-service.yaml
```

Delete the Service Account
```
kubectl delete -f service-account.yaml
```

Delete the health check and firewall rules
```
gcloud compute health-checks delete fleet-allocator-healthcheck
gcloud compute firewall-rules delete fleet-allocator-healthcheck
gcloud compute health-checks delete nodeport-rule
gcloud compute firewall-rules delete nodeport-rule
```


### Next Steps
- Customize the service by changing the constants and service key in main.go
- Make the [IP Address Permanent](https://cloud.google.com/kubernetes-engine/docs/tutorials/configuring-domain-name-static-ip)
- Create an A record that points to your permanent IP address
- [Create a Fleet Autoscaler](./create_fleetautoscaler.md)
