// Copyright 2023 Google LLC All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
using Agones;
using Grpc.Core;

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
    {
        var status = await sdk.SetLabelAsync("creationTimestamp",
            gameServer.ObjectMeta.CreationTimestamp.ToString());
        if (status.StatusCode != StatusCode.OK)
        {
            Console.Error.WriteLine(
                $"Could not set label. StatusCode={status.StatusCode}, Detail={status.Detail}");
            Environment.Exit(1);
        }
    }
    {
        var status = await sdk.SetAnnotationAsync("UID", gameServer.ObjectMeta.Uid);
        if (status.StatusCode != StatusCode.OK)
        {
            Console.Error.WriteLine(
                $"Could not set annotation. StatusCode={status.StatusCode}, Detail={status.Detail}");
            Environment.Exit(1);
        }
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

if (featureGates.Contains("CountsAndLists"))
// Tests are expected to run sequentially on the same pre-defined Counter in the localsdk server
{
    var alpha = sdk.Alpha();
    var key = "rooms";

    {
        var wantCount = 1;
        var task = alpha.GetCounterCountAsync(key);
        task.Wait();
        var gotCount = task.Result;
        if (wantCount != gotCount)
        {
            Console.Error.WriteLine($"Counter count should be {wantCount}, but is {gotCount}");
            Environment.Exit(1);
        }
    }

    {
        var wantCount = 10;
        var increment = 9;
        var task = alpha.IncrementCounterAsync(key, increment);
        task.Wait();
        var incremented = task.Result;
        if (incremented != true)
        {
            Console.Error.WriteLine($"IncrementCounterAsync for Counter {key} did not increment");
            Environment.Exit(1);
        }
        var getTask = alpha.GetCounterCountAsync(key);
        getTask.Wait();
        var gotCount = getTask.Result;
        if (wantCount != gotCount)
        {
            Console.Error.WriteLine($"Counter count should be {wantCount}, but is {gotCount}");
            Environment.Exit(1);
        }
    }

    {
        var wantCount = 5;
        var decrement = 5;
        var task = alpha.DecrementCounterAsync(key, decrement);
        task.Wait();
        var decremented = task.Result;
        if (decremented != true)
        {
            Console.Error.WriteLine($"DecrementCounterAsync for Counter {key} did not decrement");
            Environment.Exit(1);
        }
        var getTask = alpha.GetCounterCountAsync(key);
        getTask.Wait();
        var gotCount = getTask.Result;
        if (wantCount != gotCount)
        {
            Console.Error.WriteLine($"Counter count should be {wantCount}, but is {gotCount}");
            Environment.Exit(1);
        }
    }

    {
        var wantCount = 3;
        var task = alpha.SetCounterCountAsync(key, wantCount);
        task.Wait();
        var setCount = task.Result;
        if (setCount != true)
        {
            Console.Error.WriteLine($"SetCounterCountAsync for Counter {key} did not set");
            Environment.Exit(1);
        }
        var getTask = alpha.GetCounterCountAsync(key);
        getTask.Wait();
        var gotCount = getTask.Result;
        if (wantCount != gotCount)
        {
            Console.Error.WriteLine($"Counter count should be {wantCount}, but is {gotCount}");
            Environment.Exit(1);
        }
    }

    {
        var wantCapacity = 10;
        var task = alpha.GetCounterCapacityAsync(key);
        task.Wait();
        var gotCapacity = task.Result;
        if (wantCapacity != gotCapacity)
        {
            Console.Error.WriteLine($"Counter capacity should be {wantCapacity}, but is {gotCapacity}");
            Environment.Exit(1);
        }
    }


    {
        var wantCapacity = 0;
        var task = alpha.SetCounterCapacityAsync(key, wantCapacity);
        task.Wait();
        var setCapacity = task.Result;
        if (setCapacity != true)
        {
            Console.Error.WriteLine($"SetCounterCapacityAsync for Counter {key} did not set");
            Environment.Exit(1);
        }
        var getTask = alpha.GetCounterCapacityAsync(key);
        getTask.Wait();
        var gotCapacity = getTask.Result;
        if (wantCapacity != gotCapacity)
        {
            Console.Error.WriteLine($"Counter capacity should be {wantCapacity}, but is {gotCapacity}");
            Environment.Exit(1);
        }
    }
}

if (featureGates.Contains("CountsAndLists"))
// Tests are expected to run sequentially on the same pre-defined List in the localsdk server
{
    var alpha = sdk.Alpha();
    var key = "players";

    {
        var wantCapacity = 100;
        var task = alpha.GetListCapacityAsync(key);
        task.Wait();
        var gotCapacity = task.Result;
        if (wantCapacity != gotCapacity)
        {
            Console.Error.WriteLine($"List capacity should be {wantCapacity}, but is {gotCapacity}");
            Environment.Exit(1);
        }
    }

    {
        var wantCapacity = 10;
        var task = alpha.SetListCapacityAsync(key, wantCapacity);
        task.Wait();
        var setCapacity = task.Result;
        if (setCapacity != true)
        {
            Console.Error.WriteLine($"SetListCapacityAsync for List {key} did not set");
            Environment.Exit(1);
        }
        var getTask = alpha.GetListCapacityAsync(key);
        getTask.Wait();
        var gotCapacity = getTask.Result;
        if (wantCapacity != gotCapacity)
        {
            Console.Error.WriteLine($"List capacity should be {wantCapacity}, but is {gotCapacity}");
            Environment.Exit(1);
        }
    }

    {
        var value = "foo";
        var want = false;
        var task = alpha.ListContainsAsync(key, value);
        task.Wait();
        var got = task.Result;
        if (want != got)
        {
            Console.Error.WriteLine($"ListContains expected {want} for value {value}, but got {got}");
            Environment.Exit(1);
        }
        value = "test1";
        want = true;
        task = alpha.ListContainsAsync(key, value);
        task.Wait();
        got = task.Result;
        if (want != got)
        {
            Console.Error.WriteLine($"ListContains expected {want} for value {value}, but got {got}");
            Environment.Exit(1);
        }
    }

    {
        var wantValues = new List<string> { "test0", "test1", "test2" };
        var task = alpha.GetListValuesAsync(key);
        task.Wait();
        var gotValues = task.Result;
        var equal = Enumerable.SequenceEqual(wantValues, gotValues);
        if (!equal)
        {
            var wantStr = String.Join(" ", wantValues);
            var gotStr = String.Join(" ", gotValues);
            Console.Error.WriteLine($"List values should be {wantStr}, but is {gotStr}");
            Environment.Exit(1);
        }
    }

    {
        var addValue = "test3";
        var wantValues = new List<string> { "test0", "test1", "test2", "test3" };
        var task = alpha.AppendListValueAsync(key, addValue);
        task.Wait();
        var got = task.Result;
        if (got != true)
        {
            Console.Error.WriteLine($"AppendListValueAsync for List {key} did not append {addValue}");
            Environment.Exit(1);
        }
        var getTask = alpha.GetListValuesAsync(key);
        getTask.Wait();
        var gotValues = getTask.Result;
        var equal = Enumerable.SequenceEqual(wantValues, gotValues);
        if (!equal)
        {
            var wantStr = String.Join(" ", wantValues);
            var gotStr = String.Join(" ", gotValues);
            Console.Error.WriteLine($"List values should be {wantStr}, but is {gotStr}");
            Environment.Exit(1);
        }
    }

    {
        var removeValue = "test2";
        var wantValues = new List<string> { "test0", "test1", "test3" };
        var task = alpha.DeleteListValueAsync(key, removeValue);
        task.Wait();
        var got = task.Result;
        if (got != true)
        {
            Console.Error.WriteLine($"DeleteListValueAsync for List {key} did not delete {removeValue}");
            Environment.Exit(1);
        }
        var getTask = alpha.GetListValuesAsync(key);
        getTask.Wait();
        var gotValues = getTask.Result;
        var equal = Enumerable.SequenceEqual(wantValues, gotValues);
        if (!equal)
        {
            var wantStr = String.Join(" ", wantValues);
            var gotStr = String.Join(" ", gotValues);
            Console.Error.WriteLine($"List values should be {wantStr}, but is {gotStr}");
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

Console.WriteLine("Finish all tests");
