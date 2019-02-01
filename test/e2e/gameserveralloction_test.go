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
	"testing"
	"time"

	"agones.dev/agones/pkg/apis/stable/v1alpha1"
	e2e "agones.dev/agones/test/e2e/framework"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

func TestCreateFleetAndGameServerAllocate(t *testing.T) {
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

			gsa := &v1alpha1.GameServerAllocation{ObjectMeta: metav1.ObjectMeta{GenerateName: "gsa-"},
				Spec: v1alpha1.GameServerAllocationSpec{
					Scheduling: strategy,
					Required:   metav1.LabelSelector{MatchLabels: map[string]string{v1alpha1.FleetNameLabel: flt.ObjectMeta.Name}},
				}}

			gsa, err = framework.AgonesClient.StableV1alpha1().GameServerAllocations(fleet.ObjectMeta.Namespace).Create(gsa)
			if assert.Nil(t, err) {
				assert.Equal(t, string(v1alpha1.GameServerAllocationAllocated), string(gsa.Status.State))
			}
		})
	}
}

// Can't allocate more GameServers if a fleet is fully used.
func TestCreateFullFleetAndCantGameServerAllocate(t *testing.T) {
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

			gsa := &v1alpha1.GameServerAllocation{ObjectMeta: metav1.ObjectMeta{GenerateName: "allocation-"},
				Spec: v1alpha1.GameServerAllocationSpec{
					Scheduling: strategy,
					Required:   metav1.LabelSelector{MatchLabels: map[string]string{v1alpha1.FleetNameLabel: flt.ObjectMeta.Name}},
				}}

			for i := 0; i < replicasCount; i++ {
				var gsa2 *v1alpha1.GameServerAllocation
				gsa2, err = framework.AgonesClient.StableV1alpha1().GameServerAllocations(defaultNs).Create(gsa.DeepCopy())
				if assert.Nil(t, err) {
					assert.Equal(t, v1alpha1.GameServerAllocationAllocated, gsa2.Status.State)
				}
			}

			framework.WaitForFleetCondition(t, flt, func(fleet *v1alpha1.Fleet) bool {
				return fleet.Status.AllocatedReplicas == replicasCount
			})

			gsa, err = framework.AgonesClient.StableV1alpha1().GameServerAllocations(defaultNs).Create(gsa.DeepCopy())
			if assert.Nil(t, err) {
				assert.Equal(t, string(v1alpha1.GameServerAllocationUnAllocated), string(gsa.Status.State))
			}
		})
	}
}

func TestGameServerAllocationMetaDataPatch(t *testing.T) {
	t.Parallel()

	gs := defaultGameServer()
	gs.ObjectMeta.Labels = map[string]string{"test": t.Name()}

	gs, err := framework.CreateGameServerAndWaitUntilReady(defaultNs, gs)
	if !assert.Nil(t, err) {
		assert.FailNow(t, "could not create GameServer")
	}
	defer framework.AgonesClient.StableV1alpha1().GameServers(defaultNs).Delete(gs.ObjectMeta.Name, nil) // nolint: errcheck

	gsa := &v1alpha1.GameServerAllocation{ObjectMeta: metav1.ObjectMeta{GenerateName: "allocation-"},
		Spec: v1alpha1.GameServerAllocationSpec{
			Required: metav1.LabelSelector{MatchLabels: map[string]string{"test": t.Name()}},
			MetaPatch: v1alpha1.MetaPatch{
				Labels:      map[string]string{"red": "blue"},
				Annotations: map[string]string{"dog": "good"},
			},
		}}

	gsa, err = framework.AgonesClient.StableV1alpha1().GameServerAllocations(defaultNs).Create(gsa.DeepCopy())
	if assert.Nil(t, err) {
		assert.Equal(t, v1alpha1.GameServerAllocationAllocated, gsa.Status.State)
	}

	gs, err = framework.AgonesClient.StableV1alpha1().GameServers(defaultNs).Get(gsa.Status.GameServerName, metav1.GetOptions{})
	if assert.Nil(t, err) {
		assert.Equal(t, "blue", gs.ObjectMeta.Labels["red"])
		assert.Equal(t, "good", gs.ObjectMeta.Annotations["dog"])
	}
}

func TestGameServerAllocationPreferredSelection(t *testing.T) {
	t.Parallel()

	fleets := framework.AgonesClient.StableV1alpha1().Fleets(defaultNs)
	gameServers := framework.AgonesClient.StableV1alpha1().GameServers(defaultNs)
	label := map[string]string{"role": t.Name()}

	preferred := defaultFleet()
	preferred.ObjectMeta.GenerateName = "preferred-"
	preferred.Spec.Replicas = 1
	preferred.Spec.Template.ObjectMeta.Labels = label
	preferred, err := fleets.Create(preferred)
	if assert.Nil(t, err) {
		defer fleets.Delete(preferred.ObjectMeta.Name, nil) // nolint:errcheck
	} else {
		assert.FailNow(t, "could not create first fleet")
	}

	required := defaultFleet()
	required.ObjectMeta.GenerateName = "required-"
	required.Spec.Replicas = 2
	required.Spec.Template.ObjectMeta.Labels = label
	required, err = fleets.Create(required)
	if assert.Nil(t, err) {
		defer fleets.Delete(required.ObjectMeta.Name, nil) // nolint:errcheck
	} else {
		assert.FailNow(t, "could not create second fleet")
	}

	framework.WaitForFleetCondition(t, preferred, e2e.FleetReadyCount(preferred.Spec.Replicas))
	framework.WaitForFleetCondition(t, required, e2e.FleetReadyCount(required.Spec.Replicas))

	gsa := &v1alpha1.GameServerAllocation{ObjectMeta: metav1.ObjectMeta{GenerateName: "allocation-"},
		Spec: v1alpha1.GameServerAllocationSpec{
			Required: metav1.LabelSelector{MatchLabels: label},
			Preferred: []metav1.LabelSelector{
				{MatchLabels: map[string]string{v1alpha1.FleetNameLabel: preferred.ObjectMeta.Name}},
			},
		}}

	gsa1, err := framework.AgonesClient.StableV1alpha1().GameServerAllocations(defaultNs).Create(gsa.DeepCopy())
	if assert.Nil(t, err) {
		assert.Equal(t, v1alpha1.GameServerAllocationAllocated, gsa1.Status.State)
		gs, err := gameServers.Get(gsa1.Status.GameServerName, metav1.GetOptions{})
		assert.Nil(t, err)
		assert.Equal(t, preferred.ObjectMeta.Name, gs.ObjectMeta.Labels[v1alpha1.FleetNameLabel])
	} else {
		assert.FailNow(t, "could not completed gsa1 allocation")
	}

	gs2, err := framework.AgonesClient.StableV1alpha1().GameServerAllocations(defaultNs).Create(gsa.DeepCopy())
	if assert.Nil(t, err) {
		assert.Equal(t, v1alpha1.GameServerAllocationAllocated, gs2.Status.State)
		gs, err := gameServers.Get(gs2.Status.GameServerName, metav1.GetOptions{})
		assert.Nil(t, err)
		assert.Equal(t, required.ObjectMeta.Name, gs.ObjectMeta.Labels[v1alpha1.FleetNameLabel])
	} else {
		assert.FailNow(t, "could not completed gs2 allocation")
	}

	// delete the preferred gameserver, and then let's try allocating again, make sure it goes back to the
	// preferred one
	err = gameServers.Delete(gsa1.Status.GameServerName, nil)
	if !assert.Nil(t, err) {
		assert.FailNow(t, "could not delete gameserver")
	}

	// wait until the game server is deleted
	err = wait.PollImmediate(time.Second, 5*time.Minute, func() (bool, error) {
		_, err = gameServers.Get(gsa1.Status.GameServerName, metav1.GetOptions{})

		if err != nil && errors.IsNotFound(err) {
			return true, nil
		}

		return false, err
	})
	assert.Nil(t, err)

	// now wait for another one to come along
	framework.WaitForFleetCondition(t, preferred, e2e.FleetReadyCount(preferred.Spec.Replicas))

	gsa3, err := framework.AgonesClient.StableV1alpha1().GameServerAllocations(defaultNs).Create(gsa.DeepCopy())
	if assert.Nil(t, err) {
		assert.Equal(t, v1alpha1.GameServerAllocationAllocated, gsa3.Status.State)
		gs, err := gameServers.Get(gsa3.Status.GameServerName, metav1.GetOptions{})
		assert.Nil(t, err)
		assert.Equal(t, preferred.ObjectMeta.Name, gs.ObjectMeta.Labels[v1alpha1.FleetNameLabel])
	}
}

func TestGameServerAllocationDeletionOnUnAllocate(t *testing.T) {
	allocations := framework.AgonesClient.StableV1alpha1().GameServerAllocations(defaultNs)

	gsa := &v1alpha1.GameServerAllocation{ObjectMeta: metav1.ObjectMeta{GenerateName: "allocation-"},
		Spec: v1alpha1.GameServerAllocationSpec{
			Required: metav1.LabelSelector{MatchLabels: map[string]string{"never": "goingtohappen"}},
		}}

	gsa, err := allocations.Create(gsa.DeepCopy())
	if assert.Nil(t, err) {
		assert.Equal(t, v1alpha1.GameServerAllocationUnAllocated, gsa.Status.State)
	}

	// this should now delete after a while
	err = wait.PollImmediate(time.Second, 10*time.Second, func() (bool, error) {
		_, err := allocations.Get(gsa.ObjectMeta.Name, metav1.GetOptions{})

		if errors.IsNotFound(err) {
			return true, nil
		}

		return false, err
	})
	assert.Nil(t, err)
}
