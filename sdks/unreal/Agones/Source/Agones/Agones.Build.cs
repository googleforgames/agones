using UnrealBuildTool;

public class Agones : ModuleRules
{
	public Agones(ReadOnlyTargetRules target) : base(target)
	{
		PCHUsage = PCHUsageMode.UseExplicitOrSharedPCHs;
		PublicIncludePaths.AddRange(new string[] {});
		PrivateIncludePaths.AddRange(new string[] {});
		PublicDependencyModuleNames.AddRange(new[]
		{
			"Core",
			"Http",
			"Json",
			"JsonUtilities"
		});
		PrivateDependencyModuleNames.AddRange(
			new[]
			{
				"CoreUObject",
				"Engine",
				"Slate",
				"SlateCore"
			});
		DynamicallyLoadedModuleNames.AddRange(new string[]{ });
	}
}
