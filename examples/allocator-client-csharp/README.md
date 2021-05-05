# A sample Allocator service C# client

This sample serves as a gRPC C# client sample code for agones-allocator gRPC service.

Follow instructions in [Allocator Service](https://agones.dev/site/docs/advanced/allocator-service/) to set up client and server certificate.

Run the following to allocate a game server:
```
#!/bin/bash

NAMESPACE=default # replace with any namespace
EXTERNAL_IP=`kubectl get services agones-allocator -n agones-system -o jsonpath='{.status.loadBalancer.ingress[0].ip}'`
KEY_FILE=client.key
CERT_FILE=client.crt
TLS_CA_FILE=ca.crt
MULTICLUSTER_ENABLED=false

dotnet run $KEY_FILE $CERT_FILE $TLS_CA_FILE $EXTERNAL_IP $NAMESPACE $MULTICLUSTER_ENABLED
```
