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
	"bytes"
	"context"
	cryptorand "crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"math/rand"
	"strings"
	"testing"
	"time"

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

	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	allocationv1 "agones.dev/agones/pkg/apis/allocation/v1"
	autoscalingv1 "agones.dev/agones/pkg/apis/autoscaling/v1"
	"agones.dev/agones/pkg/util/runtime"
	e2e "agones.dev/agones/test/e2e/framework"
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

	defaultSyncIntervalFas := &autoscalingv1.FleetAutoscaler{}
	defaultSyncIntervalFas.ApplyDefaults()
	assert.Equal(t, defaultSyncIntervalFas.Spec.Sync.FixedInterval.Seconds, fas.Spec.Sync.FixedInterval.Seconds)
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
	err = wait.PollUntilContextTimeout(context.Background(), 1*time.Second, 30*time.Second, true, func(ctx context.Context) (bool, error) {
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
	err = wait.PollUntilContextTimeout(context.Background(), 1*time.Second, 5*time.Minute, true, func(ctx context.Context) (bool, error) {
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
	err = wait.PollUntilContextTimeout(context.Background(), time.Second, time.Minute, true, func(ctx context.Context) (bool, error) {
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
	err = wait.PollUntilContextTimeout(context.Background(), time.Second, time.Minute, true, func(ctx context.Context) (bool, error) {
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

func TestFleetAutoscalerTLSWebhook(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	// we hardcode 'default' namespace here because certificates above are generated to use this one
	defaultNS := "default"

	// certs
	caPem, _, caCert, caPrivKey, err := generateRootCA()
	require.NoError(t, err)
	clientCertPEM, clientCertPrivKeyPEM, err := generateLocalCert(caCert, caPrivKey)
	require.NoError(t, err)

	secr := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "autoscalersecret-",
		},
		Type: corev1.SecretTypeTLS,
		Data: make(map[string][]byte),
	}

	secr.Data[corev1.TLSCertKey] = clientCertPEM
	secr.Data[corev1.TLSPrivateKeyKey] = clientCertPrivKeyPEM

	secrets := framework.KubeClient.CoreV1().Secrets(defaultNS)
	secr, err = secrets.Create(ctx, secr.DeepCopy(), metav1.CreateOptions{})
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
	err = wait.PollUntilContextTimeout(context.Background(), 2*time.Second, time.Minute, true, func(ctx context.Context) (bool, error) {
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
		CABundle: caPem,
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
				Image:           "us-docker.pkg.dev/agones-images/examples/autoscaler-webhook:0.14",
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

// Instructions: https://agones.dev/site/docs/getting-started/create-webhook-fleetautoscaler/#chapter-2-configuring-https-fleetautoscaler-webhook-with-ca-bundle
// but also, credits/inspiration to https://github.com/kubernetes/autoscaler/blob/master/cluster-autoscaler/cloudprovider/aws/aws-sdk-go/awstesting/certificate_utils.go

func generateRootCA() (
	caPEM, caPrivKeyPEM []byte, caCert *x509.Certificate, caPrivKey *rsa.PrivateKey, err error,
) {
	caCert = &x509.Certificate{
		SerialNumber: big.NewInt(42),
		Subject: pkix.Name{
			Country:      []string{"US"},
			Organization: []string{"Agones"},
			CommonName:   "Test Root CA",
		},
		NotBefore: time.Now().Add(-time.Minute),
		NotAfter:  time.Now().AddDate(1, 0, 0),
		KeyUsage:  x509.KeyUsageCertSign | x509.KeyUsageCRLSign | x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{
			x509.ExtKeyUsageClientAuth,
			x509.ExtKeyUsageServerAuth,
		},
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	// Create CA private and public key
	caPrivKey, err = rsa.GenerateKey(cryptorand.Reader, 4096)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("failed generate CA RSA key, %w", err)
	}

	// Create CA certificate
	caBytes, err := x509.CreateCertificate(cryptorand.Reader, caCert, caCert, &caPrivKey.PublicKey, caPrivKey)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("failed generate CA certificate, %w", err)
	}

	// PEM encode CA certificate and private key
	var caPEMBuf bytes.Buffer
	err = pem.Encode(&caPEMBuf, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: caBytes,
	})
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("failed to endcode root PEM, %w", err)
	}

	var caPrivKeyPEMBuf bytes.Buffer
	err = pem.Encode(&caPrivKeyPEMBuf, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(caPrivKey),
	})
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("failed to endcode private root PEM, %w", err)
	}

	return caPEMBuf.Bytes(), caPrivKeyPEMBuf.Bytes(), caCert, caPrivKey, nil
}

func generateLocalCert(parentCert *x509.Certificate, parentPrivKey *rsa.PrivateKey) (
	certPEM, certPrivKeyPEM []byte, err error,
) {
	cert := &x509.Certificate{
		SerialNumber: big.NewInt(42),
		Subject: pkix.Name{
			Country:      []string{"US"},
			Organization: []string{"Agones"},
			CommonName:   "autoscaler-tls-service.default.svc",
		},
		NotBefore: time.Now().Add(-time.Minute),
		NotAfter:  time.Now().AddDate(1, 0, 0),
		ExtKeyUsage: []x509.ExtKeyUsage{
			x509.ExtKeyUsageClientAuth,
			x509.ExtKeyUsageServerAuth,
		},
		KeyUsage: x509.KeyUsageDigitalSignature,
		DNSNames: []string{"autoscaler-tls-service.default.svc"},
	}

	// Create server private and public key
	certPrivKey, err := rsa.GenerateKey(cryptorand.Reader, 4096)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate server RSA private key, %w", err)
	}

	// Create server certificate
	certBytes, err := x509.CreateCertificate(cryptorand.Reader, cert, parentCert, &certPrivKey.PublicKey, parentPrivKey)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate server certificate, %w", err)
	}

	// PEM encode certificate and private key
	var certPEMBuf bytes.Buffer
	err = pem.Encode(&certPEMBuf, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certBytes,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to endcode certificate pem, %w", err)
	}

	var certPrivKeyPEMBuf bytes.Buffer
	err = pem.Encode(&certPrivKeyPEMBuf, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(certPrivKey),
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to endcode private pem, %w", err)
	}

	return certPEMBuf.Bytes(), certPrivKeyPEMBuf.Bytes(), nil
}

func TestCounterAutoscaler(t *testing.T) {
	if !runtime.FeatureEnabled(runtime.FeatureCountsAndLists) {
		t.SkipNow()
	}
	t.Parallel()

	ctx := context.Background()
	client := framework.AgonesClient.AgonesV1()
	log := e2e.TestLogger(t)

	flt := defaultFleet(framework.Namespace)
	flt.Spec.Template.Spec.Counters = map[string]agonesv1.CounterStatus{
		"players": {
			Count:    7,  // AggregateCount 21
			Capacity: 10, // AggregateCapacity 30
		},
		"sessions": {
			Count:    0, // AggregateCount 0
			Capacity: 5, // AggregateCapacity 15
		},
	}

	flt, err := client.Fleets(framework.Namespace).Create(ctx, flt.DeepCopy(), metav1.CreateOptions{})
	require.NoError(t, err)
	defer client.Fleets(framework.Namespace).Delete(ctx, flt.ObjectMeta.Name, metav1.DeleteOptions{}) // nolint:errcheck
	framework.AssertFleetCondition(t, flt, e2e.FleetReadyCount(flt.Spec.Replicas))

	fleetautoscalers := framework.AgonesClient.AutoscalingV1().FleetAutoscalers(framework.Namespace)

	counterFas := func(f func(fap *autoscalingv1.FleetAutoscalerPolicy)) *autoscalingv1.FleetAutoscaler {
		fas := autoscalingv1.FleetAutoscaler{
			ObjectMeta: metav1.ObjectMeta{Name: flt.ObjectMeta.Name + "-counter-autoscaler", Namespace: framework.Namespace},
			Spec: autoscalingv1.FleetAutoscalerSpec{
				FleetName: flt.ObjectMeta.Name,
				Policy: autoscalingv1.FleetAutoscalerPolicy{
					Type: autoscalingv1.CounterPolicyType,
				},
				Sync: &autoscalingv1.FleetAutoscalerSync{
					Type: autoscalingv1.FixedIntervalSyncType,
					FixedInterval: autoscalingv1.FixedIntervalSync{
						Seconds: 1,
					},
				},
			},
		}
		f(&fas.Spec.Policy)
		return &fas
	}

	testCases := map[string]struct {
		fas          *autoscalingv1.FleetAutoscaler
		wantFasErr   bool
		wantReplicas int32
	}{
		"Scale Down Buffer Int": {
			fas: counterFas(func(fap *autoscalingv1.FleetAutoscalerPolicy) {
				fap.Counter = &autoscalingv1.CounterPolicy{
					Key:         "players",
					BufferSize:  intstr.FromInt(5), // Buffer refers to the available capacity (AggregateCapacity - AggregateCount)
					MinCapacity: 10,                // Min and MaxCapacity refer to the total capacity aggregated across the fleet, NOT the available capacity
					MaxCapacity: 100,
				}
			}),
			wantFasErr:   false,
			wantReplicas: 2,
		},
		"Scale Up Buffer Int": {
			fas: counterFas(func(fap *autoscalingv1.FleetAutoscalerPolicy) {
				fap.Counter = &autoscalingv1.CounterPolicy{
					Key:         "players",
					BufferSize:  intstr.FromInt(25),
					MinCapacity: 25,
					MaxCapacity: 100,
				}
			}),
			wantFasErr:   false,
			wantReplicas: 9,
		},
		"Scale Down to MaxCapacity": {
			fas: counterFas(func(fap *autoscalingv1.FleetAutoscalerPolicy) {
				fap.Counter = &autoscalingv1.CounterPolicy{
					Key:         "sessions",
					BufferSize:  intstr.FromInt(5),
					MinCapacity: 0,
					MaxCapacity: 5,
				}
			}),
			wantFasErr:   false,
			wantReplicas: 1,
		},
		"Scale Up to MinCapacity": {
			fas: counterFas(func(fap *autoscalingv1.FleetAutoscalerPolicy) {
				fap.Counter = &autoscalingv1.CounterPolicy{
					Key:         "sessions",
					BufferSize:  intstr.FromInt(1),
					MinCapacity: 30,
					MaxCapacity: 100,
				}
			}),
			wantFasErr:   false,
			wantReplicas: 6,
		},
		"Cannot scale up (MaxCapacity)": {
			fas: counterFas(func(fap *autoscalingv1.FleetAutoscalerPolicy) {
				fap.Counter = &autoscalingv1.CounterPolicy{
					Key:         "players",
					BufferSize:  intstr.FromInt(10),
					MinCapacity: 10,
					MaxCapacity: 30,
				}
			}),
			wantFasErr:   false,
			wantReplicas: 3,
		},
		"Cannot scale down (MinCapacity)": {
			fas: counterFas(func(fap *autoscalingv1.FleetAutoscalerPolicy) {
				fap.Counter = &autoscalingv1.CounterPolicy{
					Key:         "sessions",
					BufferSize:  intstr.FromInt(5),
					MinCapacity: 15,
					MaxCapacity: 100,
				}
			}),
			wantFasErr:   false,
			wantReplicas: 3,
		},
		"Buffer Greater than MinCapacity invalid FAS": {
			fas: counterFas(func(fap *autoscalingv1.FleetAutoscalerPolicy) {
				fap.Counter = &autoscalingv1.CounterPolicy{
					Key:         "players",
					BufferSize:  intstr.FromInt(25),
					MinCapacity: 10,
					MaxCapacity: 100,
				}
			}),
			wantFasErr: true,
		},
	}
	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {

			fas, err := fleetautoscalers.Create(ctx, testCase.fas, metav1.CreateOptions{})
			if testCase.wantFasErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)

			framework.AssertFleetCondition(t, flt, e2e.FleetReadyCount(testCase.wantReplicas))
			fleetautoscalers.Delete(ctx, fas.ObjectMeta.Name, metav1.DeleteOptions{}) // nolint:errcheck

			// Return to starting 3 replicas
			framework.ScaleFleet(t, log, flt, 3)
			framework.AssertFleetCondition(t, flt, e2e.FleetReadyCount(3))
		})
	}
}

func TestCounterAutoscalerAllocated(t *testing.T) {
	if !runtime.FeatureEnabled(runtime.FeatureCountsAndLists) {
		t.SkipNow()
	}
	t.Parallel()

	ctx := context.Background()
	client := framework.AgonesClient.AgonesV1()
	log := e2e.TestLogger(t)

	defaultFlt := defaultFleet(framework.Namespace)
	defaultFlt.Spec.Template.Spec.Counters = map[string]agonesv1.CounterStatus{
		"players": {
			Count:    7,  // AggregateCount 21
			Capacity: 10, // AggregateCapacity 30
		},
	}

	fleetautoscalers := framework.AgonesClient.AutoscalingV1().FleetAutoscalers(framework.Namespace)

	testCases := map[string]struct {
		fas             autoscalingv1.CounterPolicy
		wantAllocatedGs int32 // Must be >= 0 && <= 3
		wantReadyGs     int32
	}{
		"Scale Down Buffer Percent": {
			fas: autoscalingv1.CounterPolicy{
				Key:         "players",
				BufferSize:  intstr.FromString("5%"),
				MinCapacity: 10,
				MaxCapacity: 100,
			},
			wantAllocatedGs: 0,
			wantReadyGs:     1,
		},
		"Scale Up Buffer Percent": {
			fas: autoscalingv1.CounterPolicy{
				Key:         "players",
				BufferSize:  intstr.FromString("40%"),
				MinCapacity: 10,
				MaxCapacity: 100,
			},
			wantAllocatedGs: 3,
			wantReadyGs:     2,
		},
	}
	// nolint:dupl  // Linter errors on lines are duplicate of TestListAutoscalerAllocated
	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			flt, err := client.Fleets(framework.Namespace).Create(ctx, defaultFlt.DeepCopy(), metav1.CreateOptions{})
			require.NoError(t, err)
			defer client.Fleets(framework.Namespace).Delete(ctx, flt.ObjectMeta.Name, metav1.DeleteOptions{}) // nolint:errcheck
			framework.AssertFleetCondition(t, flt, e2e.FleetReadyCount(flt.Spec.Replicas))

			gsa := allocationv1.GameServerAllocation{
				Spec: allocationv1.GameServerAllocationSpec{
					Selectors: []allocationv1.GameServerSelector{
						{LabelSelector: metav1.LabelSelector{
							MatchLabels: map[string]string{agonesv1.FleetNameLabel: flt.ObjectMeta.Name}}},
					}}}

			// Allocate game servers, as Buffer Percent scales up (or down) based on allocated aggregate capacity
			for i := int32(0); i < testCase.wantAllocatedGs; i++ {
				_, err := framework.AgonesClient.AllocationV1().GameServerAllocations(flt.ObjectMeta.Namespace).Create(ctx, gsa.DeepCopy(), metav1.CreateOptions{})
				require.NoError(t, err)
			}
			framework.AssertFleetCondition(t, flt, func(entry *logrus.Entry, fleet *agonesv1.Fleet) bool {
				log.WithField("fleet", fmt.Sprintf("%+v", fleet.Status)).Info("Checking for game server allocations")
				return fleet.Status.AllocatedReplicas == testCase.wantAllocatedGs
			})

			counterFas := &autoscalingv1.FleetAutoscaler{
				ObjectMeta: metav1.ObjectMeta{Name: flt.ObjectMeta.Name + "-counter-autoscaler", Namespace: framework.Namespace},
				Spec: autoscalingv1.FleetAutoscalerSpec{
					FleetName: flt.ObjectMeta.Name,
					Policy: autoscalingv1.FleetAutoscalerPolicy{
						Type:    autoscalingv1.CounterPolicyType,
						Counter: &testCase.fas,
					},
					Sync: &autoscalingv1.FleetAutoscalerSync{
						Type: autoscalingv1.FixedIntervalSyncType,
						FixedInterval: autoscalingv1.FixedIntervalSync{
							Seconds: 1,
						},
					},
				},
			}

			fas, err := fleetautoscalers.Create(ctx, counterFas, metav1.CreateOptions{})
			assert.NoError(t, err)
			defer fleetautoscalers.Delete(ctx, fas.ObjectMeta.Name, metav1.DeleteOptions{}) // nolint:errcheck

			framework.AssertFleetCondition(t, flt, func(entry *logrus.Entry, fleet *agonesv1.Fleet) bool {
				return fleet.Status.AllocatedReplicas == testCase.wantAllocatedGs && fleet.Status.ReadyReplicas == testCase.wantReadyGs
			})
		})
	}
}

func TestListAutoscaler(t *testing.T) {
	if !runtime.FeatureEnabled(runtime.FeatureCountsAndLists) {
		t.SkipNow()
	}
	t.Parallel()

	ctx := context.Background()
	client := framework.AgonesClient.AgonesV1()
	log := e2e.TestLogger(t)

	flt := defaultFleet(framework.Namespace)
	flt.Spec.Template.Spec.Lists = map[string]agonesv1.ListStatus{
		"games": {
			Values:   []string{"game1", "game2", "game3"}, // AggregateCount 9
			Capacity: 5,                                   // AggregateCapacity 15
		},
	}

	flt, err := client.Fleets(framework.Namespace).Create(ctx, flt.DeepCopy(), metav1.CreateOptions{})
	require.NoError(t, err)
	defer client.Fleets(framework.Namespace).Delete(ctx, flt.ObjectMeta.Name, metav1.DeleteOptions{}) // nolint:errcheck
	framework.AssertFleetCondition(t, flt, e2e.FleetReadyCount(flt.Spec.Replicas))

	fleetautoscalers := framework.AgonesClient.AutoscalingV1().FleetAutoscalers(framework.Namespace)

	listFas := func(f func(fap *autoscalingv1.FleetAutoscalerPolicy)) *autoscalingv1.FleetAutoscaler {
		fas := autoscalingv1.FleetAutoscaler{
			ObjectMeta: metav1.ObjectMeta{Name: flt.ObjectMeta.Name + "-list-autoscaler", Namespace: framework.Namespace},
			Spec: autoscalingv1.FleetAutoscalerSpec{
				FleetName: flt.ObjectMeta.Name,
				Policy: autoscalingv1.FleetAutoscalerPolicy{
					Type: autoscalingv1.ListPolicyType,
				},
				Sync: &autoscalingv1.FleetAutoscalerSync{
					Type: autoscalingv1.FixedIntervalSyncType,
					FixedInterval: autoscalingv1.FixedIntervalSync{
						Seconds: 1,
					},
				},
			},
		}
		f(&fas.Spec.Policy)
		return &fas
	}
	testCases := map[string]struct {
		fas          *autoscalingv1.FleetAutoscaler
		wantFasErr   bool
		wantReplicas int32
	}{
		"Scale Down to Minimum 1 Replica": {
			fas: listFas(func(fap *autoscalingv1.FleetAutoscalerPolicy) {
				fap.List = &autoscalingv1.ListPolicy{
					Key:         "games",
					BufferSize:  intstr.FromInt(2),
					MinCapacity: 0,
					MaxCapacity: 3,
				}
			}),
			wantFasErr:   false,
			wantReplicas: 1, // Count:3 Capacity:5
		},
		"Scale Down to Buffer": {
			fas: listFas(func(fap *autoscalingv1.FleetAutoscalerPolicy) {
				fap.List = &autoscalingv1.ListPolicy{
					Key:         "games",
					BufferSize:  intstr.FromInt(3),
					MinCapacity: 0,
					MaxCapacity: 5,
				}
			}),
			wantFasErr:   false,
			wantReplicas: 2, // Count:6 Capacity:10
		},
		"MinCapacity Must Be Greater Than Zero Percentage Buffer": {
			fas: listFas(func(fap *autoscalingv1.FleetAutoscalerPolicy) {
				fap.List = &autoscalingv1.ListPolicy{
					Key:         "games",
					BufferSize:  intstr.FromString("50%"),
					MinCapacity: 0,
					MaxCapacity: 100,
				}
			}),
			wantFasErr:   true,
			wantReplicas: 3,
		},
		"Scale Up to MinCapacity": {
			fas: listFas(func(fap *autoscalingv1.FleetAutoscalerPolicy) {
				fap.List = &autoscalingv1.ListPolicy{
					Key:         "games",
					BufferSize:  intstr.FromInt(3),
					MinCapacity: 16,
					MaxCapacity: 100,
				}
			}),
			wantFasErr:   false,
			wantReplicas: 4, // Count:12 Capacity:20
		},
		"Scale Down to MinCapacity": {
			fas: listFas(func(fap *autoscalingv1.FleetAutoscalerPolicy) {
				fap.List = &autoscalingv1.ListPolicy{
					Key:         "games",
					BufferSize:  intstr.FromInt(1),
					MinCapacity: 10,
					MaxCapacity: 100,
				}
			}),
			wantFasErr:   false,
			wantReplicas: 2, // Count:6 Capacity:10
		},
		"MinCapacity Less Than Buffer Invalid": {
			fas: listFas(func(fap *autoscalingv1.FleetAutoscalerPolicy) {
				fap.List = &autoscalingv1.ListPolicy{
					Key:         "games",
					BufferSize:  intstr.FromInt(15),
					MinCapacity: 5,
					MaxCapacity: 25,
				}
			}),
			wantFasErr:   true,
			wantReplicas: 3,
		},
		"Scale Up to Buffer": {
			fas: listFas(func(fap *autoscalingv1.FleetAutoscalerPolicy) {
				fap.List = &autoscalingv1.ListPolicy{
					Key:         "games",
					BufferSize:  intstr.FromInt(15),
					MinCapacity: 15,
					MaxCapacity: 100,
				}
			}),
			wantFasErr:   false,
			wantReplicas: 8, // Count:24 Capacity:40
		},
		"Scale Up to MaxCapacity": {
			fas: listFas(func(fap *autoscalingv1.FleetAutoscalerPolicy) {
				fap.List = &autoscalingv1.ListPolicy{
					Key:         "games",
					BufferSize:  intstr.FromInt(15),
					MinCapacity: 15,
					MaxCapacity: 25,
				}
			}),
			wantFasErr:   false,
			wantReplicas: 5, // Count:15 Capacity:25
		},
	}
	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {

			fas, err := fleetautoscalers.Create(ctx, testCase.fas, metav1.CreateOptions{})
			if testCase.wantFasErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)

			framework.AssertFleetCondition(t, flt, e2e.FleetReadyCount(testCase.wantReplicas))
			fleetautoscalers.Delete(ctx, fas.ObjectMeta.Name, metav1.DeleteOptions{}) // nolint:errcheck

			// Return to starting 3 replicas
			framework.ScaleFleet(t, log, flt, 3)
			framework.AssertFleetCondition(t, flt, e2e.FleetReadyCount(3))
		})
	}
}

func TestListAutoscalerAllocated(t *testing.T) {
	if !runtime.FeatureEnabled(runtime.FeatureCountsAndLists) {
		t.SkipNow()
	}
	t.Parallel()

	ctx := context.Background()
	client := framework.AgonesClient.AgonesV1()
	log := e2e.TestLogger(t)

	defaultFlt := defaultFleet(framework.Namespace)
	defaultFlt.Spec.Template.Spec.Lists = map[string]agonesv1.ListStatus{
		"gamers": {
			Values:   []string{},
			Capacity: 6, // AggregateCapacity 18
		},
	}

	fleetautoscalers := framework.AgonesClient.AutoscalingV1().FleetAutoscalers(framework.Namespace)

	testCases := map[string]struct {
		fas                  autoscalingv1.ListPolicy
		wantAllocatedGs      int32 // Must be >= 0 && <= 3
		wantReadyGs          int32
		wantSecondAllocation int32 // Must be <= wantReadyGs
		wantSecondReady      int32
	}{
		"Scale Down Buffer Percent": {
			fas: autoscalingv1.ListPolicy{
				Key:         "gamers",
				BufferSize:  intstr.FromString("50%"),
				MinCapacity: 6,
				MaxCapacity: 60,
			},
			wantAllocatedGs: 0,
			wantReadyGs:     1,
		},
		"Scale Up Buffer Percent": {
			fas: autoscalingv1.ListPolicy{
				Key:         "gamers",
				BufferSize:  intstr.FromString("50%"),
				MinCapacity: 6,
				MaxCapacity: 60,
			},
			wantAllocatedGs:      3,
			wantReadyGs:          1,
			wantSecondAllocation: 1,
			wantSecondReady:      2,
		},
		"Scales Down to Number of Game Servers Allocated": {
			fas: autoscalingv1.ListPolicy{
				Key:         "gamers",
				BufferSize:  intstr.FromInt(2),
				MinCapacity: 6,
				MaxCapacity: 60,
			},
			wantAllocatedGs: 2,
			wantReadyGs:     0,
		},
	}
	// nolint:dupl  // Linter errors on lines are duplicate of TestCounterAutoscalerAllocated
	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			flt, err := client.Fleets(framework.Namespace).Create(ctx, defaultFlt.DeepCopy(), metav1.CreateOptions{})
			require.NoError(t, err)
			defer client.Fleets(framework.Namespace).Delete(ctx, flt.ObjectMeta.Name, metav1.DeleteOptions{}) // nolint:errcheck
			framework.AssertFleetCondition(t, flt, e2e.FleetReadyCount(flt.Spec.Replicas))

			// Adds 4 gamers to each allocated gameserver.
			gsa := allocationv1.GameServerAllocation{
				Spec: allocationv1.GameServerAllocationSpec{
					Selectors: []allocationv1.GameServerSelector{
						{LabelSelector: metav1.LabelSelector{
							MatchLabels: map[string]string{agonesv1.FleetNameLabel: flt.ObjectMeta.Name}}},
					},
					Lists: map[string]allocationv1.ListAction{
						"gamers": {
							AddValues: []string{"gamer1", "gamer2", "gamer3", "gamer4"},
						}}}}

			// Allocate game servers, as Buffer Percent scales up (or down) based on allocated aggregate capacity
			for i := int32(0); i < testCase.wantAllocatedGs; i++ {
				_, err := framework.AgonesClient.AllocationV1().GameServerAllocations(flt.ObjectMeta.Namespace).Create(ctx, gsa.DeepCopy(), metav1.CreateOptions{})
				require.NoError(t, err)
			}
			framework.AssertFleetCondition(t, flt, func(entry *logrus.Entry, fleet *agonesv1.Fleet) bool {
				log.WithField("fleet", fmt.Sprintf("%+v", fleet.Status)).Info("Checking for game server allocations")
				return fleet.Status.AllocatedReplicas == testCase.wantAllocatedGs
			})

			listFas := &autoscalingv1.FleetAutoscaler{
				ObjectMeta: metav1.ObjectMeta{Name: flt.ObjectMeta.Name + "-list-autoscaler", Namespace: framework.Namespace},
				Spec: autoscalingv1.FleetAutoscalerSpec{
					FleetName: flt.ObjectMeta.Name,
					Policy: autoscalingv1.FleetAutoscalerPolicy{
						Type: autoscalingv1.ListPolicyType,
						List: &testCase.fas,
					},
					Sync: &autoscalingv1.FleetAutoscalerSync{
						Type: autoscalingv1.FixedIntervalSyncType,
						FixedInterval: autoscalingv1.FixedIntervalSync{
							Seconds: 1,
						},
					},
				},
			}

			fas, err := fleetautoscalers.Create(ctx, listFas, metav1.CreateOptions{})
			assert.NoError(t, err)
			defer fleetautoscalers.Delete(ctx, fas.ObjectMeta.Name, metav1.DeleteOptions{}) // nolint:errcheck

			framework.AssertFleetCondition(t, flt, func(entry *logrus.Entry, fleet *agonesv1.Fleet) bool {
				return fleet.Status.AllocatedReplicas == testCase.wantAllocatedGs && fleet.Status.ReadyReplicas == testCase.wantReadyGs
			})

			// If we're not looking for a second gameserver allocation action, exit test early.
			if testCase.wantSecondAllocation == 0 {
				return
			}

			for i := int32(0); i < testCase.wantSecondAllocation; i++ {
				_, err := framework.AgonesClient.AllocationV1().GameServerAllocations(flt.ObjectMeta.Namespace).Create(ctx, gsa.DeepCopy(), metav1.CreateOptions{})
				require.NoError(t, err)
			}

			framework.AssertFleetCondition(t, flt, func(entry *logrus.Entry, fleet *agonesv1.Fleet) bool {
				log.WithField("fleet", fmt.Sprintf("%+v", fleet.Status)).Info("Checking for second game server allocations")
				return fleet.Status.AllocatedReplicas == (testCase.wantAllocatedGs+testCase.wantSecondAllocation) &&
					fleet.Status.ReadyReplicas == testCase.wantSecondReady
			})

		})
	}
}

func TestListAutoscalerWithSDKMethods(t *testing.T) {
	if !runtime.FeatureEnabled(runtime.FeatureCountsAndLists) {
		t.SkipNow()
	}
	t.Parallel()

	ctx := context.Background()
	client := framework.AgonesClient.AgonesV1()

	defaultFlt := defaultFleet(framework.Namespace)
	defaultFlt.Spec.Template.Spec.Lists = map[string]agonesv1.ListStatus{
		"sessions": {
			Values:   []string{"session1", "session2"}, // AggregateCount 6
			Capacity: 4,                                // AggregateCapacity 12
		},
	}

	fleetautoscalers := framework.AgonesClient.AutoscalingV1().FleetAutoscalers(framework.Namespace)

	testCases := map[string]struct {
		fas           autoscalingv1.ListPolicy
		order         string // Priority order Ascending or Descending for fleet ready replica deletion
		msg           string // See agones/examples/simple-game-server/README for list of commands
		startReplicas int32  // After applying autoscaler policy but before sending update message
		wantReplicas  int32  // After applying autoscaler policy and sending update message
	}{
		"Scale Up to Buffer": {
			fas: autoscalingv1.ListPolicy{
				Key:         "sessions",
				BufferSize:  intstr.FromInt(10),
				MinCapacity: 12,
				MaxCapacity: 400,
			},
			order:         agonesv1.GameServerPriorityDescending,
			msg:           "APPEND_LIST_VALUE sessions session0",
			startReplicas: 5,
			wantReplicas:  6,
		},
		"Scale Down to Buffer": {
			fas: autoscalingv1.ListPolicy{
				Key:         "sessions",
				BufferSize:  intstr.FromInt(3),
				MinCapacity: 3,
				MaxCapacity: 400,
			},
			msg:           "DELETE_LIST_VALUE sessions session1",
			order:         agonesv1.GameServerPriorityAscending,
			startReplicas: 2,
			wantReplicas:  1,
		},
	}
	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			defaultFlt.Spec.Priorities = []agonesv1.Priority{
				{
					Type:  agonesv1.GameServerPriorityList,
					Key:   "sessions",
					Order: testCase.order,
				},
			}
			flt, err := client.Fleets(framework.Namespace).Create(ctx, defaultFlt.DeepCopy(), metav1.CreateOptions{})
			require.NoError(t, err)
			defer client.Fleets(framework.Namespace).Delete(ctx, flt.ObjectMeta.Name, metav1.DeleteOptions{}) // nolint:errcheck
			framework.AssertFleetCondition(t, flt, e2e.FleetReadyCount(flt.Spec.Replicas))

			listFas := &autoscalingv1.FleetAutoscaler{
				ObjectMeta: metav1.ObjectMeta{Name: flt.ObjectMeta.Name + "-list-autoscaler", Namespace: framework.Namespace},
				Spec: autoscalingv1.FleetAutoscalerSpec{
					FleetName: flt.ObjectMeta.Name,
					Policy: autoscalingv1.FleetAutoscalerPolicy{
						Type: autoscalingv1.ListPolicyType,
						List: &testCase.fas,
					},
					Sync: &autoscalingv1.FleetAutoscalerSync{
						Type: autoscalingv1.FixedIntervalSyncType,
						FixedInterval: autoscalingv1.FixedIntervalSync{
							Seconds: 1,
						},
					},
				},
			}

			fas, err := fleetautoscalers.Create(ctx, listFas, metav1.CreateOptions{})
			assert.NoError(t, err)
			defer fleetautoscalers.Delete(ctx, fas.ObjectMeta.Name, metav1.DeleteOptions{}) // nolint:errcheck

			// Wait until autoscaler has first re-sized before getting the list of gameservers
			framework.AssertFleetCondition(t, flt, e2e.FleetReadyCount(testCase.startReplicas))

			gameservers, err := framework.ListGameServersFromFleet(flt)
			assert.NoError(t, err)

			gs := &gameservers[1]
			logrus.WithField("command", testCase.msg).WithField("gs", gs.ObjectMeta.Name).Info(name)
			_, err = framework.SendGameServerUDP(t, gs, testCase.msg)
			require.NoError(t, err)

			framework.AssertFleetCondition(t, flt, e2e.FleetReadyCount(testCase.wantReplicas))
		})
	}
}

func TestScheduleAutoscaler(t *testing.T) {
	if !runtime.FeatureEnabled(runtime.FeatureScheduledAutoscaler) {
		t.SkipNow()
	}
	t.Parallel()
	ctx := context.Background()
	log := e2e.TestLogger(t)

	stable := framework.AgonesClient.AgonesV1()
	fleets := stable.Fleets(framework.Namespace)
	flt, err := fleets.Create(ctx, defaultFleet(framework.Namespace), metav1.CreateOptions{})
	if assert.NoError(t, err) {
		defer fleets.Delete(context.Background(), flt.ObjectMeta.Name, metav1.DeleteOptions{}) // nolint:errcheck
	}

	framework.AssertFleetCondition(t, flt, e2e.FleetReadyCount(flt.Spec.Replicas))

	fleetautoscalers := framework.AgonesClient.AutoscalingV1().FleetAutoscalers(framework.Namespace)

	// Active Cron Schedule (e.g. run after 1 * * * *, which is the after the first minute of the hour)
	scheduleAutoscaler := defaultAutoscalerSchedule(t, flt)
	scheduleAutoscaler.Spec.Policy.Schedule.ActivePeriod.StartCron = nextCronMinute(time.Now())
	fas, err := fleetautoscalers.Create(ctx, scheduleAutoscaler, metav1.CreateOptions{})
	assert.NoError(t, err)

	framework.AssertFleetCondition(t, flt, e2e.FleetReadyCount(5))
	fleetautoscalers.Delete(ctx, fas.ObjectMeta.Name, metav1.DeleteOptions{}) // nolint:errcheck

	// Return to starting 3 replicas
	framework.ScaleFleet(t, log, flt, 3)
	framework.AssertFleetCondition(t, flt, e2e.FleetReadyCount(3))

	// Between Active Period Cron Schedule (e.g. run between 1-2 * * * *, which is between the first minute and second minute of the hour)
	scheduleAutoscaler = defaultAutoscalerSchedule(t, flt)
	scheduleAutoscaler.Spec.Policy.Schedule.ActivePeriod.StartCron = nextCronMinuteBetween(time.Now())
	fas, err = fleetautoscalers.Create(ctx, scheduleAutoscaler, metav1.CreateOptions{})
	assert.NoError(t, err)

	framework.AssertFleetCondition(t, flt, e2e.FleetReadyCount(5))
	fleetautoscalers.Delete(ctx, fas.ObjectMeta.Name, metav1.DeleteOptions{}) // nolint:errcheck
}

func TestChainAutoscaler(t *testing.T) {
	if !runtime.FeatureEnabled(runtime.FeatureScheduledAutoscaler) {
		t.SkipNow()
	}
	t.Parallel()
	ctx := context.Background()
	log := e2e.TestLogger(t)

	stable := framework.AgonesClient.AgonesV1()
	fleets := stable.Fleets(framework.Namespace)
	flt, err := fleets.Create(ctx, defaultFleet(framework.Namespace), metav1.CreateOptions{})
	if assert.NoError(t, err) {
		defer fleets.Delete(context.Background(), flt.ObjectMeta.Name, metav1.DeleteOptions{}) // nolint:errcheck
	}

	framework.AssertFleetCondition(t, flt, e2e.FleetReadyCount(flt.Spec.Replicas))

	fleetautoscalers := framework.AgonesClient.AutoscalingV1().FleetAutoscalers(framework.Namespace)

	// 1st Schedule Inactive, 2nd Schedule Active - 30 seconds (Fallthrough)
	chainAutoscaler := defaultAutoscalerChain(t, flt)
	fas, err := fleetautoscalers.Create(ctx, chainAutoscaler, metav1.CreateOptions{})
	assert.NoError(t, err)

	// Verify only the second schedule ran
	framework.AssertFleetCondition(t, flt, e2e.FleetReadyCount(4))
	fleetautoscalers.Delete(ctx, fas.ObjectMeta.Name, metav1.DeleteOptions{}) // nolint:errcheck

	// Return to starting 3 replicas
	framework.ScaleFleet(t, log, flt, 3)
	framework.AssertFleetCondition(t, flt, e2e.FleetReadyCount(3))

	// 2 Active Schedules back to back - 1 minute (Fallthrough)
	chainAutoscaler = defaultAutoscalerChain(t, flt)
	currentTime := time.Now()

	// First schedule runs for 1 minute
	chainAutoscaler.Spec.Policy.Chain[0].Schedule.ActivePeriod.StartCron = nextCronMinute(currentTime)
	chainAutoscaler.Spec.Policy.Chain[0].Schedule.ActivePeriod.Duration = "1m"

	// Second schedule runs 1 minute after the first schedule
	oneMinute := mustParseDuration(t, "1m")
	chainAutoscaler.Spec.Policy.Chain[0].Schedule.ActivePeriod.StartCron = nextCronMinute(currentTime.Add(oneMinute))
	chainAutoscaler.Spec.Policy.Chain[1].Schedule.ActivePeriod.Duration = "5m"

	fas, err = fleetautoscalers.Create(ctx, chainAutoscaler, metav1.CreateOptions{})
	assert.NoError(t, err)

	// Verify the first schedule has been applied
	framework.AssertFleetCondition(t, flt, e2e.FleetReadyCount(10))
	// Verify the second schedule has been applied
	framework.AssertFleetCondition(t, flt, e2e.FleetReadyCount(4))

	fleetautoscalers.Delete(ctx, fas.ObjectMeta.Name, metav1.DeleteOptions{}) // nolint:errcheck
}

// defaultAutoscalerSchedule returns a default scheduled autoscaler for testing.
func defaultAutoscalerSchedule(t *testing.T, f *agonesv1.Fleet) *autoscalingv1.FleetAutoscaler {
	return &autoscalingv1.FleetAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Name:      f.ObjectMeta.Name + "-scheduled-autoscaler",
			Namespace: framework.Namespace,
		},
		Spec: autoscalingv1.FleetAutoscalerSpec{
			FleetName: f.ObjectMeta.Name,
			Policy: autoscalingv1.FleetAutoscalerPolicy{
				Type: autoscalingv1.SchedulePolicyType,
				Schedule: &autoscalingv1.SchedulePolicy{
					Between: autoscalingv1.Between{
						Start: currentTimePlusDuration(t, "1s"),
						End:   currentTimePlusDuration(t, "1m"),
					},
					ActivePeriod: autoscalingv1.ActivePeriod{
						Timezone:  "UTC",
						StartCron: "* * * * *",
						Duration:  "",
					},
					Policy: autoscalingv1.FleetAutoscalerPolicy{
						Type: autoscalingv1.BufferPolicyType,
						Buffer: &autoscalingv1.BufferPolicy{
							BufferSize:  intstr.FromInt(5),
							MinReplicas: 5,
							MaxReplicas: 12,
						},
					},
				},
			},
			Sync: &autoscalingv1.FleetAutoscalerSync{
				Type: autoscalingv1.FixedIntervalSyncType,
				FixedInterval: autoscalingv1.FixedIntervalSync{
					Seconds: 5,
				},
			},
		},
	}
}

// defaultAutoscalerChain returns a default chain autoscaler for testing.
func defaultAutoscalerChain(t *testing.T, f *agonesv1.Fleet) *autoscalingv1.FleetAutoscaler {
	return &autoscalingv1.FleetAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Name:      f.ObjectMeta.Name + "-chain-autoscaler",
			Namespace: framework.Namespace,
		},
		Spec: autoscalingv1.FleetAutoscalerSpec{
			FleetName: f.ObjectMeta.Name,
			Policy: autoscalingv1.FleetAutoscalerPolicy{
				Type: autoscalingv1.ChainPolicyType,
				Chain: autoscalingv1.ChainPolicy{
					{
						ID: "schedule-1",
						FleetAutoscalerPolicy: autoscalingv1.FleetAutoscalerPolicy{
							Type: autoscalingv1.SchedulePolicyType,
							Schedule: &autoscalingv1.SchedulePolicy{
								Between: autoscalingv1.Between{
									Start: currentTimePlusDuration(t, "1s"),
									End:   currentTimePlusDuration(t, "2m"),
								},
								ActivePeriod: autoscalingv1.ActivePeriod{
									Timezone:  "",
									StartCron: inactiveCronSchedule(time.Now()),
									Duration:  "1m",
								},
								Policy: autoscalingv1.FleetAutoscalerPolicy{
									Type: autoscalingv1.BufferPolicyType,
									Buffer: &autoscalingv1.BufferPolicy{
										BufferSize:  intstr.FromInt(10),
										MinReplicas: 10,
										MaxReplicas: 20,
									},
								},
							},
						},
					},
					{
						ID: "schedule-2",
						FleetAutoscalerPolicy: autoscalingv1.FleetAutoscalerPolicy{
							Type: autoscalingv1.SchedulePolicyType,
							Schedule: &autoscalingv1.SchedulePolicy{
								Between: autoscalingv1.Between{
									Start: currentTimePlusDuration(t, "1s"),
									End:   currentTimePlusDuration(t, "5m"),
								},
								ActivePeriod: autoscalingv1.ActivePeriod{
									Timezone:  "",
									StartCron: nextCronMinute(time.Now()),
									Duration:  "",
								},
								Policy: autoscalingv1.FleetAutoscalerPolicy{
									Type: autoscalingv1.BufferPolicyType,
									Buffer: &autoscalingv1.BufferPolicy{
										BufferSize:  intstr.FromInt(4),
										MinReplicas: 3,
										MaxReplicas: 7,
									},
								},
							},
						},
					},
				},
			},
			Sync: &autoscalingv1.FleetAutoscalerSync{
				Type: autoscalingv1.FixedIntervalSyncType,
				FixedInterval: autoscalingv1.FixedIntervalSync{
					Seconds: 5,
				},
			},
		},
	}
}

// inactiveCronSchedule returns the time 3 minutes ago
// e.g. if the current time is 12:00, this method will return "57 * * * *"
// meaning 3 minutes before 12:00
func inactiveCronSchedule(currentTime time.Time) string {
	prevMinute := currentTime.Add(time.Minute * -3).Minute()
	return fmt.Sprintf("%d * * * *", prevMinute)
}

// nextCronMinute returns the very next minute in
// e.g. if the current time is 12:00, this method will return "1 * * * *"
// meaning after 12:01
func nextCronMinute(currentTime time.Time) string {
	nextMinute := currentTime.Add(time.Minute).Minute()
	return fmt.Sprintf("%d * * * *", nextMinute)
}

// nextCronMinuteBetween returns the minute between the very next minute
// e.g. if the current time is 12:00, this method will return "1-2 * * * *"
// meaning between 12:01 - 12:02
func nextCronMinuteBetween(currentTime time.Time) string {
	nextMinute := currentTime.Add(time.Minute).Minute()
	secondMinute := currentTime.Add(2 * time.Minute).Minute()
	return fmt.Sprintf("%d-%d * * * *", nextMinute, secondMinute)
}

// Parse a duration string and return a duration struct
func mustParseDuration(t *testing.T, duration string) time.Duration {
	d, err := time.ParseDuration(duration)
	assert.Nil(t, err)
	return d
}

// Parse a time string and return a metav1.Time
func currentTimePlusDuration(t *testing.T, duration string) metav1.Time {
	d := mustParseDuration(t, duration)
	currentTimePlusDuration := time.Now().Add(d)
	return metav1.NewTime(currentTimePlusDuration)
}
