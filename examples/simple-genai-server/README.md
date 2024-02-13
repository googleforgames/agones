# Simple Game Server for GenAI

A simple game server that serves as an example of how to integrate GenAI endpoints into your
dedicated game server. You can either interact with one GenAI endpoint via TCP, or set up two
endpoints to "chat" to each other.

## Setting up the GenAI Inference Server

You will need a separate GenAI inference server. This game server uses the
[Google for Games GenAI](https://github.com/googleforgames/GenAI-quickstart) as its inference server.
This particular inference server has the request structure for the /chat endpoint:

```
type GenAIRequest struct {
	Context         string  `json:"context,omitempty"`
	Prompt          string  `json:"prompt"`
}
```

If you need a different request structure, you will need to clone or fork this repository and
modify this in `main.go`. Update `REPOSITORY` in the `agones/examples/simple-genai-server/Makefile`
to your own container registry and run `make build && make push` from within the
`agones/examples/simple-genai-server` path. Then modify the `gameserver.yaml` to pull the image from
your container registry. If you are making a series of changes you may also want to add
`imagePullPolicy: Always` to the container image in `gameserver.yaml`.

## Setting up the Game Server

This example uses two separate clusters for the GenAI server and the game server. For the game
server follow the instructions for [creating a cluster](https://agones.dev/site/docs/installation/creating-cluster/)
and [installing Agones](https://agones.dev/site/docs/installation/install-agones/).

Modify the `gameserver.yaml` `GenAiEndpoint` value to your inference server's endpoint. If you want
to manually interact with the GenAI endpoint via netcat, remove the rest of the env variables in the
`gameserver.yaml`. Optionally, include the `GenAiContext` in your `gameserver.yaml`.

If you want to have two clients "chat" to each other, modify the `gameserver.yaml` `SimEndpoint`
value to your inference server's endpoint. Alternatively you can create a basic http server that
accepts requests in the structure noted in the above section, and returns a predetermined set of
responses for the chat. The `GenAiContext` is sent to the `GenAiEndpoint` with each request, and the
`SimContext` is sent to the `SimEndpoint` with each request as part of the GenAIRequest structure.
The default values for `GenAiContext` and `SimContext` are empty strings. The `Prompt` is the first
sent to prompt send to the GenAI endpoint to start the chat. The default values for the prompt is an
empty string. `NumChats` is the number of requests made to the `SimEndpoint` and `GenAiEndpoint`.
The default value for is `NumChats` is `1`.

## Running the Game Server

Once you have modified the game server, apply to your Agones cluster with `kubectl -f gameserver.yaml`.
Note that if your inference server is in a different cluster you'll want to make sure you're using
the kubectl context that points to your Agones cluster and not the inference cluster.

If you set up the `SimEndpoint` the chat will be in the game server logs:

```bash
kubectl logs -f gen-ai-server -c simple-genai-game-server
```

If you did not set up `SimEndpoint` you can manually send requests to the GenAI endpoint. Retreive
the IP address and port:

```bash
kubectl get gs gen-ai-server -o jsonpath='{.status.address}:{.status.ports[0].port}'
```

You can now send requests to the GenAI endpoint:

> [!NOTE]
> If you do not have netcat installed (i.e. you get a response of `nc: command not found`), you can
> install netcat by running `sudo apt install netcat`.

```
nc {IP} {PORT}
Enter your prompt for the GenAI server in the same terminal, and the response will appear here too.
```
