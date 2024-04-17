---
title: "Agones SDK Server"
linkTitle: "SDK Server"
date: 2024-04-18
weight: 1001
description: "The SDK Server is a sidecar for a GameServer that will update the GameServer Status on SDK requests."
---

## SDK Server Overview
- The SDK Server is a gRPC server. The methods for communication between the SDK Client and SDK Server are defined in the [SDK proto](https://github.com/googleforgames/agones/blob/main/proto/sdk).
- The REST API is generated directly from the SDK .proto files.
- All other SDK Client APIs are wrappers on top of the SDK proto definitions.

{{% feature publishVersion="1.43.0" %}}

{{< alert title="Note" color="info">}}
This section is specifically about the SDK proto compatibility between Agones versions and deprecation policies.
{{< /alert >}}

## SDK Proto Compatibility Guarantees
In order to allow compatibility between game server binaries and sdk-server, a game server binary using Beta and Stable SDK protos must remain compatible with a newer sdk-server.
- Our SDK Server compatibility contract: If your game server uses a non-deprecated Stable API, your binary will be compatible for 10 releases (~1y) starting from the SDK version packaged
  - For example, if the game server uses non-deprecated stable APIs in the 1.40 SDK, it will be compatible through the 1.50 SDK.
  - Stable APIs will almost certainly be compatible beyond 10 releases, but 10 releases is guaranteed.
- Similar to the compatibility contract for the Stable API, using a non-deprecated Beta API, your binary will be compatible for 5 releases (~6mo).
- Alpha SDK Protos (and APIs/RPCs) are subject to change between releases.
  - A game server binary using Alpha SDKs may not be compatible with a newer sdk-server.
  - In Alpha, incompatible changes retaining the same SDK proto message name are allowed.
  - When we make incompatible Alpha changes, we will document the APIs involved.

## SDK Deprecation Policies
Breaking changes will be called out in upgrade documentation, alongside the build horizon, allowing operators to plan their upgrades.
### Stable Deprecation Policies
A Stable API may be marked as deprecated in release X and removed from Stable in release X+10.
Changes to the stable API must first flow through at least Beta.
### Beta Deprecation Policies
When a feature graduates from Beta to Stable at release X the API is in both protos from release X to release X+5. The Beta API is marked as deprecated in release X and removed from Beta in release X+5.
A Beta API may be marked as deprecated in release X and removed from Beta in release X+5 without the API graduating to Stable.
### Alpha Deprecation Policies
There is no guaranteed proto compatibility between releases for Alpha SDK protos. When an Alpha API graduates to Beta the API will be deleted from the Alpha proto with no overlapping release.
An API may be removed from the Alpha proto during any release without graduating to Beta.

## SDK Server APIs and Stability Levels

"Legacy" indicates that this API has been in the SDK Server in a release before we began tracking proto compatibility.

{{< alert title="Note" color="info">}}
The Actions may differ from the Client SDKs depending on how each Client SDK is implemented.
{{< /alert >}}

| Area                | Action                | Stable | Beta | Alpha  |
|---------------------|-----------------------|--------|------|--------|
| Lifecycle           | Ready                 | Legacy |      |        |
| Lifecycle           | Health                | Legacy |      |        |
| Lifecycle           | Reserve               | Legacy |      |        |
| Lifecycle           | Allocate              | Legacy |      |        |
| Lifecycle           | Shutdown              | Legacy |      |        |
| Configuration       | GetGameServer         | Legacy |      |        |
| Configuration       | WatchGameServer       | Legacy |      |        |
| Metadata            | SetAnnotation         | Legacy |      |        |
| Metadata            | SetLabel              | Legacy |      |        |
| Counters            | GetCounter            |        |      | 1.37.0 |
| Counters            | UpdateCounter         |        |      | 1.37.0 |
| Lists               | GetList               |        |      | 1.37.0 |
| Lists               | UpdateList            |        |      | 1.37.0 |
| Lists               | AddListValue          |        |      | 1.37.0 |
| Lists               | RemoveListValue       |        |      | 1.37.0 |
| Player Tracking     | GetPlayerCapacity     |        |      | Legacy |
| Player Tracking     | SetPlayerCapacity     |        |      | Legacy |
| Player Tracking     | PlayerConnect         |        |      | Legacy |
| Player Tracking     | GetConnectedPlayers   |        |      | Legacy |
| Player Tracking     | IsPlayerConnected     |        |      | Legacy |
| Player Tracking     | GetPlayerCount        |        |      | Legacy |
| Player Tracking     | PlayerDisconnect      |        |      | Legacy |

{{% /feature %}}
