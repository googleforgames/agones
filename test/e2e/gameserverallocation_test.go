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
	"sync"
	"testing"
	"time"

	"agones.dev/agones/pkg/apis"
	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	allocationv1 "agones.dev/agones/pkg/apis/allocation/v1"
	multiclusterv1 "agones.dev/agones/pkg/apis/multicluster/v1"
	"agones.dev/agones/pkg/util/runtime"
	e2e "agones.dev/agones/test/e2e/framework"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/apimachinery/pkg/util/wait"
)

func TestCreateFleetAndGameServerAllocate(t *testing.T) {
	t.Parallel()

	fixtures := []apis.SchedulingStrategy{apis.Packed, apis.Distributed}

	for _, strategy := range fixtures {
		strategy := strategy
		t.Run(string(strategy), func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()

			fleets := framework.AgonesClient.AgonesV1().Fleets(framework.Namespace)
			fleet := defaultFleet(framework.Namespace)
			fleet.Spec.Scheduling = strategy
			flt, err := fleets.Create(ctx, fleet, metav1.CreateOptions{})
			if strategy != apis.Packed && framework.CloudProduct == "gke-autopilot" {
				// test that Autopilot rejects anything but Packed and skip the rest of the test
				assert.ErrorContains(t, err, "Invalid value")
				return
			}
			if assert.NoError(t, err) {
				defer fleets.Delete(ctx, flt.ObjectMeta.Name, metav1.DeleteOptions{}) // nolint:errcheck
			}

			framework.AssertFleetCondition(t, flt, e2e.FleetReadyCount(flt.Spec.Replicas))

			gsaList := []*allocationv1.GameServerAllocation{
				{Spec: allocationv1.GameServerAllocationSpec{
					Scheduling: strategy,
					Selectors:  []allocationv1.GameServerSelector{{LabelSelector: metav1.LabelSelector{MatchLabels: map[string]string{agonesv1.FleetNameLabel: flt.ObjectMeta.Name}}}}},
				},
				{Spec: allocationv1.GameServerAllocationSpec{
					Scheduling: strategy,
					Required:   allocationv1.GameServerSelector{LabelSelector: metav1.LabelSelector{MatchLabels: map[string]string{agonesv1.FleetNameLabel: flt.ObjectMeta.Name}}}},
				},
			}

			for _, gsa := range gsaList {
				gsa, err = framework.AgonesClient.AllocationV1().GameServerAllocations(fleet.ObjectMeta.Namespace).Create(ctx, gsa, metav1.CreateOptions{})
				if assert.NoError(t, err) {
					assert.Equal(t, string(allocationv1.GameServerAllocationAllocated), string(gsa.Status.State))
				}
			}
		})
	}
}

func TestCreateFleetAndGameServerStateFilterAllocation(t *testing.T) {
	t.Parallel()

	fleets := framework.AgonesClient.AgonesV1().Fleets(framework.Namespace)
	fleet := defaultFleet(framework.Namespace)
	ctx := context.Background()

	flt, err := fleets.Create(ctx, fleet, metav1.CreateOptions{})
	require.NoError(t, err)
	defer fleets.Delete(ctx, flt.ObjectMeta.Name, metav1.DeleteOptions{}) // nolint:errcheck

	framework.AssertFleetCondition(t, flt, e2e.FleetReadyCount(flt.Spec.Replicas))

	fleetSelector := metav1.LabelSelector{MatchLabels: map[string]string{agonesv1.FleetNameLabel: flt.ObjectMeta.Name}}
	gsa := &allocationv1.GameServerAllocation{
		Spec: allocationv1.GameServerAllocationSpec{
			Selectors: []allocationv1.GameServerSelector{{LabelSelector: fleetSelector}},
		}}

	// standard allocation
	gsa, err = framework.AgonesClient.AllocationV1().GameServerAllocations(fleet.ObjectMeta.Namespace).Create(ctx, gsa, metav1.CreateOptions{})
	require.NoError(t, err)
	assert.Equal(t, string(allocationv1.GameServerAllocationAllocated), string(gsa.Status.State))

	gs1, err := framework.AgonesClient.AgonesV1().GameServers(fleet.ObjectMeta.Namespace).Get(ctx, gsa.Status.GameServerName, metav1.GetOptions{})
	require.NoError(t, err)
	assert.Equal(t, agonesv1.GameServerStateAllocated, gs1.Status.State)
	assert.NotNil(t, gs1.ObjectMeta.Annotations["agones.dev/last-allocated"])

	// now let's get it back again
	gsa = gsa.DeepCopy()
	allocated := agonesv1.GameServerStateAllocated
	gsa.Spec.Selectors[0].GameServerState = &allocated

	gsa, err = framework.AgonesClient.AllocationV1().GameServerAllocations(fleet.ObjectMeta.Namespace).Create(ctx, gsa, metav1.CreateOptions{})
	require.NoError(t, err)
	assert.Equal(t, string(allocationv1.GameServerAllocationAllocated), string(gsa.Status.State))
	assert.Equal(t, gs1.ObjectMeta.Name, gsa.Status.GameServerName)

	gs2, err := framework.AgonesClient.AgonesV1().GameServers(fleet.ObjectMeta.Namespace).Get(ctx, gsa.Status.GameServerName, metav1.GetOptions{})
	require.NoError(t, err)
	assert.Equal(t, agonesv1.GameServerStateAllocated, gs2.Status.State)

	require.Equal(t, gs1.ObjectMeta.Name, gs2.ObjectMeta.Name)
	require.NotEqual(t, gs1.ObjectMeta.ResourceVersion, gs2.ObjectMeta.ResourceVersion)
	require.NotEqual(t, gs1.ObjectMeta.Annotations["agones.dev/last-allocated"], gs2.ObjectMeta.Annotations["agones.dev/last-allocated"])
}

func TestHighDensityGameServerFlow(t *testing.T) {
	t.Parallel()
	log := e2e.TestLogger(t)
	ctx := context.Background()

	fleets := framework.AgonesClient.AgonesV1().Fleets(framework.Namespace)
	fleet := defaultFleet(framework.Namespace)
	lockLabel := "agones.dev/sdk-available"
	// to start they are all available
	fleet.Spec.Template.ObjectMeta.Labels = map[string]string{lockLabel: "true"}

	flt, err := fleets.Create(ctx, fleet, metav1.CreateOptions{})
	require.NoError(t, err)
	defer fleets.Delete(ctx, flt.ObjectMeta.Name, metav1.DeleteOptions{}) // nolint:errcheck

	framework.AssertFleetCondition(t, flt, e2e.FleetReadyCount(flt.Spec.Replicas))

	fleetSelector := metav1.LabelSelector{MatchLabels: map[string]string{agonesv1.FleetNameLabel: flt.ObjectMeta.Name}}
	allocatedSelector := fleetSelector.DeepCopy()

	allocated := agonesv1.GameServerStateAllocated
	allocatedSelector.MatchLabels[lockLabel] = "true"
	gsa := &allocationv1.GameServerAllocation{
		Spec: allocationv1.GameServerAllocationSpec{
			MetaPatch: allocationv1.MetaPatch{Labels: map[string]string{lockLabel: "false"}},
			Selectors: []allocationv1.GameServerSelector{
				{LabelSelector: *allocatedSelector, GameServerState: &allocated},
				{LabelSelector: fleetSelector},
			},
		}}

	// standard allocation
	result, err := framework.AgonesClient.AllocationV1().GameServerAllocations(fleet.ObjectMeta.Namespace).Create(ctx, gsa, metav1.CreateOptions{})
	require.NoError(t, err)
	require.Equal(t, string(allocationv1.GameServerAllocationAllocated), string(result.Status.State))

	gs, err := framework.AgonesClient.AgonesV1().GameServers(fleet.ObjectMeta.Namespace).Get(ctx, result.Status.GameServerName, metav1.GetOptions{})
	require.NoError(t, err)
	require.Equal(t, allocated, gs.Status.State)

	// set the label to being available again
	_, err = framework.SendGameServerUDP(t, gs, "LABEL available true")
	require.NoError(t, err)

	// wait for the label to be applied!
	require.Eventuallyf(t, func() bool {
		gs, err := framework.AgonesClient.AgonesV1().GameServers(fleet.ObjectMeta.Namespace).Get(ctx, result.Status.GameServerName, metav1.GetOptions{})
		require.NoError(t, err)
		log.WithField("labels", gs.ObjectMeta.Labels).Info("checking labels")
		return gs.ObjectMeta.Labels[lockLabel] == "true"
	}, time.Minute, time.Second, "GameServer did not unlock")

	// Run the same allocation again, we should get back the preferred item.
	expected := result.Status.GameServerName

	// we will run this as an Eventually, as caches are eventually consistent
	require.Eventuallyf(t, func() bool {
		result, err = framework.AgonesClient.AllocationV1().GameServerAllocations(fleet.ObjectMeta.Namespace).Create(ctx, gsa, metav1.CreateOptions{})
		require.NoError(t, err)
		require.Equal(t, string(allocationv1.GameServerAllocationAllocated), string(result.Status.State))

		if expected != result.Status.GameServerName {
			log.WithField("expected", expected).WithField("gsa", result).Info("Re-allocation attempt failed. Retrying.")
			return false
		}

		return true
	}, time.Minute, time.Second, "Could not re-allocation")
}

func TestCreateFleetAndGameServerPlayerCapacityAllocation(t *testing.T) {
	if !(runtime.FeatureEnabled(runtime.FeaturePlayerAllocationFilter)) {
		t.SkipNow()
	}
	t.Parallel()

	fleets := framework.AgonesClient.AgonesV1().Fleets(framework.Namespace)
	fleet := defaultFleet(framework.Namespace)
	fleet.Spec.Template.Spec.Players = &agonesv1.PlayersSpec{InitialCapacity: 10}
	ctx := context.Background()

	flt, err := fleets.Create(ctx, fleet, metav1.CreateOptions{})
	require.NoError(t, err)
	defer fleets.Delete(ctx, flt.ObjectMeta.Name, metav1.DeleteOptions{}) // nolint:errcheck

	framework.AssertFleetCondition(t, flt, e2e.FleetReadyCount(flt.Spec.Replicas))

	fleetSelector := metav1.LabelSelector{MatchLabels: map[string]string{agonesv1.FleetNameLabel: flt.ObjectMeta.Name}}
	allocated := agonesv1.GameServerStateAllocated
	gsa := &allocationv1.GameServerAllocation{
		Spec: allocationv1.GameServerAllocationSpec{
			Selectors: []allocationv1.GameServerSelector{
				{
					LabelSelector:   fleetSelector,
					GameServerState: &allocated,
					Players: &allocationv1.PlayerSelector{
						MinAvailable: 1,
						MaxAvailable: 99,
					},
				},
				{LabelSelector: fleetSelector, Players: &allocationv1.PlayerSelector{MinAvailable: 5, MaxAvailable: 10}},
			},
		}}

	// first try should give me a Ready->Allocated server
	gsa, err = framework.AgonesClient.AllocationV1().GameServerAllocations(fleet.ObjectMeta.Namespace).Create(ctx, gsa.DeepCopy(), metav1.CreateOptions{})
	require.NoError(t, err)
	assert.Equal(t, string(allocationv1.GameServerAllocationAllocated), string(gsa.Status.State))

	gs1, err := framework.AgonesClient.AgonesV1().GameServers(fleet.ObjectMeta.Namespace).Get(ctx, gsa.Status.GameServerName, metav1.GetOptions{})
	require.NoError(t, err)
	assert.Equal(t, agonesv1.GameServerStateAllocated, gs1.Status.State)
	assert.NotNil(t, gs1.ObjectMeta.Annotations["agones.dev/last-allocated"])

	// second try should give me the same allocated server
	gsa, err = framework.AgonesClient.AllocationV1().GameServerAllocations(fleet.ObjectMeta.Namespace).Create(ctx, gsa.DeepCopy(), metav1.CreateOptions{})
	require.NoError(t, err)
	assert.Equal(t, string(allocationv1.GameServerAllocationAllocated), string(gsa.Status.State))
	assert.Equal(t, gs1.ObjectMeta.Name, gsa.Status.GameServerName)

	gs2, err := framework.AgonesClient.AgonesV1().GameServers(fleet.ObjectMeta.Namespace).Get(ctx, gsa.Status.GameServerName, metav1.GetOptions{})
	require.NoError(t, err)
	assert.Equal(t, agonesv1.GameServerStateAllocated, gs2.Status.State)

	require.Equal(t, gs1.ObjectMeta.Name, gs2.ObjectMeta.Name)
	require.NotEqual(t, gs1.ObjectMeta.ResourceVersion, gs2.ObjectMeta.ResourceVersion)
	require.NotEqual(t, gs1.ObjectMeta.Annotations["agones.dev/last-allocated"], gs2.ObjectMeta.Annotations["agones.dev/last-allocated"])
}

func TestMultiClusterAllocationOnLocalCluster(t *testing.T) {
	t.Parallel()

	fixtures := []apis.SchedulingStrategy{apis.Packed, apis.Distributed}
	for _, strategy := range fixtures {
		strategy := strategy
		t.Run(string(strategy), func(t *testing.T) {
			if strategy == apis.Distributed {
				framework.SkipOnCloudProduct(t, "gke-autopilot", "Autopilot does not support Distributed scheduling")
			}
			t.Parallel()
			ctx := context.Background()

			namespace := fmt.Sprintf("gsa-multicluster-local-%s", uuid.NewUUID())
			err := framework.CreateNamespace(namespace)
			if !assert.Nil(t, err) {
				return
			}
			defer func() {
				if derr := framework.DeleteNamespace(namespace); derr != nil {
					t.Error(derr)
				}
			}()

			fleets := framework.AgonesClient.AgonesV1().Fleets(namespace)
			fleet := defaultFleet(namespace)
			fleet.Spec.Scheduling = strategy
			flt, err := fleets.Create(ctx, fleet, metav1.CreateOptions{})
			if assert.Nil(t, err) {
				defer fleets.Delete(ctx, flt.ObjectMeta.Name, metav1.DeleteOptions{}) // nolint:errcheck
			}

			framework.AssertFleetCondition(t, flt, e2e.FleetReadyCount(flt.Spec.Replicas))

			// Allocation Policy #1: local cluster with desired label.
			// This policy allocates locally on the cluster due to matching namespace with gsa and not setting AllocationEndpoints.
			mca := &multiclusterv1.GameServerAllocationPolicy{
				Spec: multiclusterv1.GameServerAllocationPolicySpec{
					Priority: 1,
					Weight:   100,
					ConnectionInfo: multiclusterv1.ClusterConnectionInfo{
						ClusterName: "multicluster1",
						SecretName:  "sec1",
						Namespace:   namespace,
					},
				},
				ObjectMeta: metav1.ObjectMeta{
					Labels:       map[string]string{"cluster": "onprem"},
					GenerateName: "allocationpolicy-",
				},
			}
			resp, err := framework.AgonesClient.MulticlusterV1().GameServerAllocationPolicies(fleet.ObjectMeta.Namespace).Create(ctx, mca, metav1.CreateOptions{})
			if !assert.Nil(t, err) {
				assert.FailNowf(t, "GameServerAllocationPolicies(%v).Create(ctx, %v, metav1.CreateOptions{})", fleet.ObjectMeta.Namespace, mca)
			}
			assert.Equal(t, mca.Spec, resp.Spec)

			// Allocation Policy #2: another cluster with desired label, but lower priority.
			// If the policy is selected due to a bug the request fails as it cannot find the secret.
			mca = &multiclusterv1.GameServerAllocationPolicy{
				Spec: multiclusterv1.GameServerAllocationPolicySpec{
					Priority: 2,
					Weight:   100,
					ConnectionInfo: multiclusterv1.ClusterConnectionInfo{
						AllocationEndpoints: []string{"another-endpoint"},
						ClusterName:         "multicluster2",
						SecretName:          "sec2",
						Namespace:           namespace,
					},
				},
				ObjectMeta: metav1.ObjectMeta{
					Labels:       map[string]string{"cluster": "onprem"},
					GenerateName: "allocationpolicy-",
				},
			}
			resp, err = framework.AgonesClient.MulticlusterV1().GameServerAllocationPolicies(fleet.ObjectMeta.Namespace).Create(ctx, mca, metav1.CreateOptions{})
			if assert.Nil(t, err) {
				assert.Equal(t, mca.Spec, resp.Spec)
			}

			// Allocation Policy #3: another cluster with highest priority, but missing desired label (will not be selected)
			mca = &multiclusterv1.GameServerAllocationPolicy{
				Spec: multiclusterv1.GameServerAllocationPolicySpec{
					Priority: 1,
					Weight:   10,
					ConnectionInfo: multiclusterv1.ClusterConnectionInfo{
						AllocationEndpoints: []string{"another-endpoint"},
						ClusterName:         "multicluster3",
						SecretName:          "sec3",
						Namespace:           namespace,
					},
				},
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "allocationpolicy-",
				},
			}
			resp, err = framework.AgonesClient.MulticlusterV1().GameServerAllocationPolicies(fleet.ObjectMeta.Namespace).Create(ctx, mca, metav1.CreateOptions{})
			if assert.Nil(t, err) {
				assert.Equal(t, mca.Spec, resp.Spec)
			}

			gsa := &allocationv1.GameServerAllocation{
				Spec: allocationv1.GameServerAllocationSpec{
					Scheduling: strategy,
					Selectors:  []allocationv1.GameServerSelector{{LabelSelector: metav1.LabelSelector{MatchLabels: map[string]string{agonesv1.FleetNameLabel: flt.ObjectMeta.Name}}}},
					MultiClusterSetting: allocationv1.MultiClusterSetting{
						Enabled: true,
						PolicySelector: metav1.LabelSelector{
							MatchLabels: map[string]string{
								"cluster": "onprem",
							},
						},
					},
				},
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "allocation-",
					Namespace:    namespace,
				},
			}

			// wait for the allocation policies to be added.
			err = wait.PollImmediate(2*time.Second, 2*time.Minute, func() (bool, error) {
				gsa, err = framework.AgonesClient.AllocationV1().GameServerAllocations(fleet.ObjectMeta.Namespace).Create(ctx, gsa, metav1.CreateOptions{})
				if err != nil {
					t.Logf("GameServerAllocations(%v).Create(ctx, %v, metav1.CreateOptions{}) failed: %s", fleet.ObjectMeta.Namespace, gsa, err)
					return false, nil
				}

				assert.Equal(t, string(allocationv1.GameServerAllocationAllocated), string(gsa.Status.State))
				return true, nil
			})

			assert.NoError(t, err)
		})
	}
}

// Can't allocate more GameServers if a fleet is fully used.
func TestCreateFullFleetAndCantGameServerAllocate(t *testing.T) {
	t.Parallel()

	fixtures := []apis.SchedulingStrategy{apis.Packed, apis.Distributed}

	for _, strategy := range fixtures {
		strategy := strategy

		t.Run(string(strategy), func(t *testing.T) {
			if strategy == apis.Distributed {
				framework.SkipOnCloudProduct(t, "gke-autopilot", "Autopilot does not support Distributed scheduling")
			}
			t.Parallel()
			ctx := context.Background()

			fleets := framework.AgonesClient.AgonesV1().Fleets(framework.Namespace)
			fleet := defaultFleet(framework.Namespace)
			fleet.Spec.Scheduling = strategy
			flt, err := fleets.Create(ctx, fleet, metav1.CreateOptions{})
			if assert.Nil(t, err) {
				defer fleets.Delete(ctx, flt.ObjectMeta.Name, metav1.DeleteOptions{}) // nolint:errcheck
			}

			framework.AssertFleetCondition(t, flt, e2e.FleetReadyCount(flt.Spec.Replicas))

			gsa := &allocationv1.GameServerAllocation{
				Spec: allocationv1.GameServerAllocationSpec{
					Scheduling: strategy,
					Selectors:  []allocationv1.GameServerSelector{{LabelSelector: metav1.LabelSelector{MatchLabels: map[string]string{agonesv1.FleetNameLabel: flt.ObjectMeta.Name}}}},
				}}

			for i := 0; i < replicasCount; i++ {
				var gsa2 *allocationv1.GameServerAllocation
				gsa2, err = framework.AgonesClient.AllocationV1().GameServerAllocations(framework.Namespace).Create(ctx, gsa.DeepCopy(), metav1.CreateOptions{})
				if assert.Nil(t, err) {
					assert.Equal(t, allocationv1.GameServerAllocationAllocated, gsa2.Status.State)
				}
			}

			framework.AssertFleetCondition(t, flt, func(log *logrus.Entry, fleet *agonesv1.Fleet) bool {
				return fleet.Status.AllocatedReplicas == replicasCount
			})

			gsa, err = framework.AgonesClient.AllocationV1().GameServerAllocations(framework.Namespace).Create(ctx, gsa.DeepCopy(), metav1.CreateOptions{})
			if assert.Nil(t, err) {
				assert.Equal(t, string(allocationv1.GameServerAllocationUnAllocated), string(gsa.Status.State))
			}
		})
	}
}

func TestGameServerAllocationMetaDataPatch(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	log := logrus.WithField("test", t.Name())
	createAndAllocate := func(input *allocationv1.GameServerAllocation) *allocationv1.GameServerAllocation {
		gs := framework.DefaultGameServer(framework.Namespace)
		gs.ObjectMeta.Labels = map[string]string{"test": t.Name()}
		gs, err := framework.CreateGameServerAndWaitUntilReady(t, framework.Namespace, gs)
		require.NoError(t, err)

		log.WithField("gs", gs.ObjectMeta.Name).Info("👍 created and ready")

		// poll, as it may take a moment for the allocation cache to be populated
		err = wait.PollImmediate(time.Second, 30*time.Second, func() (bool, error) {
			input, err = framework.AgonesClient.AllocationV1().GameServerAllocations(framework.Namespace).Create(ctx, input, metav1.CreateOptions{})
			if err != nil {
				log.WithError(err).Info("Failed, trying again...")
				return false, err
			}

			return allocationv1.GameServerAllocationAllocated == input.Status.State, nil
		})
		require.NoError(t, err)
		return input
	}

	// two standard labels
	gsa := &allocationv1.GameServerAllocation{ObjectMeta: metav1.ObjectMeta{GenerateName: "allocation-"},
		Spec: allocationv1.GameServerAllocationSpec{
			Selectors: []allocationv1.GameServerSelector{{LabelSelector: metav1.LabelSelector{MatchLabels: map[string]string{"test": t.Name()}}}},
			MetaPatch: allocationv1.MetaPatch{
				Labels:      map[string]string{"red": "blue"},
				Annotations: map[string]string{"dog": "good"},
			},
		}}
	result := createAndAllocate(gsa)
	defer framework.AgonesClient.AgonesV1().GameServers(framework.Namespace).Delete(ctx, result.Status.GameServerName, metav1.DeleteOptions{}) // nolint: errcheck

	gs, err := framework.AgonesClient.AgonesV1().GameServers(framework.Namespace).Get(ctx, result.Status.GameServerName, metav1.GetOptions{})
	require.NoError(t, err)
	assert.Equal(t, "blue", gs.ObjectMeta.Labels["red"])
	assert.Equal(t, "good", gs.ObjectMeta.Annotations["dog"])

	// use special characters that are valid
	gsa.Spec.MetaPatch = allocationv1.MetaPatch{Labels: map[string]string{"blue-frog.fred_thing": "test"}}
	result = createAndAllocate(gsa)
	defer framework.AgonesClient.AgonesV1().GameServers(framework.Namespace).Delete(ctx, result.Status.GameServerName, metav1.DeleteOptions{}) // nolint: errcheck

	gs, err = framework.AgonesClient.AgonesV1().GameServers(framework.Namespace).Get(ctx, result.Status.GameServerName, metav1.GetOptions{})
	require.NoError(t, err)
	assert.Equal(t, "test", gs.ObjectMeta.Labels["blue-frog.fred_thing"])

	// throw something invalid at it.
	gsa.Spec.MetaPatch = allocationv1.MetaPatch{Labels: map[string]string{"$$$$$$$": "test"}}
	result, err = framework.AgonesClient.AllocationV1().GameServerAllocations(framework.Namespace).Create(ctx, gsa.DeepCopy(), metav1.CreateOptions{})
	log.WithField("result", result).WithError(err).Info("Failed allocation")
	require.Error(t, err)
	require.Contains(t, err.Error(), `GameServerAllocation.allocation.agones.dev "" is invalid`)
}

func TestGameServerAllocationPreferredSelection(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	fleets := framework.AgonesClient.AgonesV1().Fleets(framework.Namespace)
	gameServers := framework.AgonesClient.AgonesV1().GameServers(framework.Namespace)
	label := map[string]string{"role": t.Name()}

	preferred := defaultFleet(framework.Namespace)
	preferred.ObjectMeta.GenerateName = "preferred-"
	preferred.Spec.Replicas = 1
	preferred.Spec.Template.ObjectMeta.Labels = label
	preferred, err := fleets.Create(ctx, preferred, metav1.CreateOptions{})
	if assert.Nil(t, err) {
		defer fleets.Delete(ctx, preferred.ObjectMeta.Name, metav1.DeleteOptions{}) // nolint:errcheck
	} else {
		assert.FailNow(t, "could not create first fleet")
	}

	required := defaultFleet(framework.Namespace)
	required.ObjectMeta.GenerateName = "required-"
	required.Spec.Replicas = 2
	required.Spec.Template.ObjectMeta.Labels = label
	required, err = fleets.Create(ctx, required, metav1.CreateOptions{})
	if assert.Nil(t, err) {
		defer fleets.Delete(ctx, required.ObjectMeta.Name, metav1.DeleteOptions{}) // nolint:errcheck
	} else {
		assert.FailNow(t, "could not create second fleet")
	}

	framework.AssertFleetCondition(t, preferred, e2e.FleetReadyCount(preferred.Spec.Replicas))
	framework.AssertFleetCondition(t, required, e2e.FleetReadyCount(required.Spec.Replicas))

	gsa := &allocationv1.GameServerAllocation{ObjectMeta: metav1.ObjectMeta{GenerateName: "allocation-"},
		Spec: allocationv1.GameServerAllocationSpec{
			Selectors: []allocationv1.GameServerSelector{
				{LabelSelector: metav1.LabelSelector{MatchLabels: map[string]string{agonesv1.FleetNameLabel: preferred.ObjectMeta.Name}}},
				{LabelSelector: metav1.LabelSelector{MatchLabels: label}},
			},
		}}

	gsa1, err := framework.AgonesClient.AllocationV1().GameServerAllocations(framework.Namespace).Create(ctx, gsa.DeepCopy(), metav1.CreateOptions{})
	if assert.Nil(t, err) {
		assert.Equal(t, allocationv1.GameServerAllocationAllocated, gsa1.Status.State)
		gs, err := gameServers.Get(ctx, gsa1.Status.GameServerName, metav1.GetOptions{})
		assert.Nil(t, err)
		assert.Equal(t, preferred.ObjectMeta.Name, gs.ObjectMeta.Labels[agonesv1.FleetNameLabel])
	} else {
		assert.FailNow(t, "could not completed gsa1 allocation")
	}

	gs2, err := framework.AgonesClient.AllocationV1().GameServerAllocations(framework.Namespace).Create(ctx, gsa.DeepCopy(), metav1.CreateOptions{})
	if assert.Nil(t, err) {
		assert.Equal(t, allocationv1.GameServerAllocationAllocated, gs2.Status.State)
		gs, err := gameServers.Get(ctx, gs2.Status.GameServerName, metav1.GetOptions{})
		assert.Nil(t, err)
		assert.Equal(t, required.ObjectMeta.Name, gs.ObjectMeta.Labels[agonesv1.FleetNameLabel])
	} else {
		assert.FailNow(t, "could not completed gs2 allocation")
	}

	// delete the preferred gameserver, and then let's try allocating again, make sure it goes back to the
	// preferred one
	err = gameServers.Delete(ctx, gsa1.Status.GameServerName, metav1.DeleteOptions{})
	if !assert.Nil(t, err) {
		assert.FailNow(t, "could not delete gameserver")
	}

	// wait until the game server is deleted
	err = wait.PollImmediate(time.Second, 5*time.Minute, func() (bool, error) {
		_, err = gameServers.Get(ctx, gsa1.Status.GameServerName, metav1.GetOptions{})

		if err != nil && errors.IsNotFound(err) {
			return true, nil
		}

		return false, err
	})
	assert.Nil(t, err)

	// now wait for another one to come along
	framework.AssertFleetCondition(t, preferred, e2e.FleetReadyCount(preferred.Spec.Replicas))

	gsa3, err := framework.AgonesClient.AllocationV1().GameServerAllocations(framework.Namespace).Create(ctx, gsa.DeepCopy(), metav1.CreateOptions{})
	if assert.Nil(t, err) {
		assert.Equal(t, allocationv1.GameServerAllocationAllocated, gsa3.Status.State)
		gs, err := gameServers.Get(ctx, gsa3.Status.GameServerName, metav1.GetOptions{})
		assert.Nil(t, err)
		assert.Equal(t, preferred.ObjectMeta.Name, gs.ObjectMeta.Labels[agonesv1.FleetNameLabel])
	}
}

func TestGameServerAllocationReturnLabels(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	fleets := framework.AgonesClient.AgonesV1().Fleets(framework.Namespace)
	gameServers := framework.AgonesClient.AgonesV1().GameServers(framework.Namespace)
	role := "role"
	label := map[string]string{role: t.Name()}
	annotationKey := "someAnnotation"
	annotationValue := "someValue"
	annotations := map[string]string{annotationKey: annotationValue}

	flt := defaultFleet(framework.Namespace)
	flt.Spec.Replicas = 1
	flt.Spec.Template.ObjectMeta.Labels = label
	flt.Spec.Template.ObjectMeta.Annotations = annotations
	flt, err := fleets.Create(ctx, flt, metav1.CreateOptions{})
	defer fleets.Delete(ctx, flt.ObjectMeta.Name, metav1.DeleteOptions{}) // nolint:errcheck
	require.NoError(t, err)

	framework.AssertFleetCondition(t, flt, e2e.FleetReadyCount(flt.Spec.Replicas))

	gsa := &allocationv1.GameServerAllocation{ObjectMeta: metav1.ObjectMeta{GenerateName: "allocation-"},
		Spec: allocationv1.GameServerAllocationSpec{
			Selectors: []allocationv1.GameServerSelector{
				{LabelSelector: metav1.LabelSelector{MatchLabels: label}},
			},
		}}

	gsa, err = framework.AgonesClient.AllocationV1().GameServerAllocations(framework.Namespace).Create(ctx, gsa.DeepCopy(), metav1.CreateOptions{})
	require.NoError(t, err)

	assert.Equal(t, allocationv1.GameServerAllocationAllocated, gsa.Status.State)
	assert.Equal(t, t.Name(), gsa.Status.Metadata.Labels[role])
	assert.Equal(t, flt.ObjectMeta.Name, gsa.Status.Metadata.Labels[agonesv1.FleetNameLabel])
	assert.Equal(t, annotationValue, gsa.Status.Metadata.Annotations[annotationKey])
	gs, err := gameServers.Get(ctx, gsa.Status.GameServerName, metav1.GetOptions{})
	require.NoError(t, err)
	assert.Equal(t, flt.ObjectMeta.Name, gs.ObjectMeta.Labels[agonesv1.FleetNameLabel])
}

func TestGameServerAllocationDeletionOnUnAllocate(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	allocations := framework.AgonesClient.AllocationV1().GameServerAllocations(framework.Namespace)

	gsa := &allocationv1.GameServerAllocation{ObjectMeta: metav1.ObjectMeta{GenerateName: "allocation-"},
		Spec: allocationv1.GameServerAllocationSpec{
			Selectors: []allocationv1.GameServerSelector{{LabelSelector: metav1.LabelSelector{MatchLabels: map[string]string{"never": "goingtohappen"}}}},
		}}

	gsa, err := allocations.Create(ctx, gsa.DeepCopy(), metav1.CreateOptions{})
	if assert.Nil(t, err) {
		assert.Equal(t, allocationv1.GameServerAllocationUnAllocated, gsa.Status.State)
	}
}

func TestGameServerAllocationDuringMultipleAllocationClients(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	fleets := framework.AgonesClient.AgonesV1().Fleets(framework.Namespace)
	label := map[string]string{"role": t.Name()}

	preferred := defaultFleet(framework.Namespace)
	preferred.ObjectMeta.GenerateName = "preferred-"
	preferred.Spec.Replicas = 150
	preferred.Spec.Template.ObjectMeta.Labels = label
	preferred, err := fleets.Create(ctx, preferred, metav1.CreateOptions{})
	if assert.Nil(t, err) {
		defer fleets.Delete(ctx, preferred.ObjectMeta.Name, metav1.DeleteOptions{}) // nolint:errcheck
	} else {
		assert.FailNow(t, "could not create first fleet")
	}

	framework.AssertFleetCondition(t, preferred, e2e.FleetReadyCount(preferred.Spec.Replicas))

	// scale down before starting allocation
	preferred = scaleFleetPatch(ctx, t, preferred, preferred.Spec.Replicas-20)

	gsa := &allocationv1.GameServerAllocation{ObjectMeta: metav1.ObjectMeta{GenerateName: "allocation-"},
		Spec: allocationv1.GameServerAllocationSpec{
			Selectors: []allocationv1.GameServerSelector{
				{LabelSelector: metav1.LabelSelector{MatchLabels: map[string]string{agonesv1.FleetNameLabel: "preferred"}}},
				{LabelSelector: metav1.LabelSelector{MatchLabels: label}},
			},
		}}

	allocatedGS := sync.Map{}

	logrus.Infof("Starting Allocation.")
	var wg sync.WaitGroup

	// Allocate GS by 10 clients in parallel while the fleet is scaling down
	for i := 0; i < 10; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				gsa1, err := framework.AgonesClient.AllocationV1().GameServerAllocations(framework.Namespace).Create(ctx, gsa.DeepCopy(), metav1.CreateOptions{})
				if err == nil {
					allocatedGS.LoadOrStore(gsa1.Status.GameServerName, true)
				} else {
					t.Errorf("could not completed gsa1 allocation : %v", err)
				}
			}
		}()
	}

	time.Sleep(3 * time.Second)
	// scale down further while allocating
	scaleFleetPatch(ctx, t, preferred, preferred.Spec.Replicas-10)

	wg.Wait()
	logrus.Infof("Finished Allocation.")

	// count the number of unique game servers allocated
	// there should not be any duplicate
	uniqueAllocatedGSs := 0
	allocatedGS.Range(func(k, v interface{}) bool {
		uniqueAllocatedGSs++
		return true
	})
	assert.Equal(t, 100, uniqueAllocatedGSs)
}
