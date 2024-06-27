# Using Unity Test Project

This project implements basic tests for the Unity SDK using the [Unity Test Framework](https://docs.unity3d.com/Packages/com.unity.test-framework@1.1/manual/index.html). These tests are PlayMode tests, and use the Agones GameObject in the SampleScene.

To run these tests, open the Test Framework window in the Unity editor using Window > General > Test Framwork. More information [in the unity guide](https://docs.unity3d.com/Packages/com.unity.test-framework@1.1/manual/workflow-run-test.html).

# Connecting to the Agones SDK Server
These tests require a local SDK server to connect to. The easiest way to run a server is by [running it locally](https://agones.dev/site/docs/guides/client-sdks/local/#running-the-sdk-server). The tests expect the CountsAndLists flag to be enabled with a list called "players" and a counter called "rooms".