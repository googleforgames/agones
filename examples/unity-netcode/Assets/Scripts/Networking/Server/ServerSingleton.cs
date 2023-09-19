using System.Threading.Tasks;
using Unity.Netcode;
using UnityEngine;

public class ServerSingleton : MonoBehaviour
{
  // private ServerData serverData;
  private static ServerSingleton instance;
  public ServerGameManager GameManager { get; private set; }

  public static ServerSingleton Instance
  {
    get
    {
      if (instance != null) { return instance; }

      instance = FindObjectOfType<ServerSingleton>();

      if (instance == null)
      {
        Debug.LogError("No ServerSingleton found in scene");
        return null;
      }

      return instance;
    }
  }

  void Start()
  {
    DontDestroyOnLoad(gameObject);
  }

  public void CreateServer()
  {
    GameManager = new ServerGameManager(
      ApplicationData.IP(),
      ApplicationData.Port(),
      NetworkManager.Singleton
    );
  }

  private void OnDestroy()
  {
    GameManager?.Dispose();
  }
}
