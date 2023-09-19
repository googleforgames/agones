using System;
using Unity.Netcode;
using Unity.Netcode.Transports.UTP;
using UnityEngine.SceneManagement;

public class ClientGameManager : IDisposable
{

  private const string GameSceneName = "GameTest";
  private const string MainMenuSceneName = "Menu";

  public void GoToGame()
  {
    SceneManager.LoadScene(GameSceneName);
  }

  public void GoToMainMenu()
  {
    SceneManager.LoadScene(MainMenuSceneName);
  }

  public void StartClient(string ip, int port)
  {
    UnityTransport transport = NetworkManager.Singleton.GetComponent<UnityTransport>();
    transport.SetConnectionData(ip, (ushort)port);

    ConnectClient();
  }

  private void ConnectClient()
  {
    NetworkManager.Singleton.StartClient();

    GoToGame();
  }


  public void Dispose()
  {

  }
}
