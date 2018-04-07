#!/usr/bin/env bash

cd /go/src/agones.dev/agones
protoc --rust_out sdks/rust/src/grpc --grpc_out=sdks/rust/src/grpc --plugin=protoc-gen-grpc=`which grpc_rust_plugin` sdk.proto
