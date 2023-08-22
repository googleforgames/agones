# Log

[![GoDoc](https://img.shields.io/badge/godoc-reference-blue.svg?style=flat)](https://godoc.org/github.com/joonix/log)

Formatter for logrus, allowing log entries to be recognized by the fluentd
Stackdriver agent on Google Cloud Platform.

Example:

```go
package main

import (
	"time"
	"net/http"

	log "github.com/sirupsen/logrus"
	joonix "github.com/joonix/log"
)

func main() {
	log.SetFormatter(joonix.NewFormatter())
	log.Info("hello world!")

	// log a HTTP request in your handler
	log.WithField("httpRequest", &joonix.HTTPRequest{
		Request: r,
		Status: http.StatusOK,
		ResponseSize: 31337,
		Latency: 123*time.Millisecond,
	}).Info("additional info")
}
```

## Alternatives

- https://github.com/TV4/logrus-stackdriver-formatter (seems abandoned)
- https://github.com/knq/sdhook (implemented as a hook, doesn't require fluentd)
- https://github.com/joonix/log/issues/2 (you can map the format yourself)

## Kubernetes logging from outside of GCP

It is possible to run the google edition of fluentd a.k.a. stackdriver agent outside of GCP,
just a bit tricky to configure. See following references for more info:

- Fluentd build: https://github.com/GoogleCloudPlatform/k8s-stackdriver/tree/master/fluentd-gcp-image
- Manifest examples: https://github.com/kubernetes/kubernetes/tree/master/cluster/addons/fluentd-gcp
- Manifest vars (fluentd_gcp_yaml_version): https://github.com/kubernetes/kubernetes/blob/master/cluster/gce/config-default.sh
- Tutorial (old manifest): https://kubernetes.io/docs/tasks/debug-application-cluster/logging-stackdriver/
