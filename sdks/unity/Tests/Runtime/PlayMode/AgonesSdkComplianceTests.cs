// Copyright 2022 Google LLC
// All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
        private IEnumerator AwaitTask(Task task)
        {
            while (!task.IsCompleted)
                yield return null;
            if (task.Exception != null)
                throw task.Exception;
        }
    }
}
