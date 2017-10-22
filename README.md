# Agon

![Agon Dog](agon.jpg)

Agon is a library for running dedicated game servers on [Kubernetes](https://kubernetes.io).

## Roadmap for 0.1 release
- Develop a [Custom Resource Defintion](https://kubernetes.io/docs/concepts/api-extension/custom-resources/#customresourcedefinitions) for dedicated game server
- Sidecar for managing the DGS lifecycle and recorded status, e.g. registering the port the server has started on
- A Kubernetes operator that registers the CRD, and creates a Pod with the DGS in it, with the accompanying sidecar for system registration.
- A basic client library for integration with a DGS
- Simple example code
- Documentation of the above

## Licence

Apache 2.0

This is not an official Google product.