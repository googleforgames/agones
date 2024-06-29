using System.Collections;
using System.Threading.Tasks;
using Agones;
using UnityEngine;
using UnityEngine.Assertions;
using UnityEngine.TestTools;

namespace Tests.Runtime.Playmode
{
    public class AgonesSdkComplianceTests
    {
        [UnityTest]
        public IEnumerator AgonesSdk_Ready_AlwaysReturnsTrue()
        {
            var gameObject = new GameObject();
            var sut = gameObject.AddComponent<AgonesSdk>();
            var task = sut.Ready();
            yield return AwaitTask(task);
            Assert.IsTrue(task.Result);
        }
        private IEnumerator AwaitTask(Task task)
        {
            while (!task.IsCompleted)
            {
                yield return null;
            }
            if (task.Exception != null)
            {
                throw task.Exception;
            }
        }
    }
}
