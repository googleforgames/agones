// Copyright 2018 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package metrics

import (
	"net/http"
	"time"

	"contrib.go.opencensus.io/exporter/stackdriver"
	prom "github.com/prometheus/client_golang/prometheus"
	"go.opencensus.io/exporter/prometheus"
	"go.opencensus.io/stats/view"
)

// RegisterPrometheusExporter register a prometheus exporter to OpenCensus with a given prometheus metric registry.
// It will automatically add go runtime and process metrics using default prometheus collectors.
// The function return an http.handler that you can use to expose the prometheus endpoint.
func RegisterPrometheusExporter(registry *prom.Registry) (http.Handler, error) {
	pe, err := prometheus.NewExporter(prometheus.Options{
		Namespace: "agones",
		Registry:  registry,
	})
	if err != nil {
		return nil, err
	}
	if err := registry.Register(prom.NewProcessCollector(prom.ProcessCollectorOpts{})); err != nil {
		return nil, err
	}
	if err := registry.Register(prom.NewGoCollector()); err != nil {
		return nil, err
	}
	view.RegisterExporter(pe)

	return pe, nil
}

// RegisterStackdriverExporter register a Stackdriver exporter to OpenCensus.
// It will add Agones metrics into Stackdriver on Google Cloud.
func RegisterStackdriverExporter(projectID string) (sd *stackdriver.Exporter, err error) {
	// Default project will be used
	sd, err = stackdriver.NewExporter(stackdriver.Options{
		ProjectID: projectID,
		// MetricPrefix helps uniquely identify your metrics.
		MetricPrefix: "agones",
	})
	if err != nil {
		return
	}

	// Register it as a metrics exporter
	view.RegisterExporter(sd)
	return
}

// SetReportingPeriod set appropriate reporting period which depends on exporters
// we are going to use
func SetReportingPeriod(prometheus, stackdriver bool) {
	// if we're using only prometheus we can report faster as we're only exposing metrics in memory
	reportingPeriod := 1 * time.Second
	if stackdriver {
		// There is a limitation on Stackdriver that reporting should
		// be equal or more than 1 minute
		reportingPeriod = 60 * time.Second
	}

	if stackdriver || prometheus {
		view.SetReportingPeriod(reportingPeriod)
	}
}
