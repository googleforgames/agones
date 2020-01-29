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
#include "StatusPort.h"
#include "GameServerStatus.generated.h"

USTRUCT()
struct AGONES_API FStatus
{
	GENERATED_BODY()

	UPROPERTY()
	FString State;

	UPROPERTY()
	FString Address;

	UPROPERTY()
	TArray<FPort> Ports;

	/** Default constructor, no initialization */
	FStatus()
	{}

	FStatus(TSharedPtr<FJsonObject> JsonObject)
	{
		JsonObject->TryGetStringField(TEXT("state"), State);
		JsonObject->TryGetStringField(TEXT("address"), Address);
		const TArray<TSharedPtr<FJsonValue>>* PortsArray;
		if (JsonObject->TryGetArrayField(TEXT("ports"), PortsArray))
		{
			int32 ArrLen = PortsArray->Num();
			for (int32 i = 0; i < ArrLen; ++i)
			{
				const TSharedPtr<FJsonValue>& PortItem = (*PortsArray)[i];
				if (PortItem.IsValid() && !PortItem->IsNull())
				{
					FPort Port = FPort(PortItem->AsObject());
					Ports.Add(Port);
				}
			}
		}
	}
};
