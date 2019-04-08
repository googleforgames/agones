---
title: "Unreal Engine Game Server Client Plugin"
linkTitle: "Unreal Engine"
date: 2019-01-02T10:17:50Z
weight: 10
description: "This is the Unreal Engine 4 Agones Game Server Client Plugin. "
---

Check the [Client SDK Documentation]({{< relref "_index.md" >}}) for more details on each of the SDK functions and how to run the SDK locally.

## Download

Download the source from the [Releases Page](https://github.com/GoogleCloudPlatform/agones/releases) 
or {{< ghlink href="sdks/unreal" >}}directly from Github{{< /ghlink >}}.

## Usage

The Unreal Engine plugin is specifically designed to be as simple as possible. No programming should be required to use the plugin within your Unreal Engine project.

### From source

At this moment we do not provide binaries for the plugin. This requires you to compile the plugin yourself. In order to do this you need to have a C++ Unreal Engine project. 

1. Create a `Plugins` directory in your Unreal Engine project root directory.
2. Copy {{< ghlink href="sdks/unreal" >}}the Agones plugin directory{{< /ghlink >}} into the Plugins directory.
3. Build the project.

## Settings

The settings for the Agones Plugin can be found in the Unreal Engine editor `Edit > Project Settings > Plugins >  Agones`

Available settings:
- Agones Sidecar IP. (default: `http://localhost:59358`)
- Health Ping Enabled. Whether the server sends a health ping to the Agones sidecar. (default: `true`)
- Health Ping Seconds. Interval of the server sending a health ping to the Agones sidecar. (default: `5`)
- Debug Logging Enabled. Debug logging for development of this Plugin. (default: `false`)

