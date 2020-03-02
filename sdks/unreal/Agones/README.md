# Agones UE4 Plugin

For installation and usage instructions, see
[Unreal Engine Game Server Client Plugin](https://agones.dev/site/docs/guides/client-sdks/unreal/).

If you'd like to contribute to the plugin, see
[CONTRIBUTING.md](/CONTRIBUTING.md)

## Developer Information

### Directory Structure

The plugin consists of a single [UE4
Module](https://docs.unrealengine.com/en-US/Programming/BuildTools/UnrealBuildTool/ModuleFiles/index.html),
also named [Agones](Source/Agones). The structure of this
module is based on the common *Public*/*Private* module layout found in many of
UE4's own internal modules where:

- *Public* contains all public C++ header files. These files can be included by
  game code and must not include files in *Private*.
- *Private* contains private C++ header and all C++ source files.

For more information on directory structure, see:

- *PublicIncludePaths* in [Modules](https://docs.unrealengine.com/en-US/Programming/BuildTools/UnrealBuildTool/ModuleFiles/index.html)
- [UE4 Marketplace Guidelines - Code Plugins](https://www.unrealengine.com/en-US/marketplace-guidelines#26)
- [UE4 engine source code](https://github.com/EpicGames/UnrealEngine/tree/release/Engine/Source)
(requires acceptance of [UE4 EULA](https://www.unrealengine.com/en-US/ue4-on-github))

### IWYU

Code should follow the [Include What You
Use](https://docs.unrealengine.com/en-US/Programming/BuildTools/UnrealBuildTool/IWYU/index.html)
dependency model. From [General Tips](https://docs.unrealengine.com/en-US/Programming/BuildTools/UnrealBuildTool/IWYU/#generaltips):

> 1. Include `CoreMinimal.h` at the top of each header file.
> 1. To verify that all of your source files include all of their required
>    dependencies, compile your game project in non-unity mode with PCH files
>    disabled.
> 1. If you need to access `UEngine` or `GEngine`, which are defined in
>    `Runtime\Engine\Classes\Engine\Engine.h`, you can `#include
>    Engine/Engine.h` (distinguishing from the monolithic header file, which is
>    located at `Runtime\Engine\Public\Engine.h`).
> 1. If you use a class that the compiler doesn't recognize, and don't know
>    what you need to include may be missing the header file. This is
>    especially the case if you are converting from non-IWYU code that compiled
>    correctly. You can look up the class in the API Documentation, and find
>    the necessary modules and header files at the bottom of the page. 

