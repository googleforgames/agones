# Agon

Agon is a library for running dedicated game servers on [Kubernetes](https://kubernetes.io).

## Disclaimer
This software is currenty alpha, and subject to change. Not to be used in production systems.

## Roadmap for 0.1 release
- Develop a [Custom Resource Defintion](https://kubernetes.io/docs/concepts/api-extension/custom-resources/#customresourcedefinitions) for dedicated game server
- Sidecar for managing the DGS lifecycle and recorded status, e.g. registering the port the server has started on
- A Kubernetes operator that registers the CRD, and creates a Pod with the DGS in it, with the accompanying sidecar for system registration.
- A basic client library for integration with a DGS
- Simple example code
- Documentation of the above

## Requirements
- Requires a Kubernetes cluster of version 1.8+
- Open the firewall access for the range of ports that Game Servers can be connected to in the cluster.

## Installation
`kubectl apply -f install.yaml`

If you are running your own Docker repository or want to install a specific version, make a local copy of install.yaml
and edit to match your settings. 

## Usage
See the [examples](./examples) directory
 
## Development
See the tools in the [build](build/README.md) directory for testing and building Agon.

## Licence

Apache 2.0