// Copyright 2018 Google Inc. All Rights Reserved.
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

// Package webhooks manages and receives Kubernetes Webhooks
package webhooks

import (
	"encoding/json"
	"net/http"

	"agones.dev/agones/pkg/util/runtime"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"k8s.io/api/admission/v1beta1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// Server is a http server interface to enable easier testing
type Server interface {
	Close() error
	ListenAndServeTLS(certFile, keyFile string) error
}

// WebHook manage Kubernetes webhooks
type WebHook struct {
	logger   *logrus.Entry
	mux      *http.ServeMux
	server   Server
	certFile string
	keyFile  string
	handlers map[string][]operationHandler
}

// operationHandler stores the data for a handler to match against
type operationHandler struct {
	handler   Handler
	groupKind schema.GroupKind
	operation v1beta1.Operation
}

// Handler handles a webhook's AdmissionReview coming in, and will return the
// AdmissionReview that will be the return value of the webhook
type Handler func(review v1beta1.AdmissionReview) (v1beta1.AdmissionReview, error)

// NewWebHook returns a Kubernetes webhook manager
func NewWebHook(certFile, keyFile string) *WebHook {
	mux := http.NewServeMux()
	server := http.Server{
		Addr:    ":8081",
		Handler: mux,
	}

	wh := &WebHook{
		mux:      mux,
		server:   &server,
		certFile: certFile,
		keyFile:  keyFile,
		handlers: map[string][]operationHandler{},
	}
	wh.logger = runtime.NewLoggerWithType(wh)

	return wh
}

// Run runs the webhook server, starting a https listener.
// Will block on stop channel
func (wh *WebHook) Run(workers int, stop <-chan struct{}) error {
	go func() {
		<-stop
		wh.server.Close() // nolint: errcheck,gosec
	}()

	wh.logger.WithField("webook", wh).Infof("https server started")

	err := wh.server.ListenAndServeTLS(wh.certFile, wh.keyFile)
	if err == http.ErrServerClosed {
		wh.logger.WithError(err).Info("https server closed")
		return nil
	}

	return errors.Wrap(err, "Could not listen on :8081")
}

// AddHandler adds a handler for a given path, group and kind, and operation
func (wh *WebHook) AddHandler(path string, gk schema.GroupKind, op v1beta1.Operation, h Handler) {
	if len(wh.handlers[path]) == 0 {
		wh.mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
			err := wh.handle(path, w, r)
			if err != nil {
				runtime.HandleError(wh.logger.WithField("url", r.URL), err)
				w.WriteHeader(http.StatusInternalServerError)
			}
		})
	}
	wh.logger.WithField("path", path).WithField("groupKind", gk).WithField("op", op).Info("Added webhook handler")
	wh.handlers[path] = append(wh.handlers[path], operationHandler{groupKind: gk, operation: op, handler: h})
}

// handle Handles http requests for webhooks
func (wh *WebHook) handle(path string, w http.ResponseWriter, r *http.Request) error { // nolint: interfacer
	wh.logger.WithField("path", path).Info("running webhook")

	var review v1beta1.AdmissionReview
	err := json.NewDecoder(r.Body).Decode(&review)
	if err != nil {
		return errors.Wrapf(err, "error decoding decoding json for path %v", path)
	}

	// set it to true, in case there are no handlers
	if review.Response == nil {
		review.Response = &v1beta1.AdmissionResponse{Allowed: true}
	}
	for _, oh := range wh.handlers[path] {
		if oh.operation == review.Request.Operation &&
			oh.groupKind.Kind == review.Request.Kind.Kind &&
			review.Request.Kind.Group == oh.groupKind.Group {

			review, err = oh.handler(review)
			if err != nil {
				return errors.Wrapf(err, "error with webhook handler for path %v", path)
			}
		}
	}
	err = json.NewEncoder(w).Encode(review)
	if err != nil {
		return errors.Wrapf(err, "error decoding encoding json for path %v", path)
	}

	return nil
}
