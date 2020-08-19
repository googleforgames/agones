# NOTE - SDK from Netspeak Games Adopted Officially as of 1.8.0

With version 1.8.0 of Agones, the previous Unreal plugin has been replaced, this new SDK was originally developed as a [separate project](https://github.com/netspeakgames/UnrealAgonesSDK) by Netspeak Games. While this is a breaking change, the older SDK may continue to be used if desired. This SDK is far more feature complete, much more in keeping with the Unreal plugin style, and is recommended as a general solution for Unreal Engine Agones developers going forward.

# Unreal SDK

This SDK is inspired by the REST API to the Agones sidecars that allows engineers to talk to the sidecar from either C++ or Blueprints.

## Getting Started

The Agones Unreal SDK can either be used from C++ or from Blueprints.

### Getting the Code

The easiest way to get this code is to clone the repository and drop the entire plugin folder into your own `Plugins` folder. This runs the plugin as a Project plugin rather than an engine plugin.

### Migration Example and Notes

Scenario: the old SDK was used to set a label.

Remove the old plugin and add the new one as a component (per Unreal docs) and then change the calls (something like):

Previous:
```
Hook->SetLabel(TEXT("key"), TEXT("value"));
```
Current:
```
AgonesComponent->SetLabel(TEXT("key"), TEXT("value"), successDelegate, errorDelegate);
```

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
