# SuperTuxKart Game Server Example

Example using a [SuperTuxKart](https://supertuxkart.net/) dedicated game server.

This is a fork of the dedicated game server for SuperTuxKart such that it allows connection from AI bots from any
hostname. (See [this issue](https://github.com/supertuxkart/stk-code/issues/4244) for details).

This example wraps the SuperTuxKart server with a [Go](https://golang.org) binary, and introspects
the log file to provide the event hooks for the SDK integration.

It is not a direct integration, but is an approach to integrate with existing
dedicated game servers.

You will need to download the SuperTuxKart client separately to play.

For detailed instructions refer to the [Agones Documentation](https://agones.dev/site/docs/examples/supertuxkart/).
