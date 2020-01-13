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
	"sync"
	"time"

	"agones.dev/agones/pkg"
	"agones.dev/agones/pkg/allocation/converters"
	pb "agones.dev/agones/pkg/allocation/go/v1alpha1"
	allocationv1 "agones.dev/agones/pkg/apis/allocation/v1"
	"agones.dev/agones/pkg/client/clientset/versioned"
	"agones.dev/agones/pkg/client/informers/externalversions"
	"agones.dev/agones/pkg/gameserverallocations"
	"agones.dev/agones/pkg/gameservers"
	"agones.dev/agones/pkg/metrics"
	"agones.dev/agones/pkg/util/runtime"
	"agones.dev/agones/pkg/util/signals"
	"github.com/heptiolabs/healthcheck"
	prom "github.com/prometheus/client_golang/prometheus"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"go.opencensus.io/plugin/ochttp"
	"go.opencensus.io/stats/view"
	"gopkg.in/fsnotify.v1"
	k8serror "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
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
	stackdriverLabels            = "stackdriver-labels"
)

func init() {
	registerMetricViews()
}

// A handler for the web server
type handler func(w http.ResponseWriter, r *http.Request)

func main() {
	conf := parseEnvFlags()

	logger.WithField("version", pkg.Version).WithField("ctlConf", conf).
		WithField("featureGates", runtime.EncodeFeatures()).WithField("sslPort", sslPort).
		Info("Starting agones-allocator")

	health, closer := setupMetricsRecorder(conf)
	defer closer()

	// http.DefaultServerMux is used for http connection, not for https
	http.Handle("/", health)

	kubeClient, agonesClient, err := getClients()
	if err != nil {
		logger.WithError(err).Fatal("could not create clients")
	}

	// This will test the connection to agones on each readiness probe
	// so if one of the allocator pod can't reach Kubernetes it will be removed
	// from the Kubernetes service.
	health.AddReadinessCheck("allocator-agones-client", func() error {
		_, err := agonesClient.ServerVersion()
		return err
	})

	h := newServiceHandler(kubeClient, agonesClient, health)

	// mux for https server to serve gameserver allocations
	httpsMux := http.NewServeMux()
	httpsMux.HandleFunc("/v1alpha1/gameserverallocation", h.postOnly(h.allocateHandler))

	// creates a new file watcher for client certificate folder
	watcher, _ := fsnotify.NewWatcher()
	defer watcher.Close() // nolint: errcheck
	if err := watcher.Add(certDir); err != nil {
		logger.WithError(err).Fatalf("cannot watch folder %s for secret changes", certDir)
	}

	tlsCer, err := tls.LoadX509KeyPair(tlsDir+"tls.crt", tlsDir+"tls.key")
	if err != nil {
		logger.WithError(err).Fatal("server TLS could not be loaded")
	}
	caCertPool := loadCACertPool()

	// Watching for the events in certificate directory for updating certificates, when there is a change
	go func() {
		for {
			select {
			// watch for events
			case event := <-watcher.Events:
				h.certMutex.Lock()
				caCertPool = loadCACertPool()
				logger.Infof("Certificate directory change event %v", event)
				h.certMutex.Unlock()

				// watch for errors
			case err := <-watcher.Errors:
				logger.WithError(err).Error("error watching for certificate directory")
			}
		}
	}()

	cfg := &tls.Config{
		Certificates: []tls.Certificate{tlsCer},
		ClientAuth:   tls.RequireAndVerifyClientCert,
		GetConfigForClient: func(*tls.ClientHelloInfo) (*tls.Config, error) {
			h.certMutex.RLock()
			defer h.certMutex.RUnlock()
			return &tls.Config{
				Certificates: []tls.Certificate{tlsCer},
				ClientAuth:   tls.RequireAndVerifyClientCert,
				ClientCAs:    caCertPool,
			}, nil
		},
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
		// The certs are set on the config so passing empty as the cert path
		err := srv.ListenAndServeTLS("", "")
		logger.WithError(err).Fatal("allocation service crashed")
		os.Exit(1)
	}()

	// Finally listen on 8080 (http) and block the main goroutine
	// this is used to serve /live and /ready handlers for Kubernetes probes.
	err = http.ListenAndServe(":8080", http.DefaultServeMux)
	logger.WithError(err).Fatal("allocation service crashed")
}

func loadCACertPool() *x509.CertPool {
	caCertPool, err := getCACertPool(certDir)
	if err != nil {
		logger.WithError(err).Fatal("could not get CA certs")
	}
	return caCertPool
}

func newServiceHandler(kubeClient kubernetes.Interface, agonesClient versioned.Interface, health healthcheck.Handler) *httpHandler {
	defaultResync := 30 * time.Second
	agonesInformerFactory := externalversions.NewSharedInformerFactory(agonesClient, defaultResync)
	kubeInformerFactory := informers.NewSharedInformerFactory(kubeClient, defaultResync)
	gsCounter := gameservers.NewPerNodeCounter(kubeInformerFactory, agonesInformerFactory)

	allocator := gameserverallocations.NewAllocator(
		agonesInformerFactory.Multicluster().V1alpha1().GameServerAllocationPolicies(),
		kubeInformerFactory.Core().V1().Secrets(),
		kubeClient,
		gameserverallocations.NewReadyGameServerCache(agonesInformerFactory.Agones().V1().GameServers(), agonesClient.AgonesV1(), gsCounter, health))

	stop := signals.NewStopChannel()
	h := httpHandler{
		allocationCallback: func(gsa *allocationv1.GameServerAllocation) (k8sruntime.Object, error) {
			return allocator.Allocate(gsa, stop)
		},
	}

	kubeInformerFactory.Start(stop)
	agonesInformerFactory.Start(stop)
	if err := allocator.Start(stop); err != nil {
		logger.WithError(err).Fatal("starting allocator failed.")
	}

	return &h
}

// Set up our client which we will use to call the API
func getClients() (*kubernetes.Clientset, *versioned.Clientset, error) {
	// Create the in-cluster config
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, nil, errors.New("Could not create in cluster config")
	}

	// Access to the Agones resources through the Agones Clientset
	kubeClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, nil, errors.New("Could not create the kubernetes api clientset")
	}

	// Access to the Agones resources through the Agones Clientset
	agonesClient, err := versioned.NewForConfig(config)
	if err != nil {
		return nil, nil, errors.New("Could not create the agones api clientset")
	}
	return kubeClient, agonesClient, nil
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
		if !strings.HasSuffix(file.Name(), ".crt") && !strings.HasSuffix(file.Name(), ".pem") {
			continue
		}
		certFile := filepath.Join(path, file.Name())
		caCert, err := ioutil.ReadFile(certFile)
		if err != nil {
			logger.Errorf("ca cert is not readable or missing: %s", err.Error())
			continue
		}
		if !caCertPool.AppendCertsFromPEM(caCert) {
			logger.Errorf("client cert %s cannot be installed", certFile)
			continue
		}
		logger.Infof("client cert %s is installed", certFile)
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
	allocationCallback func(*allocationv1.GameServerAllocation) (k8sruntime.Object, error)
	certMutex          sync.RWMutex
}

func (h *httpHandler) allocateHandler(w http.ResponseWriter, r *http.Request) {
	request := pb.AllocationRequest{}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		logger.WithError(err).Info("bad request")
		return
	}
	logger.WithField("request", request).Infof("allocation request received")

	gsa := converters.ConvertAllocationRequestV1Alpha1ToGSAV1(&request)
	resultObj, err := h.allocationCallback(gsa)
	if err != nil {
		http.Error(w, err.Error(), httpCode(err))
		logger.WithField("gsa", gsa).WithError(err).Info("allocation failed")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if status, ok := resultObj.(*metav1.Status); ok {
		w.WriteHeader(int(status.Code))
		err = json.NewEncoder(w).Encode(status)
		if err != nil {
			http.Error(w, "internal server error", http.StatusInternalServerError)
			logger.WithError(err).Error("Unable to encode status in json")
			return
		}
	}
	allocatedGsa, ok := resultObj.(*allocationv1.GameServerAllocation)
	if !ok {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		logger.Errorf("internal server error - Bad GSA format %v", resultObj)
		return
	}
	response := converters.ConvertGSAV1ToAllocationResponseV1Alpha1(allocatedGsa)
	logger.WithField("response", response).Infof("allocation response is being sent")

	err = json.NewEncoder(w).Encode(response)
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		logger.WithError(err).Error("Unable to encode status in json")
		return
	}
}

func httpCode(err error) int {
	code := http.StatusInternalServerError
	if t, ok := err.(k8serror.APIStatus); ok {
		code = int(t.Status().Code)
	}
	return code
}

type config struct {
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

	pflag.Bool(enablePrometheusMetricsFlag, viper.GetBool(enablePrometheusMetricsFlag), "Flag to activate metrics of Agones. Can also use PROMETHEUS_EXPORTER env variable.")
	pflag.Bool(enableStackdriverMetricsFlag, viper.GetBool(enableStackdriverMetricsFlag), "Flag to activate stackdriver monitoring metrics for Agones. Can also use STACKDRIVER_EXPORTER env variable.")
	pflag.String(projectIDFlag, viper.GetString(projectIDFlag), "GCP ProjectID used for Stackdriver, if not specified ProjectID from Application Default Credentials would be used. Can also use GCP_PROJECT_ID env variable.")
	pflag.String(stackdriverLabels, viper.GetString(stackdriverLabels), "A set of default labels to add to all stackdriver metrics generated. By default metadata are automatically added using Kubernetes API and GCP metadata enpoint.")
	runtime.FeaturesBindFlags()
	pflag.Parse()

	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	runtime.Must(viper.BindEnv(enablePrometheusMetricsFlag))
	runtime.Must(viper.BindEnv(enableStackdriverMetricsFlag))
	runtime.Must(viper.BindEnv(projectIDFlag))
	runtime.Must(viper.BindEnv(stackdriverLabels))
	runtime.Must(viper.BindPFlags(pflag.CommandLine))
	runtime.Must(runtime.FeaturesBindEnv())

	runtime.Must(runtime.ParseFeaturesFromEnv())

	return config{
		PrometheusMetrics: viper.GetBool(enablePrometheusMetricsFlag),
		Stackdriver:       viper.GetBool(enableStackdriverMetricsFlag),
		GCPProjectID:      viper.GetString(projectIDFlag),
		StackdriverLabels: viper.GetString(stackdriverLabels),
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
