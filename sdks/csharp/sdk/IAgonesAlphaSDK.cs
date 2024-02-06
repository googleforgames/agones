// Copyright 2020 Google LLC All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

using System;
using System.Collections.Generic;
using System.Threading.Tasks;
using Grpc.Core;

namespace Agones
{
    public interface IAgonesAlphaSDK : IDisposable
    {
        Task<long> GetPlayerCapacityAsync();
        Task<Status> SetPlayerCapacityAsync(long capacity);
        Task<bool> PlayerConnectAsync(string id);
        Task<bool> PlayerDisconnectAsync(string id);
        Task<long> GetPlayerCountAsync();
        Task<bool> IsPlayerConnectedAsync(string id);
        Task<List<string>> GetConnectedPlayersAsync();
        Task<long> GetCounterCountAsync(string key);
        Task<bool> IncrementCounterAsync(string key, long amount);
        Task<bool> DecrementCounterAsync(string key, long amount);
        Task<bool> SetCounterCountAsync(string key, long amount);
        Task<long> GetCounterCapacityAsync(string key);
        Task<bool> SetCounterCapacityAsync(string key, long amount);
        Task<long> GetListCapacityAsync(string key);
        Task<bool> SetListCapacityAsync(string key, long amount);
        Task<bool> ListContainsAsync(string key, string value);
        Task<int> GetListLengthAsync(string key);
        Task<List<string>> GetListValuesAsync(string key);
        Task<bool> AppendListValueAsync(string key, string value);
        Task<bool> DeleteListValueAsync(string key, string value);
    }
}
