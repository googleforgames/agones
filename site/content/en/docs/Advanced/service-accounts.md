---
title: "GameServer Pod Service Accounts"
linkTitle: "Service Accounts"
date: 2019-03-14T04:30:37Z
publishDate: 2019-04-01
description: >
  RBAC permissions and service accounts for the `GameServer` Pod. 
---

## Default Settings

By default, Agones sets up service accounts and sets them appropriately for the `Pods` that are created for `GameServers`.

Since Agones provides `GameServer` `Pods` with a sidecar container that needs access to Agones Custom Resource Definitions,
`Pods` are configured with a service account with extra RBAC permissions to ensure that it can read and modify the resources it needs.

Since service accounts apply to all containers in a `Pod`, Agones will automatically overwrite the mounted key for the 
service account in the container that is running the dedicated game server in the backing `Pod`. This is done
since game server containers are exposed publicly, and generally don't require the extra permissions to access aspects 
of the Kubernetes API.

## Bringing your own Service Account

If needed, you can provide your own service account on the `Pod` specification in the `GameServer` configuration.

For example:

```yaml
apiVersion: "agones.dev/v1"
kind: GameServer
metadata:
  generateName: "simple-udp-"
spec:
  ports:
  - name: default
    containerPort: 7654
  template:
    spec:
      serviceAccountName: my-special-service-account # a custom service account
      containers:
      - name: simple-udp
        image: {{% example-image %}}
```

If a service account is configured, the mounted key is not overwritten, as it assumed that you want to have full control
of the service account and underlying RBAC permissions.
