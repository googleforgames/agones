using Agones;
using Grpc.Core;

Console.WriteLine("Hello, World!");

using var sdk = new AgonesSDK();
{
    sdk.WatchGameServer((gameServer) =>
    {
        Console.WriteLine("Received GameServer update");
        Console.WriteLine(gameServer);
    });
}

{
    var status = await sdk.ReadyAsync();
    if (status.StatusCode != StatusCode.OK)
    {
        Console.Error.WriteLine(
            $"Could not send ready message. StatusCode={status.StatusCode}, Detail={status.Detail}");
        Environment.Exit(1);
    }
}

{
    var status = await sdk.ReserveAsync(5);
    if (status.StatusCode != StatusCode.OK)
    {
        Console.Error.WriteLine(
            $"Could not send reserve command. StatusCode={status.StatusCode}, Detail={status.Detail}");
        Environment.Exit(1);
    }
}

{
    var status = await sdk.AllocateAsync();
    if (status.StatusCode != StatusCode.OK)
    {
        Console.Error.WriteLine(
            $"Err sending allocate request. StatusCode={status.StatusCode}, Detail={status.Detail}");
        Environment.Exit(1);
    }
}

{
    var status = await sdk.HealthAsync();
    if (status.StatusCode != StatusCode.OK)
    {
        Console.Error.WriteLine(
            $"Could not send health check. StatusCode={status.StatusCode}, Detail={status.Detail}");
        Environment.Exit(1);
    }
}

{
    var gameServer = await sdk.GetGameServerAsync();
    Console.WriteLine("Successfully GameServer");
    Console.WriteLine(gameServer);
}

{
    var status = await sdk.SetLabelAsync("labelKey1", "labelValue1");
    if (status.StatusCode != StatusCode.OK)
    {
        Console.Error.WriteLine(
            $"Could not set label. StatusCode={status.StatusCode}, Detail={status.Detail}");
        Environment.Exit(1);
    }
}

{
    var status = await sdk.SetAnnotationAsync("annotationKey1", "annotationValue1");
    if (status.StatusCode != StatusCode.OK)
    {
        Console.Error.WriteLine(
            $"Could not set annotation. StatusCode={status.StatusCode}, Detail={status.Detail}");
        Environment.Exit(1);
    }
}

var featureGates = Environment.GetEnvironmentVariable("FEATURE_GATES") ?? "";
if (featureGates.Contains("PlayerTracking"))
{
    var alpha = sdk.Alpha();
    var capacity = 10;
    var playerId = "1234";

    {
        var status = await alpha.SetPlayerCapacityAsync(capacity);
        if (status.StatusCode != StatusCode.OK)
        {
            Console.Error.WriteLine(
                $"Error setting player capacity. StatusCode={status.StatusCode}, Detail={status.Detail}");
            Environment.Exit(1);
        }
    }

    {
        var c = await alpha.GetPlayerCapacityAsync();
        if (c != capacity)
        {
            Console.Error.WriteLine(
                $"Player Capacity should be {capacity}, but is {c}");
            Environment.Exit(1);
        }
    }

    {
        var ok = await alpha.PlayerConnectAsync(playerId);
        if (!ok)
        {
            Console.Error.WriteLine(
                $"PlayerConnect returned false");
            Environment.Exit(1);
        }
    }

    {
        var ok = await alpha.IsPlayerConnectedAsync(playerId);
        if (!ok)
        {
            Console.Error.WriteLine(
                $"IsPlayerConnected returned false");
            Environment.Exit(1);
        }
    }

    {
        var players = await alpha.GetConnectedPlayersAsync();
        if (players.Count == 0)
        {
            Console.Error.WriteLine(
                $"No connected players returned");
            Environment.Exit(1);
        }
    }

    {
        var ok = await alpha.PlayerDisconnectAsync(playerId);
        if (!ok)
        {
            Console.Error.WriteLine(
                $"PlayerDisconnect returned false");
            Environment.Exit(1);
        }
    }

    {
        var c = await alpha.GetPlayerCountAsync();
        if (c != 0)
        {
            Console.Error.WriteLine(
                $"Player Count should be 0, but is {c}");
            Environment.Exit(1);
        }
    }
}

var shutDownStatus = await sdk.ShutDownAsync();
if (shutDownStatus.StatusCode != StatusCode.OK)
{
    Console.Error.WriteLine(
        $"Could not shutdown GameServer. StatusCode={shutDownStatus.StatusCode}, Detail={shutDownStatus.Detail}");
    Environment.Exit(1);
}