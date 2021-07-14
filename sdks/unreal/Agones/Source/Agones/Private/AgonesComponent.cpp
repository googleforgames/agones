// Copyright 2020 Google LLC All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

#include "AgonesComponent.h"

#include "Engine/World.h"
#include "HttpModule.h"
#include "Interfaces/IHttpResponse.h"
#include "JsonUtilities/Public/JsonObjectConverter.h"
#include "TimerManager.h"
#include "WebSockets/Public/IWebSocket.h"
#include "WebSockets/Public/WebSocketsModule.h"

UAgonesComponent::UAgonesComponent()
{
	PrimaryComponentTick.bCanEverTick = false;
}

void UAgonesComponent::BeginPlay()
{
	Super::BeginPlay();
	HealthPing(HealthRateSeconds);

	if (bDisableAutoConnect)
	{
		return;
	}
	Connect();
}

void UAgonesComponent::EndPlay(const EEndPlayReason::Type EndPlayReason)
{
	Super::EndPlay(EndPlayReason);

	const UWorld* World = GetWorld();
	if (World != nullptr)
	{
		World->GetTimerManager().ClearTimer(ConnectDelTimerHandle);
		World->GetTimerManager().ClearTimer(HealthTimerHandler);
	    World->GetTimerManager().ClearTimer(EnsureWebSocketTimerHandler);
	}

    if (WatchWebSocket != nullptr && WatchWebSocket->IsConnected())
    {
        WatchWebSocket->Close();
    }
}

FHttpRequestRef UAgonesComponent::BuildAgonesRequest(const FString Path, const FHttpVerb Verb, const FString Content)
{
	FHttpModule* Http = &FHttpModule::Get();
	FHttpRequestRef Request = Http->CreateRequest();

	TArray<FStringFormatArg> FormatArgs;
	FormatArgs.Add(HttpPort);
	FormatArgs.Add(Path);

	Request->SetURL(FString::Format(TEXT("http://localhost:{0}/{1}"), FormatArgs));
	Request->SetVerb(Verb.ToString());
	Request->SetHeader(TEXT("Content-Type"), TEXT("application/json"));
	Request->SetHeader(TEXT("User-Agent"), TEXT("X-UnrealEngine-Agent"));
	Request->SetHeader(TEXT("Accepts"), TEXT("application/json"));
	Request->SetContentAsString(Content);
	return Request;
}

void UAgonesComponent::HealthPing(const float RateSeconds)
{
	if (RateSeconds <= 0.0f)
	{
		return;
	}

	FTimerDelegate TimerDel;
	TimerDel.BindUObject(this, &UAgonesComponent::Health, FHealthDelegate(), FAgonesErrorDelegate());
	GetWorld()->GetTimerManager().ClearTimer(HealthTimerHandler);
	GetWorld()->GetTimerManager().SetTimer(HealthTimerHandler, TimerDel, RateSeconds, true);
}

void UAgonesComponent::Connect()
{
	FGameServerDelegate SuccessDel;
	SuccessDel.BindUFunction(this, FName("ConnectSuccess"));
	FTimerDelegate ConnectDel;
	ConnectDel.BindUObject(this, &UAgonesComponent::GameServer, SuccessDel, FAgonesErrorDelegate());
	GetWorld()->GetTimerManager().ClearTimer(ConnectDelTimerHandle);
	GetWorld()->GetTimerManager().SetTimer(ConnectDelTimerHandle, ConnectDel, 5.f, true);
}

void UAgonesComponent::ConnectSuccess(const FGameServerResponse GameServerResponse)
{
	GetWorld()->GetTimerManager().ClearTimer(ConnectDelTimerHandle);
	Ready({}, {});
	ConnectedDelegate.Broadcast(GameServerResponse);
}

void UAgonesComponent::Ready(const FReadyDelegate SuccessDelegate, const FAgonesErrorDelegate ErrorDelegate)
{
	FHttpRequestRef Request = BuildAgonesRequest("ready");
	Request->OnProcessRequestComplete().BindWeakLambda(this,
		[SuccessDelegate, ErrorDelegate](FHttpRequestPtr HttpRequest, FHttpResponsePtr HttpResponse, const bool bSucceeded) {
			if (!bSucceeded)
			{
				ErrorDelegate.ExecuteIfBound({});
				return;
			}
			if (!EHttpResponseCodes::IsOk(HttpResponse->GetResponseCode()))
			{
				ErrorDelegate.ExecuteIfBound(
					{FString::Format(TEXT("Error Code - {0}"), {FString::FromInt(HttpResponse->GetResponseCode())})});
				return;
			}

			SuccessDelegate.ExecuteIfBound({});
		});
	Request->ProcessRequest();
}

void UAgonesComponent::GameServer(const FGameServerDelegate SuccessDelegate, const FAgonesErrorDelegate ErrorDelegate)
{
	FHttpRequestRef Request = BuildAgonesRequest("gameserver", FHttpVerb::Get, "");
	Request->OnProcessRequestComplete().BindWeakLambda(this,
		[SuccessDelegate, ErrorDelegate](FHttpRequestPtr HttpRequest, const FHttpResponsePtr HttpResponse, const bool bSucceeded) {
			if (!bSucceeded)
			{
				ErrorDelegate.ExecuteIfBound({"Unsuccessful Call"});
				return;
			}

			if (!EHttpResponseCodes::IsOk(HttpResponse->GetResponseCode()))
			{
				ErrorDelegate.ExecuteIfBound(
					{FString::Format(TEXT("Error Code - {0}"), {FString::FromInt(HttpResponse->GetResponseCode())})});
				return;
			}

			const FString Json = HttpResponse->GetContentAsString();
			TSharedPtr<FJsonObject> JsonObject;
			const TSharedRef<TJsonReader<>> JsonReader = TJsonReaderFactory<>::Create(Json);
			if (!FJsonSerializer::Deserialize(JsonReader, JsonObject) || !JsonObject.IsValid())
			{
				ErrorDelegate.ExecuteIfBound({FString::Format(TEXT("Failed to parse JSON - {0}"), {*Json})});
				ErrorDelegate.ExecuteIfBound({});
				return;
			}
			SuccessDelegate.ExecuteIfBound(FGameServerResponse(JsonObject));
		});
	Request->ProcessRequest();
}

void UAgonesComponent::EnsureWebSocketConnection()
{
    if (WatchWebSocket == nullptr)
    {
        if (!FModuleManager::LoadModulePtr<FWebSocketsModule>(TEXT("WebSockets")))
        {
            return;
        }

        TMap<FString, FString> Headers;

        // Make up a WebSocket-Key value. It can be anything!
        Headers.Add(TEXT("Sec-WebSocket-Key"), FGuid::NewGuid().ToString(EGuidFormats::Short));
        Headers.Add(TEXT("Sec-WebSocket-Version"), TEXT("13"));
        Headers.Add(TEXT("User-Agent"), TEXT("X-UnrealEngine-Agent"));

        // Unreal WebSockets are not able to do DNS resolution for localhost for some reason
        // so this is using the IPv4 Loopback Address instead.
        WatchWebSocket = FWebSocketsModule::Get().CreateWebSocket(
            FString::Format(TEXT("ws://127.0.0.1:{0}/watch/gameserver"), {*HttpPort}), TEXT(""));

        WatchWebSocket->OnRawMessage().AddUObject(this, &UAgonesComponent::HandleWatchMessage);
    }

    if (WatchWebSocket != nullptr)
    {
        if (!WatchWebSocket->IsConnected())
        {
            WatchWebSocket->Connect();
        }

        // Only start the timer if there is a websocket to check.
        // This timer has nothing to do with health and only matters if the agent is somehow
        // restarted, which would be a failure condition in normal operation.
        if (!EnsureWebSocketTimerHandler.IsValid())
        {
            FTimerDelegate TimerDel;
            TimerDel.BindUObject(this, &UAgonesComponent::EnsureWebSocketConnection);
            GetWorld()->GetTimerManager().SetTimer(
                EnsureWebSocketTimerHandler, TimerDel, 15.0f, true);
        }
    }
}

void UAgonesComponent::WatchGameServer(const FGameServerDelegate WatchDelegate)
{
    WatchGameServerCallbacks.Add(WatchDelegate);
    EnsureWebSocketConnection();
}

 void UAgonesComponent::DeserializeAndBroadcastWatch(FString const& JsonString)
{
    TSharedRef<TJsonReader<TCHAR>> const JsonReader = TJsonReaderFactory<TCHAR>::Create(JsonString);

    TSharedPtr<FJsonObject> JsonObject;
    const TSharedPtr<FJsonObject>* ResultObject = nullptr;

    if (!FJsonSerializer::Deserialize(JsonReader, JsonObject) ||
        !JsonObject.IsValid() ||
        !JsonObject->TryGetObjectField(TEXT("result"), ResultObject) ||
        !ResultObject->IsValid())
    {
        return;
    }

    FGameServerResponse const Result = FGameServerResponse(*ResultObject);
    for (FGameServerDelegate const& Callback : WatchGameServerCallbacks)
    {
        if (Callback.IsBound())
        {
            Callback.Execute(Result);
        }
    }
}

void UAgonesComponent::HandleWatchMessage(const void* Data, SIZE_T Size, SIZE_T BytesRemaining)
{
    if (BytesRemaining > 0)
    {
        WatchMessageBuffer.Append(UTF8_TO_TCHAR(static_cast<const UTF8CHAR*>(Data)), Size);
        return;
    }

    FString const Message = FString(Size, UTF8_TO_TCHAR(static_cast<const UTF8CHAR*>(Data)));

    // If the LHS of FString + is empty, it just uses the RHS directly so there's no copy here with an empty buffer.
    DeserializeAndBroadcastWatch(WatchMessageBuffer + Message);

    // Faster to check and then empty vs blindly emptying - normal case is that the buffer is already empty
    if (!WatchMessageBuffer.IsEmpty())
    {
        WatchMessageBuffer.Empty();
    }
}

void UAgonesComponent::SetLabel(
	FString& Key, FString& Value, const FSetLabelDelegate SuccessDelegate, const FAgonesErrorDelegate ErrorDelegate)
{
	const FKeyValuePair Label = {Key, Value};
	FString Json;
	if (!FJsonObjectConverter::UStructToJsonObjectString(Label, Json))
	{
		TArray<FStringFormatArg> FormatArgs;
		FormatArgs.Add(Key);
		FormatArgs.Add(Value);
		ErrorDelegate.ExecuteIfBound({FString::Format(TEXT("error serializing key-value pair ({0}: {1}})"), FormatArgs)});
		return;
	}

	FHttpRequestRef Request = BuildAgonesRequest("metadata/label", FHttpVerb::Put, Json);
	Request->OnProcessRequestComplete().BindWeakLambda(this,
		[SuccessDelegate, ErrorDelegate](FHttpRequestPtr HttpRequest, const FHttpResponsePtr HttpResponse, const bool bSucceeded) {
			if (!bSucceeded)
			{
				ErrorDelegate.ExecuteIfBound({"Unsuccessful Call"});
				return;
			}

			if (!EHttpResponseCodes::IsOk(HttpResponse->GetResponseCode()))
			{
				ErrorDelegate.ExecuteIfBound(
					{FString::Format(TEXT("Error Code - {0}"), {FString::FromInt(HttpResponse->GetResponseCode())})});
				return;
			}

			SuccessDelegate.ExecuteIfBound({});
		});
	Request->ProcessRequest();
}

void UAgonesComponent::Health(const FHealthDelegate SuccessDelegate, const FAgonesErrorDelegate ErrorDelegate)
{
	FHttpRequestRef Request = BuildAgonesRequest("health");
	Request->OnProcessRequestComplete().BindWeakLambda(this,
		[SuccessDelegate, ErrorDelegate](FHttpRequestPtr HttpRequest, const FHttpResponsePtr HttpResponse, const bool bSucceeded) {
			if (!bSucceeded)
			{
				ErrorDelegate.ExecuteIfBound({"Unsuccessful Call"});
				return;
			}

			if (!EHttpResponseCodes::IsOk(HttpResponse->GetResponseCode()))
			{
				ErrorDelegate.ExecuteIfBound(
					{FString::Format(TEXT("Error Code - {0}"), {FString::FromInt(HttpResponse->GetResponseCode())})});
				return;
			}

			SuccessDelegate.ExecuteIfBound({});
		});
	Request->ProcessRequest();
}

void UAgonesComponent::Shutdown(const FShutdownDelegate SuccessDelegate, const FAgonesErrorDelegate ErrorDelegate)
{
	FHttpRequestRef Request = BuildAgonesRequest("shutdown");
	Request->OnProcessRequestComplete().BindWeakLambda(this,
		[SuccessDelegate, ErrorDelegate](FHttpRequestPtr HttpRequest, FHttpResponsePtr HttpResponse, const bool bSucceeded) {
			if (!bSucceeded)
			{
				ErrorDelegate.ExecuteIfBound({"Unsuccessful Call"});
				return;
			}
			if (!EHttpResponseCodes::IsOk(HttpResponse->GetResponseCode()))
			{
				ErrorDelegate.ExecuteIfBound(
					{FString::Format(TEXT("Error Code - {0}"), {FString::FromInt(HttpResponse->GetResponseCode())})});
				return;
			}

			SuccessDelegate.ExecuteIfBound({});
		});
	Request->ProcessRequest();
}

void UAgonesComponent::SetAnnotation(
	FString& Key, FString& Value, const FSetAnnotationDelegate SuccessDelegate, const FAgonesErrorDelegate ErrorDelegate)
{
	const FKeyValuePair Label = {Key, Value};
	FString Json;
	if (!FJsonObjectConverter::UStructToJsonObjectString(Label, Json))
	{
		TArray<FStringFormatArg> FormatArgs;
		FormatArgs.Add(Key);
		FormatArgs.Add(Value);
		ErrorDelegate.ExecuteIfBound({FString::Format(TEXT("error serializing key-value pair ({0}: {1}})"), FormatArgs)});
		return;
	}

	FHttpRequestRef Request = BuildAgonesRequest("metadata/annotation", FHttpVerb::Put, Json);
	Request->OnProcessRequestComplete().BindWeakLambda(this,
		[SuccessDelegate, ErrorDelegate](FHttpRequestPtr HttpRequest, FHttpResponsePtr HttpResponse, const bool bSucceeded) {
			if (!bSucceeded)
			{
				ErrorDelegate.ExecuteIfBound({"Unsuccessful Call"});
				return;
			}
			if (!EHttpResponseCodes::IsOk(HttpResponse->GetResponseCode()))
			{
				ErrorDelegate.ExecuteIfBound(
					{FString::Format(TEXT("Error Code - {0}"), {FString::FromInt(HttpResponse->GetResponseCode())})});
				return;
			}

			SuccessDelegate.ExecuteIfBound({});
		});
	Request->ProcessRequest();
}

void UAgonesComponent::Allocate(const FAllocateDelegate SuccessDelegate, const FAgonesErrorDelegate ErrorDelegate)
{
	FHttpRequestRef Request = BuildAgonesRequest("allocate");
	Request->OnProcessRequestComplete().BindWeakLambda(this,
		[SuccessDelegate, ErrorDelegate](FHttpRequestPtr HttpRequest, FHttpResponsePtr HttpResponse, const bool bSucceeded) {
			if (!bSucceeded)
			{
				ErrorDelegate.ExecuteIfBound({"Unsuccessful Call"});
				return;
			}
			if (!EHttpResponseCodes::IsOk(HttpResponse->GetResponseCode()))
			{
				ErrorDelegate.ExecuteIfBound(
					{FString::Format(TEXT("Error Code - {0}"), {FString::FromInt(HttpResponse->GetResponseCode())})});
				return;
			}

			SuccessDelegate.ExecuteIfBound({});
		});
	Request->ProcessRequest();
}

void UAgonesComponent::Reserve(
	const int64 Seconds, const FReserveDelegate SuccessDelegate, const FAgonesErrorDelegate ErrorDelegate)
{
	const FDuration Duration = {Seconds};
	FString Json;
	if (!FJsonObjectConverter::UStructToJsonObjectString(Duration, Json))
	{
		ErrorDelegate.ExecuteIfBound({TEXT("Failed to serializing request")});
		return;
	}

	FHttpRequestRef Request = BuildAgonesRequest("reserve", FHttpVerb::Post, Json);
	Request->OnProcessRequestComplete().BindWeakLambda(this,
		[SuccessDelegate, ErrorDelegate](FHttpRequestPtr HttpRequest, FHttpResponsePtr HttpResponse, const bool bSucceeded) {
			if (!bSucceeded)
			{
				ErrorDelegate.ExecuteIfBound({"Unsuccessful Call"});
				return;
			}
			if (!EHttpResponseCodes::IsOk(HttpResponse->GetResponseCode()))
			{
				ErrorDelegate.ExecuteIfBound(
					{FString::Format(TEXT("Error Code - {0}"), {FString::FromInt(HttpResponse->GetResponseCode())})});
				return;
			}

			SuccessDelegate.ExecuteIfBound({});
		});
	Request->ProcessRequest();
}

void UAgonesComponent::PlayerConnect(
	const FString PlayerId, const FPlayerConnectDelegate SuccessDelegate, const FAgonesErrorDelegate ErrorDelegate)
{
	const FAgonesPlayer Player = {PlayerId};
	FString Json;
	if (!FJsonObjectConverter::UStructToJsonObjectString(Player, Json))
	{
		ErrorDelegate.ExecuteIfBound({TEXT("Failed to serializing request")});
		return;
	}

	// TODO(dom) - look at JSON encoding in UE4.
	Json = Json.Replace(TEXT("playerId"), TEXT("playerID"));

	FHttpRequestRef Request = BuildAgonesRequest("alpha/player/connect", FHttpVerb::Post, Json);
	Request->OnProcessRequestComplete().BindWeakLambda(this,
		[SuccessDelegate, ErrorDelegate](FHttpRequestPtr HttpRequest, const FHttpResponsePtr HttpResponse, const bool bSucceeded) {
			if (!bSucceeded)
			{
				ErrorDelegate.ExecuteIfBound({"Unsuccessful Call"});
				return;
			}

			if (!EHttpResponseCodes::IsOk(HttpResponse->GetResponseCode()))
			{
				ErrorDelegate.ExecuteIfBound(
					{FString::Format(TEXT("Error Code - {0}"), {FString::FromInt(HttpResponse->GetResponseCode())})});
				return;
			}

			const FString Json = HttpResponse->GetContentAsString();
			TSharedPtr<FJsonObject> JsonObject;
			const TSharedRef<TJsonReader<>> JsonReader = TJsonReaderFactory<>::Create(Json);
			if (!FJsonSerializer::Deserialize(JsonReader, JsonObject) || !JsonObject.IsValid())
			{
				ErrorDelegate.ExecuteIfBound({FString::Format(TEXT("Failed to parse response - {0}"), {*Json})});
				return;
			}

			SuccessDelegate.ExecuteIfBound(FConnectedResponse(JsonObject));
		});
	Request->ProcessRequest();
}

void UAgonesComponent::PlayerDisconnect(
	const FString PlayerId, const FPlayerDisconnectDelegate SuccessDelegate, const FAgonesErrorDelegate ErrorDelegate)
{
	const FAgonesPlayer Player = {PlayerId};
	FString Json;
	if (!FJsonObjectConverter::UStructToJsonObjectString(Player, Json))
	{
		ErrorDelegate.ExecuteIfBound({TEXT("Failed to serializing request")});
		return;
	}

	// TODO(dom) - look at JSON encoding in UE4.
	Json = Json.Replace(TEXT("playerId"), TEXT("playerID"));

	FHttpRequestRef Request = BuildAgonesRequest("alpha/player/disconnect", FHttpVerb::Post, Json);
	Request->OnProcessRequestComplete().BindWeakLambda(this,
		[SuccessDelegate, ErrorDelegate](FHttpRequestPtr HttpRequest, const FHttpResponsePtr HttpResponse, const bool bSucceeded) {
			if (!bSucceeded)
			{
				ErrorDelegate.ExecuteIfBound({"Unsuccessful Call"});
				return;
			}

			if (!EHttpResponseCodes::IsOk(HttpResponse->GetResponseCode()))
			{
				ErrorDelegate.ExecuteIfBound(
					{FString::Format(TEXT("Error Code - {0}"), {FString::FromInt(HttpResponse->GetResponseCode())})});
				return;
			}

			const FString Json = HttpResponse->GetContentAsString();
			TSharedPtr<FJsonObject> JsonObject;
			const TSharedRef<TJsonReader<>> JsonReader = TJsonReaderFactory<>::Create(Json);
			if (!FJsonSerializer::Deserialize(JsonReader, JsonObject) || !JsonObject.IsValid())
			{
				ErrorDelegate.ExecuteIfBound({FString::Format(TEXT("Failed to parse response - {0}"), {*Json})});
				return;
			}

			SuccessDelegate.ExecuteIfBound(FDisconnectResponse(JsonObject));
		});
	Request->ProcessRequest();
}

void UAgonesComponent::SetPlayerCapacity(
	const int64 Count, const FSetPlayerCapacityDelegate SuccessDelegate, const FAgonesErrorDelegate ErrorDelegate)
{
	const FPlayerCapacity PlayerCapacity = {Count};
	FString Json;
	if (!FJsonObjectConverter::UStructToJsonObjectString(PlayerCapacity, Json))
	{
		ErrorDelegate.ExecuteIfBound({TEXT("Failed to serializing request")});
		return;
	}

	FHttpRequestRef Request = BuildAgonesRequest("alpha/player/capacity", FHttpVerb::Post, Json);
	Request->OnProcessRequestComplete().BindWeakLambda(this,
		[SuccessDelegate, ErrorDelegate](FHttpRequestPtr HttpRequest, FHttpResponsePtr HttpResponse, const bool bSucceeded) {
			if (!bSucceeded)
			{
				ErrorDelegate.ExecuteIfBound({"Unsuccessful Call"});
				return;
			}
			if (!EHttpResponseCodes::IsOk(HttpResponse->GetResponseCode()))
			{
				ErrorDelegate.ExecuteIfBound(
					{FString::Format(TEXT("Error Code - {0}"), {FString::FromInt(HttpResponse->GetResponseCode())})});
				return;
			}

			SuccessDelegate.ExecuteIfBound({});
		});
	Request->ProcessRequest();
}

void UAgonesComponent::GetPlayerCapacity(FGetPlayerCapacityDelegate SuccessDelegate, FAgonesErrorDelegate ErrorDelegate)
{
	FHttpRequestRef Request = BuildAgonesRequest("alpha/player/capacity", FHttpVerb::Get, "");
	Request->OnProcessRequestComplete().BindWeakLambda(this,
		[SuccessDelegate, ErrorDelegate](FHttpRequestPtr HttpRequest, const FHttpResponsePtr HttpResponse, const bool bSucceeded) {
			if (!bSucceeded)
			{
				ErrorDelegate.ExecuteIfBound({"Unsuccessful Call"});
				return;
			}
			if (!EHttpResponseCodes::IsOk(HttpResponse->GetResponseCode()))
			{
				ErrorDelegate.ExecuteIfBound(
					{FString::Format(TEXT("Error Code - {0}"), {FString::FromInt(HttpResponse->GetResponseCode())})});
				return;
			}

			const FString Json = HttpResponse->GetContentAsString();
			TSharedPtr<FJsonObject> JsonObject;
			const TSharedRef<TJsonReader<>> JsonReader = TJsonReaderFactory<>::Create(Json);
			if (!FJsonSerializer::Deserialize(JsonReader, JsonObject) || !JsonObject.IsValid())
			{
				ErrorDelegate.ExecuteIfBound({FString::Format(TEXT("Failed to parse response - {0}"), {*Json})});
				return;
			}

			SuccessDelegate.ExecuteIfBound(FCountResponse(JsonObject));
		});
	Request->ProcessRequest();
}

void UAgonesComponent::GetPlayerCount(FGetPlayerCountDelegate SuccessDelegate, FAgonesErrorDelegate ErrorDelegate)
{
	FHttpRequestRef Request = BuildAgonesRequest("alpha/player/count", FHttpVerb::Get, "");
	Request->OnProcessRequestComplete().BindWeakLambda(this,
		[SuccessDelegate, ErrorDelegate](FHttpRequestPtr HttpRequest, const FHttpResponsePtr HttpResponse, const bool bSucceeded) {
			if (!bSucceeded)
			{
				ErrorDelegate.ExecuteIfBound({"Unsuccessful Call"});
				return;
			}

			if (!EHttpResponseCodes::IsOk(HttpResponse->GetResponseCode()))
			{
				ErrorDelegate.ExecuteIfBound(
					{FString::Format(TEXT("Error Code - {0}"), {FString::FromInt(HttpResponse->GetResponseCode())})});
				return;
			}

			const FString Json = HttpResponse->GetContentAsString();
			TSharedPtr<FJsonObject> JsonObject;
			const TSharedRef<TJsonReader<>> JsonReader = TJsonReaderFactory<>::Create(Json);
			if (!FJsonSerializer::Deserialize(JsonReader, JsonObject) || !JsonObject.IsValid())
			{
				ErrorDelegate.ExecuteIfBound({FString::Format(TEXT("Failed to parse response - {0}"), {*Json})});
				return;
			}

			SuccessDelegate.ExecuteIfBound(FCountResponse(JsonObject));
		});
	Request->ProcessRequest();
}

void UAgonesComponent::IsPlayerConnected(
	const FString PlayerId, const FIsPlayerConnectedDelegate SuccessDelegate, const FAgonesErrorDelegate ErrorDelegate)
{
	FHttpRequestRef Request =
		BuildAgonesRequest(FString::Format(TEXT("alpha/player/connected/{0}"), {*PlayerId}), FHttpVerb::Get, "");
	Request->OnProcessRequestComplete().BindWeakLambda(this,
		[SuccessDelegate, ErrorDelegate](FHttpRequestPtr HttpRequest, const FHttpResponsePtr HttpResponse, const bool bSucceeded) {
			if (!bSucceeded)
			{
				ErrorDelegate.ExecuteIfBound({"Unsuccessful Call"});
				return;
			}

			if (!EHttpResponseCodes::IsOk(HttpResponse->GetResponseCode()))
			{
				ErrorDelegate.ExecuteIfBound(
					{FString::Format(TEXT("Error Code - {0}"), {FString::FromInt(HttpResponse->GetResponseCode())})});
				return;
			}

			const FString Json = HttpResponse->GetContentAsString();
			TSharedPtr<FJsonObject> JsonObject;
			const TSharedRef<TJsonReader<>> JsonReader = TJsonReaderFactory<>::Create(Json);
			if (!FJsonSerializer::Deserialize(JsonReader, JsonObject) || !JsonObject.IsValid())
			{
				ErrorDelegate.ExecuteIfBound({FString::Format(TEXT("Failed to parse response - {0}"), {*Json})});
				return;
			}

			SuccessDelegate.ExecuteIfBound(FConnectedResponse(JsonObject));
		});
	Request->ProcessRequest();
}

void UAgonesComponent::GetConnectedPlayers(
	const FGetConnectedPlayersDelegate SuccessDelegate, const FAgonesErrorDelegate ErrorDelegate)
{
	FHttpRequestRef Request = BuildAgonesRequest("alpha/player/connected/{0}", FHttpVerb::Get, "");
	Request->OnProcessRequestComplete().BindWeakLambda(this,
		[SuccessDelegate, ErrorDelegate](FHttpRequestPtr HttpRequest, const FHttpResponsePtr HttpResponse, const bool bSucceeded) {
			if (!bSucceeded)
			{
				ErrorDelegate.ExecuteIfBound({"Unsuccessful Call"});
				return;
			}

			if (!EHttpResponseCodes::IsOk(HttpResponse->GetResponseCode()))
			{
				ErrorDelegate.ExecuteIfBound(
					{FString::Format(TEXT("Error Code - {0}"), {FString::FromInt(HttpResponse->GetResponseCode())})});
				return;
			}

			const FString Json = HttpResponse->GetContentAsString();
			TSharedPtr<FJsonObject> JsonObject;
			const TSharedRef<TJsonReader<>> JsonReader = TJsonReaderFactory<>::Create(Json);
			if (!FJsonSerializer::Deserialize(JsonReader, JsonObject) || !JsonObject.IsValid())
			{
				ErrorDelegate.ExecuteIfBound({FString::Format(TEXT("Failed to parse response - {0}"), {*Json})});
				return;
			}

			SuccessDelegate.ExecuteIfBound(FConnectedPlayersResponse(JsonObject));
		});
	Request->ProcessRequest();
}
