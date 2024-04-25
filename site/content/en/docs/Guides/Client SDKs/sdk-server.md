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
