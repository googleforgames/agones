using TMPro;
using UnityEngine;

public class MainMenu : MonoBehaviour
{
  [SerializeField] private TMP_InputField serverIpField;
  [SerializeField] private TMP_InputField serverPortField;

  private void Start()
  {
    if (ClientSingleton.Instance == null) { return; }

    Cursor.SetCursor(null, Vector2.zero, CursorMode.Auto);
  }

  public void StartClient()
  {
    ClientSingleton.Instance.GameManager.StartClient(serverIpField.text, int.Parse(serverPortField.text));
  }
}
