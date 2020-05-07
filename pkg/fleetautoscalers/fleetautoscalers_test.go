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
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	autoscalingv1 "agones.dev/agones/pkg/apis/autoscaling/v1"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	scaleFactor = 2
)

func TestComputeDesiredFleetSize(t *testing.T) {
	t.Parallel()

	fas, f := defaultFixtures()

	fas.Spec.Policy.Buffer.BufferSize = intstr.FromInt(20)
	fas.Spec.Policy.Buffer.MaxReplicas = 100
	f.Spec.Replicas = 50
	f.Status.Replicas = f.Spec.Replicas
	f.Status.AllocatedReplicas = 40
	f.Status.ReadyReplicas = 10

	replicas, limited, err := computeDesiredFleetSize(fas, f)
	assert.Nil(t, err)
	assert.Equal(t, replicas, int32(60))
	assert.Equal(t, limited, false)

	// test empty Policy Type
	f.Status.Replicas = 61
	fas.Spec.Policy.Type = ""
	replicas, limited, err = computeDesiredFleetSize(fas, f)
	assert.NotNil(t, err)
	assert.Equal(t, replicas, int32(61))
	assert.Equal(t, limited, false)
}

func TestApplyBufferPolicy(t *testing.T) {
	t.Parallel()

	fas, f := defaultFixtures()
	b := fas.Spec.Policy.Buffer

	b.BufferSize = intstr.FromInt(20)
	b.MaxReplicas = 100
	f.Spec.Replicas = 50
	f.Status.Replicas = f.Spec.Replicas
	f.Status.AllocatedReplicas = 40
	f.Status.ReadyReplicas = 10

	replicas, limited, err := applyBufferPolicy(b, f)
	assert.Nil(t, err)
	assert.Equal(t, replicas, int32(60))
	assert.Equal(t, limited, false)

	b.MinReplicas = 65
	f.Spec.Replicas = 50
	f.Status.Replicas = f.Spec.Replicas
	f.Status.AllocatedReplicas = 40
	f.Status.ReadyReplicas = 10
	replicas, limited, err = applyBufferPolicy(b, f)
	assert.Nil(t, err)
	assert.Equal(t, replicas, int32(65))
	assert.Equal(t, limited, true)

	b.MinReplicas = 0
	b.MaxReplicas = 55
	f.Spec.Replicas = 50
	f.Status.Replicas = f.Spec.Replicas
	f.Status.AllocatedReplicas = 40
	f.Status.ReadyReplicas = 10
	replicas, limited, err = applyBufferPolicy(b, f)
	assert.Nil(t, err)
	assert.Equal(t, replicas, int32(55))
	assert.Equal(t, limited, true)

	b.BufferSize = intstr.FromString("20%")
	b.MinReplicas = 0
	b.MaxReplicas = 100
	f.Spec.Replicas = 50
	f.Status.Replicas = f.Spec.Replicas
	f.Status.AllocatedReplicas = 50
	f.Status.ReadyReplicas = 0
	replicas, limited, err = applyBufferPolicy(b, f)
	assert.Nil(t, err)
	assert.Equal(t, replicas, int32(63))
	assert.Equal(t, limited, false)

	b.BufferSize = intstr.FromString("10%")
	b.MinReplicas = 0
	b.MaxReplicas = 10
	f.Spec.Replicas = 1
	f.Status.Replicas = f.Spec.Replicas
	f.Status.AllocatedReplicas = 1
	f.Status.ReadyReplicas = 0
	replicas, limited, err = applyBufferPolicy(b, f)
	assert.Nil(t, err)
	assert.Equal(t, replicas, int32(2))
	assert.Equal(t, limited, false)
}

type testServer struct{}

func (t testServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r == nil {
		http.Error(w, "Empty request", http.StatusInternalServerError)
		return
	}

	var faRequest autoscalingv1.FleetAutoscaleReview

	res, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	err = json.Unmarshal(res, &faRequest)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
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
	w.Header().Set("Content-Type", "application/json")
	review := &autoscalingv1.FleetAutoscaleReview{
		Request:  faReq,
		Response: &faResp,
	}
	result, _ := json.Marshal(&review)

	_, err = io.WriteString(w, string(result))
	if err != nil {
		http.Error(w, "Error writing json from /address", http.StatusInternalServerError)
	}
}

func TestApplyWebhookPolicy(t *testing.T) {
	t.Parallel()
	ts := testServer{}
	server := httptest.NewServer(ts)
	defer server.Close()

	_, f := defaultWebhookFixtures()

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
			}
			assert.Equal(t, tc.expected.replicas, replicas)
			assert.Equal(t, tc.expected.limited, limited)
		})
	}

}
