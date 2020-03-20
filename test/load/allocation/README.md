This is a load test to determine Allocation QPS over time against a set of GameServers that are constantly being shutdown after a period.

This test creates a configured amount of GameServers at the initial step, switches them to Allocated state and finally shuts them down (`automaticShutdownDelayMin` flag in a simple-udp).

1) Run kubectl apply -f ./fleet.yaml
2) Run `runAllocation.sh` script to perform this test. You can provide a number of runs as a parameter (3 is a default value). There is a 500 seconds pause after each run.

To run this test under normal conditions the number of replicas in the yaml file should be >= numberOfClients * reqPerClient
