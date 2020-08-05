# Agones Unreal SDK

Agones is a multilayer dedicated game server scaling and orchestration platform that can run anywhere Kubernetes can run.

This is a SDK inspired by the REST API to the Agones sidecars that allows engineers to talk to the sidecar from either C++ or Blueprints.

## Getting Started

The Agones Unreal SDK can either be used from C++ or from Blueprints.

### Getting the Code

Easiest way to get this code is to clone the repository and drop the entire plugin folder into your own `Plugins` folder. This runs the plugin as a Project plugin rather than an engine plugin.

We could however turn this into a marketplace plugin that can be retrived from the marketplace directly into the UE4 editor.

### Health Calls
#### Using C++
- Add Plugin (in your own `.uplugin` file)
```
  "Plugins": [
    {
      "Enabled": true,
      "Name": "Agones"
    }
  ],
```
- Add Plugin (in your own `*.Build.cs`)
```
PublicDependencyModuleNames.AddRange(
    new[]
    {
        "Agones",
    });
```
- Add component in header
```c++
#include "AgonesComponent.h"

UPROPERTY(EditAnywhere, BlueprintReadWrite)
UAgonesComponent* AgonesSDK;
```
- Initialize component in GameMode
```c++
#include "AgonesComponent.h"
#include "Classes.h"

ATestGameMode::ATestGameMode()
{
	AgonesSDK = CreateDefaultSubobject<UAgonesComponent>(TEXT("AgonesSDK"));
}
```

- Use the Agones component to call PlayerReady
```c++
void APlatformGameSession::PostLogin(APlayerController* NewPlayer)
{
  // Empty brances are for callbacks on success and errror.
  AgonesSDK->PlayerConnect("netspeak-player", {}, {});
}
```

#### Using Blueprints
- Add Component to your Blueprint GameMode
![component](/docs/img/01_bp_component.PNG)
- This will automatically call `/health` every 10 seconds and once `/gameserver` calls are succesful it will call `/ready`.

- Accessing other functionality of Agones can be done via adding a node in Blueprints.
![actions](/docs/img/02_bp_actions.PNG)


## SDK Functionality

Additional methods have been added for ease of use (both of which are enabled by default):

- Connect
  - will call `/gameserver` till a succesful response is returned and then call `/ready`.
  - disabled by setting `bDisableAutoConnect` to `true`.
  - An event is broadcast with the `GameServer` data once the `/gameserver` call succeeds.
- Health
  - calls `/health` endpoint on supplied rate
  - enabled by default with 10 second rate
  - disabled by default by setting `HealthRateSeconds` to `0`.

Both of the above are automatically kicked off in the `BeginPlay` of the component.

This Agones SDK wraps the REST API and supports the following actions:

Stable
- Lifecycle
  - Ready
  - Health
  - Reserve
  - Allocate
  - Shutdown
- Configuration
  - GameServer
- Metadata
  - SetAnnotation
  - SetLabel

Alpha
- Player Tracking
  - GetConnectedPlayers
  - GetPlayerCapacity
  - GetPlayerCount
  - IsPlayerConnected
  - PlayerConnect
  - PlayerDisconnect
  - SetPlayerCapacity

Unimplemented
  - WatchGameServer

Current the only missing functionality is the `WatchGameServer` functionality. We welcome collaborators to help implement this, if people need it before we get around to implementing it ourselves.

## Unreal Hooks

Within the Unreal [GameMode](https://docs.unrealengine.com/en-US/API/Runtime/Engine/GameFramework/AGameMode/index.html) and [GameSession](https://docs.unrealengine.com/en-US/API/Runtime/Engine/GameFramework/AGameSession/index.html) exist a number of useful existing
funtions that can be used to fit in with making calls out to Agones.

A few examples are:
- `RegisterServer` to call `SetLabel`, `SetPlayerCapacity`
- `PostLogin` to call `PlayerConnect`
- `NotifyLogout` to call `PlayerDisconnect`