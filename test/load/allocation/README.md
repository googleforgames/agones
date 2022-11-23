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
Also we set the `automaticShutdownDelaySec` parameter to 10 so simple-game-server game servers shutdown after 10
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
                  - -automaticShutdownDelaySec=600
                  image: us-docker.pkg.dev/agones-images/examples/simple-game-server:0.14
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


# Running Scenario tests

The scenario test allows you to generate a variable number of allocations to
your cluster over time, simulating a game where clients arrive in an unsteady
pattern. The game servers used in the test are configured to shutdown after
being allocated, simulating the GameServer churn that is expected during
normal game play.

## Kubernetes Cluster Setup

For the scenario test to achieve high throughput, you can create multiple groups
of nodes in your cluster. During testing (on GKE), we created a node pool for
the Kubernetes system components (such as the metrics server and dns servers), a
node pool for the Agones system components (as recommended in the installation
guide), and a node pool for the game servers.

On GKE, to restrict the Kubernetes system components to their own set of nodes,
you can create a node pool with the taint
`components.gke.io/gke-managed-components=true:NoExecute`.

To prevent the Kubernetes system components from running on the game servers
node pool, that node pool was created with the taint
`scenario-test.io/game-servers=true:NoExecute`
and the Agones system node pool used the normal taint
`agones.dev/agones-system=true:NoExecute`.

In addition, the GKE cluster was configured as a regional cluster to ensure high
availability of the cluster control plane.

The following commands were used to construct a cluster for testing:

```bash
gcloud container clusters create scenario-test --cluster-version=1.21 \
  --tags=game-server --scopes=gke-default --num-nodes=2 \
  --no-enable-autoupgrade --machine-type=n2-standard-2 \
  --region=us-west1 --enable-ip-alias --cluster-ipv4-cidr 10.0.0.0/10

gcloud container node-pools create kube-system --cluster=scenario-test \
  --no-enable-autoupgrade \
  --node-taints components.gke.io/gke-managed-components=true:NoExecute \
  --num-nodes=1 --machine-type=n2-standard-16 --region us-west1

gcloud container node-pools create agones-system --cluster=scenario-test \
  --no-enable-autoupgrade --node-taints agones.dev/agones-system=true:NoExecute \
  --node-labels agones.dev/agones-system=true --num-nodes=1 \
  --machine-type=n2-standard-16 --region us-west1

gcloud container node-pools create game-servers --cluster=scenario-test \
  --node-taints scenario-test.io/game-servers=true:NoExecute --num-nodes=1 \
  --machine-type n2-standard-2 --no-enable-autoupgrade \
  --region us-west1 --tags=game-server --scopes=gke-default \
  --enable-autoscaling --max-nodes=300 --min-nodes=175
```

## Agones Modifications

For the scenario tests, we modified the Agones installation in a number of ways.

First, we made sure that the Agones pods would _only_ run in the Agones node
pool by changing the node affinity in the deployments for the controller,
allocator service, and ping service to
`requiredDuringSchedulingIgnoredDuringExecution`.

We also increased the resources for the controller and allocator service pods,
and made sure to specify both requests and limits to ensure that the pods were
given the highest quality of service.

These configuration changes are captured in
[scenario-values.yaml](scenario-values.yaml) and can be applied during
installation using helm:

```bash
helm install my-release --namespace agones-system -f scenario-values.yaml agones/agones --create-namespace
```

Alternatively, these changes can be applied to an existing Agones installation
by running [`./configure-agones.sh`](configure-agones.sh).

## Fleet Setting

We used the following fleet configuration:

```
apiVersion: "agones.dev/v1"
kind: Fleet
metadata:
  name: scenario-test
spec:
  replicas: 10
  template:
    metadata:
      labels:
        gameName: simple-game-server
    spec:
      ports:
      - containerPort: 7654
        name: default
      health:
        initialDelaySeconds: 30
        periodSeconds: 60
      template:
        spec:
          tolerations:
          - effect: NoExecute
            key: scenario-test.io/game-servers
            operator: Equal
            value: 'true'
          containers:
          - name: simple-game-server
            image: us-docker.pkg.dev/agones-images/examples/simple-game-server:0.14
            args:
            - -automaticShutdownDelaySec=60
            - -readyIterations=10
            resources:
              limits:
                cpu: 20m
                memory: 24Mi
              requests:
                cpu: 20m
                memory: 24Mi
```

and fleet autoscaler configuration:

```
apiVersion: "autoscaling.agones.dev/v1"
kind: FleetAutoscaler
metadata:
  name: fleet-autoscaler-scenario-test
spec:
  fleetName: scenario-test
  policy:
    type: Buffer
    buffer:
      bufferSize: 2000
      minReplicas: 10000
      maxReplicas: 20000
```

To reduce pod churn in the system, the simple game servers are configured to
return themselves to `Ready` after being allocated the first 10 times following
the [Reusing Allocated GameServers for more than one game
session](https://agones.dev/site/docs/integration-patterns/reusing-gameservers/)
integration pattern. After 10 simulated game sessions, the simple game servers
then exit automatically. The fleet configuration above sets each game session to
last for 1 minute, representing a short game.

## Configuring the Allocator Service

The allocator service uses gRPC. In order to be able to call the service, TLS
and mTLS have to be set up. For more information visit
[Allocator Service](https://agones.dev/site/docs/advanced/allocator-service/).

## Running the test

You can use the provided runScenario.sh script by providing one parameter (a
scenario file). The scenario file is a simple text file where each line
represents a "scenario" that the program will execute before moving to the next
scenario. A scenario is a duration and the number of concurrent clients to use,
separated by a comma. The program will create the desired number of clients and
those clients send allocation requests to the allocator service for the scenario
duration. At the end of each scenario the program will print out some statistics
for the scenario.

Two sample scenario files are included in this directory, one which sends a
constant rate of allocations for the duration of the test and another that sends
a variable number of allocations.

Upon concluding, the program will print out the overall statistics from the test.

```
./runScenario.sh variable.txt
...
2022-02-24 10:57:44.985216321 +0000 UTC m=+13814.879251454 :Running Scenario 24 with 15 clients for 10m0s
===================

Finished Scenario 24
Count: 100              Error: ObjectHasBeenModified
Count: 113              Error: TooManyConcurrentRequests
Count: 0                Error: NoAvailableGameServer
Count: 0                Error: Unknown

Scenario Failure Count: 213, Allocation Count: 15497

Total Failure Count: 6841, Total Allocation Count: 523204

Final Error Totals
Count: 0                Error: NoAvailableGameServer
Count: 0                Error: Unknown
Count: 3950             Error: ObjectHasBeenModified
Count: 2891             Error: TooManyConcurrentRequests


2022-02-24 11:07:45.677220867 +0000 UTC m=+14415.571255996
Final Total Failure Count: 6841, Total Allocation Count: 523204
```

Since error counts are gathered per scenario, it's recommended to keep each
scenario short (e.g. 10 minutes) to narrow down the window when errors
occurred even if the allocation rate stays at the same level for longer than
10 minutes at a time.

