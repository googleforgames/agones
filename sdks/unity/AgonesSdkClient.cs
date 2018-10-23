// Copyright 2018 Google Inc. All Rights Reserved.
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

using Newtonsoft.Json;
using System.Collections;
using System.Text;
using UnityEngine;
using UnityEngine.Networking;

namespace Agones.SDK
{
    /// <summary>
    /// JSON payload object
    /// </summary>
    internal class KeyValueMessage
    {
        public string key;
        public string value;
    }

    /// <summary>
    /// Agones SDK Unity Client
    /// </summary>
    public class AgonesSdkClient : MonoBehaviour
    {
        private const string emptyPayload = "{}";
        private static byte[] emptyPayloadBytes;
        private WaitForSeconds wait = new WaitForSeconds(0.5f);
        private float updateInterval = 0.5f;

        /// <summary>
        /// Gets or sets the rate at which Health() should be called
        /// </summary>
        public float UpdateInterval
        {
            get
            {
                return this.updateInterval;
            }
            set
            {
                this.updateInterval = value;
                this.wait = new WaitForSeconds(value);
            }
        }

        /// <summary>
        /// Unity Start
        /// </summary>
        private void Start()
        {
            AgonesSdkClient.emptyPayloadBytes = Encoding.UTF8.GetBytes(AgonesSdkClient.emptyPayload);

            if (LaunchOptions.AgonesEnabled)
            {
                this.StartCoroutine(this.UpdateLoop());
                MainContext.Instance.InstanceBinder.Bind<AgonesSdkClient>(this);
            }
        }

        /// <summary>
        /// This tells Agones that the Game Server is ready to take player connections.
        /// Once a Game Server has specified that it is Ready, then the Kubernetes GameServer
        /// record will be moved to the Ready state, and the details for its public address
        /// and connection port will be populated.
        /// </summary>
        public void Ready()
        {
            AgonesSdkClient.EmptyPost("http://localhost:59358/ready");
        }

        /// <summary>
        /// This sends a single ping to designate that the Game Server is alive and healthy.
        /// Failure to send pings within the configured thresholds will result in the GameServer
        /// being marked as Unhealthy.
        /// </summary>
        public void Health()
        {
            AgonesSdkClient.EmptyPost("http://localhost:59358/health");
        }

        /// <summary>
        /// This tells Agones to shut down the currently running game server. The GameServer
        /// state will be set Shutdown and the backing Pod will be deleted, if they have not
        /// shut themselves down already.
        /// </summary>
        public void Shutdown()
        {
            AgonesSdkClient.EmptyPost("http://localhost:59358/shutdown");
        }

        /// <summary>
        /// Request agones to set a label key/value pair
        /// </summary>
        /// <param name="key">The key name</param>
        /// <param name="value">The value</param>
        public void SetLabel(string key, string value)
        {
            this.SetProperty(key, value, "http://localhost:59358/metadata/label");
        }

        /// <summary>
        /// Requests Agones to set an annotation key/value pair
        /// </summary>
        /// <param name="key">The key value</param>
        /// <param name="value">The value</param>
        public void SetAnnotation(string key, string value)
        {
            this.SetProperty(key, value, "http://localhost:59358/metadata/annotation");
        }

        /// <summary>
        /// Co-routine that calls the health endpoint at UpdateHertz rate
        /// </summary>
        private IEnumerator UpdateLoop()
        {
            while (true)
            {
                this.Health();
                yield return this.wait;
            }
        }

        /// <summary>
        /// Makes a call to set a key/value pair
        /// </summary>
        /// <param name="key">The key name</param>
        /// <param name="value">The value</param>
        /// <param name="uri">The URI to post the key/value pair</param>
        private void SetProperty(string key, string value, string uri)
        {
            KeyValueMessage msg = new KeyValueMessage()
            {
                key = key,
                value = value
            };

            string payload = JsonConvert.SerializeObject(msg);

            UnityWebRequest request = UnityWebRequest.Put(uri, payload);
            request.uploadHandler = new UploadHandlerRaw(Encoding.UTF8.GetBytes(payload));
            AgonesSdkClient.ConfigureRequest(request);
            request.SendWebRequest();
        }

        /// <summary>
        /// Posts an empty JSON blob to the given uri
        /// </summary>
        /// <param name="uri">The URI to HTTP POST an empty blob</param>
        private static void EmptyPost(string uri)
        {
            UnityWebRequest request = UnityWebRequest.Post(uri, AgonesSdkClient.emptyPayload);
            request.uploadHandler = new UploadHandlerRaw(AgonesSdkClient.emptyPayloadBytes);
            AgonesSdkClient.ConfigureRequest(request);
            request.SendWebRequest();
        }

        /// <summary>
        /// Configures the request with content types and timeout
        /// </summary>
        /// <param name="request">The request to configure</param>
        private static void ConfigureRequest(UnityWebRequest request)
        {
            request.SetRequestHeader("Content-Type", "application/json");
            request.timeout = 500;
        }
    }
}