using System;
using Unity.Netcode;
using Unity.Netcode.Transports.UTP;
using UnityEngine;
using UnityEngine.SceneManagement;

public class ServerGameManager : IDisposable
{

  private NetworkManager networkManager;
  private string serverIP;
  private int serverPort;
  private const string GameSceneName = "GameTest";

  public ServerGameManager(string serverIP, int serverPort, NetworkManager networkManager)
  {
    this.serverIP = serverIP;
    this.serverPort = serverPort;

    this.networkManager = networkManager;
  }

    public void GoToGame()
  {
    SceneManager.LoadScene(GameSceneName);
  }

  public void StartServer()
  {
    ConnectServer();
  }

  private void ConnectServer()
  {
    UnityTransport transport = networkManager.gameObject.GetComponent<UnityTransport>();
    transport.SetConnectionData(serverIP, (ushort)serverPort, "0.0.0.0");
    networkManager.StartServer();

    GoToGame();
  }

  public void Dispose()
  {

  }
}
