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

mkdir -p /go/src/
cp -r /go/src/agones.dev/agones/vendor/* /go/src/

go install github.com/grpc-ecosystem/grpc-gateway/protoc-gen-grpc-gateway
go install github.com/grpc-ecosystem/grpc-gateway/protoc-gen-swagger

googleapis=/go/src/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis

cd /go/src/agones.dev/agones
protoc -I ${googleapis} -I . sdk.proto --go_out=plugins=grpc:pkg/sdk
protoc -I ${googleapis} -I . sdk.proto --grpc-gateway_out=logtostderr=true:pkg/sdk
protoc -I ${googleapis} -I . sdk.proto --swagger_out=logtostderr=true:.
jq 'del(.schemes[] | select(. == "https"))' sdk.swagger.json > sdk.swagger.temp.json
mv sdk.swagger.temp.json sdk.swagger.json

cat ./build/boilerplate.go.txt ./pkg/sdk/sdk.pb.go >> ./sdk.pb.go
cat ./build/boilerplate.go.txt ./pkg/sdk/sdk.pb.gw.go >> ./sdk.pb.gw.go

goimports -w ./sdk.pb.go
goimports -w ./sdk.pb.gw.go

mv ./sdk.pb.go ./pkg/sdk
mv ./sdk.pb.gw.go ./pkg/sdk

