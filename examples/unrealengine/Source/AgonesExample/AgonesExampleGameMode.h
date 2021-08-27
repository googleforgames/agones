// Copyright Epic Games, Inc. All Rights Reserved.

#pragma once

#include "CoreMinimal.h"

#include "AgonesComponent.h"
#include "GameFramework/GameModeBase.h"
#include "AgonesExampleGameMode.generated.h"

UCLASS(minimalapi)
class AAgonesExampleGameMode : public AGameModeBase
{
	GENERATED_BODY()

public:
	UPROPERTY(EditAnywhere, BlueprintReadWrite)
	UAgonesComponent* AgonesSDK;

	AAgonesExampleGameMode();
	
	UFUNCTION()
	virtual void BeginDestroy() override;
};



