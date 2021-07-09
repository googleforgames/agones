// Copyright 2018 Google LLC All Rights Reserved.
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

package fleetautoscalers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	autoscalingv1 "agones.dev/agones/pkg/apis/autoscaling/v1"
	agtesting "agones.dev/agones/pkg/testing"
	utilruntime "agones.dev/agones/pkg/util/runtime"
	"agones.dev/agones/pkg/util/webhooks"
	"github.com/heptiolabs/healthcheck"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	admissionv1 "k8s.io/api/admission/v1"
	admregv1 "k8s.io/api/admissionregistration/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	k8stesting "k8s.io/client-go/testing"
)

var (
	gvk = metav1.GroupVersionKind(agonesv1.SchemeGroupVersion.WithKind("FleetAutoscaler"))
)

func TestControllerCreationValidationHandler(t *testing.T) {
	t.Parallel()

	t.Run("valid fleet autoscaler", func(t *testing.T) {
		c, m := newFakeController()
		fas, _ := defaultFixtures()
		_, cancel := agtesting.StartInformers(m)
		defer cancel()

		review, err := newAdmissionReview(*fas)
		assert.Nil(t, err)

		result, err := c.validationHandler(review)
		assert.Nil(t, err)
		assert.True(t, result.Response.Allowed, fmt.Sprintf("%#v", result.Response))
	})

	t.Run("invalid fleet autoscaler", func(t *testing.T) {
		c, m := newFakeController()
		fas, _ := defaultFixtures()
		// this make it invalid
		fas.Spec.Policy.Buffer = nil

		_, cancel := agtesting.StartInformers(m)
		defer cancel()

		review, err := newAdmissionReview(*fas)
		assert.Nil(t, err)

		result, err := c.validationHandler(review)
		assert.Nil(t, err)
		assert.False(t, result.Response.Allowed, fmt.Sprintf("%#v", result.Response))
		assert.Equal(t, metav1.StatusFailure, result.Response.Result.Status)
		assert.Equal(t, metav1.StatusReasonInvalid, result.Response.Result.Reason)
		assert.NotEmpty(t, result.Response.Result.Details)
	})

	t.Run("unable to unmarshal AdmissionRequest", func(t *testing.T) {
		c, _ := newFakeController()

		review, err := newInvalidAdmissionReview()
		assert.Nil(t, err)

		_, err = c.validationHandler(review)

		if assert.NotNil(t, err) {
			assert.Equal(t, "error unmarshalling original FleetAutoscaler json: \"MQ==\": json: cannot unmarshal string into Go value of type v1.FleetAutoscaler", err.Error())
		}
	})
}

func TestWebhookControllerCreationValidationHandler(t *testing.T) {
	t.Parallel()

	t.Run("valid fleet autoscaler", func(t *testing.T) {
		c, m := newFakeController()
		fas, _ := defaultWebhookFixtures()
		_, cancel := agtesting.StartInformers(m)
		defer cancel()

		review, err := newAdmissionReview(*fas)
		assert.Nil(t, err)

		result, err := c.validationHandler(review)
		assert.Nil(t, err)
		assert.True(t, result.Response.Allowed, fmt.Sprintf("%#v", result.Response))
	})

	t.Run("invalid fleet autoscaler", func(t *testing.T) {
		c, m := newFakeController()
		fas, _ := defaultWebhookFixtures()
		// this make it invalid
		fas.Spec.Policy.Webhook = nil

		_, cancel := agtesting.StartInformers(m)
		defer cancel()

		review, err := newAdmissionReview(*fas)
		assert.Nil(t, err)

		result, err := c.validationHandler(review)
		assert.Nil(t, err)
		assert.False(t, result.Response.Allowed, fmt.Sprintf("%#v", result.Response))
		assert.Equal(t, metav1.StatusFailure, result.Response.Result.Status)
		assert.Equal(t, metav1.StatusReasonInvalid, result.Response.Result.Reason)
		assert.NotEmpty(t, result.Response.Result.Details)
	})
}

// nolint:dupl
func TestControllerSyncFleetAutoscaler(t *testing.T) {
	utilruntime.FeatureTestMutex.Lock()
	defer utilruntime.FeatureTestMutex.Unlock()

	assert.NoError(t, utilruntime.ParseFeatures(string(utilruntime.FeatureCustomFasSyncInterval)+"=false"))

	t.Run("scaling up, buffer policy", func(t *testing.T) {
		t.Parallel()
		c, m := newFakeController()
		fas, f := defaultFixtures()
		fas.Spec.Policy.Buffer.BufferSize = intstr.FromInt(7)

		f.Spec.Replicas = 5
		f.Status.Replicas = 5
		f.Status.AllocatedReplicas = 5
		f.Status.ReadyReplicas = 0

		fUpdated := false
		fasUpdated := false

		m.AgonesClient.AddReactor("list", "fleetautoscalers", func(action k8stesting.Action) (bool, runtime.Object, error) {
			return true, &autoscalingv1.FleetAutoscalerList{Items: []autoscalingv1.FleetAutoscaler{*fas}}, nil
		})

		m.AgonesClient.AddReactor("update", "fleetautoscalers", func(action k8stesting.Action) (bool, runtime.Object, error) {
			fasUpdated = true
			ca := action.(k8stesting.UpdateAction)
			fas := ca.GetObject().(*autoscalingv1.FleetAutoscaler)
			assert.Equal(t, fas.Status.AbleToScale, true)
			assert.Equal(t, fas.Status.ScalingLimited, false)
			assert.Equal(t, fas.Status.CurrentReplicas, int32(5))
			assert.Equal(t, fas.Status.DesiredReplicas, int32(12))
			assert.NotNil(t, fas.Status.LastScaleTime)
			return true, fas, nil
		})

		m.AgonesClient.AddReactor("list", "fleets", func(action k8stesting.Action) (bool, runtime.Object, error) {
			return true, &agonesv1.FleetList{Items: []agonesv1.Fleet{*f}}, nil
		})

		m.AgonesClient.AddReactor("update", "fleets", func(action k8stesting.Action) (bool, runtime.Object, error) {
			fUpdated = true
			ca := action.(k8stesting.UpdateAction)
			f := ca.GetObject().(*agonesv1.Fleet)
			assert.Equal(t, f.Spec.Replicas, int32(12))
			return true, f, nil
		})

		ctx, cancel := agtesting.StartInformers(m, c.fleetSynced, c.fleetAutoscalerSynced)
		defer cancel()

		err := c.syncFleetAutoscaler(ctx, "default/fas-1")
		assert.Nil(t, err)
		assert.True(t, fUpdated, "fleet should have been updated")
		assert.True(t, fasUpdated, "fleetautoscaler should have been updated")
		agtesting.AssertEventContains(t, m.FakeRecorder.Events, "AutoScalingFleet")
		agtesting.AssertNoEvent(t, m.FakeRecorder.Events)
	})

	t.Run("scaling up, webhook policy", func(t *testing.T) {
		t.Parallel()
		c, m := newFakeController()
		fas, f := defaultWebhookFixtures()
		f.Spec.Replicas = 50
		f.Status.Replicas = f.Spec.Replicas
		f.Status.AllocatedReplicas = 45
		f.Status.ReadyReplicas = 0

		ts := testServer{}
		server := httptest.NewServer(ts)
		defer server.Close()

		fas.Spec.Policy.Webhook.URL = &(server.URL)
		fas.Spec.Policy.Webhook.Service = nil

		fUpdated := false
		fasUpdated := false

		m.AgonesClient.AddReactor("list", "fleetautoscalers", func(action k8stesting.Action) (bool, runtime.Object, error) {
			return true, &autoscalingv1.FleetAutoscalerList{Items: []autoscalingv1.FleetAutoscaler{*fas}}, nil
		})

		m.AgonesClient.AddReactor("update", "fleetautoscalers", func(action k8stesting.Action) (bool, runtime.Object, error) {
			fasUpdated = true
			ca := action.(k8stesting.UpdateAction)
			fas := ca.GetObject().(*autoscalingv1.FleetAutoscaler)
			assert.Equal(t, fas.Status.AbleToScale, true)
			assert.Equal(t, fas.Status.ScalingLimited, false)
			assert.Equal(t, fas.Status.CurrentReplicas, int32(50))
			assert.Equal(t, fas.Status.DesiredReplicas, int32(100))
			assert.NotNil(t, fas.Status.LastScaleTime)
			return true, fas, nil
		})

		m.AgonesClient.AddReactor("list", "fleets", func(action k8stesting.Action) (bool, runtime.Object, error) {
			return true, &agonesv1.FleetList{Items: []agonesv1.Fleet{*f}}, nil
		})

		m.AgonesClient.AddReactor("update", "fleets", func(action k8stesting.Action) (bool, runtime.Object, error) {
			fUpdated = true
			ca := action.(k8stesting.UpdateAction)
			f := ca.GetObject().(*agonesv1.Fleet)
			assert.Equal(t, f.Spec.Replicas, int32(100))
			return true, f, nil
		})

		ctx, cancel := agtesting.StartInformers(m, c.fleetSynced, c.fleetAutoscalerSynced)
		defer cancel()

		err := c.syncFleetAutoscaler(ctx, "default/fas-1")
		assert.Nil(t, err)
		assert.True(t, fUpdated, "fleet should have been updated")
		assert.True(t, fasUpdated, "fleetautoscaler should have been updated")
		agtesting.AssertEventContains(t, m.FakeRecorder.Events, "AutoScalingFleet")
		agtesting.AssertNoEvent(t, m.FakeRecorder.Events)
	})

	t.Run("scaling down, buffer policy", func(t *testing.T) {
		t.Parallel()
		c, m := newFakeController()
		fas, f := defaultFixtures()
		fas.Spec.Policy.Buffer.BufferSize = intstr.FromInt(8)

		f.Spec.Replicas = 20
		f.Status.Replicas = 20
		f.Status.AllocatedReplicas = 5
		f.Status.ReadyReplicas = 15

		fUpdated := false
		fasUpdated := false

		m.AgonesClient.AddReactor("list", "fleetautoscalers", func(action k8stesting.Action) (bool, runtime.Object, error) {
			return true, &autoscalingv1.FleetAutoscalerList{Items: []autoscalingv1.FleetAutoscaler{*fas}}, nil
		})

		m.AgonesClient.AddReactor("update", "fleetautoscalers", func(action k8stesting.Action) (bool, runtime.Object, error) {
			fasUpdated = true
			ca := action.(k8stesting.UpdateAction)
			fas := ca.GetObject().(*autoscalingv1.FleetAutoscaler)
			assert.Equal(t, fas.Status.AbleToScale, true)
			assert.Equal(t, fas.Status.ScalingLimited, false)
			assert.Equal(t, fas.Status.CurrentReplicas, int32(20))
			assert.Equal(t, fas.Status.DesiredReplicas, int32(13))
			assert.NotNil(t, fas.Status.LastScaleTime)
			return true, fas, nil
		})

		m.AgonesClient.AddReactor("list", "fleets", func(action k8stesting.Action) (bool, runtime.Object, error) {
			return true, &agonesv1.FleetList{Items: []agonesv1.Fleet{*f}}, nil
		})

		m.AgonesClient.AddReactor("update", "fleets", func(action k8stesting.Action) (bool, runtime.Object, error) {
			fUpdated = true
			ca := action.(k8stesting.UpdateAction)
			f := ca.GetObject().(*agonesv1.Fleet)
			assert.Equal(t, f.Spec.Replicas, int32(13))

			return true, f, nil
		})

		ctx, cancel := agtesting.StartInformers(m, c.fleetSynced, c.fleetAutoscalerSynced)
		defer cancel()

		err := c.syncFleetAutoscaler(ctx, "default/fas-1")
		assert.Nil(t, err)
		assert.True(t, fUpdated, "fleet should have been updated")
		assert.True(t, fasUpdated, "fleetautoscaler should have been updated")
		agtesting.AssertEventContains(t, m.FakeRecorder.Events, "AutoScalingFleet")
		agtesting.AssertNoEvent(t, m.FakeRecorder.Events)
	})

	t.Run("no scaling no update", func(t *testing.T) {
		t.Parallel()
		c, m := newFakeController()
		fas, f := defaultFixtures()

		f.Spec.Replicas = 10
		f.Status.Replicas = 10
		f.Status.ReadyReplicas = 5
		fas.Spec.Policy.Buffer.BufferSize = intstr.FromInt(5)
		fas.Status.CurrentReplicas = 10
		fas.Status.DesiredReplicas = 10

		m.AgonesClient.AddReactor("list", "fleetautoscalers", func(action k8stesting.Action) (bool, runtime.Object, error) {
			return true, &autoscalingv1.FleetAutoscalerList{Items: []autoscalingv1.FleetAutoscaler{*fas}}, nil
		})

		m.AgonesClient.AddReactor("update", "fleetautoscalers", func(action k8stesting.Action) (bool, runtime.Object, error) {
			assert.FailNow(t, "fleetautoscaler should not update")
			return false, nil, nil
		})

		m.AgonesClient.AddReactor("update", "fleets", func(action k8stesting.Action) (bool, runtime.Object, error) {
			assert.FailNow(t, "fleet should not update")
			return false, nil, nil
		})

		ctx, cancel := agtesting.StartInformers(m, c.fleetSynced, c.fleetAutoscalerSynced)
		defer cancel()

		err := c.syncFleetAutoscaler(ctx, fas.ObjectMeta.Name)
		assert.Nil(t, err)
		agtesting.AssertNoEvent(t, m.FakeRecorder.Events)
	})

	t.Run("fleet not available", func(t *testing.T) {
		t.Parallel()
		c, m := newFakeController()
		fas, _ := defaultFixtures()
		fas.Status.DesiredReplicas = 10
		fas.Status.CurrentReplicas = 5
		updated := false

		m.AgonesClient.AddReactor("list", "fleetautoscalers", func(action k8stesting.Action) (bool, runtime.Object, error) {
			return true, &autoscalingv1.FleetAutoscalerList{Items: []autoscalingv1.FleetAutoscaler{*fas}}, nil
		})

		m.AgonesClient.AddReactor("update", "fleetautoscalers", func(action k8stesting.Action) (bool, runtime.Object, error) {
			updated = true
			ca := action.(k8stesting.UpdateAction)
			fas := ca.GetObject().(*autoscalingv1.FleetAutoscaler)
			assert.Equal(t, fas.Status.CurrentReplicas, int32(0))
			assert.Equal(t, fas.Status.DesiredReplicas, int32(0))
			return true, fas, nil
		})

		ctx, cancel := agtesting.StartInformers(m, c.fleetSynced, c.fleetAutoscalerSynced)
		defer cancel()

		err := c.syncFleetAutoscaler(ctx, "default/fas-1")
		assert.Nil(t, err)
		assert.True(t, updated)

		agtesting.AssertEventContains(t, m.FakeRecorder.Events, "FailedGetFleet")
	})

	t.Run("fleet not available, error on status update", func(t *testing.T) {
		t.Parallel()
		c, m := newFakeController()
		fas, _ := defaultFixtures()
		fas.Status.DesiredReplicas = 10
		fas.Status.CurrentReplicas = 5

		m.AgonesClient.AddReactor("list", "fleetautoscalers", func(action k8stesting.Action) (bool, runtime.Object, error) {
			return true, &autoscalingv1.FleetAutoscalerList{Items: []autoscalingv1.FleetAutoscaler{*fas}}, nil
		})

		m.AgonesClient.AddReactor("update", "fleetautoscalers", func(action k8stesting.Action) (bool, runtime.Object, error) {
			ca := action.(k8stesting.UpdateAction)
			fas := ca.GetObject().(*autoscalingv1.FleetAutoscaler)
			return true, fas, errors.New("random-err")
		})

		ctx, cancel := agtesting.StartInformers(m, c.fleetSynced, c.fleetAutoscalerSynced)
		defer cancel()

		err := c.syncFleetAutoscaler(ctx, "default/fas-1")
		if assert.NotNil(t, err) {
			assert.Equal(t, "error updating status for fleetautoscaler fas-1: random-err", err.Error())
		}

		agtesting.AssertEventContains(t, m.FakeRecorder.Events, "FailedGetFleet")
	})

	t.Run("wrong policy", func(t *testing.T) {
		t.Parallel()
		c, m := newFakeController()
		fas, f := defaultFixtures()

		// wrong policy, should fail
		fas.Spec.Policy = autoscalingv1.FleetAutoscalerPolicy{
			Type: "WRONG TYPE",
		}

		m.AgonesClient.AddReactor("list", "fleetautoscalers", func(action k8stesting.Action) (bool, runtime.Object, error) {
			return true, &autoscalingv1.FleetAutoscalerList{Items: []autoscalingv1.FleetAutoscaler{*fas}}, nil
		})

		m.AgonesClient.AddReactor("list", "fleets", func(action k8stesting.Action) (bool, runtime.Object, error) {
			return true, &agonesv1.FleetList{Items: []agonesv1.Fleet{*f}}, nil
		})

		ctx, cancel := agtesting.StartInformers(m, c.fleetSynced, c.fleetAutoscalerSynced)
		defer cancel()

		err := c.syncFleetAutoscaler(ctx, "default/fas-1")
		if assert.NotNil(t, err) {
			assert.Equal(t, "error calculating autoscaling fleet: fleet-1: wrong policy type, should be one of: Buffer, Webhook", err.Error())
		}
	})

	t.Run("wrong policy, error on status update", func(t *testing.T) {
		t.Parallel()
		c, m := newFakeController()
		fas, f := defaultFixtures()
		fas.Status.DesiredReplicas = 10
		// wrong policy, should fail
		fas.Spec.Policy = autoscalingv1.FleetAutoscalerPolicy{
			Type: "WRONG TYPE",
		}

		m.AgonesClient.AddReactor("list", "fleetautoscalers", func(action k8stesting.Action) (bool, runtime.Object, error) {
			return true, &autoscalingv1.FleetAutoscalerList{Items: []autoscalingv1.FleetAutoscaler{*fas}}, nil
		})

		m.AgonesClient.AddReactor("list", "fleets", func(action k8stesting.Action) (bool, runtime.Object, error) {
			return true, &agonesv1.FleetList{Items: []agonesv1.Fleet{*f}}, nil
		})

		m.AgonesClient.AddReactor("update", "fleetautoscalers", func(action k8stesting.Action) (bool, runtime.Object, error) {
			ca := action.(k8stesting.UpdateAction)
			fas := ca.GetObject().(*autoscalingv1.FleetAutoscaler)
			return true, fas, errors.New("random-err")
		})

		ctx, cancel := agtesting.StartInformers(m, c.fleetSynced, c.fleetAutoscalerSynced)
		defer cancel()

		err := c.syncFleetAutoscaler(ctx, "default/fas-1")
		if assert.NotNil(t, err) {
			assert.Equal(t, "error updating status for fleetautoscaler fas-1: random-err", err.Error())
		}
	})

	t.Run("error on scale fleet step", func(t *testing.T) {
		t.Parallel()
		c, m := newFakeController()
		fas, f := defaultFixtures()

		m.AgonesClient.AddReactor("list", "fleetautoscalers", func(action k8stesting.Action) (bool, runtime.Object, error) {
			return true, &autoscalingv1.FleetAutoscalerList{Items: []autoscalingv1.FleetAutoscaler{*fas}}, nil
		})

		m.AgonesClient.AddReactor("list", "fleets", func(action k8stesting.Action) (bool, runtime.Object, error) {
			return true, &agonesv1.FleetList{Items: []agonesv1.Fleet{*f}}, nil
		})

		m.AgonesClient.AddReactor("update", "fleets", func(action k8stesting.Action) (bool, runtime.Object, error) {
			ca := action.(k8stesting.UpdateAction)
			return true, ca.GetObject().(*agonesv1.Fleet), errors.New("random-err")
		})

		ctx, cancel := agtesting.StartInformers(m, c.fleetSynced, c.fleetAutoscalerSynced)
		defer cancel()

		err := c.syncFleetAutoscaler(ctx, "default/fas-1")
		if assert.NotNil(t, err) {
			assert.Equal(t, "error autoscaling fleet fleet-1 to 7 replicas: error updating replicas for fleet fleet-1: random-err", err.Error())
		}

		agtesting.AssertEventContains(t, m.FakeRecorder.Events, "AutoScalingFleetError")
	})
}

func TestControllerScaleFleet(t *testing.T) {
	t.Parallel()

	t.Run("fleet that must be scaled", func(t *testing.T) {
		c, m := newFakeController()
		fas, f := defaultFixtures()
		replicas := f.Spec.Replicas + 5

		update := false

		m.AgonesClient.AddReactor("update", "fleets", func(action k8stesting.Action) (bool, runtime.Object, error) {
			update = true
			ca := action.(k8stesting.UpdateAction)
			f := ca.GetObject().(*agonesv1.Fleet)
			assert.Equal(t, replicas, f.Spec.Replicas)

			return true, f, nil
		})

		err := c.scaleFleet(context.Background(), fas, f, replicas)
		assert.Nil(t, err)
		assert.True(t, update, "Fleet should be updated")
		agtesting.AssertEventContains(t, m.FakeRecorder.Events, "ScalingFleet")
	})

	t.Run("error on updating fleet", func(t *testing.T) {
		c, m := newFakeController()
		fas, f := defaultFixtures()
		replicas := f.Spec.Replicas + 5

		m.AgonesClient.AddReactor("update", "fleets", func(action k8stesting.Action) (bool, runtime.Object, error) {
			ca := action.(k8stesting.UpdateAction)
			return true, ca.GetObject().(*agonesv1.Fleet), errors.New("random-err")
		})

		err := c.scaleFleet(context.Background(), fas, f, replicas)
		if assert.NotNil(t, err) {
			assert.Equal(t, "error updating replicas for fleet fleet-1: random-err", err.Error())
		}
		agtesting.AssertEventContains(t, m.FakeRecorder.Events, "AutoScalingFleetError")
	})

	t.Run("equal replicas, no update", func(t *testing.T) {
		c, m := newFakeController()
		fas, f := defaultFixtures()
		replicas := f.Spec.Replicas

		m.AgonesClient.AddReactor("update", "fleets", func(action k8stesting.Action) (bool, runtime.Object, error) {
			assert.FailNow(t, "fleet should not update")
			return false, nil, nil
		})

		err := c.scaleFleet(context.Background(), fas, f, replicas)
		assert.Nil(t, err)
		agtesting.AssertNoEvent(t, m.FakeRecorder.Events)
	})
}

func TestControllerUpdateStatus(t *testing.T) {
	t.Parallel()

	t.Run("must update", func(t *testing.T) {
		c, m := newFakeController()
		fas, _ := defaultFixtures()

		fasUpdated := false

		m.AgonesClient.AddReactor("update", "fleetautoscalers", func(action k8stesting.Action) (bool, runtime.Object, error) {
			fasUpdated = true
			ca := action.(k8stesting.UpdateAction)
			fas := ca.GetObject().(*autoscalingv1.FleetAutoscaler)
			assert.Equal(t, fas.Status.AbleToScale, true)
			assert.Equal(t, fas.Status.ScalingLimited, false)
			assert.Equal(t, fas.Status.CurrentReplicas, int32(10))
			assert.Equal(t, fas.Status.DesiredReplicas, int32(20))
			assert.NotNil(t, fas.Status.LastScaleTime)
			return true, fas, nil
		})

		ctx, cancel := agtesting.StartInformers(m, c.fleetAutoscalerSynced)
		defer cancel()

		err := c.updateStatus(ctx, fas, 10, 20, true, false)
		assert.Nil(t, err)
		assert.True(t, fasUpdated)
		agtesting.AssertNoEvent(t, m.FakeRecorder.Events)
	})

	t.Run("must not update", func(t *testing.T) {
		c, m := newFakeController()
		fas, _ := defaultFixtures()

		fas.Status.AbleToScale = true
		fas.Status.ScalingLimited = false
		fas.Status.CurrentReplicas = 10
		fas.Status.DesiredReplicas = 20
		fas.Status.LastScaleTime = nil

		m.AgonesClient.AddReactor("update", "fleetautoscalers", func(action k8stesting.Action) (bool, runtime.Object, error) {
			assert.FailNow(t, "should not update")
			return false, nil, nil
		})

		ctx, cancel := agtesting.StartInformers(m, c.fleetAutoscalerSynced)
		defer cancel()

		err := c.updateStatus(ctx, fas, fas.Status.CurrentReplicas, fas.Status.DesiredReplicas, false, fas.Status.ScalingLimited)
		assert.Nil(t, err)
		agtesting.AssertNoEvent(t, m.FakeRecorder.Events)
	})

	t.Run("update with error", func(t *testing.T) {
		c, m := newFakeController()
		fas, _ := defaultFixtures()

		m.AgonesClient.AddReactor("update", "fleetautoscalers", func(action k8stesting.Action) (bool, runtime.Object, error) {
			return true, nil, errors.New("random-err")
		})

		ctx, cancel := agtesting.StartInformers(m, c.fleetAutoscalerSynced)
		defer cancel()

		err := c.updateStatus(ctx, fas, fas.Status.CurrentReplicas, fas.Status.DesiredReplicas, false, fas.Status.ScalingLimited)
		if assert.NotNil(t, err) {
			assert.Equal(t, "error updating status for fleetautoscaler fas-1: random-err", err.Error())
		}
		agtesting.AssertNoEvent(t, m.FakeRecorder.Events)
	})

	t.Run("update with a scaling limit", func(t *testing.T) {
		c, m := newFakeController()
		fas, _ := defaultFixtures()

		err := c.updateStatus(context.Background(), fas, 10, 20, true, true)
		assert.Nil(t, err)
		agtesting.AssertEventContains(t, m.FakeRecorder.Events, "ScalingLimited")
	})
}

func TestControllerUpdateStatusUnableToScale(t *testing.T) {
	t.Parallel()

	t.Run("must update", func(t *testing.T) {
		c, m := newFakeController()
		fas, _ := defaultFixtures()
		fas.Status.DesiredReplicas = 10

		fasUpdated := false

		m.AgonesClient.AddReactor("update", "fleetautoscalers", func(action k8stesting.Action) (bool, runtime.Object, error) {
			fasUpdated = true
			ca := action.(k8stesting.UpdateAction)
			fas := ca.GetObject().(*autoscalingv1.FleetAutoscaler)
			assert.Equal(t, fas.Status.AbleToScale, false)
			assert.Equal(t, fas.Status.ScalingLimited, false)
			assert.Equal(t, fas.Status.CurrentReplicas, int32(0))
			assert.Equal(t, fas.Status.DesiredReplicas, int32(0))
			assert.Nil(t, fas.Status.LastScaleTime)
			return true, fas, nil
		})

		ctx, cancel := agtesting.StartInformers(m, c.fleetAutoscalerSynced)
		defer cancel()

		err := c.updateStatusUnableToScale(ctx, fas)
		assert.Nil(t, err)
		assert.True(t, fasUpdated)
		agtesting.AssertNoEvent(t, m.FakeRecorder.Events)
	})

	t.Run("update with error", func(t *testing.T) {
		c, m := newFakeController()
		fas, _ := defaultFixtures()
		fas.Status.DesiredReplicas = 10

		m.AgonesClient.AddReactor("update", "fleetautoscalers", func(action k8stesting.Action) (bool, runtime.Object, error) {
			ca := action.(k8stesting.UpdateAction)
			fas := ca.GetObject().(*autoscalingv1.FleetAutoscaler)
			return true, fas, errors.New("random-err")
		})

		ctx, cancel := agtesting.StartInformers(m, c.fleetAutoscalerSynced)
		defer cancel()

		err := c.updateStatusUnableToScale(ctx, fas)
		if assert.NotNil(t, err) {
			assert.Equal(t, "error updating status for fleetautoscaler fas-1: random-err", err.Error())
		}
		agtesting.AssertNoEvent(t, m.FakeRecorder.Events)
	})

	t.Run("must not update", func(t *testing.T) {
		c, m := newFakeController()
		fas, _ := defaultFixtures()
		fas.Status.AbleToScale = false
		fas.Status.ScalingLimited = false
		fas.Status.CurrentReplicas = 0
		fas.Status.DesiredReplicas = 0

		m.AgonesClient.AddReactor("update", "fleetautoscalers", func(action k8stesting.Action) (bool, runtime.Object, error) {
			assert.FailNow(t, "fleetautoscaler should not update")
			return false, nil, nil
		})

		ctx, cancel := agtesting.StartInformers(m, c.fleetAutoscalerSynced)
		defer cancel()

		err := c.updateStatusUnableToScale(ctx, fas)
		assert.Nil(t, err)
		agtesting.AssertNoEvent(t, m.FakeRecorder.Events)
	})
}

func TestControllerSyncFleetAutoscalerWithCustomSyncInterval(t *testing.T) {
	t.Run("create fas thread", func(t *testing.T) {
		t.Parallel()
		c, m := newFakeController()
		fas, _ := defaultFixtures()
		fasKey := fas.Namespace + "/" + fas.Name

		m.AgonesClient.AddReactor("list", "fleetautoscalers", func(action k8stesting.Action) (bool, runtime.Object, error) {
			return true, &autoscalingv1.FleetAutoscalerList{Items: []autoscalingv1.FleetAutoscaler{*fas}}, nil
		})

		ctx, cancel := agtesting.StartInformers(m, c.fleetSynced, c.fleetAutoscalerSynced)
		defer cancel()

		err := c.syncFleetAutoscalerWithCustomSyncInterval(ctx, fasKey)
		assert.Nil(t, err)
		assert.Contains(t, c.fasThreads, fasKey)
	})

	t.Run("update fas thread", func(t *testing.T) {
		t.Parallel()
		c, m := newFakeController()
		fas, _ := defaultFixtures()
		fasKey := fas.Namespace + "/" + fas.Name
		c.fasThreads[fasKey] = fasThread{
			resourceVersion: "00000000", // an older version than fas
			terminateSignal: make(chan struct{}),
		}
		go func() {
			// start a mock function for receiving the terminate signal
			<-c.fasThreads[fasKey].terminateSignal
		}()

		m.AgonesClient.AddReactor("list", "fleetautoscalers", func(action k8stesting.Action) (bool, runtime.Object, error) {
			return true, &autoscalingv1.FleetAutoscalerList{Items: []autoscalingv1.FleetAutoscaler{*fas}}, nil
		})

		ctx, cancel := agtesting.StartInformers(m, c.fleetSynced, c.fleetAutoscalerSynced)
		defer cancel()

		err := c.syncFleetAutoscalerWithCustomSyncInterval(ctx, fasKey)
		assert.Nil(t, err)
		assert.Contains(t, c.fasThreads, fasKey)
		assert.Equal(t, fas.ResourceVersion, c.fasThreads[fasKey].resourceVersion)
	})

	t.Run("delete fas thread", func(t *testing.T) {
		t.Parallel()
		c, _ := newFakeController()
		fas, _ := defaultFixtures()
		fasKey := fas.Namespace + "/" + fas.Name
		c.fasThreads[fasKey] = fasThread{
			resourceVersion: fas.ResourceVersion,
			terminateSignal: make(chan struct{}),
		}
		go func() {
			// start a mock function for receiving the terminate signal
			<-c.fasThreads[fasKey].terminateSignal
		}()
		c.removeFasThread(fas)
		assert.NotContains(t, c.fasThreads, fasKey)
	})
}

func defaultFixtures() (*autoscalingv1.FleetAutoscaler, *agonesv1.Fleet) {
	f := &agonesv1.Fleet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "fleet-1",
			Namespace: "default",
			UID:       "1234",
		},
		Spec: agonesv1.FleetSpec{
			Replicas: 8,
			Template: agonesv1.GameServerTemplateSpec{},
		},
		Status: agonesv1.FleetStatus{
			Replicas:          5,
			ReadyReplicas:     3,
			ReservedReplicas:  3,
			AllocatedReplicas: 2,
		},
	}

	fas := &autoscalingv1.FleetAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Name:            "fas-1",
			Namespace:       "default",
			ResourceVersion: "00000001",
		},
		Spec: autoscalingv1.FleetAutoscalerSpec{
			FleetName: f.ObjectMeta.Name,
			Policy: autoscalingv1.FleetAutoscalerPolicy{
				Type: autoscalingv1.BufferPolicyType,
				Buffer: &autoscalingv1.BufferPolicy{
					BufferSize:  intstr.FromInt(5),
					MaxReplicas: 100,
				},
			},
			Sync: autoscalingv1.FleetAutoscalerSync{
				Type: autoscalingv1.FixedIntervalSyncType,
				FixedInterval: &autoscalingv1.FixedIntervalSync{
					Seconds: 30,
				},
			},
		},
	}

	return fas, f
}

func defaultWebhookFixtures() (*autoscalingv1.FleetAutoscaler, *agonesv1.Fleet) {
	fas, f := defaultFixtures()
	fas.Spec.Policy.Type = autoscalingv1.WebhookPolicyType
	fas.Spec.Policy.Buffer = nil
	url := "/autoscaler"
	fas.Spec.Policy.Webhook = &autoscalingv1.WebhookPolicy{
		Service: &admregv1.ServiceReference{
			Name: "fleetautoscaler-service",
			Path: &url,
		},
	}

	return fas, f
}

// newFakeController returns a controller, backed by the fake Clientset
func newFakeController() (*Controller, agtesting.Mocks) {
	m := agtesting.NewMocks()
	wh := webhooks.NewWebHook(http.NewServeMux())
	c := NewController(wh, healthcheck.NewHandler(), m.KubeClient, m.ExtClient, m.AgonesClient, m.AgonesInformerFactory)
	c.recorder = m.FakeRecorder
	return c, m
}

func newAdmissionReview(fas autoscalingv1.FleetAutoscaler) (admissionv1.AdmissionReview, error) {
	raw, err := json.Marshal(fas)
	if err != nil {
		return admissionv1.AdmissionReview{}, err
	}
	review := admissionv1.AdmissionReview{
		Request: &admissionv1.AdmissionRequest{
			Kind:      gvk,
			Operation: admissionv1.Create,
			Object: runtime.RawExtension{
				Raw: raw,
			},
			Namespace: "default",
		},
		Response: &admissionv1.AdmissionResponse{Allowed: true},
	}
	return review, err
}

func newInvalidAdmissionReview() (admissionv1.AdmissionReview, error) {
	raw, err := json.Marshal([]byte(`1`))
	if err != nil {
		return admissionv1.AdmissionReview{}, err
	}
	review := admissionv1.AdmissionReview{
		Request: &admissionv1.AdmissionRequest{
			Kind:      gvk,
			Operation: admissionv1.Create,
			Object: runtime.RawExtension{
				Raw: raw,
			},
			Namespace: "default",
		},
		Response: &admissionv1.AdmissionResponse{Allowed: true},
	}
	return review, nil
}
