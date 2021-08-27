# Unreal Engine Example
This is an example unreal engine game to show how the Agones Unreal Engine plugin can interact with Agones. This example is working on Unreal Engine 2.26.2 but the general principals apply to older builds and Unreal Engine 5 going into the future. To keep file sizes as small as possible and to minimize content being reliant on versions, no compiled content has been. If you would like to follow along and not use the prebuilt system, please refer to the [Following Along Documentation](FollowAlong.md)

## Prerequisites
We will be assuming you have an Unreal Engine version of choice installed and working

## Getting Started
### Compiling and Packaging
1. Clone the example files to disk
2. Right-Click `AgonesExample.uproject`, click `Generate Visual Studio Project Files`, and select the correct version of Unreal Engine you will be using. 
	- If you do not have this option, you need to find the `Unreal Build Tool` installation directory and run
	- `UnrealBuildTool.exe -projectfiles -project="Full:\Path\To\Project\AgonesExample.uproject" -game -Engine -rocket -progress`
	- The unreal build tool should be in the Unreal Engine installation directory: `UnrealEngine\Engine\Binaries\DotNET\UnrealBuildTool\UnrealBuildTool.exe`
3. Open up the project solution file `AgonesExample.sln` in your editor of choice and compile the game.
4. Launch the development editor under debug mode
5. Package for project for your OS, set the deployment to Shipping, and set the target to `Server`. Select an appropriate folder for the destination. A good example would be a `./Package/Shipping` folder in directory root.
6. Package the project for your OS, set the deployment to Shipping, and set the target to `Client`. Select the same folder you chose for the previous packaging.

You should now have a fully functional client and server. The gameserver is now reliant on the [Agones sidecar](https://agones.dev/site/docs/guides/client-sdks/local/) and will only work properly with this configured. When deploying to your Kuberentes cluster, this is handled automatically. However if you want to test locally, you will need to run the sidecar before you start the dedicated server.

### Local Debugging
1. Run the Agones sidecar. A precompiled version can be downloaded from [here](https://agones.dev/site/docs/guides/client-sdks/local/). 
2. Run the dedicated server.
3. Observe the agones sidecar and watch for ready requests, updates, and shutdowns.
### Cluster Deployment
The easiest way to get Unreal Engine working in Kuberentes is with a Linux based Docker image. There are many ways to do so, but here is one way.
1. [Configure agones in a cluster](https://agones.dev/site/docs/installation/install-agones/helm/)
2. Package the project for Linux, set the deployment to Shipping, and set the target to `Server`. Select the same folder you chose for the previous packaging.
3. Build a docker image from the packaged project. An example dockerfile [server.Dockerfile](server.Dockerfile) has been given. Be sure to edit it to point to the location of your packaged project.
4. Push the docker image to a public facing docker repository
5. Configure an Agones [GameServer Specification](https://agones.dev/site/docs/reference/gameserver/). Another template more specific to Unreal has been given under [gameserver.yaml](gameserver.yaml).
6. Deploy the gameserver to your cluster

