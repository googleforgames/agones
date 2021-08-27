// Copyright Epic Games, Inc. All Rights Reserved.

using UnrealBuildTool;

public class AgonesExample : ModuleRules
{
	public AgonesExample(ReadOnlyTargetRules Target) : base(Target)
	{
		PCHUsage = PCHUsageMode.UseExplicitOrSharedPCHs;

		PublicDependencyModuleNames.AddRange(
			new string[]
			{
				"Agones",
				"Core",
				"CoreUObject",
				"Engine",
				"InputCore",
				"HeadMountedDisplay"
			});
	}
}
