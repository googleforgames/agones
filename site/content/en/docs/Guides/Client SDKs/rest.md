---
title: "REST Game Server Client API"
linkTitle: "Rest"
date: 2019-01-02T10:18:08Z
weight: 100
description: "This is the REST version of the Agones Game Server Client SDK. "
---

Check the [Client SDK Documentation]({{< relref "_index.md" >}}) for more details on each of the SDK functions and how to run the SDK locally.

{{% feature expiryVersion="1.1.0" %}}
The REST API can be accessed from `http://localhost:59358/` from the game server process.
{{% /feature %}}

{{% feature publishVersion="1.1.0" %}}
The REST API can be accessed from `http://localhost:${AGONES_SDK_HTTP_PORT}/` from the game server process.
`AGONES_SDK_HTTP_PORT` is an environment variable automatically set for the game server process by Agones to
support binding the REST API to a dynamic port. It is advised to use the environment variable rather than a
hard coded port; otherwise your game server will not be able to contact the SDK server if it is configured to
use a non-default port.
{{% /feature %}}

Generally the REST interface gets used if gRPC isn't well supported for a given language or platform.

> Note: The SDK Server sidecar process may startup after your game server binary. So your REST SDK API calls should
> contain some retry logic to take this into account. 

## Generating clients

While you can hand write REST integrations, we also have a {{< ghlink href="sdk.swagger.json" >}}generated OpenAPI/Swagger definition{{< /ghlink >}}
available. This means you can use OpenAPI/Swagger tooling to generate clients as well, if you need them.

For example (to be run in the `agones` home directory):
```bash
docker run --rm -v ${PWD}:/local swaggerapi/swagger-codegen-cli generate -i /local/sdk.swagger.json  -l cpprest -o /local/out/cpp
```

You can read more about OpenAPI/Swagger code generation in their [Command Line Tool Documentation](https://swagger.io/docs/open-source-tools/swagger-codegen/)

## Reference 

### Ready

Call when the GameServer is ready to accept connections

- Path: `/ready`
- Method: `POST`
- Body: `{}`

#### Example

{{% feature expiryVersion="1.1.0" %}}
```bash
$ curl -d "{}" -H "Content-Type: application/json" -X POST http://localhost:59358/ready
```
{{% /feature %}}
{{% feature publishVersion="1.1.0" %}}
```bash
$ curl -d "{}" -H "Content-Type: application/json" -X POST http://localhost:${AGONES_SDK_HTTP_PORT}/ready
```
{{% /feature %}}

### Health
Send a Empty every d Duration to declare that this GameSever is healthy

- Path: `/health`
- Method: `POST`
- Body: `{}`

#### Example

{{% feature expiryVersion="1.1.0" %}}
```bash
$ curl -d "{}" -H "Content-Type: application/json" -X POST http://localhost:59358/health
```
{{% /feature %}}
{{% feature publishVersion="1.1.0" %}}
```bash
$ curl -d "{}" -H "Content-Type: application/json" -X POST http://localhost:${AGONES_SDK_HTTP_PORT}/health
```
{{% /feature %}}

### Shutdown

Call when the GameServer session is over and it's time to shut down

- Path: `/shutdown`
- Method: `POST`
- Body: `{}`

#### Example

{{% feature expiryVersion="1.1.0" %}}
```bash
$ curl -d "{}" -H "Content-Type: application/json" -X POST http://localhost:59358/shutdown
```
{{% /feature %}}
{{% feature publishVersion="1.1.0" %}}
```bash
$ curl -d "{}" -H "Content-Type: application/json" -X POST http://localhost:${AGONES_SDK_HTTP_PORT}/shutdown
```
{{% /feature %}}

### Set Label

Apply a Label with the prefix "agones.dev/sdk-" to the backing `GameServer` metadata. 

See the SDK [SetLabel]({{< ref "/docs/Guides/Client SDKs/_index.md#setlabel-key-value" >}}) documentation for restrictions.

#### Example

{{% feature expiryVersion="1.1.0" %}}
```bash
$ curl -d '{"key": "foo", "value": "bar"}' -H "Content-Type: application/json" -X PUT http://localhost:59358/metadata/label
```
{{% /feature %}}
{{% feature publishVersion="1.1.0" %}}
```bash
$ curl -d '{"key": "foo", "value": "bar"}' -H "Content-Type: application/json" -X PUT http://localhost:${AGONES_SDK_HTTP_PORT}/metadata/label
```
{{% /feature %}}

### Set Annotation

Apply a Annotation with the prefix "agones.dev/sdk-" to the backing `GameServer` metadata

#### Example

{{% feature expiryVersion="1.1.0" %}}
```bash
$ curl -d '{"key": "foo", "value": "bar"}' -H "Content-Type: application/json" -X PUT http://localhost:59358/metadata/annotation
```
{{% /feature %}}
{{% feature publishVersion="1.1.0" %}}
```bash
$ curl -d '{"key": "foo", "value": "bar"}' -H "Content-Type: application/json" -X PUT http://localhost:${AGONES_SDK_HTTP_PORT}/metadata/annotation
```
{{% /feature %}}

### GameServer

Call when you want to retrieve the backing `GameServer` configuration details

- Path: `/gameserver`
- Method: `GET`

{{% feature expiryVersion="1.1.0" %}}
```bash
$ curl -H "Content-Type: application/json" -X GET http://localhost:59358/gameserver
```
{{% /feature %}}
{{% feature publishVersion="1.1.0" %}}
```bash
$ curl -H "Content-Type: application/json" -X GET http://localhost:${AGONES_SDK_HTTP_PORT}/gameserver
```
{{% /feature %}}

Response:
```json
{
    "object_meta": {
        "name": "local",
        "namespace": "default",
        "uid": "1234",
        "resource_version": "v1",
        "generation": "1",
        "creation_timestamp": "1531795395",
        "annotations": {
            "annotation": "true"
        },
        "labels": {
            "islocal": "true"
        }
    },
    "status": {
        "state": "Ready",
        "address": "127.0.0.1",
        "ports": [
            {
                "name": "default",
                "port": 7777
            }
        ]
    }
}
```

### Watch GameServer

Call this when you want to get updates of when the backing `GameServer` configuration is updated.

These updates will come as newline delimited JSON, send on each update. To that end, you will
want to keep the http connection open, and read lines from the result stream and and process as they
come in.

{{% feature expiryVersion="1.1.0" %}}
```bash
$ curl -H "Content-Type: application/json" -X GET http://localhost:59358/watch/gameserver
```
{{% /feature %}}
{{% feature publishVersion="1.1.0" %}}
```bash
$ curl -H "Content-Type: application/json" -X GET http://localhost:${AGONES_SDK_HTTP_PORT}/watch/gameserver
```
{{% /feature %}}

Response:
```json
{"result":{"object_meta":{"name":"local","namespace":"default","uid":"1234","resource_version":"v1","generation":"1","creation_timestamp":"1533766607","annotations":{"annotation":"true"},"labels":{"islocal":"true"}},"status":{"state":"Ready","address":"127.0.0.1","ports":[{"name":"default","port":7777}]}}}
{"result":{"object_meta":{"name":"local","namespace":"default","uid":"1234","resource_version":"v1","generation":"1","creation_timestamp":"1533766607","annotations":{"annotation":"true"},"labels":{"islocal":"true"}},"status":{"state":"Ready","address":"127.0.0.1","ports":[{"name":"default","port":7777}]}}}
{"result":{"object_meta":{"name":"local","namespace":"default","uid":"1234","resource_version":"v1","generation":"1","creation_timestamp":"1533766607","annotations":{"annotation":"true"},"labels":{"islocal":"true"}},"status":{"state":"Ready","address":"127.0.0.1","ports":[{"name":"default","port":7777}]}}}
```

### Reserve

Move Gameserver into a Reserved state for a certain amount of seconds for the future allocation.

- Path: `/reserve`
- Method: `POST`
- Body: `{"seconds": "5"}`

#### Example

{{% feature expiryVersion="1.1.0" %}}
```bash
$ curl -d '{"seconds": "5"}' -H "Content-Type: application/json" -X POST http://localhost:59358/reserve
```
{{% /feature %}}
{{% feature publishVersion="1.1.0" %}}
```bash
$ curl -d '{"seconds": "5"}' -H "Content-Type: application/json" -X POST http://localhost:${AGONES_SDK_HTTP_PORT}/reserve
```
{{% /feature %}}

### Allocate

With some matchmakers and game matching strategies, it can be important for game servers to mark themselves as `Allocated`.
For those scenarios, this SDK functionality exists. 

> Note: Using a [GameServerAllocation]({{< ref "/docs/Reference/gameserverallocation.md" >}}) is preferred in all other scenarios, 
as it gives Agones control over how packed `GameServers` are scheduled within a cluster, whereas with `Allocate()` you
relinquish control to an external service which likely doesn't have as much information as Agones.

#### Example

{{% feature expiryVersion="1.1.0" %}}
```bash
$ curl -d "{}" -H "Content-Type: application/json" -X POST http://localhost:59358/allocate
```
{{% /feature %}}
{{% feature publishVersion="1.1.0" %}}
```bash
$ curl -d "{}" -H "Content-Type: application/json" -X POST http://localhost:${AGONES_SDK_HTTP_PORT}/allocate
```
{{% /feature %}}
