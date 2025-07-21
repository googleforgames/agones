package distributedallocator

import (
	"time"
)

// ProcessorConfig holds configuration for the processor's batch allocation and gRPC server.
type ProcessorConfig struct {
	GRPCPort                     int           // Port for the gRPC server
	RemoteAllocationTimeout      time.Duration // Timeout for remote allocation calls
	TotalRemoteAllocationTimeout time.Duration // Total timeout for remote allocation including retries
	BatchWaitTime                time.Duration // Wait time for batch aggregation
	LogLevel                     string        // Logging level
	EnableLeaderElection         bool          // Enable Kubernetes leader election
}
