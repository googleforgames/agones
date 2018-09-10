package main

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"agones.dev/agones/pkg/apis/stable/v1alpha1"
	"agones.dev/agones/pkg/client/clientset/versioned"
	"agones.dev/agones/pkg/util/runtime" // for the logger
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
)

// Constants which define the fleet and namespace we are using
const namespace = "default"
const fleetname = "simple-udp"
const generatename = "simple-udp-"

// Variables for the logger and Agones Clientset
var (
	logger       = runtime.NewLoggerWithSource("main")
	agonesClient = getAgonesClient()
)

// A handler for the web server
type handler func(w http.ResponseWriter, r *http.Request)

// The structure of the json response
type result struct {
	Status v1alpha1.GameServerStatus `json:"status"`
}

// Main will set up an http server and three endpoints
func main() {
	// Serve 200 status on / for k8s health checks
	http.HandleFunc("/", handleRoot)

	// Serve 200 status on /healthz for k8s health checks
	http.HandleFunc("/healthz", handleHealthz)

	// Return the GameServerStatus of the allocated replica to the authorized client
	http.HandleFunc("/address", getOnly(basicAuth(handleAddress)))

	// Run the HTTP server using the bound certificate and key for TLS
	if err := http.ListenAndServeTLS(":8000", "/home/service/certs/tls.crt", "/home/service/certs/tls.key", nil); err != nil {
		logger.WithError(err).Fatal("HTTPS server failed to run")
	} else {
		logger.Info("HTTPS server is running on port 8000")
	}
}

// Set up our client which we will use to call the API
func getAgonesClient() *versioned.Clientset {
	// Create the in-cluster config
	config, err := rest.InClusterConfig()
	if err != nil {
		logger.WithError(err).Fatal("Could not create in cluster config")
	}

	// Access to the Agones resources through the Agones Clientset
	agonesClient, err := versioned.NewForConfig(config)
	if err != nil {
		logger.WithError(err).Fatal("Could not create the agones api clientset")
	} else {
		logger.Info("Created the agones api clientset")
	}
	return agonesClient
}

// Limit verbs the web server handles
func getOnly(h handler) handler {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			h(w, r)
			return
		}
		http.Error(w, "Get Only", http.StatusMethodNotAllowed)
	}
}

// Let the web server do basic authentication
func basicAuth(pass handler) handler {
	return func(w http.ResponseWriter, r *http.Request) {
		key, value, _ := r.BasicAuth()
		if key != "v1GameClientKey" || value != "EAEC945C371B2EC361DE399C2F11E" {
			http.Error(w, "authorization failed", http.StatusUnauthorized)
			return
		}
		pass(w, r)
	}
}

// Let / return Healthy and status code 200
func handleRoot(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, err := io.WriteString(w, "Healthy")
	if err != nil {
		logger.WithError(err).Fatal("Error writing string Healthy from /")
	}
}

// Let /healthz return Healthy and status code 200
func handleHealthz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, err := io.WriteString(w, "Healthy")
	if err != nil {
		logger.WithError(err).Fatal("Error writing string Healthy from /healthz")
	}
}

// Let /address return the GameServerStatus
func handleAddress(w http.ResponseWriter, r *http.Request) {
	status, err := allocate()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	w.Header().Set("Content-Type", "application/json")
	result, _ := json.Marshal(&result{status})
	_, err = io.WriteString(w, string(result))
	if err != nil {
		logger.WithError(err).Fatal("Error writing json from /address")
	}
}

// Return the number of ready game servers available to this fleet for allocation
func checkReadyReplicas() int32 {
	// Get a FleetInterface for this namespace
	fleetInterface := agonesClient.StableV1alpha1().Fleets(namespace)
	// Get our fleet
	fleet, err := fleetInterface.Get(fleetname, v1.GetOptions{})
	if err != nil {
		logger.WithError(err).Info("Get fleet failed")
	}

	return fleet.Status.ReadyReplicas
}

// Move a replica from ready to allocated and return the GameServerStatus
func allocate() (v1alpha1.GameServerStatus, error) {
	var result v1alpha1.GameServerStatus

	// Log the values used in the fleet allocation
	logger.WithField("namespace", namespace).Info("namespace for fa")
	logger.WithField("generatename", generatename).Info("generatename for fa")
	logger.WithField("fleetname", fleetname).Info("fleetname for fa")

	// Find out how many ready replicas the fleet has - we need at least one
	readyReplicas := checkReadyReplicas()
	logger.WithField("readyReplicas", readyReplicas).Info("numer of ready replicas")

	// Log and return an error if there are no ready replicas
	if readyReplicas < 1 {
		logger.WithField("fleetname", fleetname).Info("Insufficient ready replicas, cannot create fleet allocation")
		return result, errors.New("Insufficient ready replicas, cannot create fleet allocation")
	}

	// Get a FleetAllocationInterface for this namespace
	fleetAllocationInterface := agonesClient.StableV1alpha1().FleetAllocations(namespace)

	// Define the fleet allocation using the constants set earlier
	fa := &v1alpha1.FleetAllocation{
		ObjectMeta: v1.ObjectMeta{
			GenerateName: generatename, Namespace: namespace,
		},
		Spec: v1alpha1.FleetAllocationSpec{FleetName: fleetname},
	}

	// Create a new fleet allocation
	newFleetAllocation, err := fleetAllocationInterface.Create(fa)
	if err != nil {
		// Log and return the error if the call to Create fails
		logger.WithError(err).Info("Failed to create fleet allocation")
		return result, errors.New("Failed to ceate fleet allocation")
	}

	// Log the GameServer.Staus of the new allocation, then return those values
	logger.Info("New GameServer allocated: ", newFleetAllocation.Status.GameServer.Status)
	result = newFleetAllocation.Status.GameServer.Status
	return result, nil
}
