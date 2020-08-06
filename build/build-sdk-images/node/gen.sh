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

mkdir -p ./sdks/nodejs/lib/alpha

cd /go/src/agones.dev/agones

grpc_tools_node_protoc --proto_path=${googleapis} --proto_path=${sdk} --js_out=import_style=commonjs,binary:./sdks/nodejs/lib google/api/annotations.proto google/api/http.proto
grpc_tools_node_protoc --proto_path=${googleapis} --proto_path=${sdk}/alpha --js_out=import_style=commonjs,binary:./sdks/nodejs/lib/alpha google/api/annotations.proto google/api/http.proto

grpc_tools_node_protoc --proto_path=${googleapis} --proto_path=${sdk} --grpc_out=generate_package_definition:./sdks/nodejs/lib --js_out=import_style=commonjs,binary:./sdks/nodejs/lib sdk.proto
grpc_tools_node_protoc --proto_path=${googleapis} --proto_path=${sdk}/alpha --grpc_out=generate_package_definition:./sdks/nodejs/lib/alpha --js_out=import_style=commonjs,binary:./sdks/nodejs/lib/alpha alpha.proto

header ./sdks/nodejs/lib/sdk_pb.js
header ./sdks/nodejs/lib/google/api/annotations_pb.js
header ./sdks/nodejs/lib/google/api/http_pb.js

header ./sdks/nodejs/lib/alpha/alpha_pb.js
header ./sdks/nodejs/lib/alpha/google/api/annotations_pb.js
header ./sdks/nodejs/lib/alpha/google/api/http_pb.js
