module agones.dev/agones

go 1.12

require (
	cloud.google.com/go v0.34.0 // indirect
	contrib.go.opencensus.io/exporter/stackdriver v0.8.0
	fortio.org/fortio v1.3.1
	github.com/BurntSushi/toml v0.3.1 // indirect
	github.com/aws/aws-sdk-go v1.16.20 // indirect
	github.com/evanphx/json-patch v4.1.0+incompatible // indirect
	github.com/fsnotify/fsnotify v1.4.7
	github.com/go-openapi/spec v0.19.0
	github.com/golang/groupcache v0.0.0-20171101203131-84a468cf14b4 // indirect
	github.com/golang/protobuf v1.2.0
	github.com/google/btree v0.0.0-20180813153112-4030bb1f1f0c // indirect
	github.com/google/uuid v1.1.0 // indirect
	github.com/googleapis/gnostic v0.1.0 // indirect
	github.com/gregjones/httpcache v0.0.0-20181110185634-c63ab54fda8f // indirect
	github.com/grpc-ecosystem/grpc-gateway v1.5.1
	github.com/heptiolabs/healthcheck v0.0.0-20171201210846-da5fdee475fb
	github.com/imdario/mergo v0.3.5 // indirect
	github.com/joonix/log v0.0.0-20180502111528-d2d3f2f4a806
	github.com/json-iterator/go v1.1.5 // indirect
	github.com/mattbaird/jsonpatch v0.0.0-20171005235357-81af80346b1a
	github.com/munnerz/goautoneg v0.0.0-20120707110453-a547fc61f48d
	github.com/onsi/ginkgo v1.8.0 // indirect
	github.com/onsi/gomega v1.5.0 // indirect
	github.com/pborman/uuid v0.0.0-20180906182336-adf5a7427709 // indirect
	github.com/peterbourgon/diskv v2.0.1+incompatible // indirect
	github.com/pkg/errors v0.8.0
	github.com/prometheus/client_golang v0.9.2
	github.com/sirupsen/logrus v1.2.0
	github.com/spf13/pflag v1.0.3
	github.com/spf13/viper v1.3.1
	github.com/stretchr/testify v1.3.0
	go.opencensus.io v0.18.0
	golang.org/x/net v0.0.0-20190206173232-65e2d4e15006
	golang.org/x/sync v0.0.0-20181221193216-37e7f081c4d4 // indirect
	golang.org/x/time v0.0.0-20180412165947-fbb02b2291d2
	google.golang.org/api v0.0.0-20190117000611-43037ff31f69 // indirect
	google.golang.org/genproto v0.0.0-20190111180523-db91494dd46c
	google.golang.org/grpc v1.17.0
	gopkg.in/natefinch/lumberjack.v2 v2.0.0-20170531160350-a96e63847dc3
	gopkg.in/yaml.v2 v2.2.2
	k8s.io/api v0.0.0-20181126191744-95336914c664
	k8s.io/apiextensions-apiserver v0.0.0-20181126195113-57b8dbfcc51a
	k8s.io/apimachinery v0.0.0-20181126123124-70adfbae261e
	k8s.io/client-go v8.0.0+incompatible
)

replace k8s.io/apimachinery => ./vendor_fixes/k8s.io/apimachinery
