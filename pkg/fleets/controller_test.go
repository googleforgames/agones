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

package fleets

import (
	"testing"
	"time"

	"encoding/json"

	"agones.dev/agones/pkg/apis/stable/v1alpha1"
	agtesting "agones.dev/agones/pkg/testing"
	"agones.dev/agones/pkg/util/webhooks"
	"github.com/heptiolabs/healthcheck"
	"github.com/mattbaird/jsonpatch"
	"github.com/stretchr/testify/assert"
	admv1beta1 "k8s.io/api/admission/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
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
			return true, &v1alpha1.FleetList{Items: []v1alpha1.Fleet{*f}}, nil
		})

		m.AgonesClient.AddReactor("create", "gameserversets", func(action k8stesting.Action) (bool, runtime.Object, error) {
			ca := action.(k8stesting.CreateAction)
			gsSet := ca.GetObject().(*v1alpha1.GameServerSet)

			created = true
			assert.True(t, metav1.IsControlledBy(gsSet, f))

			return true, gsSet, nil
		})

		_, cancel := agtesting.StartInformers(m)
		defer cancel()

		err := c.syncFleet("default/fleet-1")
		assert.Nil(t, err)
		assert.True(t, created, "gameserverset should have been created")

		select {
		case event := <-m.FakeRecorder.Events:
			assert.Contains(t, event, "CreatingGameServerSet")
		case <-time.After(3 * time.Second):
			assert.FailNow(t, "there should be a create event")
		}
	})

	t.Run("gamserverset with the same number of replicas", func(t *testing.T) {
		t.Parallel()
		f := defaultFixture()
		c, m := newFakeController()
		gsSet := f.GameServerSet()
		gsSet.ObjectMeta.Name = "gsSet1"
		gsSet.ObjectMeta.UID = "1234"

		m.AgonesClient.AddReactor("list", "fleets", func(action k8stesting.Action) (bool, runtime.Object, error) {
			return true, &v1alpha1.FleetList{Items: []v1alpha1.Fleet{*f}}, nil
		})

		m.AgonesClient.AddReactor("list", "gameserversets", func(action k8stesting.Action) (bool, runtime.Object, error) {
			return true, &v1alpha1.GameServerSetList{Items: []v1alpha1.GameServerSet{*gsSet}}, nil
		})

		m.AgonesClient.AddReactor("create", "gameserversets", func(action k8stesting.Action) (bool, runtime.Object, error) {
			assert.FailNow(t, "gameserverset should not be created")
			return true, nil, nil
		})

		m.AgonesClient.AddReactor("update", "gameserversets", func(action k8stesting.Action) (bool, runtime.Object, error) {
			assert.FailNow(t, "gameserverset should not have been updated")
			return true, nil, nil
		})

		_, cancel := agtesting.StartInformers(m)
		defer cancel()

		err := c.syncFleet("default/fleet-1")
		assert.Nil(t, err)

		select {
		case event := <-m.FakeRecorder.Events:
			assert.FailNow(t, "there should be no events", event)
		case <-time.After(time.Second):
		}
	})

	t.Run("gameserverset with different number of replicas", func(t *testing.T) {
		f := defaultFixture()
		c, m := newFakeController()
		gsSet := f.GameServerSet()
		gsSet.ObjectMeta.Name = "gsSet1"
		gsSet.ObjectMeta.UID = "1234"
		gsSet.Spec.Replicas = f.Spec.Replicas + 10
		updated := false

		m.AgonesClient.AddReactor("list", "fleets", func(action k8stesting.Action) (bool, runtime.Object, error) {
			return true, &v1alpha1.FleetList{Items: []v1alpha1.Fleet{*f}}, nil
		})

		m.AgonesClient.AddReactor("list", "gameserversets", func(action k8stesting.Action) (bool, runtime.Object, error) {
			return true, &v1alpha1.GameServerSetList{Items: []v1alpha1.GameServerSet{*gsSet}}, nil
		})

		m.AgonesClient.AddReactor("update", "gameserversets", func(action k8stesting.Action) (bool, runtime.Object, error) {
			updated = true

			ua := action.(k8stesting.UpdateAction)
			gsSet := ua.GetObject().(*v1alpha1.GameServerSet)
			assert.Equal(t, f.Spec.Replicas, gsSet.Spec.Replicas)

			return true, gsSet, nil
		})

		_, cancel := agtesting.StartInformers(m)
		defer cancel()

		err := c.syncFleet("default/fleet-1")
		assert.Nil(t, err)
		assert.True(t, updated, "gameserverset should have been updated")
		select {
		case event := <-m.FakeRecorder.Events:
			assert.Contains(t, event, "ScalingGameServerSet")
		case <-time.After(3 * time.Second):
			assert.FailNow(t, "there should be a scaling event")
		}
	})

	t.Run("gameserverset with different image details", func(t *testing.T) {
		f := defaultFixture()
		f.Spec.Template.Spec.HostPort = 5555
		c, m := newFakeController()
		gsSet := f.GameServerSet()
		gsSet.ObjectMeta.Name = "gsSet1"
		gsSet.ObjectMeta.UID = "1234"
		gsSet.Spec.Template.Spec.HostPort = 7777
		gsSet.Status.Replicas = 5
		updated := false
		created := false

		m.AgonesClient.AddReactor("list", "fleets", func(action k8stesting.Action) (bool, runtime.Object, error) {
			return true, &v1alpha1.FleetList{Items: []v1alpha1.Fleet{*f}}, nil
		})

		m.AgonesClient.AddReactor("list", "gameserversets", func(action k8stesting.Action) (bool, runtime.Object, error) {
			return true, &v1alpha1.GameServerSetList{Items: []v1alpha1.GameServerSet{*gsSet}}, nil
		})

		m.AgonesClient.AddReactor("create", "gameserversets", func(action k8stesting.Action) (bool, runtime.Object, error) {
			created = true
			ca := action.(k8stesting.CreateAction)
			gsSet := ca.GetObject().(*v1alpha1.GameServerSet)
			assert.Equal(t, f.Spec.Template.Spec.HostPort, gsSet.Spec.Template.Spec.HostPort)

			return true, gsSet, nil
		})

		m.AgonesClient.AddReactor("update", "gameserversets", func(action k8stesting.Action) (bool, runtime.Object, error) {
			updated = true
			ua := action.(k8stesting.UpdateAction)
			gsSet := ua.GetObject().(*v1alpha1.GameServerSet)
			assert.Equal(t, int32(0), gsSet.Spec.Replicas)
			assert.Equal(t, "gsSet1", gsSet.ObjectMeta.Name)

			return true, gsSet, nil
		})

		_, cancel := agtesting.StartInformers(m)
		defer cancel()

		err := c.syncFleet("default/fleet-1")
		assert.Nil(t, err)
		assert.True(t, updated, "gameserverset should have been updated")
		assert.True(t, created, "gameserverset should have been created")
		select {
		case event := <-m.FakeRecorder.Events:
			assert.Contains(t, event, "CreatingGameServerSet")
		case <-time.After(3 * time.Second):
			assert.FailNow(t, "there should be a creating event")
		}
	})
}

func TestControllerCreationMutationHandler(t *testing.T) {
	t.Parallel()

	c, _ := newFakeController()
	gvk := metav1.GroupVersionKind(v1alpha1.SchemeGroupVersion.WithKind("Fleet"))

	fixture := v1alpha1.Fleet{}

	raw, err := json.Marshal(fixture)
	assert.Nil(t, err)
	review := admv1beta1.AdmissionReview{
		Request: &admv1beta1.AdmissionRequest{
			Kind:      gvk,
			Operation: admv1beta1.Create,
			Object: runtime.RawExtension{
				Raw: raw,
			},
		},
		Response: &admv1beta1.AdmissionResponse{Allowed: true},
	}

	result, err := c.creationMutationHandler(review)
	assert.Nil(t, err)
	assert.True(t, result.Response.Allowed)
	assert.Equal(t, admv1beta1.PatchTypeJSONPatch, *result.Response.PatchType)

	patch := &jsonpatch.ByPath{}
	err = json.Unmarshal(result.Response.Patch, patch)
	assert.Nil(t, err)

	assertContains := func(patch *jsonpatch.ByPath, op jsonpatch.JsonPatchOperation) {
		found := false
		for _, p := range *patch {
			if assert.ObjectsAreEqualValues(p, op) {
				found = true
			}
		}

		assert.True(t, found, "Could not find operation %#v in patch %v", op, *patch)
	}

	assertContains(patch, jsonpatch.JsonPatchOperation{Operation: "add", Path: "/spec/strategy/type", Value: "Recreate"})
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
	assert.Nil(t, err)

	// test adding fleet
	fleetWatch.Add(fleet.DeepCopy())
	assert.Equal(t, expected, f())

	// test updating fleet
	fCopy := fleet.DeepCopy()
	fCopy.Spec.Replicas = fCopy.Spec.Replicas + 10
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

	fleet := defaultFixture()
	c, m := newFakeController()

	gsSet1 := fleet.GameServerSet()
	gsSet1.ObjectMeta.Name = "gsSet1"
	gsSet1.Status.Replicas = 3
	gsSet1.Status.ReadyReplicas = 2
	gsSet1.Status.AllocatedReplicas = 1

	gsSet2 := fleet.GameServerSet()
	gsSet2.ObjectMeta.Name = "gsSet2"
	gsSet2.Status.Replicas = 5
	gsSet2.Status.ReadyReplicas = 5
	gsSet2.Status.AllocatedReplicas = 2

	m.AgonesClient.AddReactor("list", "gameserversets",
		func(action k8stesting.Action) (bool, runtime.Object, error) {
			return true, &v1alpha1.GameServerSetList{Items: []v1alpha1.GameServerSet{*gsSet1, *gsSet2}}, nil
		})

	updated := false
	m.AgonesClient.AddReactor("update", "fleets",
		func(action k8stesting.Action) (bool, runtime.Object, error) {
			updated = true
			ua := action.(k8stesting.UpdateAction)
			fleet := ua.GetObject().(*v1alpha1.Fleet)

			assert.Equal(t, gsSet1.Status.Replicas+gsSet2.Status.Replicas, fleet.Status.Replicas)
			assert.Equal(t, gsSet1.Status.ReadyReplicas+gsSet2.Status.ReadyReplicas, fleet.Status.ReadyReplicas)
			assert.Equal(t, gsSet1.Status.AllocatedReplicas+gsSet2.Status.AllocatedReplicas, fleet.Status.AllocatedReplicas)
			return true, fleet, nil
		})

	_, cancel := agtesting.StartInformers(m, c.fleetSynced)
	defer cancel()

	err := c.updateFleetStatus(fleet)
	assert.Nil(t, err)
	assert.True(t, updated)
}

func TestControllerFilterGameServerSetByActive(t *testing.T) {
	f := defaultFixture()
	c, _ := newFakeController()
	// the same GameServer Template
	gsSet1 := f.GameServerSet()
	gsSet1.ObjectMeta.Name = "gsSet1"

	// different GameServer Template
	gsSet2 := f.GameServerSet()
	gsSet2.Spec.Template.Spec.HostPort = 9999

	// one active
	active, rest := c.filterGameServerSetByActive(f, []*v1alpha1.GameServerSet{gsSet1, gsSet2})
	assert.Equal(t, gsSet1, active)
	assert.Equal(t, []*v1alpha1.GameServerSet{gsSet2}, rest)

	// none active
	gsSet1.Spec.Template.Spec.HostPort = 9999
	active, rest = c.filterGameServerSetByActive(f, []*v1alpha1.GameServerSet{gsSet1, gsSet2})
	assert.Nil(t, active)
	assert.Equal(t, []*v1alpha1.GameServerSet{gsSet1, gsSet2}, rest)
}

func TestControllerRecreateDeployment(t *testing.T) {
	t.Parallel()

	f := defaultFixture()
	gsSet1 := f.GameServerSet()
	gsSet1.ObjectMeta.Name = "gsSet1"
	gsSet1.Spec.Replicas = 10
	gsSet2 := f.GameServerSet()
	gsSet2.ObjectMeta.Name = "gsSet2"
	gsSet2.Spec.Replicas = 0

	c, m := newFakeController()

	updated := false
	m.AgonesClient.AddReactor("update", "gameserversets", func(action k8stesting.Action) (bool, runtime.Object, error) {
		updated = true
		ua := action.(k8stesting.UpdateAction)
		gsSet := ua.GetObject().(*v1alpha1.GameServerSet)
		assert.Equal(t, gsSet1.ObjectMeta.Name, gsSet.ObjectMeta.Name)
		assert.Equal(t, int32(0), gsSet.Spec.Replicas)

		return true, gsSet, nil
	})

	err := c.recreateDeployment([]*v1alpha1.GameServerSet{gsSet1, gsSet2})
	assert.Nil(t, err)
	assert.True(t, updated)
}

func TestControllerApplyDeploymentStrategy(t *testing.T) {
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

	updated := false
	m.AgonesClient.AddReactor("update", "gameserversets", func(action k8stesting.Action) (bool, runtime.Object, error) {
		updated = true
		ua := action.(k8stesting.UpdateAction)
		gsSet := ua.GetObject().(*v1alpha1.GameServerSet)
		assert.Equal(t, gsSet1.ObjectMeta.Name, gsSet.ObjectMeta.Name)
		assert.Equal(t, int32(0), gsSet.Spec.Replicas)

		return true, gsSet, nil
	})

	err := c.applyDeploymentStrategy(f, []*v1alpha1.GameServerSet{gsSet1, gsSet2})
	assert.Nil(t, err)
	assert.True(t, updated, "update should happen")
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
			gsSet := ca.GetObject().(*v1alpha1.GameServerSet)
			assert.Equal(t, replicas, gsSet.Spec.Replicas)

			return true, gsSet, nil
		})

		err := c.upsertGameServerSet(f, gsSet, replicas)
		assert.Nil(t, err)

		assert.True(t, created, "Should be created")
		select {
		case event := <-m.FakeRecorder.Events:
			assert.Contains(t, event, "CreatingGameServerSet")
		case <-time.After(3 * time.Second):
			assert.FailNow(t, "there should be a create event")
		}
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
			gsSet := ca.GetObject().(*v1alpha1.GameServerSet)
			assert.Equal(t, replicas, gsSet.Spec.Replicas)

			return true, gsSet, nil
		})

		err := c.upsertGameServerSet(f, gsSet, replicas)
		assert.Nil(t, err)

		assert.True(t, update, "Should be update")
		select {
		case event := <-m.FakeRecorder.Events:
			assert.Contains(t, event, "ScalingGameServerSet")
		case <-time.After(3 * time.Second):
			assert.FailNow(t, "there should be a create event")
		}
	})

	t.Run("noop", func(t *testing.T) {
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

		select {
		case event := <-m.FakeRecorder.Events:
			assert.Fail(t, "should not have an event", event)
		case <-time.After(1 * time.Second):
		}
	})
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

	err := c.deleteEmptyGameServerSets(f, []*v1alpha1.GameServerSet{gsSet1, gsSet2})
	assert.Nil(t, err)
	assert.True(t, deleted, "delete should happen")
}

// newFakeController returns a controller, backed by the fake Clientset
func newFakeController() (*Controller, agtesting.Mocks) {
	m := agtesting.NewMocks()
	wh := webhooks.NewWebHook("", "")
	c := NewController(wh, healthcheck.NewHandler(), m.KubeClient, m.ExtClient, m.AgonesClient, m.AgonesInformerFactory)
	c.recorder = m.FakeRecorder
	return c, m
}

func defaultFixture() *v1alpha1.Fleet {
	f := &v1alpha1.Fleet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "fleet-1",
			Namespace: "default",
			UID:       "1234",
		},
		Spec: v1alpha1.FleetSpec{
			Replicas: 5,
			Template: v1alpha1.GameServerTemplateSpec{},
		},
	}
	f.ApplyDefaults()
	return f
}
