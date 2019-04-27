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

#include "Agones.h"
#include "AgonesHook.h"

#if WITH_EDITOR
#include "AgonesSettings.h"
#include "ISettingsModule.h"
#include "ISettingsSection.h"
#include "UObject/Class.h"
#include "UObject/WeakObjectPtr.h"
#endif

#define LOCTEXT_NAMESPACE "AgonesModule"

void FAgonesModule::StartupModule()
{
	FWorldDelegates::OnPostWorldInitialization.AddRaw(this, &FAgonesModule::OnWorldInitialized);


#if WITH_EDITOR
	// register Agones settings
	ISettingsModule* SettingsModule = FModuleManager::GetModulePtr<ISettingsModule>("Settings");

	if (SettingsModule != nullptr)
	{
		ISettingsSectionPtr SettingsSection = SettingsModule->RegisterSettings("Project", "Plugins", "Agones",
			LOCTEXT("AgonesSettingsName", "Agones"),
			LOCTEXT("AgonesSettingsDescription", "Configure the Agones plug-in."),
			GetMutableDefault<UAgonesSettings>()
		);
	}
#endif //WITH_EDITOR
}

void FAgonesModule::ShutdownModule()
{
	FWorldDelegates::OnPostWorldInitialization.RemoveAll(this);

	if (HookPtr.IsValid())
	{
		HookPtr->Shutdown();
	}

#if WITH_EDITOR
	// unregister Agones settings
	ISettingsModule* SettingsModule = FModuleManager::GetModulePtr<ISettingsModule>("Settings");

	if (SettingsModule != nullptr)
	{
		SettingsModule->UnregisterSettings("Project", "Plugins", "Agones");
	}
#endif //WITH_EDITOR
}

void FAgonesModule::OnWorldInitialized(UWorld* World, const UWorld::InitializationValues IVS)
{
	// Only start the agones hook if this is a dedicated server.
	if (World != nullptr && World->GetNetMode() == ENetMode::NM_DedicatedServer)
	{
		HookPtr = MakeShareable(new FAgonesHook());
		HookPtr->Ready();
	}
}

#undef LOCTEXT_NAMESPACE

IMPLEMENT_MODULE(FAgonesModule, Agones)