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

	"agones.dev/agones/pkg/apis/stable/v1alpha1"
	agtesting "agones.dev/agones/pkg/testing"
	"github.com/heptiolabs/healthcheck"
	"github.com/stretchr/testify/assert"
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
		assert.Contains(t, <-m.FakeRecorder.Events, "CreatingGameServerSet")
	})

	t.Run("gamserverset with the same number of replicas", func(t *testing.T) {
		t.Parallel()
		f := defaultFixture()
		c, m := newFakeController()
		gsSet := f.GameServerSet()
		gsSet.ObjectMeta.Name = "gsSet1"
		updated := false

		m.AgonesClient.AddReactor("list", "fleets", func(action k8stesting.Action) (bool, runtime.Object, error) {
			return true, &v1alpha1.FleetList{Items: []v1alpha1.Fleet{*f}}, nil
		})

		m.AgonesClient.AddReactor("list", "gameserversets", func(action k8stesting.Action) (bool, runtime.Object, error) {
			return true, &v1alpha1.GameServerSetList{Items: []v1alpha1.GameServerSet{*gsSet}}, nil
		})

		m.AgonesClient.AddReactor("update", "gameserversets", func(action k8stesting.Action) (bool, runtime.Object, error) {
			updated = true
			return true, nil, nil
		})

		_, cancel := agtesting.StartInformers(m)
		defer cancel()

		err := c.syncFleet("default/fleet-1")
		assert.Nil(t, err)
		assert.False(t, updated, "gameserverset should not have been updated")

		select {
		case <-m.FakeRecorder.Events:
			assert.FailNow(t, "there should be no events")
		case <-time.After(time.Second):
		}
	})

	t.Run("gameserverset with different number of replicas", func(t *testing.T) {
		f := defaultFixture()
		c, m := newFakeController()
		gsSet := f.GameServerSet()
		gsSet.ObjectMeta.Name = "gsSet1"
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
		assert.Contains(t, <-m.FakeRecorder.Events, "ScalingGameServerSet")
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

	gsSet2 := fleet.GameServerSet()
	gsSet2.ObjectMeta.Name = "gsSet2"
	gsSet2.Status.Replicas = 5
	gsSet2.Status.ReadyReplicas = 5

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
			return true, fleet, nil
		})

	_, cancel := agtesting.StartInformers(m, c.fleetSynced)
	defer cancel()

	err := c.updateFleetStatus(fleet)
	assert.Nil(t, err)
	assert.True(t, updated)
}

// newFakeController returns a controller, backed by the fake Clientset
func newFakeController() (*Controller, agtesting.Mocks) {
	m := agtesting.NewMocks()
	c := NewController(healthcheck.NewHandler(), m.KubeClient, m.ExtClient, m.AgonesClient, m.AgonesInformerFactory)
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
	return f
}
