// Fill out your copyright notice in the Description page of Project Settings.

#pragma once

#include "CoreMinimal.h"
#include "GameFramework/GameSession.h"
#include "AgonesComponent.h"
#include "AgonesExampleGameSession.generated.h"

/**
 * 
 */
UCLASS()
class AGONESEXAMPLE_API AAgonesExampleGameSession : public AGameSession
{
	GENERATED_BODY()

protected:
	UPROPERTY(EditAnywhere, BlueprintReadWrite)
	UAgonesComponent* AgonesSDK;

public:
	AAgonesExampleGameSession();

	UPROPERTY()
	FPlayerConnectDelegate PlayerConnectDelegate;

	UPROPERTY()
	FAgonesErrorDelegate AgonesErrorDelegate;

	UFUNCTION()
	virtual void RegisterServer() override;

	UFUNCTION()
	virtual void PostLogin(APlayerController* NewPlayer) override;

	UFUNCTION()
	virtual void NotifyLogout(const APlayerController* PC) override;

	UFUNCTION()
	void OnAgonesSuccessful(const FConnectedResponse& Response);

	UFUNCTION()
	void OnAgonesError(const FAgonesError& Error);
};
