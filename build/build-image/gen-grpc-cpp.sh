#!/usr/bin/env bash

# Copyright 2017 Google Inc. All Rights Reserved.
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

cd /go/src/github.com/agonio/agon
protoc -I . --grpc_out=./sdks/cpp --plugin=protoc-gen-grpc=`which grpc_cpp_plugin` sdk.proto
protoc -I . --cpp_out=./sdks/cpp sdk.proto
mkdir /tmp/cpp
find ./sdks/cpp/ -type f \( -name '*.pb.cc' -or -name '*.pb.h' \) -printf "%f\n" | xargs -I@ bash -c "cat ./build/boilerplate.go.txt ./sdks/cpp/@ >> /tmp/cpp/@"
# already has a header, so we'll remove it
rm /tmp/cpp/sdk.grpc.pb.h
mv /tmp/cpp/* ./sdks/cpp/

