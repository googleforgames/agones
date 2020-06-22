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
	admregv1b "k8s.io/api/admissionregistration/v1beta1"
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

var waitForDeletion = &metav1.DeleteOptions{
	PropagationPolicy: &deletePropagationForeground,
}

func TestAutoscalerBasicFunctions(t *testing.T) {
	t.Parallel()

	stable := framework.AgonesClient.AgonesV1()
	fleets := stable.Fleets(framework.Namespace)
	flt, err := fleets.Create(defaultFleet(framework.Namespace))
	if assert.Nil(t, err) {
		defer fleets.Delete(flt.ObjectMeta.Name, nil) // nolint:errcheck
	}

	framework.AssertFleetCondition(t, flt, e2e.FleetReadyCount(flt.Spec.Replicas))

	fleetautoscalers := framework.AgonesClient.AutoscalingV1().FleetAutoscalers(framework.Namespace)
	fas, err := fleetautoscalers.Create(defaultFleetAutoscaler(flt, framework.Namespace))
	if assert.Nil(t, err) {
		defer fleetautoscalers.Delete(fas.ObjectMeta.Name, nil) // nolint:errcheck
	} else {
		// if we could not create the autoscaler, their is no point going further
		logrus.Error("Failed creating autoscaler, aborting TestAutoscalerBasicFunctions")
		return
	}

	// the fleet autoscaler should scale the fleet up now up to BufferSize
	bufferSize := int32(fas.Spec.Policy.Buffer.BufferSize.IntValue())
	framework.AssertFleetCondition(t, flt, e2e.FleetReadyCount(bufferSize))

	// patch the autoscaler to increase MinReplicas and watch the fleet scale up
	fas, err = patchFleetAutoscaler(fas, intstr.FromInt(int(bufferSize)), bufferSize+2, fas.Spec.Policy.Buffer.MaxReplicas)
	assert.Nil(t, err, "could not patch fleetautoscaler")

	// min replicas is now higher than buffer size, will scale to that level
	framework.AssertFleetCondition(t, flt, e2e.FleetReadyCount(fas.Spec.Policy.Buffer.MinReplicas))

	// patch the autoscaler to remove MinReplicas and watch the fleet scale down to bufferSize
	fas, err = patchFleetAutoscaler(fas, intstr.FromInt(int(bufferSize)), 0, fas.Spec.Policy.Buffer.MaxReplicas)
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
	_, err = patchFleetAutoscaler(fas, intstr.FromString("10%"), 1, fas.Spec.Policy.Buffer.MaxReplicas)
	assert.Nil(t, err, "could not patch fleetautoscaler")

	//10% with only one allocated GS means only one ready server
	framework.AssertFleetCondition(t, flt, e2e.FleetReadyCount(1))

	// get the Status of the fleetautoscaler
	fas, err = framework.AgonesClient.AutoscalingV1().FleetAutoscalers(fas.ObjectMeta.Namespace).Get(fas.Name, metav1.GetOptions{})
	assert.Nil(t, err, "could not get fleetautoscaler")
	assert.True(t, fas.Status.AbleToScale, "Could not get AbleToScale status")

	// check that we are able to scale
	framework.WaitForFleetAutoScalerCondition(t, fas, func(fas *autoscalingv1.FleetAutoscaler) bool {
		return !fas.Status.ScalingLimited
	})

	// patch autoscaler to a maxReplicas count equal to current replicas count
	_, err = patchFleetAutoscaler(fas, intstr.FromInt(1), 1, 1)
	assert.Nil(t, err, "could not patch fleetautoscaler")

	// check that we are not able to scale
	framework.WaitForFleetAutoScalerCondition(t, fas, func(fas *autoscalingv1.FleetAutoscaler) bool {
		return fas.Status.ScalingLimited
	})

	// delete the allocated GameServer and watch the fleet scale down
	gp := int64(1)
	err = stable.GameServers(framework.Namespace).Delete(gsa.Status.GameServerName, &metav1.DeleteOptions{GracePeriodSeconds: &gp})
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

	flt, err := fleets.Create(flt)
	if assert.Nil(t, err) {
		defer fleets.Delete(flt.ObjectMeta.Name, nil) // nolint:errcheck
	}

	framework.AssertFleetCondition(t, flt, e2e.FleetReadyCount(flt.Spec.Replicas))

	fleetautoscalers := framework.AgonesClient.AutoscalingV1().FleetAutoscalers(framework.Namespace)

	// Create FleetAutoScaler with 7 Buffer and MinReplicas
	targetScale := 7
	fas := defaultFleetAutoscaler(flt, framework.Namespace)
	fas.Spec.Policy.Buffer.BufferSize = intstr.FromInt(targetScale)
	fas.Spec.Policy.Buffer.MinReplicas = int32(targetScale)
	fas, err = fleetautoscalers.Create(fas)
	if assert.Nil(t, err) {
		defer fleetautoscalers.Delete(fas.ObjectMeta.Name, nil) // nolint:errcheck
	} else {
		// if we could not create the autoscaler, their is no point going further
		logrus.Error("Failed creating autoscaler, aborting TestAutoscalerBasicFunctions")
		return
	}
	framework.AssertFleetCondition(t, flt, e2e.FleetReadyCount(int32(targetScale)))

	// get the Status of the fleetautoscaler
	fas, err = framework.AgonesClient.AutoscalingV1().FleetAutoscalers(fas.ObjectMeta.Namespace).Get(fas.Name, metav1.GetOptions{})
	assert.Nil(t, err, "could not get fleetautoscaler")
	assert.True(t, fas.Status.AbleToScale, "Could not get AbleToScale status")

	// check that we are able to scale
	framework.WaitForFleetAutoScalerCondition(t, fas, func(fas *autoscalingv1.FleetAutoscaler) bool {
		return !fas.Status.ScalingLimited
	})

	// Change ContainerPort to trigger creating a new GSSet
	flt, err = framework.AgonesClient.AgonesV1().Fleets(framework.Namespace).Get(flt.ObjectMeta.Name, metav1.GetOptions{})

	assert.Nil(t, err, "Able to get the Fleet")
	fltCopy := flt.DeepCopy()
	fltCopy.Spec.Template.Spec.Ports[0].ContainerPort++
	logrus.Info("Current fleet replicas count: ", fltCopy.Spec.Replicas)

	// In ticket #1156 we apply new Replicas size 2, which is smaller than 7
	// And RollingUpdate is broken, scaling immediately from 7 to 2 and then back to 7
	// Uncomment line below to break this test
	//fltCopy.Spec.Replicas = 2

	flt, err = framework.AgonesClient.AgonesV1().Fleets(framework.Namespace).Update(fltCopy)
	assert.NoError(t, err)

	selector := labels.SelectorFromSet(labels.Set{agonesv1.FleetNameLabel: flt.ObjectMeta.Name})
	// Wait till new GSS is created
	err = wait.PollImmediate(1*time.Second, 30*time.Second, func() (bool, error) {
		gssList, err := framework.AgonesClient.AgonesV1().GameServerSets(framework.Namespace).List(
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
		list, err := framework.AgonesClient.AgonesV1().GameServers(framework.Namespace).List(
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
		gssList, err := framework.AgonesClient.AgonesV1().GameServerSets(framework.Namespace).List(
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

	alpha1 := framework.AgonesClient.AgonesV1()
	fleets := alpha1.Fleets(framework.Namespace)
	flt, err := fleets.Create(defaultFleet(framework.Namespace))
	if assert.Nil(t, err) {
		defer fleets.Delete(flt.ObjectMeta.Name, nil) // nolint:errcheck
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
func patchFleetAutoscaler(fas *autoscalingv1.FleetAutoscaler, bufferSize intstr.IntOrString, minReplicas int32, maxReplicas int32) (*autoscalingv1.FleetAutoscaler, error) {
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

	fas, err := framework.AgonesClient.AutoscalingV1().FleetAutoscalers(framework.Namespace).Patch(fas.ObjectMeta.Name, types.JSONPatchType, []byte(patch))
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
		},
	}
}

//Test fleetautoscaler with webhook policy type
// scaling from Replicas equals to 1 to 2
func TestAutoscalerWebhook(t *testing.T) {
	t.Parallel()
	pod, svc := defaultAutoscalerWebhook(framework.Namespace)
	pod, err := framework.KubeClient.CoreV1().Pods(framework.Namespace).Create(pod)
	if assert.Nil(t, err) {
		defer framework.KubeClient.CoreV1().Pods(framework.Namespace).Delete(pod.ObjectMeta.Name, nil) // nolint:errcheck
	} else {
		// if we could not create the webhook pod, there is no point going further
		assert.FailNow(t, "Failed creating webhook pod, aborting TestAutoscalerWebhook")
	}
	svc.ObjectMeta.Name = ""
	svc.ObjectMeta.GenerateName = "test-service-"

	svc, err = framework.KubeClient.CoreV1().Services(framework.Namespace).Create(svc)
	if assert.Nil(t, err) {
		defer framework.KubeClient.CoreV1().Services(framework.Namespace).Delete(svc.ObjectMeta.Name, nil) // nolint:errcheck
	} else {
		// if we could not create the webhook service, there is no point going further
		assert.FailNow(t, "Failed creating webhook service, aborting TestAutoscalerWebhook")
	}

	alpha1 := framework.AgonesClient.AgonesV1()
	fleets := alpha1.Fleets(framework.Namespace)
	flt := defaultFleet(framework.Namespace)
	initialReplicasCount := int32(1)
	flt.Spec.Replicas = initialReplicasCount
	flt, err = fleets.Create(flt)
	if assert.Nil(t, err) {
		defer fleets.Delete(flt.ObjectMeta.Name, nil) // nolint:errcheck
	}

	framework.AssertFleetCondition(t, flt, e2e.FleetReadyCount(flt.Spec.Replicas))

	fleetautoscalers := framework.AgonesClient.AutoscalingV1().FleetAutoscalers(framework.Namespace)
	fas := defaultFleetAutoscaler(flt, framework.Namespace)
	fas.Spec.Policy.Type = autoscalingv1.WebhookPolicyType
	fas.Spec.Policy.Buffer = nil
	path := "scale"
	fas.Spec.Policy.Webhook = &autoscalingv1.WebhookPolicy{
		Service: &admregv1b.ServiceReference{
			Name:      svc.ObjectMeta.Name,
			Namespace: framework.Namespace,
			Path:      &path,
		},
	}
	fas, err = fleetautoscalers.Create(fas)
	if assert.NoError(t, err) {
		defer fleetautoscalers.Delete(fas.ObjectMeta.Name, nil) // nolint:errcheck
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
		fas, err = fleetautoscalers.Get(fas.ObjectMeta.Name, metav1.GetOptions{})
		if err != nil {
			return true, err
		}
		newPath := path + "2"
		fas.Spec.Policy.Webhook.Service.Path = &newPath
		labels := map[string]string{"fleetautoscaler": "wrong"}
		fas.ObjectMeta.Labels = labels
		_, err = fleetautoscalers.Update(fas)
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
		l, err = events.List(metav1.ListOptions{FieldSelector: fields.AndSelectors(fields.OneTermEqualSelector("involvedObject.name", fas.ObjectMeta.Name), fields.OneTermEqualSelector("type", "Warning")).String()})
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
MIIEpAIBAAKCAQEAxNq5ql070cs/foKC+Abj6ETgcIq8HJlYT85wbl8wE5sUqsGx
5iItWZMwFJpEFyW97YNOizc8dYOAGCSLUeEY1aHg0vVSj3tV3jjVD82PtSGhTuE8
HGN5WdzEHavJHraRZB4wYMdjm4cFB6TJ3k8qBz5I8Rjp9CcLtpXTcvm0uWieScVy
yOvhO3XCYXbGo1HLj4wkxL+bYrci8XltTICXmQxT365+YmGNWvaWAUYKm2krmzdv
ObZBowzXFohWIsCYpGMaVRbVoYHry/a1S/DXxQ32WuoyRqnFERrV8mrq6kxYQ0Lc
GvWay5n/p0/glyU1IEOi7gEuy9Wccaf1miegQwIDAQABAoIBAQCj/mdwYw2DoCP8
O7Pp9quFA2RKvXkrBiDJE30cpdYCb06PVp/izZQkLHeAomeZNQr9xEb5uYF3kJ50
/nTGOJUc3CfU9yTZfXEymPv+l0xiJGsisIcIS2J8F2uWIFeDa6rB0liRN2pm1du9
222E80RbFmtj11KH4MNkT3sBLL9/OQ21Hymyd+ACTbK+lls9crxq9IqQNMYz2Gxu
HRHsVFGc+j5+gxVLY4tRrTQb5BGr51g3LOFo7LgYguWxBDJdL+f2qNaOtpFfzysO
3bReYqImxfTT3pFEionSHU83UQdpGUgdinj7XC/8N//vyEQDWmyk6p53LrYGaan9
+NoZ4YY5AoGBAPiBo8QKQa/MBNM0ro84fAXIOI8q+XanjB0Qa9eOB3ZxSp63sR3M
Xm3CcrMRLz6To+AOE8uyvxV22NWmN3jZgUhoMS59EdCRlsI1otMUfpAw4Iw1wZ4g
fZ1Phj8fr9B7zNHzRDm55KXhp99AX+WKEFAUA+z2Q5B4LJuUHa14or+lAoGBAMrK
Ws7vknNx4Xju/JBur0vLKFOqoQdmLcPCJWsf7kz+wfCbolal4ohAQ01zVKGwj8yD
MFyZXm1A6MyjLUrn+r4BHtYfS4bBZUyykyGTyiWpfh4AvOqKqBTPWK/XGFfl2dT4
Os8osgmh2DlutyIk7piEx9K6+yeW0h5T1SAsYVvHAoGAXlhRhU7ji0tolYrNruAh
7cwK9Qe6t/p6LlqaprZsTOJMEx/oJUj+nKsTArrGdfp1X83YZCBTfWGmhs5ZBw+E
jqnH6j9fcRCk7MySKZMBTdrQlUqfXFo3dm7Hp9Vu2Tb3FspFn6jcjsGyCwcUoT+e
W9iNePwxwHpvbQ15iu9e0mUCgYAEsBbXT8x35LsMm6G1CQn+W4zsGjaswBzwuI06
47sThpQfJsni7OTGt42WvcLIFhfM5393tIftSKHZETCb2a7/M3FuC70oOVJJKpui
HBOBOWDT+rpjRZ9LE9v9/J/wcDzP4okhftRWyqn/8eJD5MyrM+6WnYHu0Vq8Hr3/
h2ccwwKBgQCha4ox+SaXzlYROr1qge8xgqK+lpg1i2f32PgK3Ipodjk3esAuzeNM
L5o5pDLorHu4EFtveIneRsV+kf8YPVuid18lYzMAJBqlXvcUik2Izk57cWB/P9so
3/03jXI8iT8BbIU+PoII2EvQEPeAI07BYMU9cvsiFvFoB6z162DJhw==
-----END RSA PRIVATE KEY-----`

var caPem = `
-----BEGIN CERTIFICATE-----
MIIDazCCAlOgAwIBAgIUQWAeRC5nziGEM6YlkpDp4ZIY/jEwDQYJKoZIhvcNAQEL
BQAwRTELMAkGA1UEBhMCQVUxEzARBgNVBAgMClNvbWUtU3RhdGUxITAfBgNVBAoM
GEludGVybmV0IFdpZGdpdHMgUHR5IEx0ZDAeFw0yMDA1MTgyMDM4MzRaFw0yMzAz
MDgyMDM4MzRaMEUxCzAJBgNVBAYTAkFVMRMwEQYDVQQIDApTb21lLVN0YXRlMSEw
HwYDVQQKDBhJbnRlcm5ldCBXaWRnaXRzIFB0eSBMdGQwggEiMA0GCSqGSIb3DQEB
AQUAA4IBDwAwggEKAoIBAQDV8IUgXP7sJxpSGZdpbK0fs+5JGO8kZDMM0AOGi5ne
HRPjmxIjOPyUQ3xRA1D/l+gvYflfdksfvSLfyz/yL/Sbsun+TatL25xfTcSP5d14
r99kZoARD9ZWyr1L+0DjnkzhzKIZuucuXiitQ4EX5IBIutwpmpPG4BIOLA8ADYct
IjeKNuh37FdDqCbgEsJxkU3oODE+JUve+ZS+ft6VR8IqvSYmigsvSV2tUyabQ/c3
+iXoTg+3yXmNnMeIYczW674YVHqMnMxbPXo2MI3uYX8b/3gqYPywWYVolF+TgPjO
LpUOOy+dfF2gQYNAlv+/PAjwYm7Q5wNwArUH/gFz8467AgMBAAGjUzBRMB0GA1Ud
DgQWBBS8VyQTvN8fDv7GRpN7j8OCFRx9hjAfBgNVHSMEGDAWgBS8VyQTvN8fDv7G
RpN7j8OCFRx9hjAPBgNVHRMBAf8EBTADAQH/MA0GCSqGSIb3DQEBCwUAA4IBAQCd
sS/UT6lU8AK4gvidULJhtz05fKP9vFZC2h/vc2zViREUHzQBHOKPHG7Hi76/e/sh
vTZEz7KSDygYbv+wWYtM1sLbt6OaEzNYHUqv6TCi5E6Pdisy+XBjwIqdB0RxxcrP
VQBmsXMBhM4qvhyANuw6O40GgTs2vCbxJkPwFjGwOcOIu5eQm9G/DqJ/Tm66u4YU
C4ll3vZPDgrJoZgou8ufa/+ekLZx0eJ/y/Wn3Wiqm/uEOewoVHCpf70cXNzJ2tT/
Tur3yJmj1KbPlTY5RTZaB/TZSGVBRhPRcu8nMJlp2nZVQtq2Z1NKqF3qFMDy9wyM
kUZOQ8SewyYktz5l+z8N
-----END CERTIFICATE-----`

var webhookCrt = `
-----BEGIN CERTIFICATE-----
MIIDPjCCAiYCFEB3Nm7h3CeNWSnE/YtRSTAOqUw7MA0GCSqGSIb3DQEBCwUAMEUx
CzAJBgNVBAYTAkFVMRMwEQYDVQQIDApTb21lLVN0YXRlMSEwHwYDVQQKDBhJbnRl
cm5ldCBXaWRnaXRzIFB0eSBMdGQwHhcNMjAwNTE4MjA0MDIzWhcNMjEwOTMwMjA0
MDIzWjByMQswCQYDVQQGEwJBVTETMBEGA1UECAwKU29tZS1TdGF0ZTEhMB8GA1UE
CgwYSW50ZXJuZXQgV2lkZ2l0cyBQdHkgTHRkMSswKQYDVQQDDCJhdXRvc2NhbGVy
LXRscy1zZXJ2aWNlLmRlZmF1bHQuc3ZjMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8A
MIIBCgKCAQEAxNq5ql070cs/foKC+Abj6ETgcIq8HJlYT85wbl8wE5sUqsGx5iIt
WZMwFJpEFyW97YNOizc8dYOAGCSLUeEY1aHg0vVSj3tV3jjVD82PtSGhTuE8HGN5
WdzEHavJHraRZB4wYMdjm4cFB6TJ3k8qBz5I8Rjp9CcLtpXTcvm0uWieScVyyOvh
O3XCYXbGo1HLj4wkxL+bYrci8XltTICXmQxT365+YmGNWvaWAUYKm2krmzdvObZB
owzXFohWIsCYpGMaVRbVoYHry/a1S/DXxQ32WuoyRqnFERrV8mrq6kxYQ0LcGvWa
y5n/p0/glyU1IEOi7gEuy9Wccaf1miegQwIDAQABMA0GCSqGSIb3DQEBCwUAA4IB
AQC7S1ZBndfsMDK+58l2N1N/8Gm50XhqsG8u0dFf5bVhLgohOhUCMpj246z0lSLo
hWbdokSjrnUWyvM1Dv+ZWTQ+eS/4UamDyr6993Je1p9fVvHAGped97YAxlSAj5dL
CYr+9xqTPtOAVEwiddbEK2wId4XNuD2yPt0YHP22bATh7UyeyqxWTks6LamNJitZ
qrh1J4ZuqSJtnSXdwh3Zm9aoDxAd966dFXZgsoEg9/Au/C7PpyUx4JH5eTV9wBSy
6T4qnpkTnD01dUdLpwBlshAkVrdJRuKVE/152gQcOJ+tm+eXO0VCs6JWpUowZWAt
rteG9laTLeoJFDeCvc+pzWX+
-----END CERTIFICATE-----`

func TestFleetAutoscalerTLSWebhook(t *testing.T) {
	t.Parallel()

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
	secr, err := secrets.Create(secr.DeepCopy())
	if assert.Nil(t, err) {
		defer secrets.Delete(secr.ObjectMeta.Name, nil) // nolint:errcheck
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
	pod, err = framework.KubeClient.CoreV1().Pods(defaultNS).Create(pod.DeepCopy())
	if assert.Nil(t, err) {
		defer framework.KubeClient.CoreV1().Pods(defaultNS).Delete(pod.ObjectMeta.Name, nil) // nolint:errcheck
	} else {
		// if we could not create the webhook, there is no point going further
		assert.FailNow(t, "Failed creating webhook pod, aborting TestTlsWebhook")
	}

	// since we're using statically-named service, perform a best-effort delete of a previous service
	err = framework.KubeClient.CoreV1().Services(defaultNS).Delete(svc.ObjectMeta.Name, waitForDeletion)
	if err != nil {
		assert.True(t, k8serrors.IsNotFound(err))
	}

	// making sure the service is really gone.
	err = wait.PollImmediate(2*time.Second, time.Minute, func() (bool, error) {
		_, err := framework.KubeClient.CoreV1().Services(defaultNS).Get(svc.ObjectMeta.Name, metav1.GetOptions{})
		return k8serrors.IsNotFound(err), nil
	})
	assert.Nil(t, err)

	svc, err = framework.KubeClient.CoreV1().Services(defaultNS).Create(svc.DeepCopy())
	if assert.Nil(t, err) {
		defer framework.KubeClient.CoreV1().Services(defaultNS).Delete(svc.ObjectMeta.Name, nil) // nolint:errcheck
	} else {
		// if we could not create the service, there is no point going further
		assert.FailNow(t, "Failed creating service, aborting TestTlsWebhook")
	}

	alpha1 := framework.AgonesClient.AgonesV1()
	fleets := alpha1.Fleets(defaultNS)
	flt := defaultFleet(defaultNS)
	initialReplicasCount := int32(1)
	flt.Spec.Replicas = initialReplicasCount
	flt, err = fleets.Create(flt.DeepCopy())
	if assert.Nil(t, err) {
		defer fleets.Delete(flt.ObjectMeta.Name, nil) // nolint:errcheck
	}

	framework.AssertFleetCondition(t, flt, e2e.FleetReadyCount(flt.Spec.Replicas))

	fleetautoscalers := framework.AgonesClient.AutoscalingV1().FleetAutoscalers(defaultNS)
	fas := defaultFleetAutoscaler(flt, defaultNS)
	fas.Spec.Policy.Type = autoscalingv1.WebhookPolicyType
	fas.Spec.Policy.Buffer = nil
	path := "scale"

	fas.Spec.Policy.Webhook = &autoscalingv1.WebhookPolicy{
		Service: &admregv1b.ServiceReference{
			Name:      svc.ObjectMeta.Name,
			Namespace: defaultNS,
			Path:      &path,
		},
		CABundle: []byte(caPem),
	}
	fas, err = fleetautoscalers.Create(fas.DeepCopy())
	if assert.Nil(t, err) {
		defer fleetautoscalers.Delete(fas.ObjectMeta.Name, nil) // nolint:errcheck
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
