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
#include "SpecHealth.h"
#include "GameServerSpec.generated.h"

USTRUCT()
struct AGONES_API FSpec
{
	GENERATED_BODY()

	UPROPERTY()
	FHealth Health;

	/** Default constructor, no initialization */
	FSpec()
	{}

	FSpec(TSharedPtr<FJsonObject> JsonObject)
	{
		const TSharedPtr<FJsonObject>* HealthJsonObject;
		if (JsonObject->TryGetObjectField(TEXT("health"), HealthJsonObject))
		{
			Health = FHealth(*HealthJsonObject);
		}
	}
};
