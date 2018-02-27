// Copyright 2017 by the contributors.
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

package healthcheck

import (
	"net/http"
)

// Check is a health/readiness check.
type Check func() error

// Handler is an http.Handler with additional methods that register health and
// readiness checks. It handles handle "/live" and "/ready" HTTP
// endpoints.
type Handler interface {
	// The Handler is an http.Handler, so it can be exposed directly and handle
	// /live and /ready endpoints.
	http.Handler

	// AddLivenessCheck adds a check that indicates that this instance of the
	// application should be destroyed or restarted. A failed liveness check
	// indicates that this instance is unhealthy, not some upstream dependency.
	// Every liveness check is also included as a readiness check.
	AddLivenessCheck(name string, check Check)

	// AddReadinessCheck adds a check that indicates that this instance of the
	// application is currently unable to serve requests because of an upstream
	// or some transient failure. If a readiness check fails, this instance
	// should no longer receiver requests, but should not be restarted or
	// destroyed.
	AddReadinessCheck(name string, check Check)

	// LiveEndpoint is the HTTP handler for just the /live endpoint, which is
	// useful if you need to attach it into your own HTTP handler tree.
	LiveEndpoint(http.ResponseWriter, *http.Request)

	// ReadyEndpoint is the HTTP handler for just the /ready endpoint, which is
	// useful if you need to attach it into your own HTTP handler tree.
	ReadyEndpoint(http.ResponseWriter, *http.Request)
}
