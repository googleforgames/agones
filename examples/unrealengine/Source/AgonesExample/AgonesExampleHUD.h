// Copyright Epic Games, Inc. All Rights Reserved.

#pragma once 

#include "CoreMinimal.h"
#include "GameFramework/HUD.h"
#include "AgonesExampleHUD.generated.h"

UCLASS()
class AAgonesExampleHUD : public AHUD
{
	GENERATED_BODY()

public:
	AAgonesExampleHUD();

	/** Primary draw call for the HUD */
	virtual void DrawHUD() override;

private:
	/** Crosshair asset pointer */
	class UTexture2D* CrosshairTex;

};

