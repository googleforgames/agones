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
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"agones.dev/agones/pkg"
	"agones.dev/agones/pkg/allocation/converters"
	pb "agones.dev/agones/pkg/allocation/go"
	allocationv1 "agones.dev/agones/pkg/apis/allocation/v1"
	"agones.dev/agones/pkg/client/clientset/versioned"
	"agones.dev/agones/pkg/client/informers/externalversions"
	"agones.dev/agones/pkg/gameserverallocations"
	"agones.dev/agones/pkg/gameservers"
	"agones.dev/agones/pkg/util/runtime"
	"agones.dev/agones/pkg/util/signals"
	gw_runtime "github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/heptiolabs/healthcheck"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/plugin/ocgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/status"
	"gopkg.in/fsnotify.v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

var logger = runtime.NewLoggerWithSource("main")

const (
	certDir = "/home/allocator/client-ca/"
	tlsDir  = "/home/allocator/tls/"
)

// grpcHandlerFunc returns an http.Handler that delegates to grpcServer on incoming gRPC
// connections or otherHandler otherwise. Copied from https://github.com/philips/grpc-gateway-example.
func grpcHandlerFunc(grpcServer http.Handler, otherHandler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// This is a partial recreation of gRPC's internal checks https://github.com/grpc/grpc-go/pull/514/files#diff-95e9a25b738459a2d3030e1e6fa2a718R61
		// We switch on HTTP/1.1 or HTTP/2 by checking the ProtoMajor
		if r.ProtoMajor == 2 && strings.Contains(r.Header.Get("Content-Type"), "application/grpc") {
			grpcServer.ServeHTTP(w, r)
		} else {
			otherHandler.ServeHTTP(w, r)
		}
	})
}
func main() {
	conf := parseEnvFlags()

	logger.WithField("version", pkg.Version).WithField("ctlConf", conf).
		WithField("featureGates", runtime.EncodeFeatures()).
		Info("Starting agones-allocator")

	logger.WithField("logLevel", conf.LogLevel).Info("Setting LogLevel configuration")
	level, err := logrus.ParseLevel(strings.ToLower(conf.LogLevel))
	if err == nil {
		runtime.SetLevel(level)
	} else {
		logger.WithError(err).Info("Specified wrong Logging.SdkServer. Setting default loglevel - Info")
		runtime.SetLevel(logrus.InfoLevel)
	}

	if !validPort(conf.GRPCPort) && !validPort(conf.HTTPPort) {
		logger.WithField("grpc-port", conf.GRPCPort).WithField("http-port", conf.HTTPPort).Fatal("Must specify a valid gRPC port or an HTTP port for the allocator service")
	}

	health, closer := setupMetricsRecorder(conf)
	defer closer()

	// http.DefaultServerMux is used for http connection, not for https
	http.Handle("/", health)

	kubeClient, agonesClient, err := getClients(conf)
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

	h := newServiceHandler(kubeClient, agonesClient, health, conf.MTLSDisabled, conf.TLSDisabled, conf.remoteAllocationTimeout, conf.totalRemoteAllocationTimeout)

	if !h.tlsDisabled {
		watcherTLS, err := fsnotify.NewWatcher()
		if err != nil {
			logger.WithError(err).Fatal("could not create watcher for tls certs")
		}
		defer watcherTLS.Close() // nolint: errcheck
		if err := watcherTLS.Add(tlsDir); err != nil {
			logger.WithError(err).Fatalf("cannot watch folder %s for secret changes", tlsDir)
		}

		// Watching for the events in certificate directory for updating certificates, when there is a change
		go func() {
			for {
				select {
				// watch for events
				case event := <-watcherTLS.Events:
					tlsCert, err := readTLSCert()
					if err != nil {
						logger.WithError(err).Error("could not load TLS cert; keeping old one")
					} else {
						h.tlsMutex.Lock()
						h.tlsCert = tlsCert
						h.tlsMutex.Unlock()
					}
					logger.Infof("Tls directory change event %v", event)

				// watch for errors
				case err := <-watcherTLS.Errors:
					logger.WithError(err).Error("error watching for TLS directory")
				}
			}
		}()

		if !h.mTLSDisabled {
			// creates a new file watcher for client certificate folder
			watcher, err := fsnotify.NewWatcher()
			if err != nil {
				logger.WithError(err).Fatal("could not create watcher for client certs")
			}
			defer watcher.Close() // nolint: errcheck
			if err := watcher.Add(certDir); err != nil {
				logger.WithError(err).Fatalf("cannot watch folder %s for secret changes", certDir)
			}

			go func() {
				for {
					select {
					// watch for events
					case event := <-watcher.Events:
						h.certMutex.Lock()
						caCertPool, err := getCACertPool(certDir)
						if err != nil {
							logger.WithError(err).Error("could not load CA certs; keeping old ones")
						} else {
							h.caCertPool = caCertPool
						}
						logger.Infof("Certificate directory change event %v", event)
						h.certMutex.Unlock()

					// watch for errors
					case err := <-watcher.Errors:
						logger.WithError(err).Error("error watching for certificate directory")
					}
				}
			}()
		}
	}

	// If grpc and http use the same port then use a mux.
	if conf.GRPCPort == conf.HTTPPort {
		runMux(h, conf.HTTPPort)
	} else {
		// Otherwise, run each on a dedicated port.
		if validPort(conf.HTTPPort) {
			runREST(h, conf.HTTPPort)
		}
		if validPort(conf.GRPCPort) {
			runGRPC(h, conf.GRPCPort)
		}
	}

	// Finally listen on 8080 (http) and block the main goroutine
	// this is used to serve /live and /ready handlers for Kubernetes probes.
	err = http.ListenAndServe(":8080", http.DefaultServeMux)
	logger.WithError(err).Fatal("allocation service crashed")
}

func validPort(port int) bool {
	const maxPort = 65535
	return port >= 0 && port < maxPort
}

func runMux(h *serviceHandler, httpPort int) {
	logger.Infof("Running the mux handler on port %d", httpPort)
	grpcServer := grpc.NewServer(h.getMuxServerOptions()...)
	pb.RegisterAllocationServiceServer(grpcServer, h)

	mux := gw_runtime.NewServeMux()
	if err := pb.RegisterAllocationServiceHandlerServer(context.Background(), mux, h); err != nil {
		panic(err)
	}

	runHTTP(h, httpPort, grpcHandlerFunc(grpcServer, mux))
}

func runREST(h *serviceHandler, httpPort int) {
	logger.WithField("port", httpPort).Info("Running the rest handler")
	mux := gw_runtime.NewServeMux()
	if err := pb.RegisterAllocationServiceHandlerServer(context.Background(), mux, h); err != nil {
		panic(err)
	}
	runHTTP(h, httpPort, mux)
}

func runHTTP(h *serviceHandler, httpPort int, handler http.Handler) {
	cfg := &tls.Config{}
	if !h.tlsDisabled {
		cfg.GetCertificate = h.getTLSCert
	}
	if !h.mTLSDisabled {
		cfg.ClientAuth = tls.RequireAnyClientCert
		cfg.VerifyPeerCertificate = h.verifyClientCertificate
	}

	// Create a Server instance to listen on the http port with the TLS config.
	server := &http.Server{
		Addr:      fmt.Sprintf(":%d", httpPort),
		TLSConfig: cfg,
		Handler:   handler,
	}

	go func() {
		var err error
		if !h.tlsDisabled {
			err = server.ListenAndServeTLS("", "")
		} else {
			err = server.ListenAndServe()
		}

		if err != nil {
			logger.WithError(err).Fatal("Unable to start HTTP/HTTPS listener")
			os.Exit(1)
		}
	}()
}

func runGRPC(h *serviceHandler, grpcPort int) {
	logger.WithField("port", grpcPort).Info("Running the grpc handler on port")
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", grpcPort))
	if err != nil {
		logger.WithError(err).Fatalf("failed to listen on TCP port %d", grpcPort)
		os.Exit(1)
	}

	grpcServer := grpc.NewServer(h.getGRPCServerOptions()...)
	pb.RegisterAllocationServiceServer(grpcServer, h)

	go func() {
		err := grpcServer.Serve(listener)
		logger.WithError(err).Fatal("allocation service crashed")
		os.Exit(1)
	}()
}

func newServiceHandler(kubeClient kubernetes.Interface, agonesClient versioned.Interface, health healthcheck.Handler, mTLSDisabled bool, tlsDisabled bool, remoteAllocationTimeout time.Duration, totalRemoteAllocationTimeout time.Duration) *serviceHandler {
	defaultResync := 30 * time.Second
	agonesInformerFactory := externalversions.NewSharedInformerFactory(agonesClient, defaultResync)
	kubeInformerFactory := informers.NewSharedInformerFactory(kubeClient, defaultResync)
	gsCounter := gameservers.NewPerNodeCounter(kubeInformerFactory, agonesInformerFactory)

	allocator := gameserverallocations.NewAllocator(
		agonesInformerFactory.Multicluster().V1().GameServerAllocationPolicies(),
		kubeInformerFactory.Core().V1().Secrets(),
		agonesClient.AgonesV1(),
		kubeClient,
		gameserverallocations.NewAllocationCache(agonesInformerFactory.Agones().V1().GameServers(), gsCounter, health),
		remoteAllocationTimeout,
		totalRemoteAllocationTimeout)

	ctx := signals.NewSigKillContext()
	h := serviceHandler{
		allocationCallback: func(gsa *allocationv1.GameServerAllocation) (k8sruntime.Object, error) {
			return allocator.Allocate(ctx, gsa)
		},
		mTLSDisabled: mTLSDisabled,
		tlsDisabled:  tlsDisabled,
	}

	kubeInformerFactory.Start(ctx.Done())
	agonesInformerFactory.Start(ctx.Done())
	if err := allocator.Run(ctx); err != nil {
		logger.WithError(err).Fatal("starting allocator failed.")
	}

	if !h.tlsDisabled {
		tlsCert, err := readTLSCert()
		if err != nil {
			logger.WithError(err).Fatal("could not load TLS certs.")
		}
		h.tlsMutex.Lock()
		h.tlsCert = tlsCert
		h.tlsMutex.Unlock()

		if !h.mTLSDisabled {
			caCertPool, err := getCACertPool(certDir)
			if err != nil {
				logger.WithError(err).Fatal("could not load CA certs.")
			}
			h.certMutex.Lock()
			h.caCertPool = caCertPool
			h.certMutex.Unlock()
		}
	}

	return &h
}

func readTLSCert() (*tls.Certificate, error) {
	tlsCert, err := tls.LoadX509KeyPair(tlsDir+"tls.crt", tlsDir+"tls.key")
	if err != nil {
		return nil, err
	}
	return &tlsCert, nil
}

// getMuxServerOptions returns a list of GRPC server option to use when
// serving gRPC and REST over an HTTP multiplexer.
// Current options are opencensus stats handler.
func (h *serviceHandler) getMuxServerOptions() []grpc.ServerOption {
	// Add options for  OpenCensus stats handler to enable stats and tracing.
	// The keepalive options are useful for efficiency purposes (keeping a single connection alive
	// instead of constantly recreating connections), when placing the Agones allocator behind load balancers.
	return []grpc.ServerOption{
		grpc.StatsHandler(&ocgrpc.ServerHandler{}),
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
			MinTime:             1 * time.Minute,
			PermitWithoutStream: true,
		}),
		grpc.KeepaliveParams(keepalive.ServerParameters{
			MaxConnectionIdle: 5 * time.Minute,
			Timeout:           10 * time.Minute,
		}),
	}
}

// getGRPCServerOptions returns a list of GRPC server options to use when
// only serving gRPC requests.
// Current options are TLS certs and opencensus stats handler.
func (h *serviceHandler) getGRPCServerOptions() []grpc.ServerOption {
	// Add options for  OpenCensus stats handler to enable stats and tracing.
	// The keepalive options are useful for efficiency purposes (keeping a single connection alive
	// instead of constantly recreating connections), when placing the Agones allocator behind load balancers.
	opts := []grpc.ServerOption{
		grpc.StatsHandler(&ocgrpc.ServerHandler{}),
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
			MinTime:             1 * time.Minute,
			PermitWithoutStream: true,
		}),
		grpc.KeepaliveParams(keepalive.ServerParameters{
			MaxConnectionIdle: 5 * time.Minute,
			Timeout:           10 * time.Minute,
		}),
	}
	if h.tlsDisabled {
		return opts
	}

	cfg := &tls.Config{
		GetCertificate: h.getTLSCert,
	}

	if !h.mTLSDisabled {
		cfg.ClientAuth = tls.RequireAnyClientCert
		cfg.VerifyPeerCertificate = h.verifyClientCertificate
	}

	return append([]grpc.ServerOption{grpc.Creds(credentials.NewTLS(cfg))}, opts...)
}

func (h *serviceHandler) getTLSCert(ch *tls.ClientHelloInfo) (*tls.Certificate, error) {
	h.tlsMutex.RLock()
	defer h.tlsMutex.RUnlock()
	return h.tlsCert, nil
}

// verifyClientCertificate verifies that the client certificate is accepted
// This method is used as GetConfigForClient is cross lang incompatible.
func (h *serviceHandler) verifyClientCertificate(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
	opts := x509.VerifyOptions{
		Roots:         h.caCertPool,
		CurrentTime:   time.Now(),
		Intermediates: x509.NewCertPool(),
		KeyUsages:     []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}

	for _, cert := range rawCerts[1:] {
		opts.Intermediates.AppendCertsFromPEM(cert)
	}

	c, err := x509.ParseCertificate(rawCerts[0])
	if err != nil {
		logger.WithError(err).Warning("cannot parse client certificate")
		return errors.New("bad client certificate: " + err.Error())
	}

	h.certMutex.RLock()
	defer h.certMutex.RUnlock()
	_, err = c.Verify(opts)
	if err != nil {
		logger.WithError(err).Warning("failed to verify client certificate")
		return errors.New("failed to verify client certificate: " + err.Error())
	}
	return nil
}

// Set up our client which we will use to call the API
func getClients(ctlConfig config) (*kubernetes.Clientset, *versioned.Clientset, error) {
	// Create the in-cluster config
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, nil, errors.New("Could not create in cluster config")
	}

	config.QPS = float32(ctlConfig.APIServerSustainedQPS)
	config.Burst = ctlConfig.APIServerBurstQPS

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
			logger.Errorf("CA cert is not readable or missing: %s", err.Error())
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

type serviceHandler struct {
	allocationCallback func(*allocationv1.GameServerAllocation) (k8sruntime.Object, error)

	certMutex  sync.RWMutex
	caCertPool *x509.CertPool

	tlsMutex sync.RWMutex
	tlsCert  *tls.Certificate

	mTLSDisabled bool
	tlsDisabled  bool
}

// Allocate implements the Allocate gRPC method definition
func (h *serviceHandler) Allocate(ctx context.Context, in *pb.AllocationRequest) (*pb.AllocationResponse, error) {
	logger.WithField("request", in).Infof("allocation request received.")
	gsa := converters.ConvertAllocationRequestToGSA(in)
	gsa.ApplyDefaults()
	resultObj, err := h.allocationCallback(gsa)
	if err != nil {
		logger.WithField("gsa", gsa).WithError(err).Info("allocation failed")
		return nil, err
	}

	if s, ok := resultObj.(*metav1.Status); ok {
		return nil, status.Errorf(codes.Code(s.Code), s.Message, resultObj)
	}

	allocatedGsa, ok := resultObj.(*allocationv1.GameServerAllocation)
	if !ok {
		logger.Errorf("internal server error - Bad GSA format %v", resultObj)
		return nil, status.Errorf(codes.Internal, "internal server error- Bad GSA format %v", resultObj)
	}
	response, err := converters.ConvertGSAToAllocationResponse(allocatedGsa)
	logger.WithField("response", response).WithError(err).Infof("allocation response is being sent")

	return response, err
}
