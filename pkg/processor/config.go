package processor

import (
	"time"
)

// Config holds the processor client configuration
type Config struct {
	// ClientID is a unique identifier for this processor client instance
	ClientID string

	// ProcessorAddress specifies the address of the processor service to connect to
	ProcessorAddress string

	// MaxBatchSize determines the maximum number of allocation requests to batch together
	MaxBatchSize int

	// AllocationTimeout is the maximum duration to wait for an allocation response
	AllocationTimeout time.Duration

	// ReconnectInterval is the time to wait before retrying a failed connection
	ReconnectInterval time.Duration
}
