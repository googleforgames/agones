// Copyright 2024 Google LLC All Rights Reserved.
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

// Package httpserver implements an http server that conforms to the
// controller runner interface.
package httpserver

import (
	"context"
	"net/http"

	"agones.dev/agones/pkg/util/runtime"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// Server is a HTTPs server that conforms to the runner interface
// we use in /cmd/controller.
//
//nolint:govet // ignore field alignment complaint, this is a singleton
type Server struct {
	http.ServeMux

	Logger *logrus.Entry
}

// Run runs an http server on port :8080.
func (s *Server) Run(ctx context.Context, _ int) error {
	s.Logger.Info("Starting http server...")
	srv := &http.Server{
		Addr:    ":8080",
		Handler: s,
	}
	go func() {
		<-ctx.Done()
		_ = srv.Shutdown(context.Background())
	}()

	if err := srv.ListenAndServe(); err != nil {
		if err == http.ErrServerClosed {
			s.Logger.WithError(err).Info("http server closed")
		} else {
			wrappedErr := errors.Wrap(err, "Could not listen on :8080")
			runtime.HandleError(s.Logger.WithError(wrappedErr), wrappedErr)
		}
	}
	return nil
}
