// Copyright 2019 Google LLC
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
        /// <summary>
        /// Interval of the server sending a health ping to the Agones sidecar.
        /// </summary>
        [Range(0.01f, 5)]
        public float healthIntervalSecond = 5.0f;

        /// <summary>
        /// Whether the server sends a health ping to the Agones sidecar.
        /// </summary>
        public bool healthEnabled = true;

        /// <summary>
        /// Debug Logging Enabled. Debug logging for development of this Plugin.
        /// </summary>
        public bool logEnabled = false;

        private const string sidecarAddress = "http://localhost:59358";
        private readonly CancellationTokenSource cancellationTokenSource = new CancellationTokenSource();

        private struct KeyValueMessage
        {
            public string key;
            public string value;
            public KeyValueMessage(string k, string v) => (key, value) = (k, v);
        }

        #region Unity Methods
        // Use this for initialization.
        private void Start()
        {
            HealthCheckAsync();
        }

        private void OnApplicationQuit()
        {
            cancellationTokenSource.Dispose();
        }
        #endregion

        #region AgonesRestClient Public Methods
        /// <summary>
        /// Marks this Game Server as ready to receive connections.
        /// </summary>
        /// <returns>
        /// A task that represents the asynchronous operation and returns true if the request was successful.
        /// </returns>
        public async Task<bool> Ready()
        {
            return await SendRequestAsync("/ready", "{}");
        }

        /// <summary>
        /// Marks this Game Server as ready to shutdown.
        /// </summary>
        /// <returns>
        /// A task that represents the asynchronous operation and returns true if the request was successful.
        /// </returns>
        public async Task<bool> Shutdown()
        {
            return await SendRequestAsync("/shutdown", "{}");
        }

        /// <summary>
        /// Marks this Game Server as Allocated.
        /// </summary>
        /// <returns>
        /// A task that represents the asynchronous operation and returns true if the request was successful.
        /// </returns>
        public async Task<bool> Allocate()
        {
            return await SendRequestAsync("/allocate", "{}");
        }

        /// <summary>
        /// Set a metadata label that is stored in k8s.
        /// </summary>
        /// <param name="key">label key</param>
        /// <param name="value">label value</param>
        /// <returns>
        /// A task that represents the asynchronous operation and returns true if the request was successful.
        /// </returns>
        public async Task<bool> SetLabel(string key, string value)
        {
            string json = JsonUtility.ToJson(new KeyValueMessage(key, value));
            return await SendRequestAsync("/metadata/label", json, UnityWebRequest.kHttpVerbPUT);
        }

        /// <summary>
        /// Set a metadata annotation that is stored in k8s.
        /// </summary>
        /// <param name="key">annotation key</param>
        /// <param name="value">annotation value</param>
        /// <returns>
        /// A task that represents the asynchronous operation and returns true if the request was successful.
        /// </returns>
        public async Task<bool> SetAnnotation(string key, string value)
        {
            string json = JsonUtility.ToJson(new KeyValueMessage(key, value));
            return await SendRequestAsync("/metadata/annotation", json, UnityWebRequest.kHttpVerbPUT);
        }
        #endregion

        #region AgonesRestClient Private Methods
        private async void HealthCheckAsync()
        {
            while (healthEnabled)
            {
                await Task.Delay(TimeSpan.FromSeconds(healthIntervalSecond));

                try
                {
                    await SendRequestAsync("/health", "{}");
                }
                catch (ObjectDisposedException)
                {
                    break;
                }
            }
        }

        private async Task<bool> SendRequestAsync(string api, string json, string method = UnityWebRequest.kHttpVerbPOST)
        {
            // To prevent that an async method leaks after destroying this gameObject.
            cancellationTokenSource.Token.ThrowIfCancellationRequested();

            var req = new UnityWebRequest(sidecarAddress + api, method)
            {
                uploadHandler = new UploadHandlerRaw(Encoding.UTF8.GetBytes(json)),
                downloadHandler = new DownloadHandlerBuffer()
            };
            req.SetRequestHeader("Content-Type", "application/json");

            await new AgonesAsyncOperationWrapper(req.SendWebRequest());

            bool ok = req.responseCode == (long)System.Net.HttpStatusCode.OK;

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

        private void Log(object message)
        {
            if (!logEnabled)
            {
                return;
            }

            Debug.Log(message);
        }
        #endregion

        #region AgonesRestClient Nested Classes
        private class AgonesAsyncOperationWrapper
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

        private class AgonesAsyncOperationAwaiter : INotifyCompletion
        {
            private UnityWebRequestAsyncOperation asyncOp;
            private Action continuation;
            public bool IsCompleted => asyncOp.isDone;

            public AgonesAsyncOperationAwaiter(AgonesAsyncOperationWrapper wrapper)
            {
                asyncOp = wrapper.AsyncOp;
                asyncOp.completed += OnRequestCompleted;
            }

            // C# Awaiter Pattern requires that the GetAwaiter method has GetResult(),
            // And AgonesAsyncOperationAwaiter does not return a value in this case.
            public void GetResult()
            {
                asyncOp.completed -= OnRequestCompleted;
            }

            public void OnCompleted(Action continuation)
            {
                this.continuation = continuation;
            }

            private void OnRequestCompleted(AsyncOperation _)
            {
                continuation?.Invoke();
                continuation = null;
            }
        }
        #endregion
    }
}
