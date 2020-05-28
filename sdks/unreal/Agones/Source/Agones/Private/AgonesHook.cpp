// Copyright 2019 Google LLC All Rights Reserved.
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

#include "AgonesHook.h"
#include "AgonesSettings.h"
#include "HttpRetrySystem.h"
#include "Dom/JsonObject.h"
#include "JsonObjectConverter.h"
#include "Serialization/JsonReader.h"
#include "GenericPlatform/GenericPlatformMisc.h"
#include "Model/Duration.h"
#include "Model/KeyValuePair.h"
#include "Runtime/Online/HTTP/Public/Http.h"
#include "Serialization/JsonSerializer.h"

#define LOCTEXT_NAMESPACE "AgonesHook"
DEFINE_LOG_CATEGORY(LogAgonesHook);

static FString GetSidecarAddress()
{
	FString port = FPlatformMisc::GetEnvironmentVariable(TEXT("AGONES_SDK_HTTP_PORT"));
	return FString(TEXT("http://localhost:")) + (!port.IsEmpty() ? port : FString(TEXT("9358")));
}

FAgonesHook::FAgonesHook()
	: FTickableGameObject()
	, CurrentHealthTime(0.0f)
	, Settings(nullptr)
	, SidecarAddress(GetSidecarAddress())
	, ReadySuffix(FString(TEXT("/ready")))
	, HealthSuffix(FString(TEXT("/health")))
	, ShutdownSuffix(FString(TEXT("/shutdown")))
	, SetLabelSuffix(FString(TEXT("/metadata/label")))
	, SetAnnotationSuffix(FString(TEXT("/metadata/annotation")))
	, GetGameServerSuffix(FString(TEXT("/gameserver")))
	, AllocateSuffix(FString(TEXT("/allocate")))
	, ReserveSuffix(FString(TEXT("/reserve")))
{
	Settings = GetDefault<UAgonesSettings>();
	check(Settings != nullptr);

	uint32 RetryLimitCount = Settings->RequestRetryLimit;
	HttpRetryManager = MakeShared<FHttpRetrySystem::FManager>(
		FHttpRetrySystem::FRetryLimitCountSetting(RetryLimitCount),
		FHttpRetrySystem::FRetryTimeoutRelativeSecondsSetting());

	UE_LOG(LogAgonesHook, Log, TEXT("Initialized Agones Hook, Sidecar address: %s, Health Enabled: %s, Health Ping: %f, Request Retry Limit: %d, Send Ready at Startup: %s")
		, *SidecarAddress
		, (Settings->bHealthPingEnabled ? TEXT("True") : TEXT("False"))
		, Settings->HealthPingSeconds
		, Settings->RequestRetryLimit
		, (Settings->bSendReadyAtStartup ? TEXT("True") : TEXT("False")));

	if (Settings->bSendReadyAtStartup)
	{
		Ready();
	}
}

FAgonesHook::~FAgonesHook()
{
	Settings = nullptr;
}

void FAgonesHook::Tick(float DeltaTime)
{
	if (Settings->bHealthPingEnabled)
	{
		CurrentHealthTime += DeltaTime;
		if (CurrentHealthTime >= Settings->HealthPingSeconds)
		{
			Health();
			CurrentHealthTime = 0.0f;
		}
	}

	HttpRetryManager->Update();
}

bool FAgonesHook::IsTickable() const
{
	return true;
}

TStatId FAgonesHook::GetStatId() const
{
	RETURN_QUICK_DECLARE_CYCLE_STAT(FAgonesHook, STATGROUP_Tickables);
}

bool FAgonesHook::IsTickableWhenPaused() const
{
	return true;
}

void FAgonesHook::Ready()
{
	TSharedRef<IHttpRequest> Req = MakeRequest(SidecarAddress + ReadySuffix, TEXT("{}"), FHttpVerb::POST, true);
	SendRequest(Req);
}

void FAgonesHook::Health()
{
	// Health requests are sent repeatedly, don't retry if request fails.
	TSharedRef<IHttpRequest> Req = MakeRequest(SidecarAddress + HealthSuffix, TEXT("{}"), FHttpVerb::POST, false);
	SendRequest(Req);
}

void FAgonesHook::Shutdown()
{
	TSharedRef<IHttpRequest> Req = MakeRequest(SidecarAddress + ShutdownSuffix, TEXT("{}"), FHttpVerb::POST, true);
	SendRequest(Req);
}

void FAgonesHook::SetLabel(const FString& Key, const FString& Value)
{
	FKeyValuePair Label = { Key, Value };
	FString Json;
	if (!FJsonObjectConverter::UStructToJsonObjectString(Label, Json))
	{
		UE_LOG(LogAgonesHook, Error, TEXT("Failed to set label, error serializing key-value pair (%s: %s)"), *Key, *Value);
		return;
	}

	TSharedRef<IHttpRequest> Req = MakeRequest(SidecarAddress + SetLabelSuffix, Json, FHttpVerb::PUT, true);
	SendRequest(Req);
}

void FAgonesHook::SetAnnotation(const FString& Key, const FString& Value)
{
	FKeyValuePair Annotation = { Key, Value };
	FString Json;
	if (!FJsonObjectConverter::UStructToJsonObjectString(Annotation, Json))
	{
		UE_LOG(LogAgonesHook, Error, TEXT("Failed to set annotation, error serializing key-value pair (%s: %s)"), *Key, *Value);
		return;
	}

	TSharedRef<IHttpRequest> Req = MakeRequest(SidecarAddress + SetAnnotationSuffix, Json, FHttpVerb::PUT, true);
	SendRequest(Req);
}

void FAgonesHook::GetGameServer(const FGameServerRequestCompleteDelegate& Delegate)
{
	TSharedRef<IHttpRequest> Req = MakeRequest(SidecarAddress + GetGameServerSuffix, TEXT(""), FHttpVerb::GET, true);
	Req->OnProcessRequestComplete().BindLambda([&Delegate](FHttpRequestPtr Request, FHttpResponsePtr Response, bool bWasSuccessful)
	{
		TSharedPtr<FGameServer> GameServer;
		if (!bWasSuccessful)
		{
			UE_LOG(LogAgonesHook, Error, TEXT("Failed to request game server details"));
			Delegate.ExecuteIfBound(GameServer, false);
			return;
		}
		
		if (!EHttpResponseCodes::IsOk(Response->GetResponseCode()))
		{
			UE_LOG(LogAgonesHook, Error, TEXT("Failed to get game server details (ResponseCode: %s )"), *FString::FromInt(Response->GetResponseCode()));
			Delegate.ExecuteIfBound(GameServer, false);
			return;
		}

		FString Json = Response->GetContentAsString();
		TSharedPtr<FJsonObject> JsonObject;
		TSharedRef<TJsonReader<>> JsonReader = TJsonReaderFactory<>::Create(Json);
		if (!FJsonSerializer::Deserialize(JsonReader, JsonObject) || !JsonObject.IsValid())
		{
			UE_LOG(LogAgonesHook, Error, TEXT("Failed to parse GetGameServer response json: %s"), *Json);
			Delegate.ExecuteIfBound(GameServer, false);
			return;
		}

		GameServer = MakeShared<FGameServer>(FGameServer(JsonObject));
		Delegate.ExecuteIfBound(GameServer, true);
	});
	SendRequest(Req);
}

void FAgonesHook::Allocate()
{
	TSharedRef<IHttpRequest> Req = MakeRequest(SidecarAddress + AllocateSuffix, TEXT("{}"), FHttpVerb::POST, true);
	SendRequest(Req);
}

void FAgonesHook::Reserve(const int64 Seconds)
{
	FDuration Duration = { Seconds };
	FString Json;
	if (!FJsonObjectConverter::UStructToJsonObjectString(Duration, Json))
	{
		UE_LOG(LogAgonesHook, Error, TEXT("Failed to send reserve request, error serializing duration (%d)"), Seconds);
		return;
	}

	TSharedRef<IHttpRequest> Req = MakeRequest(SidecarAddress + ReserveSuffix, Json, FHttpVerb::POST, true);
	SendRequest(Req);
}

TSharedRef<IHttpRequest> FAgonesHook::MakeRequest(const FString& URL, const FString& JsonContent, const FHttpVerb Verb, const bool bRetryOnFailure)
{
	TSharedRef<IHttpRequest> Req = bRetryOnFailure
		? HttpRetryManager->CreateRequest()
		: FHttpModule::Get().CreateRequest();

	Req->SetURL(URL);
	Req->SetVerb(Verb.ToString());
	Req->SetHeader(TEXT("Content-Type"), TEXT("application/json"));
	Req->SetContentAsString(JsonContent);
	return Req;
}

TSharedRef<IHttpRequest> FAgonesHook::SendRequest(TSharedRef<IHttpRequest> Req)
{
	bool bSuccess = Req->ProcessRequest();
	FString URL = Req->GetURL();
	if (bSuccess)
	{
		UE_LOG(LogAgonesHook, Verbose, TEXT("Send: %s"), *URL);
	}
	else
	{
		UE_LOG(LogAgonesHook, Error, TEXT("Failed sending: %s"), *URL);
	}

	return Req;
}

#undef LOCTEXT_NAMESPACE
