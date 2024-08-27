# Agones Unity Test Project

This project is intended to be used for developing the Agones Unity SDK.

## Add Agones Unity Test Project to Unity Hub

Contained within this folder is a Unity project. After opening the Unity Hub application find the "Add" button. In the file path dialog, select this directory. You will then be able to use the Unity Editor to run Agones Unity SDK tests!

# Running Tests

This project implements basic tests for the Unity SDK using the [Unity Test Framework](https://docs.unity3d.com/Packages/com.unity.test-framework@1.1/manual/index.html). These tests are PlayMode tests.

To run these tests, open the Test Framework window in the Unity editor using Window > General > Test Framwork. More information [in the Unity guide](https://docs.unity3d.com/Packages/com.unity.test-framework@1.1/manual/workflow-run-test.html).

# Connecting to the Agones SDK Server

These tests require a local SDK server to connect to. The easiest way to run a server is by [running it locally](https://agones.dev/site/docs/guides/client-sdks/local/#running-the-sdk-server). The tests expect the CountsAndLists flag to be enabled with a list called "players" and a counter called "rooms".
