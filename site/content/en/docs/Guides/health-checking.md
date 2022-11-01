---
title: "GameServer Health Checking"
linkTitle: "GameServer Health Checking"
date: 2019-01-03T01:20:19Z
weight: 20
description: >
  Health checking exists to track the overall healthy state of the GameServer, 
  such that action can be taken when a something goes wrong or a GameServer drops into an Unhealthy state
---

## Disabling Health Checking

By default, health checking is enabled, but it can be turned off by setting the `spec.health.disabled` property to
true.

## SDK API

The `Health()` function on the [SDK object]({{< relref "./Client SDKs/_index.md" >}}) needs to be called at an
interval less than the `spec.health.periodSeconds`
threshold time to be considered before it will be considered a `failure`.

The health check will also need to have not been called a consecutive number of times (`spec.health.failureThreshold`),
giving it a chance to heal if it there is an issue.

## Health Failure Strategy

The following is the process for what happens to a `GameServer` when it is unhealthy.

1. If the `GameServer` container exits with an error before the `GameServer` moves to `Ready` then, 
   it is restarted as per the `restartPolicy` (which defaults to "Always").
1. If the `GameServer` fails health checking at any point, then it doesn't restart, 
   but moves to an `Unhealthy` state.
1. If the `GameServer` container exits while in `Ready`, `Allocated` or `Reserved` state, it will be restarted 
   as per the `restartPolicy`  (which defaults to "Always", since `RestartPolicy` is a `Pod` wide setting), 
   but will immediately move to an `Unhealthy` state.
1. If the SDK sidecar fails, then it will be restarted, assuming the `RestartPolicy` is Always/OnFailure.

## Fleet Management of Unhealthy GameServers

If a `GameServer` moves into an `Unhealthy` state when it is not part of a Fleet, the `GameServer` will remain in the
Unhealthy state until explicitly deleted.  This is useful for debugging `Unhealthy` `GameServers`, or if you are
creating your own `GameServer` management layer, you can explicitly choose what to do if a `GameServer` becomes
`Unhealthy`.
  
If a `GameServer` is part of a `Fleet`, the `Fleet` management system will _delete_ any `Unhealthy` `GameServers` and
immediately replace them with a brand new `GameServer` to ensure it has the configured number of Replicas.

## Configuration Reference
```yaml
  # Health checking for the running game server
  health:
    # Disable health checking. defaults to false, but can be set to true
    disabled: false
    # Number of seconds after the container has started before health check is initiated. Defaults to 5 seconds
    initialDelaySeconds: 5
    # If the `Health()` function doesn't get called at least once every period (seconds), then
    # the game server is not healthy. Defaults to 5
    periodSeconds: 5
    # Minimum consecutive failures for the health probe to be considered failed after having succeeded.
    # Defaults to 3. Minimum value is 1
    failureThreshold: 3
```

See the {{< ghlink href="examples/gameserver.yaml" >}}full GameServer example{{< /ghlink >}} for more details

## Example

### C++

For a configuration that requires a health ping every 5 seconds, the example below sends a request every 2 seconds
to be sure that the GameServer is under the threshold.

```cpp
void doHealth(agones::SDK *sdk) {
    while (true) {
        if (!sdk->Health()) {
            std::cout << "Health ping failed" << std::endl;
        } else {
            std::cout << "Health ping sent" << std::endl;
        }
        std::this_thread::sleep_for(std::chrono::seconds(2));
    }
}

int main() {
    agones::SDK *sdk = new agones::SDK();
    bool connected = sdk->Connect();
    if (!connected) {
        return -1;
    }
    std::thread health (doHealth, sdk);

    // ...  run the game server code

}
```

### Full Game Server

Also look in the {{< ghlink href="examples" >}}examples{{< /ghlink >}} directory.

