# Simple Allocator Service

This service provides an example of using the Agones API to allocate a GameServer from a Fleet.

:warning: Requires Agones installed with agones.image.tag=0.5.0-03f4866 or newer :warning:

## Allocator Service
The service exposes an endpoint which allows client calls to FleetAllocationInterface.Create() over a secure connection.  It also provides examples of how to create a service account with the least necessary privileges, how to create an Ingress, and how services can use secrets specific to their respective accounts.

When the endpoint is called and a GameServer is allocated, it returns the JSON encoded IP addresss and port number of the freshly allocated GameServer.

To learn how to deploy your edited version of an allocator service to GKE, please see the quick start [Create an Allocator Service (Go)](../../docs/create_allocator_service.md).
