# Simple Game Server

A very simple "toy" game server created to demo and test running a UDP and/or
TCP game server on Agones.

To learn how to deploy your edited version of go server to gcp, please check
out this link: [Edit Your First Game Server (Go)](https://agones.dev/site/docs/getting-started/edit-first-gameserver-go/),
or also look at the [`Makefile`](./Makefile).

## Interacting with the Server

When the server receives a text packet, it will send back "ACK:<text content>"
for UDP as an echo or "ACK TCP:<text content>" for TCP.

There are some text commands you can send the server to affect its behavior:

| Command             | Behavior                                                                                 |
| ------------------- | ---------------------------------------------------------------------------------------- |
| "EXIT"              | Causes the game server to exit cleanly calling `os.Exit(0)`                              |
| "UNHEATHY"          | Stopping sending health checks                                                           |
| "GAMESERVER"        | Sends back the game server name                                                          |
| "READY"             | Marks the server as Ready                                                                |
| "ALLOCATE"          | Allocates the game server                                                                |
| "RESERVE"           | Reserves the game server after the specified duration                                    |
| "WATCH"             | Instructs the game server to log changes to the resource                                 |
| "LABEL"             | Sets the specified label on the game server resource                                     |
| "CRASH"             | Causes the game server to exit / crash immediately                                       |
| "ANNOTATION"        | Sets the specified annotation on the game server resource                                |
| "PLAYER_CAPACITY"   | With one argument, gets the player capacity; with two arguments sets the player capacity |
| "PLAYER_CONNECT"    | Connects the specified player to the game server                                         |
| "PLAYER_DISCONNECT" | Disconnects the specified player from the game server                                    |
| "PLAYER_CONNECTED"  | Returns true/false depending on whether the specified player is connected                |
| "GET_PLAYERS"       | Returns a list of the connected players                                                  |
| "PLAYER_COUNT"      | Returns a count of the connected players                                                 |


## Configuration

The server has a few configuration options that can be set via command line
flags. Some can also be set using environment variables.

| Flag                      | Environment Variable | Default |
| ------------------------- | -------------------- | ------- |
| port                      | PORT                 | 7654    |
| passthrough               | PASSTHROUGH          | false   |
| ready                     | READY                | true    |
| automaticShutdownDelayMin | _n/a_                | 0       |
| readyDelaySec             | _n/a_                | 0       |
| udp                       | UDP                  | true    |
| tcp                       | TCP                  | false   |

