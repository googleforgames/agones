# Simple Allocator Service

This service provides an example of using the [Agones API](https://godoc.org/agones.dev/agones/pkg/client/clientset/versioned/typed/agones/v1) to allocate a GameServer from a Fleet, and is used in the [Create an Allocator Service (Go)](https://agones.dev/site/docs/tutorials/allocator-service-go/) tutorial.

## Allocator Service
The service exposes an endpoint which allows client calls to AllocationInterface.Create() over a secure connection.  It also provides examples of how to create a service account with the least necessary privileges, how to create an Ingress, and how services can use secrets specific to their respective accounts.

When the endpoint is called and a GameServer is allocated, it returns the JSON encoded GameServerStatus of the freshly allocated GameServer.

To learn how to deploy this allocator service to GKE, please see the tutorial [Create an Allocator Service (Go)](https://agones.dev/site/docs/tutorials/allocator-service-go/).
