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
	"crypto/tls"
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

var tlsMutex sync.Mutex

// tls is a http server interface to enable easier testing
type testTLS interface {
	Close() error
	ListenAndServeTLS(certFile, keyFile string) error
}

// Server is a HTTPs server that conforms to the runner interface
// we use in /cmd/controller, and has a public Mux that can be updated
// has a default 404 handler, to make discovery of k8s services a bit easier.
type Server struct {
	logger   *logrus.Entry
	Mux      *http.ServeMux
	tls      testTLS
	certFile string
	keyFile  string
}

// NewServer returns a Server instance.
func NewServer(certFile, keyFile string, logger *logrus.Entry) *Server {
	mux := http.NewServeMux()
	tls_server := &http.Server{
		Addr:    ":8081",
		Handler: mux,
	}

	go func() {
		cancelTLS, err := fswatch.Watch(logger, tlsDir, time.Second, func() {
			tlsCert, err := readTLSCert()
			if err != nil {
				logger.WithError(err).Error("could not load TLS certs; keeping old one")
				return
			}
			tlsMutex.Lock()
			defer tlsMutex.Unlock()
			tls_server.TLSConfig = &tls.Config{
				GetCertificate: func(*tls.ClientHelloInfo) (*tls.Certificate, error) {
					return tlsCert, nil
				},
			}
			logger.Info("TLS certs updated")
		})
		if err != nil {
			logger.WithError(err).Fatal("could not create watcher for TLS certs")
		}
		defer cancelTLS()

	}()

	wh := &Server{
		Mux:      mux,
		tls:      tls_server,
		certFile: certFile,
		keyFile:  keyFile,
	}
	wh.Mux.HandleFunc("/", wh.defaultHandler)
	wh.logger = runtime.NewLoggerWithType(wh)

	return wh
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

func readTLSCert() (*tls.Certificate, error) {
	tlsCert, err := tls.LoadX509KeyPair(tlsDir+"server.crt", tlsDir+"server.key")
	if err != nil {
		return nil, err
	}
	return &tlsCert, nil
}
