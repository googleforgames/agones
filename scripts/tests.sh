#!/usr/bin/env bash

function unity_editor_path(){
  local version=$1
  case "$(uname -s)" in
    CYGWIN*|MINGW32*|MSYS*|MINGW*)
      echo "C:\\Program Files\\Unity\\Hub\\Editor\\${version}\\Editor\\Unity.exe"
      ;;
    Darwin)
      echo "/Applications/Unity/Hub/Editor/${version}/Unity.app/Contents/MacOS/Unity"
      ;;
    Linux)
      echo "/opt/Unity/Hub/Editor/${version}/Editor/Unity"
      ;;
    *)
      echo "Unsupported OS"
      exit 1
      ;;
  esac
}

function agones_unity_sdk_tests(){
  local unity_editor=$1
  local project_path=$2
  local edit_mode_results_file="$project_path/agones-unity-sdk-test-results.editmode.xml"
  local play_mode_results_file="$project_path/agones-unity-sdk-test-results.playmode.xml"
  echo "Agones Unity SDK test(s)"
  "$unity_editor" -projectPath "$project_path" -runTests -testPlatform EditMode -testResults "$edit_mode_results_file"
  "$unity_editor" -projectPath "$project_path" -runTests -testPlatform PlayMode -testResults "$play_mode_results_file"
  cat "$edit_mode_results_file"
  cat "$play_mode_results_file"
}

function main() {
  repo_root=$(git rev-parse --show-toplevel)
  unity_project_path="${repo_root}/test/sdk/unity"
  unity_project_version_file="${unity_project_path}/ProjectSettings/ProjectVersion.txt"
  echo "unity_project_path: $unity_project_path"
  echo "unity_project_version_file: $unity_project_version_file"
  unity_project_version=$(grep 'm_EditorVersion:' "$unity_project_version_file" | awk '{print $2}')
  unity_editor=$(unity_editor_path $unity_project_version)

  agones_unity_sdk_tests "$unity_editor" "$unity_project_path"
}

main "$@"