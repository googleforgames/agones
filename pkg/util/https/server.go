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

package https

import (
	"context"
	cryptotls "crypto/tls"
	"net/http"
	"sync"
	"time"

	"agones.dev/agones/pkg/util/fswatch"
	"agones.dev/agones/pkg/util/runtime"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const (
	tlsDir = "/certs/"
)

var closeCh = make(chan struct{})

// tls is a http server interface to enable easier testing
type tls interface {
	Close() error
	ListenAndServeTLS(certFile, keyFile string) error
}

// Server is a HTTPs server that conforms to the runner interface
// we use in /cmd/controller, and has a public Mux that can be updated
// has a default 404 handler, to make discovery of k8s services a bit easier.
type Server struct {
	logger   *logrus.Entry
	Mux      *http.ServeMux
	Certs    *cryptotls.Certificate
	CertMu   sync.Mutex
	tls      tls
	certFile string
	keyFile  string
}

// NewServer returns a Server instance.
func NewServer(certFile, keyFile string) *Server {
	mux := http.NewServeMux()

	wh := &Server{
		Mux:      mux,
		certFile: certFile,
		keyFile:  keyFile,
	}

	tlsCert, err := readTLSCert()
	if err != nil {
		logrus.WithError(err).Fatal("could not load TLS certs.")
	}
	wh.CertMu.Lock()
	wh.Certs = tlsCert
	wh.CertMu.Unlock()

	wh.tls = &http.Server{
		Addr:    ":8081",
		Handler: wh.Mux,
	}

	// Start a goroutine to watch for certificate changes
	go watchForCertificateChanges(wh)

	wh.Mux.HandleFunc("/", wh.defaultHandler)
	wh.logger = runtime.NewLoggerWithType(wh)

	return wh
}

// It will load the key pair certificate
func readTLSCert() (*cryptotls.Certificate, error) {
	tlsCert, err := cryptotls.LoadX509KeyPair(tlsDir+"server.crt", tlsDir+"server.key")
	if err != nil {
		return nil, err
	}
	return &tlsCert, nil
}

// watchForCertificateChanges watches for changes in the certificate files
func watchForCertificateChanges(s *Server) {
	// Watch for changes in the tlsDir
	cancelTLS, err := fswatch.Watch(s.logger, tlsDir, time.Second, func() {
		// Load the new TLS certificate
		tlsCert, err := cryptotls.LoadX509KeyPair(tlsDir+"server.crt", tlsDir+"server.key")
		if err != nil {
			s.logger.WithError(err).Error("could not load TLS certs; keeping old one")
			return
		}
		s.CertMu.Lock()
		defer s.CertMu.Unlock()
		// Update the Certs structure with the new certificate
		s.Certs = &tlsCert
		s.logger.Info("TLS certs updated")
	})
	if err != nil {
		s.logger.WithError(err).Fatal("could not create watcher for TLS certs")
	}

	// Wait for the signal to close
	<-closeCh
	cancelTLS()
}

// Run runs the webhook server, starting a https listener.
// Will close the http server on stop channel close.
func (s *Server) Run(ctx context.Context, _ int) error {
	go func() {
		<-ctx.Done()
		s.tls.Close() // nolint: errcheck,gosec
		close(closeCh)
	}()

	s.logger.WithField("server", s).Infof("https server started")

	err := s.tls.ListenAndServeTLS(s.certFile, s.keyFile)
	if err == http.ErrServerClosed {
		s.logger.WithError(err).Info("https server closed")
		return nil
	}

	return errors.Wrap(err, "Could not listen on :8081")
}

// defaultHandler Handles all the HTTP requests
// useful for debugging requests
func (s *Server) defaultHandler(w http.ResponseWriter, r *http.Request) {
	// "/" is the default health check used by APIServers
	if r.URL.Path == "/" {
		w.WriteHeader(http.StatusOK)
		return
	}

	FourZeroFour(s.logger, w, r)
}
