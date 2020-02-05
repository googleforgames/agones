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
#include "Dom/JsonObject.h"
#include "GameServerObjectMeta.generated.h"

USTRUCT()
struct AGONES_API FObjectMeta
{
	GENERATED_BODY()

	UPROPERTY()
	FString Name;

	UPROPERTY()
	FString Namespace;

	UPROPERTY()
	FString Uid;

	UPROPERTY()
	FString ResourceVersion;

	UPROPERTY()
	int64 Generation = 0;

	UPROPERTY()
	int64 CreationTimestamp = 0;

	UPROPERTY()
	int64 DeletionTimestamp = 0;

	UPROPERTY()
	TMap<FString, FString> Annotations;

	UPROPERTY()
	TMap<FString, FString> Labels;

	/** Default constructor, no initialization */
	FObjectMeta()
	{}

	FObjectMeta(TSharedPtr<FJsonObject> JsonObject)
	{
		JsonObject->TryGetStringField(TEXT("name"), Name);
		JsonObject->TryGetStringField(TEXT("namespace"), Namespace);
		JsonObject->TryGetStringField(TEXT("uid"), Uid);
		JsonObject->TryGetStringField(TEXT("resource_version"), ResourceVersion);
		JsonObject->TryGetNumberField(TEXT("generation"), Generation);
		JsonObject->TryGetNumberField(TEXT("creation_timestamp"), CreationTimestamp);
		JsonObject->TryGetNumberField(TEXT("deletion_timestamp"), DeletionTimestamp);
		const TSharedPtr<FJsonObject>* AnnotationsJsonObject;
		if (JsonObject->TryGetObjectField(TEXT("annotations"), AnnotationsJsonObject))
		{
			for (const auto& Entry : (*AnnotationsJsonObject)->Values)
			{
				if (Entry.Value.IsValid() && !Entry.Value->IsNull())
				{
					FJsonValueString Key = Entry.Key;
					TSharedPtr<FJsonValue> Value = Entry.Value;
					FString AnnotationKey = Key.AsString();
					FString AnnotationValue = Value->AsString();
					Annotations.Add(AnnotationKey, AnnotationValue);
				}
			}
		}
		const TSharedPtr<FJsonObject>* LabelsObject;
		if (JsonObject->TryGetObjectField(TEXT("labels"), LabelsObject))
		{
			for (const auto& Entry : (*LabelsObject)->Values)
			{
				if (Entry.Value.IsValid() && !Entry.Value->IsNull())
				{
					FJsonValueString Key = Entry.Key;
					TSharedPtr<FJsonValue> Value = Entry.Value;
					FString LabelKey = Key.AsString();
					FString LabelValue = Value->AsString();
					Labels.Add(LabelKey, LabelValue);
				}
			}
		}
	}
};
