module agones.dev/agones

go 1.14

require (
	cloud.google.com/go v0.38.0
	contrib.go.opencensus.io/exporter/prometheus v0.1.0
	contrib.go.opencensus.io/exporter/stackdriver v0.8.0
	fortio.org/fortio v1.3.1
	github.com/ahmetb/gen-crd-api-reference-docs v0.1.1
	github.com/aws/aws-sdk-go v1.16.20 // indirect
	github.com/fsnotify/fsnotify v1.4.7
	github.com/go-openapi/spec v0.19.3
	github.com/golang/protobuf v1.3.2
	github.com/gorilla/websocket v1.4.0
	github.com/grpc-ecosystem/grpc-gateway v1.11.3
	github.com/hashicorp/golang-lru v0.5.1
	github.com/heptiolabs/healthcheck v0.0.0-20171201210846-da5fdee475fb
	github.com/joonix/log v0.0.0-20200409080653-9c1d2ceb5f1d
	github.com/mattbaird/jsonpatch v0.0.0-20171005235357-81af80346b1a
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822
	github.com/pkg/errors v0.8.1
	github.com/prometheus/client_golang v1.0.0
	github.com/sirupsen/logrus v1.8.1
	github.com/spf13/pflag v1.0.5
	github.com/spf13/viper v1.3.2
	github.com/stretchr/testify v1.5.0
	github.com/tmc/grpc-websocket-proxy v0.0.0-20170815181823-89b8d40f7ca8
	go.opencensus.io v0.22.3
	golang.org/x/net v0.0.0-20191004110552-13f9640d40b9
	golang.org/x/time v0.0.0-20190308202827-9d24e82272b4
	golang.org/x/tools v0.0.0-20190920225731-5eefd052ad72
	google.golang.org/genproto v0.0.0-20191009194640-548a555dbc03
	google.golang.org/grpc v1.26.0
	gopkg.in/DATA-DOG/go-sqlmock.v1 v1.3.0 // indirect
	gopkg.in/fsnotify.v1 v1.4.7
	gopkg.in/natefinch/lumberjack.v2 v2.0.0
	k8s.io/api v0.18.15
	k8s.io/apiextensions-apiserver v0.18.15
	k8s.io/apimachinery v0.18.15
	k8s.io/client-go v0.18.15
	k8s.io/kube-openapi v0.0.0-20200410163147-594e756bea31 // indirect
	k8s.io/utils v0.0.0-20200324210504-a9aa75ae1b89
)

replace google.golang.org/grpc v1.26.0 => google.golang.org/grpc v1.20.1 // apiserver updated grpc, but we aren't using that, so it's fine.
