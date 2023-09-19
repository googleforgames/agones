using System.Threading.Tasks;
using Agones;
using Agones.Model;
using UnityEngine;

public class ApplicationController : MonoBehaviour
{
  [SerializeField] private ClientSingleton clientPrefab;
  [SerializeField] private ServerSingleton serverPrefab;
  [SerializeField] private AgonesSdk agonesPrefab;

  // private ServerData serverData;
  private ApplicationData appData;

  private async void Start()
  {
    DontDestroyOnLoad(gameObject);

    await LaunchInMode(SystemInfo.graphicsDeviceType == UnityEngine.Rendering.GraphicsDeviceType.Null);
  }

  private async Task LaunchInMode(bool isDedicatedServer)
  {

    if (isDedicatedServer)
    {
      AgonesSdk agones = Instantiate(agonesPrefab);

      bool ok = await agones.Connect();

      if (ok)
      {
        Debug.Log("Server - Connected");
      }
      else
      {
        Debug.Log("Server - Failed to connect, exiting");
        Application.Quit(1);
      }

      ok = await agones.Ready();
      if (ok)
      {
        Debug.Log("Server - Ready");

        // serverData = new ServerData(agones);
        // await serverData.InitializeServerDataAsync();

        appData = new ApplicationData();

        ServerSingleton serverSingleton = Instantiate(serverPrefab);

        serverSingleton.CreateServer();

        serverSingleton.GameManager.StartServer();

        Debug.Log("Running in server mode");
      }
      else
      {
        Debug.Log("Server - Ready failed");
        Application.Quit();
      }
    }
    else
    {
      ClientSingleton clientSingleton = Instantiate(clientPrefab);
      clientSingleton.CreateClient();

      Debug.Log("Running client");

      clientSingleton.GameManager.GoToMainMenu();
    }
  }
}
