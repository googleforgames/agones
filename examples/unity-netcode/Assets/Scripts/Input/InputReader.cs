using System;
using UnityEngine;
using UnityEngine.InputSystem;
using static Controls;

[CreateAssetMenu(fileName = "New Input Reader", menuName = "Input/Input Reader")]
public class InputReader : ScriptableObject, IPlayerActions
{
  public event Action<Vector3> MoveEvent;

  private Controls controls;
  private void OnEnable()
  {
    if (controls == null)
    {
      controls = new Controls();
      controls.Player.SetCallbacks(this);
    }
    controls.Player.Enable();
  }

  public void OnMove(InputAction.CallbackContext context)
  {
    MoveEvent?.Invoke(context.ReadValue<Vector3>());
  }
}