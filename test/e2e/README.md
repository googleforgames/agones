# E2E Testing

End-to-end (e2e) testing is automated testing for real user scenarios.

## Build and run test

Prerequisites:
- a running k8s cluster (kube config is passed as arguments).
- Have kubeconfig file ready.
- (optional) set the `IMAGE_PULL_SECRET` env var to the secret name needed to pull the gameserver and/or Agones SDK images, if needed

e2e tests are written as Go test. All go test techniques apply, e.g. picking
what to run, timeout length.

To run e2e tests on your kubectl configured cluster:

```
make test-e2e
```

To run a single test on your kubectl configured cluster you can optionally include any flags listed
in e2e test args in the agones/build/Makefile such as `FEATURE_GATES="CountsAndLists=true"`:

```
FEATURE_GATES="CountsAndLists=true" go test -race -run ^TestCounterAutoscaler$
```

To run on minikube use the special target:

```
make minikube-test-e2e
```
