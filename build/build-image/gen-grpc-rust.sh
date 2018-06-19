#!/usr/bin/env bash

# Copyright 2018 Google Inc. All Rights Reserved.
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

googleapis=/go/src/agones.dev/agones/vendor/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis

cd /go/src/agones.dev/agones
protoc \
    -I ${googleapis} -I . sdk.proto \
    --rust_out=sdks/rust/src/grpc --grpc_out=sdks/rust/src/grpc \
    --plugin=protoc-gen-grpc=`which grpc_rust_plugin` \

cat ./build/boilerplate.go.txt ./sdks/rust/src/grpc/sdk.rs >> ./sdk.rs
cat ./build/boilerplate.go.txt ./sdks/rust/src/grpc/sdk_grpc.rs >> ./sdk_grpc.rs
mv ./sdk.rs ./sdks/rust/src/grpc/
mv ./sdk_grpc.rs ./sdks/rust/src/grpc/