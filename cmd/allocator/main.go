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
	"github.com/heptiolabs/healthcheck"
	"github.com/pkg/errors"
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

var (
	logger = runtime.NewLoggerWithSource("main")
)

const (
	certDir = "/home/allocator/client-ca/"
	tlsDir  = "/home/allocator/tls/"
	sslPort = "8443"
)

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

	h := newServiceHandler(kubeClient, agonesClient, health, conf.MTLSDisabled, conf.TLSDisabled)

	listener, err := net.Listen("tcp", fmt.Sprintf(":%s", sslPort))
	if err != nil {
		logger.WithError(err).Fatalf("failed to listen on TCP port %s", sslPort)
	}

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

	opts := h.getServerOptions()

	grpcServer := grpc.NewServer(opts...)
	pb.RegisterAllocationServiceServer(grpcServer, h)

	// serve GRPC for allocation
	go func() {
		err := grpcServer.Serve(listener)
		logger.WithError(err).Fatal("allocation service crashed")
		os.Exit(1)
	}()

	// Finally listen on 8080 (http) and block the main goroutine
	// this is used to serve /live and /ready handlers for Kubernetes probes.
	err = http.ListenAndServe(":8080", http.DefaultServeMux)
	logger.WithError(err).Fatal("allocation service crashed")
}

func newServiceHandler(kubeClient kubernetes.Interface, agonesClient versioned.Interface, health healthcheck.Handler, mTLSDisabled bool, tlsDisabled bool) *serviceHandler {
	defaultResync := 30 * time.Second
	agonesInformerFactory := externalversions.NewSharedInformerFactory(agonesClient, defaultResync)
	kubeInformerFactory := informers.NewSharedInformerFactory(kubeClient, defaultResync)
	gsCounter := gameservers.NewPerNodeCounter(kubeInformerFactory, agonesInformerFactory)

	allocator := gameserverallocations.NewAllocator(
		agonesInformerFactory.Multicluster().V1().GameServerAllocationPolicies(),
		kubeInformerFactory.Core().V1().Secrets(),
		kubeClient,
		gameserverallocations.NewReadyGameServerCache(agonesInformerFactory.Agones().V1().GameServers(), agonesClient.AgonesV1(), gsCounter, health))

	stop := signals.NewStopChannel()
	h := serviceHandler{
		allocationCallback: func(gsa *allocationv1.GameServerAllocation) (k8sruntime.Object, error) {
			return allocator.Allocate(gsa, stop)
		},
		mTLSDisabled: mTLSDisabled,
		tlsDisabled:  tlsDisabled,
	}

	kubeInformerFactory.Start(stop)
	agonesInformerFactory.Start(stop)
	if err := allocator.Start(stop); err != nil {
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

// getServerOptions returns a list of GRPC server options.
// Current options are TLS certs and opencensus stats handler.
func (h *serviceHandler) getServerOptions() []grpc.ServerOption {
	if h.tlsDisabled {
		return []grpc.ServerOption{grpc.StatsHandler(&ocgrpc.ServerHandler{})}
	}

	cfg := &tls.Config{
		GetCertificate: h.getTLSCert,
	}

	if !h.mTLSDisabled {
		cfg.ClientAuth = tls.RequireAnyClientCert
		cfg.VerifyPeerCertificate = h.verifyClientCertificate
	}

	// Add options for creds and  OpenCensus stats handler to enable stats and tracing.
	// The keepalive options are useful for efficiency purposes (keeping a single connection alive
	// instead of constantly recreating connections), when placing the Agones allocator behind load balancers.
	return []grpc.ServerOption{
		grpc.Creds(credentials.NewTLS(cfg)),
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
		return errors.New("bad client certificate: " + err.Error())
	}

	h.certMutex.RLock()
	defer h.certMutex.RUnlock()
	_, err = c.Verify(opts)
	if err != nil {
		return errors.New("failed to verify client certificate: " + err.Error())
	}
	return nil
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
