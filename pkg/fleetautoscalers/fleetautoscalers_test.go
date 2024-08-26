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
	"time"

	"github.com/stretchr/testify/assert"
	admregv1 "k8s.io/api/admissionregistration/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	k8stesting "k8s.io/client-go/testing"

	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	autoscalingv1 "agones.dev/agones/pkg/apis/autoscaling/v1"
	"agones.dev/agones/pkg/gameservers"
	agtesting "agones.dev/agones/pkg/testing"
	utilruntime "agones.dev/agones/pkg/util/runtime"
)

const (
	scaleFactor = 2
	webhookURL  = "scale"
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

	nc := map[string]gameservers.NodeCount{}

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
				err:      "wrong policy type, should be one of: Buffer, Webhook, Counter, List",
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

			m := agtesting.NewMocks()
			m.AgonesClient.AddReactor("list", "gameservers", func(action k8stesting.Action) (bool, runtime.Object, error) {
				return true, &agonesv1.GameServerList{Items: []agonesv1.GameServer{}}, nil
			})

			gameServers := m.AgonesInformerFactory.Agones().V1().GameServers()
			_, cancel := agtesting.StartInformers(m, gameServers.Informer().HasSynced)
			defer cancel()

			replicas, limited, err := computeDesiredFleetSize(fas.Spec.Policy, f, gameServers.Lister(), nc)

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
	url := webhookURL
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

	url := webhookURL
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

// nolint:dupl  // Linter errors on lines are duplicate of TestApplyListPolicy
func TestApplyCounterPolicy(t *testing.T) {
	t.Parallel()

	nc := map[string]gameservers.NodeCount{
		"n1": {Ready: 1, Allocated: 1},
	}

	modifiedFleet := func(f func(*agonesv1.Fleet)) *agonesv1.Fleet {
		_, fleet := defaultFixtures() // The ObjectMeta.Name of the defaultFixtures fleet is "fleet-1"
		f(fleet)
		return fleet
	}

	type expected struct {
		replicas int32
		limited  bool
		wantErr  bool
	}

	testCases := map[string]struct {
		fleet        *agonesv1.Fleet
		featureFlags string
		cp           *autoscalingv1.CounterPolicy
		gsList       []agonesv1.GameServer
		want         expected
	}{
		"counts and lists not enabled": {
			fleet: modifiedFleet(func(f *agonesv1.Fleet) {
				f.Spec.Template.Spec.Counters = make(map[string]agonesv1.CounterStatus)
				f.Spec.Template.Spec.Counters["rooms"] = agonesv1.CounterStatus{
					Count:    0,
					Capacity: 7}
				f.Status.Replicas = 10
				f.Status.ReadyReplicas = 5
				f.Status.AllocatedReplicas = 5
				f.Status.Counters = make(map[string]agonesv1.AggregatedCounterStatus)
				f.Status.Counters["rooms"] = agonesv1.AggregatedCounterStatus{
					Count:    31,
					Capacity: 70,
				}
			}),
			featureFlags: string(utilruntime.FeatureCountsAndLists) + "=false",
			cp: &autoscalingv1.CounterPolicy{
				Key:         "rooms",
				MaxCapacity: 100,
				MinCapacity: 10,
				BufferSize:  intstr.FromInt(10),
			},
			want: expected{
				replicas: 0,
				limited:  false,
				wantErr:  true,
			},
		},
		"fleet spec does not have counter": {
			fleet: modifiedFleet(func(f *agonesv1.Fleet) {
				f.Spec.Template.Spec.Counters = make(map[string]agonesv1.CounterStatus)
				f.Spec.Template.Spec.Counters["brooms"] = agonesv1.CounterStatus{
					Count:    0,
					Capacity: 7}
				f.Status.Replicas = 10
				f.Status.ReadyReplicas = 5
				f.Status.AllocatedReplicas = 5
				f.Status.Counters = make(map[string]agonesv1.AggregatedCounterStatus)
				f.Status.Counters["rooms"] = agonesv1.AggregatedCounterStatus{
					Count:    31,
					Capacity: 70,
				}
			}),
			featureFlags: string(utilruntime.FeatureCountsAndLists) + "=true",
			cp: &autoscalingv1.CounterPolicy{
				Key:         "rooms",
				MaxCapacity: 100,
				MinCapacity: 10,
				BufferSize:  intstr.FromInt(10),
			},
			want: expected{
				replicas: 0,
				limited:  false,
				wantErr:  true,
			},
		},
		"fleet status does not have counter": {
			fleet: modifiedFleet(func(f *agonesv1.Fleet) {
				f.Spec.Template.Spec.Counters = make(map[string]agonesv1.CounterStatus)
				f.Spec.Template.Spec.Counters["rooms"] = agonesv1.CounterStatus{
					Count:    0,
					Capacity: 7}
				f.Status.Replicas = 10
				f.Status.ReadyReplicas = 5
				f.Status.AllocatedReplicas = 5
				f.Status.Counters = make(map[string]agonesv1.AggregatedCounterStatus)
				f.Status.Counters["brooms"] = agonesv1.AggregatedCounterStatus{
					Count:    31,
					Capacity: 70,
				}
			}),
			featureFlags: string(utilruntime.FeatureCountsAndLists) + "=true",
			cp: &autoscalingv1.CounterPolicy{
				Key:         "rooms",
				MaxCapacity: 100,
				MinCapacity: 10,
				BufferSize:  intstr.FromInt(10),
			},
			want: expected{
				replicas: 0,
				limited:  false,
				wantErr:  true,
			},
		},
		"scale down": {
			fleet: modifiedFleet(func(f *agonesv1.Fleet) {
				f.Spec.Template.Spec.Counters = make(map[string]agonesv1.CounterStatus)
				f.Spec.Template.Spec.Counters["rooms"] = agonesv1.CounterStatus{
					Count:    0,
					Capacity: 7}
				f.Spec.Priorities = []agonesv1.Priority{{Type: "Counter", Key: "rooms", Order: "Ascending"}}
				f.Status.Replicas = 8
				f.Status.ReadyReplicas = 4
				f.Status.AllocatedReplicas = 4
				f.Status.Counters = make(map[string]agonesv1.AggregatedCounterStatus)
				f.Status.Counters["rooms"] = agonesv1.AggregatedCounterStatus{
					Count:    31,
					Capacity: 70,
				}
			}),
			featureFlags: string(utilruntime.FeatureCountsAndLists) + "=true",
			cp: &autoscalingv1.CounterPolicy{
				Key:         "rooms",
				MaxCapacity: 100,
				MinCapacity: 10,
				BufferSize:  intstr.FromInt(10),
			},
			gsList: []agonesv1.GameServer{
				{ObjectMeta: metav1.ObjectMeta{
					Name: "gs1",
					// We need the Label here so that the Lister can pick up that the gameserver is a part of
					// the fleet. If this was a real gameserver it would also have a label for
					// "agones.dev/gameserverset": "gameServerSetName".
					Labels: map[string]string{"agones.dev/fleet": "fleet-1"}},
					Status: agonesv1.GameServerStatus{
						// We need NodeName here for sorting, otherwise sortGameServersByLeastFullNodes
						// will return the list of GameServers in the opposite order the were return by
						// ListGameServersByGameServerSetOwner (which is a nondeterministic order).
						NodeName: "n1",
						Counters: map[string]agonesv1.CounterStatus{
							"rooms": {
								Count:    10,
								Capacity: 10,
							}}}},
				{ObjectMeta: metav1.ObjectMeta{
					Name:   "gs2",
					Labels: map[string]string{"agones.dev/fleet": "fleet-1"}},
					Status: agonesv1.GameServerStatus{
						NodeName: "n1",
						Counters: map[string]agonesv1.CounterStatus{
							"rooms": {
								Count:    3,
								Capacity: 5,
							}}}},
				{ObjectMeta: metav1.ObjectMeta{
					Name:   "gs3",
					Labels: map[string]string{"agones.dev/fleet": "fleet-1"}},
					Status: agonesv1.GameServerStatus{
						NodeName: "n1",
						Counters: map[string]agonesv1.CounterStatus{
							"rooms": {
								Count:    7,
								Capacity: 7,
							}}}},
				{ObjectMeta: metav1.ObjectMeta{
					Name:   "gs4",
					Labels: map[string]string{"agones.dev/fleet": "fleet-1"}},
					Status: agonesv1.GameServerStatus{
						NodeName: "n1",
						Counters: map[string]agonesv1.CounterStatus{
							"rooms": {
								Count:    11,
								Capacity: 14,
							}}}},
				{ObjectMeta: metav1.ObjectMeta{
					Name:   "gs5",
					Labels: map[string]string{"agones.dev/fleet": "fleet-1"}},
					Status: agonesv1.GameServerStatus{
						NodeName: "n1",
						Counters: map[string]agonesv1.CounterStatus{
							"rooms": {
								Count:    0,
								Capacity: 13,
							}}}},
				{ObjectMeta: metav1.ObjectMeta{
					Name:   "gs6",
					Labels: map[string]string{"agones.dev/fleet": "fleet-1"}},
					Status: agonesv1.GameServerStatus{
						NodeName: "n1",
						Counters: map[string]agonesv1.CounterStatus{
							"rooms": {
								Count:    0,
								Capacity: 7,
							}}}},
				{ObjectMeta: metav1.ObjectMeta{
					Name:   "gs7",
					Labels: map[string]string{"agones.dev/fleet": "fleet-1"}},
					Status: agonesv1.GameServerStatus{
						NodeName: "n1",
						Counters: map[string]agonesv1.CounterStatus{
							"rooms": {
								Count:    0,
								Capacity: 7,
							}}}},
				{ObjectMeta: metav1.ObjectMeta{
					Name:   "gs8",
					Labels: map[string]string{"agones.dev/fleet": "fleet-1"}},
					Status: agonesv1.GameServerStatus{
						NodeName: "n1",
						Counters: map[string]agonesv1.CounterStatus{
							"rooms": {
								Count:    0,
								Capacity: 7,
							}}}},
			},
			want: expected{
				replicas: 1,
				limited:  false,
				wantErr:  false,
			},
		},
		"scale up": {
			fleet: modifiedFleet(func(f *agonesv1.Fleet) {
				f.Spec.Template.Spec.Counters = make(map[string]agonesv1.CounterStatus)
				f.Spec.Template.Spec.Counters["rooms"] = agonesv1.CounterStatus{
					Count:    0,
					Capacity: 7}
				f.Status.Replicas = 10
				f.Status.ReadyReplicas = 0
				f.Status.AllocatedReplicas = 10
				f.Status.Counters = make(map[string]agonesv1.AggregatedCounterStatus)
				f.Status.Counters["rooms"] = agonesv1.AggregatedCounterStatus{
					Count:    68,
					Capacity: 70}
			}),
			featureFlags: string(utilruntime.FeatureCountsAndLists) + "=true",
			cp: &autoscalingv1.CounterPolicy{
				Key:         "rooms",
				MaxCapacity: 100,
				MinCapacity: 10,
				BufferSize:  intstr.FromInt(10),
			},
			want: expected{
				replicas: 12,
				limited:  false,
				wantErr:  false,
			},
		},
		"scale up integer": {
			fleet: modifiedFleet(func(f *agonesv1.Fleet) {
				f.Spec.Template.Spec.Counters = make(map[string]agonesv1.CounterStatus)
				f.Spec.Template.Spec.Counters["rooms"] = agonesv1.CounterStatus{
					Count:    7,
					Capacity: 10}
				f.Status.Replicas = 3
				f.Status.ReadyReplicas = 3
				f.Status.AllocatedReplicas = 0
				f.Status.Counters = make(map[string]agonesv1.AggregatedCounterStatus)
				f.Status.Counters["rooms"] = agonesv1.AggregatedCounterStatus{
					Count:    21,
					Capacity: 30}
			}),
			featureFlags: string(utilruntime.FeatureCountsAndLists) + "=true",
			cp: &autoscalingv1.CounterPolicy{
				Key:         "rooms",
				MaxCapacity: 100,
				MinCapacity: 10,
				BufferSize:  intstr.FromInt(25),
			},
			want: expected{
				replicas: 9,
				limited:  false,
				wantErr:  false,
			},
		},
		"scale same": {
			fleet: modifiedFleet(func(f *agonesv1.Fleet) {
				f.Spec.Template.Spec.Counters = make(map[string]agonesv1.CounterStatus)
				f.Spec.Template.Spec.Counters["rooms"] = agonesv1.CounterStatus{
					Count:    0,
					Capacity: 7}
				f.Status.Replicas = 10
				f.Status.ReadyReplicas = 0
				f.Status.AllocatedReplicas = 10
				f.Status.Counters = make(map[string]agonesv1.AggregatedCounterStatus)
				f.Status.Counters["rooms"] = agonesv1.AggregatedCounterStatus{
					Count:    60,
					Capacity: 70}
			}),
			featureFlags: string(utilruntime.FeatureCountsAndLists) + "=true",
			cp: &autoscalingv1.CounterPolicy{
				Key:         "rooms",
				MaxCapacity: 100,
				MinCapacity: 10,
				BufferSize:  intstr.FromInt(10),
			},
			want: expected{
				replicas: 10,
				limited:  false,
				wantErr:  false,
			},
		},
		"scale down at MinCapacity": {
			fleet: modifiedFleet(func(f *agonesv1.Fleet) {
				f.Spec.Template.Spec.Counters = make(map[string]agonesv1.CounterStatus)
				f.Spec.Template.Spec.Counters["rooms"] = agonesv1.CounterStatus{
					Count:    0,
					Capacity: 7}
				f.Status.Replicas = 10
				f.Status.ReadyReplicas = 9
				f.Status.AllocatedReplicas = 1
				f.Status.Counters = make(map[string]agonesv1.AggregatedCounterStatus)
				f.Status.Counters["rooms"] = agonesv1.AggregatedCounterStatus{
					Count:    1,
					Capacity: 70}
			}),
			featureFlags: string(utilruntime.FeatureCountsAndLists) + "=true",
			cp: &autoscalingv1.CounterPolicy{
				Key:         "rooms",
				MaxCapacity: 700,
				MinCapacity: 70,
				BufferSize:  intstr.FromInt(10),
			},
			want: expected{
				replicas: 10,
				limited:  true,
				wantErr:  false,
			},
		},
		"scale down limited": {
			fleet: modifiedFleet(func(f *agonesv1.Fleet) {
				f.Spec.Template.Spec.Counters = make(map[string]agonesv1.CounterStatus)
				f.Spec.Template.Spec.Counters["rooms"] = agonesv1.CounterStatus{
					Count:    0,
					Capacity: 7}
				f.Spec.Priorities = []agonesv1.Priority{{Type: "Counter", Key: "rooms", Order: "Descending"}}
				f.Status.Replicas = 4
				f.Status.ReadyReplicas = 3
				f.Status.AllocatedReplicas = 1
				f.Status.Counters = make(map[string]agonesv1.AggregatedCounterStatus)
				f.Status.Counters["rooms"] = agonesv1.AggregatedCounterStatus{
					Count:    1,
					Capacity: 36}
			}),
			featureFlags: string(utilruntime.FeatureCountsAndLists) + "=true",
			cp: &autoscalingv1.CounterPolicy{
				Key:         "rooms",
				MaxCapacity: 700,
				MinCapacity: 7,
				BufferSize:  intstr.FromInt(1),
			},
			gsList: []agonesv1.GameServer{
				{ObjectMeta: metav1.ObjectMeta{
					Name:   "gs1",
					Labels: map[string]string{"agones.dev/fleet": "fleet-1"}},
					Status: agonesv1.GameServerStatus{
						NodeName: "n1",
						Counters: map[string]agonesv1.CounterStatus{
							"rooms": {
								Count:    0,
								Capacity: 10,
							}}}},
				{ObjectMeta: metav1.ObjectMeta{
					Name:   "gs2",
					Labels: map[string]string{"agones.dev/fleet": "fleet-1"}},
					Status: agonesv1.GameServerStatus{
						NodeName: "n1",
						Counters: map[string]agonesv1.CounterStatus{
							"rooms": {
								Count:    1,
								Capacity: 5,
							}}}},
				{ObjectMeta: metav1.ObjectMeta{
					Name:   "gs3",
					Labels: map[string]string{"agones.dev/fleet": "fleet-1"}},
					Status: agonesv1.GameServerStatus{
						NodeName: "n1",
						Counters: map[string]agonesv1.CounterStatus{
							"rooms": {
								Count:    0,
								Capacity: 7,
							}}}},
				{ObjectMeta: metav1.ObjectMeta{
					Name:   "gs4",
					Labels: map[string]string{"agones.dev/fleet": "fleet-1"}},
					Status: agonesv1.GameServerStatus{
						NodeName: "n1",
						Counters: map[string]agonesv1.CounterStatus{
							"rooms": {
								Count:    0,
								Capacity: 14,
							}}}}},
			want: expected{
				replicas: 2,
				limited:  true,
				wantErr:  false,
			},
		},
		"scale down limited must scale up": {
			fleet: modifiedFleet(func(f *agonesv1.Fleet) {
				f.Spec.Template.Spec.Counters = make(map[string]agonesv1.CounterStatus)
				f.Spec.Template.Spec.Counters["rooms"] = agonesv1.CounterStatus{
					Count:    0,
					Capacity: 7}
				f.Status.Replicas = 7
				f.Status.ReadyReplicas = 6
				f.Status.AllocatedReplicas = 1
				f.Status.Counters = make(map[string]agonesv1.AggregatedCounterStatus)
				f.Status.Counters["rooms"] = agonesv1.AggregatedCounterStatus{
					Count:    1,
					Capacity: 49}
			}),
			featureFlags: string(utilruntime.FeatureCountsAndLists) + "=true",
			cp: &autoscalingv1.CounterPolicy{
				Key:         "rooms",
				MaxCapacity: 700,
				MinCapacity: 70,
				BufferSize:  intstr.FromInt(10),
			},
			want: expected{
				replicas: 10,
				limited:  true,
				wantErr:  false,
			},
		},
		"scale up limited": {
			fleet: modifiedFleet(func(f *agonesv1.Fleet) {
				f.Spec.Template.Spec.Counters = make(map[string]agonesv1.CounterStatus)
				f.Spec.Template.Spec.Counters["rooms"] = agonesv1.CounterStatus{
					Count:    0,
					Capacity: 7}
				f.Status.Replicas = 14
				f.Status.ReadyReplicas = 0
				f.Status.AllocatedReplicas = 14
				f.Status.Counters = make(map[string]agonesv1.AggregatedCounterStatus)
				f.Status.Counters["rooms"] = agonesv1.AggregatedCounterStatus{
					Count:    98,
					Capacity: 98}
			}),
			featureFlags: string(utilruntime.FeatureCountsAndLists) + "=true",
			cp: &autoscalingv1.CounterPolicy{
				Key:         "rooms",
				MaxCapacity: 100,
				MinCapacity: 10,
				BufferSize:  intstr.FromInt(10),
			},
			want: expected{
				replicas: 14,
				limited:  true,
				wantErr:  false,
			},
		},
		"scale up limited must scale down": {
			fleet: modifiedFleet(func(f *agonesv1.Fleet) {
				f.Spec.Template.Spec.Counters = make(map[string]agonesv1.CounterStatus)
				f.Spec.Template.Spec.Counters["rooms"] = agonesv1.CounterStatus{
					Count:    0,
					Capacity: 7}
				f.Spec.Priorities = []agonesv1.Priority{{Type: "Counter", Key: "rooms", Order: "Descending"}}
				f.Status.Replicas = 1
				f.Status.ReadyReplicas = 0
				f.Status.AllocatedReplicas = 1
				f.Status.Counters = make(map[string]agonesv1.AggregatedCounterStatus)
				f.Status.Counters["rooms"] = agonesv1.AggregatedCounterStatus{
					Count:    7,
					Capacity: 7}
			}),
			featureFlags: string(utilruntime.FeatureCountsAndLists) + "=true",
			cp: &autoscalingv1.CounterPolicy{
				Key:         "rooms",
				MaxCapacity: 2,
				MinCapacity: 0,
				BufferSize:  intstr.FromInt(1),
			},
			gsList: []agonesv1.GameServer{
				{ObjectMeta: metav1.ObjectMeta{
					Name:   "gs1",
					Labels: map[string]string{"agones.dev/fleet": "fleet-1"}},
					Status: agonesv1.GameServerStatus{
						NodeName: "n1",
						Counters: map[string]agonesv1.CounterStatus{
							"rooms": {
								Count:    7,
								Capacity: 7,
							}}}}},
			want: expected{
				replicas: 1,
				limited:  true,
				wantErr:  false,
			},
		},
		"scale down to max capacity": {
			fleet: modifiedFleet(func(f *agonesv1.Fleet) {
				f.Spec.Template.Spec.Counters = make(map[string]agonesv1.CounterStatus)
				f.Spec.Template.Spec.Counters["rooms"] = agonesv1.CounterStatus{
					Count:    0,
					Capacity: 5}
				f.Spec.Priorities = []agonesv1.Priority{{Type: "Counter", Key: "rooms", Order: "Descending"}}
				f.Status.Replicas = 3
				f.Status.ReadyReplicas = 3
				f.Status.AllocatedReplicas = 0
				f.Status.Counters = make(map[string]agonesv1.AggregatedCounterStatus)
				f.Status.Counters["rooms"] = agonesv1.AggregatedCounterStatus{
					Count:    0,
					Capacity: 15}
			}),
			featureFlags: string(utilruntime.FeatureCountsAndLists) + "=true",
			cp: &autoscalingv1.CounterPolicy{
				Key:         "rooms",
				MaxCapacity: 5,
				MinCapacity: 1,
				BufferSize:  intstr.FromInt(5),
			},
			gsList: []agonesv1.GameServer{
				{ObjectMeta: metav1.ObjectMeta{
					Name:   "gs1",
					Labels: map[string]string{"agones.dev/fleet": "fleet-1"}},
					Status: agonesv1.GameServerStatus{
						NodeName: "n1",
						Counters: map[string]agonesv1.CounterStatus{
							"rooms": {
								Count:    0,
								Capacity: 5,
							}}}},
				{ObjectMeta: metav1.ObjectMeta{
					Name:   "gs2",
					Labels: map[string]string{"agones.dev/fleet": "fleet-1"}},
					Status: agonesv1.GameServerStatus{
						NodeName: "n1",
						Counters: map[string]agonesv1.CounterStatus{
							"rooms": {
								Count:    0,
								Capacity: 5,
							}}}},
				{ObjectMeta: metav1.ObjectMeta{
					Name:   "gs3",
					Labels: map[string]string{"agones.dev/fleet": "fleet-1"}},
					Status: agonesv1.GameServerStatus{
						NodeName: "n1",
						Counters: map[string]agonesv1.CounterStatus{
							"rooms": {
								Count:    0,
								Capacity: 5,
							}}}},
			},
			want: expected{
				replicas: 1,
				limited:  false,
				wantErr:  false,
			},
		},
		"scale up to MinCapacity": {
			fleet: modifiedFleet(func(f *agonesv1.Fleet) {
				f.Spec.Template.Spec.Counters = make(map[string]agonesv1.CounterStatus)
				f.Spec.Template.Spec.Counters["rooms"] = agonesv1.CounterStatus{
					Count:    0,
					Capacity: 10}
				f.Spec.Priorities = []agonesv1.Priority{{Type: "Counter", Key: "rooms", Order: "Descending"}}
				f.Status.Replicas = 3
				f.Status.ReadyReplicas = 0
				f.Status.AllocatedReplicas = 3
				f.Status.Counters = make(map[string]agonesv1.AggregatedCounterStatus)
				f.Status.Counters["rooms"] = agonesv1.AggregatedCounterStatus{
					Count:    20,
					Capacity: 30,
				}
			}),
			featureFlags: string(utilruntime.FeatureCountsAndLists) + "=true",
			cp: &autoscalingv1.CounterPolicy{
				Key:         "rooms",
				MaxCapacity: 100,
				MinCapacity: 50,
				BufferSize:  intstr.FromString("10%"),
			},
			want: expected{
				replicas: 5,
				limited:  true,
				wantErr:  false,
			},
		},
		"scale up by percent": {
			fleet: modifiedFleet(func(f *agonesv1.Fleet) {
				f.Spec.Template.Spec.Counters = make(map[string]agonesv1.CounterStatus)
				f.Spec.Template.Spec.Counters["players"] = agonesv1.CounterStatus{
					Count:    0,
					Capacity: 1}
				f.Status.Replicas = 10
				f.Status.ReadyReplicas = 2
				f.Status.AllocatedReplicas = 8
				f.Status.Counters = make(map[string]agonesv1.AggregatedCounterStatus)
				f.Status.Counters["players"] = agonesv1.AggregatedCounterStatus{
					AllocatedCount:    8,
					AllocatedCapacity: 10,
					Count:             8,
					Capacity:          10,
				}
			}),
			featureFlags: string(utilruntime.FeatureCountsAndLists) + "=true",
			cp: &autoscalingv1.CounterPolicy{
				Key:         "players",
				MaxCapacity: 100,
				MinCapacity: 10,
				BufferSize:  intstr.FromString("30%"),
			},
			want: expected{
				replicas: 12,
				limited:  false,
				wantErr:  false,
			},
		},
		"scale up by percent with Count": {
			fleet: modifiedFleet(func(f *agonesv1.Fleet) {
				f.Spec.Template.Spec.Counters = make(map[string]agonesv1.CounterStatus)
				f.Spec.Template.Spec.Counters["players"] = agonesv1.CounterStatus{
					Count:    3,
					Capacity: 10}
				f.Status.Replicas = 3
				f.Status.ReadyReplicas = 0
				f.Status.AllocatedReplicas = 3
				f.Status.Counters = make(map[string]agonesv1.AggregatedCounterStatus)
				f.Status.Counters["players"] = agonesv1.AggregatedCounterStatus{
					AllocatedCount:    20,
					AllocatedCapacity: 30,
					Count:             20,
					Capacity:          30,
				}
			}),
			featureFlags: string(utilruntime.FeatureCountsAndLists) + "=true",
			cp: &autoscalingv1.CounterPolicy{
				Key:         "players",
				MaxCapacity: 100,
				MinCapacity: 10,
				BufferSize:  intstr.FromString("50%"),
			},
			want: expected{
				replicas: 5,
				limited:  false,
				wantErr:  false,
			},
		},
		"scale down by integer buffer": {
			fleet: modifiedFleet(func(f *agonesv1.Fleet) {
				f.Spec.Template.Spec.Counters = make(map[string]agonesv1.CounterStatus)
				f.Spec.Template.Spec.Counters["players"] = agonesv1.CounterStatus{
					Count:    7,
					Capacity: 10}
				f.Status.Replicas = 3
				f.Status.ReadyReplicas = 0
				f.Status.AllocatedReplicas = 3
				f.Status.Counters = make(map[string]agonesv1.AggregatedCounterStatus)
				f.Status.Counters["players"] = agonesv1.AggregatedCounterStatus{
					Count:    21,
					Capacity: 30,
				}
			}),
			featureFlags: string(utilruntime.FeatureCountsAndLists) + "=true",
			cp: &autoscalingv1.CounterPolicy{
				Key:         "players",
				MaxCapacity: 100,
				MinCapacity: 10,
				BufferSize:  intstr.FromInt(5),
			},
			gsList: []agonesv1.GameServer{
				{ObjectMeta: metav1.ObjectMeta{
					Name:   "gs1",
					Labels: map[string]string{"agones.dev/fleet": "fleet-1"}},
					Status: agonesv1.GameServerStatus{
						NodeName: "n1",
						Counters: map[string]agonesv1.CounterStatus{
							"players": {
								Count:    7,
								Capacity: 10,
							}}}},
				{ObjectMeta: metav1.ObjectMeta{
					Name:   "gs2",
					Labels: map[string]string{"agones.dev/fleet": "fleet-1"}},
					Status: agonesv1.GameServerStatus{
						NodeName: "n1",
						Counters: map[string]agonesv1.CounterStatus{
							"players": {
								Count:    7,
								Capacity: 10,
							}}}},
				{ObjectMeta: metav1.ObjectMeta{
					Name:   "gs3",
					Labels: map[string]string{"agones.dev/fleet": "fleet-1"}},
					Status: agonesv1.GameServerStatus{
						NodeName: "n1",
						Counters: map[string]agonesv1.CounterStatus{
							"players": {
								Count:    7,
								Capacity: 10,
							}}}},
			},
			want: expected{
				replicas: 2,
				limited:  false,
				wantErr:  false,
			},
		},
	}

	utilruntime.FeatureTestMutex.Lock()
	defer utilruntime.FeatureTestMutex.Unlock()

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			err := utilruntime.ParseFeatures(tc.featureFlags)
			assert.NoError(t, err)

			m := agtesting.NewMocks()
			m.AgonesClient.AddReactor("list", "gameservers", func(action k8stesting.Action) (bool, runtime.Object, error) {
				return true, &agonesv1.GameServerList{Items: tc.gsList}, nil
			})

			informer := m.AgonesInformerFactory.Agones().V1()
			_, cancel := agtesting.StartInformers(m,
				informer.GameServers().Informer().HasSynced)
			defer cancel()

			replicas, limited, err := applyCounterOrListPolicy(tc.cp, nil, tc.fleet, informer.GameServers().Lister(), nc)

			if tc.want.wantErr {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
				assert.Equal(t, tc.want.replicas, replicas)
				assert.Equal(t, tc.want.limited, limited)
			}
		})
	}
}

// nolint:dupl  // Linter errors on lines are duplicate of TestApplyCounterPolicy
// NOTE: Does not test for the validity of a fleet autoscaler policy (ValidateListPolicy)
func TestApplyListPolicy(t *testing.T) {
	t.Parallel()

	nc := map[string]gameservers.NodeCount{
		"n1": {Ready: 0, Allocated: 2},
		"n2": {Ready: 1},
	}

	modifiedFleet := func(f func(*agonesv1.Fleet)) *agonesv1.Fleet {
		_, fleet := defaultFixtures()
		f(fleet)
		return fleet
	}

	type expected struct {
		replicas int32
		limited  bool
		wantErr  bool
	}

	testCases := map[string]struct {
		fleet        *agonesv1.Fleet
		featureFlags string
		lp           *autoscalingv1.ListPolicy
		gsList       []agonesv1.GameServer
		want         expected
	}{
		"counts and lists not enabled": {
			fleet: modifiedFleet(func(f *agonesv1.Fleet) {
				f.Spec.Template.Spec.Lists = make(map[string]agonesv1.ListStatus)
				f.Spec.Template.Spec.Lists["gamers"] = agonesv1.ListStatus{
					Values:   []string{},
					Capacity: 7}
				f.Status.Replicas = 10
				f.Status.ReadyReplicas = 5
				f.Status.AllocatedReplicas = 5
				f.Status.Lists = make(map[string]agonesv1.AggregatedListStatus)
				f.Status.Lists["gamers"] = agonesv1.AggregatedListStatus{
					Count:    31,
					Capacity: 70,
				}
			}),
			featureFlags: string(utilruntime.FeatureCountsAndLists) + "=false",
			lp: &autoscalingv1.ListPolicy{
				Key:         "gamers",
				MaxCapacity: 100,
				MinCapacity: 10,
				BufferSize:  intstr.FromInt(10),
			},
			want: expected{
				replicas: 0,
				limited:  false,
				wantErr:  true,
			},
		},
		"fleet spec does not have list": {
			fleet: modifiedFleet(func(f *agonesv1.Fleet) {
				f.Spec.Template.Spec.Lists = make(map[string]agonesv1.ListStatus)
				f.Spec.Template.Spec.Lists["tamers"] = agonesv1.ListStatus{
					Values:   []string{},
					Capacity: 7}
				f.Status.Replicas = 10
				f.Status.ReadyReplicas = 5
				f.Status.AllocatedReplicas = 5
				f.Status.Lists = make(map[string]agonesv1.AggregatedListStatus)
				f.Status.Lists["gamers"] = agonesv1.AggregatedListStatus{
					Count:    31,
					Capacity: 70,
				}
			}),
			featureFlags: string(utilruntime.FeatureCountsAndLists) + "=true",
			lp: &autoscalingv1.ListPolicy{
				Key:         "gamers",
				MaxCapacity: 100,
				MinCapacity: 10,
				BufferSize:  intstr.FromInt(10),
			},
			want: expected{
				replicas: 0,
				limited:  false,
				wantErr:  true,
			},
		},
		"fleet status does not have list": {
			fleet: modifiedFleet(func(f *agonesv1.Fleet) {
				f.Spec.Template.Spec.Lists = make(map[string]agonesv1.ListStatus)
				f.Spec.Template.Spec.Lists["gamers"] = agonesv1.ListStatus{
					Values:   []string{},
					Capacity: 7}
				f.Status.Replicas = 10
				f.Status.ReadyReplicas = 5
				f.Status.AllocatedReplicas = 5
				f.Status.Lists = make(map[string]agonesv1.AggregatedListStatus)
				f.Status.Lists["tamers"] = agonesv1.AggregatedListStatus{
					Count:    31,
					Capacity: 70,
				}
			}),
			featureFlags: string(utilruntime.FeatureCountsAndLists) + "=true",
			lp: &autoscalingv1.ListPolicy{
				Key:         "gamers",
				MaxCapacity: 100,
				MinCapacity: 10,
				BufferSize:  intstr.FromInt(10),
			},
			want: expected{
				replicas: 0,
				limited:  false,
				wantErr:  true,
			},
		},
		"scale up": {
			fleet: modifiedFleet(func(f *agonesv1.Fleet) {
				f.Spec.Template.Spec.Lists = make(map[string]agonesv1.ListStatus)
				f.Spec.Template.Spec.Lists["gamers"] = agonesv1.ListStatus{
					Values:   []string{"default", "default2"},
					Capacity: 3}
				f.Status.Replicas = 10
				f.Status.ReadyReplicas = 0
				f.Status.AllocatedReplicas = 10
				f.Status.Lists = make(map[string]agonesv1.AggregatedListStatus)
				f.Status.Lists["gamers"] = agonesv1.AggregatedListStatus{
					Count:    29,
					Capacity: 30,
				}
			}),
			featureFlags: string(utilruntime.FeatureCountsAndLists) + "=true",
			lp: &autoscalingv1.ListPolicy{
				Key:         "gamers",
				MaxCapacity: 100,
				MinCapacity: 10,
				BufferSize:  intstr.FromInt(5),
			},
			want: expected{
				replicas: 14,
				limited:  false,
				wantErr:  false,
			},
		},
		"scale up to maxcapacity": {
			fleet: modifiedFleet(func(f *agonesv1.Fleet) {
				f.Spec.Template.Spec.Lists = make(map[string]agonesv1.ListStatus)
				f.Spec.Template.Spec.Lists["gamers"] = agonesv1.ListStatus{
					Values:   []string{"default", "default2", "default3"},
					Capacity: 5}
				f.Status.Replicas = 3
				f.Status.ReadyReplicas = 3
				f.Status.AllocatedReplicas = 0
				f.Status.Lists = make(map[string]agonesv1.AggregatedListStatus)
				f.Status.Lists["gamers"] = agonesv1.AggregatedListStatus{
					Count:    9,
					Capacity: 15,
				}
			}),
			featureFlags: string(utilruntime.FeatureCountsAndLists) + "=true",
			lp: &autoscalingv1.ListPolicy{
				Key:         "gamers",
				MaxCapacity: 25,
				MinCapacity: 15,
				BufferSize:  intstr.FromInt(15),
			},
			want: expected{
				replicas: 5,
				limited:  true,
				wantErr:  false,
			},
		},
		"scale down": {
			fleet: modifiedFleet(func(f *agonesv1.Fleet) {
				f.Spec.Template.Spec.Lists = make(map[string]agonesv1.ListStatus)
				f.Spec.Template.Spec.Lists["gamers"] = agonesv1.ListStatus{
					Values:   []string{"default"},
					Capacity: 10}
				f.Spec.Priorities = []agonesv1.Priority{{Type: "List", Key: "gamers", Order: "Descending"}}
				f.Status.Replicas = 8
				f.Status.ReadyReplicas = 6
				f.Status.AllocatedReplicas = 4
				f.Status.Lists = make(map[string]agonesv1.AggregatedListStatus)
				f.Status.Lists["gamers"] = agonesv1.AggregatedListStatus{
					Count:    15,
					Capacity: 70,
				}
			}),
			featureFlags: string(utilruntime.FeatureCountsAndLists) + "=true",
			lp: &autoscalingv1.ListPolicy{
				Key:         "gamers",
				MaxCapacity: 70,
				MinCapacity: 10,
				BufferSize:  intstr.FromInt(10),
			},
			gsList: []agonesv1.GameServer{
				{ObjectMeta: metav1.ObjectMeta{
					Name:   "gs1",
					Labels: map[string]string{"agones.dev/fleet": "fleet-1"}},
					Status: agonesv1.GameServerStatus{
						NodeName: "n1",
						Lists: map[string]agonesv1.ListStatus{
							"gamers": {
								Values:   []string{},
								Capacity: 10,
							}}}},
				{ObjectMeta: metav1.ObjectMeta{
					Name:   "gs2",
					Labels: map[string]string{"agones.dev/fleet": "fleet-1"}},
					Status: agonesv1.GameServerStatus{
						NodeName: "n1",
						Lists: map[string]agonesv1.ListStatus{
							"gamers": {
								Values:   []string{},
								Capacity: 10,
							}}}},
				{ObjectMeta: metav1.ObjectMeta{
					Name:   "gs3",
					Labels: map[string]string{"agones.dev/fleet": "fleet-1"}},
					Status: agonesv1.GameServerStatus{
						NodeName: "n1",
						Lists: map[string]agonesv1.ListStatus{
							"gamers": {
								Values:   []string{},
								Capacity: 10,
							}}}},
				{ObjectMeta: metav1.ObjectMeta{
					Name:   "gs4",
					Labels: map[string]string{"agones.dev/fleet": "fleet-1"}},
					Status: agonesv1.GameServerStatus{
						NodeName: "n1",
						Lists: map[string]agonesv1.ListStatus{
							"gamers": {
								Values:   []string{"default1", "default2", "default3", "default4", "default5", "default6", "default7", "default8"},
								Capacity: 8,
							}}}},
				{ObjectMeta: metav1.ObjectMeta{
					Name:   "gs5",
					Labels: map[string]string{"agones.dev/fleet": "fleet-1"}},
					Status: agonesv1.GameServerStatus{
						NodeName: "n1",
						Lists: map[string]agonesv1.ListStatus{
							"gamers": {
								Values:   []string{"default9", "default10", "default11", "default12"},
								Capacity: 10,
							}}}},
				{ObjectMeta: metav1.ObjectMeta{
					Name:   "gs6",
					Labels: map[string]string{"agones.dev/fleet": "fleet-1"}},
					Status: agonesv1.GameServerStatus{
						NodeName: "n1",
						Lists: map[string]agonesv1.ListStatus{
							"gamers": {
								Values:   []string{"default"},
								Capacity: 4,
							}}}},
				{ObjectMeta: metav1.ObjectMeta{
					Name:   "gs7",
					Labels: map[string]string{"agones.dev/fleet": "fleet-1"}},
					Status: agonesv1.GameServerStatus{
						NodeName: "n1",
						Lists: map[string]agonesv1.ListStatus{
							"gamers": {
								Values:   []string{"default"},
								Capacity: 8,
							}}}},
				{ObjectMeta: metav1.ObjectMeta{
					Name:   "gs8",
					Labels: map[string]string{"agones.dev/fleet": "fleet-1"}},
					Status: agonesv1.GameServerStatus{
						NodeName: "n1",
						Lists: map[string]agonesv1.ListStatus{
							"gamers": {
								Values:   []string{"default"},
								Capacity: 10,
							}}}},
			},
			want: expected{
				replicas: 4,
				limited:  false,
				wantErr:  false,
			},
		},
		"scale up limited": {
			fleet: modifiedFleet(func(f *agonesv1.Fleet) {
				f.Spec.Template.Spec.Lists = make(map[string]agonesv1.ListStatus)
				f.Spec.Template.Spec.Lists["gamers"] = agonesv1.ListStatus{
					Values:   []string{"default", "default2"},
					Capacity: 3}
				f.Status.Replicas = 10
				f.Status.ReadyReplicas = 0
				f.Status.AllocatedReplicas = 10
				f.Status.Lists = make(map[string]agonesv1.AggregatedListStatus)
				f.Status.Lists["gamers"] = agonesv1.AggregatedListStatus{
					Count:    29,
					Capacity: 30,
				}
			}),
			featureFlags: string(utilruntime.FeatureCountsAndLists) + "=true",
			lp: &autoscalingv1.ListPolicy{
				Key:         "gamers",
				MaxCapacity: 30,
				MinCapacity: 10,
				BufferSize:  intstr.FromInt(5),
			},
			want: expected{
				replicas: 10,
				limited:  true,
				wantErr:  false,
			},
		},
		"scale down limited": {
			fleet: modifiedFleet(func(f *agonesv1.Fleet) {
				f.Spec.Template.Spec.Lists = make(map[string]agonesv1.ListStatus)
				f.Spec.Template.Spec.Lists["gamers"] = agonesv1.ListStatus{
					Values:   []string{},
					Capacity: 5}
				f.Spec.Priorities = []agonesv1.Priority{{Type: "List", Key: "gamers", Order: "Ascending"}}
				f.Status.Replicas = 4
				f.Status.ReadyReplicas = 3
				f.Status.AllocatedReplicas = 1
				f.Status.Lists = make(map[string]agonesv1.AggregatedListStatus)
				f.Status.Lists["gamers"] = agonesv1.AggregatedListStatus{
					Count:    3,
					Capacity: 20,
				}
			}),
			featureFlags: string(utilruntime.FeatureCountsAndLists) + "=true",
			lp: &autoscalingv1.ListPolicy{
				Key:         "gamers",
				MaxCapacity: 100,
				MinCapacity: 10,
				BufferSize:  intstr.FromInt(1),
			},
			gsList: []agonesv1.GameServer{
				{ObjectMeta: metav1.ObjectMeta{
					Name:   "gs1",
					Labels: map[string]string{"agones.dev/fleet": "fleet-1"}},
					Status: agonesv1.GameServerStatus{
						NodeName: "n1",
						Lists: map[string]agonesv1.ListStatus{
							"gamers": {
								Values:   []string{},
								Capacity: 5,
							}}}},
				{ObjectMeta: metav1.ObjectMeta{
					Name:   "gs2",
					Labels: map[string]string{"agones.dev/fleet": "fleet-1"}},
					Status: agonesv1.GameServerStatus{
						NodeName: "n1",
						Lists: map[string]agonesv1.ListStatus{
							"gamers": {
								Values:   []string{},
								Capacity: 5,
							}}}},
				{ObjectMeta: metav1.ObjectMeta{
					Name:   "gs3",
					Labels: map[string]string{"agones.dev/fleet": "fleet-1"}},
					Status: agonesv1.GameServerStatus{
						NodeName: "n1",
						Lists: map[string]agonesv1.ListStatus{
							"gamers": {
								Values:   []string{},
								Capacity: 5,
							}}}},
				{ObjectMeta: metav1.ObjectMeta{
					Name:   "gs4",
					Labels: map[string]string{"agones.dev/fleet": "fleet-1"}},
					Status: agonesv1.GameServerStatus{
						NodeName: "n1",
						Lists: map[string]agonesv1.ListStatus{
							"gamers": {
								Values:   []string{"default1", "default2", "default3"},
								Capacity: 5,
							}}}}},
			want: expected{
				replicas: 2,
				limited:  true,
				wantErr:  false,
			},
		},
		"scale up by percent limited": {
			fleet: modifiedFleet(func(f *agonesv1.Fleet) {
				f.Spec.Template.Spec.Lists = make(map[string]agonesv1.ListStatus)
				f.Spec.Template.Spec.Lists["gamers"] = agonesv1.ListStatus{
					Values:   []string{"default", "default2", "default3"},
					Capacity: 10}
				f.Status.Replicas = 3
				f.Status.ReadyReplicas = 0
				f.Status.AllocatedReplicas = 3
				f.Status.Lists = make(map[string]agonesv1.AggregatedListStatus)
				f.Status.Lists["gamers"] = agonesv1.AggregatedListStatus{
					AllocatedCount:    20,
					AllocatedCapacity: 30,
					Count:             20,
					Capacity:          30,
				}
			}),
			featureFlags: string(utilruntime.FeatureCountsAndLists) + "=true",
			lp: &autoscalingv1.ListPolicy{
				Key:         "gamers",
				MaxCapacity: 45,
				MinCapacity: 10,
				BufferSize:  intstr.FromString("50%"),
			},
			want: expected{
				replicas: 4,
				limited:  true,
				wantErr:  false,
			},
		},
		"scale up by percent": {
			fleet: modifiedFleet(func(f *agonesv1.Fleet) {
				f.Spec.Template.Spec.Lists = make(map[string]agonesv1.ListStatus)
				f.Spec.Template.Spec.Lists["gamers"] = agonesv1.ListStatus{
					Values:   []string{"default"},
					Capacity: 3}
				f.Status.Replicas = 11
				f.Status.ReadyReplicas = 1
				f.Status.AllocatedReplicas = 10
				f.Status.Lists = make(map[string]agonesv1.AggregatedListStatus)
				f.Status.Lists["gamers"] = agonesv1.AggregatedListStatus{
					AllocatedCount:    29,
					AllocatedCapacity: 30,
					Count:             30,
					Capacity:          30,
				}
			}),
			featureFlags: string(utilruntime.FeatureCountsAndLists) + "=true",
			lp: &autoscalingv1.ListPolicy{
				Key:         "gamers",
				MaxCapacity: 50,
				MinCapacity: 10,
				BufferSize:  intstr.FromString("10%"),
			},
			want: expected{
				replicas: 13,
				limited:  false,
				wantErr:  false,
			},
		},
		"scale down by percent to Zero": {
			fleet: modifiedFleet(func(f *agonesv1.Fleet) {
				f.Spec.Template.Spec.Lists = make(map[string]agonesv1.ListStatus)
				f.Spec.Template.Spec.Lists["gamers"] = agonesv1.ListStatus{
					Values:   []string{"default", "default2"},
					Capacity: 10}
				f.Spec.Priorities = []agonesv1.Priority{{Type: "List", Key: "gamers", Order: "Descending"}}
				f.Status.Replicas = 3
				f.Status.ReadyReplicas = 3
				f.Status.AllocatedReplicas = 0
				f.Status.Lists = make(map[string]agonesv1.AggregatedListStatus)
				f.Status.Lists["gamers"] = agonesv1.AggregatedListStatus{
					AllocatedCount:    0,
					AllocatedCapacity: 0,
					Count:             15,
					Capacity:          30,
				}
			}),
			featureFlags: string(utilruntime.FeatureCountsAndLists) + "=true",
			lp: &autoscalingv1.ListPolicy{
				Key:         "gamers",
				MaxCapacity: 50,
				MinCapacity: 0,
				BufferSize:  intstr.FromString("20%"),
			},
			gsList: []agonesv1.GameServer{
				{ObjectMeta: metav1.ObjectMeta{
					Name:   "gs1",
					Labels: map[string]string{"agones.dev/fleet": "fleet-1"}},
					Status: agonesv1.GameServerStatus{
						NodeName: "n1",
						Lists: map[string]agonesv1.ListStatus{
							"gamers": {
								Values:   []string{"1", "2", "3", "4", "5"},
								Capacity: 15,
							}}}},
				{ObjectMeta: metav1.ObjectMeta{
					Name:   "gs2",
					Labels: map[string]string{"agones.dev/fleet": "fleet-1"}},
					Status: agonesv1.GameServerStatus{
						NodeName: "n1",
						Lists: map[string]agonesv1.ListStatus{
							"gamers": {
								Values:   []string{"1", "2", "3", "4", "5", "6", "7"},
								Capacity: 10,
							}}}},
				{ObjectMeta: metav1.ObjectMeta{
					Name:   "gs3",
					Labels: map[string]string{"agones.dev/fleet": "fleet-1"}},
					Status: agonesv1.GameServerStatus{
						NodeName: "n1",
						Lists: map[string]agonesv1.ListStatus{
							"gamers": {
								Values:   []string{"1", "2", "3"},
								Capacity: 5,
							}}}},
			},
			want: expected{
				replicas: 1,
				limited:  true,
				wantErr:  false,
			},
		},
		"scale down by percent": {
			fleet: modifiedFleet(func(f *agonesv1.Fleet) {
				f.Spec.Template.Spec.Lists = make(map[string]agonesv1.ListStatus)
				f.Spec.Template.Spec.Lists["gamers"] = agonesv1.ListStatus{
					Values:   []string{"default", "default2"},
					Capacity: 10}
				f.Spec.Priorities = []agonesv1.Priority{{Type: "List", Key: "gamers", Order: "Descending"}}
				f.Status.Replicas = 5
				f.Status.ReadyReplicas = 2
				f.Status.AllocatedReplicas = 3
				f.Status.Lists = make(map[string]agonesv1.AggregatedListStatus)
				f.Status.Lists["gamers"] = agonesv1.AggregatedListStatus{
					AllocatedCount:    15,
					AllocatedCapacity: 30,
					Count:             18,
					Capacity:          50,
				}
			}),
			featureFlags: string(utilruntime.FeatureCountsAndLists) + "=true",
			lp: &autoscalingv1.ListPolicy{
				Key:         "gamers",
				MaxCapacity: 50,
				MinCapacity: 0,
				BufferSize:  intstr.FromString("50%"),
			},
			gsList: []agonesv1.GameServer{
				{ObjectMeta: metav1.ObjectMeta{
					Name:   "gs1",
					Labels: map[string]string{"agones.dev/fleet": "fleet-1"}},
					Status: agonesv1.GameServerStatus{
						NodeName: "n1",
						Lists: map[string]agonesv1.ListStatus{
							"gamers": {
								Values:   []string{"1", "2", "3", "4", "5"},
								Capacity: 15,
							}}}},
				{ObjectMeta: metav1.ObjectMeta{
					Name:   "gs2",
					Labels: map[string]string{"agones.dev/fleet": "fleet-1"}},
					Status: agonesv1.GameServerStatus{
						NodeName: "n1",
						Lists: map[string]agonesv1.ListStatus{
							"gamers": {
								Values:   []string{"1", "2", "3", "4", "5", "6", "7"},
								Capacity: 10,
							}}}},
				{ObjectMeta: metav1.ObjectMeta{
					Name:   "gs3",
					Labels: map[string]string{"agones.dev/fleet": "fleet-1"}},
					Status: agonesv1.GameServerStatus{
						NodeName: "n1",
						Lists: map[string]agonesv1.ListStatus{
							"gamers": {
								Values:   []string{"1", "2", "3"},
								Capacity: 5,
							}}}},
				{ObjectMeta: metav1.ObjectMeta{
					Name:   "gs4",
					Labels: map[string]string{"agones.dev/fleet": "fleet-1"}},
					Status: agonesv1.GameServerStatus{
						NodeName: "n2",
						Lists: map[string]agonesv1.ListStatus{
							"gamers": {
								Values:   []string{"1", "2", "3"},
								Capacity: 5,
							}}}},
				{ObjectMeta: metav1.ObjectMeta{
					Name:   "gs5",
					Labels: map[string]string{"agones.dev/fleet": "fleet-1"}},
					Status: agonesv1.GameServerStatus{
						NodeName: "n2",
						Lists: map[string]agonesv1.ListStatus{
							"gamers": {
								Values:   []string{},
								Capacity: 15,
							}}}},
			},
			want: expected{
				replicas: 3,
				limited:  false,
				wantErr:  false,
			},
		},
		"scale down by percent limited": {
			fleet: modifiedFleet(func(f *agonesv1.Fleet) {
				f.Spec.Template.Spec.Lists = make(map[string]agonesv1.ListStatus)
				f.Spec.Template.Spec.Lists["gamers"] = agonesv1.ListStatus{
					Values:   []string{"default", "default2"},
					Capacity: 10}
				f.Spec.Priorities = []agonesv1.Priority{{Type: "List", Key: "gamers", Order: "Descending"}}
				f.Status.Replicas = 3
				f.Status.ReadyReplicas = 3
				f.Status.AllocatedReplicas = 0
				f.Status.Lists = make(map[string]agonesv1.AggregatedListStatus)
				f.Status.Lists["gamers"] = agonesv1.AggregatedListStatus{
					AllocatedCount:    0,
					AllocatedCapacity: 0,
					Count:             15,
					Capacity:          30,
				}
			}),
			featureFlags: string(utilruntime.FeatureCountsAndLists) + "=true",
			lp: &autoscalingv1.ListPolicy{
				Key:         "gamers",
				MaxCapacity: 50,
				MinCapacity: 1,
				BufferSize:  intstr.FromString("20%"),
			},
			gsList: []agonesv1.GameServer{
				{ObjectMeta: metav1.ObjectMeta{
					Name:   "gs1",
					Labels: map[string]string{"agones.dev/fleet": "fleet-1"}},
					Status: agonesv1.GameServerStatus{
						NodeName: "n1",
						Lists: map[string]agonesv1.ListStatus{
							"gamers": {
								Values:   []string{"1", "2", "3", "4", "5"},
								Capacity: 15,
							}}}},
				{ObjectMeta: metav1.ObjectMeta{
					Name:   "gs2",
					Labels: map[string]string{"agones.dev/fleet": "fleet-1"}},
					Status: agonesv1.GameServerStatus{
						NodeName: "n1",
						Lists: map[string]agonesv1.ListStatus{
							"gamers": {
								Values:   []string{"1", "2", "3", "4", "5", "6", "7"},
								Capacity: 10,
							}}}},
				{ObjectMeta: metav1.ObjectMeta{
					Name:   "gs3",
					Labels: map[string]string{"agones.dev/fleet": "fleet-1"}},
					Status: agonesv1.GameServerStatus{
						NodeName: "n1",
						Lists: map[string]agonesv1.ListStatus{
							"gamers": {
								Values:   []string{"1", "2", "3"},
								Capacity: 5,
							}}}},
			},
			want: expected{
				replicas: 1,
				limited:  true,
				wantErr:  false,
			},
		},
	}

	utilruntime.FeatureTestMutex.Lock()
	defer utilruntime.FeatureTestMutex.Unlock()

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			err := utilruntime.ParseFeatures(tc.featureFlags)
			assert.NoError(t, err)

			m := agtesting.NewMocks()
			m.AgonesClient.AddReactor("list", "gameservers", func(action k8stesting.Action) (bool, runtime.Object, error) {
				return true, &agonesv1.GameServerList{Items: tc.gsList}, nil
			})

			informer := m.AgonesInformerFactory.Agones().V1()
			_, cancel := agtesting.StartInformers(m,
				informer.GameServers().Informer().HasSynced)
			defer cancel()

			replicas, limited, err := applyCounterOrListPolicy(nil, tc.lp, tc.fleet, informer.GameServers().Lister(), nc)

			if tc.want.wantErr {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
				assert.Equal(t, tc.want.replicas, replicas)
				assert.Equal(t, tc.want.limited, limited)
			}
		})
	}
}

// nolint:dupl  // Linter errors on lines are duplicate of TestApplySchedulePolicy
// NOTE: Does not test for the validity of a fleet autoscaler policy (ValidateSchedulePolicy)
func TestApplySchedulePolicy(t *testing.T) {
	t.Parallel()

	type expected struct {
		replicas int32
		limited  bool
		wantErr  bool
	}

	bufferPolicy := autoscalingv1.FleetAutoscalerPolicy{
		Type: autoscalingv1.BufferPolicyType,
		Buffer: &autoscalingv1.BufferPolicy{
			BufferSize:  intstr.FromInt(1),
			MinReplicas: 3,
			MaxReplicas: 10,
		},
	}
	expectedWhenActive := expected{
		replicas: 3,
		limited:  false,
		wantErr:  false,
	}
	expectedWhenInactive := expected{
		replicas: 0,
		limited:  false,
		wantErr:  true,
	}

	testCases := map[string]struct {
		featureFlags            string
		specReplicas            int32
		statusReplicas          int32
		statusAllocatedReplicas int32
		statusReadyReplicas     int32
		now                     time.Time
		sp                      *autoscalingv1.SchedulePolicy
		gsList                  []agonesv1.GameServer
		want                    expected
	}{
		"scheduled autoscaler feature flag not enabled": {
			featureFlags: string(utilruntime.FeatureScheduledAutoscaler) + "=false",
			sp:           &autoscalingv1.SchedulePolicy{},
			want: expected{
				replicas: 0,
				limited:  false,
				wantErr:  true,
			},
		},
		"no start time": {
			featureFlags: string(utilruntime.FeatureScheduledAutoscaler) + "=true",
			now:          mustParseTime("2020-12-26T08:30:00Z"),
			sp: &autoscalingv1.SchedulePolicy{
				Between: autoscalingv1.Between{
					End: mustParseMetav1Time("2021-01-01T00:00:00Z"),
				},
				ActivePeriod: autoscalingv1.ActivePeriod{
					Timezone:  "UTC",
					StartCron: "* * * * *",
					Duration:  "48h",
				},
				Policy: bufferPolicy,
			},
			want: expectedWhenActive,
		},
		"no end time": {
			featureFlags: string(utilruntime.FeatureScheduledAutoscaler) + "=true",
			now:          mustParseTime("2021-01-02T00:00:00Z"),
			sp: &autoscalingv1.SchedulePolicy{
				Between: autoscalingv1.Between{
					Start: mustParseMetav1Time("2021-01-01T00:00:00Z"),
				},
				ActivePeriod: autoscalingv1.ActivePeriod{
					Timezone:  "UTC",
					StartCron: "* * * * *",
					Duration:  "1h",
				},
				Policy: bufferPolicy,
			},
			want: expectedWhenActive,
		},
		"no cron time": {
			featureFlags: string(utilruntime.FeatureScheduledAutoscaler) + "=true",
			now:          mustParseTime("2021-01-01T0:30:00Z"),
			sp: &autoscalingv1.SchedulePolicy{
				Between: autoscalingv1.Between{
					Start: mustParseMetav1Time("2021-01-01T00:00:00Z"),
					End:   mustParseMetav1Time("2021-01-01T01:00:00Z"),
				},
				ActivePeriod: autoscalingv1.ActivePeriod{
					Timezone: "UTC",
					Duration: "1h",
				},
				Policy: bufferPolicy,
			},
			want: expectedWhenActive,
		},
		"no duration": {
			featureFlags: string(utilruntime.FeatureScheduledAutoscaler) + "=true",
			now:          mustParseTime("2021-01-01T0:30:00Z"),
			sp: &autoscalingv1.SchedulePolicy{
				Between: autoscalingv1.Between{
					Start: mustParseMetav1Time("2021-01-01T00:00:00Z"),
					End:   mustParseMetav1Time("2021-01-01T01:00:00Z"),
				},
				ActivePeriod: autoscalingv1.ActivePeriod{
					Timezone:  "UTC",
					StartCron: "* * * * *",
				},
				Policy: bufferPolicy,
			},
			want: expectedWhenActive,
		},
		"no start time, end time, cron time, duration": {
			featureFlags: string(utilruntime.FeatureScheduledAutoscaler) + "=true",
			now:          mustParseTime("2021-01-01T00:00:00Z"),
			sp: &autoscalingv1.SchedulePolicy{
				Policy: bufferPolicy,
			},
			want: expectedWhenActive,
		},
		"daylight saving time start": {
			featureFlags: string(utilruntime.FeatureScheduledAutoscaler) + "=true",
			now:          mustParseTime("2021-03-14T02:00:00Z"),
			sp: &autoscalingv1.SchedulePolicy{
				Between: autoscalingv1.Between{
					Start: mustParseMetav1Time("2021-03-13T00:00:00Z"),
					End:   mustParseMetav1Time("2021-03-15T00:00:00Z"),
				},
				ActivePeriod: autoscalingv1.ActivePeriod{
					Timezone:  "UTC",
					StartCron: "* 2 * * *",
					Duration:  "1h",
				},
				Policy: bufferPolicy,
			},
			want: expectedWhenActive,
		},
		"daylight saving time end": {
			featureFlags: string(utilruntime.FeatureScheduledAutoscaler) + "=true",
			now:          mustParseTime("2021-11-07T01:59:59Z"),
			sp: &autoscalingv1.SchedulePolicy{
				Between: autoscalingv1.Between{
					Start: mustParseMetav1Time("2021-11-07T00:00:00Z"),
					End:   mustParseMetav1Time("2021-11-08T00:00:00Z"),
				},
				ActivePeriod: autoscalingv1.ActivePeriod{
					Timezone:  "UTC",
					StartCron: "0 2 * * *",
					Duration:  "1h",
				},
				Policy: bufferPolicy,
			},
			want: expectedWhenActive,
		},
		"new year": {
			featureFlags: string(utilruntime.FeatureScheduledAutoscaler) + "=true",
			now:          mustParseTime("2021-01-01T00:00:00Z"),
			sp: &autoscalingv1.SchedulePolicy{
				Between: autoscalingv1.Between{
					Start: mustParseMetav1Time("2020-12-31T24:59:59Z"),
					End:   mustParseMetav1Time("2021-01-02T00:00:00Z"),
				},
				ActivePeriod: autoscalingv1.ActivePeriod{
					Timezone:  "UTC",
					StartCron: "* 0 * * *",
					Duration:  "1h",
				},
				Policy: bufferPolicy,
			},
			want: expectedWhenActive,
		},
		"inactive schedule": {
			featureFlags: string(utilruntime.FeatureScheduledAutoscaler) + "=true",
			now:          mustParseTime("2023-12-12T03:49:00Z"),
			sp: &autoscalingv1.SchedulePolicy{
				Between: autoscalingv1.Between{
					Start: mustParseMetav1Time("2022-12-31T24:59:59Z"),
					End:   mustParseMetav1Time("2023-03-02T00:00:00Z"),
				},
				ActivePeriod: autoscalingv1.ActivePeriod{
					Timezone:  "UTC",
					StartCron: "* 0 * 3 *",
					Duration:  "",
				},
				Policy: bufferPolicy,
			},
			want: expectedWhenInactive,
		},
	}

	utilruntime.FeatureTestMutex.Lock()
	defer utilruntime.FeatureTestMutex.Unlock()

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			err := utilruntime.ParseFeatures(tc.featureFlags)
			assert.NoError(t, err)

			_, f := defaultFixtures()
			replicas, limited, err := applySchedulePolicy(tc.sp, f, nil, nil, tc.now)

			if tc.want.wantErr {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
				assert.Equal(t, tc.want.replicas, replicas)
				assert.Equal(t, tc.want.limited, limited)
			}
		})
	}
}

// nolint:dupl  // Linter errors on lines are duplicate of TestApplyChainPolicy
// NOTE: Does not test for the validity of a fleet autoscaler policy (ValidateChainPolicy)
func TestApplyChainPolicy(t *testing.T) {
	t.Parallel()

	// For Webhook Policy
	ts := testServer{}
	server := httptest.NewServer(ts)
	defer server.Close()
	url := webhookURL

	type expected struct {
		replicas int32
		limited  bool
		wantErr  bool
	}

	scheduleOne := autoscalingv1.ChainEntry{
		ID: "schedule-1",
		FleetAutoscalerPolicy: autoscalingv1.FleetAutoscalerPolicy{
			Type: autoscalingv1.SchedulePolicyType,
			Schedule: &autoscalingv1.SchedulePolicy{
				Between: autoscalingv1.Between{
					Start: mustParseMetav1Time("2024-08-01T10:07:36-06:00"),
				},
				ActivePeriod: autoscalingv1.ActivePeriod{
					Timezone:  "America/Chicago",
					StartCron: "* * * * *",
					Duration:  "",
				},
				Policy: autoscalingv1.FleetAutoscalerPolicy{
					Type: autoscalingv1.BufferPolicyType,
					Buffer: &autoscalingv1.BufferPolicy{
						BufferSize:  intstr.FromInt(1),
						MinReplicas: 10,
						MaxReplicas: 10,
					},
				},
			},
		},
	}
	scheduleTwo := autoscalingv1.ChainEntry{
		ID: "schedule-2",
		FleetAutoscalerPolicy: autoscalingv1.FleetAutoscalerPolicy{
			Type: autoscalingv1.SchedulePolicyType,
			Schedule: &autoscalingv1.SchedulePolicy{
				Between: autoscalingv1.Between{
					End: mustParseMetav1Time("2021-01-02T4:53:00-05:00"),
				},
				ActivePeriod: autoscalingv1.ActivePeriod{
					Timezone:  "America/New_York",
					StartCron: "0 1 3 * *",
					Duration:  "",
				},
				Policy: autoscalingv1.FleetAutoscalerPolicy{
					Type: autoscalingv1.BufferPolicyType,
					Buffer: &autoscalingv1.BufferPolicy{
						BufferSize:  intstr.FromInt(1),
						MinReplicas: 3,
						MaxReplicas: 10,
					},
				},
			},
		},
	}
	webhookEntry := autoscalingv1.ChainEntry{
		ID: "webhook policy",
		FleetAutoscalerPolicy: autoscalingv1.FleetAutoscalerPolicy{
			Type: autoscalingv1.WebhookPolicyType,
			Webhook: &autoscalingv1.WebhookPolicy{
				Service: &admregv1.ServiceReference{
					Name:      "service1",
					Namespace: "default",
					Path:      &url,
				},
				CABundle: []byte("invalid-value"),
			},
		},
	}
	defaultEntry := autoscalingv1.ChainEntry{
		ID: "default",
		FleetAutoscalerPolicy: autoscalingv1.FleetAutoscalerPolicy{
			Type: autoscalingv1.BufferPolicyType,
			Buffer: &autoscalingv1.BufferPolicy{
				BufferSize:  intstr.FromInt(1),
				MinReplicas: 6,
				MaxReplicas: 10,
			},
		},
	}

	testCases := map[string]struct {
		fleet                   *agonesv1.Fleet
		featureFlags            string
		specReplicas            int32
		statusReplicas          int32
		statusAllocatedReplicas int32
		statusReadyReplicas     int32
		now                     time.Time
		cp                      *autoscalingv1.ChainPolicy
		gsList                  []agonesv1.GameServer
		want                    expected
	}{
		"scheduled autoscaler feature flag not enabled": {
			featureFlags: string(utilruntime.FeatureScheduledAutoscaler) + "=false",
			cp:           &autoscalingv1.ChainPolicy{},
			want: expected{
				replicas: 0,
				limited:  false,
				wantErr:  true,
			},
		},
		"default policy": {
			featureFlags: string(utilruntime.FeatureScheduledAutoscaler) + "=true",
			cp:           &autoscalingv1.ChainPolicy{defaultEntry},
			want: expected{
				replicas: 6,
				limited:  true,
				wantErr:  false,
			},
		},
		"one invalid webhook policy, one default (fallthrough)": {
			featureFlags: string(utilruntime.FeatureScheduledAutoscaler) + "=true",
			cp:           &autoscalingv1.ChainPolicy{webhookEntry, defaultEntry},
			want: expected{
				replicas: 6,
				limited:  true,
				wantErr:  false,
			},
		},
		"two inactive schedule entries, no default (fall off chain)": {
			featureFlags: string(utilruntime.FeatureScheduledAutoscaler) + "=true",
			now:          mustParseTime("2021-01-01T0:30:00Z"),
			cp:           &autoscalingv1.ChainPolicy{scheduleOne, scheduleOne},
			want: expected{
				replicas: 5,
				limited:  false,
				wantErr:  true,
			},
		},
		"two inactive schedules entries, one default (fallthrough)": {
			featureFlags: string(utilruntime.FeatureScheduledAutoscaler) + "=true",
			now:          mustParseTime("2021-11-05T5:30:00Z"),
			cp:           &autoscalingv1.ChainPolicy{scheduleOne, scheduleTwo, defaultEntry},
			want: expected{
				replicas: 6,
				limited:  true,
				wantErr:  false,
			},
		},
		"two overlapping/active schedule entries, schedule-1 applied": {
			featureFlags: string(utilruntime.FeatureScheduledAutoscaler) + "=true",
			now:          mustParseTime("2024-08-01T10:07:36-06:00"),
			cp:           &autoscalingv1.ChainPolicy{scheduleOne, scheduleTwo},
			want: expected{
				replicas: 10,
				limited:  true,
				wantErr:  false,
			},
		},
	}

	utilruntime.FeatureTestMutex.Lock()
	defer utilruntime.FeatureTestMutex.Unlock()

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			err := utilruntime.ParseFeatures(tc.featureFlags)
			assert.NoError(t, err)

			_, f := defaultFixtures()
			replicas, limited, err := applyChainPolicy(*tc.cp, f, nil, nil, tc.now)

			if tc.want.wantErr {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
				assert.Equal(t, tc.want.replicas, replicas)
				assert.Equal(t, tc.want.limited, limited)
			}
		})
	}
}

// Parse a time string and return a metav1.Time
func mustParseMetav1Time(timeStr string) metav1.Time {
	t, _ := time.Parse(time.RFC3339, timeStr)
	return metav1.NewTime(t)
}

// Parse a time string and return a time.Time
func mustParseTime(timeStr string) time.Time {
	t, _ := time.Parse(time.RFC3339, timeStr)
	return t
}
