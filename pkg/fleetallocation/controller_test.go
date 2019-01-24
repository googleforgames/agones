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

package fleetallocation

import (
	"encoding/json"
	"fmt"
	"strconv"
	"sync"
	"testing"

	"agones.dev/agones/pkg/apis/stable/v1alpha1"
	agtesting "agones.dev/agones/pkg/testing"
	"agones.dev/agones/pkg/util/webhooks"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	admv1beta1 "k8s.io/api/admission/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	k8stesting "k8s.io/client-go/testing"
)

var (
	gvk = metav1.GroupVersionKind(v1alpha1.SchemeGroupVersion.WithKind("FleetAllocation"))
)

func TestControllerCreationMutationHandler(t *testing.T) {
	t.Parallel()
	f, gsSet, gsList := defaultFixtures(3)

	fa := v1alpha1.FleetAllocation{ObjectMeta: metav1.ObjectMeta{Name: "fa-1"},
		Spec: v1alpha1.FleetAllocationSpec{FleetName: f.ObjectMeta.Name}}

	c, m := newFakeController()

	m.AgonesClient.AddReactor("list", "fleets", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, &v1alpha1.FleetList{Items: []v1alpha1.Fleet{*f}}, nil
	})
	m.AgonesClient.AddReactor("list", "gameserversets", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, &v1alpha1.GameServerSetList{Items: []v1alpha1.GameServerSet{*gsSet}}, nil
	})
	m.AgonesClient.AddReactor("list", "gameservers", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, &v1alpha1.GameServerList{Items: gsList}, nil
	})

	alpha1 := m.AgonesInformerFactory.Stable().V1alpha1()
	fltSynced := alpha1.Fleets().Informer().HasSynced
	gsSetSynced := alpha1.GameServerSets().Informer().HasSynced
	gsSynced := alpha1.GameServers().Informer().HasSynced
	_, cancel := agtesting.StartInformers(m, fltSynced, gsSetSynced, gsSynced)
	defer cancel()

	review, err := newAdmissionReview(fa)
	assert.Nil(t, err)

	result, err := c.creationMutationHandler(review)
	assert.Nil(t, err)
	assert.True(t, result.Response.Allowed, fmt.Sprintf("%#v", result.Response))
	assert.Equal(t, admv1beta1.PatchTypeJSONPatch, *result.Response.PatchType)
	assert.Contains(t, string(result.Response.Patch), "/status/gameServer")
	assert.Contains(t, string(result.Response.Patch), "/metadata/ownerReferences")
}

func TestControllerCreationValidationHandler(t *testing.T) {
	t.Parallel()

	c, _ := newFakeController()

	t.Run("fleet allocation has a gameserver", func(t *testing.T) {
		fa := v1alpha1.FleetAllocation{ObjectMeta: metav1.ObjectMeta{Name: "fa-1", Namespace: "default"},
			Spec:   v1alpha1.FleetAllocationSpec{FleetName: "doesnotexist"},
			Status: v1alpha1.FleetAllocationStatus{GameServer: &v1alpha1.GameServer{}},
		}

		review, err := newAdmissionReview(fa)
		assert.Nil(t, err)

		result, err := c.creationValidationHandler(review)
		assert.Nil(t, err)
		assert.True(t, result.Response.Allowed)
	})

	t.Run("fleet allocation does not have a gameserver", func(t *testing.T) {
		fa := v1alpha1.FleetAllocation{ObjectMeta: metav1.ObjectMeta{Name: "fa-1", Namespace: "default"},
			Spec: v1alpha1.FleetAllocationSpec{FleetName: "doesnotexist"},
		}

		review, err := newAdmissionReview(fa)
		assert.Nil(t, err)

		result, err := c.creationValidationHandler(review)
		assert.Nil(t, err)
		assert.False(t, result.Response.Allowed)
		assert.Equal(t, "fleetName", result.Response.Result.Details.Causes[0].Field)
	})
}

func TestControllerMutationValidationHandler(t *testing.T) {
	t.Parallel()
	c, _ := newFakeController()

	fa := v1alpha1.FleetAllocation{ObjectMeta: metav1.ObjectMeta{Name: "fa-1", Namespace: "default"},
		Spec: v1alpha1.FleetAllocationSpec{FleetName: "my-fleet-name"},
	}

	t.Run("same fleetName", func(t *testing.T) {
		review, err := newAdmissionReview(fa)
		assert.Nil(t, err)
		review.Request.OldObject = *review.Request.Object.DeepCopy()

		result, err := c.mutationValidationHandler(review)
		assert.Nil(t, err)
		assert.True(t, result.Response.Allowed)
	})

	t.Run("different fleetname", func(t *testing.T) {
		review, err := newAdmissionReview(fa)
		assert.Nil(t, err)
		oldObject := fa.DeepCopy()
		oldObject.Spec.FleetName = "changed"

		json, err := json.Marshal(oldObject)
		assert.Nil(t, err)
		review.Request.OldObject = runtime.RawExtension{Raw: json}

		result, err := c.mutationValidationHandler(review)
		assert.Nil(t, err)
		assert.False(t, result.Response.Allowed)
		assert.Equal(t, metav1.StatusReasonInvalid, result.Response.Result.Reason)
		assert.NotNil(t, result.Response.Result.Details)
	})
}

func TestControllerAllocate(t *testing.T) {
	t.Parallel()

	f, gsSet, gsList := defaultFixtures(4)
	c, m := newFakeController()
	n := metav1.Now()
	l := map[string]string{"mode": "deathmatch"}
	a := map[string]string{"map": "searide"}
	fam := &v1alpha1.MetaPatch{Labels: l, Annotations: a}

	gsList[3].ObjectMeta.DeletionTimestamp = &n

	m.AgonesClient.AddReactor("list", "fleets", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, &v1alpha1.FleetList{Items: []v1alpha1.Fleet{*f}}, nil
	})
	m.AgonesClient.AddReactor("list", "gameserversets", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, &v1alpha1.GameServerSetList{Items: []v1alpha1.GameServerSet{*gsSet}}, nil
	})
	m.AgonesClient.AddReactor("list", "gameservers", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, &v1alpha1.GameServerList{Items: gsList}, nil
	})

	updated := false
	gsWatch := watch.NewFake()
	m.AgonesClient.AddWatchReactor("gameservers", k8stesting.DefaultWatchReactor(gsWatch, nil))
	m.AgonesClient.AddReactor("update", "gameservers", func(action k8stesting.Action) (bool, runtime.Object, error) {
		ua := action.(k8stesting.UpdateAction)
		gs := ua.GetObject().(*v1alpha1.GameServer)
		updated = true
		assert.Equal(t, v1alpha1.GameServerStateAllocated, gs.Status.State)
		gsWatch.Modify(gs)

		return true, gs, nil
	})

	_, cancel := agtesting.StartInformers(m)
	defer cancel()

	gs, err := c.allocate(f, fam)
	assert.Nil(t, err)
	assert.Equal(t, v1alpha1.GameServerStateAllocated, gs.Status.State)
	assert.True(t, updated)
	for key, value := range fam.Labels {
		v, ok := gs.ObjectMeta.Labels[key]
		assert.True(t, ok)
		assert.Equal(t, v, value)
	}
	for key, value := range fam.Annotations {
		v, ok := gs.ObjectMeta.Annotations[key]
		assert.True(t, ok)
		assert.Equal(t, v, value)
	}

	updated = false
	gs, err = c.allocate(f, nil)
	assert.Nil(t, err)
	assert.Equal(t, v1alpha1.GameServerStateAllocated, gs.Status.State)
	assert.True(t, updated)

	updated = false
	gs, err = c.allocate(f, nil)
	assert.Nil(t, err)
	assert.Equal(t, v1alpha1.GameServerStateAllocated, gs.Status.State)
	assert.True(t, updated)

	updated = false
	_, err = c.allocate(f, nil)
	assert.NotNil(t, err)
	assert.Equal(t, ErrNoGameServerReady, err)
	assert.False(t, updated)
}

func TestControllerAllocatePriority(t *testing.T) {
	t.Parallel()

	n1 := "node1"
	n2 := "node2"

	run := func(t *testing.T, name string, test func(t *testing.T, c *Controller, fleet *v1alpha1.Fleet)) {
		f, gsSet, gsList := defaultFixtures(4)
		c, m := newFakeController()

		gsList[0].Status.NodeName = n1
		gsList[1].Status.NodeName = n2
		gsList[2].Status.NodeName = n1
		gsList[3].Status.NodeName = n1

		m.AgonesClient.AddReactor("list", "fleets", func(action k8stesting.Action) (bool, runtime.Object, error) {
			return true, &v1alpha1.FleetList{Items: []v1alpha1.Fleet{*f}}, nil
		})
		m.AgonesClient.AddReactor("list", "gameserversets", func(action k8stesting.Action) (bool, runtime.Object, error) {
			return true, &v1alpha1.GameServerSetList{Items: []v1alpha1.GameServerSet{*gsSet}}, nil
		})
		m.AgonesClient.AddReactor("list", "gameservers", func(action k8stesting.Action) (bool, runtime.Object, error) {
			return true, &v1alpha1.GameServerList{Items: gsList}, nil
		})

		gsWatch := watch.NewFake()
		m.AgonesClient.AddWatchReactor("gameservers", k8stesting.DefaultWatchReactor(gsWatch, nil))
		m.AgonesClient.AddReactor("update", "gameservers", func(action k8stesting.Action) (bool, runtime.Object, error) {
			ua := action.(k8stesting.UpdateAction)
			gs := ua.GetObject().(*v1alpha1.GameServer)
			gsWatch.Modify(gs)
			return true, gs, nil
		})

		_, cancel := agtesting.StartInformers(m)
		defer cancel()

		t.Run(name, func(t *testing.T) {
			test(t, c, f)
		})
	}

	run(t, "packed", func(t *testing.T, c *Controller, f *v1alpha1.Fleet) {
		// priority should be node1, then node2
		gs, err := c.allocate(f, nil)
		assert.Nil(t, err)
		assert.Equal(t, n1, gs.Status.NodeName)

		gs, err = c.allocate(f, nil)
		assert.Nil(t, err)
		assert.Equal(t, n1, gs.Status.NodeName)

		gs, err = c.allocate(f, nil)
		assert.Nil(t, err)
		assert.Equal(t, n1, gs.Status.NodeName)

		gs, err = c.allocate(f, nil)
		assert.Nil(t, err)
		assert.Equal(t, n2, gs.Status.NodeName)

		// should have none left
		_, err = c.allocate(f, nil)
		assert.NotNil(t, err)
	})

	run(t, "distributed", func(t *testing.T, c *Controller, f *v1alpha1.Fleet) {
		// make a copy, to avoid the race check
		f = f.DeepCopy()
		f.Spec.Scheduling = v1alpha1.Distributed
		// should go node2, then node1
		gs, err := c.allocate(f, nil)
		assert.Nil(t, err)
		assert.Equal(t, n2, gs.Status.NodeName)

		gs, err = c.allocate(f, nil)
		assert.Nil(t, err)
		assert.Equal(t, n1, gs.Status.NodeName)

		gs, err = c.allocate(f, nil)
		assert.Nil(t, err)
		assert.Equal(t, n1, gs.Status.NodeName)

		gs, err = c.allocate(f, nil)
		assert.Nil(t, err)
		assert.Equal(t, n1, gs.Status.NodeName)

		// should have none left
		_, err = c.allocate(f, nil)
		assert.NotNil(t, err)
	})
}

func TestControllerAllocateMutex(t *testing.T) {
	t.Parallel()

	f, gsSet, gsList := defaultFixtures(100)
	c, m := newFakeController()

	m.AgonesClient.AddReactor("list", "fleets", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, &v1alpha1.FleetList{Items: []v1alpha1.Fleet{*f}}, nil
	})
	m.AgonesClient.AddReactor("list", "gameserversets", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, &v1alpha1.GameServerSetList{Items: []v1alpha1.GameServerSet{*gsSet}}, nil
	})
	m.AgonesClient.AddReactor("list", "gameservers", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, &v1alpha1.GameServerList{Items: gsList}, nil
	})

	//m.AgonesClient.AddReactor("update", "gameservers", func(action k8stesting.Action) (bool, runtime.Object, error) {
	//	ua := action.(k8stesting.UpdateAction)
	//	return true, ua.GetObject(), nil
	//})

	m.AgonesClient.AddWatchReactor("gameservers", k8stesting.DefaultWatchReactor(watch.NewFake(), nil))
	m.AgonesClient.AddWatchReactor("gameserversets", k8stesting.DefaultWatchReactor(watch.NewFake(), nil))
	m.AgonesClient.AddWatchReactor("fleets", k8stesting.DefaultWatchReactor(watch.NewFake(), nil))
	m.AgonesClient.AddWatchReactor("fleetallocations", k8stesting.DefaultWatchReactor(watch.NewFake(), nil))

	_, cancel := agtesting.StartInformers(m)
	defer cancel()

	wg := sync.WaitGroup{}
	// start 10 threads, each one gets 10 allocations
	allocate := func() {
		defer wg.Done()
		for i := 1; i <= 10; i++ {
			_, err := c.allocate(f, nil)
			assert.Nil(t, err)
		}
	}

	for i := 1; i <= 10; i++ {
		wg.Add(1)
		go allocate()
	}

	logrus.Info("waiting...")
	wg.Wait()
}

func defaultFixtures(gsLen int) (*v1alpha1.Fleet, *v1alpha1.GameServerSet, []v1alpha1.GameServer) {
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
	gsSet := f.GameServerSet()
	gsSet.ObjectMeta.Name = "gsSet1"
	var gsList []v1alpha1.GameServer
	for i := 1; i <= gsLen; i++ {
		gs := gsSet.GameServer()
		gs.ObjectMeta.Name = "gs" + strconv.Itoa(i)
		gs.Status.State = v1alpha1.GameServerStateReady
		gsList = append(gsList, *gs)
	}
	return f, gsSet, gsList
}

// newFakeController returns a controller, backed by the fake Clientset
func newFakeController() (*Controller, agtesting.Mocks) {
	m := agtesting.NewMocks()
	wh := webhooks.NewWebHook("", "")
	c := NewController(wh, &sync.Mutex{}, m.KubeClient, m.ExtClient, m.AgonesClient, m.AgonesInformerFactory)
	c.recorder = m.FakeRecorder
	return c, m
}

func newAdmissionReview(fa v1alpha1.FleetAllocation) (admv1beta1.AdmissionReview, error) {
	raw, err := json.Marshal(fa)
	if err != nil {
		return admv1beta1.AdmissionReview{}, err
	}
	review := admv1beta1.AdmissionReview{
		Request: &admv1beta1.AdmissionRequest{
			Kind:      gvk,
			Operation: admv1beta1.Create,
			Object: runtime.RawExtension{
				Raw: raw,
			},
			Namespace: "default",
		},
		Response: &admv1beta1.AdmissionResponse{Allowed: true},
	}
	return review, err
}
