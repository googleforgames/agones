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

proto=/go/src/agones.dev/agones/proto
sdk=${proto}/sdk
googleapis=${proto}/googleapis
protoc_intermediate=/go/src/agones.dev/agones/sdks/csharp/proto

echo "Copying protobuffers to csharp sdk"
mkdir -p ${protoc_intermediate}
cp -r ${sdk} ${protoc_intermediate}
cp -r ${googleapis} ${protoc_intermediate}

# Remove protoc-gen-openapiv2 definitions because C# package doesn't support grpc-gateway
sed -i -e 's/import "protoc-gen-openapiv2\/options\/annotations.proto";//' ${protoc_intermediate}/sdk/sdk.proto
sed -i -e 's/option (grpc.gateway.protoc_gen_openapiv2.options.openapiv2_swagger) = {//' ${protoc_intermediate}/sdk/sdk.proto
sed -i -e 's/info: {//' ${protoc_intermediate}/sdk/sdk.proto
sed -i -e 's/title: "sdk.proto";//' ${protoc_intermediate}/sdk/sdk.proto
sed -i -z 's/version: "version not set";\n    };//' ${protoc_intermediate}/sdk/sdk.proto
sed -i -e 's/schemes: HTTP;//' ${protoc_intermediate}/sdk/sdk.proto
sed -i -e 's/consumes: "application\/json";//' ${protoc_intermediate}/sdk/sdk.proto
sed -i -z 's/produces: "application\/json";\n};//' ${protoc_intermediate}/sdk/sdk.proto
sed -i -e 's/bool disabled = 1.*/bool disabled = 1;/' ${protoc_intermediate}/sdk/sdk.proto
sed -i -e 's/^ *$//' ${protoc_intermediate}/sdk/sdk.proto

sed -i -e 's/import "protoc-gen-openapiv2\/options\/annotations.proto";//' ${protoc_intermediate}/sdk/alpha/alpha.proto
sed -i -e 's/option (grpc.gateway.protoc_gen_openapiv2.options.openapiv2_swagger) = {//' ${protoc_intermediate}/sdk/alpha/alpha.proto
sed -i -e 's/info: {//' ${protoc_intermediate}/sdk/alpha/alpha.proto
sed -i -e 's/title: "alpha.proto";//' ${protoc_intermediate}/sdk/alpha/alpha.proto
sed -i -z 's/version: "version not set";\n    };//' ${protoc_intermediate}/sdk/alpha/alpha.proto
sed -i -e 's/schemes: HTTP;//' ${protoc_intermediate}/sdk/alpha/alpha.proto
sed -i -e 's/consumes: "application\/json";//' ${protoc_intermediate}/sdk/alpha/alpha.proto
sed -i -z 's/produces: "application\/json";\n};//' ${protoc_intermediate}/sdk/alpha/alpha.proto
sed -i -e 's/bool bool = 1.*/bool bool = 1;/' ${protoc_intermediate}/sdk/alpha/alpha.proto
sed -i -e 's/^ *$//' ${protoc_intermediate}/sdk/alpha/alpha.proto

echo "csharp code is generated at build time"
