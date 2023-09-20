# Simple Unity Netcode for GameObjects Example

This is a simple Unity example that demonstrates Agones integrations with
Unity's [Netcode for GameObjects](https://docs-multiplayer.unity3d.com/netcode/current/about/).

>Netcode for GameObjects (NGO) is a high-level networking library built for
>Unity for you to abstract networking logic. It enables you to send GameObjects
>and world data across a networking session to many players at once. With NGO,
>you can focus on building your game instead of low-level protocols and
>networking frameworks.

The example creates a seperate build both for the client and dedicated game
server. And builds off of the [example](https://docs-multiplayer.unity3d.com/netcode/current/tutorials/get-started-ngo/) from Unity but with Agones integration

## Prerequisites

This example is working on

```
Unity Editor: Unity 2022.3.3f1 or later
OS: MacOS
```

### Install Text Mesh Pro in Unity Editor

After opening UnityEditor install TextMeshPro under **Window** > **TextMeshPro** > **Import TMP Esstenrial Resources** and click **Import**.

## Getting Started

There are a few steps.

### Install and configure Agones on Kubernetes

Check out [these instructions](https://agones.dev/site/docs/installation/).

## Testing locally

To test locally, create one build for the dedicated game server and a seperate
for the client. Locally, you can run 2 clients to test.

### Building the Dedicated Game Server

* Open this project with UnityEditor.
* Click on the **File** > **Build Settings** menu item in the menu bar.
  * Make sure that for **Platform** that **Dedicated Game Server** is selected
  and **Target Platform** is selected to **Linux**. If needed click
  **Switch Platform** before build.
  * Verify the following scenes are selected in order:

  ```
  Scenes/NetworkBootstrap
  Scenes/Menu
  Scnes/GameTest
  ```

  * The Builds are created in a `Builds/Server` Folder and **Saved As** `Server`.

### Running the Dedicated Game Server

To test the dedicated game server build locally run `docker-compose`. The
Dockerfile instructions for this example are dependent on the above build
location `Builds/Server`.

```
docker-compose up --build
```

If successful, you will see positive healtchecks from the SDK, similar to below:

```
unity-netcode-gameserver-1  | Agones SendRequest ok: /health {}
unity-netcode-sdk-server-1  | {"message":"Health Ping Received!","severity":"info","source":"*sdkserver.LocalSDKServer","time":"2023-09-15T21:41:46.550060529Z"}
```

### Building the Client

Again, similar to the dedicated game server build.

* Open this project with UnityEditor.
* Click on the **File** > **Build Settings** menu item in the menu bar.
  * Make sure that for **Platform** that **Windows, Mac, Linux** is selected
  and **Target Platform** is selected to **MacOS** or as appropriate to your
  machine. If needed click **Switch Platform** before build.
  * Verify the following scenes are selected in order:
* Select **Build And Run** and wait for the Unity client to start.
* In parallel start a second client directly in UnityEditor by selecting **Play**
as the second client to test connectivity between clients.
* Once both clients are active, input the server IP address as `127.0.0.1` and
server port as `7777`.

## Running on Kubernetes

### Build and push with Docker

Build docker image and push to registry.

```
docker build -t agones-example/unity-netcode:latest .
```

### Google Cloud Build option

As a another option, you can build with [Google Cloud Build](https://cloud.google.com/build/docs/build-config-file-schema). This will also push image to Google Artifact Registry at `us-docker.pkg.dev/${PROJECT_ID}/${_ARTIFACT_REPO_NAME}/${_IMAGE_NAME}`.

To create an Artifact Registry if you have a Google Cloud Organization.

```
gcloud artifacts repositories create $_ARTIFACT_REPO_NAME \
    --repository-format=docker \
    --location=us \
    --async
```

Run Cloud Build config.

```
gcloud builds submit --config cloudbuild.yaml
```

### Update Agones `gameserver.yaml` and create

Update image spec in manifest for `gameserver.yaml`

```
...
      containers:
      - name: simple-game-server
        image: us-docker.pkg.dev/${PROJECT_ID}/${_ARTIFACT_REPO_NAME}/${_IMAGE_NAME}:${_IMAGE_VERSION}
        resources:
          requests:
            memory: "128Mi"
            cpu: "128m"
```

Create Agones Gameserver.

```
$ kubectl create -f gameserver.yaml
```

```
kubectl get gs
NAME                        STATE   ADDRESS         PORT   NODE                                              AGE
simple-unity-server-b4hnz   Ready   34.69.***.***   7953   gke-gke-agones-gke-agones-primary-65d17602-ld6z   5d21h
```

When running client use above `Address` and `Port` for clients

## Verifying client.

When connected you will be able to move around the capsule player with `WASD`. If
connected to second client the two players, two capsules will appear sharing state
via Unity's Netcode for Game Objects.
