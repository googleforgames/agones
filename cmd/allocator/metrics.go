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
	"time"

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
	httpPortFlag                     = "http-port"
	grpcPortFlag                     = "grpc-port"
	enableStackdriverMetricsFlag     = "stackdriver-exporter"
	enablePrometheusMetricsFlag      = "prometheus-exporter"
	projectIDFlag                    = "gcp-project-id"
	stackdriverLabels                = "stackdriver-labels"
	mTLSDisabledFlag                 = "disable-mtls"
	tlsDisabledFlag                  = "disable-tls"
	remoteAllocationTimeoutFlag      = "remote-allocation-timeout"
	totalRemoteAllocationTimeoutFlag = "total-remote-allocation-timeout"
	apiServerSustainedQPSFlag        = "api-server-qps"
	apiServerBurstQPSFlag            = "api-server-qps-burst"
	logLevelFlag                     = "log-level"
)

func init() {
	registerMetricViews()
}

type config struct {
	GRPCPort                     int
	HTTPPort                     int
	APIServerSustainedQPS        int
	APIServerBurstQPS            int
	TLSDisabled                  bool
	MTLSDisabled                 bool
	PrometheusMetrics            bool
	Stackdriver                  bool
	GCPProjectID                 string
	StackdriverLabels            string
	LogLevel                     string
	totalRemoteAllocationTimeout time.Duration
	remoteAllocationTimeout      time.Duration
}

func parseEnvFlags() config {

	viper.SetDefault(httpPortFlag, -1)
	viper.SetDefault(grpcPortFlag, -1)
	viper.SetDefault(apiServerSustainedQPSFlag, 400)
	viper.SetDefault(apiServerBurstQPSFlag, 500)
	viper.SetDefault(enablePrometheusMetricsFlag, true)
	viper.SetDefault(enableStackdriverMetricsFlag, false)
	viper.SetDefault(projectIDFlag, "")
	viper.SetDefault(stackdriverLabels, "")
	viper.SetDefault(mTLSDisabledFlag, false)
	viper.SetDefault(tlsDisabledFlag, false)
	viper.SetDefault(remoteAllocationTimeoutFlag, 10*time.Second)
	viper.SetDefault(totalRemoteAllocationTimeoutFlag, 30*time.Second)
	viper.SetDefault(logLevelFlag, "Info")

	pflag.Int32(httpPortFlag, viper.GetInt32(httpPortFlag), "Port to listen on for REST requests")
	pflag.Int32(grpcPortFlag, viper.GetInt32(grpcPortFlag), "Port to listen on for gRPC requests")
	pflag.Int32(apiServerSustainedQPSFlag, viper.GetInt32(apiServerSustainedQPSFlag), "Maximum sustained queries per second to send to the API server")
	pflag.Int32(apiServerBurstQPSFlag, viper.GetInt32(apiServerBurstQPSFlag), "Maximum burst queries per second to send to the API server")
	pflag.Bool(enablePrometheusMetricsFlag, viper.GetBool(enablePrometheusMetricsFlag), "Flag to activate metrics of Agones. Can also use PROMETHEUS_EXPORTER env variable.")
	pflag.Bool(enableStackdriverMetricsFlag, viper.GetBool(enableStackdriverMetricsFlag), "Flag to activate stackdriver monitoring metrics for Agones. Can also use STACKDRIVER_EXPORTER env variable.")
	pflag.String(projectIDFlag, viper.GetString(projectIDFlag), "GCP ProjectID used for Stackdriver, if not specified ProjectID from Application Default Credentials would be used. Can also use GCP_PROJECT_ID env variable.")
	pflag.String(stackdriverLabels, viper.GetString(stackdriverLabels), "A set of default labels to add to all stackdriver metrics generated. By default metadata are automatically added using Kubernetes API and GCP metadata enpoint.")
	pflag.Bool(mTLSDisabledFlag, viper.GetBool(mTLSDisabledFlag), "Flag to enable/disable mTLS in the allocator.")
	pflag.Bool(tlsDisabledFlag, viper.GetBool(tlsDisabledFlag), "Flag to enable/disable TLS in the allocator.")
	pflag.Duration(remoteAllocationTimeoutFlag, viper.GetDuration(remoteAllocationTimeoutFlag), "Flag to set remote allocation call timeout.")
	pflag.Duration(totalRemoteAllocationTimeoutFlag, viper.GetDuration(totalRemoteAllocationTimeoutFlag), "Flag to set total remote allocation timeout including retries.")
	pflag.String(logLevelFlag, viper.GetString(logLevelFlag), "Agones Log level")
	runtime.FeaturesBindFlags()
	pflag.Parse()

	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	runtime.Must(viper.BindEnv(httpPortFlag))
	runtime.Must(viper.BindEnv(grpcPortFlag))
	runtime.Must(viper.BindEnv(apiServerSustainedQPSFlag))
	runtime.Must(viper.BindEnv(apiServerBurstQPSFlag))
	runtime.Must(viper.BindEnv(enablePrometheusMetricsFlag))
	runtime.Must(viper.BindEnv(enableStackdriverMetricsFlag))
	runtime.Must(viper.BindEnv(projectIDFlag))
	runtime.Must(viper.BindEnv(stackdriverLabels))
	runtime.Must(viper.BindEnv(mTLSDisabledFlag))
	runtime.Must(viper.BindEnv(tlsDisabledFlag))
	runtime.Must(viper.BindEnv(logLevelFlag))
	runtime.Must(viper.BindPFlags(pflag.CommandLine))
	runtime.Must(runtime.FeaturesBindEnv())

	runtime.Must(runtime.ParseFeaturesFromEnv())

	return config{
		HTTPPort:                     int(viper.GetInt32(httpPortFlag)),
		GRPCPort:                     int(viper.GetInt32(grpcPortFlag)),
		APIServerSustainedQPS:        int(viper.GetInt32(apiServerSustainedQPSFlag)),
		APIServerBurstQPS:            int(viper.GetInt32(apiServerBurstQPSFlag)),
		PrometheusMetrics:            viper.GetBool(enablePrometheusMetricsFlag),
		Stackdriver:                  viper.GetBool(enableStackdriverMetricsFlag),
		GCPProjectID:                 viper.GetString(projectIDFlag),
		StackdriverLabels:            viper.GetString(stackdriverLabels),
		MTLSDisabled:                 viper.GetBool(mTLSDisabledFlag),
		TLSDisabled:                  viper.GetBool(tlsDisabledFlag),
		LogLevel:                     viper.GetString(logLevelFlag),
		remoteAllocationTimeout:      viper.GetDuration(remoteAllocationTimeoutFlag),
		totalRemoteAllocationTimeout: viper.GetDuration(totalRemoteAllocationTimeoutFlag),
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
