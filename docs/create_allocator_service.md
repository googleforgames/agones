# Quickstart Create an Allocator Service

This quickstart describes one way to implement a service which will allocate a Game Server on demand using the Agones API.  This allows a game client or matchmaker service to call the allocator service in order to allocate a Game Server and retrieve its IP address and port number. The game client can then connect directly to the Game Server, so that the minimized latency that is in the Agones design is preserved.

## Objectives
- Create a secure allocator service
- Deploy the service to GKE
- Allocate a Game Server from a Fleet using the service


:warning: This quickstart requires Agones installed with agones.image.tag=0.5.0-03f4866 or newer :warning:

## Prerequisites
1. A [Go](https://golang.org/dl/) environment
2. [Docker](https://www.docker.com/get-started/)
3. Agones installed on GKE, with a simple-udp fleet
4. kubectl properly configured
5. A clone or fork of the [Agones repo](https://github.com/GoogleCloudPlatform/agones)


>NOTE: Agones required Kubernetes versions 1.9 with role-based access controls (RBAC) and MutatingAdmissionWebhook features activated. To check your version, enter `kubectl version`.

To install on GKE, follow the install instructions (if you haven't already) at
[Setting up a Google Kubernetes Engine (GKE) cluster](../install/README.md#setting-up-a-google-kubernetes-engine-gke-cluster). Also complete the "Enabling creation of RBAC resources" and "Installing Agones" sets of instructions on the same page.

While not required, you may wish to review the [Create a Game Server](../docs/create_gameserver.md), [Create Game Server Fleet](../docs/create_fleet.md), and/or [Edit a Game Server](../docs/edit_first_game_server.md) quickstarts.


### 1. Create Firewall Rules

Let's start off by making some firewall rules that will be used by kubernetes health checks and the ingress which we will create shortly.  

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


### 2. Make It Secure
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


### 3. Create Kubernetes Secrets

The allocatorsecret will allow the service to use TLS for connections with workers.

Create the secret by running this command:
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


### 4. Create the Service Account  
This service will interact with the Agones via the Agones API by using a service account named fleet-allocator.  Specifically, the fleet-allocator service account is granted permissions to perform create operations against FleetAllocation objects.  The complete service account, role, and role binding definitions are in agones/examples/allocator-service/service-account.yaml.

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


### 5. Define and Deploy the Service
The service definition defines a nodePort service which uses https, and sets up ports and names.  The deployment describes the number of replicas we would like, which account to use, which image to use, and defines a health check.  The complete service definition is in agones/examples/allocator-service/allocator-service.yaml.

Define and Deploy the service by running this command:
```
kubectl create -f allocator-service.yaml
```

The output should look like this:
```
service "fleet-allocator-backend" created
deployment "fleet-allocator" created
```


### 6. Define an Ingress Resource
This Ingress directs traffic to the allocator service using an ephemeral IP address.  Notice that we are specifying the ingress class as gce, which is specific to the Google Cloud Platform.  We also define which secret to use for TLS here.  

This is in the file agones/examples/allocator-service/allocator-ingress.yaml
```
apiVersion: extensions/v1beta1
kind: Ingress
metadata:
  name: fleet-allocator-ingress
  namespace: default
  annotations:
    kubernetes.io/ingress.class: "gce"
    kubernetes.io/ingress.allow-http: "false"
spec:
  tls:
  - secretName: allocatorsecret
  backend:
    serviceName: fleet-allocator-backend
    servicePort: 8000

```


### 7. Deploy the Ingress Resource  
The allocator service pod needs to exist and the readiness probe should be passing health checks before the ingress is created.

Deploy the Ingress with this command:
```
kubectl apply -f allocator-ingress.yaml
```

The output should look like this:
```
ingress "fleet-allocator-ingress" created
```


### 8. Retrieve the Ephemeral Public IP Address
After deployment, it will take about a minute for the IP address to be present, and about 5 or 6 minutes before it can start returning data.

Run this command to get the IP address:
```
kubectl get ingress fleet-allocator-ingress
```

The output should look something like this:
```
NAME                      HOSTS     ADDRESS         PORTS     AGE
fleet-allocator-ingress   *         130.211.42.92   80, 443   1m
```

To learn more about the status of the ingress, run:
```
kubectl get ingress fleet-allocator-ingress -o yaml
```

When the output shows the ingress.kubernetes.io/backends as 'HEALTHY' rather than 'UNHEALTHY' or 'UNKOWN' it is ready.
```
apiVersion: extensions/v1beta1
kind: Ingress
metadata:
  annotations:
    ingress.kubernetes.io/backends: '{"k8s-be-30055--e34f41617775a061":"HEALTHY"}'
    ingress.kubernetes.io/https-forwarding-rule: k8s-fws-default-fleet-allocator-ingress--e34f41617775a061
    ingress.kubernetes.io/https-target-proxy: k8s-tps-default-fleet-allocator-ingress--e34f41617775a061
    ingress.kubernetes.io/ssl-cert: k8s-ssl-e34f98d0d23b2aca-99c16222402fe1c4--e34f41617775a061
    ingress.kubernetes.io/url-map: k8s-um-default-fleet-allocator-ingress--e34f41617775a061
    kubectl.kubernetes.io/last-applied-configuration: |
      {"apiVersion":"extensions/v1beta1","kind":"Ingress","metadata":{"annotations":{"kubernetes.io/ingress.allow-http":"false","kubernetes.io/ingress.class":"gce"},"name":"fleet-allocator-ingress","namespace":"default"},"spec":{"backend":{"serviceName":"fleet-allocator-backend","servicePort":8000},"tls":[{"secretName":"allocatorsecret"}]}}
    kubernetes.io/ingress.allow-http: "false"
    kubernetes.io/ingress.class: gce
  creationTimestamp: 2018-09-04T20:00:11Z
  generation: 1
  name: fleet-allocator-ingress
  namespace: default
  resourceVersion: "4127"
  selfLink: /apis/extensions/v1beta1/namespaces/default/ingresses/fleet-allocator-ingress
  uid: 218e0102-b07d-11e8-b86f-42010a8e0072
spec:
  backend:
    serviceName: fleet-allocator-backend
    servicePort: 8000
  tls:
  - secretName: allocatorsecret
status:
  loadBalancer:
    ingress:
    - ip: 130.211.42.92

```


### 9. Check the /healthz Endpoint.
Inside of examples/allocator-service/main.go, we set up an endpoint /healthz so that kubernetes can determine the health status of the pod.  Let's hit that endpoint now from outside the cluster to see if it returns healthy or not, and to see if the ingress is working.

Since the cert we created earlier is a self signed cert, ignoring insecure warning by adding the -k flag might be necessary:
```
curl -k https://the.ip.address/healthz
```

The output should look like this:
```
Healthy
```

You may need to wait a few moments longer if the output looks like this:
```
curl: (35) error:14094410:SSL routines:ssl3_read_bytes:sslv3 alert handshake failure
```


### 10. Check Game Servers
Let's make sure that we have one or more Game Servers in a ready state by running this command:
```
kubectl get gs -o=custom-columns=NAME:.metadata.name,STATUS:.status.state,IP:.status.address,PORT:.status.ports
```

For a fleet of 2 replicas, you should see 2 Game Servers with a Status of Ready:
```
NAME                     STATUS    IP              PORT
simple-udp-thbvz-mqntp   Ready     35.196.249.83   [map[name:default port:7374]]
simple-udp-thbvz-smchl   Ready     35.229.96.214   [map[name:default port:7016]]
```

If there is no fleet, please review [Create Game Server Fleet](../docs/create_fleet.md).


### 11. Allocate a Game Server
The service uses Basic Auth to provide some security as to who can allocate gameserver resources.  A generated key is in main.go that you can change.

Now that the ingress has been created and we know it is healthy, let's allocate a Game Server by passing in our user and key to the /gameclient/address endpoint.

Allocate a Game Server and retrieve its IP address and port by running this command:
```
curl -k -u v1GameClientKey:EAEC945C371B2EC361DE399C2F11E https://[the.ip.address]/gameclient/address
```

The output should show an IP address and port, similar to this:
```
{"address":"35.229.96.214","port":7016}
```

Check the Game Servers again, and notice the Allocated Status.  You should see something like this:
```
NAME                     STATUS      IP              PORT
simple-udp-thbvz-mqntp   Ready       35.196.249.83   [map[name:default port:7374]]
simple-udp-thbvz-smchl   Allocated   35.229.96.214   [map[name:default port:7016]]
```


### 12. Make the IP Address Permanent
Since the ephemeral IP address previously created will only live for as long as the service lives, allocating a permanent IP address is necessary for longer term reliability.

:warning: This step may cause additional billing charges. :warning:

Reserve a static external IP address named allocator-static-ip by running:
```
gcloud compute addresses create allocator-static-ip --global
```

Configure the existing Ingress to use the reserved IP address by adding an annotation to allocator-ingress.yaml:
```
  annotations:
    kubernetes.io/ingress.global-static-ip-name: "allocator-static-ip"
```

Apply the change by running:
```
kubectl apply -f allocator-ingress.yaml
```


### 13. Create an A Record
Setting up a DNS A record that points to the allocator service's permanent IP address is a flexible way to call it from other services or clients.

This step requires that you have a registered domain available.  Use the permanent IP address created previously when creating a new A record with your registrar, and follow your registrars documentation on adding an A record.  When finished, a subdomain, such as allocator-service, should exist.

Now https://allocator-service.yourdomain.ext should return the IP address and port of the allocated Game Server.


### 14. Customize
Edit main.go to reflect your kubernetes configuration by changing which namespace is used and which fleet GameServers are being allocated from.  Also create a unique key or password that will be used when calling this allocator service.


Change some constants to reflect your environment.  These are the current values:
```
const namespace    = "default"
const fleetname    = "simple-udp"
const generatename = "simple-udp-"
```

Add a unique password to the authorized group in main.go, for example:
```
// Group that can allocate game servers
authorized := router.Group("/gameclient", gin.BasicAuth(gin.Accounts{
  "v1GameClientKey": "92d5cca8-1ef0-4ee3-b10c-3a3ea802bff7",
}))
```

Since the Docker image is using Alpine Linux, the "go build" command has to include few more environment variables.

Let's get our dependencies and build with these commands:
```
go get net/http
go get agones.dev/agones/pkg/client/clientset/versioned
go get agones.dev/agones/pkg/apis/stable/v1alpha1
go get agones.dev/agones/pkg/util/runtime
go get github.com/gin-gonic/gin
go get k8s.io/api/core/v1
go get k8s.io/apimachinery/pkg/apis/meta/v1
go get k8s.io/client-go/rest
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o bin/server -a -v main.go
```

Build a new docker image, where REPO is your repository and TAG is your tag:
```
docker build -t [REPO]/allocator-service:[TAG] .
```

Push it to whatever repo you use:
```
docker push [REPO]/allocator-service:[TAG]
```

Edit allocator-service.yaml to point to the new image:
```
containers:
- name: fleet-allocator
  image: your.repo/user/allocator-service:tag
  imagePullPolicy: Always
```

Edit your game client, matchmaker, or other dependent service to hit the subdomain.domain.ext/gameclient/address endpoint to retrieve the IP address and port of the newly allocated Game Server.

Consider adding some code to the dedicated game server that checks how many connected users there are, or how many players are in the world.  If it returns 0, call the Agones SDK /shutdown endpoint to destroy the stale Pod.


### 15. Cleanup
You can delete the allocator service and associated resources with the following commands.

Delete the Ingress
```
kubectl delete ingress fleet-allocator-ingress
```

Delete the Reserved IP address
```
gcloud compute addresses delete allocator-static-ip --global
```

Delete the Service
```
kubectl delete -f allocator-service.yaml
```

Delete the Service Account
```
kubectl delete -f service-account.yaml
```


### 16. Related Documentation

The [Gin Web Framework](https://github.com/gin-gonic/gin/)

The Kubernetes documentation on [Services](https://kubernetes.io/docs/concepts/services-networking/service/)

The Kubernetes documentation on [Connecting Applications with Services](https://kubernetes.io/docs/concepts/services-networking/connect-applications-service/)
