// Copyright 2025 Google LLC All Rights Reserved.
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
