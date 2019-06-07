// Copyright 2019 Google Inc. All Rights Reserved.
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

using System;
using System.Runtime.CompilerServices;
using System.Text;
using System.Threading;
using System.Threading.Tasks;
using UnityEngine;
using UnityEngine.Networking;

namespace Agones
{
    /// <summary>
    /// Agones SDK for Unity.
    /// </summary>
    public class AgonesSdk : MonoBehaviour
    {
        [Range(1, 5)]
        public float healthIntervalSecond = 5.0f;
        public bool healthEnabled = true;
        public bool logEnabled = false;

        private const string sidecarAddress = "http://localhost:59358";
        private readonly CancellationTokenSource cancellationTokenSource = new CancellationTokenSource();

        private float CurrentHealthTime { get; set; } = 0;

        private struct KeyValueMessage
        {
            public string key;
            public string value;
            public KeyValueMessage(string k, string v) => (key, value) = (k, v);
        }

        #region Unity Methods
        // Use this for initialization
        void Start()
        {
            CurrentHealthTime = 0;
        }

        // Update is called once per frame
        void Update()
        {
            if (!healthEnabled) { return; }

            CurrentHealthTime += Time.unscaledDeltaTime;
            if (CurrentHealthTime >= healthIntervalSecond)
            {
                Health();
                CurrentHealthTime = 0;
            }
        }

        void OnApplicationQuit()
        {
            cancellationTokenSource.Dispose();
        }
        #endregion

        #region AgonesRestClient Public Methods
        /// <summary>
        /// Marks this Game Server as ready to receive connections
        /// </summary>
        /// <returns>
        /// A task that represents the asynchronous operation.
        /// The task result contains that the request is success or failure.
        /// </returns>
        public async Task<bool> Ready()
        {
            return await SendRequestAsync("/ready", "{}");
        }

        /// <summary>
        /// Marks this Game Server as ready to shutdown
        /// </summary>
        /// <returns>
        /// A task that represents the asynchronous operation.
        /// The task result contains that the request is success or failure.
        /// </returns>
        public async Task<bool> Shutdown()
        {
            return await SendRequestAsync("/shutdown", "{}");
        }

        /// <summary>
        /// Marks this Game Server as Allocated
        /// </summary>
        /// <returns>
        /// A task that represents the asynchronous operation.
        /// The task result contains that the request is success or failure.
        /// </returns>
        public async Task<bool> Allocate()
        {
            return await SendRequestAsync("/allocate", "{}");
        }

        /// <summary>
        /// Set a metadata label that is stored in k8s
        /// </summary>
        /// <param name="key">label key</param>
        /// <param name="value">label value</param>
        /// <returns>
        /// A task that represents the asynchronous operation.
        /// The task result contains that the request is success or failure.
        /// </returns>
        public async Task<bool> SetLabel(string key, string value)
        {
            string json = JsonUtility.ToJson(new KeyValueMessage(key, value));
            return await SendRequestAsync("/metadata/label", json, UnityWebRequest.kHttpVerbPUT);
        }

        /// <summary>
        /// Set a metadata annotation that is stored in k8s
        /// </summary>
        /// <param name="key">annotation key</param>
        /// <param name="value">annotation value</param>
        /// <returns>
        /// A task that represents the asynchronous operation.
        /// The task result contains that the request is success or failure.
        /// </returns>
        public async Task<bool> SetAnnotation(string key, string value)
        {
            string json = JsonUtility.ToJson(new KeyValueMessage(key, value));
            return await SendRequestAsync("/metadata/annotation", json, UnityWebRequest.kHttpVerbPUT);
        }
        #endregion

        #region AgonesRestClient Private Methods
        void Health()
        {
            _ = SendRequestAsync("/health", "{}");
        }

        async Task<bool> SendRequestAsync(string api, string json, string method = UnityWebRequest.kHttpVerbPOST)
        {
            // To prevent that an async method leaks after destroying this gameObject
            cancellationTokenSource.Token.ThrowIfCancellationRequested();

            var req = new UnityWebRequest(sidecarAddress + api, method)
            {
                uploadHandler = new UploadHandlerRaw(Encoding.UTF8.GetBytes(json)),
                downloadHandler = new DownloadHandlerBuffer()
            };
            req.SetRequestHeader("Content-Type", "application/json");

            await new AgonesAsyncOperationWrapper(req.SendWebRequest());

            bool ok = req.responseCode == 200;

            if (ok)
            {
                Log($"Agones SendRequest ok: {api} {req.downloadHandler.text}");
            }
            else
            {
                Log($"Agones SendRequest failed: {api} {req.error}");
            }

            return ok;
        }

        void Log(object message)
        {
            if (!logEnabled) { return; }

#if UNITY_EDITOR
            Debug.Log(message);
#else
            Console.WriteLine(message);
#endif
        }
        #endregion

        #region AgonesRestClient Nested Classes
        class AgonesAsyncOperationWrapper
        {
            public UnityWebRequestAsyncOperation AsyncOp { get; }
            public AgonesAsyncOperationWrapper(UnityWebRequestAsyncOperation unityOp)
            {
                AsyncOp = unityOp;
            }

            public AgonesAsyncOperationAwaiter GetAwaiter()
            {
                return new AgonesAsyncOperationAwaiter(this);
            }
        }

        class AgonesAsyncOperationAwaiter : INotifyCompletion
        {
            private UnityWebRequestAsyncOperation asyncOp;
            private Action continuation;

            public AgonesAsyncOperationAwaiter(AgonesAsyncOperationWrapper wrapper)
            {
                asyncOp = wrapper.AsyncOp;
                asyncOp.completed += OnRequestCompleted;
            }

            public bool IsCompleted => asyncOp.isDone;

            public void GetResult()
            {
                asyncOp.completed -= OnRequestCompleted;

                // remove references
                asyncOp = null;
                continuation = null;
            }

            public void OnCompleted(Action continuation)
            {
                this.continuation = continuation;
            }

            private void OnRequestCompleted(AsyncOperation _)
            {
                if (continuation != null)
                {
                    continuation();
                }
            }
        }
        #endregion
    }
}
