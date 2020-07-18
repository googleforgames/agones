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

// nolint:goconst
package fleets

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"agones.dev/agones/pkg/apis"
	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	v1 "agones.dev/agones/pkg/apis/agones/v1"
	agonesv1clientset "agones.dev/agones/pkg/client/clientset/versioned/typed/agones/v1"
	agonesv1client "agones.dev/agones/pkg/client/listers/agones/v1"
	agtesting "agones.dev/agones/pkg/testing"
	utilruntime "agones.dev/agones/pkg/util/runtime"
	"agones.dev/agones/pkg/util/webhooks"
	"github.com/heptiolabs/healthcheck"
	"github.com/mattbaird/jsonpatch"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	admv1beta1 "k8s.io/api/admission/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	v1beta1 "k8s.io/api/extensions/v1beta1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/watch"
	k8stesting "k8s.io/client-go/testing"
	"k8s.io/client-go/tools/cache"
)

func TestControllerSyncFleet(t *testing.T) {
	t.Parallel()

	t.Run("no gameserverset, create it", func(t *testing.T) {
		f := defaultFixture()
		c, m := newFakeController()

		created := false
		m.AgonesClient.AddReactor("list", "fleets", func(action k8stesting.Action) (bool, runtime.Object, error) {
			return true, &agonesv1.FleetList{Items: []agonesv1.Fleet{*f}}, nil
		})

		m.AgonesClient.AddReactor("create", "gameserversets", func(action k8stesting.Action) (bool, runtime.Object, error) {
			ca := action.(k8stesting.CreateAction)
			gsSet := ca.GetObject().(*agonesv1.GameServerSet)

			created = true
			assert.True(t, metav1.IsControlledBy(gsSet, f))
			assert.Equal(t, f.Spec.Replicas, gsSet.Spec.Replicas)

			return true, gsSet, nil
		})

		_, cancel := agtesting.StartInformers(m, c.fleetSynced)
		defer cancel()

		err := c.syncFleet("default/fleet-1")
		assert.Nil(t, err)
		assert.True(t, created, "gameserverset should have been created")
		agtesting.AssertEventContains(t, m.FakeRecorder.Events, "CreatingGameServerSet")
	})

	t.Run("gameserverset with the same number of replicas", func(t *testing.T) {
		t.Parallel()
		f := defaultFixture()
		f.Spec.Strategy.Type = appsv1.RecreateDeploymentStrategyType
		c, m := newFakeController()
		gsSet := f.GameServerSet()

		gsSet.ObjectMeta.Name = "gsSet1"
		gsSet.ObjectMeta.UID = "4321"
		gsSet.Spec.Replicas = f.Spec.Replicas

		m.AgonesClient.AddReactor("list", "fleets", func(action k8stesting.Action) (bool, runtime.Object, error) {
			return true, &agonesv1.FleetList{Items: []agonesv1.Fleet{*f}}, nil
		})

		m.AgonesClient.AddReactor("list", "gameserversets", func(action k8stesting.Action) (bool, runtime.Object, error) {
			return true, &agonesv1.GameServerSetList{Items: []agonesv1.GameServerSet{*gsSet}}, nil
		})

		m.AgonesClient.AddReactor("create", "gameserversets", func(action k8stesting.Action) (bool, runtime.Object, error) {
			assert.FailNow(t, "gameserverset should not be created")
			return true, nil, nil
		})

		m.AgonesClient.AddReactor("update", "gameserversets", func(action k8stesting.Action) (bool, runtime.Object, error) {
			assert.FailNow(t, "gameserverset should not have been updated")
			return true, nil, nil
		})

		_, cancel := agtesting.StartInformers(m, c.fleetSynced, c.gameServerSetSynced)
		defer cancel()

		err := c.syncFleet("default/fleet-1")
		assert.Nil(t, err)
		agtesting.AssertNoEvent(t, m.FakeRecorder.Events)
	})

	t.Run("gameserverset with different number of replicas", func(t *testing.T) {
		f := defaultFixture()
		f.Spec.Strategy.Type = appsv1.RecreateDeploymentStrategyType
		c, m := newFakeController()
		gsSet := f.GameServerSet()
		gsSet.ObjectMeta.Name = "gsSet1"
		gsSet.ObjectMeta.UID = "1234"
		gsSet.Spec.Replicas = f.Spec.Replicas + 10
		updated := false

		m.AgonesClient.AddReactor("list", "fleets", func(action k8stesting.Action) (bool, runtime.Object, error) {
			return true, &agonesv1.FleetList{Items: []agonesv1.Fleet{*f}}, nil
		})

		m.AgonesClient.AddReactor("list", "gameserversets", func(action k8stesting.Action) (bool, runtime.Object, error) {
			return true, &agonesv1.GameServerSetList{Items: []agonesv1.GameServerSet{*gsSet}}, nil
		})

		m.AgonesClient.AddReactor("update", "gameserversets", func(action k8stesting.Action) (bool, runtime.Object, error) {
			updated = true

			ua := action.(k8stesting.UpdateAction)
			gsSet := ua.GetObject().(*agonesv1.GameServerSet)
			assert.Equal(t, f.Spec.Replicas, gsSet.Spec.Replicas)

			return true, gsSet, nil
		})

		_, cancel := agtesting.StartInformers(m, c.fleetSynced, c.gameServerSetSynced)
		defer cancel()

		err := c.syncFleet("default/fleet-1")
		assert.Nil(t, err)
		assert.True(t, updated, "gameserverset should have been updated")
		agtesting.AssertEventContains(t, m.FakeRecorder.Events, "ScalingGameServerSet")
	})

	t.Run("gameserverset with different scheduling", func(t *testing.T) {
		f := defaultFixture()
		f.Spec.Strategy.Type = appsv1.RecreateDeploymentStrategyType
		c, m := newFakeController()
		gsSet := f.GameServerSet()
		gsSet.ObjectMeta.Name = "gsSet1"
		gsSet.ObjectMeta.UID = "1234"
		gsSet.Spec.Replicas = f.Spec.Replicas
		gsSet.Spec.Scheduling = apis.Distributed
		updated := false

		m.AgonesClient.AddReactor("list", "fleets", func(action k8stesting.Action) (bool, runtime.Object, error) {
			return true, &agonesv1.FleetList{Items: []agonesv1.Fleet{*f}}, nil
		})

		m.AgonesClient.AddReactor("list", "gameserversets", func(action k8stesting.Action) (bool, runtime.Object, error) {
			return true, &agonesv1.GameServerSetList{Items: []agonesv1.GameServerSet{*gsSet}}, nil
		})

		m.AgonesClient.AddReactor("update", "gameserversets", func(action k8stesting.Action) (bool, runtime.Object, error) {
			updated = true

			ua := action.(k8stesting.UpdateAction)
			gsSet := ua.GetObject().(*agonesv1.GameServerSet)
			assert.Equal(t, f.Spec.Replicas, gsSet.Spec.Replicas)
			assert.Equal(t, f.Spec.Scheduling, gsSet.Spec.Scheduling)
			return true, gsSet, nil
		})

		_, cancel := agtesting.StartInformers(m, c.fleetSynced, c.gameServerSetSynced)
		defer cancel()

		err := c.syncFleet("default/fleet-1")
		assert.Nil(t, err)
		assert.True(t, updated, "gameserverset should have been updated")
		agtesting.AssertEventContains(t, m.FakeRecorder.Events, "ScalingGameServerSet")
	})

	t.Run("gameserverset with different image details", func(t *testing.T) {
		f := defaultFixture()
		f.Spec.Strategy.Type = appsv1.RollingUpdateDeploymentStrategyType
		f.Spec.Template.Spec.Ports = []agonesv1.GameServerPort{{HostPort: 5555}}
		c, m := newFakeController()
		gsSet := f.GameServerSet()
		gsSet.ObjectMeta.Name = "gsSet1"
		gsSet.ObjectMeta.UID = "4321"
		gsSet.Spec.Template.Spec.Ports = []agonesv1.GameServerPort{{HostPort: 7777}}
		gsSet.Spec.Replicas = f.Spec.Replicas
		gsSet.Spec.Scheduling = f.Spec.Scheduling
		gsSet.Status.Replicas = 5
		updated := false
		created := false

		m.AgonesClient.AddReactor("list", "fleets", func(action k8stesting.Action) (bool, runtime.Object, error) {
			return true, &agonesv1.FleetList{Items: []agonesv1.Fleet{*f}}, nil
		})

		m.AgonesClient.AddReactor("list", "gameserversets", func(action k8stesting.Action) (bool, runtime.Object, error) {
			return true, &agonesv1.GameServerSetList{Items: []agonesv1.GameServerSet{*gsSet}}, nil
		})

		m.AgonesClient.AddReactor("create", "gameserversets", func(action k8stesting.Action) (bool, runtime.Object, error) {
			created = true
			ca := action.(k8stesting.CreateAction)
			gsSet := ca.GetObject().(*agonesv1.GameServerSet)
			assert.Equal(t, int32(2), gsSet.Spec.Replicas)
			assert.Equal(t, f.Spec.Template.Spec.Ports[0].HostPort, gsSet.Spec.Template.Spec.Ports[0].HostPort)

			return true, gsSet, nil
		})

		m.AgonesClient.AddReactor("update", "gameserversets", func(action k8stesting.Action) (bool, runtime.Object, error) {
			updated = true
			ua := action.(k8stesting.UpdateAction)
			// separate update of status subresource
			if ua.GetSubresource() != "" {
				assert.Equal(t, ua.GetSubresource(), "status")
				return true, nil, nil
			}
			// update main resource
			gsSet := ua.GetObject().(*agonesv1.GameServerSet)
			assert.Equal(t, int32(3), gsSet.Spec.Replicas)
			assert.Equal(t, "gsSet1", gsSet.ObjectMeta.Name)

			return true, gsSet, nil
		})

		_, cancel := agtesting.StartInformers(m, c.fleetSynced, c.gameServerSetSynced)
		defer cancel()

		err := c.syncFleet("default/fleet-1")
		assert.Nil(t, err)
		assert.True(t, updated, "gameserverset should have been updated")
		assert.True(t, created, "gameserverset should have been created")
		agtesting.AssertEventContains(t, m.FakeRecorder.Events, "ScalingGameServerSet")
		agtesting.AssertEventContains(t, m.FakeRecorder.Events, "CreatingGameServerSet")
	})

	t.Run("error on getting fleet", func(t *testing.T) {
		c, _ := newFakeController()
		c.fleetLister = &fakeFleetListerWithErr{}

		err := c.syncFleet("default/fleet-1")
		assert.EqualError(t, err, "error retrieving fleet fleet-1 from namespace default: err-from-namespace-lister")
	})

	t.Run("error on getting list of GS", func(t *testing.T) {
		f := defaultFixture()
		c, m := newFakeController()
		c.gameServerSetLister = &fakeGSSListerWithErr{}

		m.AgonesClient.AddReactor("list", "fleets", func(action k8stesting.Action) (bool, runtime.Object, error) {
			return true, &agonesv1.FleetList{Items: []agonesv1.Fleet{*f}}, nil
		})

		_, cancel := agtesting.StartInformers(m, c.fleetSynced, c.gameServerSetSynced)
		defer cancel()

		err := c.syncFleet("default/fleet-1")
		assert.EqualError(t, err, "error listing gameserversets for fleet fleet-1: random-err")
	})

	t.Run("fleet not found", func(t *testing.T) {
		c, _ := newFakeController()

		err := c.syncFleet("default/fleet-1")
		assert.Nil(t, err)
	})

	t.Run("fleet invalid strategy type", func(t *testing.T) {
		f := defaultFixture()
		c, m := newFakeController()
		f.Spec.Strategy.Type = "invalid-strategy-type"

		gsSet := f.GameServerSet()
		// make gsSet.Spec.Template and f.Spec.Template different in order to make 'rest' list not empty
		gsSet.Spec.Template.ClusterName = "qqqqqqqqqqqqqqqqqqq"

		m.AgonesClient.AddReactor("list", "fleets", func(action k8stesting.Action) (bool, runtime.Object, error) {
			return true, &agonesv1.FleetList{Items: []agonesv1.Fleet{*f}}, nil
		})

		m.AgonesClient.AddReactor("list", "gameserversets", func(action k8stesting.Action) (bool, runtime.Object, error) {
			return true, &agonesv1.GameServerSetList{Items: []agonesv1.GameServerSet{*gsSet}}, nil
		})

		_, cancel := agtesting.StartInformers(m, c.fleetSynced, c.gameServerSetSynced)
		defer cancel()

		err := c.syncFleet("default/fleet-1")
		assert.EqualError(t, err, "unexpected deployment strategy type: invalid-strategy-type")
	})

	t.Run("error on deleteEmptyGameServerSets", func(t *testing.T) {
		f := defaultFixture()
		c, m := newFakeController()

		gsSet := f.GameServerSet()
		// make gsSet.Spec.Template and f.Spec.Template different in order to make 'rest' list not empty
		gsSet.Spec.Template.ClusterName = "qqqqqqqqqqqqqqqqqqq"

		m.AgonesClient.AddReactor("list", "fleets", func(action k8stesting.Action) (bool, runtime.Object, error) {
			return true, &agonesv1.FleetList{Items: []agonesv1.Fleet{*f}}, nil
		})

		m.AgonesClient.AddReactor("list", "gameserversets", func(action k8stesting.Action) (bool, runtime.Object, error) {
			return true, &agonesv1.GameServerSetList{Items: []agonesv1.GameServerSet{*gsSet}}, nil
		})

		m.AgonesClient.AddReactor("delete", "gameserversets", func(action k8stesting.Action) (bool, runtime.Object, error) {
			return true, nil, errors.New("random-err")
		})

		_, cancel := agtesting.StartInformers(m, c.fleetSynced, c.gameServerSetSynced)
		defer cancel()

		err := c.syncFleet("default/fleet-1")
		assert.EqualError(t, err, "error updating gameserverset : random-err")
	})

	t.Run("error on upsertGameServerSet", func(t *testing.T) {
		f := defaultFixture()
		c, m := newFakeController()

		gsSet := f.GameServerSet()

		m.AgonesClient.AddReactor("list", "fleets", func(action k8stesting.Action) (bool, runtime.Object, error) {
			return true, &agonesv1.FleetList{Items: []agonesv1.Fleet{*f}}, nil
		})

		m.AgonesClient.AddReactor("list", "gameserversets", func(action k8stesting.Action) (bool, runtime.Object, error) {
			return true, &agonesv1.GameServerSetList{Items: []agonesv1.GameServerSet{*gsSet}}, nil
		})

		m.AgonesClient.AddReactor("create", "gameserversets", func(action k8stesting.Action) (bool, runtime.Object, error) {
			return true, nil, errors.New("random-err")
		})

		_, cancel := agtesting.StartInformers(m, c.fleetSynced, c.gameServerSetSynced)
		defer cancel()

		err := c.syncFleet("default/fleet-1")
		assert.EqualError(t, err, "error creating gameserverset for fleet fleet-1: random-err")
	})
}

func TestControllerCreationValidationHandler(t *testing.T) {
	t.Parallel()

	t.Run("invalid JSON", func(t *testing.T) {
		c, _ := newFakeController()
		raw, err := json.Marshal([]byte(`1`))
		require.NoError(t, err)
		review := getAdmissionReview(raw)

		_, err = c.creationValidationHandler(review)
		assert.EqualError(t, err, "error unmarshalling original Fleet json: \"MQ==\": json: cannot unmarshal string into Go value of type v1.Fleet")
	})

	t.Run("invalid fleet", func(t *testing.T) {
		c, _ := newFakeController()
		fixture := agonesv1.Fleet{}

		raw, err := json.Marshal(fixture)
		require.NoError(t, err)
		review := getAdmissionReview(raw)

		result, err := c.creationValidationHandler(review)
		require.NoError(t, err)
		assert.False(t, result.Response.Allowed)
		assert.Equal(t, "Failure", result.Response.Result.Status)
	})

	t.Run("valid fleet", func(t *testing.T) {
		c, _ := newFakeController()

		gsSpec := *defaultGSSpec()
		f := defaultFixture()
		f.Spec.Template = gsSpec

		raw, err := json.Marshal(f)
		require.NoError(t, err)
		review := getAdmissionReview(raw)

		result, err := c.creationValidationHandler(review)
		require.NoError(t, err)
		assert.True(t, result.Response.Allowed)
	})
}

func TestControllerCreationMutationHandler(t *testing.T) {
	t.Parallel()

	t.Run("ok scenario", func(t *testing.T) {
		c, _ := newFakeController()
		fixture := agonesv1.Fleet{}

		raw, err := json.Marshal(fixture)
		require.NoError(t, err)
		review := getAdmissionReview(raw)

		result, err := c.creationMutationHandler(review)
		require.NoError(t, err)
		assert.True(t, result.Response.Allowed)
		assert.Equal(t, admv1beta1.PatchTypeJSONPatch, *result.Response.PatchType)

		patch := &jsonpatch.ByPath{}
		err = json.Unmarshal(result.Response.Patch, patch)
		require.NoError(t, err)

		assertContains := func(patch *jsonpatch.ByPath, op jsonpatch.JsonPatchOperation) {
			found := false
			for _, p := range *patch {
				if assert.ObjectsAreEqualValues(p, op) {
					found = true
				}
			}

			assert.True(t, found, "Could not find operation %#v in patch %v", op, *patch)
		}

		assertContains(patch, jsonpatch.JsonPatchOperation{Operation: "add", Path: "/spec/strategy/type", Value: "RollingUpdate"})
	})

	t.Run("invalid JSON", func(t *testing.T) {
		c, _ := newFakeController()
		raw, err := json.Marshal([]byte(`1`))
		require.NoError(t, err)
		review := getAdmissionReview(raw)

		_, err = c.creationMutationHandler(review)
		assert.EqualError(t, err, "error unmarshalling original Fleet json: \"MQ==\": json: cannot unmarshal string into Go value of type v1.Fleet")
	})
}

func TestControllerRun(t *testing.T) {
	t.Parallel()

	fleet := defaultFixture()
	c, m := newFakeController()
	received := make(chan string)
	defer close(received)

	m.ExtClient.AddReactor("get", "customresourcedefinitions", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, agtesting.NewEstablishedCRD(), nil
	})

	fleetWatch := watch.NewFake()
	m.AgonesClient.AddWatchReactor("fleets", k8stesting.DefaultWatchReactor(fleetWatch, nil))

	gsSetWatch := watch.NewFake()
	m.AgonesClient.AddWatchReactor("gameserversets", k8stesting.DefaultWatchReactor(gsSetWatch, nil))

	c.workerqueue.SyncHandler = func(name string) error {
		received <- name
		return nil
	}

	stop, cancel := agtesting.StartInformers(m, c.fleetSynced)
	defer cancel()

	go func() {
		err := c.Run(1, stop)
		assert.Nil(t, err)
	}()

	f := func() string {
		select {
		case result := <-received:
			return result
		case <-time.After(3 * time.Second):
			assert.FailNow(t, "timeout occurred")
		}
		return ""
	}

	expected, err := cache.MetaNamespaceKeyFunc(fleet)
	require.NoError(t, err)

	// test adding fleet
	fleetWatch.Add(fleet.DeepCopy())
	assert.Equal(t, expected, f())

	// test updating fleet
	fCopy := fleet.DeepCopy()
	fCopy.Spec.Replicas += 10
	fleetWatch.Modify(fCopy)
	assert.Equal(t, expected, f())

	// test add/update of gameserver set
	gsSet := fleet.GameServerSet()
	gsSet.ObjectMeta.Name = "gs1"
	gsSet.ObjectMeta.GenerateName = ""
	gsSetWatch.Add(gsSet)
	assert.Equal(t, expected, f())

	gsSet.Spec.Replicas += 10
	gsSetWatch.Modify(gsSet)
	assert.Equal(t, expected, f())
}

func TestControllerUpdateFleetStatus(t *testing.T) {
	t.Parallel()

	t.Run("update, no errors", func(t *testing.T) {
		fleet := defaultFixture()
		c, m := newFakeController()

		gsSet1 := fleet.GameServerSet()
		gsSet1.ObjectMeta.Name = "gsSet1"
		gsSet1.Status.Replicas = 3
		gsSet1.Status.ReadyReplicas = 2
		gsSet1.Status.ReservedReplicas = 4
		gsSet1.Status.AllocatedReplicas = 1

		gsSet2 := fleet.GameServerSet()
		// nolint:goconst
		gsSet2.ObjectMeta.Name = "gsSet2"
		gsSet2.Status.Replicas = 5
		gsSet2.Status.ReadyReplicas = 5
		gsSet2.Status.ReservedReplicas = 3
		gsSet2.Status.AllocatedReplicas = 2

		m.AgonesClient.AddReactor("list", "gameserversets",
			func(action k8stesting.Action) (bool, runtime.Object, error) {
				return true, &agonesv1.GameServerSetList{Items: []agonesv1.GameServerSet{*gsSet1, *gsSet2}}, nil
			})

		updated := false
		m.AgonesClient.AddReactor("update", "fleets",
			func(action k8stesting.Action) (bool, runtime.Object, error) {
				updated = true
				ua := action.(k8stesting.UpdateAction)
				fleet := ua.GetObject().(*agonesv1.Fleet)

				assert.Equal(t, gsSet1.Status.Replicas+gsSet2.Status.Replicas, fleet.Status.Replicas)
				assert.Equal(t, gsSet1.Status.ReadyReplicas+gsSet2.Status.ReadyReplicas, fleet.Status.ReadyReplicas)
				assert.Equal(t, gsSet1.Status.ReservedReplicas+gsSet2.Status.ReservedReplicas, fleet.Status.ReservedReplicas)
				assert.Equal(t, gsSet1.Status.AllocatedReplicas+gsSet2.Status.AllocatedReplicas, fleet.Status.AllocatedReplicas)
				return true, fleet, nil
			})

		_, cancel := agtesting.StartInformers(m, c.fleetSynced, c.gameServerSetSynced)
		defer cancel()

		err := c.updateFleetStatus(fleet)
		assert.Nil(t, err)
		assert.True(t, updated)
	})

	t.Run("list gameservers returns an error", func(t *testing.T) {
		fleet := defaultFixture()
		c, _ := newFakeController()
		c.gameServerSetLister = &fakeGSSListerWithErr{}

		err := c.updateFleetStatus(fleet)
		assert.EqualError(t, err, "error listing gameserversets for fleet fleet-1: random-err")
	})

	t.Run("fleets getter returns an error", func(t *testing.T) {
		fleet := defaultFixture()
		c, _ := newFakeController()

		c.fleetGetter = &fakeFleetsGetterWithErr{}

		err := c.updateFleetStatus(fleet)

		assert.EqualError(t, err, "err-from-fleet-getter")
	})

}

func TestControllerUpdateFleetPlayerStatus(t *testing.T) {
	t.Parallel()

	utilruntime.FeatureTestMutex.Lock()
	defer utilruntime.FeatureTestMutex.Unlock()

	require.NoError(t, utilruntime.ParseFeatures(string(utilruntime.FeaturePlayerTracking)+"=true"))

	fleet := defaultFixture()
	c, m := newFakeController()

	gsSet1 := fleet.GameServerSet()
	gsSet1.ObjectMeta.Name = "gsSet1"
	gsSet1.Status.Players = &agonesv1.AggregatedPlayerStatus{
		Count:    5,
		Capacity: 10,
	}

	gsSet2 := fleet.GameServerSet()
	gsSet2.ObjectMeta.Name = "gsSet2"
	gsSet2.Status.Players = &agonesv1.AggregatedPlayerStatus{
		Count:    10,
		Capacity: 20,
	}

	m.AgonesClient.AddReactor("list", "gameserversets",
		func(action k8stesting.Action) (bool, runtime.Object, error) {
			return true, &agonesv1.GameServerSetList{Items: []agonesv1.GameServerSet{*gsSet1, *gsSet2}}, nil
		})

	updated := false
	m.AgonesClient.AddReactor("update", "fleets",
		func(action k8stesting.Action) (bool, runtime.Object, error) {
			updated = true
			ua := action.(k8stesting.UpdateAction)
			fleet := ua.GetObject().(*agonesv1.Fleet)

			assert.Equal(t, gsSet1.Status.Players.Count+gsSet2.Status.Players.Count, fleet.Status.Players.Count)
			assert.Equal(t, gsSet1.Status.Players.Capacity+gsSet2.Status.Players.Capacity, fleet.Status.Players.Capacity)

			return true, fleet, nil
		})

	_, cancel := agtesting.StartInformers(m, c.fleetSynced, c.gameServerSetSynced)
	defer cancel()

	err := c.updateFleetStatus(fleet)
	assert.Nil(t, err)
	assert.True(t, updated)
}

func TestControllerFilterGameServerSetByActive(t *testing.T) {
	t.Parallel()

	f := defaultFixture()
	c, _ := newFakeController()
	// the same GameServer Template
	gsSet1 := f.GameServerSet()
	gsSet1.ObjectMeta.Name = "gsSet1"

	// different GameServer Template
	gsSet2 := f.GameServerSet()
	gsSet2.Spec.Template.Spec.Ports = []agonesv1.GameServerPort{{HostPort: 9999}}

	// one active
	active, rest := c.filterGameServerSetByActive(f, []*agonesv1.GameServerSet{gsSet1, gsSet2})
	assert.Equal(t, gsSet1, active)
	assert.Equal(t, []*agonesv1.GameServerSet{gsSet2}, rest)

	// none active
	gsSet1.Spec.Template.Spec.Ports = []agonesv1.GameServerPort{{HostPort: 9999}}
	active, rest = c.filterGameServerSetByActive(f, []*agonesv1.GameServerSet{gsSet1, gsSet2})
	assert.Nil(t, active)
	assert.Equal(t, []*agonesv1.GameServerSet{gsSet1, gsSet2}, rest)
}

func TestControllerRecreateDeployment(t *testing.T) {
	t.Parallel()

	f := defaultFixture()
	f.Spec.Strategy.Type = appsv1.RecreateDeploymentStrategyType
	f.Spec.Replicas = 10
	gsSet1 := f.GameServerSet()
	gsSet1.ObjectMeta.Name = "gsSet1"
	gsSet1.Spec.Replicas = 10
	gsSet2 := f.GameServerSet()
	gsSet2.ObjectMeta.Name = "gsSet2"
	gsSet2.Spec.Replicas = 0
	gsSet2.Status.AllocatedReplicas = 1

	t.Run("ok scenario", func(t *testing.T) {
		c, m := newFakeController()
		updated := false
		m.AgonesClient.AddReactor("update", "gameserversets", func(action k8stesting.Action) (bool, runtime.Object, error) {
			updated = true
			ua := action.(k8stesting.UpdateAction)
			gsSet := ua.GetObject().(*agonesv1.GameServerSet)
			assert.Equal(t, gsSet1.ObjectMeta.Name, gsSet.ObjectMeta.Name)
			assert.Equal(t, int32(0), gsSet.Spec.Replicas)

			return true, gsSet, nil
		})

		replicas, err := c.recreateDeployment(f, []*agonesv1.GameServerSet{gsSet1, gsSet2})

		require.NoError(t, err)
		assert.True(t, updated)
		assert.Equal(t, f.Spec.Replicas-1, replicas)
		agtesting.AssertEventContains(t, m.FakeRecorder.Events, "ScalingGameServerSet")
	})

	t.Run("error on update", func(t *testing.T) {
		c, m := newFakeController()
		m.AgonesClient.AddReactor("update", "gameserversets", func(action k8stesting.Action) (bool, runtime.Object, error) {
			return true, nil, errors.New("random-err")
		})

		_, err := c.recreateDeployment(f, []*agonesv1.GameServerSet{gsSet1, gsSet2})

		assert.EqualError(t, err, "error updating gameserverset gsSet1: random-err")
	})
}

func TestControllerApplyDeploymentStrategy(t *testing.T) {
	t.Parallel()

	type expected struct {
		inactiveReplicas int32
		replicas         int32
	}

	fixtures := map[string]struct {
		strategyType         appsv1.DeploymentStrategyType
		gsSet1StatusReplicas int32
		gsSet2StatusReplicas int32
		expected             expected
	}{
		string(appsv1.RecreateDeploymentStrategyType): {
			strategyType:         appsv1.RecreateDeploymentStrategyType,
			gsSet1StatusReplicas: 0,
			gsSet2StatusReplicas: 0,
			expected: expected{
				inactiveReplicas: 0,
				replicas:         10,
			},
		},
		string(appsv1.RollingUpdateDeploymentStrategyType): {
			strategyType:         appsv1.RecreateDeploymentStrategyType,
			gsSet1StatusReplicas: 10,
			gsSet2StatusReplicas: 1,
			expected: expected{
				inactiveReplicas: 7,
				replicas:         2,
			},
		},
	}

	for k, v := range fixtures {
		t.Run(k, func(t *testing.T) {
			f := defaultFixture()
			f.Spec.Strategy.Type = v.strategyType
			f.Spec.Replicas = 10

			gsSet1 := f.GameServerSet()
			gsSet1.ObjectMeta.Name = "gsSet1"
			gsSet1.Spec.Replicas = 10
			gsSet1.Status.Replicas = v.gsSet1StatusReplicas

			gsSet2 := f.GameServerSet()
			gsSet2.ObjectMeta.Name = "gsSet2"
			gsSet2.Spec.Replicas = 0
			gsSet2.Status.Replicas = v.gsSet2StatusReplicas

			c, m := newFakeController()

			updated := false
			m.AgonesClient.AddReactor("update", "gameserversets", func(action k8stesting.Action) (bool, runtime.Object, error) {
				updated = true
				ua := action.(k8stesting.UpdateAction)
				gsSet := ua.GetObject().(*agonesv1.GameServerSet)
				assert.Equal(t, gsSet1.ObjectMeta.Name, gsSet.ObjectMeta.Name)
				assert.Equal(t, int32(0), gsSet.Spec.Replicas)

				return true, gsSet, nil
			})

			replicas, err := c.applyDeploymentStrategy(f, f.GameServerSet(), []*agonesv1.GameServerSet{gsSet1, gsSet2})
			require.NoError(t, err)
			assert.True(t, updated, "update should happen")
			assert.Equal(t, f.Spec.Replicas, replicas)
		})
	}

	t.Run("a single gameserverset", func(t *testing.T) {
		f := defaultFixture()
		f.Spec.Replicas = 10

		gsSet1 := f.GameServerSet()
		gsSet1.ObjectMeta.Name = "gsSet1"

		c, _ := newFakeController()

		replicas, err := c.applyDeploymentStrategy(f, f.GameServerSet(), []*agonesv1.GameServerSet{})
		require.NoError(t, err)
		assert.Equal(t, f.Spec.Replicas, replicas)
	})
}

func TestControllerUpsertGameServerSet(t *testing.T) {
	t.Parallel()
	f := defaultFixture()
	replicas := int32(10)

	t.Run("insert", func(t *testing.T) {
		c, m := newFakeController()
		gsSet := f.GameServerSet()
		created := false
		m.AgonesClient.AddReactor("create", "gameserversets", func(action k8stesting.Action) (bool, runtime.Object, error) {
			created = true
			ca := action.(k8stesting.CreateAction)
			gsSet := ca.GetObject().(*agonesv1.GameServerSet)
			assert.Equal(t, replicas, gsSet.Spec.Replicas)

			return true, gsSet, nil
		})

		err := c.upsertGameServerSet(f, gsSet, replicas)
		assert.Nil(t, err)

		assert.True(t, created, "Should be created")
		agtesting.AssertEventContains(t, m.FakeRecorder.Events, "CreatingGameServerSet")
	})

	t.Run("update", func(t *testing.T) {
		c, m := newFakeController()
		gsSet := f.GameServerSet()
		gsSet.ObjectMeta.UID = "1234"
		gsSet.Spec.Replicas = replicas + 10
		update := false

		m.AgonesClient.AddReactor("update", "gameserversets", func(action k8stesting.Action) (bool, runtime.Object, error) {
			update = true
			ca := action.(k8stesting.UpdateAction)
			gsSet := ca.GetObject().(*agonesv1.GameServerSet)
			assert.Equal(t, replicas, gsSet.Spec.Replicas)

			return true, gsSet, nil
		})

		err := c.upsertGameServerSet(f, gsSet, replicas)
		assert.Nil(t, err)

		assert.True(t, update, "Should be updated")
		agtesting.AssertEventContains(t, m.FakeRecorder.Events, "ScalingGameServerSet")
	})

	t.Run("error updating gss replicas", func(t *testing.T) {
		c, m := newFakeController()
		gsSet := f.GameServerSet()
		gsSet.ObjectMeta.UID = "1234"
		gsSet.Spec.Replicas = replicas + 10

		m.AgonesClient.AddReactor("update", "gameserversets", func(action k8stesting.Action) (bool, runtime.Object, error) {
			return true, nil, errors.New("random-err")
		})

		err := c.upsertGameServerSet(f, gsSet, replicas)

		assert.EqualError(t, err, "error updating replicas for gameserverset for fleet fleet-1: random-err")
	})

	t.Run("error on gs status update", func(t *testing.T) {
		c, m := newFakeController()
		gsSet := f.GameServerSet()

		m.AgonesClient.AddReactor("update", "gameserversets", func(action k8stesting.Action) (bool, runtime.Object, error) {
			return true, nil, errors.New("random-err")
		})

		err := c.upsertGameServerSet(f, gsSet, replicas)

		assert.EqualError(t, err, "error updating status of gameserverset for fleet fleet-1: random-err")
	})

	t.Run("nothing happens, nil is returned", func(t *testing.T) {
		t.Parallel()

		c, m := newFakeController()
		gsSet := f.GameServerSet()
		gsSet.ObjectMeta.UID = "1234"
		gsSet.Spec.Replicas = replicas

		m.AgonesClient.AddReactor("create", "gameserversets", func(action k8stesting.Action) (bool, runtime.Object, error) {
			assert.FailNow(t, "should not create")
			return false, nil, nil
		})
		m.AgonesClient.AddReactor("update", "gameserversets", func(action k8stesting.Action) (bool, runtime.Object, error) {
			assert.FailNow(t, "should not update")
			return false, nil, nil
		})

		err := c.upsertGameServerSet(f, gsSet, replicas)
		assert.Nil(t, err)
		agtesting.AssertNoEvent(t, m.FakeRecorder.Events)
	})
}

func TestResourcesRequestsAndLimits(t *testing.T) {
	t.Parallel()

	gsSpec := *defaultGSSpec()
	c, _ := newFakeController()
	f := defaultFixture()
	f.Spec.Template = gsSpec
	f.Spec.Template.Spec.Template.Spec.Containers[0].Resources.Limits[corev1.ResourceCPU] = resource.MustParse("1000m")

	// Semantically equal definition, 1 == 1000m CPU
	gsSet1 := f.GameServerSet()
	gsSet1.Spec.Template = gsSpec
	gsSet1.Spec.Template.Spec.Template.Spec.Containers[0].Resources.Limits[corev1.ResourceCPU] = resource.MustParse("1")

	// Absolutely different GameServer Spec, 1.1 CPU
	gsSet3 := f.GameServerSet()
	gsSet3.Spec.Template = *gsSpec.DeepCopy()
	gsSet3.Spec.Template.Spec.Template.Spec.Containers[0].Resources.Limits[corev1.ResourceCPU] = resource.MustParse("1.1")
	active, rest := c.filterGameServerSetByActive(f, []*agonesv1.GameServerSet{gsSet1, gsSet3})
	assert.Equal(t, gsSet1, active)
	assert.Equal(t, []*agonesv1.GameServerSet{gsSet3}, rest)

	gsSet2 := f.GameServerSet()
	gsSet2.Spec.Template = *gsSpec.DeepCopy()
	gsSet2.Spec.Template.Spec.Template.Spec.Containers[0].Resources.Limits[corev1.ResourceCPU] = resource.MustParse("1000m")
	active, rest = c.filterGameServerSetByActive(f, []*agonesv1.GameServerSet{gsSet2, gsSet3})
	assert.Equal(t, gsSet2, active)
	assert.Equal(t, []*agonesv1.GameServerSet{gsSet3}, rest)
}

func TestControllerDeleteEmptyGameServerSets(t *testing.T) {
	t.Parallel()

	f := defaultFixture()
	gsSet1 := f.GameServerSet()
	gsSet1.ObjectMeta.Name = "gsSet1"
	gsSet1.Spec.Replicas = 10
	gsSet1.Status.Replicas = 10
	gsSet2 := f.GameServerSet()
	gsSet2.ObjectMeta.Name = "gsSet2"
	gsSet2.Spec.Replicas = 0
	gsSet2.Status.Replicas = 0

	c, m := newFakeController()
	deleted := false

	m.AgonesClient.AddReactor("delete", "gameserversets", func(action k8stesting.Action) (bool, runtime.Object, error) {
		deleted = true
		da := action.(k8stesting.DeleteAction)
		assert.Equal(t, gsSet2.ObjectMeta.Name, da.GetName())
		return true, nil, nil
	})

	err := c.deleteEmptyGameServerSets(f, []*agonesv1.GameServerSet{gsSet1, gsSet2})
	assert.Nil(t, err)
	assert.True(t, deleted, "delete should happen")
}
func TestControllerRollingUpdateDeploymentNoInactiveGSSNoErrors(t *testing.T) {
	t.Parallel()

	f := defaultFixture()

	f.Spec.Replicas = 100

	active := f.GameServerSet()
	active.ObjectMeta.Name = "active"

	c, _ := newFakeController()

	replicas, err := c.rollingUpdateDeployment(f, active, []*agonesv1.GameServerSet{})
	assert.Nil(t, err)
	assert.Equal(t, int32(25), replicas)
}

func TestControllerRollingUpdateDeploymentGSSUpdateFailedErrExpected(t *testing.T) {
	t.Parallel()

	f := defaultFixture()
	f.Spec.Replicas = 75

	active := f.GameServerSet()
	active.ObjectMeta.Name = "active"
	active.Spec.Replicas = 75
	active.Status.Replicas = 75

	inactive := f.GameServerSet()
	inactive.ObjectMeta.Name = "inactive"
	inactive.Spec.Replicas = 10
	inactive.Status.Replicas = 10
	inactive.Status.AllocatedReplicas = 5

	c, m := newFakeController()

	// triggered inside rollingUpdateRest
	m.AgonesClient.AddReactor("update", "gameserversets", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, errors.New("random-err")
	})

	_, err := c.rollingUpdateDeployment(f, active, []*agonesv1.GameServerSet{inactive})
	assert.EqualError(t, err, "error updating gameserverset inactive: random-err")
}

func TestControllerRollingUpdateDeployment(t *testing.T) {
	t.Parallel()

	type expected struct {
		inactiveSpecReplicas int32
		replicas             int32
		updated              bool
		err                  string
	}

	fixtures := map[string]struct {
		fleetSpecReplicas                int32
		activeSpecReplicas               int32
		activeStatusReplicas             int32
		inactiveSpecReplicas             int32
		inactiveStatusReplicas           int32
		inactiveStatusAllocationReplicas int32
		nilMaxSurge                      bool
		nilMaxUnavailable                bool
		expected                         expected
	}{
		"nil MaxUnavailable, err excpected": {
			fleetSpecReplicas:      100,
			activeSpecReplicas:     0,
			activeStatusReplicas:   0,
			inactiveSpecReplicas:   100,
			inactiveStatusReplicas: 100,
			nilMaxUnavailable:      true,
			expected: expected{
				err: "error calculating scaling gameserverset: fleet-1: nil value for IntOrString",
			},
		},
		"nil MaxSurge, err excpected": {
			fleetSpecReplicas:      100,
			activeSpecReplicas:     0,
			activeStatusReplicas:   0,
			inactiveSpecReplicas:   100,
			inactiveStatusReplicas: 100,
			nilMaxSurge:            true,
			expected: expected{
				err: "error calculating scaling gameserverset: fleet-1: nil value for IntOrString",
			},
		},
		"full inactive, empty inactive": {
			fleetSpecReplicas:      100,
			activeSpecReplicas:     0,
			activeStatusReplicas:   0,
			inactiveSpecReplicas:   100,
			inactiveStatusReplicas: 100,
			expected: expected{
				inactiveSpecReplicas: 70,
				replicas:             25,
				updated:              true,
			},
		},
		"almost empty inactive with allocated, almost full active": {
			fleetSpecReplicas:                100,
			activeSpecReplicas:               75,
			activeStatusReplicas:             75,
			inactiveSpecReplicas:             10,
			inactiveStatusReplicas:           10,
			inactiveStatusAllocationReplicas: 5,

			expected: expected{
				inactiveSpecReplicas: 0,
				replicas:             95,
				updated:              true,
			},
		},
		"attempt to drive replicas over the max surge": {
			fleetSpecReplicas:      100,
			activeSpecReplicas:     25,
			activeStatusReplicas:   25,
			inactiveSpecReplicas:   95,
			inactiveStatusReplicas: 95,
			expected: expected{
				inactiveSpecReplicas: 65,
				replicas:             30,
				updated:              true,
			},
		},
		"statuses don't match the spec. nothing should happen": {
			fleetSpecReplicas:      100,
			activeSpecReplicas:     75,
			activeStatusReplicas:   70,
			inactiveSpecReplicas:   15,
			inactiveStatusReplicas: 10,
			expected: expected{
				inactiveSpecReplicas: 15,
				replicas:             75,
				updated:              false,
			},
		},
		"test smalled numbers of active and allocated": {
			fleetSpecReplicas:                5,
			activeSpecReplicas:               0,
			activeStatusReplicas:             0,
			inactiveSpecReplicas:             5,
			inactiveStatusReplicas:           5,
			inactiveStatusAllocationReplicas: 2,

			expected: expected{
				inactiveSpecReplicas: 3,
				replicas:             2,
				updated:              true,
			},
		},
		"activeSpecReplicas >= (fleetSpecReplicas - inactiveStatusAllocationReplicas)": {
			fleetSpecReplicas:                75,
			activeSpecReplicas:               75,
			activeStatusReplicas:             75,
			inactiveSpecReplicas:             10,
			inactiveStatusReplicas:           10,
			inactiveStatusAllocationReplicas: 5,

			expected: expected{
				inactiveSpecReplicas: 0,
				replicas:             70,
				updated:              true,
			},
		},
	}

	for k, v := range fixtures {
		t.Run(k, func(t *testing.T) {
			f := defaultFixture()

			mu := intstr.FromString("30%")
			f.Spec.Strategy.RollingUpdate.MaxUnavailable = &mu
			f.Spec.Replicas = v.fleetSpecReplicas

			if v.nilMaxSurge {
				f.Spec.Strategy.RollingUpdate.MaxSurge = nil
			} else {
				assert.Equal(t, "25%", f.Spec.Strategy.RollingUpdate.MaxSurge.String())
			}

			if v.nilMaxUnavailable {
				f.Spec.Strategy.RollingUpdate.MaxUnavailable = nil
			} else {
				assert.Equal(t, "30%", f.Spec.Strategy.RollingUpdate.MaxUnavailable.String())
			}

			active := f.GameServerSet()
			active.ObjectMeta.Name = "active"
			active.Spec.Replicas = v.activeSpecReplicas
			active.Status.Replicas = v.activeStatusReplicas

			inactive := f.GameServerSet()
			inactive.ObjectMeta.Name = "inactive"
			inactive.Spec.Replicas = v.inactiveSpecReplicas
			inactive.Status.Replicas = v.inactiveStatusReplicas
			inactive.Status.AllocatedReplicas = v.inactiveStatusAllocationReplicas

			logrus.WithField("inactive", inactive).Info("Setting up the initial inactive")

			updated := false
			c, m := newFakeController()

			m.AgonesClient.AddReactor("update", "gameserversets", func(action k8stesting.Action) (bool, runtime.Object, error) {
				updated = true
				ua := action.(k8stesting.UpdateAction)
				gsSet := ua.GetObject().(*agonesv1.GameServerSet)
				assert.Equal(t, inactive.ObjectMeta.Name, gsSet.ObjectMeta.Name)
				assert.Equal(t, v.expected.inactiveSpecReplicas, gsSet.Spec.Replicas)

				return true, gsSet, nil
			})

			replicas, err := c.rollingUpdateDeployment(f, active, []*agonesv1.GameServerSet{inactive})

			if v.expected.err != "" {
				assert.EqualError(t, err, v.expected.err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, v.expected.replicas, replicas)
				assert.Equal(t, v.expected.updated, updated)
				if updated {
					agtesting.AssertEventContains(t, m.FakeRecorder.Events, "ScalingGameServerSet")
				} else {
					agtesting.AssertNoEvent(t, m.FakeRecorder.Events)
				}
			}
		})
	}
}

// newFakeController returns a controller, backed by the fake Clientset
func newFakeController() (*Controller, agtesting.Mocks) {
	m := agtesting.NewMocks()
	wh := webhooks.NewWebHook(http.NewServeMux())
	c := NewController(wh, healthcheck.NewHandler(), m.KubeClient, m.ExtClient, m.AgonesClient, m.AgonesInformerFactory)
	c.recorder = m.FakeRecorder
	return c, m
}

func defaultFixture() *agonesv1.Fleet {
	f := &agonesv1.Fleet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "fleet-1",
			Namespace: "default",
			UID:       "1234",
		},
		Spec: agonesv1.FleetSpec{
			Replicas:   5,
			Scheduling: apis.Packed,
			Template:   agonesv1.GameServerTemplateSpec{},
		},
	}
	f.ApplyDefaults()
	return f
}

func defaultGSSpec() *agonesv1.GameServerTemplateSpec {
	return &agonesv1.GameServerTemplateSpec{
		Spec: agonesv1.GameServerSpec{
			Container: "udp-server",
			Ports: []agonesv1.GameServerPort{{
				ContainerPort: 7654,
				Name:          "gameport",
				PortPolicy:    agonesv1.Dynamic,
				Protocol:      corev1.ProtocolUDP,
			}},
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Name:            "udp-server",
						Image:           "gcr.io/images/new:0.2",
						ImagePullPolicy: corev1.PullIfNotPresent,
						Resources: corev1.ResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("30m"),
								corev1.ResourceMemory: resource.MustParse("32Mi"),
							},
							Limits: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("30m"),
								corev1.ResourceMemory: resource.MustParse("32Mi"),
							},
						},
					}},
				},
			},
		},
	}
}

func getAdmissionReview(raw []byte) admv1beta1.AdmissionReview {
	gvk := metav1.GroupVersionKind(agonesv1.SchemeGroupVersion.WithKind("Fleet"))

	return admv1beta1.AdmissionReview{
		Request: &admv1beta1.AdmissionRequest{
			Kind:      gvk,
			Operation: admv1beta1.Create,
			Object: runtime.RawExtension{
				Raw: raw,
			},
		},
		Response: &admv1beta1.AdmissionResponse{Allowed: true},
	}
}

// MOCKS SECTION

type fakeGSSListerWithErr struct {
}

// GameServerSetLister interface implementation
func (fgsl *fakeGSSListerWithErr) List(selector labels.Selector) (ret []*v1.GameServerSet, err error) {
	return nil, errors.New("random-err")
}

func (fgsl *fakeGSSListerWithErr) GameServerSets(namespace string) agonesv1client.GameServerSetNamespaceLister {
	panic("not implemented")
}

type fakeFleetsGetterWithErr struct{}

// // FleetsGetter interface implementation
func (ffg *fakeFleetsGetterWithErr) Fleets(namespace string) agonesv1clientset.FleetInterface {
	return &fakeFleetsGetterWithErr{}
}

func (ffg *fakeFleetsGetterWithErr) Create(*v1.Fleet) (*v1.Fleet, error) {
	panic("not implemented")
}
func (ffg *fakeFleetsGetterWithErr) Update(*v1.Fleet) (*v1.Fleet, error) {
	panic("not implemented")
}
func (ffg *fakeFleetsGetterWithErr) UpdateStatus(*v1.Fleet) (*v1.Fleet, error) {
	panic("not implemented")
}
func (ffg *fakeFleetsGetterWithErr) Delete(name string, options *metav1.DeleteOptions) error {
	panic("not implemented")
}
func (ffg *fakeFleetsGetterWithErr) DeleteCollection(options *metav1.DeleteOptions, listOptions metav1.ListOptions) error {
	panic("not implemented")
}
func (ffg *fakeFleetsGetterWithErr) Get(name string, options metav1.GetOptions) (*v1.Fleet, error) {
	return nil, errors.New("err-from-fleet-getter")
}
func (ffg *fakeFleetsGetterWithErr) List(opts metav1.ListOptions) (*v1.FleetList, error) {
	panic("not implemented")
}
func (ffg *fakeFleetsGetterWithErr) Watch(opts metav1.ListOptions) (watch.Interface, error) {
	panic("not implemented")
}
func (ffg *fakeFleetsGetterWithErr) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1.Fleet, err error) {
	panic("not implemented")
}
func (ffg *fakeFleetsGetterWithErr) GetScale(fleetName string, options metav1.GetOptions) (*v1beta1.Scale, error) {
	panic("not implemented")
}
func (ffg *fakeFleetsGetterWithErr) UpdateScale(fleetName string, scale *v1beta1.Scale) (*v1beta1.Scale, error) {
	panic("not implemented")
}

type fakeFleetListerWithErr struct{}

// FleetLister interface implementation
func (ffl *fakeFleetListerWithErr) List(selector labels.Selector) (ret []*v1.Fleet, err error) {
	return nil, errors.New("err-from-fleet-lister")
}

func (ffl *fakeFleetListerWithErr) Fleets(namespace string) agonesv1client.FleetNamespaceLister {
	return &fakeFleetNamespaceListerWithErr{}
}

type fakeFleetNamespaceListerWithErr struct{}

// FleetNamespaceLister interface implementation
func (ffnl *fakeFleetNamespaceListerWithErr) List(selector labels.Selector) (ret []*v1.Fleet, err error) {
	return nil, errors.New("err-from-namespace-lister")
}

func (ffnl *fakeFleetNamespaceListerWithErr) Get(name string) (*v1.Fleet, error) {
	return nil, errors.New("err-from-namespace-lister")
}
