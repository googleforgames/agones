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
    cat /go/src/agones.dev/agones/build/boilerplate.go.txt "$1" | sponge "$1"
}

export GO111MODULE=on

mkdir -p /go/src/
cp -r /go/src/agones.dev/agones/vendor/* /go/src/

cd /go/src/agones.dev/agones
go install -mod=vendor github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway
go install -mod=vendor github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2

outputpath=pkg/allocation/go
protopath=proto/allocation
googleapis=/go/src/agones.dev/agones/proto/googleapis
gatewaygrpc=/go/src/agones.dev/agones/proto/grpc-gateway
protofile=${protopath}/allocation.proto

rm ./${outputpath}/allocation.pb.go || true
rm ./${outputpath}/allocation.gw.pb.go || true
rm ./${outputpath}/allocation_grpc.pb.go || true

# generate the go code
protoc -I ${googleapis} -I ${gatewaygrpc} -I . -I ./vendor ${protofile} --go_out=proto --go-grpc_opt=require_unimplemented_servers=false --go-grpc_out=proto

# generate grpc gateway
protoc -I ${googleapis} -I ${gatewaygrpc} -I . -I ./vendor ${protofile} --go_out=proto --grpc-gateway_out=logtostderr=true:proto

# generate openapi v2
protoc -I ${googleapis} -I ${gatewaygrpc} -I . -I ./vendor ${protofile} --openapiv2_opt=logtostderr=true,simple_operation_ids=true,disable_default_errors=true --openapiv2_out=logtostderr=true:.

jq 'del(.schemes[] | select(. == "http"))' ./${protopath}/allocation.swagger.json > ./${outputpath}/allocation.swagger.json

rm ${protopath}/allocation.swagger.json

header ${protopath}/allocation.pb.go
header ${protopath}/allocation.pb.gw.go
header ${protopath}/allocation_grpc.pb.go

mv ${protopath}/allocation.pb.go ${outputpath}/allocation.pb.go
mv ${protopath}/allocation.pb.gw.go ${outputpath}/allocation.pb.gw.go
mv ${protopath}/allocation_grpc.pb.go ${outputpath}/allocation_grpc.pb.go

goimports -w ./${outputpath}
