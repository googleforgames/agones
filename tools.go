// +build tools

package agones

/*
	github.com/ahmetb/gen-crd-api-reference-docs                   site/gen-api-docs.sh
	github.com/golang/protobuf/protoc-gen-go                       build-sdk-images/go/Dockerfile
	github.com/grpc-ecosystem/grpc-gateway/protoc-gen-grpc-gateway build-sdk-images/go/gen.sh
	github.com/grpc-ecosystem/grpc-gateway/protoc-gen-swagger      build-sdk-images/go/gen.sh
	golang.org/x/tools/cmd/goimports                               build-image/Dockerfile
*/

import (
	_ "github.com/ahmetb/gen-crd-api-reference-docs"
	_ "github.com/golang/protobuf/protoc-gen-go"
	_ "github.com/grpc-ecosystem/grpc-gateway/protoc-gen-grpc-gateway"
	_ "github.com/grpc-ecosystem/grpc-gateway/protoc-gen-swagger"
	_ "golang.org/x/tools/cmd/goimports"
)
