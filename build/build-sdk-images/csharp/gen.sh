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

header() {
    cat /go/src/agones.dev/agones/build/boilerplate.go.txt "$1" | sponge "$1"
}

sdk=/go/src/agones.dev/agones/proto/sdk
googleapis=/go/src/agones.dev/agones/proto/googleapis
protoc_intermediate=/go/src/agones.dev/agones/sdks/csharp/proto
protoc_destination=/go/src/agones.dev/agones/sdks/csharp/sdk/generated

# Create temporary proto files
mkdir -p ${protoc_intermediate}
cp -r ${sdk} ${protoc_intermediate}

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

sed -i -e 's/import "protoc-gen-openapiv2\/options\/annotations.proto";//' ${protoc_intermediate}/sdk/alpha/alpha.proto
sed -i -e 's/option (grpc.gateway.protoc_gen_openapiv2.options.openapiv2_swagger) = {//' ${protoc_intermediate}/sdk/alpha/alpha.proto
sed -i -e 's/info: {//' ${protoc_intermediate}/sdk/alpha/alpha.proto
sed -i -e 's/title: "alpha.proto";//' ${protoc_intermediate}/sdk/alpha/alpha.proto
sed -i -z 's/version: "version not set";\n    };//' ${protoc_intermediate}/sdk/alpha/alpha.proto
sed -i -e 's/schemes: HTTP;//' ${protoc_intermediate}/sdk/alpha/alpha.proto
sed -i -e 's/consumes: "application\/json";//' ${protoc_intermediate}/sdk/alpha/alpha.proto
sed -i -z 's/produces: "application\/json";\n};//' ${protoc_intermediate}/sdk/alpha/alpha.proto
sed -i -e 's/bool bool = 1.*/bool bool = 1;/' ${protoc_intermediate}/sdk/alpha/alpha.proto

# Generate C# proto file like `Sdk.cs`
protoc -I ${googleapis} -I ${protoc_intermediate}/sdk --plugin=protoc-gen-grpc=`which grpc_csharp_plugin` --csharp_out=${protoc_destination}  ${protoc_intermediate}/sdk/sdk.proto
protoc -I ${googleapis} -I ${protoc_intermediate}/sdk/alpha --plugin=protoc-gen-grpc=`which grpc_csharp_plugin` --csharp_out=${protoc_destination} ${protoc_intermediate}/sdk/alpha/alpha.proto

# Generate grpc file like `SdkGrpc.cs`
protoc -I ${googleapis} -I ${protoc_intermediate}/sdk --plugin=protoc-gen-grpc=`which grpc_csharp_plugin` --grpc_out=${protoc_destination} ${protoc_intermediate}/sdk/sdk.proto
protoc -I ${googleapis} -I ${protoc_intermediate}/sdk/alpha --plugin=protoc-gen-grpc=`which grpc_csharp_plugin`  --grpc_out=${protoc_destination} ${protoc_intermediate}/sdk/alpha/alpha.proto

cd ${protoc_destination}
header Alpha.cs
header AlphaGrpc.cs
header Sdk.cs
header SdkGrpc.cs

rm -rf ${protoc_intermediate}