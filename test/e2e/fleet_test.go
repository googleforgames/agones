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
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
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
	flt, err = scaleFleet(flt, 3)
	assert.Nil(t, err)
	assert.Equal(t, int32(3), flt.Spec.Replicas)

	framework.WaitForFleetCondition(t, flt, e2e.FleetReadyCount(flt.Spec.Replicas))

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
	flt, err = scaleFleet(flt, 1)
	assert.Nil(t, err)
	framework.WaitForFleetCondition(t, flt, e2e.FleetReadyCount(0))

	// delete the allocated GameServer
	gp := int64(1)
	err = alpha1.GameServers(defaultNs).Delete(fa.Status.GameServer.ObjectMeta.Name, &metav1.DeleteOptions{GracePeriodSeconds: &gp})
	assert.Nil(t, err)
	framework.WaitForFleetCondition(t, flt, e2e.FleetReadyCount(1))

	framework.WaitForFleetCondition(t, flt, func(fleet *v1alpha1.Fleet) bool {
		return fleet.Status.AllocatedReplicas == 0
	})
}

func TestScaleFleetUpAndDownWithGameServerAllocation(t *testing.T) {
	t.Parallel()
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
	flt, err = scaleFleet(flt, 3)
	assert.Nil(t, err)
	assert.Equal(t, int32(3), flt.Spec.Replicas)

	framework.WaitForFleetCondition(t, flt, e2e.FleetReadyCount(flt.Spec.Replicas))

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
	flt, err = scaleFleet(flt, 1)
	assert.Nil(t, err)
	framework.WaitForFleetCondition(t, flt, e2e.FleetReadyCount(0))

	// delete the allocated GameServer
	gp := int64(1)
	err = alpha1.GameServers(defaultNs).Delete(gsa.Status.GameServerName, &metav1.DeleteOptions{GracePeriodSeconds: &gp})
	assert.Nil(t, err)
	framework.WaitForFleetCondition(t, flt, e2e.FleetReadyCount(1))

	framework.WaitForFleetCondition(t, flt, func(fleet *v1alpha1.Fleet) bool {
		return fleet.Status.AllocatedReplicas == 0
	})
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
				fltResult, err := scaleFleet(flt, 0)
				assert.Nil(t, err)
				assert.Equal(t, int32(0), fltResult.Spec.Replicas)
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
				fltResult, err := scaleFleet(flt, 0)
				assert.Nil(t, err)
				assert.Equal(t, int32(0), fltResult.Spec.Replicas)
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

// scaleFleet creates a patch to apply to a Fleet.
// easier for testing, as it removes object generational issues.
func scaleFleet(f *v1alpha1.Fleet, scale int32) (*v1alpha1.Fleet, error) {
	patch := fmt.Sprintf(`[{ "op": "replace", "path": "/spec/replicas", "value": %d }]`, scale)
	logrus.WithField("fleet", f.ObjectMeta.Name).WithField("scale", scale).WithField("patch", patch).Info("Scaling fleet")

	return framework.AgonesClient.StableV1alpha1().Fleets(defaultNs).Patch(f.ObjectMeta.Name, types.JSONPatchType, []byte(patch))
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
