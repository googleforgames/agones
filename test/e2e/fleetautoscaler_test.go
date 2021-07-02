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
	e2e "agones.dev/agones/test/e2e/framework"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
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
	fas, err := fleetautoscalers.Create(ctx, defaultFleetAutoscaler(flt, framework.Namespace), metav1.CreateOptions{})
	if assert.Nil(t, err) {
		defer fleetautoscalers.Delete(ctx, fas.ObjectMeta.Name, metav1.DeleteOptions{}) // nolint:errcheck
	} else {
		// if we could not create the autoscaler, their is no point going further
		logrus.Error("Failed creating autoscaler, aborting TestAutoscalerBasicFunctions")
		return
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
	framework.AssertFleetCondition(t, flt, func(fleet *agonesv1.Fleet) bool {
		return fleet.Status.AllocatedReplicas == 1
	})

	framework.AssertFleetCondition(t, flt, e2e.FleetReadyCount(bufferSize))

	// patch autoscaler to switch to relative buffer size and check if the fleet adjusts
	_, err = patchFleetAutoscaler(ctx, fas, intstr.FromString("10%"), 1, fas.Spec.Policy.Buffer.MaxReplicas)
	assert.Nil(t, err, "could not patch fleetautoscaler")

	//10% with only one allocated GS means only one ready server
	framework.AssertFleetCondition(t, flt, e2e.FleetReadyCount(1))

	// get the Status of the fleetautoscaler
	fas, err = framework.AgonesClient.AutoscalingV1().FleetAutoscalers(fas.ObjectMeta.Namespace).Get(ctx, fas.Name, metav1.GetOptions{})
	assert.Nil(t, err, "could not get fleetautoscaler")
	assert.True(t, fas.Status.AbleToScale, "Could not get AbleToScale status")

	// check that we are able to scale
	framework.WaitForFleetAutoScalerCondition(t, fas, func(fas *autoscalingv1.FleetAutoscaler) bool {
		return !fas.Status.ScalingLimited
	})

	// patch autoscaler to a maxReplicas count equal to current replicas count
	_, err = patchFleetAutoscaler(ctx, fas, intstr.FromInt(1), 1, 1)
	assert.Nil(t, err, "could not patch fleetautoscaler")

	// check that we are not able to scale
	framework.WaitForFleetAutoScalerCondition(t, fas, func(fas *autoscalingv1.FleetAutoscaler) bool {
		return fas.Status.ScalingLimited
	})

	// delete the allocated GameServer and watch the fleet scale down
	gp := int64(1)
	err = stable.GameServers(framework.Namespace).Delete(ctx, gsa.Status.GameServerName, metav1.DeleteOptions{GracePeriodSeconds: &gp})
	assert.Nil(t, err)
	framework.AssertFleetCondition(t, flt, func(fleet *agonesv1.Fleet) bool {
		return fleet.Status.AllocatedReplicas == 0 &&
			fleet.Status.ReadyReplicas == 1 &&
			fleet.Status.Replicas == 1
	})
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
	assert.Nil(t, err, "could not get fleetautoscaler")
	assert.True(t, fas.Status.AbleToScale, "Could not get AbleToScale status")

	// check that we are able to scale
	framework.WaitForFleetAutoScalerCondition(t, fas, func(fas *autoscalingv1.FleetAutoscaler) bool {
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
	//fltCopy.Spec.Replicas = 2

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

		// create a closure to have defered delete func called on each loop iteration.
		func() {
			fas, err := fleetautoscalers.Create(ctx, fas, metav1.CreateOptions{})
			if err == nil {
				defer fleetautoscalers.Delete(ctx, fas.ObjectMeta.Name, metav1.DeleteOptions{}) // nolint:errcheck
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
				framework.AssertFleetCondition(t, flt, e2e.FleetReadyCount(expectedReplicas))
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
func patchFleetAutoscaler(ctx context.Context, fas *autoscalingv1.FleetAutoscaler, bufferSize intstr.IntOrString, minReplicas int32, maxReplicas int32) (*autoscalingv1.FleetAutoscaler, error) {
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
			Sync: autoscalingv1.FleetAutoscalerSync{
				Type: autoscalingv1.FixedIntervalSyncType,
				FixedInterval: &autoscalingv1.FixedIntervalSync{
					Seconds: 30,
				},
			},
		},
	}
}

//Test fleetautoscaler with webhook policy type
// scaling from Replicas equals to 1 to 2
func TestAutoscalerWebhook(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	pod, svc := defaultAutoscalerWebhook(framework.Namespace)
	pod, err := framework.KubeClient.CoreV1().Pods(framework.Namespace).Create(ctx, pod, metav1.CreateOptions{})
	if assert.Nil(t, err) {
		defer framework.KubeClient.CoreV1().Pods(framework.Namespace).Delete(ctx, pod.ObjectMeta.Name, metav1.DeleteOptions{}) // nolint:errcheck
	} else {
		// if we could not create the webhook pod, there is no point going further
		assert.FailNow(t, "Failed creating webhook pod, aborting TestAutoscalerWebhook")
	}
	svc.ObjectMeta.Name = ""
	svc.ObjectMeta.GenerateName = "test-service-"

	svc, err = framework.KubeClient.CoreV1().Services(framework.Namespace).Create(ctx, svc, metav1.CreateOptions{})
	if assert.Nil(t, err) {
		defer framework.KubeClient.CoreV1().Services(framework.Namespace).Delete(ctx, svc.ObjectMeta.Name, metav1.DeleteOptions{}) // nolint:errcheck
	} else {
		// if we could not create the webhook service, there is no point going further
		assert.FailNow(t, "Failed creating webhook service, aborting TestAutoscalerWebhook")
	}

	alpha1 := framework.AgonesClient.AgonesV1()
	fleets := alpha1.Fleets(framework.Namespace)
	flt := defaultFleet(framework.Namespace)
	initialReplicasCount := int32(1)
	flt.Spec.Replicas = initialReplicasCount
	flt, err = fleets.Create(ctx, flt, metav1.CreateOptions{})
	if assert.Nil(t, err) {
		defer fleets.Delete(ctx, flt.ObjectMeta.Name, metav1.DeleteOptions{}) // nolint:errcheck
	}

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
	if assert.NoError(t, err) {
		defer fleetautoscalers.Delete(ctx, fas.ObjectMeta.Name, metav1.DeleteOptions{}) // nolint:errcheck
	} else {
		// if we could not create the autoscaler, there is no point going further
		assert.FailNow(t, "Failed creating autoscaler, aborting TestAutoscalerWebhook")
	}
	framework.CreateAndApplyAllocation(t, flt)
	framework.AssertFleetCondition(t, flt, func(fleet *agonesv1.Fleet) bool {
		return fleet.Status.AllocatedReplicas == 1
	})

	framework.AssertFleetCondition(t, flt, func(fleet *agonesv1.Fleet) bool {
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
	assert.NoError(t, err)

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
// Expiration: Sept 30, 2021

var webhookKey = `
-----BEGIN RSA PRIVATE KEY-----
MIIEpQIBAAKCAQEAzTtFY02SAY4jHiryJbBRT4+2wn1OlqL4WTWUFtKaWEjm+gAn
vLlmNB/dBPL36r1vDbYGPO2MiWF5ULfoe1y/YsQzmwQLnGFhEX5Ou72J2pHQfa1h
VqYiLDz7Pi70qxjABIh/4U/x6x6nfDpvk3a0PBcCnPDDrJvkqpYqJnqfYkKT7LNr
NYCTn12RTIRJTxffDkuQBs9y4RGhmX0Nh9bb5iwpYXQRrKHnwMniR1D9MvJ27aKC
KwpDR+TkdYl/FXYIguFgipPUdw78KYcFA8DJipwkyIqcRvWml1o5yhhwJpNDck3s
S73g+IvoF/YPRy42dCQHhRu4b+JoqOT2jpUD8QIDAQABAoIBAD0orpLbKOmBvAFf
du24T2LQRvxKb0MAqdWb29e5RvmMMBjMNwtMjKJ35Ft3NF4luZRybAV4HOtLuuVN
CODKUNZT9bT6TaN6eXzHERttbklOLr1lD57Mv15DhfOP9qWOKJqxOrqgIk2YwvyI
RXvCYg+OI980+HrVsh0Lxt/Upu6Ws84L+PHSr0McAr7bWaR2ATRKsfQYxNFTtk3v
7ckSGzyIFCy7ijE4g5m2MrZZb/AUzaXfp8PSZgoFC+2dQcKhPshJb9tEBHAv9wPc
JkKZZfmR5n4VbtRekvf9rU5oODfbDOHXzt8b3dsskiZvAbeBdrHVS0kXrPsKj4Li
a1OVRokCgYEA8FKB88YK3PYNH+X7nlUWLHsXjbP9r8bMtYBuvPKkBI06gvJfrDKr
cOhkoZyWzDQbwf1F1UiWvxAkIWNmvGezps+rOY8YyIOELOnWe/5MOBzK2KlEys88
fbJ8G2uHe59N/1cAS6jvPq57TT2SQPe1jjibr0QvbitQ93tt3sjQhdsCgYEA2p67
RX8W5ubToU/oeykzQBXkQa1ppWYG2PBCuW8bqYNsR5mG+YjsQWNDxOTEiwBm5hXO
xb6IdwaOsKHc7dT3ItcLyQPEPvBdYzbxwl9NZMvooPvbFKLHJD8Rlwp4pQpO4K7+
3XH7r82cjiAH4+6WYjFBi/JXQDEEVwVvJYkzVSMCgYEAndsUUTO83vcgF9vRM2dg
cUdJaWLZOCS1QmNiWepnojXCQVFDVrDRvBBqSV26D9gKg5oBzN8pZccMdIH+cbMM
Zn3yUpSUCuGYaIgQwtF+7zy6YSaOcUk+yrH6o2g2ThWN/jL/lrMYs2uYwlu3PcV4
FDtKyA1ZulvpiyYgPT5a+hECgYEApFl4B4LHQMZ+imJ8LzqF4MOUWRt4tHLC6wuT
3bt9XC4ElL8CDU217mIlbDte1fBzar0yOM5H4NL5KihE4jabo4FuxqsiOP6R9ig0
Dx9+Gyx/saYkyJqmgsU3AAlLMSdSrO5hgzBROZSlAONrixqtyxukXwTMOuGelZzs
NZezE2kCgYEAyScA3it+gul6VXvOQIgSh656iCFZBxkWnElC+5FJ7MhMbIzNRFU6
ez6oDiXtqPuC5Y24zYzFsSV+ufTdF2NUNX44aopz+zmhMua04Fs41gZC12thIDPK
DhTkOAfCsisbX2bdrLJaSFySr+lf/yqQdTZo2YBRL3sHAhB0RKjDxaQ=
-----END RSA PRIVATE KEY-----`

var caPem = `
-----BEGIN CERTIFICATE-----
MIIDazCCAlOgAwIBAgIUZcFCLeoSpuSYusLxYkgGvaAF3cMwDQYJKoZIhvcNAQEL
BQAwRTELMAkGA1UEBhMCQVUxEzARBgNVBAgMClNvbWUtU3RhdGUxITAfBgNVBAoM
GEludGVybmV0IFdpZGdpdHMgUHR5IEx0ZDAeFw0yMDExMjUxMzM4MzlaFw0yMzA5
MTUxMzM4MzlaMEUxCzAJBgNVBAYTAkFVMRMwEQYDVQQIDApTb21lLVN0YXRlMSEw
HwYDVQQKDBhJbnRlcm5ldCBXaWRnaXRzIFB0eSBMdGQwggEiMA0GCSqGSIb3DQEB
AQUAA4IBDwAwggEKAoIBAQDEm12qM8qZ4BsFRCXAMTutdvRRLWAeICjOkoK037eX
F+X8P0yDY9d8PkWafhvtL0qSfS30a0Hj3tyazzsF7GqdRWGadzPMARTlIxij9w7f
Odd1KQ4/zFpA9WhciBcoIQsiwEhojMoXx8kLrX6ELnbh4zdGnsn0K9g3ZdYKKIu7
kc3iwzDMTJ2hzvgdB/hAOoYRSZXr/+wLGL8DzriT7slkI/T+n83UuaoTMPO41d7v
5272Ify+DJxI6bUQTAxS/5UAh9DHCUKuR9fB4eIniv/Zc5TA8BLQaa7DERRhJPfp
o9pjQVGFsb4gzxY0kY3QPnQHjkSpVc5qQ5KgJHFB7hLvAgMBAAGjUzBRMB0GA1Ud
DgQWBBRmwkHbhsi96Kuz09+UML+gVV/GkzAfBgNVHSMEGDAWgBRmwkHbhsi96Kuz
09+UML+gVV/GkzAPBgNVHRMBAf8EBTADAQH/MA0GCSqGSIb3DQEBCwUAA4IBAQAv
8hh3hxuLgpMfWNe8C/ODBwUkkSuPhvCHJyevTihLV07go+Cj/cBEN6R3hQbMJ+qJ
JfD3T1fYAn3phq+kcPrZ4auYCDMTw+RoTWjpiknak+iDpUXbBV5Y/Km6ybgFtnlR
MNvCirTyBjE/uQ3PuJpLMGyaePzwhk0sVdvt8Ei7ZXJVMr6APcXJA3TotdTu3wPZ
LKBEJ3UpxePZr0IiZjpdlLngkcIzbHQbBhxQzVCHFtH31A56w/l6N75H/sdA/Jcr
1OJXwNg9mWihC2HVsJl5RKq7WibRmjNICf3v8Mqgkn+2MOps1DXLNqKsKuNOUDhR
0/GqbK5s1fmTucB5JysM
-----END CERTIFICATE-----`

var webhookCrt = `
-----BEGIN CERTIFICATE-----
MIIDUzCCAjugAwIBAgIUB3HgoTF9rHLt++aLHjEAzU80KHYwDQYJKoZIhvcNAQEL
BQAwRTELMAkGA1UEBhMCQVUxEzARBgNVBAgMClNvbWUtU3RhdGUxITAfBgNVBAoM
GEludGVybmV0IFdpZGdpdHMgUHR5IEx0ZDAeFw0yMDExMjUxMzM4MzlaFw0yMjA0
MDkxMzM4MzlaMC0xKzApBgNVBAMMImF1dG9zY2FsZXItdGxzLXNlcnZpY2UuZGVm
YXVsdC5zdmMwggEiMA0GCSqGSIb3DQEBAQUAA4IBDwAwggEKAoIBAQDNO0VjTZIB
jiMeKvIlsFFPj7bCfU6WovhZNZQW0ppYSOb6ACe8uWY0H90E8vfqvW8NtgY87YyJ
YXlQt+h7XL9ixDObBAucYWERfk67vYnakdB9rWFWpiIsPPs+LvSrGMAEiH/hT/Hr
Hqd8Om+TdrQ8FwKc8MOsm+Sqliomep9iQpPss2s1gJOfXZFMhElPF98OS5AGz3Lh
EaGZfQ2H1tvmLClhdBGsoefAyeJHUP0y8nbtooIrCkNH5OR1iX8VdgiC4WCKk9R3
DvwphwUDwMmKnCTIipxG9aaXWjnKGHAmk0NyTexLveD4i+gX9g9HLjZ0JAeFG7hv
4mio5PaOlQPxAgMBAAGjUzBRMAsGA1UdDwQEAwIHgDATBgNVHSUEDDAKBggrBgEF
BQcDATAtBgNVHREEJjAkgiJhdXRvc2NhbGVyLXRscy1zZXJ2aWNlLmRlZmF1bHQu
c3ZjMA0GCSqGSIb3DQEBCwUAA4IBAQBsaIyIEFzPthS73i0+3EiJh3mIJ1vAJPUQ
E2TEG8Nh/IsqnsCOkwX1LBN3PWhwS1KDVK4Ed1Ct0y7Q7kcni7kTj3TqPVXXm/M9
33K5SBOdcl4GPVREMMmy7spttHbrydMoHMojbTn5/Dk6tDlGUdreMXWzaN9m3Mtd
wbX2rOVB9Uq7S077wTviS08Wvox+ia4rnSOquCId6XSPUziBbBccbxLfVSyzvDvm
ZPGiDqt2GpLYQp2VnVpigB0AACqk8QWjN6hIQ+mmyuXaN+kzwbPWPt5K0Yf9UT75
0WFzbkggv+mugnl9t1HEdImGdKmUNx1mL/dbODIikDz3bKsQHutd
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
	framework.AssertFleetCondition(t, flt, func(fleet *agonesv1.Fleet) bool {
		return fleet.Status.AllocatedReplicas == 1
	})

	framework.AssertFleetCondition(t, flt, func(fleet *agonesv1.Fleet) bool {
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
				Image:           "gcr.io/agones-images/autoscaler-webhook:0.3",
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
