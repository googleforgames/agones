This test creates a proper amount of GameServers at the initial step, switch them to Allocated state and finally shut them down (`automaticShutdownDelayMin` flag in a simple-udp).

1) Run kubectl apply -f ./fleet.yaml
2) Run `runAllocation.sh` script to perform this test. You can provide a number of runs as a parameter (3 is a default value)

Number of allocated GameServers = numberOfClients * reqPerClient (see `allocationload.go`)

To run this test in under the normal conditions number of replicas in the yaml file should be >= allocated GameServers.

This test can be used as a continuous hours running test.
