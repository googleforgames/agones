// Copyright 2019 Google Inc. All Rights Reserved.
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
#include "Runtime/Online/HTTP/Public/Http.h"

#define LOCTEXT_NAMESPACE "AgonesHook"
DEFINE_LOG_CATEGORY(LogAgonesHook);

FAgonesHook::FAgonesHook()
	: FTickableGameObject()
	, CurrentHealthTime(0.0f)
	, Settings(nullptr)
	, ReadySuffix(FString(TEXT("/ready")))
	, HealthSuffix(FString(TEXT("/health")))
	, ShutdownSuffix(FString(TEXT("/shutdown")))
{
	Settings = GetDefault<UAgonesSettings>();
	check(Settings != nullptr);

	UE_LOG(LogAgonesHook, Log, TEXT("Initialized Agones Hook, Sidecar address: %s, Health Enabled: %s, Health Ping: %f, Debug: %s")
		, *Settings->AgonesSidecarAddress
		, (Settings->bHealthPingEnabled ? TEXT("True") : TEXT("False"))
		, Settings->HealthPingSeconds
		, (Settings->bDebugLogEnabled ? TEXT("True") : TEXT("False")));
}

FAgonesHook::~FAgonesHook()
{
	Settings = nullptr;
}

void FAgonesHook::Tick(float DeltaTime)
{
	CurrentHealthTime += DeltaTime;
	if (CurrentHealthTime >= Settings->HealthPingSeconds)
	{
		Health();
		CurrentHealthTime = 0.0f;
	}
}

bool FAgonesHook::IsTickable() const
{
	return Settings->bHealthPingEnabled;
}

TStatId FAgonesHook::GetStatId() const
{
	RETURN_QUICK_DECLARE_CYCLE_STAT(FAgonesHook, STATGROUP_Tickables);
}

bool FAgonesHook::IsTickableWhenPaused() const
{
	return true;
}

static TSharedRef<IHttpRequest> MakeRequest(const FString& URL)
{
	FHttpModule* http = &FHttpModule::Get();
	TSharedRef<IHttpRequest> req = http->CreateRequest();
	req->SetURL(URL);
	req->SetVerb("POST");
	req->SetHeader("Content-Type", "application/json");
	req->SetContentAsString("{}");
	return req;
}

void FAgonesHook::Ready()
{
	SendRequest(Settings->AgonesSidecarAddress + ReadySuffix);
}

void FAgonesHook::Health()
{
	SendRequest(Settings->AgonesSidecarAddress + HealthSuffix);
}

void FAgonesHook::Shutdown()
{
	SendRequest(Settings->AgonesSidecarAddress + ShutdownSuffix);
}


bool FAgonesHook::SendRequest(const FString& URL)
{
	TSharedRef<IHttpRequest> req = MakeRequest(URL);
	bool bSuccess = req->ProcessRequest();
	if (Settings->bDebugLogEnabled)
	{
		if (bSuccess)
		{
			UE_LOG(LogAgonesHook, Log, TEXT("Send: %s"), *URL);
		}
		else
		{
			UE_LOG(LogAgonesHook, Error, TEXT("Failed sending: %s"), *URL);
		}
	}
	return bSuccess;
}

#undef LOCTEXT_NAMESPACE