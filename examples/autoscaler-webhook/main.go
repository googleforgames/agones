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

//Autoscaler webhook server which handles FleetAutoscaleReview json payload
package main

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"

	"agones.dev/agones/pkg/apis/stable/v1alpha1"
	"agones.dev/agones/pkg/util/runtime" // for the logger
)

// Constants which define thresholds to trigger scalling up and scale factor
const (
	replicaUpperThreshold = 0.7
	replicaLowerThreshold = 0.3
	scaleFactor           = 2
	minReplicasCount      = 2
)

// Variables for the logger
var (
	logger = runtime.NewLoggerWithSource("main")
)

// Main will set up an http server and three endpoints
func main() {
	// Serve 200 status on /health for k8s health checks
	http.HandleFunc("/health", handleHealth)

	// Return the target replica count which is used by Webhook fleet autoscaling policy
	http.HandleFunc("/scale", handleAutoscale)

	logger.Info("Starting HTTP server on port 8000")
	if err := http.ListenAndServe(":8000", nil); err != nil {
		logger.WithError(err).Fatal("HTTP server failed to run")
	}
}

// Let /health return Healthy and status code 200
func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, err := io.WriteString(w, "Healthy")
	if err != nil {
		logger.WithError(err).Fatal("Error writing string Healthy from /health")
	}
}

// handleAutoscale is a handler function which return the replica count
// based on received status of the fleet
func handleAutoscale(w http.ResponseWriter, r *http.Request) {
	if r == nil {
		http.Error(w, "Empty request", http.StatusInternalServerError)
		return
	}

	var faReq v1alpha1.FleetAutoscaleReview
	res, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	err = json.Unmarshal(res, &faReq)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	faResp := v1alpha1.FleetAutoscaleResponse{
		Scale:    false,
		Replicas: faReq.Request.Status.Replicas,
		UID:      faReq.Request.UID,
	}

	if faReq.Request.Status.Replicas != 0 {
		allocatedPercent := float32(faReq.Request.Status.AllocatedReplicas) / float32(faReq.Request.Status.Replicas)
		if allocatedPercent > replicaUpperThreshold {
			// After scaling we would have percentage of 0.7/2 = 0.35 > replicaLowerThreshold
			// So we won't scale down immediately after scale up
			faResp.Scale = true
			faResp.Replicas = faReq.Request.Status.Replicas * scaleFactor
		} else if allocatedPercent < replicaLowerThreshold && faReq.Request.Status.Replicas > minReplicasCount {
			faResp.Scale = true
			faResp.Replicas = faReq.Request.Status.Replicas / scaleFactor
		}
	}
	w.Header().Set("Content-Type", "application/json")
	review := &v1alpha1.FleetAutoscaleReview{
		Request:  faReq.Request,
		Response: &faResp,
	}
	logger.WithField("review", review).Info("FleetAutoscaleReview")
	result, _ := json.Marshal(&review)

	_, err = io.WriteString(w, string(result))
	if err != nil {
		logger.WithError(err).Fatal("Error writing json from /scale")
	}
}
