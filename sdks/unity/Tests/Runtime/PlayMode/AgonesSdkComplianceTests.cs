using System.Collections;
using System.Threading.Tasks;
using Agones;
using NUnit.Framework;
using Tests.TestingEnvironment;
using UnityEngine;
using UnityEngine.TestTools;

namespace Tests.Runtime.Playmode
{
    public class AgonesSdkComplianceTests
    {
        private MockAgonesSdkServer _mockSdkServer;
        private GameObject _gameObject;
        [SetUp]
        public void SetupTestEnvironment()
        {
            _mockSdkServer = new MockAgonesSdkServer();
            _mockSdkServer.StartServer("http://localhost:9358");
            _gameObject = new GameObject();
        }
        [TearDown]
        public void TearDownTestEnvironment()
        {
            _mockSdkServer.StopServer();
            Object.Destroy(_gameObject);
        }
        [UnityTest]
        public IEnumerator AgonesSdk_Ready_ShouldAlwaysSucceed()
        {
            _mockSdkServer.RegisterResponseHandler("/ready", _ => "{}");
            var sut = _gameObject.AddComponent<AgonesSdk>();
            var task = sut.Ready();
            yield return AwaitTask(task);
            Assert.IsTrue(task.Result);
            _mockSdkServer.DeregisterResponseHandler("/ready");
        }
        [UnityTest]
        public IEnumerator AgonesSdk_()
        {
            _mockSdkServer.RegisterResponseHandler("/ready", _ => "{}");
            var sut = _gameObject.AddComponent<AgonesSdk>();
            var task = sut.Ready();
            yield return AwaitTask(task);
            Assert.IsTrue(task.Result);
            _mockSdkServer.DeregisterResponseHandler("/ready");
        }
        private IEnumerator AwaitTask(Task task)
        {
            while (!task.IsCompleted)
                yield return null;
            if (task.Exception != null)
                throw task.Exception;
        }
    }
}
