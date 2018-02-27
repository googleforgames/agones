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
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewHandler(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		path       string
		live       bool
		ready      bool
		expect     int
		expectBody string
	}{
		{
			name:   "GET /foo should generate a 404",
			method: "POST",
			path:   "/foo",
			live:   true,
			ready:  true,
			expect: http.StatusNotFound,
		},
		{
			name:   "POST /live should generate a 405 Method Not Allowed",
			method: "POST",
			path:   "/live",
			live:   true,
			ready:  true,
			expect: http.StatusMethodNotAllowed,
		},
		{
			name:   "POST /ready should generate a 405 Method Not Allowed",
			method: "POST",
			path:   "/ready",
			live:   true,
			ready:  true,
			expect: http.StatusMethodNotAllowed,
		},
		{
			name:       "with no checks, /live should succeed",
			method:     "GET",
			path:       "/live",
			live:       true,
			ready:      true,
			expect:     http.StatusOK,
			expectBody: "{}\n",
		},
		{
			name:       "with no checks, /ready should succeed",
			method:     "GET",
			path:       "/ready",
			live:       true,
			ready:      true,
			expect:     http.StatusOK,
			expectBody: "{}\n",
		},
		{
			name:       "with a failing readiness check, /live should still succeed",
			method:     "GET",
			path:       "/live?full=1",
			live:       true,
			ready:      false,
			expect:     http.StatusOK,
			expectBody: "{}\n",
		},
		{
			name:       "with a failing readiness check, /ready should fail",
			method:     "GET",
			path:       "/ready?full=1",
			live:       true,
			ready:      false,
			expect:     http.StatusServiceUnavailable,
			expectBody: "{\n    \"test-readiness-check\": \"failed readiness check\"\n}\n",
		},
		{
			name:       "with a failing liveness check, /live should fail",
			method:     "GET",
			path:       "/live?full=1",
			live:       false,
			ready:      true,
			expect:     http.StatusServiceUnavailable,
			expectBody: "{\n    \"test-liveness-check\": \"failed liveness check\"\n}\n",
		},
		{
			name:       "with a failing liveness check, /ready should fail",
			method:     "GET",
			path:       "/ready?full=1",
			live:       false,
			ready:      true,
			expect:     http.StatusServiceUnavailable,
			expectBody: "{\n    \"test-liveness-check\": \"failed liveness check\"\n}\n",
		},
		{
			name:       "with a failing liveness check, /ready without full=1 should fail with an empty body",
			method:     "GET",
			path:       "/ready",
			live:       false,
			ready:      true,
			expect:     http.StatusServiceUnavailable,
			expectBody: "{}\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewHandler()

			if !tt.live {
				h.AddLivenessCheck("test-liveness-check", func() error {
					return errors.New("failed liveness check")
				})
			}

			if !tt.ready {
				h.AddReadinessCheck("test-readiness-check", func() error {
					return errors.New("failed readiness check")
				})
			}

			req, err := http.NewRequest(tt.method, tt.path, nil)
			assert.NoError(t, err)

			reqStr := tt.method + " " + tt.path
			rr := httptest.NewRecorder()
			h.ServeHTTP(rr, req)
			assert.Equal(t, tt.expect, rr.Code, "wrong code for %q", reqStr)

			if tt.expectBody != "" {
				assert.Equal(t, tt.expectBody, rr.Body.String(), "wrong body for %q", reqStr)
			}
		})
	}
}
