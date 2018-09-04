// Copyright 2018 Google Inc. All Rights Reserved.
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
	"encoding/json"
	"fmt"
	"testing"

	"agones.dev/agones/pkg/apis/stable/v1alpha1"
	agtesting "agones.dev/agones/pkg/testing"
	"agones.dev/agones/pkg/util/webhooks"
	"github.com/heptiolabs/healthcheck"
	"github.com/stretchr/testify/assert"
	admv1beta1 "k8s.io/api/admission/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	k8stesting "k8s.io/client-go/testing"
)

var (
	gvk = metav1.GroupVersionKind(v1alpha1.SchemeGroupVersion.WithKind("FleetAutoscaler"))
)

func TestControllerCreationMutationHandler(t *testing.T) {
	t.Parallel()

	t.Run("fleet scaler has a fleet", func(t *testing.T) {
		c, m := newFakeController()
		fas, f := defaultFixtures()
		m.AgonesClient.AddReactor("list", "fleets", func(action k8stesting.Action) (bool, runtime.Object, error) {
			return true, &v1alpha1.FleetList{Items: []v1alpha1.Fleet{*f}}, nil
		})

		_, cancel := agtesting.StartInformers(m)
		defer cancel()

		review, err := newAdmissionReview(*fas)
		assert.Nil(t, err)

		result, err := c.creationMutationHandler(review)
		assert.Nil(t, err)
		assert.True(t, result.Response.Allowed, fmt.Sprintf("%#v", result.Response))
		assert.Equal(t, admv1beta1.PatchTypeJSONPatch, *result.Response.PatchType)
		assert.Contains(t, string(result.Response.Patch), "/metadata/ownerReferences")
	})

	t.Run("fleet does not exist", func(t *testing.T) {
		c, _ := newFakeController()
		fas, _ := defaultFixtures()

		review, err := newAdmissionReview(*fas)
		assert.Nil(t, err)

		result, err := c.creationMutationHandler(review)
		assert.Nil(t, err)
		assert.False(t, result.Response.Allowed)
		assert.Equal(t, "fleetName", result.Response.Result.Details.Causes[0].Field)
	})
}

func TestControllerMutationValidationHandler(t *testing.T) {
	t.Parallel()
	c, _ := newFakeController()

	t.Run("same fleetName", func(t *testing.T) {
		fas, _ := defaultFixtures()

		review, err := newAdmissionReview(*fas)
		assert.Nil(t, err)
		review.Request.OldObject = *review.Request.Object.DeepCopy()

		result, err := c.mutationValidationHandler(review)
		assert.Nil(t, err)
		assert.True(t, result.Response.Allowed, fmt.Sprintf("%#v", result.Response))
	})

	t.Run("different fleetname", func(t *testing.T) {
		fas, _ := defaultFixtures()

		review, err := newAdmissionReview(*fas)
		assert.Nil(t, err)
		oldObject := fas.DeepCopy()
		oldObject.Spec.FleetName = "changed"

		json, err := json.Marshal(oldObject)
		assert.Nil(t, err)
		review.Request.OldObject = runtime.RawExtension{Raw: json}

		result, err := c.mutationValidationHandler(review)
		assert.Nil(t, err)
		assert.False(t, result.Response.Allowed)
		assert.Equal(t, metav1.StatusReasonInvalid, result.Response.Result.Reason)
		assert.NotNil(t, result.Response.Result.Details)
	})
}

// nolint:dupl
func TestControllerSyncFleetAutoscaler(t *testing.T) {
	t.Parallel()

	t.Run("scaling up", func(t *testing.T) {
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
			return true, &v1alpha1.FleetAutoscalerList{Items: []v1alpha1.FleetAutoscaler{*fas}}, nil
		})

		m.AgonesClient.AddReactor("update", "fleetautoscalers", func(action k8stesting.Action) (bool, runtime.Object, error) {
			fasUpdated = true
			ca := action.(k8stesting.UpdateAction)
			fas := ca.GetObject().(*v1alpha1.FleetAutoscaler)
			assert.Equal(t, fas.Status.AbleToScale, true)
			assert.Equal(t, fas.Status.ScalingLimited, false)
			assert.Equal(t, fas.Status.CurrentReplicas, int32(5))
			assert.Equal(t, fas.Status.DesiredReplicas, int32(12))
			assert.NotNil(t, fas.Status.LastScaleTime)
			return true, fas, nil
		})

		m.AgonesClient.AddReactor("list", "fleets", func(action k8stesting.Action) (bool, runtime.Object, error) {
			return true, &v1alpha1.FleetList{Items: []v1alpha1.Fleet{*f}}, nil
		})

		m.AgonesClient.AddReactor("update", "fleets", func(action k8stesting.Action) (bool, runtime.Object, error) {
			fUpdated = true
			ca := action.(k8stesting.UpdateAction)
			f := ca.GetObject().(*v1alpha1.Fleet)
			assert.Equal(t, f.Spec.Replicas, int32(12))
			return true, f, nil
		})

		_, cancel := agtesting.StartInformers(m, c.fleetAutoscalerSynced)
		defer cancel()

		err := c.syncFleetAutoscaler("default/fas-1")
		assert.Nil(t, err)
		assert.True(t, fUpdated, "fleet should have been updated")
		assert.True(t, fasUpdated, "fleetautoscaler should have been updated")
		agtesting.AssertEventContains(t, m.FakeRecorder.Events, "AutoScalingFleet")
	})

	t.Run("scaling down", func(t *testing.T) {
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
			return true, &v1alpha1.FleetAutoscalerList{Items: []v1alpha1.FleetAutoscaler{*fas}}, nil
		})

		m.AgonesClient.AddReactor("update", "fleetautoscalers", func(action k8stesting.Action) (bool, runtime.Object, error) {
			fasUpdated = true
			ca := action.(k8stesting.UpdateAction)
			fas := ca.GetObject().(*v1alpha1.FleetAutoscaler)
			assert.Equal(t, fas.Status.AbleToScale, true)
			assert.Equal(t, fas.Status.ScalingLimited, false)
			assert.Equal(t, fas.Status.CurrentReplicas, int32(20))
			assert.Equal(t, fas.Status.DesiredReplicas, int32(13))
			assert.NotNil(t, fas.Status.LastScaleTime)
			return true, fas, nil
		})

		m.AgonesClient.AddReactor("list", "fleets", func(action k8stesting.Action) (bool, runtime.Object, error) {
			return true, &v1alpha1.FleetList{Items: []v1alpha1.Fleet{*f}}, nil
		})

		m.AgonesClient.AddReactor("update", "fleets", func(action k8stesting.Action) (bool, runtime.Object, error) {
			fUpdated = true
			ca := action.(k8stesting.UpdateAction)
			f := ca.GetObject().(*v1alpha1.Fleet)
			assert.Equal(t, f.Spec.Replicas, int32(13))

			return true, f, nil
		})

		_, cancel := agtesting.StartInformers(m, c.fleetAutoscalerSynced)
		defer cancel()

		err := c.syncFleetAutoscaler("default/fas-1")
		assert.Nil(t, err)
		assert.True(t, fUpdated, "fleet should have been updated")
		assert.True(t, fasUpdated, "fleetautoscaler should have been updated")
		agtesting.AssertEventContains(t, m.FakeRecorder.Events, "AutoScalingFleet")
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
			return true, &v1alpha1.FleetAutoscalerList{Items: []v1alpha1.FleetAutoscaler{*fas}}, nil
		})

		m.AgonesClient.AddReactor("update", "fleetautoscalers", func(action k8stesting.Action) (bool, runtime.Object, error) {
			assert.FailNow(t, "fleetautoscaler should not update")
			return false, nil, nil
		})

		m.AgonesClient.AddReactor("update", "fleets", func(action k8stesting.Action) (bool, runtime.Object, error) {
			assert.FailNow(t, "fleet should not update")
			return false, nil, nil
		})

		_, cancel := agtesting.StartInformers(m, c.fleetAutoscalerSynced)
		defer cancel()

		err := c.syncFleetAutoscaler(fas.ObjectMeta.Name)
		assert.Nil(t, err)
		agtesting.AssertNoEvent(t, m.FakeRecorder.Events)
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
			f := ca.GetObject().(*v1alpha1.Fleet)
			assert.Equal(t, replicas, f.Spec.Replicas)

			return true, f, nil
		})

		err := c.scaleFleet(fas, f, replicas)
		assert.Nil(t, err)
		assert.True(t, update, "Fleet should be updated")
		agtesting.AssertEventContains(t, m.FakeRecorder.Events, "ScalingFleet")
	})

	t.Run("noop", func(t *testing.T) {
		c, m := newFakeController()
		fas, f := defaultFixtures()
		replicas := f.Spec.Replicas

		m.AgonesClient.AddReactor("update", "fleets", func(action k8stesting.Action) (bool, runtime.Object, error) {
			assert.FailNow(t, "fleet should not update")
			return false, nil, nil
		})

		err := c.scaleFleet(fas, f, replicas)
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
			fas := ca.GetObject().(*v1alpha1.FleetAutoscaler)
			assert.Equal(t, fas.Status.AbleToScale, true)
			assert.Equal(t, fas.Status.ScalingLimited, false)
			assert.Equal(t, fas.Status.CurrentReplicas, int32(10))
			assert.Equal(t, fas.Status.DesiredReplicas, int32(20))
			assert.NotNil(t, fas.Status.LastScaleTime)
			return true, fas, nil
		})

		_, cancel := agtesting.StartInformers(m, c.fleetAutoscalerSynced)
		defer cancel()

		err := c.updateStatus(fas, 10, 20, true, false)
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

		_, cancel := agtesting.StartInformers(m, c.fleetAutoscalerSynced)
		defer cancel()

		err := c.updateStatus(fas, fas.Status.CurrentReplicas, fas.Status.DesiredReplicas, false, fas.Status.ScalingLimited)
		assert.Nil(t, err)
		agtesting.AssertNoEvent(t, m.FakeRecorder.Events)
	})
}

func TestControllerUpdateStatusUnableToScale(t *testing.T) {
	t.Parallel()

	t.Run("must update", func(t *testing.T) {
		c, m := newFakeController()
		fas, _ := defaultFixtures()
		fas.Status.AbleToScale = true

		fasUpdated := false

		m.AgonesClient.AddReactor("update", "fleetautoscalers", func(action k8stesting.Action) (bool, runtime.Object, error) {
			fasUpdated = true
			ca := action.(k8stesting.UpdateAction)
			fas := ca.GetObject().(*v1alpha1.FleetAutoscaler)
			assert.Equal(t, fas.Status.AbleToScale, false)
			assert.Equal(t, fas.Status.ScalingLimited, false)
			assert.Equal(t, fas.Status.CurrentReplicas, int32(0))
			assert.Equal(t, fas.Status.DesiredReplicas, int32(0))
			assert.Nil(t, fas.Status.LastScaleTime)
			return true, fas, nil
		})

		_, cancel := agtesting.StartInformers(m, c.fleetAutoscalerSynced)
		defer cancel()

		err := c.updateStatusUnableToScale(fas)
		assert.Nil(t, err)
		assert.True(t, fasUpdated)
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

		_, cancel := agtesting.StartInformers(m, c.fleetAutoscalerSynced)
		defer cancel()

		err := c.updateStatusUnableToScale(fas)
		assert.Nil(t, err)
		agtesting.AssertNoEvent(t, m.FakeRecorder.Events)
	})
}

func defaultFixtures() (*v1alpha1.FleetAutoscaler, *v1alpha1.Fleet) {
	f := &v1alpha1.Fleet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "fleet-1",
			Namespace: "default",
			UID:       "1234",
		},
		Spec: v1alpha1.FleetSpec{
			Replicas: 5,
			Template: v1alpha1.GameServerTemplateSpec{},
		},
		Status: v1alpha1.FleetStatus{
			Replicas:          5,
			ReadyReplicas:     3,
			AllocatedReplicas: 2,
		},
	}

	fas := &v1alpha1.FleetAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "fas-1",
			Namespace: "default",
		},
		Spec: v1alpha1.FleetAutoscalerSpec{
			FleetName: f.ObjectMeta.Name,
			Policy: v1alpha1.FleetAutoscalerPolicy{
				Type: v1alpha1.BufferPolicyType,
				Buffer: &v1alpha1.BufferPolicy{
					BufferSize:  intstr.FromInt(5),
					MaxReplicas: 100,
				},
			},
		},
	}

	return fas, f
}

// newFakeController returns a controller, backed by the fake Clientset
func newFakeController() (*Controller, agtesting.Mocks) {
	m := agtesting.NewMocks()
	wh := webhooks.NewWebHook("", "")
	c := NewController(wh, healthcheck.NewHandler(), m.KubeClient, m.ExtClient, m.AgonesClient, m.AgonesInformerFactory)
	c.recorder = m.FakeRecorder
	return c, m
}

func newAdmissionReview(fas v1alpha1.FleetAutoscaler) (admv1beta1.AdmissionReview, error) {
	raw, err := json.Marshal(fas)
	if err != nil {
		return admv1beta1.AdmissionReview{}, err
	}
	review := admv1beta1.AdmissionReview{
		Request: &admv1beta1.AdmissionRequest{
			Kind:      gvk,
			Operation: admv1beta1.Create,
			Object: runtime.RawExtension{
				Raw: raw,
			},
			Namespace: "default",
		},
		Response: &admv1beta1.AdmissionResponse{Allowed: true},
	}
	return review, err
}
