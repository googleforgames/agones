using Unity.Netcode;
using UnityEngine;

public class PlayerMovement : NetworkBehaviour
{
  [Header("References")]
  [SerializeField] private InputReader inputReader;
  [SerializeField] private Transform bodyTransform;
  // [SerializeField] private Rigidbody rb;

  [Header("Settings")]
  [SerializeField] private float movementSpeed = 4f;
  [SerializeField] private float turningRate = 270f;

  private Vector3 previousMovementInput;

  public override void OnNetworkSpawn()
  {
    if (!IsOwner) { return; }

    inputReader.MoveEvent += HandleMove;
  }

  public override void OnNetworkDespawn()
  {
    if (!IsOwner) { return; }

    inputReader.MoveEvent -= HandleMove;
  }

  // Update is called once per frame
  private void Update()
  {
    if (!IsOwner) { return; }

    Vector3 movement = Vector3.ClampMagnitude(previousMovementInput, 1);

    transform.Translate(movement * movementSpeed * Time.deltaTime);
  }

  private void HandleMove(Vector3 movementInput)
  {
    previousMovementInput = movementInput;
  }
}
