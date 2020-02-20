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

#pragma once

#include "CoreMinimal.h"
#include "HttpRetrySystem.h"
#include "Model/GameServer.h"
#include "Tickable.h"

DECLARE_LOG_CATEGORY_EXTERN(LogAgonesHook, Verbose, All);

/**
 * Delegate called when a GameServer request completes.
 *
 * @param GameServer - game server received from request to the Agones sidecar if successful
 * @param bWasSuccessful - indicates whether or not the request was successful
 */
DECLARE_DELEGATE_TwoParams(FGameServerRequestCompleteDelegate, TSharedPtr<FGameServer> /*GameServer*/, bool /*bWasSuccessful*/);

class FHttpVerb
{
public:
	enum FVerb
	{
		GET,
		POST,
		PUT
	};

	FHttpVerb(FVerb Verb)
		: Verb(Verb)
	{};

	FString ToString() const
	{
		switch (Verb)
		{
		case GET:
			return TEXT("GET");
		case POST:
			return TEXT("POST");
		case PUT:
			return TEXT("PUT");
		}

		return TEXT("");
	}

private:
	const FVerb Verb;
};

class AGONES_API FAgonesHook : public FTickableGameObject
{
public:

	/** Default constructor */
	FAgonesHook();

	/** Deconstructor */
	~FAgonesHook();

	// FTickableObjectBase interface
	virtual void Tick(float DeltaTime) override;
	virtual bool IsTickable() const override;
	virtual TStatId GetStatId() const override;
	// End FTickableObjectBase interface

	// FTickableGameObject interface
	virtual bool IsTickableWhenPaused() const override;
	// End FTickableGameObject interface

	/** Sends ready request to sidecar **/
	void Ready();
	/** Sends health ping request to sidecar **/
	void Health();
	/** Sends shutdown request to sidecar **/
	void Shutdown();
	/** Sends set label request to sidecar **/
	void SetLabel(const FString& Key, const FString& Value);
	/** Sends set annotation request to sidecar **/
	void SetAnnotation(const FString& Key, const FString& Value);
	/** Retrieve the GameServer details from the sidecar */
	void GetGameServer(const FGameServerRequestCompleteDelegate& Delegate);
	/** Sends a request to allocate the GameServer **/
	void Allocate();
	/** Sends a request to mark the GameServer as reserved for the specified duration */
	void Reserve(const int64 Seconds);

private:

	/** Helper function to create requests */
	TSharedRef<class IHttpRequest> MakeRequest(const FString& URL, const FString& JsonContent, const FHttpVerb Verb, const bool bRetryOnFailure);
	/** Helper function to send requests with default debug output */
	TSharedRef<class IHttpRequest> SendRequest(const FString& URL, const FString& JsonContent, const FHttpVerb Verb, const bool bRetryOnFailure);
	/** Retry manager to retry failed http requests */
	TSharedPtr<class FHttpRetrySystem::FManager> HttpRetryManager;

	/** Time since last health ping */
	float CurrentHealthTime;

	/** Agones settings */
	const class UAgonesSettings* Settings;

	const FString SidecarAddress;
	const FString ReadySuffix;
	const FString HealthSuffix;
	const FString ShutdownSuffix;
	const FString SetLabelSuffix;
	const FString SetAnnotationSuffix;
	const FString GetGameServerSuffix;
	const FString AllocateSuffix;
	const FString ReserveSuffix;
};
