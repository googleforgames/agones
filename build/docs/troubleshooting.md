# Troubleshooting Guide

This guide covers frequent issues and possible solutions when developing with Agones.

## Common Development Issues

### $GOPATH/$GOROOT error when building in WSL

If you get this error when building Agones in WSL (`make build`, `make test` or any other related target):

```
can't load package: package agones.dev/agones/cmd/controller: cannot find package "agones.dev/agones/cmd/controller" in any of:
       /usr/local/go/src/agones.dev/agones/cmd/controller (from $GOROOT)
       /go/src/agones.dev/agones/cmd/controller (from $GOPATH)
```

- Are your project files on a different folder than C? If yes, then you should either move them on drive C or set up Docker for Windows to share your project drive as well
- Did you set up the volume mount for Docker correctly? By default, drive C is mapped by WSL as /mnt/c, but Docker expects it as /c. You can test by executing `ls /c` in your linux shell. If you get an error, then follow the instructions for [setting up volume mount for Docker](https://nickjanetakis.com/blog/setting-up-docker-for-windows-and-wsl-to-work-flawlessly#ensure-volume-mounts-work)

### Error: cluster-admin-binding already exists

This surfaces while running `make gcloud-auth-cluster`. The solution is to run `kubectl describe clusterrolebinding | grep cluster-admin-binding- -A10`, find clusterrolebinding which belongs to your `User` account and then run `kubectl delete clusterrolebindings cluster-admin-binding-<md5Hash>` where `<md5Hash>` is a value specific to your account. Now you can execute `make gcloud-auth-cluster` again. If you run into a permission denied error when attempting the delete operation, you need to run `sudo chown <your username> <path to .kube/config>` to change ownership of the file to yourself.

### Error: releases do not exist

Run `make uninstall` then run `make install` again.

## Cluster Configuration Issues

### Error: Kubernetes cluster unreachable: invalid configuration: no configuration has been provided

If you run into this error while creating a test cluster run `make terraform-clean`.

### Error: project: required field is not set Error: Invalid value for network: project: required field is not set

Run `make gcloud-init`. If you still get the same error, log into the developer
shell `make shell` and run `gcloud init`.

## Registry and Chart Issues

### Error: could not download chart: failed to download "https://agones.dev/chart/stable/agones-1.28.0.tgz"

Run `helm repo add agones https://agones.dev/chart/stable` followed by `helm repo update` which will add
the latest stable version of the agones tar file to your ~/.cache/helm/repository/.

### Invalid argument "/agones-controller:1.29.0-961d8ae-amd64" for "-t, --tag" flag: invalid reference format

The $(REGISTRY) variable is not set in ~/agones/build/Makefile. For GKE follow instructions under the [Cluster Setup Guide](cluster-setup.md) for creating a registry and exporting the path to the registry.

## Next Steps

- See [Platform Setup Guide](platform-setup.md) for platform-specific setup issues
- See [Cluster Setup Guide](cluster-setup.md) for cluster-related issues
- See [Development Workflow Guide](development-workflow.md) for debugging and development issues