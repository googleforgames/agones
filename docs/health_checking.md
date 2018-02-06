# GameServer Health Checking

Health checking exists to track the overall healthy state of the GameServer, 
such that action can be taken when a something goes wrong or a GameServer drops into an Unhealthy state

## Disabling Health Checking

By default, health checking is enabled, but it can be turned off by setting the `health > disabled` property to true

## SDK API

The `Health()` function on the [SDK object](../sdks) needs to be called at an interval less than the `health > periodSeconds`
threshold time to be considered before it will be considered a `failure`.

The health check will also need to have not been called a consecutive number of times (`health > failureTheshold`),
giving it a chance to heal if it there is an issue.

## Health Failure Strategy

The following is the process for what happens to a `GameServer` when it is unhealthy.

1. If any of the GameServer container fails health checking, or exits before the GameServer moves to `Ready` then, 
   it is restart as per the `restartPolicy` (which defaults to "Always")
1. If the GameServer container fails healthy checking after the `Ready` state, then it doesn't restart, 
   but moves the GameServer to an `Unhealthy` state.
1. If the GameServer container exits while in `Ready` state, it will restart as per the `restartPolicy` 
   (which defaults to "Always", since `RestartPolicy` is a Pod wide setting), 
   but will immediately move to an `Unhealthy` state.
1. If the SDK sidecar fails, then it wiil restarted, assuming the `RestartPolicy` is Always/OnFailure.

## Reference
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

See the [full GameServer example](../examples/gameserver.yaml) for more details

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

Also look in the [examples](../examples) directory.