using Agones;
using Newtonsoft.Json;
using System;
using System.Threading.Tasks;

namespace AgonesTestCore
{
	class Program
	{
		static void Main(string[] args)
		{
			Console.WriteLine("Starting Agones Test");

			Run().Wait();
		}

		async static Task Run()
		{
			var serverInfo = await AgonesClient.GetGameServer();

			Console.WriteLine("Starting server: {0}", serverInfo.MetaData?.Name);

			await AgonesClient.Ready();

			bool isPingingHealth = true;

			_ = Task.Run(async () =>
			{
				while (isPingingHealth)
				{
					await AgonesClient.Health();

					await Task.Delay(2000);
				}
			});

			AgonesClient.WatchGameServer(info =>
			{
				Console.WriteLine("Info Changed:\n" + JsonConvert.SerializeObject(info, Formatting.Indented));
			});

			await Task.Delay(5000);

			await AgonesClient.SetLabel("server-id", 12345.ToString());

			await Task.Delay(5000);

			await AgonesClient.SetLabel("player-count", 5.ToString());

			await Task.Delay(5000);

			await AgonesClient.Shutdown();

			isPingingHealth = false;

			Console.WriteLine("The End...");

			Console.ReadLine();
		}
	}
}
