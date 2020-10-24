# Load Test for gRPC allocation service

This load tests aims to validate performance of the gRPC allocation service.

## Kubernetes Cluster Setup

For the load test you can follow the regular Kubernetes and Agones setup. In order to test the the allocation performance in isolation, we let the Agones to actuate all
the game servers beefor starting the test.
Here are the few important things:
- Install Kubernetes in a region not zone to avoid being impacted by the rollouts
- Consider using multiple cpu VMs for the agones-system node pool. We used n1-4 type VMs 
- In the default-node pool (where the Game Server pods are created), make sure there are enough nodes available so Agones can actuate the game servers. We set the clustre count to 25 pre zonee (total of 75 per region)

## Configuring the Allocator Service

The allocator service uses gRPC. In order to be able to call the service, TLS and optionally mTLS has to be setup.
For more information visit [agones-allocator]({{< relref "https://agones.dev/site/docs/advanced/allocator-service/">}}).

## Fleet Setting

We used the following fleet configuration during out testing. It creates 4000 simple-udp game servers. 
Also uaing the automaticShutdownDelayMin parameter to 10, simple-udp game servers shutdown after 10 minutes.
This helps to easily re-run the test without requiring to delete the game servers. 

```yaml
apiVersion: "agones.dev/v1"
kind: Fleet
metadata:
  name: fleet-example
spec:
  # the number of GameServers to keep Ready or Allocated in this Fleet
  replicas: 4000
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxSurge: 25%
      maxUnavailable: 25%
  template:
    metadata:
      labels:
        foo: bar
    # GameServer specification
    spec:
        health: {}
        ports:
        - containerPort: 7654
          name: default
        sdkServer: {}
        # The GameServer's Pod template
        template:
            spec:
                containers:
                - args:
                  # We setup the simple-udp server to shutdown 10 mins after allocation 
                  - -automaticShutdownDelayMin=10
                  image: gcr.io/agones-images/udp-server:0.21
                  name: simple-udp
                  resources:
                  limits:
                    cpu: 20m
                    memory: 32Mi
                  requests:
                    cpu: 20m
                    memory: 32Mi
```

## Configuring the Allocator Service

The allocator service uses gRPC. In order to be able to call the service, TLS and optionally mTLS has to be setup.
For more information visit [agones-allocator]({{< relref "https://agones.dev/site/docs/advanced/allocator-service/">}}).

## Running the test

You can use the provided runAllocation.sh file. Before you run, you need to update the values in runAllocation.sh

```
TESTRUNSCOUNT=3
NAMESPACE=default      # Namescape of the fleet
EXTERNAL_IP=<IP_ADRESSS_TO_THE_ALLOCATOR_SERVICES_LOAD_BALANCER>
KEY_FILE=client.key    # Path to mTLS client key file 
CERT_FILE=client.crt   # Path to mTLS client cert file
TLS_CA_FILE=ca.crt     # Path to TLS CA file
```
Once you update these values, you can run the run.sh script by providing two parameters: 
- number of clients (to do parallel allocations)
- numbre of allocations for client

For making 4000 allocations call, you can provide 40 and 100

```
./run.sh 40 100
```

Script will print out the start and end date/time:
```
started: 2020-10-22 23:33:25.828724579 -0700 PDT m=+0.005921014
finished: 2020-10-22 23:34:18.381396416 -0700 PDT m=+52.558592912
```

If some errors occurred, the error message will be printed:
```
started: 2020-10-22 22:16:47.322731849 -0700 PDT m=+0.002953843
(failed(client=3,allocation=43): rpc error: code = Unknown desc = error updating allocated gameserver: Operation cannot be fulfilled on gameservers.agones.dev "simple-udp-mlljx-g9crp": the object has been modified; please apply your changes to the latest version and try again
(failed(client=2,allocation=47): rpc error: code = Unknown desc = error updating allocated gameserver: Operation cannot be fulfilled on gameservers.agones.dev "simple-udp-mlljx-rxflv": the object has been modified; please apply your changes to the latest version and try again
(failed(client=7,allocation=45): rpc error: code = Unknown desc = error updating allocated gameserver: Operation cannot be fulfilled on gameservers.agones.dev "simple-udp-mlljx-x4khw": the object has been modified; please apply your changes to the latest version and try again
finished: 2020-10-22 22:17:18.822039094 -0700 PDT m=+31.502261092
```
