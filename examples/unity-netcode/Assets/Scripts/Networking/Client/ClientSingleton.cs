using System.Threading.Tasks;
using UnityEngine;

public class ClientSingleton : MonoBehaviour
{
  private static ClientSingleton instance;
  public ClientGameManager GameManager { get; private set; }

  public static ClientSingleton Instance
  {
    get
    {
      if (instance != null) { return instance; }

      instance = FindObjectOfType<ClientSingleton>();

      if (instance == null)
      {
        Debug.LogError("No ClientSingleton found in scene");
        return null;
      }

      return instance;
    }
  }

  void Start()
  {
    DontDestroyOnLoad(gameObject);
  }

  public void CreateClient()
  {
    GameManager = new ClientGameManager();
  }

  private void OnDestroy()
  {
    GameManager?.Dispose();
  }
}
