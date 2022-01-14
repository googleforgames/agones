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

sdk=/go/src/agones.dev/agones/proto/sdk
googleapis=/go/src/agones.dev/agones/proto/googleapis

export GO111MODULE=on

mkdir -p /go/src/
cp -r /go/src/agones.dev/agones/vendor/* /go/src/

cd /go/src/agones.dev/agones
go install -mod=vendor github.com/grpc-ecosystem/grpc-gateway/protoc-gen-grpc-gateway
go install -mod=vendor github.com/grpc-ecosystem/grpc-gateway/protoc-gen-swagger

mkdir -p ./pkg/sdk/{alpha,beta} || true

rm ./pkg/sdk/beta/beta.pb.gw.go || true
rm ./pkg/sdk/alpha/alpha.pb.gw.go || true

# generate the go code for each feature stage
protoc -I ${googleapis} -I ${sdk} sdk.proto --go_out=plugins=grpc:pkg/sdk
protoc -I ${googleapis} -I ${sdk}/alpha alpha.proto --go_out=plugins=grpc:pkg/sdk/alpha
protoc -I ${googleapis} -I ${sdk}/beta beta.proto --go_out=plugins=grpc:pkg/sdk/beta

# generate grpc gateway
protoc -I ${googleapis} -I ${sdk} sdk.proto --grpc-gateway_out=logtostderr=true:pkg/sdk
protoc -I ${googleapis} -I ${sdk}/alpha alpha.proto --grpc-gateway_out=logtostderr=true:pkg/sdk/alpha
protoc -I ${googleapis} -I ${sdk}/beta beta.proto --grpc-gateway_out=logtostderr=true:pkg/sdk/beta

protoc -I ${googleapis} -I ${sdk} sdk.proto --swagger_out=logtostderr=true:sdks/swagger
protoc -I ${googleapis} -I ${sdk}/alpha alpha.proto --swagger_out=logtostderr=true:sdks/swagger
protoc -I ${googleapis} -I ${sdk}/beta beta.proto --swagger_out=logtostderr=true:sdks/swagger

jq 'del(.schemes[] | select(. == "https"))' ./sdks/swagger/sdk.swagger.json | sponge ./sdks/swagger/sdk.swagger.json
jq 'del(.schemes[] | select(. == "https"))' ./sdks/swagger/alpha.swagger.json | sponge ./sdks/swagger/alpha.swagger.json
jq 'del(.schemes[] | select(. == "https"))' ./sdks/swagger/beta.swagger.json | sponge ./sdks/swagger/beta.swagger.json

header ./pkg/sdk/sdk.pb.go
header ./pkg/sdk/sdk.pb.gw.go
header ./pkg/sdk/alpha/alpha.pb.go
header ./pkg/sdk/beta/beta.pb.go

# these files may not exist if there are no grpc services
if [ -f "./pkg/sdk/alpha/alpha.pb.gw.go" ]; then
    header ./pkg/sdk/alpha/alpha.pb.gw.go
fi
if [ -f "./pkg/sdk/beta/beta.pb.gw.go" ]; then
    header ./pkg/sdk/beta/beta.pb.gw.go
fi

goimports -w ./pkg/sdk/*
goimports -w ./pkg/sdk/alpha/*
goimports -w ./pkg/sdk/beta/*