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

using System;
using System.Collections.Generic;
using System.Net;
using System.Threading.Tasks;

namespace Tests.TestingEnvironment
{
    public class MockAgonesSdkServer
    {
        private HttpListener _listener;
        private bool _isRunning;
        private readonly Dictionary<string, Func<HttpListenerRequest, string>> _responseHandlers = new();
        // public MockAgonesSdkServer(Dictionary<string, Func<HttpListenerRequest, string>> responseHandlers) => _responseHandlers = responseHandlers;
        public void StartServer(string baseAddress)
        {
            _listener = new HttpListener();
            if (!baseAddress.EndsWith("/")) baseAddress += "/";
            _listener.Prefixes.Add(baseAddress); // Example: "http://localhost:9358/"
            _listener.Start();
            _isRunning = true;
            Task.Run(HandleRequests);
        }
        public void StopServer()
        {
            _isRunning = false;
            _listener.Stop();
        }
        public void RegisterResponseHandler(string path, Func<HttpListenerRequest, string> handler) => _responseHandlers[path] = handler;
        public void DeregisterResponseHandler(string path) => _responseHandlers.Remove(path);
        private void HandleRequests()
        {
            while (_isRunning)
                try
                {
                    var context = _listener.GetContext();
                    ProcessRequest(context);
                }
                catch (Exception ex)
                {
                    Console.WriteLine("Error handling request: " + ex.Message);
                }
        }
        private void ProcessRequest(HttpListenerContext context)
        {
            var request = context.Request;
            var response = context.Response;
            var responseString = GenerateResponseBasedOnRequest(request);
            byte[] buffer = System.Text.Encoding.UTF8.GetBytes(responseString);
            response.ContentLength64 = buffer.Length;
            response.OutputStream.Write(buffer, 0, buffer.Length);
            response.OutputStream.Close();
        }
        private string GenerateResponseBasedOnRequest(HttpListenerRequest request)
        {
            if (_responseHandlers.TryGetValue(request.RawUrl, out var handler))
                return handler(request);
            return "{\"status\": \"Unhandled request\"}";  // Default response
        }
    }
}
