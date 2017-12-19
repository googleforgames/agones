# Agon Game Server Client SDKs

The SDKs are integration points for game servers with Agon itself.

They are required for a game server to work with Agon.

There are currently two support SDKs:
- [C++](cpp)
- [Go](go)

The SDKs are relatively thin wrappers around [gRPC](https://grpc.io), generated clients,
which connects to a small process that Agon coordinates to run alongside the Game Server
in a Kubernetes [`Pod`](https://kubernetes.io/docs/concepts/workloads/pods/pod-overview/).
This means that more languages can be supported in the future with minimal effort
(but pull requests are welcome! 😊 ).

## Function Reference

While each of the SDKs are canonical to their languages, they all have the following
functions that implement the core responsibilities of the SDK.

For language specific documentation, have a look at the respective source (linked above), 
and the [examples](../examples)

### Ready()
This tells Agon that the Game Server is ready to take player connections.
One a Game Server has specified that it is `Ready`, then the Kubernetes
GameServer record will be moved to the `Ready` state, and the details
for its public address and connection port will be populated.

### Shutdown()
This tells Agon to shut down the currently running game server.
The GameServer state will be set `Shutdown` and the 
backing Pod will be deleted, if they have not shut themselves down already. 

## Local Development
(Coming soon: Track [this bug](https://github.com/googleprivate/agon/issues/8) for details)
