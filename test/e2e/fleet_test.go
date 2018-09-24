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
	"k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
)

const (
	key   = "test-state"
	red   = "red"
	green = "green"
)

func TestCreateFleetAndAllocate(t *testing.T) {
	t.Parallel()

	fleets := framework.AgonesClient.StableV1alpha1().Fleets(defaultNs)
	flt, err := fleets.Create(defaultFleet())
	if assert.Nil(t, err) {
		defer fleets.Delete(flt.ObjectMeta.Name, nil) // nolint:errcheck
	}

	err = framework.WaitForFleetCondition(flt, e2e.FleetReadyCount(flt.Spec.Replicas))
	assert.Nil(t, err, "fleet not ready")

	fa := &v1alpha1.FleetAllocation{
		ObjectMeta: metav1.ObjectMeta{GenerateName: "allocatioon-", Namespace: defaultNs},
		Spec: v1alpha1.FleetAllocationSpec{
			FleetName: flt.ObjectMeta.Name,
		},
	}

	fa, err = framework.AgonesClient.StableV1alpha1().FleetAllocations(defaultNs).Create(fa)
	assert.Nil(t, err)
	assert.Equal(t, v1alpha1.Allocated, fa.Status.GameServer.Status.State)
}

func TestScaleFleetUpAndDownWithAllocation(t *testing.T) {
	t.Parallel()
	alpha1 := framework.AgonesClient.StableV1alpha1()

	flt := defaultFleet()
	flt.Spec.Replicas = 1
	flt, err := alpha1.Fleets(defaultNs).Create(flt)
	if assert.Nil(t, err) {
		defer alpha1.Fleets(defaultNs).Delete(flt.ObjectMeta.Name, nil) // nolint:errcheck
	}

	assert.Equal(t, int32(1), flt.Spec.Replicas)

	err = framework.WaitForFleetCondition(flt, e2e.FleetReadyCount(flt.Spec.Replicas))
	assert.Nil(t, err, "fleet not ready")

	// scale up
	flt, err = scaleFleet(flt, 3)
	assert.Nil(t, err)
	assert.Equal(t, int32(3), flt.Spec.Replicas)

	err = framework.WaitForFleetCondition(flt, e2e.FleetReadyCount(flt.Spec.Replicas))
	assert.Nil(t, err)

	// get an allocation
	fa := &v1alpha1.FleetAllocation{
		ObjectMeta: metav1.ObjectMeta{GenerateName: "allocation-", Namespace: defaultNs},
		Spec: v1alpha1.FleetAllocationSpec{
			FleetName: flt.ObjectMeta.Name,
		},
	}

	fa, err = alpha1.FleetAllocations(defaultNs).Create(fa)
	assert.Nil(t, err)
	assert.Equal(t, v1alpha1.Allocated, fa.Status.GameServer.Status.State)
	err = framework.WaitForFleetCondition(flt, func(fleet *v1alpha1.Fleet) bool {
		return fleet.Status.AllocatedReplicas == 1
	})
	assert.Nil(t, err)

	// scale down, with allocation
	flt, err = scaleFleet(flt, 1)
	assert.Nil(t, err)
	err = framework.WaitForFleetCondition(flt, e2e.FleetReadyCount(0))
	assert.Nil(t, err)

	// delete the allocated GameServer
	gp := int64(1)
	err = alpha1.GameServers(defaultNs).Delete(fa.Status.GameServer.ObjectMeta.Name, &metav1.DeleteOptions{GracePeriodSeconds: &gp})
	assert.Nil(t, err)
	err = framework.WaitForFleetCondition(flt, e2e.FleetReadyCount(1))
	assert.Nil(t, err)

	err = framework.WaitForFleetCondition(flt, func(fleet *v1alpha1.Fleet) bool {
		return fleet.Status.AllocatedReplicas == 0
	})
	assert.Nil(t, err)
}

func TestFleetUpdates(t *testing.T) {
	t.Parallel()

	fixtures := map[string]func() *v1alpha1.Fleet{
		"recreate": func() *v1alpha1.Fleet {
			flt := defaultFleet()
			flt.Spec.Strategy.Type = v1.RecreateDeploymentStrategyType
			return flt
		},
		"rolling": func() *v1alpha1.Fleet {
			flt := defaultFleet()
			flt.Spec.Strategy.Type = v1.RollingUpdateDeploymentStrategyType
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

// TestFleetAllocationDuringGameServerDeletion is built to specifically
// test for race conditions of allocations when doing scale up/down,
// rolling updates, etc. Failures my not happen ALL the time -- as that is the
// nature of race conditions.
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

		err = framework.WaitForFleetCondition(flt, e2e.FleetReadyCount(flt.Spec.Replicas))
		assert.Nil(t, err, "fleet not ready")

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
			flt.Spec.Strategy.Type = v1.RecreateDeploymentStrategyType
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
				assert.Nil(t, err)
			})
	})

	t.Run("rolling update", func(t *testing.T) {
		t.Parallel()

		fleet := func() *v1alpha1.Fleet {
			flt := defaultFleet()
			flt.Spec.Strategy.Type = v1.RollingUpdateDeploymentStrategyType
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
				assert.Nil(t, err)
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

	return &v1alpha1.Fleet{
		ObjectMeta: metav1.ObjectMeta{GenerateName: "simple-fleet-", Namespace: defaultNs},
		Spec: v1alpha1.FleetSpec{
			Replicas: 3,
			Template: v1alpha1.GameServerTemplateSpec{
				Spec: gs.Spec,
			},
		},
	}
}
