// Copyright 2020 Google LLC All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package main

import (
	"net/http"

	"agones.dev/agones/pkg/metrics"
	"github.com/heptiolabs/healthcheck"
	prom "github.com/prometheus/client_golang/prometheus"
	"go.opencensus.io/plugin/ocgrpc"
	"go.opencensus.io/stats/view"
)

func init() {
	registerMetricViews()
}

func registerMetricViews() {
	if err := view.Register(ocgrpc.DefaultServerViews...); err != nil {
		logger.WithError(err).Error("could not register view")
	}
}

func setupMetricsRecorder(conf config) (health healthcheck.Handler, closer func()) {
	health = healthcheck.NewHandler()
	closer = func() {}

	// Stackdriver metrics
	if conf.Stackdriver {
		sd, err := metrics.RegisterStackdriverExporter(conf.GCPProjectID, conf.StackdriverLabels)
		if err != nil {
			logger.WithError(err).Fatal("Could not register stackdriver exporter")
		}
		// It is imperative to invoke flush before your main function exits
		closer = func() { sd.Flush() }
	}

	// Prometheus metrics
	if conf.PrometheusMetrics {
		registry := prom.NewRegistry()
		metricHandler, err := metrics.RegisterPrometheusExporter(registry)
		if err != nil {
			logger.WithError(err).Fatal("Could not register prometheus exporter")
		}
		http.Handle("/metrics", metricHandler)
		health = healthcheck.NewMetricsHandler(registry, "agones")
	}

	metrics.SetReportingPeriod(conf.PrometheusMetrics, conf.Stackdriver)
	return
}
