using System;
using UnityEditor;
using UnityEditor.SceneManagement;

[InitializeOnLoad]
public static class StartupSceneLoader
{
  static StartupSceneLoader()
  {
    EditorApplication.playModeStateChanged += LoadStartupScene;
  }

  private static void LoadStartupScene(PlayModeStateChange state)
  {
    if (state == PlayModeStateChange.ExitingEditMode)
    {
      EditorSceneManager.SaveCurrentModifiedScenesIfUserWantsTo();
    }

    if (state == PlayModeStateChange.EnteredPlayMode)
    {
      if (EditorSceneManager.GetActiveScene().buildIndex != 0)
      {
        EditorSceneManager.LoadScene(0);
      }
    }
  }
}
