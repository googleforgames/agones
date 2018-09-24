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

package gameserversets

import (
	"encoding/json"
	"strconv"
	"sync"
	"testing"
	"time"

	"agones.dev/agones/pkg/apis/stable/v1alpha1"
	agtesting "agones.dev/agones/pkg/testing"
	"agones.dev/agones/pkg/util/webhooks"
	"github.com/heptiolabs/healthcheck"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	admv1beta1 "k8s.io/api/admission/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	k8stesting "k8s.io/client-go/testing"
	"k8s.io/client-go/tools/cache"
)

func TestControllerWatchGameServers(t *testing.T) {
	gsSet := defaultFixture()

	c, m := newFakeController()

	received := make(chan string)
	defer close(received)

	m.ExtClient.AddReactor("get", "customresourcedefinitions", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, agtesting.NewEstablishedCRD(), nil
	})
	gsSetWatch := watch.NewFake()
	m.AgonesClient.AddWatchReactor("gameserversets", k8stesting.DefaultWatchReactor(gsSetWatch, nil))
	gsWatch := watch.NewFake()
	m.AgonesClient.AddWatchReactor("gameservers", k8stesting.DefaultWatchReactor(gsWatch, nil))

	c.workerqueue.SyncHandler = func(name string) error {
		received <- name
		return nil
	}

	stop, cancel := agtesting.StartInformers(m, c.gameServerSynced)
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

	expected, err := cache.MetaNamespaceKeyFunc(gsSet)
	assert.Nil(t, err)

	// gsSet add
	logrus.Info("adding gsSet")
	gsSetWatch.Add(gsSet.DeepCopy())
	assert.Nil(t, err)
	assert.Equal(t, expected, f())
	// gsSet update
	logrus.Info("modify gsSet")
	gsSetCopy := gsSet.DeepCopy()
	gsSetCopy.Spec.Replicas = 5
	gsSetWatch.Modify(gsSetCopy)
	assert.Equal(t, expected, f())

	gs := gsSet.GameServer()
	gs.ObjectMeta.Name = "test-gs"
	// gs add
	logrus.Info("add gs")
	gsWatch.Add(gs.DeepCopy())
	assert.Equal(t, expected, f())

	// gs update
	gsCopy := gs.DeepCopy()
	now := metav1.Now()
	gsCopy.ObjectMeta.DeletionTimestamp = &now

	logrus.Info("modify gs - noop")
	gsWatch.Modify(gsCopy.DeepCopy())
	select {
	case <-received:
		assert.Fail(t, "Should be no value")
	case <-time.After(time.Second):
	}

	gsCopy = gs.DeepCopy()
	gsCopy.Status.State = v1alpha1.Unhealthy
	logrus.Info("modify gs - unhealthy")
	gsWatch.Modify(gsCopy.DeepCopy())
	assert.Equal(t, expected, f())

	// gs delete
	logrus.Info("delete gs")
	gsWatch.Delete(gsCopy.DeepCopy())
	assert.Equal(t, expected, f())
}

func TestSyncGameServerSet(t *testing.T) {
	t.Run("adding and deleting unhealthy gameservers", func(t *testing.T) {
		gsSet := defaultFixture()
		list := createGameServers(gsSet, 5)

		// make some as unhealthy
		list[0].Status.State = v1alpha1.Unhealthy

		deleted := false
		count := 0

		c, m := newFakeController()
		m.AgonesClient.AddReactor("list", "gameserversets", func(action k8stesting.Action) (bool, runtime.Object, error) {
			return true, &v1alpha1.GameServerSetList{Items: []v1alpha1.GameServerSet{*gsSet}}, nil
		})
		m.AgonesClient.AddReactor("list", "gameservers", func(action k8stesting.Action) (bool, runtime.Object, error) {
			return true, &v1alpha1.GameServerList{Items: list}, nil
		})

		m.AgonesClient.AddReactor("delete", "gameservers", func(action k8stesting.Action) (bool, runtime.Object, error) {
			da := action.(k8stesting.DeleteAction)
			deleted = true
			assert.Equal(t, "test-0", da.GetName())
			return true, nil, nil
		})
		m.AgonesClient.AddReactor("create", "gameservers", func(action k8stesting.Action) (bool, runtime.Object, error) {
			ca := action.(k8stesting.CreateAction)
			gs := ca.GetObject().(*v1alpha1.GameServer)

			assert.True(t, metav1.IsControlledBy(gs, gsSet))
			count++
			return true, gs, nil
		})

		_, cancel := agtesting.StartInformers(m, c.gameServerSetSynced, c.gameServerSynced)
		defer cancel()

		c.syncGameServerSet(gsSet.ObjectMeta.Namespace + "/" + gsSet.ObjectMeta.Name) // nolint: errcheck

		assert.Equal(t, 5, count)
		assert.True(t, deleted, "A game servers should have been deleted")
	})

	t.Run("removing gamservers", func(t *testing.T) {
		gsSet := defaultFixture()
		list := createGameServers(gsSet, 15)
		count := 0

		c, m := newFakeController()
		m.AgonesClient.AddReactor("list", "gameserversets", func(action k8stesting.Action) (bool, runtime.Object, error) {
			return true, &v1alpha1.GameServerSetList{Items: []v1alpha1.GameServerSet{*gsSet}}, nil
		})
		m.AgonesClient.AddReactor("list", "gameservers", func(action k8stesting.Action) (bool, runtime.Object, error) {
			return true, &v1alpha1.GameServerList{Items: list}, nil
		})
		m.AgonesClient.AddReactor("delete", "gameservers", func(action k8stesting.Action) (bool, runtime.Object, error) {
			count++
			return true, nil, nil
		})

		_, cancel := agtesting.StartInformers(m, c.gameServerSetSynced, c.gameServerSynced)
		defer cancel()

		c.syncGameServerSet(gsSet.ObjectMeta.Namespace + "/" + gsSet.ObjectMeta.Name) // nolint: errcheck

		assert.Equal(t, 5, count)
	})
}

func TestControllerSyncUnhealthyGameServers(t *testing.T) {
	gsSet := defaultFixture()

	gs1 := gsSet.GameServer()
	gs1.ObjectMeta.Name = "test-1"
	gs1.Status = v1alpha1.GameServerStatus{State: v1alpha1.Unhealthy}

	gs2 := gsSet.GameServer()
	gs2.ObjectMeta.Name = "test-2"
	gs2.Status = v1alpha1.GameServerStatus{State: v1alpha1.Ready}

	gs3 := gsSet.GameServer()
	gs3.ObjectMeta.Name = "test-3"
	now := metav1.Now()
	gs3.ObjectMeta.DeletionTimestamp = &now
	gs3.Status = v1alpha1.GameServerStatus{State: v1alpha1.Ready}

	deleted := false

	c, m := newFakeController()
	m.AgonesClient.AddReactor("delete", "gameservers", func(action k8stesting.Action) (bool, runtime.Object, error) {
		deleted = true
		da := action.(k8stesting.DeleteAction)
		assert.Equal(t, gs1.ObjectMeta.Name, da.GetName())

		return true, nil, nil
	})

	_, cancel := agtesting.StartInformers(m)
	defer cancel()

	err := c.syncUnhealthyGameServers(gsSet, []*v1alpha1.GameServer{gs1, gs2, gs3})
	assert.Nil(t, err)

	assert.True(t, deleted, "Deletion should have occured")
}

func TestSyncMoreGameServers(t *testing.T) {
	gsSet := defaultFixture()

	c, m := newFakeController()
	count := 0
	expected := 10

	m.AgonesClient.AddReactor("create", "gameservers", func(action k8stesting.Action) (bool, runtime.Object, error) {
		ca := action.(k8stesting.CreateAction)
		gs := ca.GetObject().(*v1alpha1.GameServer)

		assert.True(t, metav1.IsControlledBy(gs, gsSet))
		count++

		return true, gs, nil
	})

	_, cancel := agtesting.StartInformers(m)
	defer cancel()

	err := c.syncMoreGameServers(gsSet, int32(expected))
	assert.Nil(t, err)
	assert.Equal(t, expected, count)
	agtesting.AssertEventContains(t, m.FakeRecorder.Events, "SuccessfulCreate")
}

func TestSyncLessGameServers(t *testing.T) {
	gsSet := defaultFixture()

	c, m := newFakeController()
	count := 0
	expected := 5

	list := createGameServers(gsSet, 11)

	// make some as unhealthy
	list[0].Status.State = v1alpha1.Allocated
	list[3].Status.State = v1alpha1.Allocated

	// make the last one already being deleted
	now := metav1.Now()
	list[10].ObjectMeta.DeletionTimestamp = &now

	// gate
	assert.Equal(t, v1alpha1.Allocated, list[0].Status.State)
	assert.Equal(t, v1alpha1.Allocated, list[3].Status.State)
	assert.False(t, list[10].ObjectMeta.DeletionTimestamp.IsZero())

	m.AgonesClient.AddReactor("list", "gameservers", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, &v1alpha1.GameServerList{Items: list}, nil
	})
	m.AgonesClient.AddReactor("delete", "gameservers", func(action k8stesting.Action) (bool, runtime.Object, error) {
		da := action.(k8stesting.DeleteAction)

		found := false
		for _, gs := range list {
			if gs.ObjectMeta.Name == da.GetName() {
				found = true
				assert.NotEqual(t, gs.Status.State, v1alpha1.Allocated)
			}
		}
		assert.True(t, found)
		count++

		return true, nil, nil
	})

	_, cancel := agtesting.StartInformers(m)
	defer cancel()

	list2, err := ListGameServersByGameServerSetOwner(c.gameServerLister, gsSet)
	assert.Nil(t, err)
	assert.Len(t, list2, 11)

	err = c.syncLessGameSevers(gsSet, int32(-expected))
	assert.Nil(t, err)

	// subtract one, because one is already deleted
	assert.Equal(t, expected-1, count)
	agtesting.AssertEventContains(t, m.FakeRecorder.Events, "SuccessfulDelete")
}

func TestControllerSyncGameServerSetState(t *testing.T) {
	t.Parallel()

	t.Run("empty list", func(t *testing.T) {
		gsSet := defaultFixture()
		c, m := newFakeController()

		updated := false
		m.AgonesClient.AddReactor("update", "gameserversets", func(action k8stesting.Action) (bool, runtime.Object, error) {
			updated = true
			return true, nil, nil
		})

		err := c.syncGameServerSetState(gsSet, nil)
		assert.Nil(t, err)
		assert.False(t, updated)
	})

	t.Run("all ready list", func(t *testing.T) {
		gsSet := defaultFixture()
		c, m := newFakeController()

		updated := false
		m.AgonesClient.AddReactor("update", "gameserversets", func(action k8stesting.Action) (bool, runtime.Object, error) {
			updated = true
			ua := action.(k8stesting.UpdateAction)
			gsSet := ua.GetObject().(*v1alpha1.GameServerSet)

			assert.Equal(t, int32(1), gsSet.Status.Replicas)
			assert.Equal(t, int32(1), gsSet.Status.ReadyReplicas)
			assert.Equal(t, int32(0), gsSet.Status.AllocatedReplicas)

			return true, nil, nil
		})

		list := []*v1alpha1.GameServer{{Status: v1alpha1.GameServerStatus{State: v1alpha1.Ready}}}
		err := c.syncGameServerSetState(gsSet, list)
		assert.Nil(t, err)
		assert.True(t, updated)
	})

	t.Run("only some ready list", func(t *testing.T) {
		gsSet := defaultFixture()
		c, m := newFakeController()

		updated := false
		m.AgonesClient.AddReactor("update", "gameserversets", func(action k8stesting.Action) (bool, runtime.Object, error) {
			updated = true
			ua := action.(k8stesting.UpdateAction)
			gsSet := ua.GetObject().(*v1alpha1.GameServerSet)

			assert.Equal(t, int32(8), gsSet.Status.Replicas)
			assert.Equal(t, int32(1), gsSet.Status.ReadyReplicas)
			assert.Equal(t, int32(2), gsSet.Status.AllocatedReplicas)

			return true, nil, nil
		})

		list := []*v1alpha1.GameServer{
			{Status: v1alpha1.GameServerStatus{State: v1alpha1.Ready}},
			{Status: v1alpha1.GameServerStatus{State: v1alpha1.Starting}},
			{Status: v1alpha1.GameServerStatus{State: v1alpha1.Unhealthy}},
			{Status: v1alpha1.GameServerStatus{State: v1alpha1.PortAllocation}},
			{Status: v1alpha1.GameServerStatus{State: v1alpha1.Error}},
			{Status: v1alpha1.GameServerStatus{State: v1alpha1.Creating}},
			{Status: v1alpha1.GameServerStatus{State: v1alpha1.Allocated}},
			{Status: v1alpha1.GameServerStatus{State: v1alpha1.Allocated}},
		}
		err := c.syncGameServerSetState(gsSet, list)
		assert.Nil(t, err)
		assert.True(t, updated)
	})
}

func TestControllerUpdateValidationHandler(t *testing.T) {
	t.Parallel()

	c, _ := newFakeController()
	gvk := metav1.GroupVersionKind(v1alpha1.SchemeGroupVersion.WithKind("GameServerSet"))
	fixture := &v1alpha1.GameServerSet{ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "default"},
		Spec: v1alpha1.GameServerSetSpec{Replicas: 5},
	}
	raw, err := json.Marshal(fixture)
	assert.Nil(t, err)

	t.Run("valid gameserverset update", func(t *testing.T) {
		new := fixture.DeepCopy()
		new.Spec.Replicas = 10
		newRaw, err := json.Marshal(new)
		assert.Nil(t, err)

		review := admv1beta1.AdmissionReview{
			Request: &admv1beta1.AdmissionRequest{
				Kind:      gvk,
				Operation: admv1beta1.Create,
				Object: runtime.RawExtension{
					Raw: newRaw,
				},
				OldObject: runtime.RawExtension{
					Raw: raw,
				},
			},
			Response: &admv1beta1.AdmissionResponse{Allowed: true},
		}

		result, err := c.updateValidationHandler(review)
		assert.Nil(t, err)
		assert.True(t, result.Response.Allowed)
	})

	t.Run("invalid gameserverset update", func(t *testing.T) {
		new := fixture.DeepCopy()
		new.Spec.Template = v1alpha1.GameServerTemplateSpec{
			Spec: v1alpha1.GameServerSpec{
				Ports: []v1alpha1.GameServerPort{{PortPolicy: v1alpha1.Static}},
			},
		}
		newRaw, err := json.Marshal(new)
		assert.Nil(t, err)

		assert.NotEqual(t, string(raw), string(newRaw))

		review := admv1beta1.AdmissionReview{
			Request: &admv1beta1.AdmissionRequest{
				Kind:      gvk,
				Operation: admv1beta1.Create,
				Object: runtime.RawExtension{
					Raw: newRaw,
				},
				OldObject: runtime.RawExtension{
					Raw: raw,
				},
			},
			Response: &admv1beta1.AdmissionResponse{Allowed: true},
		}

		logrus.Info("here?")
		result, err := c.updateValidationHandler(review)
		assert.Nil(t, err)
		assert.False(t, result.Response.Allowed)
		assert.Equal(t, metav1.StatusFailure, result.Response.Result.Status)
		assert.Equal(t, metav1.StatusReasonInvalid, result.Response.Result.Reason)
	})
}

// defaultFixture creates the default GameServerSet fixture
func defaultFixture() *v1alpha1.GameServerSet {
	gsSet := &v1alpha1.GameServerSet{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "test", UID: "1234"},
		Spec: v1alpha1.GameServerSetSpec{
			Replicas: 10,
			Template: v1alpha1.GameServerTemplateSpec{},
		},
	}
	return gsSet
}

// createGameServers create an array of GameServers from the GameServerSet
func createGameServers(gsSet *v1alpha1.GameServerSet, size int) []v1alpha1.GameServer {
	var list []v1alpha1.GameServer
	for i := 0; i < size; i++ {
		gs := gsSet.GameServer()
		gs.Name = gs.GenerateName + strconv.Itoa(i)
		gs.Status = v1alpha1.GameServerStatus{State: v1alpha1.Ready}
		list = append(list, *gs)
	}
	return list
}

// newFakeController returns a controller, backed by the fake Clientset
func newFakeController() (*Controller, agtesting.Mocks) {
	m := agtesting.NewMocks()
	wh := webhooks.NewWebHook("", "")
	c := NewController(wh, healthcheck.NewHandler(), &sync.Mutex{}, m.KubeClient, m.ExtClient, m.AgonesClient, m.AgonesInformerFactory)
	c.recorder = m.FakeRecorder
	return c, m
}
