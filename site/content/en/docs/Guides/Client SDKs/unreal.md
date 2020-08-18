---
title: "Unreal Engine Game Server Client Plugin"
linkTitle: "Unreal Engine"
date: 2019-06-13T10:17:50Z
publishDate: 2019-05-13
weight: 10
description: "This is the Unreal Engine 4 Agones Game Server Client Plugin. "
---

{{< alert title="Note" color="info" >}}
The Unreal SDK is functional, but not yet feature complete.
[Pull requests](https://github.com/googleforgames/agones/pulls) to finish the functionality are appreciated.
{{< /alert >}}

Check the [Client SDK Documentation]({{< relref "_index.md" >}}) for more details on each of the SDK functions and how to run the SDK locally.

## Download

Download the source from the [Releases Page](https://github.com/googleforgames/agones/releases)
or {{< ghlink href="sdks/unreal" >}}directly from GitHub{{< /ghlink >}}.

## Getting Started

This is a SDK inspired by the REST API to the Agones sidecars that allows engineers to communicate with the sidecar from either C++ or Blueprints.

### Getting the Code

Easiest way to get this code is to clone the repository and drop the entire plugin folder into your own `Plugins` folder. This runs the plugin as a Project plugin rather than an engine plugin.

We could however turn this into a marketplace plugin that can be retrived from the marketplace directly into the UE4 editor.

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
![component](../../../../images/unreal_bp_component.png)
- This will automatically call `/health` every 10 seconds and once `/gameserver` calls are succesful it will call `/ready`.

- Accessing other functionality of Agones can be done via adding a node in Blueprints.
![actions](../../../../images/unreal_bp_actions.png)

## Configuration Options

A number of options can be altered via config files in Unreal these are supplied via `Game` configuration eg. `DefaultGame.ini`.

```
[/Script/Agones.AgonesComponent]
HttpPort=1337
HealthRateSeconds=5.0
bDisableAutoConnect=true
```

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
