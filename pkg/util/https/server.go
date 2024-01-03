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
	tls      tls
	certFile string
	keyFile  string
	CertMu   sync.Mutex
	Certs    *cryptotls.Certificate
}

// NewServer returns a Server instance.
func NewServer(certFile, keyFile string) *Server {
	mux := http.NewServeMux()

	wh := &Server{
		Mux:      mux,
		certFile: certFile,
		keyFile:  keyFile,
	}
	wh.setupServer()

	wh.Mux.HandleFunc("/", wh.defaultHandler)
	wh.logger = runtime.NewLoggerWithType(wh)

	return wh
}

func (s *Server) setupServer() {
	s.tls = &http.Server{
		Addr:    ":8081",
		Handler: s.Mux,
		TLSConfig: &cryptotls.Config{
			GetCertificate: s.getCertificate,
		},
	}

	// Start a goroutine to watch for certificate changes
	go s.watchForCertificateChanges()
}

// getCertificate returns the current TLS certificate
func (s *Server) getCertificate(hello *cryptotls.ClientHelloInfo) (*cryptotls.Certificate, error) {
	s.CertMu.Lock()
	defer s.CertMu.Unlock()
	return s.Certs, nil
}

// watchForCertificateChanges watches for changes in the certificate files
func (s *Server) watchForCertificateChanges() {
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

	defer cancelTLS()
}

// Run runs the webhook server, starting a https listener.
// Will close the http server on stop channel close.
func (s *Server) Run(ctx context.Context, _ int) error {
	go func() {
		<-ctx.Done()
		s.tls.Close() // nolint: errcheck,gosec
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
