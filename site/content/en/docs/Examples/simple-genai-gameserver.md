---
title: "Build and Run a Simple Game Server that Connects to an Inference Server"
linkTitle: "GenAI Game Server"
date:
publishDate:
description: >
  This example shows how to set up a simple game server that integrates with your inference server's GenAI endpoints. You can either interact with one GenAI endpoint via TCP, or set up two endpoints to "chat" to each other.
---

## Setting up the GenAI Inference Server

You will need a separate GenAI inference server. This example uses the
[Google for Games GenAI](https://github.com/googleforgames/GenAI-quickstart) as its inference server.
This particular inference server has the request structure for the /chat endpoint:

```
type GenAIRequest struct {
	Context         string  `json:"context,omitempty"`
	Prompt          string  `json:"prompt"`
}
```

### (Optional) Modify the GenAIRequest Structure

If you need a different request structure for your GenAI endpoint, you will need to clone or fork the Agones repository and
modify the above `GenAIRequest struct` in [main.go](https://github.com/googleforgames/agones/blob/main/examples/simple-genai-server/main.go). Update `REPOSITORY` in the
[agones/examples/simple-genai-server/Makefile](https://github.com/googleforgames/agones/blob/main/examples/simple-genai-server/Makefile)
to your own container registry and run `make build && make push` from within the
`agones/examples/simple-genai-server` path. Then modify the `gameserver_*.yaml` to pull the image from
your container registry. If you are making a series of changes you may also want to add
`imagePullPolicy: Always` to the container image in `gameserver_*.yaml`.

## Setting up Agones

To set up the Game Servers on a different cluster than the GenAI inference server follow the
instructions for [creating a cluster](https://agones.dev/site/docs/installation/creating-cluster/).

If you set up a separate cluster for your game servers [install Agones](https://agones.dev/site/docs/installation/install-agones/).
on that cluster, otherwise install Agones into your GenAI inference server cluster.

## Setting up the Game Server

To manually interact with GenAI endpoint via netcat, modify the
[gameserver_manualchat.yaml](https://github.com/googleforgames/agones/blob/main/examples/simple-genai-server/gameserver_manualchat.yaml) `GenAiEndpoint` value to your inference
server's endpoint. Optionally, include the `GenAiContext` in your `gameserver.yaml` that will
include the context with each chat (post request) that you make to the GenAI endpoint.

If you want to have two clients "chat" to each other, modify the
[gameserver_autochat.yaml](https://github.com/googleforgames/agones/blob/main/examples/simple-genai-server/gameserver_autochat.yaml) `GenAiEndpoint` value to your inference
server's endpoint. Also modify the `SimEndpoint` value to your inference server's endpoint.
Alternatively you can create a basic http server that accepts requests in the structure noted in the
above section, and returns a predetermined set of responses for the chat. The `GenAiContext` is sent
to the `GenAiEndpoint` with each request, and the `SimContext` is sent to the `SimEndpoint` with
each request as part of the GenAIRequest structure. The default values for `GenAiContext` and
`SimContext` are empty strings. The `Prompt` is the first sent to prompt send to the GenAI endpoint
to start the chat. The default values for the prompt is an empty string. `NumChats` is the number of
requests made to the `SimEndpoint` and `GenAiEndpoint`. The default value for is `NumChats` is `1`.

If you want to set up the chat with the npc-chat-api from the [Google for Games GenAI](https://github.com/googleforgames/GenAI-quickstart/tree/main/genai/api/npc_chat_api)
modify the [gameserver_npcchat.yaml](https://github.com/googleforgames/agones/blob/main/examples/simple-genai-server/gameserver_npcchat.yaml). Set either the
`GenAiEndpoint` or `SimEndpoint` to the NPC service `"http://genai-api.genai.svc/genai/npc_chat"`.
Set whichever endpoint is pointing to the NPC service, either the `GenAiNpc` or `SimNpc`,
to be `"true"`. The `GenAIRequest` to the NPC endpoint only sends the message (prompt), so any
additional context outside of the prompt is ignored. `FromID` is the entity sending messages to NPC,
and `ToID` is the entity receiving the message (the NPC ID).
```
type NPCRequest struct {
	Msg    string `json:"message,omitempty"`
	FromId int `json:"from_id,omitempty"`
	ToId   int `json:"to_id,omitempty"`
}
```

## Running the Game Server

Once you have modified the `gameserver_*.yaml` to use your
endpoint(s), apply to your Agones cluster with `kubectl apply -f gameserver_autochat.yaml`,
`kubectl apply -f gameserver_manualchat.yaml`, or `kubectl apply -f gameserver_npcchat`.

Note that if your inference server is in a different cluster you'll want to make sure you're using
the kubectl context that points to your Agones cluster and not the inference cluster.

If you set up the `gameserver_autochat.yaml` or `gameserver_npcchat` the chat will be in the game server logs:

```bash
kubectl logs -f gen-ai-server-auto -c simple-genai-game-server
```

In autochat mode, the game server will stay running forever until the game server is deleted.
While running, we keep `--ConcurrentPlayers` slots of players running - each simulated player
will initiate a chat and then go until they send `--StopPhrase` or until `--NumChats`, whichever
comes first, after which a new player will fill the slot.

If you set up the `gameserver_manualchat.yaml` you can manually send requests to the GenAI endpoint.
Retreive the IP address and port:

```bash
kubectl get gs gen-ai-server-manual -o jsonpath='{.status.address}:{.status.ports[0].port}'
```

You can now send requests to the GenAI endpoint:

{{< alert title="Note" color="info" >}}
If you do not have netcat installed (i.e. you get a response of `nc: command not found`), you can
install netcat by running `sudo apt install netcat`.
{{< /alert >}}


```
nc {IP} {PORT}
Enter your prompt for the GenAI server in the same terminal, and the response will appear here too.
```

After you're done you will need to manually delete the game server `kubectl delete gs gen-ai-server-manual`.
