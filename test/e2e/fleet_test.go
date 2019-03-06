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
	"sync"
	"testing"
	"time"

	"agones.dev/agones/pkg/apis/stable/v1alpha1"
	e2e "agones.dev/agones/test/e2e/framework"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	v1betaext "k8s.io/api/extensions/v1beta1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/retry"
)

const (
	key           = "test-state"
	red           = "red"
	green         = "green"
	replicasCount = 3
)

func TestCreateFleetAndFleetAllocate(t *testing.T) {
	t.Parallel()

	fixtures := []v1alpha1.SchedulingStrategy{v1alpha1.Packed, v1alpha1.Distributed}

	for _, strategy := range fixtures {
		t.Run(string(strategy), func(t *testing.T) {
			fleets := framework.AgonesClient.StableV1alpha1().Fleets(defaultNs)
			fleet := defaultFleet()
			fleet.Spec.Scheduling = strategy
			flt, err := fleets.Create(fleet)
			if assert.Nil(t, err) {
				defer fleets.Delete(flt.ObjectMeta.Name, nil) // nolint:errcheck
			}

			framework.WaitForFleetCondition(t, flt, e2e.FleetReadyCount(flt.Spec.Replicas))

			fa := &v1alpha1.FleetAllocation{
				ObjectMeta: metav1.ObjectMeta{GenerateName: "allocation-", Namespace: defaultNs},
				Spec: v1alpha1.FleetAllocationSpec{
					FleetName: flt.ObjectMeta.Name,
				},
			}

			fa, err = framework.AgonesClient.StableV1alpha1().FleetAllocations(defaultNs).Create(fa)
			assert.Nil(t, err)
			assert.Equal(t, v1alpha1.GameServerStateAllocated, fa.Status.GameServer.Status.State)
		})

	}
}

// Can't allocate more GameServers if a fleet is fully used.
func TestCreateFullFleetAndCantFleetAllocate(t *testing.T) {
	t.Parallel()

	fixtures := []v1alpha1.SchedulingStrategy{v1alpha1.Packed, v1alpha1.Distributed}

	for _, strategy := range fixtures {
		t.Run(string(strategy), func(t *testing.T) {
			fleets := framework.AgonesClient.StableV1alpha1().Fleets(defaultNs)
			fleet := defaultFleet()
			fleet.Spec.Scheduling = strategy
			flt, err := fleets.Create(fleet)
			if assert.Nil(t, err) {
				defer fleets.Delete(flt.ObjectMeta.Name, nil) // nolint:errcheck
			}

			framework.WaitForFleetCondition(t, flt, e2e.FleetReadyCount(flt.Spec.Replicas))

			fa := &v1alpha1.FleetAllocation{
				ObjectMeta: metav1.ObjectMeta{GenerateName: "allocation-", Namespace: defaultNs},
				Spec: v1alpha1.FleetAllocationSpec{
					FleetName: flt.ObjectMeta.Name,
				},
			}

			for i := 0; i < replicasCount; i++ {
				fa2, err := framework.AgonesClient.StableV1alpha1().FleetAllocations(defaultNs).Create(fa)
				assert.Nil(t, err)
				assert.Equal(t, v1alpha1.GameServerStateAllocated, fa2.Status.GameServer.Status.State)
			}

			framework.WaitForFleetCondition(t, flt, func(fleet *v1alpha1.Fleet) bool {
				return fleet.Status.AllocatedReplicas == replicasCount
			})

			fa2, err := framework.AgonesClient.StableV1alpha1().FleetAllocations(defaultNs).Create(fa)
			assert.NotNil(t, err)

			assert.Nil(t, fa2.Status.GameServer)
		})

	}
}

func TestScaleFleetUpAndDownWithFleetAllocation(t *testing.T) {
	t.Parallel()

	//Use scaleFleetPatch (true) or scaleFleetSubresource (false)
	fixtures := []bool{true, false}

	for _, usePatch := range fixtures {
		t.Run("Use fleet Patch "+fmt.Sprint(usePatch), func(t *testing.T) {
			alpha1 := framework.AgonesClient.StableV1alpha1()

			flt := defaultFleet()
			flt.Spec.Replicas = 1
			flt, err := alpha1.Fleets(defaultNs).Create(flt)
			if assert.Nil(t, err) {
				defer alpha1.Fleets(defaultNs).Delete(flt.ObjectMeta.Name, nil) // nolint:errcheck
			}

			assert.Equal(t, int32(1), flt.Spec.Replicas)

			framework.WaitForFleetCondition(t, flt, e2e.FleetReadyCount(flt.Spec.Replicas))

			// scale up
			const targetScale = 3
			if usePatch {
				flt = scaleFleetPatch(t, flt, targetScale)
				assert.Equal(t, int32(targetScale), flt.Spec.Replicas)
			} else {
				flt = scaleFleetSubresource(t, flt, targetScale)
			}

			framework.WaitForFleetCondition(t, flt, e2e.FleetReadyCount(targetScale))

			// get an allocation
			fa := &v1alpha1.FleetAllocation{
				ObjectMeta: metav1.ObjectMeta{GenerateName: "allocation-", Namespace: defaultNs},
				Spec: v1alpha1.FleetAllocationSpec{
					FleetName: flt.ObjectMeta.Name,
				},
			}

			fa, err = alpha1.FleetAllocations(defaultNs).Create(fa)
			assert.Nil(t, err)
			assert.Equal(t, v1alpha1.GameServerStateAllocated, fa.Status.GameServer.Status.State)
			framework.WaitForFleetCondition(t, flt, func(fleet *v1alpha1.Fleet) bool {
				return fleet.Status.AllocatedReplicas == 1
			})

			// scale down, with allocation
			const scaleDownTarget = 1
			if usePatch {
				flt = scaleFleetPatch(t, flt, scaleDownTarget)
			} else {
				flt = scaleFleetSubresource(t, flt, scaleDownTarget)
			}

			framework.WaitForFleetCondition(t, flt, e2e.FleetReadyCount(0))
			// delete the allocated GameServer
			gp := int64(1)
			err = alpha1.GameServers(defaultNs).Delete(fa.Status.GameServer.ObjectMeta.Name, &metav1.DeleteOptions{GracePeriodSeconds: &gp})
			assert.Nil(t, err)
			framework.WaitForFleetCondition(t, flt, e2e.FleetReadyCount(1))

			framework.WaitForFleetCondition(t, flt, func(fleet *v1alpha1.Fleet) bool {
				return fleet.Status.AllocatedReplicas == 0
			})
		})
	}
}

func TestScaleFleetUpAndDownWithGameServerAllocation(t *testing.T) {
	t.Parallel()

	fixtures := []bool{false, true}

	for _, usePatch := range fixtures {
		t.Run("Use fleet Patch "+fmt.Sprint(usePatch), func(t *testing.T) {
			alpha1 := framework.AgonesClient.StableV1alpha1()

			flt := defaultFleet()
			flt.Spec.Replicas = 1
			flt, err := alpha1.Fleets(defaultNs).Create(flt)
			if assert.Nil(t, err) {
				defer alpha1.Fleets(defaultNs).Delete(flt.ObjectMeta.Name, nil) // nolint:errcheck
			}

			assert.Equal(t, int32(1), flt.Spec.Replicas)

			framework.WaitForFleetCondition(t, flt, e2e.FleetReadyCount(flt.Spec.Replicas))

			// scale up
			const targetScale = 3
			if usePatch {
				flt = scaleFleetPatch(t, flt, targetScale)
				assert.Equal(t, int32(targetScale), flt.Spec.Replicas)
			} else {
				flt = scaleFleetSubresource(t, flt, targetScale)
			}

			framework.WaitForFleetCondition(t, flt, e2e.FleetReadyCount(targetScale))

			// get an allocation
			gsa := &v1alpha1.GameServerAllocation{ObjectMeta: metav1.ObjectMeta{GenerateName: "allocation-"},
				Spec: v1alpha1.GameServerAllocationSpec{
					Required: metav1.LabelSelector{MatchLabels: map[string]string{v1alpha1.FleetNameLabel: flt.ObjectMeta.Name}},
				}}

			gsa, err = alpha1.GameServerAllocations(defaultNs).Create(gsa)
			assert.Nil(t, err)
			assert.Equal(t, v1alpha1.GameServerAllocationAllocated, gsa.Status.State)
			framework.WaitForFleetCondition(t, flt, func(fleet *v1alpha1.Fleet) bool {
				return fleet.Status.AllocatedReplicas == 1
			})

			// scale down, with allocation
			const scaleDownTarget = 1
			if usePatch {
				flt = scaleFleetPatch(t, flt, scaleDownTarget)
			} else {
				flt = scaleFleetSubresource(t, flt, scaleDownTarget)
			}

			framework.WaitForFleetCondition(t, flt, e2e.FleetReadyCount(0))

			// delete the allocated GameServer
			gp := int64(1)
			err = alpha1.GameServers(defaultNs).Delete(gsa.Status.GameServerName, &metav1.DeleteOptions{GracePeriodSeconds: &gp})
			assert.Nil(t, err)
			framework.WaitForFleetCondition(t, flt, e2e.FleetReadyCount(1))

			framework.WaitForFleetCondition(t, flt, func(fleet *v1alpha1.Fleet) bool {
				return fleet.Status.AllocatedReplicas == 0
			})
		})
	}
}

func TestFleetUpdates(t *testing.T) {
	t.Parallel()

	fixtures := map[string]func() *v1alpha1.Fleet{
		"recreate": func() *v1alpha1.Fleet {
			flt := defaultFleet()
			flt.Spec.Strategy.Type = appsv1.RecreateDeploymentStrategyType
			return flt
		},
		"rolling": func() *v1alpha1.Fleet {
			flt := defaultFleet()
			flt.Spec.Strategy.Type = appsv1.RollingUpdateDeploymentStrategyType
			return flt
		},
	}

	for k, v := range fixtures {
		t.Run(k, func(t *testing.T) {
			alpha1 := framework.AgonesClient.StableV1alpha1()

			flt := v()
			flt.Spec.Template.ObjectMeta.Annotations = map[string]string{key: red}
			flt, err := alpha1.Fleets(defaultNs).Create(flt)
			if assert.Nil(t, err) {
				defer alpha1.Fleets(defaultNs).Delete(flt.ObjectMeta.Name, nil) // nolint:errcheck
			}

			err = framework.WaitForFleetGameServersCondition(flt, func(gs v1alpha1.GameServer) bool {
				return gs.ObjectMeta.Annotations[key] == red
			})
			assert.Nil(t, err)

			// if the generation has been updated, it's time to try again.
			err = wait.PollImmediate(time.Second, 10*time.Second, func() (bool, error) {
				flt, err = framework.AgonesClient.StableV1alpha1().Fleets(defaultNs).Get(flt.ObjectMeta.Name, metav1.GetOptions{})
				if err != nil {
					return false, err
				}
				fltCopy := flt.DeepCopy()
				fltCopy.Spec.Template.ObjectMeta.Annotations[key] = green
				_, err = framework.AgonesClient.StableV1alpha1().Fleets(defaultNs).Update(fltCopy)
				if err != nil {
					logrus.WithError(err).Warn("Could not update fleet, trying again")
					return false, nil
				}

				return true, nil
			})
			assert.Nil(t, err)

			err = framework.WaitForFleetGameServersCondition(flt, func(gs v1alpha1.GameServer) bool {
				return gs.ObjectMeta.Annotations[key] == green
			})
			assert.Nil(t, err)
		})
	}
}

func TestUpdateGameServerConfigurationInFleet(t *testing.T) {
	t.Parallel()

	alpha1 := framework.AgonesClient.StableV1alpha1()

	gsSpec := defaultGameServer().Spec
	oldPort := int32(7111)
	gsSpec.Ports = []v1alpha1.GameServerPort{{
		ContainerPort: oldPort,
		Name:          "gameport",
		PortPolicy:    v1alpha1.Dynamic,
		Protocol:      corev1.ProtocolUDP,
	}}
	flt := fleetWithGameServerSpec(gsSpec)
	flt, err := alpha1.Fleets(defaultNs).Create(flt)
	assert.Nil(t, err, "could not create fleet")
	defer alpha1.Fleets(defaultNs).Delete(flt.ObjectMeta.Name, nil) // nolint:errcheck

	assert.Equal(t, int32(replicasCount), flt.Spec.Replicas)

	framework.WaitForFleetCondition(t, flt, e2e.FleetReadyCount(flt.Spec.Replicas))

	// get an allocation
	gsa := &v1alpha1.GameServerAllocation{ObjectMeta: metav1.ObjectMeta{GenerateName: "allocation-"},
		Spec: v1alpha1.GameServerAllocationSpec{
			Required: metav1.LabelSelector{MatchLabels: map[string]string{v1alpha1.FleetNameLabel: flt.ObjectMeta.Name}},
		}}

	gsa, err = alpha1.GameServerAllocations(defaultNs).Create(gsa)
	assert.Nil(t, err, "cloud not create gameserver allocation")
	assert.Equal(t, v1alpha1.GameServerAllocationAllocated, gsa.Status.State)
	framework.WaitForFleetCondition(t, flt, func(fleet *v1alpha1.Fleet) bool {
		return fleet.Status.AllocatedReplicas == 1
	})

	flt, err = framework.AgonesClient.StableV1alpha1().Fleets(defaultNs).Get(flt.Name, metav1.GetOptions{})
	assert.Nil(t, err, "could not get fleet")

	// Update the configuration of the gameservers of the fleet, i.e. container port.
	// The changes should only be rolled out to gameservers in ready state, but not the allocated gameserver.
	newPort := int32(7222)
	fltCopy := flt.DeepCopy()
	fltCopy.Spec.Template.Spec.Ports[0].ContainerPort = newPort

	_, err = framework.AgonesClient.StableV1alpha1().Fleets(defaultNs).Update(fltCopy)
	assert.Nil(t, err, "could not update fleet")

	err = framework.WaitForFleetGameServersCondition(flt, func(gs v1alpha1.GameServer) bool {
		containerPort := gs.Spec.Ports[0].ContainerPort
		return (gs.Name == gsa.Status.GameServerName && containerPort == oldPort) ||
			(gs.Name != gsa.Status.GameServerName && containerPort == newPort)
	})
	assert.Nil(t, err, "gameservers don't have expected container port")
}

// TestFleetAllocationDuringGameServerDeletion is built to specifically
// test for race conditions of allocations when doing scale up/down,
// rolling updates, etc. Failures may not happen ALL the time -- as that is the
// nature of race conditions.
// nolint: dupl
func TestFleetAllocationDuringGameServerDeletion(t *testing.T) {
	t.Parallel()

	testAllocationRaceCondition := func(t *testing.T, fleet func() *v1alpha1.Fleet, deltaSleep time.Duration, delta func(t *testing.T, flt *v1alpha1.Fleet)) {
		alpha1 := framework.AgonesClient.StableV1alpha1()

		flt := fleet()
		flt.ApplyDefaults()
		size := int32(10)
		flt.Spec.Replicas = size
		flt, err := alpha1.Fleets(defaultNs).Create(flt)
		if assert.Nil(t, err) {
			defer alpha1.Fleets(defaultNs).Delete(flt.ObjectMeta.Name, nil) // nolint:errcheck
		}

		assert.Equal(t, size, flt.Spec.Replicas)

		framework.WaitForFleetCondition(t, flt, e2e.FleetReadyCount(flt.Spec.Replicas))

		var allocs []*v1alpha1.GameServer

		wg := sync.WaitGroup{}
		wg.Add(2)
		go func() {
			for {
				// this gives room for fleet scaling to go down - makes it more likely for the race condition to fire
				time.Sleep(100 * time.Millisecond)
				fa := &v1alpha1.FleetAllocation{
					ObjectMeta: metav1.ObjectMeta{GenerateName: "allocation-", Namespace: defaultNs},
					Spec: v1alpha1.FleetAllocationSpec{
						FleetName: flt.ObjectMeta.Name,
					},
				}
				fa, err = framework.AgonesClient.StableV1alpha1().FleetAllocations(defaultNs).Create(fa)
				if err != nil {
					logrus.WithError(err).Info("Allocation ended")
					break
				}
				logrus.WithField("gs", fa.Status.GameServer.ObjectMeta.Name).Info("Allocated")
				allocs = append(allocs, fa.Status.GameServer)
			}
			wg.Done()
		}()
		go func() {
			// this tends to force the scaling to happen as we are fleet allocating
			time.Sleep(deltaSleep)
			// call the function that makes the change to the fleet
			logrus.Info("Applying delta function")
			delta(t, flt)
			wg.Done()
		}()

		wg.Wait()
		assert.NotEmpty(t, allocs)

		for _, gs := range allocs {
			gsCheck, err := alpha1.GameServers(defaultNs).Get(gs.ObjectMeta.Name, metav1.GetOptions{})
			assert.Nil(t, err)
			assert.True(t, gsCheck.ObjectMeta.DeletionTimestamp.IsZero())
		}
	}

	t.Run("scale down", func(t *testing.T) {
		t.Parallel()

		testAllocationRaceCondition(t, defaultFleet, time.Second,
			func(t *testing.T, flt *v1alpha1.Fleet) {
				const targetScale = int32(0)
				flt = scaleFleetPatch(t, flt, targetScale)
				assert.Equal(t, targetScale, flt.Spec.Replicas)
			})
	})

	t.Run("recreate update", func(t *testing.T) {
		t.Parallel()

		fleet := func() *v1alpha1.Fleet {
			flt := defaultFleet()
			flt.Spec.Strategy.Type = appsv1.RecreateDeploymentStrategyType
			flt.Spec.Template.ObjectMeta.Annotations = map[string]string{key: red}

			return flt
		}

		testAllocationRaceCondition(t, fleet, time.Second,
			func(t *testing.T, flt *v1alpha1.Fleet) {
				flt, err := framework.AgonesClient.StableV1alpha1().Fleets(defaultNs).Get(flt.ObjectMeta.Name, metav1.GetOptions{})
				assert.Nil(t, err)
				fltCopy := flt.DeepCopy()
				fltCopy.Spec.Template.ObjectMeta.Annotations[key] = green
				_, err = framework.AgonesClient.StableV1alpha1().Fleets(defaultNs).Update(fltCopy)
				assertSuccessOrUpdateConflict(t, err)
			})
	})

	t.Run("rolling update", func(t *testing.T) {
		t.Parallel()

		fleet := func() *v1alpha1.Fleet {
			flt := defaultFleet()
			flt.Spec.Strategy.Type = appsv1.RollingUpdateDeploymentStrategyType
			flt.Spec.Template.ObjectMeta.Annotations = map[string]string{key: red}

			return flt
		}

		testAllocationRaceCondition(t, fleet, time.Duration(0),
			func(t *testing.T, flt *v1alpha1.Fleet) {
				flt, err := framework.AgonesClient.StableV1alpha1().Fleets(defaultNs).Get(flt.ObjectMeta.Name, metav1.GetOptions{})
				assert.Nil(t, err)
				fltCopy := flt.DeepCopy()
				fltCopy.Spec.Template.ObjectMeta.Annotations[key] = green
				_, err = framework.AgonesClient.StableV1alpha1().Fleets(defaultNs).Update(fltCopy)
				assertSuccessOrUpdateConflict(t, err)
			})
	})
}

// TestFleetNameValidation is built to test Fleet Name length validation,
// Fleet Name should have at most 63 chars
func TestFleetNameValidation(t *testing.T) {
	t.Parallel()
	alpha1 := framework.AgonesClient.StableV1alpha1()

	flt := defaultFleet()
	nameLen := validation.LabelValueMaxLength + 1
	bytes := make([]byte, nameLen)
	for i := 0; i < nameLen; i++ {
		bytes[i] = 'f'
	}
	flt.Name = string(bytes)
	_, err := alpha1.Fleets(defaultNs).Create(flt)
	assert.NotNil(t, err)
	statusErr, ok := err.(*k8serrors.StatusError)
	assert.True(t, ok)
	assert.True(t, len(statusErr.Status().Details.Causes) > 0)
	assert.Equal(t, statusErr.Status().Details.Causes[0].Type, metav1.CauseTypeFieldValueInvalid)
	goodFlt := defaultFleet()
	goodFlt.Name = string(bytes[0 : nameLen-1])
	goodFlt, err = alpha1.Fleets(defaultNs).Create(goodFlt)
	if assert.Nil(t, err) {
		defer alpha1.Fleets(defaultNs).Delete(goodFlt.ObjectMeta.Name, nil) // nolint:errcheck
	}

}

func assertSuccessOrUpdateConflict(t *testing.T, err error) {
	if !k8serrors.IsConflict(err) {
		// update conflicts are sometimes ok, we simply lost the race.
		assert.Nil(t, err)
	}
}

// TestGameServerAllocationDuringGameServerDeletion is built to specifically
// test for race conditions of allocations when doing scale up/down,
// rolling updates, etc. Failures may not happen ALL the time -- as that is the
// nature of race conditions.
// nolint: dupl
func TestGameServerAllocationDuringGameServerDeletion(t *testing.T) {
	t.Parallel()

	testAllocationRaceCondition := func(t *testing.T, fleet func() *v1alpha1.Fleet, deltaSleep time.Duration, delta func(t *testing.T, flt *v1alpha1.Fleet)) {
		alpha1 := framework.AgonesClient.StableV1alpha1()

		flt := fleet()
		flt.ApplyDefaults()
		size := int32(10)
		flt.Spec.Replicas = size
		flt, err := alpha1.Fleets(defaultNs).Create(flt)
		if assert.Nil(t, err) {
			defer alpha1.Fleets(defaultNs).Delete(flt.ObjectMeta.Name, nil) // nolint:errcheck
		}

		assert.Equal(t, size, flt.Spec.Replicas)

		framework.WaitForFleetCondition(t, flt, e2e.FleetReadyCount(flt.Spec.Replicas))

		var allocs []string

		wg := sync.WaitGroup{}
		wg.Add(2)
		go func() {
			for {
				// this gives room for fleet scaling to go down - makes it more likely for the race condition to fire
				time.Sleep(100 * time.Millisecond)
				gsa := &v1alpha1.GameServerAllocation{ObjectMeta: metav1.ObjectMeta{GenerateName: "allocation-"},
					Spec: v1alpha1.GameServerAllocationSpec{
						Required: metav1.LabelSelector{MatchLabels: map[string]string{v1alpha1.FleetNameLabel: flt.ObjectMeta.Name}},
					}}
				gsa, err = framework.AgonesClient.StableV1alpha1().GameServerAllocations(defaultNs).Create(gsa)
				if err != nil || gsa.Status.State == v1alpha1.GameServerAllocationUnAllocated {
					logrus.WithError(err).Info("Allocation ended")
					break
				}
				logrus.WithField("gs", gsa.Status.GameServerName).Info("Allocated")
				allocs = append(allocs, gsa.Status.GameServerName)
			}
			wg.Done()
		}()
		go func() {
			// this tends to force the scaling to happen as we are fleet allocating
			time.Sleep(deltaSleep)
			// call the function that makes the change to the fleet
			logrus.Info("Applying delta function")
			delta(t, flt)
			wg.Done()
		}()

		wg.Wait()
		assert.NotEmpty(t, allocs)

		for _, name := range allocs {
			gsCheck, err := alpha1.GameServers(defaultNs).Get(name, metav1.GetOptions{})
			assert.Nil(t, err)
			assert.True(t, gsCheck.ObjectMeta.DeletionTimestamp.IsZero())
		}
	}

	t.Run("scale down", func(t *testing.T) {
		t.Parallel()

		testAllocationRaceCondition(t, defaultFleet, time.Second,
			func(t *testing.T, flt *v1alpha1.Fleet) {
				const targetScale = int32(0)
				flt = scaleFleetPatch(t, flt, targetScale)
				assert.Equal(t, targetScale, flt.Spec.Replicas)
			})
	})

	t.Run("recreate update", func(t *testing.T) {
		t.Parallel()

		fleet := func() *v1alpha1.Fleet {
			flt := defaultFleet()
			flt.Spec.Strategy.Type = appsv1.RecreateDeploymentStrategyType
			flt.Spec.Template.ObjectMeta.Annotations = map[string]string{key: red}

			return flt
		}

		testAllocationRaceCondition(t, fleet, time.Second,
			func(t *testing.T, flt *v1alpha1.Fleet) {
				flt, err := framework.AgonesClient.StableV1alpha1().Fleets(defaultNs).Get(flt.ObjectMeta.Name, metav1.GetOptions{})
				assert.Nil(t, err)
				fltCopy := flt.DeepCopy()
				fltCopy.Spec.Template.ObjectMeta.Annotations[key] = green
				_, err = framework.AgonesClient.StableV1alpha1().Fleets(defaultNs).Update(fltCopy)
				assertSuccessOrUpdateConflict(t, err)
			})
	})

	t.Run("rolling update", func(t *testing.T) {
		t.Parallel()

		fleet := func() *v1alpha1.Fleet {
			flt := defaultFleet()
			flt.Spec.Strategy.Type = appsv1.RollingUpdateDeploymentStrategyType
			flt.Spec.Template.ObjectMeta.Annotations = map[string]string{key: red}

			return flt
		}

		testAllocationRaceCondition(t, fleet, time.Duration(0),
			func(t *testing.T, flt *v1alpha1.Fleet) {
				flt, err := framework.AgonesClient.StableV1alpha1().Fleets(defaultNs).Get(flt.ObjectMeta.Name, metav1.GetOptions{})
				assert.Nil(t, err)
				fltCopy := flt.DeepCopy()
				fltCopy.Spec.Template.ObjectMeta.Annotations[key] = green
				_, err = framework.AgonesClient.StableV1alpha1().Fleets(defaultNs).Update(fltCopy)
				assertSuccessOrUpdateConflict(t, err)
			})
	})
}

// TestCreateFleetAndUpdateScaleSubresource is built to
// test scale subresource usage and its ability to change Fleet Replica size.
// Both scaling up and down.
func TestCreateFleetAndUpdateScaleSubresource(t *testing.T) {
	alpha1 := framework.AgonesClient.StableV1alpha1()

	flt := defaultFleet()
	const initialReplicas int32 = 1
	flt.Spec.Replicas = initialReplicas
	flt, err := alpha1.Fleets(defaultNs).Create(flt)
	if assert.Nil(t, err) {
		defer alpha1.Fleets(defaultNs).Delete(flt.ObjectMeta.Name, nil) // nolint:errcheck
	}
	assert.Equal(t, initialReplicas, flt.Spec.Replicas)
	framework.WaitForFleetCondition(t, flt, e2e.FleetReadyCount(flt.Spec.Replicas))

	newReplicas := initialReplicas * 2
	scaleFleetSubresource(t, flt, newReplicas)
	framework.WaitForFleetCondition(t, flt, e2e.FleetReadyCount(newReplicas))

	scaleFleetSubresource(t, flt, initialReplicas)
	framework.WaitForFleetCondition(t, flt, e2e.FleetReadyCount(initialReplicas))
}

// TestScaleUpAndDownInParallelStressTest creates N fleets, half of which start with replicas=0
// and the other half with 0 and scales them up/down 3 times in parallel expecting it to reach
// the desired number of ready replicas each time.
// This test is also used as a stress test with 'make stress-test-e2e', in which case it creates
// many more fleets of bigger sizes and runs many more repetitions.
func TestScaleUpAndDownInParallelStressTest(t *testing.T) {
	t.Parallel()

	alpha1 := framework.AgonesClient.StableV1alpha1()
	fleetCount := 2
	fleetSize := int32(10)
	repeatCount := 3
	deadline := time.Now().Add(1 * time.Minute)

	logrus.WithField("fleetCount", fleetCount).
		WithField("fleetSize", fleetSize).
		WithField("repeatCount", repeatCount).
		WithField("deadline", deadline).
		Info("starting scale up/down test")

	if framework.StressTestLevel > 0 {
		fleetSize = 10 * int32(framework.StressTestLevel)
		repeatCount = 10
		fleetCount = 10
		deadline = time.Now().Add(45 * time.Minute)
	}

	var fleets []*v1alpha1.Fleet

	scaleUpStats := framework.NewStatsCollector(fmt.Sprintf("fleet_%v_scale_up", fleetSize))
	scaleDownStats := framework.NewStatsCollector(fmt.Sprintf("fleet_%v_scale_down", fleetSize))

	defer scaleUpStats.Report()
	defer scaleDownStats.Report()

	for fleetNumber := 0; fleetNumber < fleetCount; fleetNumber++ {
		flt := defaultFleet()
		flt.ObjectMeta.GenerateName = fmt.Sprintf("scale-fleet-%v-", fleetNumber)
		if fleetNumber%2 == 0 {
			// even-numbered fleets starts at fleetSize and are scaled down to zero and back.
			flt.Spec.Replicas = fleetSize
		} else {
			// odd-numbered fleets starts at zero and are scaled up to fleetSize and back.
			flt.Spec.Replicas = 0
		}

		flt, err := alpha1.Fleets(defaultNs).Create(flt)
		if assert.Nil(t, err) {
			defer alpha1.Fleets(defaultNs).Delete(flt.ObjectMeta.Name, nil) // nolint:errcheck
		}
		fleets = append(fleets, flt)
	}

	// wait for initial fleet conditions.
	for fleetNumber, flt := range fleets {
		if fleetNumber%2 == 0 {
			framework.WaitForFleetCondition(t, flt, e2e.FleetReadyCount(fleetSize))
		} else {
			framework.WaitForFleetCondition(t, flt, e2e.FleetReadyCount(0))
		}
	}

	var wg sync.WaitGroup

	for fleetNumber, flt := range fleets {
		wg.Add(1)
		go func(fleetNumber int, flt *v1alpha1.Fleet) {
			defer wg.Done()
			defer func() {
				if err := recover(); err != nil {
					t.Errorf("recovered panic: %v", err)
				}
			}()

			if fleetNumber%2 == 0 {
				scaleDownStats.ReportDuration(scaleAndWait(t, flt, 0), nil)
			}
			for i := 0; i < repeatCount; i++ {
				if time.Now().After(deadline) {
					break
				}
				scaleUpStats.ReportDuration(scaleAndWait(t, flt, fleetSize), nil)
				scaleDownStats.ReportDuration(scaleAndWait(t, flt, 0), nil)
			}
		}(fleetNumber, flt)
	}

	wg.Wait()
}

// Creates a fleet and one GameServer with Packed scheduling.
// Scale to two GameServers with Distributed scheduling.
// The old GameServer has Scheduling set to 5 and the new one has it set to Distributed.
func TestUpdateFleetScheduling(t *testing.T) {
	t.Parallel()
	t.Run("Updating Spec.Scheduling on fleet should be updated in GameServer",
		func(t *testing.T) {
			alpha1 := framework.AgonesClient.StableV1alpha1()

			flt := defaultFleet()
			flt.Spec.Replicas = 1
			flt.Spec.Scheduling = v1alpha1.Packed
			flt, err := alpha1.Fleets(defaultNs).Create(flt)

			if assert.Nil(t, err) {
				defer alpha1.Fleets(defaultNs).Delete(flt.ObjectMeta.Name, nil) // nolint:errcheck
			}

			assert.Equal(t, int32(1), flt.Spec.Replicas)
			assert.Equal(t, v1alpha1.Packed, flt.Spec.Scheduling)

			framework.WaitForFleetCondition(t, flt, e2e.FleetReadyCount(flt.Spec.Replicas))

			const targetScale = 2
			flt = schedulingFleetPatch(t, flt, v1alpha1.Distributed, targetScale)
			framework.WaitForFleetCondition(t, flt, e2e.FleetReadyCount(targetScale))

			assert.Equal(t, int32(targetScale), flt.Spec.Replicas)
			assert.Equal(t, v1alpha1.Distributed, flt.Spec.Scheduling)

			err = framework.WaitForFleetGameServerListCondition(flt,
				func(gsList []v1alpha1.GameServer) bool {
					return countFleetScheduling(gsList, v1alpha1.Distributed) == 1 &&
						countFleetScheduling(gsList, v1alpha1.Packed) == 1
				})
			assert.Nil(t, err)
		})
}

// Counts the number of gameservers with the specified scheduling strategy in a fleet
func countFleetScheduling(gsList []v1alpha1.GameServer, scheduling v1alpha1.SchedulingStrategy) int {
	count := 0
	for _, gs := range gsList {
		if gs.Spec.Scheduling == scheduling {
			count++
		}
	}
	return count
}

// Patches fleet with scheduling and scale values
func schedulingFleetPatch(t *testing.T,
	f *v1alpha1.Fleet,
	scheduling v1alpha1.SchedulingStrategy,
	scale int32) *v1alpha1.Fleet {

	patch := fmt.Sprintf(`[{ "op": "replace", "path": "/spec/scheduling", "value": "%s" },
	                       { "op": "replace", "path": "/spec/replicas", "value": %d }]`,
		scheduling, scale)

	logrus.WithField("fleet", f.ObjectMeta.Name).
		WithField("scheduling", scheduling).
		WithField("scale", scale).
		WithField("patch", patch).
		Info("updating scheduling")

	fltRes, err := framework.AgonesClient.
		StableV1alpha1().
		Fleets(defaultNs).
		Patch(f.ObjectMeta.Name, types.JSONPatchType, []byte(patch))

	assert.Nil(t, err)
	return fltRes
}

func scaleAndWait(t *testing.T, flt *v1alpha1.Fleet, fleetSize int32) time.Duration {
	t0 := time.Now()
	scaleFleetSubresource(t, flt, fleetSize)
	framework.WaitForFleetCondition(t, flt, e2e.FleetReadyCount(fleetSize))
	return time.Since(t0)
}

// scaleFleetPatch creates a patch to apply to a Fleet.
// Easier for testing, as it removes object generational issues.
func scaleFleetPatch(t *testing.T, f *v1alpha1.Fleet, scale int32) *v1alpha1.Fleet {
	patch := fmt.Sprintf(`[{ "op": "replace", "path": "/spec/replicas", "value": %d }]`, scale)
	logrus.WithField("fleet", f.ObjectMeta.Name).WithField("scale", scale).WithField("patch", patch).Info("Scaling fleet")

	fltRes, err := framework.AgonesClient.StableV1alpha1().Fleets(defaultNs).Patch(f.ObjectMeta.Name, types.JSONPatchType, []byte(patch))
	assert.Nil(t, err)
	return fltRes
}

// scaleFleetSubresource uses scale subresource to change Replicas size of the Fleet.
// Returns the same f as in parameter, just to keep signature in sync with scaleFleetPatch
func scaleFleetSubresource(t *testing.T, f *v1alpha1.Fleet, scale int32) *v1alpha1.Fleet {
	logrus.WithField("fleet", f.ObjectMeta.Name).WithField("scale", scale).Info("Scaling fleet")

	err := retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		alpha1 := framework.AgonesClient.StableV1alpha1()
		// GetScale returns current Scale object with resourceVersion which is opaque object
		// and it will be used to create new Scale object
		opts := metav1.GetOptions{}
		sc, err := alpha1.Fleets(defaultNs).GetScale(f.ObjectMeta.Name, opts)
		if err != nil {
			return err
		}

		sc2 := newScale(f.Name, scale, sc.ObjectMeta.ResourceVersion)
		_, err = alpha1.Fleets(defaultNs).UpdateScale(f.ObjectMeta.Name, sc2)
		return err
	})

	if err != nil {
		t.Fatal("could not update the scale subresource")
	}
	return f
}

// defaultFleet returns a default fleet configuration
func defaultFleet() *v1alpha1.Fleet {
	gs := defaultGameServer()
	return fleetWithGameServerSpec(gs.Spec)
}

// fleetWithGameServerSpec returns a fleet with specified gameserver spec
func fleetWithGameServerSpec(gsSpec v1alpha1.GameServerSpec) *v1alpha1.Fleet {
	return &v1alpha1.Fleet{
		ObjectMeta: metav1.ObjectMeta{GenerateName: "simple-fleet-", Namespace: defaultNs},
		Spec: v1alpha1.FleetSpec{
			Replicas: replicasCount,
			Template: v1alpha1.GameServerTemplateSpec{
				Spec: gsSpec,
			},
		},
	}
}

// newScale returns a scale with specified Replicas spec
func newScale(fleetName string, newReplicas int32, resourceVersion string) *v1betaext.Scale {
	return &v1betaext.Scale{
		ObjectMeta: metav1.ObjectMeta{Name: fleetName, Namespace: defaultNs, ResourceVersion: resourceVersion},
		Spec: v1betaext.ScaleSpec{
			Replicas: newReplicas,
		},
	}
}
