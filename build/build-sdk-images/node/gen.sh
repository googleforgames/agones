#!/usr/bin/env bash

# Copyright 2019 Google LLC All Rights Reserved.
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

header() {
    cat ./build/boilerplate.go.txt $1 >> ./tmp.js && mv ./tmp.js $1
}

sdk=/go/src/agones.dev/agones/proto/sdk
googleapis=/go/src/agones.dev/agones/proto/googleapis

cd /go/src/agones.dev/agones

protoc -I ${googleapis} -I ${sdk} --grpc_out=./sdks/nodejs/lib --plugin=protoc-gen-grpc=`which grpc_node_plugin` sdk.proto
protoc -I ${googleapis} -I ${sdk} --js_out=import_style=commonjs,binary:./sdks/nodejs/lib sdk.proto ${googleapis}/google/api/annotations.proto ${googleapis}/google/api/http.proto

header ./sdks/nodejs/lib/sdk_pb.js
header ./sdks/nodejs/lib/google/api/annotations_pb.js
header ./sdks/nodejs/lib/google/api/http_pb.js
