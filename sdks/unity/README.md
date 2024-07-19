# Agones Unity Client SDK

## Contributing
Thank you for your interest in contributing to the Agones Unity client library. Setting up a local workspace to help develop the Agones Game Server Unity Client SDK involves cloning the Agones source code, having a host Unity project, and allowing the Unity project to run the external tests in the Unity test runner.

### A copy of Agones source code
If you don't already have a local copy of the Agones source code you can either (1) clone the official [googleforgames/agones repository](https://github.com/googleforgames/agones) for quicker access or (2) create a fork then clone your fork. Make note of where you clone the project to for the next setup steps!

### Setting up the Host Unity Project
You will want a Unity project to "host" a local copy of the Agones Unity SDK source code. We like the Unity project name "Agones Unity SDK Host Project" but name the host Unity project whatever you like. Open the project so Unity can complete its initial project setup.

Next we will tell the Unity package manager about the local Agones Unity SDK package. Open up the Unity Package Manager window (`Window > Package Manager`). Click the `+` (top left side of Package Manager window) to then click `Add package from disk`. Select your local copy of [SDKs/Unity/package.json](https://github.com/googleforgames/Agones/blob/main/SDKs/Unity/package.json) in the file finder window. Unity package manager will then load up the Agones Unity SDK which will reflect in the package manager window.

Next will be configuring the host Unity project to run Agones Unity SDK tests. Open up `Packages/manifest.json` from the root of the host Unity project. Add the follow to the JSON file after the `dependencies` property:
```json
...
  },
  "testables": [
    "com.googleforgames.agones"
  ]
}
```
You are now able to see tests for Agones Unity SDK in the test runner of your host Unity project.

Open up a code editor to your Unity project as you normally would but when contributing you will be selecting files from the `Packages/Agones Unity SDK` section to make edits. Tests covering contributions is always encouraged. You are now ready to develop the Agones Unity SDK using a Unity host project!
