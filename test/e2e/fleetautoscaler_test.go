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

package e2e

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"testing"
	"time"

	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	autoscalingv1 "agones.dev/agones/pkg/apis/autoscaling/v1"
	"agones.dev/agones/pkg/util/runtime"
	e2e "agones.dev/agones/test/e2e/framework"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	admregv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/wait"
)

var deletePropagationForeground = metav1.DeletePropagationForeground

var waitForDeletion = metav1.DeleteOptions{
	PropagationPolicy: &deletePropagationForeground,
}

func TestAutoscalerBasicFunctions(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	stable := framework.AgonesClient.AgonesV1()
	fleets := stable.Fleets(framework.Namespace)
	flt, err := fleets.Create(ctx, defaultFleet(framework.Namespace), metav1.CreateOptions{})
	if assert.Nil(t, err) {
		defer fleets.Delete(ctx, flt.ObjectMeta.Name, metav1.DeleteOptions{}) // nolint:errcheck
	}

	framework.AssertFleetCondition(t, flt, e2e.FleetReadyCount(flt.Spec.Replicas))

	fleetautoscalers := framework.AgonesClient.AutoscalingV1().FleetAutoscalers(framework.Namespace)
	defaultFas := defaultFleetAutoscaler(flt, framework.Namespace)
	fas, err := fleetautoscalers.Create(ctx, defaultFas, metav1.CreateOptions{})
	require.NoError(t, err)
	defer fleetautoscalers.Delete(ctx, fas.ObjectMeta.Name, metav1.DeleteOptions{}) // nolint:errcheck

	// If the CustomFasSyncInterval feature flag is enabled, the value of fas.spec.sync are equal
	if runtime.FeatureEnabled(runtime.FeatureCustomFasSyncInterval) {
		require.Equal(t, defaultFas.Spec.Sync.FixedInterval.Seconds, fas.Spec.Sync.FixedInterval.Seconds)
	}

	// the fleet autoscaler should scale the fleet up now up to BufferSize
	bufferSize := int32(fas.Spec.Policy.Buffer.BufferSize.IntValue())
	framework.AssertFleetCondition(t, flt, e2e.FleetReadyCount(bufferSize))

	// patch the autoscaler to increase MinReplicas and watch the fleet scale up
	fas, err = patchFleetAutoscaler(ctx, fas, intstr.FromInt(int(bufferSize)), bufferSize+2, fas.Spec.Policy.Buffer.MaxReplicas)
	assert.Nil(t, err, "could not patch fleetautoscaler")

	// min replicas is now higher than buffer size, will scale to that level
	framework.AssertFleetCondition(t, flt, e2e.FleetReadyCount(fas.Spec.Policy.Buffer.MinReplicas))

	// patch the autoscaler to remove MinReplicas and watch the fleet scale down to bufferSize
	fas, err = patchFleetAutoscaler(ctx, fas, intstr.FromInt(int(bufferSize)), 0, fas.Spec.Policy.Buffer.MaxReplicas)
	assert.Nil(t, err, "could not patch fleetautoscaler")

	bufferSize = int32(fas.Spec.Policy.Buffer.BufferSize.IntValue())
	framework.AssertFleetCondition(t, flt, e2e.FleetReadyCount(bufferSize))

	// do an allocation and watch the fleet scale up
	gsa := framework.CreateAndApplyAllocation(t, flt)
	framework.AssertFleetCondition(t, flt, func(log *logrus.Entry, fleet *agonesv1.Fleet) bool {
		return fleet.Status.AllocatedReplicas == 1
	})

	framework.AssertFleetCondition(t, flt, e2e.FleetReadyCount(bufferSize))

	// patch autoscaler to switch to relative buffer size and check if the fleet adjusts
	_, err = patchFleetAutoscaler(ctx, fas, intstr.FromString("10%"), 1, fas.Spec.Policy.Buffer.MaxReplicas)
	require.NoError(t, err, "could not patch fleetautoscaler")

	// 10% with only one allocated GS means only one ready server
	framework.AssertFleetCondition(t, flt, e2e.FleetReadyCount(1))

	// get the Status of the fleetautoscaler
	fas, err = framework.AgonesClient.AutoscalingV1().FleetAutoscalers(fas.ObjectMeta.Namespace).Get(ctx, fas.Name, metav1.GetOptions{})
	require.NoError(t, err, "could not get fleetautoscaler")
	require.True(t, fas.Status.AbleToScale, "Could not get AbleToScale status")

	// check that we are able to scale
	framework.WaitForFleetAutoScalerCondition(t, fas, func(log *logrus.Entry, fas *autoscalingv1.FleetAutoscaler) bool {
		return !fas.Status.ScalingLimited
	})

	// patch autoscaler to a maxReplicas count equal to current replicas count
	_, err = patchFleetAutoscaler(ctx, fas, intstr.FromInt(1), 1, 1)
	require.NoError(t, err, "could not patch fleetautoscaler")

	// check that we are not able to scale
	framework.WaitForFleetAutoScalerCondition(t, fas, func(log *logrus.Entry, fas *autoscalingv1.FleetAutoscaler) bool {
		return fas.Status.ScalingLimited
	})

	// delete the allocated GameServer and watch the fleet scale down
	gp := int64(1)
	err = stable.GameServers(framework.Namespace).Delete(ctx, gsa.Status.GameServerName, metav1.DeleteOptions{GracePeriodSeconds: &gp})
	require.NoError(t, err)
	framework.AssertFleetCondition(t, flt, func(log *logrus.Entry, fleet *agonesv1.Fleet) bool {
		return fleet.Status.AllocatedReplicas == 0 &&
			fleet.Status.ReadyReplicas == 1 &&
			fleet.Status.Replicas == 1
	})
}

func TestFleetAutoscalerDefaultSyncInterval(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	stable := framework.AgonesClient.AgonesV1()
	fleets := stable.Fleets(framework.Namespace)
	flt, err := fleets.Create(ctx, defaultFleet(framework.Namespace), metav1.CreateOptions{})
	if assert.Nil(t, err) {
		defer fleets.Delete(ctx, flt.ObjectMeta.Name, metav1.DeleteOptions{}) // nolint:errcheck
	}

	framework.AssertFleetCondition(t, flt, e2e.FleetReadyCount(flt.Spec.Replicas))

	fleetautoscalers := framework.AgonesClient.AutoscalingV1().FleetAutoscalers(framework.Namespace)
	dummyFleetName := "dummy-fleet"
	defaultFas := &autoscalingv1.FleetAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Name:      dummyFleetName + "-autoscaler",
			Namespace: framework.Namespace,
		},
		Spec: autoscalingv1.FleetAutoscalerSpec{
			FleetName: dummyFleetName,
			Policy: autoscalingv1.FleetAutoscalerPolicy{
				Type: autoscalingv1.BufferPolicyType,
				Buffer: &autoscalingv1.BufferPolicy{
					BufferSize:  intstr.FromInt(3),
					MaxReplicas: 10,
				},
			},
		},
	}
	fas, err := fleetautoscalers.Create(ctx, defaultFas, metav1.CreateOptions{})
	if assert.Nil(t, err) {
		defer fleetautoscalers.Delete(ctx, fas.ObjectMeta.Name, metav1.DeleteOptions{}) // nolint:errcheck
	} else {
		// if we could not create the autoscaler, their is no point going further
		logrus.Error("Failed creating autoscaler, aborting TestFleetAutoscalerDefaultSyncInterval")
		return
	}

	// If the CustomFasSyncInterval feature flag is enabled, fas.spec.sync should be set to its default value
	if runtime.FeatureEnabled(runtime.FeatureCustomFasSyncInterval) {
		defaultSyncIntervalFas := &autoscalingv1.FleetAutoscaler{}
		defaultSyncIntervalFas.ApplyDefaults()
		assert.Equal(t, defaultSyncIntervalFas.Spec.Sync.FixedInterval.Seconds, fas.Spec.Sync.FixedInterval.Seconds)
	}
}

// TestFleetAutoScalerRollingUpdate - test fleet with RollingUpdate strategy work with
// FleetAutoscaler, verify that number of GameServers does not goes down below RollingUpdate strategy
// defined level on Fleet updates.
func TestFleetAutoScalerRollingUpdate(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	stable := framework.AgonesClient.AgonesV1()
	fleets := stable.Fleets(framework.Namespace)
	flt := defaultFleet(framework.Namespace)
	flt.Spec.Replicas = 2
	maxSurge := 1
	rollingUpdateCount := intstr.FromInt(maxSurge)

	flt.Spec.Strategy.RollingUpdate = &appsv1.RollingUpdateDeployment{}
	// Set both MaxSurge and MaxUnavaible to 1
	flt.Spec.Strategy.RollingUpdate.MaxSurge = &rollingUpdateCount
	flt.Spec.Strategy.RollingUpdate.MaxUnavailable = &rollingUpdateCount

	flt, err := fleets.Create(ctx, flt, metav1.CreateOptions{})
	if assert.Nil(t, err) {
		defer fleets.Delete(ctx, flt.ObjectMeta.Name, metav1.DeleteOptions{}) // nolint:errcheck
	}

	framework.AssertFleetCondition(t, flt, e2e.FleetReadyCount(flt.Spec.Replicas))

	fleetautoscalers := framework.AgonesClient.AutoscalingV1().FleetAutoscalers(framework.Namespace)

	// Create FleetAutoScaler with 7 Buffer and MinReplicas
	targetScale := 7
	fas := defaultFleetAutoscaler(flt, framework.Namespace)
	fas.Spec.Policy.Buffer.BufferSize = intstr.FromInt(targetScale)
	fas.Spec.Policy.Buffer.MinReplicas = int32(targetScale)
	fas, err = fleetautoscalers.Create(ctx, fas, metav1.CreateOptions{})
	if assert.Nil(t, err) {
		defer fleetautoscalers.Delete(ctx, fas.ObjectMeta.Name, metav1.DeleteOptions{}) // nolint:errcheck
	} else {
		// if we could not create the autoscaler, their is no point going further
		logrus.Error("Failed creating autoscaler, aborting TestAutoscalerBasicFunctions")
		return
	}
	framework.AssertFleetCondition(t, flt, e2e.FleetReadyCount(int32(targetScale)))

	// get the Status of the fleetautoscaler
	fas, err = framework.AgonesClient.AutoscalingV1().FleetAutoscalers(fas.ObjectMeta.Namespace).Get(ctx, fas.Name, metav1.GetOptions{})
	require.NoError(t, err, "could not get fleetautoscaler")
	assert.True(t, fas.Status.AbleToScale, "Could not get AbleToScale status")

	// check that we are able to scale
	framework.WaitForFleetAutoScalerCondition(t, fas, func(log *logrus.Entry, fas *autoscalingv1.FleetAutoscaler) bool {
		return !fas.Status.ScalingLimited
	})

	// Change ContainerPort to trigger creating a new GSSet
	flt, err = framework.AgonesClient.AgonesV1().Fleets(framework.Namespace).Get(ctx, flt.ObjectMeta.Name, metav1.GetOptions{})

	assert.Nil(t, err, "Able to get the Fleet")
	fltCopy := flt.DeepCopy()
	fltCopy.Spec.Template.Spec.Ports[0].ContainerPort++
	logrus.Info("Current fleet replicas count: ", fltCopy.Spec.Replicas)

	// In ticket #1156 we apply new Replicas size 2, which is smaller than 7
	// And RollingUpdate is broken, scaling immediately from 7 to 2 and then back to 7
	// Uncomment line below to break this test
	// fltCopy.Spec.Replicas = 2

	flt, err = framework.AgonesClient.AgonesV1().Fleets(framework.Namespace).Update(ctx, fltCopy, metav1.UpdateOptions{})
	assert.NoError(t, err)

	selector := labels.SelectorFromSet(labels.Set{agonesv1.FleetNameLabel: flt.ObjectMeta.Name})
	// Wait till new GSS is created
	err = wait.PollImmediate(1*time.Second, 30*time.Second, func() (bool, error) {
		gssList, err := framework.AgonesClient.AgonesV1().GameServerSets(framework.Namespace).List(ctx,
			metav1.ListOptions{LabelSelector: selector.String()})
		if err != nil {
			return false, err
		}
		return len(gssList.Items) == 2, nil
	})
	assert.NoError(t, err)

	// Check that total number of gameservers in the system does not goes lower than RollingUpdate
	// parameters (deleting no more than maxUnavailable servers at a time)
	// Wait for old GSSet to be deleted
	err = wait.PollImmediate(1*time.Second, 5*time.Minute, func() (bool, error) {
		list, err := framework.AgonesClient.AgonesV1().GameServers(framework.Namespace).List(ctx,
			metav1.ListOptions{LabelSelector: selector.String()})
		if err != nil {
			return false, err
		}

		maxUnavailable, err := intstr.GetValueFromIntOrPercent(flt.Spec.Strategy.RollingUpdate.MaxUnavailable, 100, true)
		assert.Nil(t, err)
		if len(list.Items) < targetScale-maxUnavailable {
			err = errors.New("New replicas should be not less than (target - maxUnavailable)")
		}
		if err != nil {
			return false, err
		}
		gssList, err := framework.AgonesClient.AgonesV1().GameServerSets(framework.Namespace).List(ctx,
			metav1.ListOptions{LabelSelector: selector.String()})
		if err != nil {
			return false, err
		}
		return len(gssList.Items) == 1, nil
	})

	assert.NoError(t, err)
}

// TestAutoscalerStressCreate creates many fleetautoscalers with random values
// to check if the creation validation works as expected and if the fleet scales
// to the expected number of replicas (when the creation is valid)
func TestAutoscalerStressCreate(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	log := e2e.TestLogger(t)

	alpha1 := framework.AgonesClient.AgonesV1()
	fleets := alpha1.Fleets(framework.Namespace)
	flt, err := fleets.Create(ctx, defaultFleet(framework.Namespace), metav1.CreateOptions{})
	if assert.Nil(t, err) {
		defer fleets.Delete(ctx, flt.ObjectMeta.Name, metav1.DeleteOptions{}) // nolint:errcheck
	}

	framework.AssertFleetCondition(t, flt, e2e.FleetReadyCount(flt.Spec.Replicas))

	r := rand.New(rand.NewSource(1783))

	fleetautoscalers := framework.AgonesClient.AutoscalingV1().FleetAutoscalers(framework.Namespace)

	for i := 0; i < 5; i++ {
		fas := defaultFleetAutoscaler(flt, framework.Namespace)
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

		log.WithField("buffer", fmt.Sprintf("%#v", fas.Spec.Policy.Buffer)).Info("This is the FAS policy!")

		// create a closure to have defered delete func called on each loop iteration.
		func() {
			fas, err := fleetautoscalers.Create(ctx, fas, metav1.CreateOptions{})
			if err == nil {
				log.WithField("fas", fas.ObjectMeta.Name).Info("Created!")
				defer fleetautoscalers.Delete(ctx, fas.ObjectMeta.Name, metav1.DeleteOptions{}) // nolint:errcheck
				require.True(t, valid,
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
				framework.AssertFleetCondition(t, flt, e2e.FleetReadyCount(expectedReplicas))
			} else {
				require.False(t, valid,
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
func patchFleetAutoscaler(ctx context.Context, fas *autoscalingv1.FleetAutoscaler, bufferSize intstr.IntOrString, minReplicas int32, maxReplicas int32) (*autoscalingv1.FleetAutoscaler, error) {
	var bufferSizeFmt string
	if bufferSize.Type == intstr.Int {
		bufferSizeFmt = fmt.Sprintf("%d", bufferSize.IntValue())
	} else {
		bufferSizeFmt = fmt.Sprintf("%q", bufferSize.String())
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

	fas, err := framework.AgonesClient.AutoscalingV1().FleetAutoscalers(framework.Namespace).
		Patch(ctx, fas.ObjectMeta.Name, types.JSONPatchType, []byte(patch), metav1.PatchOptions{})
	logrus.WithField("fleetautoscaler", fas).Info("Patched fleet autoscaler")
	return fas, err
}

// defaultFleetAutoscaler returns a default fleet autoscaler configuration for a given fleet
func defaultFleetAutoscaler(f *agonesv1.Fleet, namespace string) *autoscalingv1.FleetAutoscaler {
	return &autoscalingv1.FleetAutoscaler{
		ObjectMeta: metav1.ObjectMeta{Name: f.ObjectMeta.Name + "-autoscaler", Namespace: namespace},
		Spec: autoscalingv1.FleetAutoscalerSpec{
			FleetName: f.ObjectMeta.Name,
			Policy: autoscalingv1.FleetAutoscalerPolicy{
				Type: autoscalingv1.BufferPolicyType,
				Buffer: &autoscalingv1.BufferPolicy{
					BufferSize:  intstr.FromInt(3),
					MaxReplicas: 10,
				},
			},
			Sync: &autoscalingv1.FleetAutoscalerSync{
				Type: autoscalingv1.FixedIntervalSyncType,
				FixedInterval: autoscalingv1.FixedIntervalSync{
					Seconds: 30,
				},
			},
		},
	}
}

// Test fleetautoscaler with webhook policy type
// scaling from Replicas equals to 1 to 2
func TestAutoscalerWebhook(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	pod, svc := defaultAutoscalerWebhook(framework.Namespace)
	pod, err := framework.KubeClient.CoreV1().Pods(framework.Namespace).Create(ctx, pod, metav1.CreateOptions{})
	require.NoError(t, err)
	defer framework.KubeClient.CoreV1().Pods(framework.Namespace).Delete(ctx, pod.ObjectMeta.Name, metav1.DeleteOptions{}) // nolint:errcheck
	svc.ObjectMeta.Name = ""
	svc.ObjectMeta.GenerateName = "test-service-"

	svc, err = framework.KubeClient.CoreV1().Services(framework.Namespace).Create(ctx, svc, metav1.CreateOptions{})
	require.NoError(t, err)
	defer framework.KubeClient.CoreV1().Services(framework.Namespace).Delete(ctx, svc.ObjectMeta.Name, metav1.DeleteOptions{}) // nolint:errcheck

	alpha1 := framework.AgonesClient.AgonesV1()
	fleets := alpha1.Fleets(framework.Namespace)
	flt := defaultFleet(framework.Namespace)
	initialReplicasCount := int32(1)
	flt.Spec.Replicas = initialReplicasCount
	flt, err = fleets.Create(ctx, flt, metav1.CreateOptions{})
	require.NoError(t, err)
	defer fleets.Delete(ctx, flt.ObjectMeta.Name, metav1.DeleteOptions{}) // nolint:errcheck

	framework.AssertFleetCondition(t, flt, e2e.FleetReadyCount(flt.Spec.Replicas))

	fleetautoscalers := framework.AgonesClient.AutoscalingV1().FleetAutoscalers(framework.Namespace)
	fas := defaultFleetAutoscaler(flt, framework.Namespace)
	fas.Spec.Policy.Type = autoscalingv1.WebhookPolicyType
	fas.Spec.Policy.Buffer = nil
	path := "scale"
	fas.Spec.Policy.Webhook = &autoscalingv1.WebhookPolicy{
		Service: &admregv1.ServiceReference{
			Name:      svc.ObjectMeta.Name,
			Namespace: framework.Namespace,
			Path:      &path,
		},
	}
	fas, err = fleetautoscalers.Create(ctx, fas, metav1.CreateOptions{})
	require.NoError(t, err)
	defer fleetautoscalers.Delete(ctx, fas.ObjectMeta.Name, metav1.DeleteOptions{}) // nolint:errcheck

	framework.CreateAndApplyAllocation(t, flt)
	framework.AssertFleetCondition(t, flt, func(log *logrus.Entry, fleet *agonesv1.Fleet) bool {
		log.WithField("fleetStatus", fmt.Sprintf("%+v", fleet.Status)).WithField("fleet", fleet.ObjectMeta.Name).Info("Awaiting fleet.Status.AllocatedReplicas == 1")
		return fleet.Status.AllocatedReplicas == 1
	})

	framework.AssertFleetCondition(t, flt, func(log *logrus.Entry, fleet *agonesv1.Fleet) bool {
		log.WithField("fleetStatus", fmt.Sprintf("%+v", fleet.Status)).
			WithField("fleet", fleet.ObjectMeta.Name).
			WithField("initialReplicasCount", initialReplicasCount).
			Info("Awaiting fleet.Status.Replicas > initialReplicasCount")
		return fleet.Status.Replicas > initialReplicasCount
	})

	// Cause an error in Webhook config
	// Use wrong service Path
	err = wait.PollImmediate(time.Second, time.Minute, func() (bool, error) {
		fas, err = fleetautoscalers.Get(ctx, fas.ObjectMeta.Name, metav1.GetOptions{})
		if err != nil {
			return true, err
		}
		newPath := path + "2"
		fas.Spec.Policy.Webhook.Service.Path = &newPath
		labels := map[string]string{"fleetautoscaler": "wrong"}
		fas.ObjectMeta.Labels = labels
		_, err = fleetautoscalers.Update(ctx, fas, metav1.UpdateOptions{})
		if err != nil {
			logrus.WithError(err).Warn("could not update fleet autoscaler")
			return false, nil
		}

		return true, nil
	})
	require.NoError(t, err)

	var l *corev1.EventList
	errString := "Error calculating desired fleet size on FleetAutoscaler"
	found := false

	// Error - net/http: request canceled while waiting for connection (Client.Timeout exceeded
	// while awaiting headers)
	err = wait.PollImmediate(time.Second, time.Minute, func() (bool, error) {
		events := framework.KubeClient.CoreV1().Events(framework.Namespace)
		l, err = events.List(ctx, metav1.ListOptions{FieldSelector: fields.AndSelectors(fields.OneTermEqualSelector("involvedObject.name", fas.ObjectMeta.Name), fields.OneTermEqualSelector("type", "Warning")).String()})
		if err != nil {
			return false, err
		}
		for _, v := range l.Items {
			if strings.Contains(v.Message, errString) {
				found = true
			}
		}
		return found, nil
	})
	assert.NoError(t, err, "Received unexpected error")
	assert.True(t, found, "Expected error was not received")
}

// Instructions: https://agones.dev/site/docs/getting-started/create-webhook-fleetautoscaler/#chapter-2-configuring-https-fleetautoscaler-webhook-with-ca-bundle
// Validity
// 	Not Before: Apr 12 01:37:57 2022 GMT
//	Not After : Aug 25 01:37:57 2023 GMT

var webhookKey = `
-----BEGIN RSA PRIVATE KEY-----
MIIEpgIBAAKCAQEAwMPoSpFJan9iW/W+hbdh8BmMrCmJLGBVFx/4T2l63b5Iz/pb
dg6+++mByDNa1SxzM9JfKkK9Sc+xwYwF3u88NniWlx69NfBpQpnHApgTTMsysQFK
RrvtOmvfon9FXL8B1i1u0e9/sOXobQn2yZfxrQIADl5VDYTIoARPwkf/V71TlEsX
Z8Q9K4zrWU+KiIluf/krGqLObhg9nnkNvOB7eHYHtdJPfUJfduegQoJ6ZCPD/6eP
368Wr6Vp3qLuW9hQFWJOb+8WShMpNBD7V5BQULYYxm/v65wpz/YRx2SXGD+Im00V
FpZt5+SZzQm9d3dfb8Glzt1KaXqWlO0OlkuKqwIDAQABAoIBAQCPSI+/7aKOoMUx
6caGijsoRzWDOxSVgb1+BOuDy7niXXCt90BIzskzYuxvLY0U64duO681MIqW9OUC
ItyyS02Mh7IX/mdSUrNLKBb/XJ7r9BZn77eQQFwjks+Wb9fVCr2IwBihv85AZYSQ
mFlym5iuqs/z3jaGZ+7g0pOeq/mm8u5e1W8r3Ncizqwe9g4yz3+4WXH7TGzXJIb5
BjGbJ7IJJLZBSOpjxnbej5n/lezxwZ/WSqfy7q37g9eleWgtY9qnCXJ4v+x8xVHa
Y9P06MliPiXCwUCuYf4AOb7zBIjY9hvAct7KCY5Vqn2vefeXesRpZruXX0uHvNxE
s8CdVl5RAoGBAPHX9+PvTMz/kQVmItkFjtj8iaDTF8qxSBz5WcDcGpe9bZ0Sjb3N
m+BJnnuRvGL+7DXNmua+nr3O+WcPEINZkaTzDo0IlY7zta9J+cLQCGWjcH6bDFu1
0ZJglJ82reyCPsgfcYFi2LZgsFiFc5u3WkGdqI9dyBnmusMTknWGy48XAoGBAMwM
f7D1ANr9d15hDyHcyoneaC5iP2RfK1iVVGJ4Ov/3qbtw8DGjn0B01mjgEha5cZsq
apcVyHD0rIp4n//J83fxsgPBHcgOaZdjC5LUi/N9VjVhmI3PrhXk6I/lRZYisGrv
9kb2IgvovFCHwPoQ/SJW2RBl0lFsdHIz91zs3P2NAoGBAK7Ox5yXHTFUTXPUlr29
mbpYF/cKfjkBmblvtyODNSmXP8L4ZUHbe59MN2TkO4Jm90AQpLXC9SUHlRicN/hp
ZrAPC+Z/XPNeT2Yrl3/sNRWaZLbuxakIrDoc23CV6nN41X570+SNGU4CZ5UkqSLW
DkQ9fFhclkW6lCZrYELZMwvzAoGBALMNwLtis0Z3p1jdaO75FY4X6WnScvg7/whz
uaHTCUr2ZC4Ec/HLOALSxBcxkQ352vQjK3e6+LIOMp4sLZLC/2/gWqqqutyDsSrU
EiLdepXHBXBAXSML/CJgRaeHtCGD/TVJrt4kPEohB6bPCYsmf0qz1TRrdTxYJHLW
oRkdDOs9AoGBAOm+mMrA+twhaS+ggU37UHvUQGVTra33sQVd008dnzaK5wRuIxSu
J3MH0y8KjS+UKBn8PjsEXMQO/t9LIBqo8A4HuZIZqowoGR6GzrOfnE7lnwf3BKvY
kgmVel9Ssrf7VeJPsb/w2TgL3IIZDR+VXtC1czlwNaQLPxBfmOQa4VyP
-----END RSA PRIVATE KEY-----`

var caPem = `
-----BEGIN CERTIFICATE-----
MIIDRzCCAi+gAwIBAgIUY3Cpf8jTCBaoNZ++5gyZiRq27hMwDQYJKoZIhvcNAQEL
BQAwMzELMAkGA1UEBhMCQVUxEzARBgNVBAgMClNvbWUtU3RhdGUxDzANBgNVBAoM
BkFnb25lczAeFw0yMjA0MTIwMTM1NTNaFw0yNTAxMzAwMTM1NTNaMDMxCzAJBgNV
BAYTAkFVMRMwEQYDVQQIDApTb21lLVN0YXRlMQ8wDQYDVQQKDAZBZ29uZXMwggEi
MA0GCSqGSIb3DQEBAQUAA4IBDwAwggEKAoIBAQCq1uP65QAXdFCYUC+JiH71pxFy
8AA7KwzkFGEIqObE5JpJwnuahkWy29QZzKJ1RKneN34+WtrHVoRhjUiFXYtBOzTb
twd//EJkvYe/5uhV4K4FxMI5+VE81W/66FDc4q0YD6whyT5qyjrLyu8tb2jqbXgY
UaPiiGsbhCwDpq/FdOT71N2V5kBKu+z8PhqDrMUiDsAnKgwIuk6s8D947jv44Q7e
bxandSXnYqAUwqAFhT2YyZnfsqQGuOdD83A62Day1eUQnsv3dKL6H7F25jN5/G42
7GT34Wex/JC7VqfC5RfcrUsr4UOSn4ZyzlCLN8mIM1kU5nMtrYBHZZWxH8ghAgMB
AAGjUzBRMB0GA1UdDgQWBBRWt1C5GFuIdXqmQA8uhqqwUPmjtDAfBgNVHSMEGDAW
gBRWt1C5GFuIdXqmQA8uhqqwUPmjtDAPBgNVHRMBAf8EBTADAQH/MA0GCSqGSIb3
DQEBCwUAA4IBAQApFN9bzBYBWZp4sTyglIjQZzeRdZ/S8WyjhbFeHoqA42izAGGB
rLiKHKym43U/qDxp93Y0Si0K2dv4fyJWqlRZ+gtmVPLwjqkFCQEs/K3+BUHTWE5+
Tx4EStkIJs8M2PipnUMCAICEO9JCp5bw5lloTI4fpIvxOXHiCC6pRmDW9GyyUywT
MoWF7VO43V+aKjMdYxqSK1928Foql4QltnOtPtwySAQujr4kTAdhPuOdnMOdXIS5
6r3Qftfyui85HzhimrAaQ3ZulbNvw7lCWl1BIiidn6VgpXZM4GNFxL5RWIixAyWK
V3FOACAGS/XJ2IirQ0+Ed5B7GCGXx58CqBN5
-----END CERTIFICATE-----`

var webhookCrt = `
-----BEGIN CERTIFICATE-----
MIIDQTCCAimgAwIBAgIUA9ADz3wPH/XvuIei9mWSLOvnzXswDQYJKoZIhvcNAQEL
BQAwMzELMAkGA1UEBhMCQVUxEzARBgNVBAgMClNvbWUtU3RhdGUxDzANBgNVBAoM
BkFnb25lczAeFw0yMjA0MTIwMTM3NTdaFw0yMzA4MjUwMTM3NTdaMC0xKzApBgNV
BAMMImF1dG9zY2FsZXItdGxzLXNlcnZpY2UuZGVmYXVsdC5zdmMwggEiMA0GCSqG
SIb3DQEBAQUAA4IBDwAwggEKAoIBAQDAw+hKkUlqf2Jb9b6Ft2HwGYysKYksYFUX
H/hPaXrdvkjP+lt2Dr776YHIM1rVLHMz0l8qQr1Jz7HBjAXe7zw2eJaXHr018GlC
mccCmBNMyzKxAUpGu+06a9+if0VcvwHWLW7R73+w5ehtCfbJl/GtAgAOXlUNhMig
BE/CR/9XvVOUSxdnxD0rjOtZT4qIiW5/+Ssaos5uGD2eeQ284Ht4dge10k99Ql92
56BCgnpkI8P/p4/frxavpWneou5b2FAVYk5v7xZKEyk0EPtXkFBQthjGb+/rnCnP
9hHHZJcYP4ibTRUWlm3n5JnNCb13d19vwaXO3UppepaU7Q6WS4qrAgMBAAGjUzBR
MAsGA1UdDwQEAwIHgDATBgNVHSUEDDAKBggrBgEFBQcDATAtBgNVHREEJjAkgiJh
dXRvc2NhbGVyLXRscy1zZXJ2aWNlLmRlZmF1bHQuc3ZjMA0GCSqGSIb3DQEBCwUA
A4IBAQBO8MVRJVeaCg80XxnIgcYFXqwgPVqmugYure8cPwsD/tMaISeSavYT/X7L
YIRUnvOgZtjXpX2+43PZjmoxCtKJUa9Q8qWO4MU/6aD1j6wSasjygaOiW5UEKV4j
AWt5U8Jbzf5NZLV0udYErSNE1PqbI8zkELxZ5Usf11C2Nu892lrpJrg6CZjiG82w
PZEUAxKzv6X3w9nF+3fqHkBgRzSwZF9jEAZUkqgqVGAeh2Pzp5O7ciFCL4jAwX9y
DjTCc3SwhWOqeVVnwjmrpPb14t74boH4TijTuK+umGI6U9g0WVmZA8heYil0x7iP
xgD9ZcK4JyVWRkFtu1UFbMuR/M1P
-----END CERTIFICATE-----`

func TestFleetAutoscalerTLSWebhook(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	// we hardcode 'default' namespace here because certificates above are generated to use this one
	defaultNS := "default"

	secr := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "autoscalersecret-",
		},
		Type: corev1.SecretTypeTLS,
		Data: make(map[string][]byte),
	}

	secr.Data[corev1.TLSCertKey] = []byte(webhookCrt)
	secr.Data[corev1.TLSPrivateKeyKey] = []byte(webhookKey)

	secrets := framework.KubeClient.CoreV1().Secrets(defaultNS)
	secr, err := secrets.Create(ctx, secr.DeepCopy(), metav1.CreateOptions{})
	if assert.Nil(t, err) {
		defer secrets.Delete(ctx, secr.ObjectMeta.Name, metav1.DeleteOptions{}) // nolint:errcheck
	}

	pod, svc := defaultAutoscalerWebhook(defaultNS)
	pod.Spec.Volumes = make([]corev1.Volume, 1)
	pod.Spec.Volumes[0] = corev1.Volume{
		Name: "secret-volume",
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName: secr.ObjectMeta.Name,
			},
		},
	}
	pod.Spec.Containers[0].VolumeMounts = []corev1.VolumeMount{{
		Name:      "secret-volume",
		MountPath: "/home/service/certs",
	}}
	pod, err = framework.KubeClient.CoreV1().Pods(defaultNS).Create(ctx, pod.DeepCopy(), metav1.CreateOptions{})
	if assert.Nil(t, err) {
		defer framework.KubeClient.CoreV1().Pods(defaultNS).Delete(ctx, pod.ObjectMeta.Name, metav1.DeleteOptions{}) // nolint:errcheck
	} else {
		// if we could not create the webhook, there is no point going further
		assert.FailNow(t, "Failed creating webhook pod, aborting TestTlsWebhook")
	}

	// since we're using statically-named service, perform a best-effort delete of a previous service
	err = framework.KubeClient.CoreV1().Services(defaultNS).Delete(ctx, svc.ObjectMeta.Name, waitForDeletion)
	if err != nil {
		assert.True(t, k8serrors.IsNotFound(err))
	}

	// making sure the service is really gone.
	err = wait.PollImmediate(2*time.Second, time.Minute, func() (bool, error) {
		_, err := framework.KubeClient.CoreV1().Services(defaultNS).Get(ctx, svc.ObjectMeta.Name, metav1.GetOptions{})
		return k8serrors.IsNotFound(err), nil
	})
	assert.Nil(t, err)

	svc, err = framework.KubeClient.CoreV1().Services(defaultNS).Create(ctx, svc.DeepCopy(), metav1.CreateOptions{})
	if assert.Nil(t, err) {
		defer framework.KubeClient.CoreV1().Services(defaultNS).Delete(ctx, svc.ObjectMeta.Name, metav1.DeleteOptions{}) // nolint:errcheck
	} else {
		// if we could not create the service, there is no point going further
		assert.FailNow(t, "Failed creating service, aborting TestTlsWebhook")
	}

	alpha1 := framework.AgonesClient.AgonesV1()
	fleets := alpha1.Fleets(defaultNS)
	flt := defaultFleet(defaultNS)
	initialReplicasCount := int32(1)
	flt.Spec.Replicas = initialReplicasCount
	flt, err = fleets.Create(ctx, flt.DeepCopy(), metav1.CreateOptions{})
	if assert.Nil(t, err) {
		defer fleets.Delete(ctx, flt.ObjectMeta.Name, metav1.DeleteOptions{}) // nolint:errcheck
	}

	framework.AssertFleetCondition(t, flt, e2e.FleetReadyCount(flt.Spec.Replicas))

	fleetautoscalers := framework.AgonesClient.AutoscalingV1().FleetAutoscalers(defaultNS)
	fas := defaultFleetAutoscaler(flt, defaultNS)
	fas.Spec.Policy.Type = autoscalingv1.WebhookPolicyType
	fas.Spec.Policy.Buffer = nil
	path := "scale"

	fas.Spec.Policy.Webhook = &autoscalingv1.WebhookPolicy{
		Service: &admregv1.ServiceReference{
			Name:      svc.ObjectMeta.Name,
			Namespace: defaultNS,
			Path:      &path,
		},
		CABundle: []byte(caPem),
	}
	fas, err = fleetautoscalers.Create(ctx, fas.DeepCopy(), metav1.CreateOptions{})
	if assert.Nil(t, err) {
		defer fleetautoscalers.Delete(ctx, fas.ObjectMeta.Name, metav1.DeleteOptions{}) // nolint:errcheck
	} else {
		// if we could not create the autoscaler, their is no point going further
		assert.FailNow(t, "Failed creating autoscaler, aborting TestTlsWebhook")
	}
	framework.CreateAndApplyAllocation(t, flt)
	framework.AssertFleetCondition(t, flt, func(log *logrus.Entry, fleet *agonesv1.Fleet) bool {
		return fleet.Status.AllocatedReplicas == 1
	})

	framework.AssertFleetCondition(t, flt, func(log *logrus.Entry, fleet *agonesv1.Fleet) bool {
		return fleet.Status.Replicas > initialReplicasCount
	})
}

func defaultAutoscalerWebhook(namespace string) (*corev1.Pod, *corev1.Service) {
	l := make(map[string]string)
	appName := fmt.Sprintf("autoscaler-webhook-%v", time.Now().UnixNano())
	l["app"] = appName
	l[e2e.AutoCleanupLabelKey] = e2e.AutoCleanupLabelValue
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "auto-webhook-",
			Namespace:    namespace,
			Labels:       l,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{{Name: "webhook",
				Image:           "gcr.io/agones-images/autoscaler-webhook:0.5",
				ImagePullPolicy: corev1.PullAlways,
				Ports: []corev1.ContainerPort{{
					ContainerPort: 8000,
					Name:          "autoscaler",
				}},
			}},
		},
	}
	m := make(map[string]string)
	m["app"] = appName
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "autoscaler-tls-service",
			Namespace: namespace,
		},
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
