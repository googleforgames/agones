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

using System;
using System.Threading;
using System.Threading.Tasks;
using Grpc.Core;
using Agones.Dev.Sdk;

[assembly: System.Runtime.CompilerServices.InternalsVisibleTo("Agones.Test")]
namespace Agones
{
	public sealed class AgonesSDK : IDisposable
	{
		public string Host { get; } = Environment.GetEnvironmentVariable("AGONES_SDK_GRPC_HOST") ?? "localhost";
		public int Port { get; } = Convert.ToInt32((Environment.GetEnvironmentVariable("AGONES_SDK_GRPC_PORT") ?? "9357"));

		/// <summary>
		/// The timeout for gRPC calls.
		/// </summary>
		public double RequestTimeout { get; set; }
		
		/// <summary>
		/// Interval between Health pings in seconds.
		/// </summary>
		public int HealthInterval { get; set; } 
		
		/// <summary>
		/// Set to true when you want health pings to occur periodically.
		/// </summary>
		public bool HealthEnabled { get; set; } 

		internal SDK.SDKClient client;
		internal Channel channel;
		internal IClientStreamWriter<Empty> healthStream;
		internal CancellationTokenSource cts = new CancellationTokenSource();
		internal CancellationToken ctoken => cts.Token;
		internal volatile bool isWatchingGameServer = false;
		internal AsyncServerStreamingCall<GameServer> watchStreamingCall;

		/// <summary>
		/// Fired everytime the GameServer k8s data is updated.
		/// A more efficient way to emulate WatchGameServer behavior
		/// without starting a new task for every subscrbtion request.
		/// </summary>
		private event Action<GameServer> GameServerUpdated;
		internal Delegate[] GameServerUpdatedCallbacks => GameServerUpdated.GetInvocationList();

		public AgonesSDK(int healthInterval = 5, bool healthEnabled = true, double requestTimeout = 15)
		{
			this.HealthInterval = healthInterval;
			this.HealthEnabled = healthEnabled;
			this.RequestTimeout = requestTimeout;
			this.channel = new Channel(Host, Port, ChannelCredentials.Insecure);
			this.client = new SDK.SDKClient(channel);
			this.healthStream = client.Health().RequestStream;
		}

		/// <summary>
		/// Connect the underlying gRPC channel.
		/// </summary>
		/// <returns>True if successful</returns>
		public async Task<bool> ConnectAsync()
		{
			await channel.ConnectAsync(DateTime.UtcNow.AddSeconds(30));
			if (channel.State != ChannelState.Ready){
				Console.Error.WriteLine($"Could not connect to the sidecar at {Host}:{Port}. Exited with connection state: {channel.State}");
				return false;
			}
			return true;
		}

		/// <summary>
		/// Tells Agones that the Game Server is ready to take player connections
		/// </summary>
		/// <returns>gRPC Status of the request</returns>
		public async Task<Status> ReadyAsync()
		{
			try
			{
				await client.ReadyAsync(new Empty(), deadline: DateTime.UtcNow.AddSeconds(RequestTimeout), cancellationToken: ctoken);
				return new Status(StatusCode.OK, "Ready request successful.");
			}
			catch (RpcException ex)
			{
				Console.Error.WriteLine(ex.Message);
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
				await client.AllocateAsync(new Empty(), deadline: DateTime.UtcNow.AddSeconds(RequestTimeout), cancellationToken: ctoken);
				return new Status(StatusCode.OK, "Allocate request successful.");
			}
			catch (RpcException ex)
			{
				Console.Error.WriteLine(ex.Message);
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
				await client.ReserveAsync(new Duration() {
					Seconds = seconds
				}, deadline: DateTime.UtcNow.AddSeconds(RequestTimeout), cancellationToken: ctoken);
				return new Status(StatusCode.OK, $"Reserve({seconds}) request successful.");
			}
			catch (RpcException ex)
			{
				Console.Error.WriteLine(ex.Message);
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
				return await client.GetGameServerAsync(new Empty(), deadline: DateTime.UtcNow.AddSeconds(RequestTimeout), cancellationToken: ctoken);
			}
			catch (RpcException ex)
			{
				Console.Error.WriteLine(ex.Message);
				//Should I rethrow the exception?
				return null;
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
				Console.Error.WriteLine(ex.Message);
				throw ex;
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
			if (!isWatchingGameServer)
			{
				isWatchingGameServer = true;
				//Ignoring this warning as design is intentional.
				#pragma warning disable 4014
				BeginInternalWatch().ContinueWith((t)=> { this.Dispose(); }, TaskContinuationOptions.OnlyOnFaulted);
				#pragma warning restore 4014
			}
		}

		/// <summary>
		/// Cancels all running tasks & tells Agones to shut down the currently running game server.
		/// </summary>
		/// <returns>gRPC Status of the request</returns>
		public async Task<Status> ShutDownAsync()
		{
			try
			{
				await client.ShutdownAsync(new Empty(), deadline: DateTime.UtcNow.AddSeconds(RequestTimeout));
				return new Status(StatusCode.OK, "Shutdown request successful.");
			}
			catch (RpcException ex)
			{
				Console.Error.WriteLine(ex.Message);
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
				}, deadline: DateTime.UtcNow.AddSeconds(RequestTimeout), cancellationToken: ctoken);
				return new Status(StatusCode.OK, $"SetLabel {key}:{value} request successful.");
			}
			catch(RpcException ex)
			{
				Console.Error.WriteLine(ex.Message);
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
				}, deadline: DateTime.UtcNow.AddSeconds(RequestTimeout), cancellationToken: ctoken);
				return new Status(StatusCode.OK, $"SetAnnotation {key}:{value} request successful.");
			}
			catch (RpcException ex)
			{
				Console.Error.WriteLine(ex.Message);
				return ex.Status;
			}
		}

		/// <summary>
		/// Sends a single ping to designate that the Game Server is alive and healthy.
		/// </summary>
		/// <returns>gRPC Status of the request</returns>
		internal async Task<Status> HealthAsync()
		{
			try
			{
				await healthStream.WriteAsync(new Empty());
				return new Status(StatusCode.OK, "Health ping successful.");
			}
			catch (RpcException ex)
			{
				Console.Error.WriteLine(ex.Message);
				return ex.Status;
			}
		}

		private async Task HealthCheckLoop()
		{
			while(!cts.IsCancellationRequested)
			{
				if(HealthEnabled)
					await HealthAsync();
				await Task.Delay(HealthInterval * 1000);
			}
		}

		public void Dispose()
		{
			cts.Cancel();
		}

		~AgonesSDK()
		{
			this.Dispose();
		}
	}
}
