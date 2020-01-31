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

#pragma once

#include "CoreMinimal.h"
#include "JsonObject.h"
#include "SpecHealth.generated.h"

USTRUCT()
struct AGONES_API FHealth
{
	GENERATED_BODY()

	UPROPERTY()
	bool bDisabled = false;

	UPROPERTY()
	int32 PeriodSeconds = 0;

	UPROPERTY()
	int32 FailureThreshold = 0;

	UPROPERTY()
	int32 InitialDelaySeconds = 0;

	/** Default constructor, no initialization */
	FHealth()
	{}

	FHealth(TSharedPtr<FJsonObject> JsonObject)
	{
		JsonObject->TryGetBoolField(TEXT("disabled"), bDisabled);
		JsonObject->TryGetNumberField(TEXT("period_seconds"), PeriodSeconds);
		JsonObject->TryGetNumberField(TEXT("failure_threshold"), FailureThreshold);
		JsonObject->TryGetNumberField(TEXT("initial_delay_seconds"), InitialDelaySeconds);
	}
};
