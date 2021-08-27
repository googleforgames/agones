// Copyright Epic Games, Inc. All Rights Reserved.

#include "AgonesExampleGameMode.h"
#include "AgonesExampleHUD.h"
#include "AgonesExampleCharacter.h"
#include "UObject/ConstructorHelpers.h"

AAgonesExampleGameMode::AAgonesExampleGameMode()
	: Super()
{
	// set default pawn class to our Blueprinted character
	static ConstructorHelpers::FClassFinder<APawn> PlayerPawnClassFinder(TEXT("/Game/FirstPersonCPP/Blueprints/FirstPersonCharacter"));
	DefaultPawnClass = PlayerPawnClassFinder.Class;

	// use our custom HUD class
	HUDClass = AAgonesExampleHUD::StaticClass();
}
