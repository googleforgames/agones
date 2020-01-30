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
#include "GameServerObjectMeta.h"
#include "GameServerSpec.h"
#include "GameServerStatus.h"
#include "JsonObject.h"
#include "GameServer.generated.h"

USTRUCT()
struct AGONES_API FGameServer
{
	GENERATED_BODY()

	UPROPERTY()
	FObjectMeta ObjectMeta;

	UPROPERTY()
	FSpec Spec;

	UPROPERTY()
	FStatus Status;

	/** Default constructor, no initialization */
	FGameServer()
	{}

	FGameServer(TSharedPtr<FJsonObject> JsonObject)
	{
		const TSharedPtr<FJsonObject>* ObjectMetaJsonObject;
		if (JsonObject->TryGetObjectField(TEXT("object_meta"), ObjectMetaJsonObject))
		{
			ObjectMeta = FObjectMeta(*ObjectMetaJsonObject);
		}
		const TSharedPtr<FJsonObject>* SpecJsonObject;
		if (JsonObject->TryGetObjectField(TEXT("spec"), SpecJsonObject))
		{
			Spec = FSpec(*SpecJsonObject);
		}
		const TSharedPtr<FJsonObject>* StatusJsonObject;
		if (JsonObject->TryGetObjectField(TEXT("status"), StatusJsonObject))
		{
			Status = FStatus(*StatusJsonObject);
		}
	}
};
