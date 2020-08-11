// Copyright 2020 Google LLC All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
using Agones.Dev.Sdk;
using Grpc.Core;
using Microsoft.Extensions.Logging;
using System;
using System.Threading;
using System.Threading.Tasks;

[assembly: System.Runtime.CompilerServices.InternalsVisibleTo("Agones.Test")]
namespace Agones
{
	public sealed class AgonesSDK : IAgonesSDK
	{
		public string Host { get; } = Environment.GetEnvironmentVariable("AGONES_SDK_GRPC_HOST") ?? "localhost";
		public int Port { get; } = Convert.ToInt32(Environment.GetEnvironmentVariable("AGONES_SDK_GRPC_PORT") ?? "9357");

		/// <summary>
		/// The timeout for gRPC calls.
		/// </summary>
		public double RequestTimeoutSec { get; set; }

		internal SDK.SDKClient client;
		internal readonly Channel channel;
		internal readonly IClientStreamWriter<Empty> healthStream;
		internal readonly CancellationTokenSource cts;
		internal CancellationToken ctoken;
		internal volatile bool isWatchingGameServer;
		internal AsyncServerStreamingCall<GameServer> watchStreamingCall;

		/// <summary>
		/// Fired every time the GameServer k8s data is updated.
		/// A more efficient way to emulate WatchGameServer behavior
		/// without starting a new task for every subscription request.
		/// </summary>
		private event Action<GameServer> GameServerUpdated;
		internal Delegate[] GameServerUpdatedCallbacks => GameServerUpdated.GetInvocationList();
		private readonly ILogger _logger;
		internal readonly Alpha alpha;

		public AgonesSDK(
			double requestTimeoutSec = 15,
			SDK.SDKClient sdkClient = null,
			CancellationTokenSource cancellationTokenSource = null,
			ILogger logger = null)
		{
			_logger = logger;
			RequestTimeoutSec = requestTimeoutSec;
			cts = cancellationTokenSource ?? new CancellationTokenSource();
			ctoken = cts.Token;
			channel = new Channel(Host, Port, ChannelCredentials.Insecure);
			client = sdkClient ?? new SDK.SDKClient(channel);
			healthStream = client.Health().RequestStream;
			alpha = new Alpha(channel, requestTimeoutSec, cancellationTokenSource, logger);
		}

		/// <summary>
		/// Alpha returns the Alpha SDK
		/// </summary>
		/// <returns>Agones alpha SDK</returns>
		public IAgonesAlphaSDK Alpha()
		{
			return alpha;
		}

		/// <summary>
		/// Connect the underlying gRPC channel.
		/// </summary>
		/// <returns>True if successful</returns>
		public async Task<bool> ConnectAsync()
		{
			await channel.ConnectAsync(DateTime.UtcNow.AddSeconds(RequestTimeoutSec));
			if (channel.State == ChannelState.Ready)
			{
				return true;
			}
			LogError(null, $"Could not connect to the sidecar at {Host}:{Port}. Exited with connection state: {channel.State}.");
			return false;
		}

		/// <summary>
		/// Tells Agones that the Game Server is ready to take player connections
		/// </summary>
		/// <returns>gRPC Status of the request</returns>
		public async Task<Status> ReadyAsync()
		{
			try
			{
				await client.ReadyAsync(new Empty(), deadline: DateTime.UtcNow.AddSeconds(RequestTimeoutSec), cancellationToken: ctoken);
				return new Status(StatusCode.OK, "Ready request successful.");
			}
			catch (RpcException ex)
			{
				LogError(ex, "Unable to mark GameServer to 'Ready' state.");
				return ex.Status;
			}
		}

		/// <summary>
		/// Marks the game server as Allocated.
		/// </summary>
		/// <returns>gRPC Status of the request</returns>
		public async Task<Status> AllocateAsync()
		{
			try
			{
				await client.AllocateAsync(new Empty(),
					deadline: DateTime.UtcNow.AddSeconds(RequestTimeoutSec),
					cancellationToken: ctoken);
				return new Status(StatusCode.OK, "Allocate request successful.");
			}
			catch (RpcException ex)
			{
				LogError(ex, "Unable to mark the GameServer to 'Allocated' state.");
				return ex.Status;
			}
		}

		/// <summary>
		/// Reserve(seconds) will move the GameServer into the Reserved state for the specified number of seconds (0 is forever),
		/// and then it will be moved back to Ready state. While in Reserved state,
		/// the GameServer will not be deleted on scale down or Fleet update, and also it could not be Allocated using GameServerAllocation.
		/// </summary>
		/// <param name="seconds">Amount of seconds to reserve.</param>
		/// <returns>gRPC Status of the request</returns>
		public async Task<Status> ReserveAsync(long seconds)
		{
			try
			{
				await client.ReserveAsync(new Duration { Seconds = seconds},
					deadline: DateTime.UtcNow.AddSeconds(RequestTimeoutSec),
					cancellationToken: ctoken);
				return new Status(StatusCode.OK, $"Reserve({seconds}) request successful.");
			}
			catch (RpcException ex)
			{
				LogError(ex, "Unable to mark the GameServer to 'Reserved' state.");
				return ex.Status;
			}
		}

		/// <summary>
		/// This returns most of the backing GameServer configuration and Status.
		/// This can be useful for instances where you may want to know Health check configuration, or the IP and Port the GameServer is currently allocated to.
		/// </summary>
		/// <returns>A GameServer object containing this GameServer's configuration data</returns>
		public async Task<GameServer> GetGameServerAsync()
		{
			try
			{
				return await client.GetGameServerAsync(new Empty(),
					deadline: DateTime.UtcNow.AddSeconds(RequestTimeoutSec),
					cancellationToken: ctoken);
			}
			catch (RpcException ex)
			{
				LogError(ex, "Unable to get GameServer configuration and status.");
				throw;
			}
		}

		/// <summary>
		/// Starts watching the GameServer updates in the background in it's own task.
		/// On update, it fires the GameServerUpdate event.
		/// </summary>
		private async Task BeginInternalWatch()
		{
			//Begin WatchGameServer in the background for the provided callback.
			try
			{
				using (watchStreamingCall = client.WatchGameServer(new Empty(), cancellationToken: ctoken))
				{
					var reader = watchStreamingCall.ResponseStream;
					while (await reader.MoveNext(ctoken))
					{
						GameServerUpdated?.Invoke(reader.Current);
					}
				}
			}
			catch (RpcException ex)
			{
				LogError(ex, "Unable to subscribe to GameServer events.");
				throw;
			}
		}

		/// <summary>
		/// This executes the passed in callback with the current GameServer details whenever the underlying GameServer configuration is updated.
		/// This can be useful to track GameServer > Status > State changes, metadata changes, such as labels and annotations, and more.
		/// </summary>
		/// <param name="callback">The action to be called when the underlying GameServer metadata changes.</param>
		public void WatchGameServer(Action<GameServer> callback)
		{
			GameServerUpdated += callback;
			if (isWatchingGameServer) return;
			isWatchingGameServer = true;
			//Ignoring this warning as design is intentional.
			#pragma warning disable 4014
			BeginInternalWatch().ContinueWith((t)=> { Dispose(); },
				TaskContinuationOptions.OnlyOnFaulted);
			#pragma warning restore 4014
		}

		/// <summary>
		/// Cancels all running tasks & tells Agones to shut down the currently running game server.
		/// </summary>
		/// <returns>gRPC Status of the request</returns>
		public async Task<Status> ShutDownAsync()
		{
			try
			{
				await client.ShutdownAsync(new Empty(), deadline: DateTime.UtcNow.AddSeconds(RequestTimeoutSec));
				return new Status(StatusCode.OK, "Shutdown request successful.");
			}
			catch (RpcException ex)
			{
				LogError(ex, "Unable to mark the GameServer to 'Shutdown' state.");
				return ex.Status;
			}
		}

		/// <summary>
		/// Set a Label value on the backing GameServer record that is stored in Kubernetes.
		/// </summary>
		/// <param name="key">Label key</param>
		/// <param name="value">Label value</param>
		/// <returns>gRPC Status of the request</returns>
		public async Task<Status> SetLabelAsync(string key, string value)
		{
			try
			{
				await client.SetLabelAsync(new KeyValue()
				{
					Key = key,
					Value = value
				}, deadline: DateTime.UtcNow.AddSeconds(RequestTimeoutSec), cancellationToken: ctoken);
				return new Status(StatusCode.OK, $"SetLabel {key}:{value} request successful.");
			}
			catch(RpcException ex)
			{
				LogError(ex, $"Unable to set the GameServer label '{key}' to '{value}'.");
				return ex.Status;
			}
		}

		/// <summary>
		/// Set a Annotation value on the backing Gameserver record that is stored in Kubernetes.
		/// </summary>
		/// <param name="key">Annotation key</param>
		/// <param name="value">Annotation value</param>
		/// <returns>gRPC Status of the request</returns>
		public async Task<Status> SetAnnotationAsync(string key, string value)
		{
			try
			{
				await client.SetAnnotationAsync(new KeyValue()
				{
					Key = key,
					Value = value
				}, deadline: DateTime.UtcNow.AddSeconds(RequestTimeoutSec), cancellationToken: ctoken);
				return new Status(StatusCode.OK, $"SetAnnotation {key}:{value} request successful.");
			}
			catch (RpcException ex)
			{
				LogError(ex, $"Unable to set the GameServer annotation '{key}' to '{value}'.");
				return ex.Status;
			}
		}

		/// <summary>
		/// Sends a single ping to designate that the Game Server is alive and healthy.
		/// </summary>
		/// <returns>gRPC Status of the request</returns>
		public async Task<Status> HealthAsync()
		{
			try
			{
				await healthStream.WriteAsync(new Empty());
				return new Status(StatusCode.OK, "Health ping successful.");
			}
			catch (RpcException ex)
			{
				LogError(ex, "Unable to invoke the GameServer health check.");
				return ex.Status;
			}
		}

		public void Dispose()
		{
			cts.Cancel();
		}

		~AgonesSDK()
		{
			Dispose();
		}

		private void LogError(Exception ex, string message)
		{
			_logger?.LogError(ex, message);
		}
	}
}
