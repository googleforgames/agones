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
	// since we're using prometheus we can report faster as we're only exposing metrics in memory
	view.SetReportingPeriod(1 * time.Second)
	return pe, nil
}
