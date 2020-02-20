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
#include "UObject/ObjectMacros.h"
#include "UObject/Object.h"
#include "AgonesSettings.generated.h"

/**
 * Implements the settings for Agones.
 */
UCLASS(config = Game, defaultconfig)
class AGONES_API UAgonesSettings : public UObject
{
	GENERATED_BODY()

public:

	/** Default constructor */
	UAgonesSettings();

	UPROPERTY(EditAnywhere, config, Category = "Agones", meta = (DisplayName = "Health Ping Enabled"))
	bool bHealthPingEnabled;

	UPROPERTY(EditAnywhere, config, Category = "Agones", meta = (DisplayName = "Health Ping Seconds"))
	float HealthPingSeconds;

	UPROPERTY(EditAnywhere, config, Category = "Agones", meta = (DisplayName = "Request Retry Limit"))
	uint32 RequestRetryLimit;

	UPROPERTY(EditAnywhere, config, Category = "Agones", meta = (DisplayName = "Send Ready at Startup"))
	bool bSendReadyAtStartup;
};
