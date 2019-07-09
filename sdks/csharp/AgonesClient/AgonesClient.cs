using Agones.Models;
using Agones.Utility;
using Newtonsoft.Json;
using System;
using System.IO;
using System.Net;
using System.Net.Http;
using System.Net.Http.Headers;
using System.Threading.Tasks;

namespace Agones
{
	/// <summary>
	/// The delegate type used to receive info when watching a game server
	/// </summary>
	/// <param name="serverInfo"></param>
	public delegate void GameServerInfoHandler(GameServerInfo serverInfo);

	/// <summary>
	/// Agones Client used to interface with the agones sidecar
	/// </summary>
	public static class AgonesClient
	{
		const string BaseUrl = "http://localhost:59358/";

		static HttpClient httpClient;
		
		/// <summary>
		/// The logger that is used for all agones client logging
		/// </summary>
		public static ILogger Logger { get; set; } = new ConsoleLogger()
		{
			Prefix = "[AGONES] - "
		};

		static AgonesClient()
		{
			httpClient = new HttpClient()
			{
				BaseAddress = new Uri(BaseUrl)
			};

			httpClient.DefaultRequestHeaders.Accept.Add(new MediaTypeWithQualityHeaderValue("application/json"));
		}

		/// <summary>
		/// Mark the Game Server as Allocated
		/// It is usually preferred that this happens through a GameServerAllocation
		/// in a top down manner
		/// </summary>
		/// <returns></returns>
		public static async Task Allocate()
		{
			Logger.Log(LogLevel.Trace, "Allocating");

			var response = await httpClient.PostAsync("allocate", null);

			HandleEmptyResponse(response);
		}

		/// <summary>
		/// Mark the server as ready to receive players
		/// </summary>
		/// <returns></returns>
		public static async Task Ready()
		{
			Logger.Log(LogLevel.Trace, "Ready");

			var response = await httpClient.PostAsync("ready", null);

			HandleEmptyResponse(response);
		}

		/// <summary>
		/// Call periodically to inform Agones of the servers health
		/// </summary>
		/// <returns></returns>
		public static async Task Health()
		{
			Logger.Log(LogLevel.Trace, "Health Check");

			var response = await httpClient.PostAsync("health", null);

			HandleEmptyResponse(response);
		}

		/// <summary>
		/// Call to shutdown the game server
		/// </summary>
		/// <returns></returns>
		public static async Task Shutdown()
		{
			Logger.Log(LogLevel.Trace, "Shutting Down");

			var response = await httpClient.PostAsync("shutdown", null);

			HandleEmptyResponse(response);
		}

		/// <summary>
		/// Set a Label on the Game Servers metadata
		/// </summary>
		/// <param name="key"></param>
		/// <param name="value"></param>
		/// <returns></returns>
		public static async Task SetLabel(string key, string value)
		{
			Logger.Log(LogLevel.Trace, "Setting Label: {0} to: {1}", key, value);

			var response = await httpClient.PutAsJsonAsync("metadata/label", new { key, value });

			HandleEmptyResponse(response);
		}

		/// <summary>
		/// Set an Annotation on the Game Servers metadata
		/// </summary>
		/// <param name="key"></param>
		/// <param name="value"></param>
		/// <returns></returns>
		public static async Task SetAnnotation(string key, string value)
		{
			Logger.Log(LogLevel.Trace, "Setting Annotation: {0} to: {1}", key, value);

			var response = await httpClient.PutAsJsonAsync("metadata/annotation", new { key, value });

			HandleEmptyResponse(response);
		}

		/// <summary>
		/// Get a object describing the Game Server
		/// </summary>
		/// <returns></returns>
		public static async Task<GameServerInfo> GetGameServer()
		{
			Logger.Log(LogLevel.Trace, "Getting Game Server Info");

			var response = await httpClient.GetAsync("gameserver");

			return await HandleResponse<GameServerInfo>(response);
		}

		/// <summary>
		/// Receive a callback whenever the game server info changes, this can
		/// be useful to monitor for status or metadata changes
		/// </summary>
		/// <param name="handler"></param>
		public static void WatchGameServer(GameServerInfoHandler handler)
		{
			Logger.Log(LogLevel.Trace, "Watching for Game Server Info changes");

			Task.Run(() => StartObserving(new Uri(BaseUrl + "watch/gameserver"), handler));
		}

		static async void StartObserving(Uri watchUri, GameServerInfoHandler handler)
		{
			using (var client = new WebClient())
			using (var stream = await client.OpenReadTaskAsync(watchUri))
			using (var reader = new StreamReader(stream))
			{
				while (true)
				{
					try
					{
						var nextLine = await reader.ReadLineAsync();

						if (string.IsNullOrEmpty(nextLine))
						{
							Logger.Log(LogLevel.Info, "Received empty line while watching game server, stopped watch");

							break;
						}

						var watchResult = JsonConvert.DeserializeObject<GameServerInfoWatchResult>(nextLine);

						handler(watchResult.GameServerInfo);
					}
					catch (IOException e)
					{
						Logger.Log(LogLevel.Error, e, "IOException while watching game server");

						break;
					}
					catch (Exception e)
					{
						Logger.Log(LogLevel.Error, e, "Exception while watching game server");

						break;
					}
				}
			}
		}

		static void HandleEmptyResponse(HttpResponseMessage response)
		{
			if (response.IsSuccessStatusCode)
			{
				return;
			}
			else
			{
				throw CreateHttpException(response);
			}
		}

		static async Task<T> HandleResponse<T>(HttpResponseMessage response)
		{
			if (response.IsSuccessStatusCode)
			{
				return await response.Content.ReadAsAsync<T>();
			}
			else
			{
				throw CreateHttpException(response);
			}
		}

		static Exception CreateHttpException(HttpResponseMessage response)
		{
			Logger.Log(LogLevel.Error, "Encountered HTTP Exception, Status: {0}", response.StatusCode);

			return new Exception("Agones Request Failed with status code: " + response.StatusCode);
		}
	}
}
