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
	"net/http"

	"agones.dev/agones/pkg/util/runtime"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
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
}

// NewServer returns a Server instance.
func NewServer(certFile, keyFile string) *Server {
	mux := http.NewServeMux()
	tls := &http.Server{
		Addr:    ":8081",
		Handler: mux,
	}

	wh := &Server{
		Mux:      mux,
		tls:      tls,
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
