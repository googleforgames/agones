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

#pragma once

#include "CoreMinimal.h"
#include "Tickable.h"

DECLARE_LOG_CATEGORY_EXTERN(LogAgonesHook, Verbose, All);

class FAgonesHook : public FTickableGameObject
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

private:

	/** Helper function to send requests with default debug output */
	bool SendRequest(const FString& URL);

	/** Time since last health ping */
	float CurrentHealthTime;

	/** Agones settings */
	const class UAgonesSettings* Settings;

	const FString ReadySuffix;
	const FString HealthSuffix;
	const FString ShutdownSuffix;
};