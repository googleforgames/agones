// Copyright 2020 Google LLC All Rights Reserved.
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
package main

import (
	"net/http"
	"strings"

	"agones.dev/agones/pkg/metrics"
	"agones.dev/agones/pkg/util/runtime"
	"github.com/heptiolabs/healthcheck"
	prom "github.com/prometheus/client_golang/prometheus"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"go.opencensus.io/plugin/ocgrpc"
	"go.opencensus.io/stats/view"
)

const (
	enableStackdriverMetricsFlag = "stackdriver-exporter"
	enablePrometheusMetricsFlag  = "prometheus-exporter"
	projectIDFlag                = "gcp-project-id"
	stackdriverLabels            = "stackdriver-labels"
	mTLSDisabledFlag             = "disable-mtls"
	tlsDisabledFlag              = "disable-tls"
)

func init() {
	registerMetricViews()
}

type config struct {
	TLSDisabled       bool
	MTLSDisabled      bool
	PrometheusMetrics bool
	Stackdriver       bool
	GCPProjectID      string
	StackdriverLabels string
}

func parseEnvFlags() config {

	viper.SetDefault(enablePrometheusMetricsFlag, true)
	viper.SetDefault(enableStackdriverMetricsFlag, false)
	viper.SetDefault(projectIDFlag, "")
	viper.SetDefault(stackdriverLabels, "")
	viper.SetDefault(mTLSDisabledFlag, false)
	viper.SetDefault(tlsDisabledFlag, false)

	pflag.Bool(enablePrometheusMetricsFlag, viper.GetBool(enablePrometheusMetricsFlag), "Flag to activate metrics of Agones. Can also use PROMETHEUS_EXPORTER env variable.")
	pflag.Bool(enableStackdriverMetricsFlag, viper.GetBool(enableStackdriverMetricsFlag), "Flag to activate stackdriver monitoring metrics for Agones. Can also use STACKDRIVER_EXPORTER env variable.")
	pflag.String(projectIDFlag, viper.GetString(projectIDFlag), "GCP ProjectID used for Stackdriver, if not specified ProjectID from Application Default Credentials would be used. Can also use GCP_PROJECT_ID env variable.")
	pflag.String(stackdriverLabels, viper.GetString(stackdriverLabels), "A set of default labels to add to all stackdriver metrics generated. By default metadata are automatically added using Kubernetes API and GCP metadata enpoint.")
	pflag.Bool(mTLSDisabledFlag, viper.GetBool(mTLSDisabledFlag), "Flag to enable/disable mTLS in the allocator.")
	pflag.Bool(tlsDisabledFlag, viper.GetBool(tlsDisabledFlag), "Flag to enable/disable TLS in the allocator.")
	runtime.FeaturesBindFlags()
	pflag.Parse()

	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	runtime.Must(viper.BindEnv(enablePrometheusMetricsFlag))
	runtime.Must(viper.BindEnv(enableStackdriverMetricsFlag))
	runtime.Must(viper.BindEnv(projectIDFlag))
	runtime.Must(viper.BindEnv(stackdriverLabels))
	runtime.Must(viper.BindEnv(mTLSDisabledFlag))
	runtime.Must(viper.BindEnv(tlsDisabledFlag))
	runtime.Must(viper.BindPFlags(pflag.CommandLine))
	runtime.Must(runtime.FeaturesBindEnv())

	runtime.Must(runtime.ParseFeaturesFromEnv())

	return config{
		PrometheusMetrics: viper.GetBool(enablePrometheusMetricsFlag),
		Stackdriver:       viper.GetBool(enableStackdriverMetricsFlag),
		GCPProjectID:      viper.GetString(projectIDFlag),
		StackdriverLabels: viper.GetString(stackdriverLabels),
		MTLSDisabled:      viper.GetBool(mTLSDisabledFlag),
		TLSDisabled:       viper.GetBool(tlsDisabledFlag),
	}
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
