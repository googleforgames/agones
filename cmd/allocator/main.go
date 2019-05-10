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

	allocationv1alpha1 "agones.dev/agones/pkg/apis/allocation/v1alpha1"
	"agones.dev/agones/pkg/client/clientset/versioned"
	"agones.dev/agones/pkg/util/runtime"
	k8serror "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/rest"
)

var (
	logger = runtime.NewLoggerWithSource("main")
)

const (
	certDir = "/home/allocator/client-ca/"
	tlsDir  = "/home/allocator/tls/"
	port    = "8443"
)

// A handler for the web server
type handler func(w http.ResponseWriter, r *http.Request)

func main() {
	agonesClient, err := getAgonesClient()
	if err != nil {
		logger.WithError(err).Fatal("could not create agones client")
	}

	h := httpHandler{
		agonesClient: agonesClient,
		namespace:    os.Getenv("NAMESPACE"),
	}

	// TODO: add liveness probe
	http.HandleFunc("/v1alpha1/gameserverallocation", h.postOnly(h.allocateHandler))

	caCertPool, err := getCACertPool(certDir)
	if err != nil {
		logger.WithError(err).Fatal("could not get CA certs")
	}

	cfg := &tls.Config{
		ClientAuth: tls.RequireAndVerifyClientCert,
		ClientCAs:  caCertPool,
	}
	srv := &http.Server{
		Addr:      ":" + port,
		TLSConfig: cfg,
	}

	err = srv.ListenAndServeTLS(tlsDir+"tls.crt", tlsDir+"tls.key")
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
	namespace    string
}

func (h *httpHandler) allocateHandler(w http.ResponseWriter, r *http.Request) {
	gsa := allocationv1alpha1.GameServerAllocation{}
	if err := json.NewDecoder(r.Body).Decode(&gsa); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	allocation := h.agonesClient.AllocationV1alpha1().GameServerAllocations(h.namespace)
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
