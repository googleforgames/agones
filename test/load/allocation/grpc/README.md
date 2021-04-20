# Load Test for gRPC allocation service

This load tests aims to validate performance of the gRPC allocation service.

## Kubernetes Cluster Setup

For the load test you can follow the regular Kubernetes and Agones setup. In order to test the allocation performance in isolation, we let Agones to get all
the game servers to Ready state before starting a test.
Here are the few important things:
- If you are running in GCP, use a regional cluster instead of a zonal cluster to ensure high availability of the cluster control plane
- Use a dedicated node pool for the Agones controllers with multiple CPUs per node, e.g. `e2-standard-4'
- In the default node pool (where the Game Server pods are created), 75 nodes are required to make sure there are enough nodes available for all game servers to move into the ready state. When using a regional cluster, with three zones with the region, that will require a configuration of 25 nodes per zone.

## Fleet Setting

We used the sample [fleet configuration](./fleet.yaml) with some minor modifications. We updated the `replicas` to 4000.
Also we set the `automaticShutdownDelayMin` parameter to 10 so simple-game-server game servers shutdown after 10
minutes (see below).
This helps to easily re-run the test without having to delete the game servers and allows to run tests continously.

```yaml
apiVersion: "agones.dev/v1"
kind: Fleet
 ...
 spec:
  # the number of GameServers to keep Ready
  replicas: 4000
  ...
        # The GameServer's Pod template
        template:
            spec:
                containers:
                - args:
                  # We setup the simple-game-server server to shutdown 10 mins after allocation
                  - -automaticShutdownDelayMin=10
                  image: gcr.io/agones-images/simple-game-server:0.3
                  name: simple-game-server
  ...
```

## Configuring the Allocator Service

The allocator service uses gRPC. In order to be able to call the service, TLS and mTLS has to be setup.
For more information visit [Allocator Service](https://agones.dev/site/docs/advanced/allocator-service/).

## Running the test

You can use the provided runAllocation.sh script by providing two parameters: 
- number of clients (to do parallel allocations)
- number of allocations for client

For making 4000 allocations calls, you can provide 40 and 100

```
./runAllocation.sh 40 100
```

Script will print out the start and end date/time:
```
started: 2020-10-22 23:33:25.828724579 -0700 PDT m=+0.005921014
finished: 2020-10-22 23:34:18.381396416 -0700 PDT m=+52.558592912
```

If some errors occurred, the error message will be printed:
```
started: 2020-10-22 22:16:47.322731849 -0700 PDT m=+0.002953843
(failed(client=3,allocation=43): rpc error: code = Unknown desc = error updating allocated gameserver: Operation cannot be fulfilled on gameservers.agones.dev "simple-game-server-mlljx-g9crp": the object has been modified; please apply your changes to the latest version and try again
(failed(client=2,allocation=47): rpc error: code = Unknown desc = error updating allocated gameserver: Operation cannot be fulfilled on gameservers.agones.dev "simple-game-server-mlljx-rxflv": the object has been modified; please apply your changes to the latest version and try again
(failed(client=7,allocation=45): rpc error: code = Unknown desc = error updating allocated gameserver: Operation cannot be fulfilled on gameservers.agones.dev "simple-game-server-mlljx-x4khw": the object has been modified; please apply your changes to the latest version and try again
finished: 2020-10-22 22:17:18.822039094 -0700 PDT m=+31.502261092
```

You can use environment variables overwrite defaults. To run only a single run of tests, you can use:
```
TESTRUNSCOUNT=1 ./runAllocation.sh 40 10
```
