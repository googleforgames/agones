# REST Game Server Client API

This is the REST version of the Agones Game Server Client SDK. 
Check the [Client SDK Documentation](../sdks/README.md) for more details on each of the SDK functions and how to run the SDK locally.

The REST API can be accessed from `http://localhost:59358/` from the game server process.

Generally the REST interface gets used if gRPC isn't well supported for a given language or platform.

## Generating clients

While you can hand write REST integrations, we also have a [generated OpenAPI/Swagger definition](../sdk.swagger.json)
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

```bash
$ curl -d "{}" -H "Content-Type: application/json" -X POST http://localhost:59358/ready
```

### Health
Send a Empty every d Duration to declare that this GameSever is healthy

- Path: `/health`
- Method: `POST`
- Body: `{}`

#### Example

```bash
$ curl -d "{}" -H "Content-Type: application/json" -X POST http://localhost:59358/health
```

### Shutdown

Call when the GameServer session is over and it's time to shut down

- Path: `/shutdown`
- Method: `POST`
- Body: `{}`

#### Example

```bash
$ curl -d "{}" -H "Content-Type: application/json" -X POST http://localhost:59358/shutdown
```

### Set Label

Apply a Label with the prefix "stable.agones.dev/sdk-" to the backing `GameServer` metadata. 

See the SDK [SetLabel](../sdks/README.md#setlabelkey-value) documentation for restrictions.

#### Example

```bash
$ curl -d '{"key": "foo", "value": "bar"}' -H "Content-Type: application/json" -X PUT http://localhost:59358/metadata/label
```

### Set Annotation

Apply a Annotation with the prefix "stable.agones.dev/sdk-" to the backing `GameServer` metadata

#### Example

```bash
$ curl -d '{"key": "foo", "value": "bar"}' -H "Content-Type: application/json" -X PUT http://localhost:59358/metadata/annotation
```

### GameServer

Call when you want to retrieve the backing `GameServer` configuration details

- Path: `/gameserver`
- Method: `GET`

```bash
$ curl -H "Content-Type: application/json" -X GET http://localhost:59358/gameserver
```

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

```bash
$ curl -H "Content-Type: application/json" -X GET http://localhost:59358/watch/gameserver
```

Response:
```json
{"result":{"object_meta":{"name":"local","namespace":"default","uid":"1234","resource_version":"v1","generation":"1","creation_timestamp":"1533766607","annotations":{"annotation":"true"},"labels":{"islocal":"true"}},"status":{"state":"Ready","address":"127.0.0.1","ports":[{"name":"default","port":7777}]}}}
{"result":{"object_meta":{"name":"local","namespace":"default","uid":"1234","resource_version":"v1","generation":"1","creation_timestamp":"1533766607","annotations":{"annotation":"true"},"labels":{"islocal":"true"}},"status":{"state":"Ready","address":"127.0.0.1","ports":[{"name":"default","port":7777}]}}}
{"result":{"object_meta":{"name":"local","namespace":"default","uid":"1234","resource_version":"v1","generation":"1","creation_timestamp":"1533766607","annotations":{"annotation":"true"},"labels":{"islocal":"true"}},"status":{"state":"Ready","address":"127.0.0.1","ports":[{"name":"default","port":7777}]}}}
```