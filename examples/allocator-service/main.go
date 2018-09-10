package main

import (
	"net/http"
	"agones.dev/agones/pkg/client/clientset/versioned"
	"agones.dev/agones/pkg/apis/stable/v1alpha1"
	"agones.dev/agones/pkg/util/runtime"	// for the logger
	"github.com/gin-gonic/gin"            // for the web server
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
)

const namespace    = "default"
const fleetname    = "simple-udp"
const generatename = "simple-udp-"

var (
	logger  = runtime.NewLoggerWithSource("main")
	address = "NoReadyGS"	// default response if no gameservers are available
	port int32            // 0 until populated
)

// Move a replica from ready to allocated and return the ip and port
func getIPAddress() (string, int32) {
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

  // Log the values used in the fleet allocation
	logger.WithField("namespace", namespace).Info("namespace for fa")
	logger.WithField("generatename", generatename).Info("generatename for fa")
	logger.WithField("fleetname", fleetname).Info("fleetname for fa")

	// Get a FleetAllocationInterface for this namespace
	fleetAllocationInterface := agonesClient.StableV1alpha1().FleetAllocations(namespace)

	// Define the fleet allocation
	fa := &v1alpha1.FleetAllocation{
		ObjectMeta: v1.ObjectMeta{
			GenerateName: generatename, Namespace: namespace,
			},
				Spec: v1alpha1.FleetAllocationSpec{FleetName: fleetname},
		}

	// Create a new fleet allocation
	newFleetAllocation, err := fleetAllocationInterface.Create(fa)
	if err != nil {
		logger.WithError(err).Fatal("Failed to create fleet allocation for ", fleetname)
	} else {
		logger.Info("Created a fleet allocation for ", fleetname)
	}

	// Log the address and port of the new allocation, then return those values
	logger.Info(
		"New GameServer allocated at %s:%d",
		newFleetAllocation.Status.GameServer.Status.Address,
		newFleetAllocation.Status.GameServer.Status.Ports[0].Port,
	)
	address = newFleetAllocation.Status.GameServer.Status.Address
	port = newFleetAllocation.Status.GameServer.Status.Ports[0].Port

	return address, port
}

// Set up an http server, fetch ip of allocated gameserver set, and return it
func main() {
	// Set up the HTTP server with default values
	router := gin.Default()

	// Serve 200 status on / for k8s health checks
	router.GET("/", func(c *gin.Context) {
		c.String(http.StatusOK, "Healthy")
	})

	// Serve 200 status on /healthz for k8s health checks
	router.GET("/healthz", func(c *gin.Context) {
		c.String(http.StatusOK, "Healthy")
	})

	// Group that can allocate game servers
	authorized := router.Group("/gameclient", gin.BasicAuth(gin.Accounts{
		"v1GameClientKey": "EAEC945C371B2EC361DE399C2F11E",
	}))

	// Return the ip and port of the allocated replica to the authorized client
	authorized.GET("/address", func(c *gin.Context) {
		address, port := getIPAddress()
		c.JSON(http.StatusOK, gin.H{"address": address, "port": port})
	})

	// Run the HTTP server
	if err := router.RunTLS(":8000", "/home/service/certs/tls.crt", "/home/service/certs/tls.key"); err != nil {
		logger.WithError(err).Fatal("HTTPS server failed to run")
	} else {
		logger.Info("HTTPS server is running on port 8000")
	}
}
