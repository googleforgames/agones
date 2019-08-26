// Copyright 2019 Google LLC All Rights Reserved.
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
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	allocationv1 "agones.dev/agones/pkg/apis/allocation/v1"
	"agones.dev/agones/pkg/client/clientset/versioned"
	"agones.dev/agones/pkg/metrics"
	"agones.dev/agones/pkg/util/runtime"
	"github.com/heptiolabs/healthcheck"
	prom "github.com/prometheus/client_golang/prometheus"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"go.opencensus.io/plugin/ochttp"
	"go.opencensus.io/stats/view"
	k8serror "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/rest"
)

var (
	logger = runtime.NewLoggerWithSource("main")
)

const (
	certDir = "/home/allocator/client-ca/"
	tlsDir  = "/home/allocator/tls/"
	sslPort = "8443"

	enableStackdriverMetricsFlag = "stackdriver-exporter"
	enablePrometheusMetricsFlag  = "prometheus-exporter"
	projectIDFlag                = "gcp-project-id"
)

func init() {
	registerMetricViews()
}

// A handler for the web server
type handler func(w http.ResponseWriter, r *http.Request)

func main() {
	conf := parseEnvFlags()

	health, closer := setupMetricsRecorder(conf)
	defer closer()

	// http.DefaultServerMux is used for http connection, not for https
	http.Handle("/", health)

	agonesClient, err := getAgonesClient()
	if err != nil {
		logger.WithError(err).Fatal("could not create agones client")
	}
	// This will test the connection to agones on each readiness probe
	// so if one of the allocator pod can't reach Kubernetes it will be removed
	// from the Kubernetes service.
	health.AddReadinessCheck("allocator-agones-client", func() error {
		_, err := agonesClient.ServerVersion()
		return err
	})

	h := httpHandler{
		agonesClient: agonesClient,
	}

	// mux for https server to serve gameserver allocations
	httpsMux := http.NewServeMux()
	httpsMux.HandleFunc("/v1alpha1/gameserverallocation", h.postOnly(h.allocateHandler))

	caCertPool, err := getCACertPool(certDir)
	if err != nil {
		logger.WithError(err).Fatal("could not get CA certs")
	}

	cfg := &tls.Config{
		ClientAuth: tls.RequireAndVerifyClientCert,
		ClientCAs:  caCertPool,
	}
	srv := &http.Server{
		Addr:      ":" + sslPort,
		TLSConfig: cfg,
		// add http OC metrics (opencensus.io/http/server/*)
		Handler: &ochttp.Handler{
			Handler: httpsMux,
		},
	}

	// listen on https to serve allocations
	go func() {
		err := srv.ListenAndServeTLS(tlsDir+"tls.crt", tlsDir+"tls.key")
		logger.WithError(err).Fatal("allocation service crashed")
		os.Exit(1)
	}()

	// Finally listen on 8080 (http) and block the main goroutine
	// this is used to serve /live and /ready handlers for Kubernetes probes.
	err = http.ListenAndServe(":8080", http.DefaultServeMux)
	logger.WithError(err).Fatal("allocation service crashed")
}

// Set up our client which we will use to call the API
func getAgonesClient() (*versioned.Clientset, error) {
	// Create the in-cluster config
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, errors.New("Could not create in cluster config")
	}

	// Access to the Agones resources through the Agones Clientset
	agonesClient, err := versioned.NewForConfig(config)
	if err != nil {
		return nil, errors.New("Could not create the agones api clientset")
	}
	return agonesClient, nil
}

func getCACertPool(path string) (*x509.CertPool, error) {
	// Add all certificates under client-certs path because there could be multiple clusters
	// and all client certs should be added.
	caCertPool := x509.NewCertPool()
	filesInfo, err := ioutil.ReadDir(path)
	if err != nil {
		return nil, fmt.Errorf("error reading certs from dir %s: %s", path, err.Error())
	}

	for _, file := range filesInfo {
		if strings.HasSuffix(file.Name(), ".crt") || strings.HasSuffix(file.Name(), ".pem") {
			certFile := filepath.Join(path, file.Name())
			caCert, err := ioutil.ReadFile(certFile)
			if err != nil {
				return nil, fmt.Errorf("ca cert is not readable or missing: %s", err.Error())
			}
			if !caCertPool.AppendCertsFromPEM(caCert) {
				return nil, fmt.Errorf("client cert %s cannot be installed", certFile)
			}
			logger.Infof("client cert %s is installed", certFile)
		}
	}

	return caCertPool, nil
}

// Limit verbs the web server handles
func (h *httpHandler) postOnly(in handler) handler {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			in(w, r)
			return
		}
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

type httpHandler struct {
	agonesClient versioned.Interface
}

func (h *httpHandler) allocateHandler(w http.ResponseWriter, r *http.Request) {
	gsa := allocationv1.GameServerAllocation{}
	if err := json.NewDecoder(r.Body).Decode(&gsa); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	allocation := h.agonesClient.AllocationV1().GameServerAllocations(gsa.ObjectMeta.Namespace)
	allocatedGsa, err := allocation.Create(&gsa)
	if err != nil {
		http.Error(w, err.Error(), httpCode(err))
		logger.Debug(err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(allocatedGsa)
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		logger.Error(err)
		return
	}
}

func httpCode(err error) int {
	code := http.StatusInternalServerError
	switch t := err.(type) {
	case k8serror.APIStatus:
		code = int(t.Status().Code)
	}
	return code
}

type config struct {
	PrometheusMetrics bool
	Stackdriver       bool
	GCPProjectID      string
}

func parseEnvFlags() config {

	viper.SetDefault(enablePrometheusMetricsFlag, true)
	viper.SetDefault(enableStackdriverMetricsFlag, false)
	viper.SetDefault(projectIDFlag, "")

	pflag.Bool(enablePrometheusMetricsFlag, viper.GetBool(enablePrometheusMetricsFlag), "Flag to activate metrics of Agones. Can also use PROMETHEUS_EXPORTER env variable.")
	pflag.Bool(enableStackdriverMetricsFlag, viper.GetBool(enableStackdriverMetricsFlag), "Flag to activate stackdriver monitoring metrics for Agones. Can also use STACKDRIVER_EXPORTER env variable.")
	pflag.String(projectIDFlag, viper.GetString(projectIDFlag), "GCP ProjectID used for Stackdriver, if not specified ProjectID from Application Default Credentials would be used. Can also use GCP_PROJECT_ID env variable.")
	pflag.Parse()

	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	runtime.Must(viper.BindEnv(enablePrometheusMetricsFlag))
	runtime.Must(viper.BindEnv(enableStackdriverMetricsFlag))
	runtime.Must(viper.BindEnv(projectIDFlag))
	runtime.Must(viper.BindPFlags(pflag.CommandLine))

	return config{
		PrometheusMetrics: viper.GetBool(enablePrometheusMetricsFlag),
		Stackdriver:       viper.GetBool(enableStackdriverMetricsFlag),
		GCPProjectID:      viper.GetString(projectIDFlag),
	}
}

func registerMetricViews() {
	if err := view.Register(ochttp.DefaultServerViews...); err != nil {
		logger.WithError(err).Error("could not register view")
	}
}

func setupMetricsRecorder(conf config) (health healthcheck.Handler, closer func()) {
	health = healthcheck.NewHandler()
	closer = func() {}

	// Stackdriver metrics
	if conf.Stackdriver {
		sd, err := metrics.RegisterStackdriverExporter(conf.GCPProjectID)
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
