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

set -x

header() {
    cat ./build/boilerplate.go.txt $1 >> ./tmp.js && mv ./tmp.js $1
}

googleapis=/go/src/agones.dev/agones/vendor/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis

cd /go/src/agones.dev/agones

protoc -I ${googleapis} -I . --grpc_out=./sdks/nodejs/lib --plugin=protoc-gen-grpc=`which grpc_node_plugin` sdk.proto
protoc -I ${googleapis} -I . --js_out=import_style=commonjs,binary:./sdks/nodejs/lib sdk.proto ${googleapis}/google/api/annotations.proto ${googleapis}/google/api/http.proto 

header ./sdks/nodejs/lib/sdk_pb.js
header ./sdks/nodejs/lib/google/api/annotations_pb.js
header ./sdks/nodejs/lib/google/api/http_pb.js
