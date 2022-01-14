#!/usr/bin/env bash

# Copyright 2020 Google LLC All Rights Reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -ex

sdk=/go/src/agones.dev/agones/proto/sdk
googleapis=/go/src/agones.dev/agones/proto/googleapis
protoc_destination=/go/src/agones.dev/agones/sdks/csharp/sdk/generated

# Generate C# proto file `Sdk.cs`
protoc --csharp_out=${protoc_destination} -I ${sdk} -I ${googleapis} --plugin=protoc-gen-grpc=`which grpc_csharp_plugin` sdk.proto
protoc --csharp_out=${protoc_destination} -I ${sdk}/alpha -I ${googleapis} --plugin=protoc-gen-grpc=`which grpc_csharp_plugin` alpha.proto
# Generate proto stub?
protoc --grpc_out=${protoc_destination} -I ${sdk} -I ${googleapis} --plugin=protoc-gen-grpc=`which grpc_csharp_plugin` sdk.proto
protoc --grpc_out=${protoc_destination} -I ${sdk}/alpha -I ${googleapis} --plugin=protoc-gen-grpc=`which grpc_csharp_plugin` alpha.proto
