/*
 * Copyright 2018 Google LLC All Rights Reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package fleetautoscalers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	autoscalingv1 "agones.dev/agones/pkg/apis/autoscaling/v1"
	"github.com/stretchr/testify/assert"
	admregv1 "k8s.io/api/admissionregistration/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	scaleFactor = 2
)

type testServer struct{}

func (t testServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r == nil {
		http.Error(w, "Empty request", http.StatusInternalServerError)
		return
	}

	var faRequest autoscalingv1.FleetAutoscaleReview

	res, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	err = json.Unmarshal(res, &faRequest)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// return different errors for tests
	if faRequest.Request.Status.AllocatedReplicas == -10 {
		http.Error(w, "Wrong Status Replicas Parameter", http.StatusInternalServerError)
		return
	}

	if faRequest.Request.Status.AllocatedReplicas == -20 {
		_, err = io.WriteString(w, "invalid data")
		if err != nil {
			http.Error(w, "Error writing json from /address", http.StatusInternalServerError)
			return
		}
	}

	faReq := faRequest.Request
	faResp := autoscalingv1.FleetAutoscaleResponse{
		Scale:    false,
		Replicas: faReq.Status.Replicas,
		UID:      faReq.UID,
	}
	allocatedPercent := float32(faReq.Status.AllocatedReplicas) / float32(faReq.Status.Replicas)
	if allocatedPercent > 0.7 {
		faResp.Scale = true
		faResp.Replicas = faReq.Status.Replicas * scaleFactor
	}

	review := &autoscalingv1.FleetAutoscaleReview{
		Request:  faReq,
		Response: &faResp,
	}

	result, err := json.Marshal(&review)
	if err != nil {
		http.Error(w, "Error marshaling json", http.StatusInternalServerError)
		return
	}

	_, err = io.WriteString(w, string(result))
	if err != nil {
		http.Error(w, "Error writing json from /address", http.StatusInternalServerError)
		return
	}
}

func TestComputeDesiredFleetSize(t *testing.T) {
	t.Parallel()

	fas, f := defaultFixtures()

	type expected struct {
		replicas int32
		limited  bool
		err      string
	}

	var testCases = []struct {
		description             string
		specReplicas            int32
		statusReplicas          int32
		statusAllocatedReplicas int32
		statusReadyReplicas     int32
		policy                  autoscalingv1.FleetAutoscalerPolicy
		expected                expected
	}{
		{
			description:             "Increase replicas",
			specReplicas:            50,
			statusReplicas:          50,
			statusAllocatedReplicas: 40,
			statusReadyReplicas:     10,
			policy: autoscalingv1.FleetAutoscalerPolicy{
				Type: autoscalingv1.BufferPolicyType,
				Buffer: &autoscalingv1.BufferPolicy{
					BufferSize:  intstr.FromInt(20),
					MaxReplicas: 100,
				},
			},
			expected: expected{
				replicas: 60,
				limited:  false,
				err:      "",
			},
		},
		{
			description:             "Wrong policy",
			specReplicas:            50,
			statusReplicas:          60,
			statusAllocatedReplicas: 40,
			statusReadyReplicas:     10,
			policy: autoscalingv1.FleetAutoscalerPolicy{
				Type: "",
				Buffer: &autoscalingv1.BufferPolicy{
					BufferSize:  intstr.FromInt(20),
					MaxReplicas: 100,
				},
			},
			expected: expected{
				replicas: 0,
				limited:  false,
				err:      "wrong policy type, should be one of: Buffer, Webhook",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			fas.Spec.Policy = tc.policy
			f.Spec.Replicas = tc.specReplicas
			f.Status.Replicas = tc.statusReplicas
			f.Status.AllocatedReplicas = tc.statusAllocatedReplicas
			f.Status.ReadyReplicas = tc.statusReadyReplicas

			replicas, limited, err := computeDesiredFleetSize(fas, f)

			if tc.expected.err != "" && assert.NotNil(t, err) {
				assert.Equal(t, tc.expected.err, err.Error())
			} else {
				assert.Nil(t, err)
				assert.Equal(t, tc.expected.replicas, replicas)
				assert.Equal(t, tc.expected.limited, limited)
			}
		})
	}
}

func TestApplyBufferPolicy(t *testing.T) {
	t.Parallel()

	_, f := defaultFixtures()

	type expected struct {
		replicas int32
		limited  bool
		err      string
	}

	var testCases = []struct {
		description             string
		specReplicas            int32
		statusReplicas          int32
		statusAllocatedReplicas int32
		statusReadyReplicas     int32
		buffer                  *autoscalingv1.BufferPolicy
		expected                expected
	}{
		{
			description:             "Increase replicas",
			specReplicas:            50,
			statusReplicas:          50,
			statusAllocatedReplicas: 40,
			statusReadyReplicas:     10,
			buffer: &autoscalingv1.BufferPolicy{
				BufferSize:  intstr.FromInt(20),
				MaxReplicas: 100,
			},
			expected: expected{
				replicas: 60,
				limited:  false,
				err:      "",
			},
		},
		{
			description:             "Min replicas set, limited == true",
			specReplicas:            50,
			statusReplicas:          50,
			statusAllocatedReplicas: 40,
			statusReadyReplicas:     10,
			buffer: &autoscalingv1.BufferPolicy{
				BufferSize:  intstr.FromInt(20),
				MinReplicas: 65,
				MaxReplicas: 100,
			},
			expected: expected{
				replicas: 65,
				limited:  true,
				err:      "",
			},
		},
		{
			description:             "Replicas == max",
			specReplicas:            50,
			statusReplicas:          50,
			statusAllocatedReplicas: 40,
			statusReadyReplicas:     10,
			buffer: &autoscalingv1.BufferPolicy{
				BufferSize:  intstr.FromInt(20),
				MinReplicas: 0,
				MaxReplicas: 55,
			},
			expected: expected{
				replicas: 55,
				limited:  true,
				err:      "",
			},
		},
		{
			description:             "FromString buffer size, scale up",
			specReplicas:            50,
			statusReplicas:          50,
			statusAllocatedReplicas: 50,
			statusReadyReplicas:     0,
			buffer: &autoscalingv1.BufferPolicy{
				BufferSize:  intstr.FromString("20%"),
				MinReplicas: 0,
				MaxReplicas: 100,
			},
			expected: expected{
				replicas: 63,
				limited:  false,
				err:      "",
			},
		},
		{
			description:             "FromString buffer size, scale up twice",
			specReplicas:            1,
			statusReplicas:          1,
			statusAllocatedReplicas: 1,
			statusReadyReplicas:     0,
			buffer: &autoscalingv1.BufferPolicy{
				BufferSize:  intstr.FromString("10%"),
				MinReplicas: 0,
				MaxReplicas: 10,
			},
			expected: expected{
				replicas: 2,
				limited:  false,
				err:      "",
			},
		},
		{
			description:             "FromString buffer size is invalid, err received",
			specReplicas:            1,
			statusReplicas:          1,
			statusAllocatedReplicas: 1,
			statusReadyReplicas:     0,
			buffer: &autoscalingv1.BufferPolicy{
				BufferSize:  intstr.FromString("asd"),
				MinReplicas: 0,
				MaxReplicas: 10,
			},
			expected: expected{
				replicas: 0,
				limited:  false,
				err:      "invalid value for IntOrString: invalid value \"asd\": strconv.Atoi: parsing \"asd\": invalid syntax",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			f.Spec.Replicas = tc.specReplicas
			f.Status.Replicas = tc.statusReplicas
			f.Status.AllocatedReplicas = tc.statusAllocatedReplicas
			f.Status.ReadyReplicas = tc.statusReadyReplicas

			replicas, limited, err := applyBufferPolicy(tc.buffer, f)

			if tc.expected.err != "" && assert.NotNil(t, err) {
				assert.Equal(t, tc.expected.err, err.Error())
			} else {
				assert.Nil(t, err)
				assert.Equal(t, tc.expected.replicas, replicas)
				assert.Equal(t, tc.expected.limited, limited)
			}
		})
	}
}

func TestApplyWebhookPolicy(t *testing.T) {
	t.Parallel()
	ts := testServer{}
	server := httptest.NewServer(ts)
	defer server.Close()

	_, f := defaultWebhookFixtures()
	url := "scale"
	emptyString := ""
	invalidURL := ")1golang.org/"
	wrongServerURL := "http://127.0.0.1:1"

	type expected struct {
		replicas int32
		limited  bool
		err      string
	}

	var testCases = []struct {
		description             string
		webhookPolicy           *autoscalingv1.WebhookPolicy
		specReplicas            int32
		statusReplicas          int32
		statusAllocatedReplicas int32
		statusReadyReplicas     int32
		expected                expected
	}{
		{
			description: "Allocated replicas per cent < 70%, no scaling",
			webhookPolicy: &autoscalingv1.WebhookPolicy{
				Service: nil,
				URL:     &(server.URL),
			},
			specReplicas:            50,
			statusReplicas:          50,
			statusAllocatedReplicas: 10,
			statusReadyReplicas:     40,
			expected: expected{
				replicas: 50,
				limited:  false,
				err:      "",
			},
		},
		{
			description: "Allocated replicas per cent == 70%, no scaling",
			webhookPolicy: &autoscalingv1.WebhookPolicy{
				Service: nil,
				URL:     &(server.URL),
			},
			specReplicas:            50,
			statusReplicas:          50,
			statusAllocatedReplicas: 35,
			statusReadyReplicas:     15,
			expected: expected{
				replicas: 50,
				limited:  false,
				err:      "",
			},
		},
		{
			description: "Allocated replicas per cent 80% > 70%, scale up",
			webhookPolicy: &autoscalingv1.WebhookPolicy{
				Service: nil,
				URL:     &(server.URL),
			},
			specReplicas:            50,
			statusReplicas:          50,
			statusAllocatedReplicas: 40,
			statusReadyReplicas:     10,
			expected: expected{
				replicas: 50 * scaleFactor,
				limited:  false,
				err:      "",
			},
		},
		{
			description:   "nil WebhookPolicy, error returned",
			webhookPolicy: nil,
			expected: expected{
				replicas: 0,
				limited:  false,
				err:      "webhookPolicy parameter must not be nil",
			},
		},
		{
			description: "URL and Service are not nil",
			webhookPolicy: &autoscalingv1.WebhookPolicy{
				Service: &admregv1.ServiceReference{
					Name:      "service1",
					Namespace: "default",
					Path:      &url,
				},
				URL: &(server.URL),
			},
			expected: expected{
				replicas: 0,
				limited:  false,
				err:      "service and URL cannot be used simultaneously",
			},
		},
		{
			description: "URL not nil but empty",
			webhookPolicy: &autoscalingv1.WebhookPolicy{
				Service: nil,
				URL:     &emptyString,
			},
			expected: expected{
				replicas: 0,
				limited:  false,
				err:      "URL was not provided",
			},
		},
		{
			description: "Invalid URL",
			webhookPolicy: &autoscalingv1.WebhookPolicy{
				Service: nil,
				URL:     &invalidURL,
			},
			expected: expected{
				replicas: 0,
				limited:  false,
				err:      "parse \")1golang.org/\": invalid URI for request",
			},
		},
		{
			description: "Service name is empty",
			webhookPolicy: &autoscalingv1.WebhookPolicy{
				Service: &admregv1.ServiceReference{
					Name:      "",
					Namespace: "default",
					Path:      &url,
				},
			},
			expected: expected{
				replicas: 0,
				limited:  false,
				err:      "service name was not provided",
			},
		},
		{
			description: "No certs",
			webhookPolicy: &autoscalingv1.WebhookPolicy{
				Service: &admregv1.ServiceReference{
					Name:      "service1",
					Namespace: "default",
					Path:      &url,
				},
				CABundle: []byte("invalid-value"),
			},
			expected: expected{
				replicas: 0,
				limited:  false,
				err:      "no certs were appended from caBundle",
			},
		},
		{
			description: "Wrong server URL",
			webhookPolicy: &autoscalingv1.WebhookPolicy{
				Service: nil,
				URL:     &wrongServerURL,
			},
			expected: expected{
				replicas: 0,
				limited:  false,
				err:      "Post \"http://127.0.0.1:1\": dial tcp 127.0.0.1:1: connect: connection refused",
			},
		},
		{
			description: "Handle server error",
			webhookPolicy: &autoscalingv1.WebhookPolicy{
				Service: nil,
				URL:     &(server.URL),
			},
			specReplicas:   50,
			statusReplicas: 50,
			// hardcoded value in a server implementation
			statusAllocatedReplicas: -10,
			statusReadyReplicas:     40,
			expected: expected{
				replicas: 50,
				limited:  false,
				err:      fmt.Sprintf("bad status code %d from the server: %s", http.StatusInternalServerError, server.URL),
			},
		},
		{
			description: "Handle invalid response from the server",
			webhookPolicy: &autoscalingv1.WebhookPolicy{
				Service: nil,
				URL:     &(server.URL),
			},
			specReplicas:            50,
			statusReplicas:          50,
			statusAllocatedReplicas: -20,
			statusReadyReplicas:     40,
			expected: expected{
				replicas: 50,
				limited:  false,
				err:      "invalid character 'i' looking for beginning of value",
			},
		},
		{
			description: "Service and URL are nil",
			webhookPolicy: &autoscalingv1.WebhookPolicy{
				Service: nil,
				URL:     nil,
			},
			expected: expected{
				replicas: 0,
				limited:  false,
				err:      "service was not provided, either URL or Service must be provided",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			f.Spec.Replicas = tc.specReplicas
			f.Status.Replicas = tc.statusReplicas
			f.Status.AllocatedReplicas = tc.statusAllocatedReplicas
			f.Status.ReadyReplicas = tc.statusReadyReplicas

			replicas, limited, err := applyWebhookPolicy(tc.webhookPolicy, f)

			if tc.expected.err != "" && assert.NotNil(t, err) {
				assert.Equal(t, tc.expected.err, err.Error())
			} else {
				assert.Equal(t, tc.expected.replicas, replicas)
				assert.Equal(t, tc.expected.limited, limited)
				assert.Nil(t, err)
			}
		})
	}
}

func TestApplyWebhookPolicyNilFleet(t *testing.T) {
	t.Parallel()

	url := "scale"
	w := &autoscalingv1.WebhookPolicy{
		Service: &admregv1.ServiceReference{
			Name:      "service1",
			Namespace: "default",
			Path:      &url,
		},
	}

	replicas, limited, err := applyWebhookPolicy(w, nil)

	if assert.NotNil(t, err) {
		assert.Equal(t, "fleet parameter must not be nil", err.Error())
	}

	assert.False(t, limited)
	assert.Zero(t, replicas)
}

func TestCreateURL(t *testing.T) {
	t.Parallel()
	var nonStandardPort int32 = 8888

	var testCases = []struct {
		description string
		scheme      string
		name        string
		namespace   string
		path        string
		port        *int32
		expected    string
	}{
		{
			description: "OK, path not empty",
			scheme:      "http",
			name:        "service1",
			namespace:   "default",
			path:        "scale",
			expected:    "http://service1.default.svc:8000/scale",
		},
		{
			description: "OK, path not empty with slash",
			scheme:      "http",
			name:        "service1",
			namespace:   "default",
			path:        "/scale",
			expected:    "http://service1.default.svc:8000/scale",
		},
		{
			description: "OK, path is empty",
			scheme:      "http",
			name:        "service1",
			namespace:   "default",
			path:        "",
			expected:    "http://service1.default.svc:8000",
		},
		{
			description: "OK, port specified",
			scheme:      "http",
			name:        "service1",
			namespace:   "default",
			path:        "scale",
			port:        &nonStandardPort,
			expected:    "http://service1.default.svc:8888/scale",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			res := createURL(tc.scheme, tc.name, tc.namespace, tc.path, tc.port)

			if assert.NotNil(t, res) {
				assert.Equal(t, tc.expected, res.String())
			}
		})
	}
}

func TestBuildURLFromWebhookPolicyNoNamespace(t *testing.T) {
	url := "testurl"

	type expected struct {
		url string
		err string
	}

	var testCases = []struct {
		description   string
		webhookPolicy *autoscalingv1.WebhookPolicy
		expected      expected
	}{
		{
			description: "No namespace provided, default should be used",
			webhookPolicy: &autoscalingv1.WebhookPolicy{
				Service: &admregv1.ServiceReference{
					Name:      "service1",
					Namespace: "",
					Path:      &url,
				},
			},
			expected: expected{
				url: "http://service1.default.svc:8000/testurl",
				err: "",
			},
		},
		{
			description: "No url provided, empty string should be used",
			webhookPolicy: &autoscalingv1.WebhookPolicy{
				Service: &admregv1.ServiceReference{
					Name:      "service1",
					Namespace: "test",
					Path:      nil,
				},
			},
			expected: expected{
				url: "http://service1.test.svc:8000",
				err: "",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			url, err := buildURLFromWebhookPolicy(tc.webhookPolicy)

			if tc.expected.err != "" && assert.NotNil(t, err) {
				assert.Equal(t, tc.expected.err, err.Error())
			} else {
				assert.Nil(t, err)
				assert.Equal(t, tc.expected.url, url.String())
			}
		})
	}
}
