# Platform Setup Guide

This guide covers setting up your development environment for building Agones on different platforms.

## Building on Different Platforms

### Linux
- Install Make, either via `apt install make` or `yum install make` depending on platform.
- [Install Docker](https://docs.docker.com/engine/installation/) for your Linux platform.

### Windows
Building and developing Agones requires you to use the
[Windows Subsystem for Linux](https://blogs.msdn.microsoft.com/wsl/)(WSL),
as this makes it easy to create a (relatively) cross platform development and build system.

- [Install WSL](https://docs.microsoft.com/en-us/windows/wsl/install-win10)
  - Preferred release is [Ubuntu 16.04](https://www.microsoft.com/en-us/store/p/ubuntu/9nblggh4msv6) or greater.
- [Install Docker for Windows](https://docs.docker.com/docker-for-windows/install/)
- Within WSL, Install [Docker for Ubuntu](https://docs.docker.com/engine/installation/linux/docker-ce/ubuntu/)
- Follow [this guide](https://nickjanetakis.com/blog/setting-up-docker-for-windows-and-wsl-to-work-flawlessly)
  from "Configure WSL to Connect to Docker for Windows" forward
  for integrating the Docker on WSL with the Windows Docker installation
  - Note the binding of `/c` to `/mnt/c` (or drive of your choice) - this is very important!
- Agones will need to be cloned somewhere on your `/c` (or drive of your choice) path, as that is what Docker will support mounts from
- All interaction with Agones must be on the `/c` (or drive of your choice) path, otherwise Docker mounts will not work
- Now the `make` commands can all be run from within your WSL shell
- The Minikube setup on Windows has not been tested. Pull Requests would be appreciated!

### macOS

- Install Make, `brew install make`, if it's not installed already
- Install [Docker for Mac](https://docs.docker.com/docker-for-mac/install/)

## Next Steps

After setting up your platform:
1. See [Building and Testing Guide](building-testing.md) for basic build workflows
2. See [Cluster Setup Guide](cluster-setup.md) to set up a development cluster
3. See [Development Workflow Guide](development-workflow.md) for common development patterns