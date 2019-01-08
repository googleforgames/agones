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

package e2e

import (
	"fmt"
	"math/rand"
	"testing"

	"agones.dev/agones/pkg/apis/stable/v1alpha1"
	e2e "agones.dev/agones/test/e2e/framework"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	admregv1b "k8s.io/api/admissionregistration/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func TestAutoscalerBasicFunctions(t *testing.T) {
	t.Parallel()

	alpha1 := framework.AgonesClient.StableV1alpha1()
	fleets := alpha1.Fleets(defaultNs)
	flt, err := fleets.Create(defaultFleet())
	if assert.Nil(t, err) {
		defer fleets.Delete(flt.ObjectMeta.Name, nil) // nolint:errcheck
	}

	err = framework.WaitForFleetCondition(flt, e2e.FleetReadyCount(flt.Spec.Replicas))
	assert.Nil(t, err, "fleet not ready")

	fleetautoscalers := alpha1.FleetAutoscalers(defaultNs)
	fas, err := fleetautoscalers.Create(defaultFleetAutoscaler(flt))
	if assert.Nil(t, err) {
		defer fleetautoscalers.Delete(fas.ObjectMeta.Name, nil) // nolint:errcheck
	} else {
		// if we could not create the autoscaler, their is no point going further
		logrus.Error("Failed creating autoscaler, aborting TestAutoscalerBasicFunctions")
		return
	}

	// the fleet autoscaler should scale the fleet up now up to BufferSize
	bufferSize := int32(fas.Spec.Policy.Buffer.BufferSize.IntValue())
	err = framework.WaitForFleetCondition(flt, e2e.FleetReadyCount(bufferSize))
	assert.Nil(t, err, "fleet did not sync with autoscaler")

	// patch the autoscaler to increase MinReplicas and watch the fleet scale up
	fas, err = patchFleetAutoscaler(fas, intstr.FromInt(int(bufferSize)), bufferSize+2, fas.Spec.Policy.Buffer.MaxReplicas)
	assert.Nil(t, err, "could not patch fleetautoscaler")

	bufferSize = int32(fas.Spec.Policy.Buffer.BufferSize.IntValue())
	err = framework.WaitForFleetCondition(flt, e2e.FleetReadyCount(bufferSize))
	assert.Nil(t, err, "fleet did not sync with autoscaler")

	// patch the autoscaler to remove MinReplicas and watch the fleet scale down
	fas, err = patchFleetAutoscaler(fas, intstr.FromInt(int(bufferSize)), 0, fas.Spec.Policy.Buffer.MaxReplicas)
	assert.Nil(t, err, "could not patch fleetautoscaler")

	bufferSize = int32(fas.Spec.Policy.Buffer.BufferSize.IntValue())
	err = framework.WaitForFleetCondition(flt, e2e.FleetReadyCount(bufferSize))
	assert.Nil(t, err, "fleet did not sync with autoscaler")

	// do an allocation and watch the fleet scale up
	fa := getAllocation(flt)
	fa, err = alpha1.FleetAllocations(defaultNs).Create(fa)
	assert.Nil(t, err)
	assert.Equal(t, v1alpha1.GameServerStateAllocated, fa.Status.GameServer.Status.State)
	err = framework.WaitForFleetCondition(flt, func(fleet *v1alpha1.Fleet) bool {
		return fleet.Status.AllocatedReplicas == 1
	})
	assert.Nil(t, err)

	err = framework.WaitForFleetCondition(flt, e2e.FleetReadyCount(bufferSize))
	assert.Nil(t, err, "fleet did not sync with autoscaler")

	// patch autoscaler to switch to relative buffer size and check if the fleet adjusts
	_, err = patchFleetAutoscaler(fas, intstr.FromString("10%"), 1, fas.Spec.Policy.Buffer.MaxReplicas)
	assert.Nil(t, err, "could not patch fleetautoscaler")

	//10% with only one allocated GS means only one ready server
	err = framework.WaitForFleetCondition(flt, e2e.FleetReadyCount(1))
	assert.Nil(t, err, "fleet did not sync with autoscaler")

	// delete the allocated GameServer and watch the fleet scale down
	gp := int64(1)
	err = alpha1.GameServers(defaultNs).Delete(fa.Status.GameServer.ObjectMeta.Name, &metav1.DeleteOptions{GracePeriodSeconds: &gp})
	assert.Nil(t, err)
	err = framework.WaitForFleetCondition(flt, func(fleet *v1alpha1.Fleet) bool {
		return fleet.Status.AllocatedReplicas == 0 &&
			fleet.Status.ReadyReplicas == 1 &&
			fleet.Status.Replicas == 1
	})
	assert.Nil(t, err)
}

// TestAutoscalerStressCreate creates many fleetautoscalers with random values
// to check if the creation validation works as expected and if the fleet scales
// to the expected number of replicas (when the creation is valid)
func TestAutoscalerStressCreate(t *testing.T) {
	t.Parallel()

	alpha1 := framework.AgonesClient.StableV1alpha1()
	fleets := alpha1.Fleets(defaultNs)
	flt, err := fleets.Create(defaultFleet())
	if assert.Nil(t, err) {
		defer fleets.Delete(flt.ObjectMeta.Name, nil) // nolint:errcheck
	}

	err = framework.WaitForFleetCondition(flt, e2e.FleetReadyCount(flt.Spec.Replicas))
	assert.Nil(t, err, "fleet not ready")

	r := rand.New(rand.NewSource(1783))

	fleetautoscalers := alpha1.FleetAutoscalers(defaultNs)

	for i := 0; i < 5; i++ {
		fas := defaultFleetAutoscaler(flt)
		bufferSize := r.Int31n(5)
		minReplicas := r.Int31n(5)
		maxReplicas := r.Int31n(8)
		fas.Spec.Policy.Buffer.BufferSize = intstr.FromInt(int(bufferSize))
		fas.Spec.Policy.Buffer.MinReplicas = minReplicas
		fas.Spec.Policy.Buffer.MaxReplicas = maxReplicas

		valid := bufferSize > 0 &&
			fas.Spec.Policy.Buffer.MaxReplicas > 0 &&
			fas.Spec.Policy.Buffer.MaxReplicas >= bufferSize &&
			fas.Spec.Policy.Buffer.MinReplicas <= fas.Spec.Policy.Buffer.MaxReplicas &&
			(fas.Spec.Policy.Buffer.MinReplicas == 0 || fas.Spec.Policy.Buffer.MinReplicas >= bufferSize)

		// create a closure to have defered delete func called on each loop iteration.
		func() {
			fas, err := fleetautoscalers.Create(fas)
			if err == nil {
				defer fleetautoscalers.Delete(fas.ObjectMeta.Name, nil) // nolint:errcheck
				assert.True(t, valid,
					fmt.Sprintf("FleetAutoscaler created even if the parameters are NOT valid: %d %d %d",
						bufferSize,
						fas.Spec.Policy.Buffer.MinReplicas,
						fas.Spec.Policy.Buffer.MaxReplicas))

				expectedReplicas := bufferSize
				if expectedReplicas < fas.Spec.Policy.Buffer.MinReplicas {
					expectedReplicas = fas.Spec.Policy.Buffer.MinReplicas
				}
				if expectedReplicas > fas.Spec.Policy.Buffer.MaxReplicas {
					expectedReplicas = fas.Spec.Policy.Buffer.MaxReplicas
				}
				// the fleet autoscaler should scale the fleet now to expectedReplicas
				err = framework.WaitForFleetCondition(flt, e2e.FleetReadyCount(expectedReplicas))
				assert.Nil(t, err, fmt.Sprintf("fleet did not sync with autoscaler, expected %d ready replicas", expectedReplicas))
			} else {
				assert.False(t, valid,
					fmt.Sprintf("FleetAutoscaler NOT created even if the parameters are valid: %d %d %d (%s)",
						bufferSize,
						minReplicas,
						maxReplicas, err))
			}
		}()
	}
}

// scaleFleet creates a patch to apply to a Fleet.
// easier for testing, as it removes object generational issues.
func patchFleetAutoscaler(fas *v1alpha1.FleetAutoscaler, bufferSize intstr.IntOrString, minReplicas int32, maxReplicas int32) (*v1alpha1.FleetAutoscaler, error) {
	var bufferSizeFmt string
	if bufferSize.Type == intstr.Int {
		bufferSizeFmt = fmt.Sprintf("%d", bufferSize.IntValue())
	} else {
		bufferSizeFmt = fmt.Sprintf(`"%s"`, bufferSize.String())
	}

	patch := fmt.Sprintf(
		`[{ "op": "replace", "path": "/spec/policy/buffer/bufferSize", "value": %s },`+
			`{ "op": "replace", "path": "/spec/policy/buffer/minReplicas", "value": %d },`+
			`{ "op": "replace", "path": "/spec/policy/buffer/maxReplicas", "value": %d }]`,
		bufferSizeFmt, minReplicas, maxReplicas)
	logrus.
		WithField("fleetautoscaler", fas.ObjectMeta.Name).
		WithField("bufferSize", bufferSize.String()).
		WithField("minReplicas", minReplicas).
		WithField("maxReplicas", maxReplicas).
		WithField("patch", patch).
		Info("Patching fleetautoscaler")

	return framework.AgonesClient.StableV1alpha1().FleetAutoscalers(defaultNs).Patch(fas.ObjectMeta.Name, types.JSONPatchType, []byte(patch))
}

// defaultFleetAutoscaler returns a default fleet autoscaler configuration for a given fleet
func defaultFleetAutoscaler(f *v1alpha1.Fleet) *v1alpha1.FleetAutoscaler {
	return &v1alpha1.FleetAutoscaler{
		ObjectMeta: metav1.ObjectMeta{Name: f.ObjectMeta.Name + "-autoscaler", Namespace: defaultNs},
		Spec: v1alpha1.FleetAutoscalerSpec{
			FleetName: f.ObjectMeta.Name,
			Policy: v1alpha1.FleetAutoscalerPolicy{
				Type: v1alpha1.BufferPolicyType,
				Buffer: &v1alpha1.BufferPolicy{
					BufferSize:  intstr.FromInt(3),
					MaxReplicas: 10,
				},
			},
		},
	}
}

func getAllocation(f *v1alpha1.Fleet) *v1alpha1.FleetAllocation {
	// get an allocation
	return &v1alpha1.FleetAllocation{
		ObjectMeta: metav1.ObjectMeta{GenerateName: "allocation-", Namespace: defaultNs},
		Spec: v1alpha1.FleetAllocationSpec{
			FleetName: f.ObjectMeta.Name,
		},
	}
}

//Test fleetautoscaler with webhook policy type
// scaling from Replicas equals to 1 to 2
func TestAutoscalerWebhook(t *testing.T) {
	t.Parallel()
	pod, svc := defaultAutoscalerWebhook()
	pod, err := framework.KubeClient.CoreV1().Pods(defaultNs).Create(pod)
	if assert.Nil(t, err) {
		defer framework.KubeClient.CoreV1().Pods(defaultNs).Delete(pod.ObjectMeta.Name, nil) // nolint:errcheck
	} else {
		// if we could not create the autoscaler, their is no point going further
		assert.FailNow(t, "Failed creating autoscaler, aborting TestAutoscalerBasicFunctions")
	}
	svc, err = framework.KubeClient.CoreV1().Services(defaultNs).Create(svc)
	if assert.Nil(t, err) {
		defer framework.KubeClient.CoreV1().Services(defaultNs).Delete(svc.ObjectMeta.Name, nil) // nolint:errcheck
	} else {
		// if we could not create the autoscaler, their is no point going further
		assert.FailNow(t, "Failed creating autoscaler, aborting TestAutoscalerBasicFunctions")
	}

	alpha1 := framework.AgonesClient.StableV1alpha1()
	fleets := alpha1.Fleets(defaultNs)
	flt := defaultFleet()
	initialReplicasCount := int32(1)
	flt.Spec.Replicas = initialReplicasCount
	flt, err = fleets.Create(flt)
	if assert.Nil(t, err) {
		defer fleets.Delete(flt.ObjectMeta.Name, nil) // nolint:errcheck
	}

	err = framework.WaitForFleetCondition(flt, e2e.FleetReadyCount(flt.Spec.Replicas))
	assert.Nil(t, err, "fleet not ready")

	fleetautoscalers := alpha1.FleetAutoscalers(defaultNs)
	fas := defaultFleetAutoscaler(flt)
	fas.Spec.Policy.Type = v1alpha1.WebhookPolicyType
	fas.Spec.Policy.Buffer = nil
	path := "scale"
	fas.Spec.Policy.Webhook = &v1alpha1.WebhookPolicy{
		Service: &admregv1b.ServiceReference{
			Name:      svc.ObjectMeta.Name,
			Namespace: defaultNs,
			Path:      &path,
		},
	}
	fas, err = fleetautoscalers.Create(fas)
	if assert.Nil(t, err) {
		defer fleetautoscalers.Delete(fas.ObjectMeta.Name, nil) // nolint:errcheck
	} else {
		// if we could not create the autoscaler, their is no point going further
		assert.FailNow(t, "Failed creating autoscaler, aborting TestAutoscalerBasicFunctions")
	}
	fa := getAllocation(flt)
	fa, err = alpha1.FleetAllocations(defaultNs).Create(fa)
	assert.Nil(t, err)
	assert.Equal(t, v1alpha1.GameServerStateAllocated, fa.Status.GameServer.Status.State)
	err = framework.WaitForFleetCondition(flt, func(fleet *v1alpha1.Fleet) bool {
		return fleet.Status.AllocatedReplicas == 1
	})
	assert.Nil(t, err)

	err = framework.WaitForFleetCondition(flt, func(fleet *v1alpha1.Fleet) bool {
		return fleet.Status.Replicas > initialReplicasCount
	})
	assert.Nil(t, err)
}

func defaultAutoscalerWebhook() (*corev1.Pod, *corev1.Service) {
	l := make(map[string]string)
	l["app"] = "autoscaler-webhook"
	pod := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{
		GenerateName: "auto-webhook",
		Namespace:    defaultNs,
		Labels:       l,
	},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{{Name: "webhook",
				Image:           "gcr.io/agones-images/autoscaler-webhook:0.1",
				ImagePullPolicy: corev1.PullAlways,
				Ports: []corev1.ContainerPort{{
					ContainerPort: 8000,
					Name:          "autoscaler",
				}},
			}},
		},
	}
	m := make(map[string]string)
	m["app"] = "autoscaler-webhook"
	service := &corev1.Service{ObjectMeta: metav1.ObjectMeta{GenerateName: "auto-webhook", Namespace: defaultNs},
		Spec: corev1.ServiceSpec{
			Selector: m,
			Ports: []corev1.ServicePort{{
				Name:       "newport",
				Port:       8000,
				TargetPort: intstr.IntOrString{StrVal: "autoscaler"},
			}},
		},
	}

	return pod, service
}
