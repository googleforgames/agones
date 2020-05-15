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

export GO111MODULE=on

mkdir -p /go/src/
cp -r /go/src/agones.dev/agones/vendor/* /go/src/

cd /go/src/agones.dev/agones
go install -mod=vendor github.com/grpc-ecosystem/grpc-gateway/protoc-gen-grpc-gateway
go install -mod=vendor github.com/grpc-ecosystem/grpc-gateway/protoc-gen-swagger

outputpath=pkg/allocation/go
protopath=proto/allocation
googleapis=/go/src/agones.dev/agones/proto/googleapis
protofile=${protopath}/allocation.proto


protoc -I ${googleapis} -I . -I ./vendor ${protofile} --go_out=plugins=grpc:.
protoc -I ${googleapis} -I . -I ./vendor ${protofile} --grpc-gateway_out=logtostderr=true:.
protoc -I ${googleapis} -I . -I ./vendor ${protofile} --swagger_out=logtostderr=true:.
jq 'del(.schemes[] | select(. == "http"))' ./${protopath}/allocation.swagger.json > ./${outputpath}/allocation.swagger.json

cat ./build/boilerplate.go.txt ./${protopath}/allocation.pb.go >> ./allocation.pb.go
cat ./build/boilerplate.go.txt ./${protopath}/allocation.pb.gw.go >> ./allocation.pb.gw.go

goimports -w ./allocation.pb.go
goimports -w ./allocation.pb.gw.go

mv ./allocation.pb.go ./${outputpath}/allocation.pb.go
mv ./allocation.pb.gw.go ./${outputpath}/allocation.gw.pb.go

rm ./${protopath}/allocation.pb.go
rm ./${protopath}/allocation.pb.gw.go
rm ./${protopath}/allocation.swagger.json
