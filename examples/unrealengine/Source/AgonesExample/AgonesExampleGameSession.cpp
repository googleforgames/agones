// Fill out your copyright notice in the Description page of Project Settings.


#include "AgonesExampleGameSession.h"

#include "AgonesExampleGameMode.h"
#include "Kismet/GameplayStatics.h"

AAgonesExampleGameSession::AAgonesExampleGameSession()
{
	AAgonesExampleGameMode* GameMode = Cast<AAgonesExampleGameMode>(UGameplayStatics::GetGameMode(GetWorld()));

	if (GameMode)
	{
		AgonesSDK = GameMode->AgonesSDK;
	}
}

void AAgonesExampleGameSession::RegisterServer()
{
	AgonesSDK->SetPlayerCapacity(100, {}, AgonesErrorDelegate);
	FString Label = "map";
	FString Value = GetWorld()->GetCurrentLevel()->GetName();
	AgonesSDK->SetLabel(Label, Value, {}, AgonesErrorDelegate);
}

void AAgonesExampleGameSession::PostLogin(APlayerController* NewPlayer)
{
	AgonesSDK->PlayerConnect(NewPlayer->GetNetConnection()->PlayerId.ToString(), PlayerConnectDelegate, AgonesErrorDelegate);
}

void AAgonesExampleGameSession::NotifyLogout(const APlayerController* PC)
{
	AgonesSDK->PlayerDisconnect(PC->GetNetConnection()->PlayerId.ToString(), {}, AgonesErrorDelegate);
}

void AAgonesExampleGameSession::OnAgonesSuccessful(const FConnectedResponse& Response)
{
	UE_LOG(LogTemp, Verbose, TEXT("Agones player connection succcessful!"));
}

void AAgonesExampleGameSession::OnAgonesError(const FAgonesError& Error)
{
	UE_LOG(LogTemp, Error, TEXT("Agones Error: %s"), *(Error.ErrorMessage));
}
