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
modify this in [main.go](main.go). Update `REPOSITORY` in the
[agones/examples/simple-genai-server/Makefile](Makefile)
to your own container registry and run `make build && make push` from within the
`agones/examples/simple-genai-server` path. Then modify the `gameserver.yaml` to pull the image from
your container registry. If you are making a series of changes you may also want to add
`imagePullPolicy: Always` to the container image in `gameserver.yaml`.

## Setting up Agones

### Set up Agones on a Different Cluster than the GenAI Inference Server

To set up the Game Servers on a different cluster than the GenAI inference server follow the
instructions for [creating a cluster](https://agones.dev/site/docs/installation/creating-cluster/).
Follow the instructions for
[installing Agones](https://agones.dev/site/docs/installation/install-agones/).


### Set the Game Servers on the Same Cluster as the GenAI Inference Server

To set the Game Servers on the same cluster as Google for Games GenAI inference server
[create an Agones controller node pool](https://agones.dev/site/docs/installation/creating-cluster/gke/#optional-creating-a-dedicated-node-pool),
optionally [create a metrics node pool](https://agones.dev/site/docs/installation/creating-cluster/gke/#optional-creating-a-metrics-node-pool),
create a node pool for your game servers:
```bash
gcloud container node-pools create game-servers  \
  --cluster=[CLUSTER_NAME] \
  --zone=[COMPUTE_ZONE] \
  --tags=game-server \
  --node-taints agones.dev/role=gameserver:NoExecute \
  --node-labels agones.dev/role=gameserver \
  --num-nodes=4 \
  --machine-type=e2-standard-4
```
and create a firewall in the same VPC network as your inference server to access the game servers:
```bash
gcloud compute firewall-rules create gke-genai-quickstart-game-server-firewall-tcp \
  --description "Firewall to allow game server tcp traffic" \
  --allow tcp:7000-8000 \
  --target-tags game-server \
  --network vpc-genai-quickstart \
  --priority 998
```
Note that if you use this dedicated game-servers node-pool you will also need to use the `toleration`
and `nodeSelector` in the [gameserver_manualchat.yaml](gameserver_manualchat.yaml) or
[gameserver_autochat.yaml](gameserver_autochat.yaml).

Lastly, follow the instructions for
[installing Agones](https://agones.dev/site/docs/installation/install-agones/).

## Setting up the Game Server

To manually interact with GenAI endpoint via netcat, modify the
[gameserver_manualchat.yaml](gameserver_manualchat.yaml) `GenAiEndpoint` value to your inference
server's endpoint. Optionally, include the `GenAiContext` in your `gameserver.yaml` that will
include the context with each chat (post request) that you make to the GenAI endpoint.

If you want to have two clients "chat" to each other, modify the
[gameserver_autochat.yaml](gameserver_autochat.yaml) `GenAiEndpoint` value to your inference
server's endpoint. Also modift the `SimEndpoint` value to your inference server's endpoint.
Alternatively you can create a basic http server that accepts requests in the structure noted in the
above section, and returns a predetermined set of responses for the chat. The `GenAiContext` is sent
to the `GenAiEndpoint` with each request, and the `SimContext` is sent to the `SimEndpoint` with
each request as part of the GenAIRequest structure. The default values for `GenAiContext` and
`SimContext` are empty strings. The `Prompt` is the first sent to prompt send to the GenAI endpoint
to start the chat. The default values for the prompt is an empty string. `NumChats` is the number of
requests made to the `SimEndpoint` and `GenAiEndpoint`. The default value for is `NumChats` is `1`.

## Running the Game Server

Once you have modified the `gameserver_autochat.yaml` or `gameserver_manualchat.yaml` to use your
endpoint(s), apply to your Agones cluster with `kubectl apply -f gameserver_autochat.yaml` or
`kubectl apply -f gameserver_manualchat.yaml`

Note that if your inference server is in a different cluster you'll want to make sure you're using
the kubectl context that points to your Agones cluster and not the inference cluster.

If you set up the `gameserver_autochat.yaml` the chat will be in the game server logs:

```bash
kubectl logs -f gen-ai-server-auto -c simple-genai-game-server
```

In autochat mode the game server will shutdown automatically once the chat is complete.

If you set up the `gameserver_manualchat.yaml` you can manually send requests to the GenAI endpoint.
Retreive the IP address and port:

```bash
kubectl get gs gen-ai-server-manual -o jsonpath='{.status.address}:{.status.ports[0].port}'
```

You can now send requests to the GenAI endpoint:

> [!NOTE]
> If you do not have netcat installed (i.e. you get a response of `nc: command not found`), you can
> install netcat by running `sudo apt install netcat`.

```
nc {IP} {PORT}
Enter your prompt for the GenAI server in the same terminal, and the response will appear here too.
```

In manual chat mode the game server will need to be manually deleted `kubectl delete gs gen-ai-server-manual`.
