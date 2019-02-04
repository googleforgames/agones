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

package gameserverallocations

import (
	"encoding/json"
	"fmt"
	"strconv"
	"sync"
	"testing"
	"time"

	"agones.dev/agones/pkg/apis/stable/v1alpha1"
	agtesting "agones.dev/agones/pkg/testing"
	"agones.dev/agones/pkg/util/webhooks"
	applypatch "github.com/evanphx/json-patch"
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

var (
	gvk = metav1.GroupVersionKind(v1alpha1.SchemeGroupVersion.WithKind("GameServerAllocation"))
)

func TestControllerCreationMutationHandler(t *testing.T) {
	t.Parallel()
	f, _, gsList := defaultFixtures(3)

	gsa := v1alpha1.GameServerAllocation{ObjectMeta: metav1.ObjectMeta{Name: "fa-1"},
		Spec: v1alpha1.GameServerAllocationSpec{
			Required: metav1.LabelSelector{MatchLabels: map[string]string{v1alpha1.FleetNameLabel: f.ObjectMeta.Name}},
		}}

	c, m := newFakeController()

	gsWatch := watch.NewFake()
	m.AgonesClient.AddWatchReactor("gameservers", k8stesting.DefaultWatchReactor(gsWatch, nil))
	m.AgonesClient.AddReactor("list", "gameservers", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, &v1alpha1.GameServerList{Items: gsList}, nil
	})
	m.AgonesClient.AddReactor("patch", "gameservers", func(action k8stesting.Action) (bool, runtime.Object, error) {
		gs := applyGameServerPatch(t, m, action)
		gsWatch.Modify(gs)
		return true, gs, nil
	})

	_, cancel := agtesting.StartInformers(m)
	defer cancel()

	test := func(gsa v1alpha1.GameServerAllocation, expectedState v1alpha1.GameServerAllocationState) {
		review, err := newAdmissionReview(gsa)
		assert.Nil(t, err)

		result, err := c.creationMutationHandler(review)
		assert.Nil(t, err)
		assert.True(t, result.Response.Allowed, fmt.Sprintf("%#v", result.Response))
		assert.Equal(t, admv1beta1.PatchTypeJSONPatch, *result.Response.PatchType)

		patch, err := applypatch.DecodePatch(result.Response.Patch)
		assert.Nil(t, err)

		JSON, err := patch.Apply(result.Request.Object.Raw)
		assert.Nil(t, err)
		newGSA := &v1alpha1.GameServerAllocation{}

		err = json.Unmarshal(JSON, newGSA)
		assert.Nil(t, err)

		assert.Equal(t, v1alpha1.Packed, newGSA.Spec.Scheduling)
		assert.Equal(t, expectedState, newGSA.Status.State)
		if expectedState == v1alpha1.GameServerAllocationAllocated {
			assert.NotEmpty(t, newGSA.ObjectMeta.OwnerReferences)
		} else {
			assert.Empty(t, newGSA.ObjectMeta.OwnerReferences)
		}
	}

	test(*gsa.DeepCopy(), v1alpha1.GameServerAllocationAllocated)
	test(*gsa.DeepCopy(), v1alpha1.GameServerAllocationAllocated)
	test(*gsa.DeepCopy(), v1alpha1.GameServerAllocationAllocated)
	test(*gsa.DeepCopy(), v1alpha1.GameServerAllocationUnAllocated)
}

func TestControllerMutationValidationHandler(t *testing.T) {
	t.Parallel()
	c, _ := newFakeController()

	gsa := v1alpha1.GameServerAllocation{ObjectMeta: metav1.ObjectMeta{Name: "fa-1", Namespace: defaultNs},
		Spec: v1alpha1.GameServerAllocationSpec{
			Required: metav1.LabelSelector{MatchLabels: map[string]string{v1alpha1.FleetNameLabel: "my-fleet-name"}},
		},
	}
	gsa.ApplyDefaults()

	t.Run("same fleetName", func(t *testing.T) {
		review, err := newAdmissionReview(gsa)
		assert.Nil(t, err)
		review.Request.OldObject = *review.Request.Object.DeepCopy()

		result, err := c.mutationValidationHandler(review)
		assert.Nil(t, err)
		assert.True(t, result.Response.Allowed)
	})

	t.Run("different fleetname", func(t *testing.T) {
		review, err := newAdmissionReview(gsa)
		assert.Nil(t, err)
		oldObject := gsa.DeepCopy()
		oldObject.Spec.Required.MatchLabels[v1alpha1.FleetNameLabel] = "changed"

		gsaJSON, err := json.Marshal(oldObject)
		assert.Nil(t, err)
		review.Request.OldObject = runtime.RawExtension{Raw: gsaJSON}

		result, err := c.mutationValidationHandler(review)
		assert.Nil(t, err)
		assert.False(t, result.Response.Allowed)
		assert.Equal(t, metav1.StatusReasonInvalid, result.Response.Result.Reason)
		assert.NotNil(t, result.Response.Result.Details)
	})
}

func TestControllerAllocate(t *testing.T) {
	t.Parallel()

	f, _, gsList := defaultFixtures(4)
	c, m := newFakeController()
	n := metav1.Now()
	l := map[string]string{"mode": "deathmatch"}
	a := map[string]string{"map": "searide"}
	fam := v1alpha1.MetaPatch{Labels: l, Annotations: a}

	gsList[3].ObjectMeta.DeletionTimestamp = &n

	m.AgonesClient.AddReactor("list", "gameservers", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, &v1alpha1.GameServerList{Items: gsList}, nil
	})

	updated := false
	gsWatch := watch.NewFake()
	m.AgonesClient.AddWatchReactor("gameservers", k8stesting.DefaultWatchReactor(gsWatch, nil))
	m.AgonesClient.AddReactor("patch", "gameservers", func(action k8stesting.Action) (bool, runtime.Object, error) {
		gs := applyGameServerPatch(t, m, action)

		updated = true
		assert.Equal(t, v1alpha1.GameServerStateAllocated, gs.Status.State)
		gsWatch.Modify(gs)

		return true, gs, nil
	})

	stop, cancel := agtesting.StartInformers(m, m.AgonesInformerFactory.Stable().V1alpha1().GameServers().Informer().HasSynced)
	defer cancel()

	err := c.counter.Run(stop)
	assert.Nil(t, err)

	gsa := v1alpha1.GameServerAllocation{ObjectMeta: metav1.ObjectMeta{Name: "gsa-1"},
		Spec: v1alpha1.GameServerAllocationSpec{
			Required:  metav1.LabelSelector{MatchLabels: map[string]string{v1alpha1.FleetNameLabel: f.ObjectMeta.Name}},
			MetaPatch: fam,
		}}
	gsa.ApplyDefaults()

	gs, err := c.allocate(&gsa)
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
	gs, err = c.allocate(&gsa)
	assert.Nil(t, err)
	assert.Equal(t, v1alpha1.GameServerStateAllocated, gs.Status.State)
	assert.True(t, updated)

	updated = false
	gs, err = c.allocate(&gsa)
	assert.Nil(t, err)
	assert.Equal(t, v1alpha1.GameServerStateAllocated, gs.Status.State)
	assert.True(t, updated)

	updated = false
	_, err = c.allocate(&gsa)
	assert.NotNil(t, err)
	assert.Equal(t, ErrNoGameServerReady, err)
	assert.False(t, updated)
}

func TestControllerAllocatePriority(t *testing.T) {
	t.Parallel()

	run := func(t *testing.T, name string, test func(t *testing.T, c *Controller, gas *v1alpha1.GameServerAllocation)) {
		f, _, gsList := defaultFixtures(4)
		c, m := newFakeController()

		gsList[0].Status.NodeName = n1
		gsList[1].Status.NodeName = n2
		gsList[2].Status.NodeName = n1
		gsList[3].Status.NodeName = n1

		m.AgonesClient.AddReactor("list", "gameservers", func(action k8stesting.Action) (bool, runtime.Object, error) {
			return true, &v1alpha1.GameServerList{Items: gsList}, nil
		})

		gsWatch := watch.NewFake()
		m.AgonesClient.AddWatchReactor("gameservers", k8stesting.DefaultWatchReactor(gsWatch, nil))
		m.AgonesClient.AddReactor("patch", "gameservers", func(action k8stesting.Action) (bool, runtime.Object, error) {
			gs := applyGameServerPatch(t, m, action)
			gsWatch.Modify(gs)
			return true, gs, nil
		})

		stop, cancel := agtesting.StartInformers(m)
		defer cancel()

		err := c.counter.Run(stop)
		assert.Nil(t, err)

		gas := &v1alpha1.GameServerAllocation{ObjectMeta: metav1.ObjectMeta{Name: "fa-1"},
			Spec: v1alpha1.GameServerAllocationSpec{
				Required: metav1.LabelSelector{MatchLabels: map[string]string{v1alpha1.FleetNameLabel: f.ObjectMeta.Name}},
			}}
		gas.ApplyDefaults()

		t.Run(name, func(t *testing.T) {
			test(t, c, gas.DeepCopy())
		})
	}

	run(t, "packed", func(t *testing.T, c *Controller, gas *v1alpha1.GameServerAllocation) {
		// priority should be node1, then node2
		gs, err := c.allocate(gas)
		assert.Nil(t, err)
		assert.Equal(t, n1, gs.Status.NodeName)

		gs, err = c.allocate(gas)
		assert.Nil(t, err)
		assert.Equal(t, n1, gs.Status.NodeName)

		gs, err = c.allocate(gas)
		assert.Nil(t, err)
		assert.Equal(t, n1, gs.Status.NodeName)

		gs, err = c.allocate(gas)
		assert.Nil(t, err)
		assert.Equal(t, n2, gs.Status.NodeName)

		// should have none left
		_, err = c.allocate(gas)
		assert.NotNil(t, err)
	})

	run(t, "distributed", func(t *testing.T, c *Controller, gas *v1alpha1.GameServerAllocation) {
		// make a copy, to avoid the race check
		gas = gas.DeepCopy()
		gas.Spec.Scheduling = v1alpha1.Distributed
		// should go node2, then node1
		gs, err := c.allocate(gas)
		assert.Nil(t, err)
		assert.Equal(t, n2, gs.Status.NodeName)

		gs, err = c.allocate(gas)
		assert.Nil(t, err)
		assert.Equal(t, n1, gs.Status.NodeName)

		gs, err = c.allocate(gas)
		assert.Nil(t, err)
		assert.Equal(t, n1, gs.Status.NodeName)

		gs, err = c.allocate(gas)
		assert.Nil(t, err)
		assert.Equal(t, n1, gs.Status.NodeName)

		// should have none left
		_, err = c.allocate(gas)
		assert.NotNil(t, err)
	})
}

func TestControllerAllocateMutex(t *testing.T) {
	t.Parallel()

	f, _, gsList := defaultFixtures(100)
	c, m := newFakeController()

	m.AgonesClient.AddReactor("list", "gameservers", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, &v1alpha1.GameServerList{Items: gsList}, nil
	})

	gas := &v1alpha1.GameServerAllocation{ObjectMeta: metav1.ObjectMeta{Name: "fa-1"},
		Spec: v1alpha1.GameServerAllocationSpec{
			Required: metav1.LabelSelector{MatchLabels: map[string]string{v1alpha1.FleetNameLabel: f.ObjectMeta.Name}},
		}}
	gas.ApplyDefaults()

	hasSync := m.AgonesInformerFactory.Stable().V1alpha1().GameServers().Informer().HasSynced
	stop, cancel := agtesting.StartInformers(m, hasSync)
	defer cancel()

	err := c.counter.Run(stop)
	assert.Nil(t, err)

	wg := sync.WaitGroup{}
	// start 10 threads, each one gets 10 allocations
	allocate := func() {
		defer wg.Done()
		for i := 1; i <= 10; i++ {
			_, err := c.allocate(gas.DeepCopy())
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

func TestControllerFindPackedReadyGameServer(t *testing.T) {
	t.Parallel()

	labels := map[string]string{"role": "gameserver"}
	gsa := &v1alpha1.GameServerAllocation{
		Spec: v1alpha1.GameServerAllocationSpec{
			Required: metav1.LabelSelector{
				MatchLabels: labels,
			},
			Scheduling: "",
		},
	}

	t.Run("test just required", func(t *testing.T) {
		c, m := newFakeController()
		watch := watch.NewFake()
		m.AgonesClient.AddWatchReactor("gameservers", k8stesting.DefaultWatchReactor(watch, nil))
		hasSync := m.AgonesInformerFactory.Stable().V1alpha1().GameServers().Informer().HasSynced

		n := metav1.Now()
		gsList := []v1alpha1.GameServer{
			{ObjectMeta: metav1.ObjectMeta{Name: "gs6", Namespace: defaultNs, Labels: labels, DeletionTimestamp: &n}, Status: v1alpha1.GameServerStatus{NodeName: "node1", State: v1alpha1.GameServerStateReady}},
			{ObjectMeta: metav1.ObjectMeta{Name: "gs1", Namespace: defaultNs, Labels: labels}, Status: v1alpha1.GameServerStatus{NodeName: "node1", State: v1alpha1.GameServerStateReady}},
			{ObjectMeta: metav1.ObjectMeta{Name: "gs2", Namespace: defaultNs, Labels: labels}, Status: v1alpha1.GameServerStatus{NodeName: "node2", State: v1alpha1.GameServerStateReady}},
			{ObjectMeta: metav1.ObjectMeta{Name: "gs3", Namespace: defaultNs, Labels: labels}, Status: v1alpha1.GameServerStatus{NodeName: "node1", State: v1alpha1.GameServerStateAllocated}},
			{ObjectMeta: metav1.ObjectMeta{Name: "gs4", Namespace: defaultNs, Labels: labels}, Status: v1alpha1.GameServerStatus{NodeName: "node1", State: v1alpha1.GameServerStateAllocated}},
			{ObjectMeta: metav1.ObjectMeta{Name: "gs5", Namespace: defaultNs, Labels: labels}, Status: v1alpha1.GameServerStatus{NodeName: "node1", State: v1alpha1.GameServerStateError}},
		}

		m.AgonesClient.AddReactor("list", "gameservers", func(action k8stesting.Action) (bool, runtime.Object, error) {
			return true, &v1alpha1.GameServerList{Items: gsList}, nil
		})

		stop, cancel := agtesting.StartInformers(m, hasSync)
		defer cancel()

		err := c.counter.Run(stop)
		assert.Nil(t, err)

		gs, err := c.findReadyGameServerForAllocation(gsa, packedComparator)
		assert.Nil(t, err)
		assert.Equal(t, "node1", gs.Status.NodeName)
		assert.Equal(t, v1alpha1.GameServerStateReady, gs.Status.State)

		// mock that the first game server is allocated
		change := gsList[1].DeepCopy()
		change.Status.State = v1alpha1.GameServerStateAllocated
		watch.Modify(change)
		assert.True(t, cache.WaitForCacheSync(stop, hasSync))

		gs, err = c.findReadyGameServerForAllocation(gsa, packedComparator)
		assert.Nil(t, err)
		assert.Equal(t, "node2", gs.Status.NodeName)
		assert.Equal(t, v1alpha1.GameServerStateReady, gs.Status.State)

		gsList[2].Status.State = v1alpha1.GameServerStateAllocated
		change = gsList[2].DeepCopy()
		change.Status.State = v1alpha1.GameServerStateAllocated
		watch.Modify(change)
		assert.True(t, cache.WaitForCacheSync(stop, hasSync))

		gs, err = c.findReadyGameServerForAllocation(gsa, packedComparator)
		assert.Equal(t, ErrNoGameServerReady, err)
		assert.Nil(t, gs)
	})

	t.Run("test preferred", func(t *testing.T) {
		c, m := newFakeController()
		watch := watch.NewFake()
		m.AgonesClient.AddWatchReactor("gameservers", k8stesting.DefaultWatchReactor(watch, nil))
		hasSync := m.AgonesInformerFactory.Stable().V1alpha1().GameServers().Informer().HasSynced

		prefLabels := map[string]string{"role": "gameserver", "preferred": "true"}
		gsList := []v1alpha1.GameServer{
			{ObjectMeta: metav1.ObjectMeta{Name: "gs1", Namespace: defaultNs, Labels: prefLabels}, Status: v1alpha1.GameServerStatus{NodeName: "node1", State: v1alpha1.GameServerStateReady}},
			{ObjectMeta: metav1.ObjectMeta{Name: "gs2", Namespace: defaultNs, Labels: labels}, Status: v1alpha1.GameServerStatus{NodeName: "node2", State: v1alpha1.GameServerStateReady}},
			{ObjectMeta: metav1.ObjectMeta{Name: "gs3", Namespace: defaultNs, Labels: labels}, Status: v1alpha1.GameServerStatus{NodeName: "node1", State: v1alpha1.GameServerStateReady}},
			{ObjectMeta: metav1.ObjectMeta{Name: "gs4", Namespace: defaultNs, Labels: prefLabels}, Status: v1alpha1.GameServerStatus{NodeName: "node2", State: v1alpha1.GameServerStateReady}},
			{ObjectMeta: metav1.ObjectMeta{Name: "gs5", Namespace: defaultNs, Labels: labels}, Status: v1alpha1.GameServerStatus{NodeName: "node1", State: v1alpha1.GameServerStateReady}},
			{ObjectMeta: metav1.ObjectMeta{Name: "gs6", Namespace: defaultNs, Labels: labels}, Status: v1alpha1.GameServerStatus{NodeName: "node1", State: v1alpha1.GameServerStateReady}},
		}

		m.AgonesClient.AddReactor("list", "gameservers", func(action k8stesting.Action) (bool, runtime.Object, error) {
			return true, &v1alpha1.GameServerList{Items: gsList}, nil
		})

		prefGsa := gsa.DeepCopy()
		prefGsa.Spec.Preferred = append(prefGsa.Spec.Preferred, metav1.LabelSelector{
			MatchLabels: map[string]string{"preferred": "true"},
		})

		stop, cancel := agtesting.StartInformers(m, hasSync)
		defer cancel()

		err := c.counter.Run(stop)
		assert.Nil(t, err)

		gs, err := c.findReadyGameServerForAllocation(prefGsa, packedComparator)
		assert.Nil(t, err)
		assert.Equal(t, "node1", gs.Status.NodeName)
		assert.Equal(t, "gs1", gs.ObjectMeta.Name)
		assert.Equal(t, v1alpha1.GameServerStateReady, gs.Status.State)

		change := gsList[0].DeepCopy()
		change.Status.State = v1alpha1.GameServerStateAllocated
		watch.Modify(change)
		assert.True(t, cache.WaitForCacheSync(stop, hasSync))

		gs, err = c.findReadyGameServerForAllocation(prefGsa, packedComparator)
		assert.Nil(t, err)
		assert.Equal(t, "node2", gs.Status.NodeName)
		assert.Equal(t, "gs4", gs.ObjectMeta.Name)
		assert.Equal(t, v1alpha1.GameServerStateReady, gs.Status.State)

		change = gsList[3].DeepCopy()
		change.Status.State = v1alpha1.GameServerStateAllocated
		watch.Modify(change)
		assert.True(t, cache.WaitForCacheSync(stop, hasSync))

		gs, err = c.findReadyGameServerForAllocation(prefGsa, packedComparator)
		assert.Nil(t, err)
		assert.Equal(t, "node1", gs.Status.NodeName)
		assert.Equal(t, v1alpha1.GameServerStateReady, gs.Status.State)
	})

	t.Run("allocation trap", func(t *testing.T) {
		c, m := newFakeController()
		hasSync := m.AgonesInformerFactory.Stable().V1alpha1().GameServers().Informer().HasSynced

		gsList := []v1alpha1.GameServer{
			{ObjectMeta: metav1.ObjectMeta{Name: "gs1", Labels: labels},
				Status: v1alpha1.GameServerStatus{NodeName: "node1", State: v1alpha1.GameServerStateAllocated}},
			{ObjectMeta: metav1.ObjectMeta{Name: "gs2", Labels: labels},
				Status: v1alpha1.GameServerStatus{NodeName: "node1", State: v1alpha1.GameServerStateAllocated}},
			{ObjectMeta: metav1.ObjectMeta{Name: "gs3", Labels: labels},
				Status: v1alpha1.GameServerStatus{NodeName: "node1", State: v1alpha1.GameServerStateAllocated}},
			{ObjectMeta: metav1.ObjectMeta{Name: "gs4", Labels: labels},
				Status: v1alpha1.GameServerStatus{NodeName: "node1", State: v1alpha1.GameServerStateAllocated}},
			{ObjectMeta: metav1.ObjectMeta{Name: "gs5", Labels: labels},
				Status: v1alpha1.GameServerStatus{NodeName: "node2", State: v1alpha1.GameServerStateReady}},
			{ObjectMeta: metav1.ObjectMeta{Name: "gs6", Labels: labels},
				Status: v1alpha1.GameServerStatus{NodeName: "node2", State: v1alpha1.GameServerStateReady}},
			{ObjectMeta: metav1.ObjectMeta{Name: "gs7", Labels: labels},
				Status: v1alpha1.GameServerStatus{NodeName: "node2", State: v1alpha1.GameServerStateReady}},
			{ObjectMeta: metav1.ObjectMeta{Name: "gs8", Labels: labels},
				Status: v1alpha1.GameServerStatus{NodeName: "node2", State: v1alpha1.GameServerStateReady}},
		}

		m.AgonesClient.AddReactor("list", "gameservers", func(action k8stesting.Action) (bool, runtime.Object, error) {
			return true, &v1alpha1.GameServerList{Items: gsList}, nil
		})

		stop, cancel := agtesting.StartInformers(m, hasSync)
		defer cancel()

		err := c.counter.Run(stop)
		assert.Nil(t, err)

		gs, err := c.findReadyGameServerForAllocation(gsa, packedComparator)
		assert.Nil(t, err)
		assert.Equal(t, "node2", gs.Status.NodeName)
		assert.Equal(t, v1alpha1.GameServerStateReady, gs.Status.State)
	})
}

func TestControllerFindDistributedReadyGameServer(t *testing.T) {
	t.Parallel()

	c, m := newFakeController()
	watch := watch.NewFake()
	m.AgonesClient.AddWatchReactor("gameservers", k8stesting.DefaultWatchReactor(watch, nil))
	hasSync := m.AgonesInformerFactory.Stable().V1alpha1().GameServers().Informer().HasSynced

	labels := map[string]string{"role": "gameserver"}

	gsa := &v1alpha1.GameServerAllocation{
		Spec: v1alpha1.GameServerAllocationSpec{
			Required: metav1.LabelSelector{
				MatchLabels: labels,
			},
			Scheduling: "",
		},
	}

	gsList := []v1alpha1.GameServer{
		{ObjectMeta: metav1.ObjectMeta{Name: "gs1", Namespace: defaultNs, Labels: labels},
			Status: v1alpha1.GameServerStatus{NodeName: "node1", State: v1alpha1.GameServerStateReady}},
		{ObjectMeta: metav1.ObjectMeta{Name: "gs2", Namespace: defaultNs, Labels: labels},
			Status: v1alpha1.GameServerStatus{NodeName: "node1", State: v1alpha1.GameServerStateReady}},
		{ObjectMeta: metav1.ObjectMeta{Name: "gs3", Namespace: defaultNs, Labels: labels},
			Status: v1alpha1.GameServerStatus{NodeName: "node1", State: v1alpha1.GameServerStateReady}},
		{ObjectMeta: metav1.ObjectMeta{Name: "gs4", Namespace: defaultNs, Labels: labels},
			Status: v1alpha1.GameServerStatus{NodeName: "node1", State: v1alpha1.GameServerStateError}},
		{ObjectMeta: metav1.ObjectMeta{Name: "gs5", Namespace: defaultNs, Labels: labels},
			Status: v1alpha1.GameServerStatus{NodeName: "node2", State: v1alpha1.GameServerStateReady}},
		{ObjectMeta: metav1.ObjectMeta{Name: "gs6", Namespace: defaultNs, Labels: labels},
			Status: v1alpha1.GameServerStatus{NodeName: "node2", State: v1alpha1.GameServerStateReady}},
		{ObjectMeta: metav1.ObjectMeta{Name: "gs7", Namespace: defaultNs, Labels: labels},
			Status: v1alpha1.GameServerStatus{NodeName: "node3", State: v1alpha1.GameServerStateReady}},
	}

	m.AgonesClient.AddReactor("list", "gameservers", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, &v1alpha1.GameServerList{Items: gsList}, nil
	})

	stop, cancel := agtesting.StartInformers(m, hasSync)
	defer cancel()

	err := c.counter.Run(stop)
	assert.Nil(t, err)

	gs, err := c.findReadyGameServerForAllocation(gsa, distributedComparator)
	assert.Nil(t, err)
	assert.Equal(t, "node3", gs.Status.NodeName)
	assert.Equal(t, v1alpha1.GameServerStateReady, gs.Status.State)

	change := gsList[6].DeepCopy()
	change.Status.State = v1alpha1.GameServerStateAllocated
	watch.Modify(change)
	assert.True(t, cache.WaitForCacheSync(stop, hasSync))

	gs, err = c.findReadyGameServerForAllocation(gsa, distributedComparator)
	assert.Nil(t, err)
	assert.Equal(t, "node2", gs.Status.NodeName)
	assert.Equal(t, v1alpha1.GameServerStateReady, gs.Status.State)

	change = gsList[4].DeepCopy()
	change.Status.State = v1alpha1.GameServerStateAllocated
	watch.Modify(change)
	assert.True(t, cache.WaitForCacheSync(stop, hasSync))

	gs, err = c.findReadyGameServerForAllocation(gsa, distributedComparator)
	assert.Nil(t, err)
	assert.Equal(t, "node1", gs.Status.NodeName)
	assert.Equal(t, v1alpha1.GameServerStateReady, gs.Status.State)

	change = gsList[0].DeepCopy()
	change.Status.State = v1alpha1.GameServerStateAllocated
	watch.Modify(change)
	assert.True(t, cache.WaitForCacheSync(stop, hasSync))

	gs, err = c.findReadyGameServerForAllocation(gsa, distributedComparator)
	assert.Nil(t, err)
	assert.Equal(t, "node2", gs.Status.NodeName)
	assert.Equal(t, v1alpha1.GameServerStateReady, gs.Status.State)

	change = gsList[5].DeepCopy()
	change.Status.State = v1alpha1.GameServerStateAllocated
	watch.Modify(change)
	assert.True(t, cache.WaitForCacheSync(stop, hasSync))

	gs, err = c.findReadyGameServerForAllocation(gsa, distributedComparator)
	assert.Nil(t, err)
	assert.Equal(t, "node1", gs.Status.NodeName)
	assert.Equal(t, v1alpha1.GameServerStateReady, gs.Status.State)

	change = gsList[1].DeepCopy()
	change.Status.State = v1alpha1.GameServerStateAllocated
	watch.Modify(change)
	assert.True(t, cache.WaitForCacheSync(stop, hasSync))

	gs, err = c.findReadyGameServerForAllocation(gsa, distributedComparator)
	assert.Nil(t, err)
	assert.Equal(t, "node1", gs.Status.NodeName)
	assert.Equal(t, v1alpha1.GameServerStateReady, gs.Status.State)

	change = gsList[2].DeepCopy()
	change.Status.State = v1alpha1.GameServerStateAllocated
	watch.Modify(change)
	assert.True(t, cache.WaitForCacheSync(stop, hasSync))

	gs, err = c.findReadyGameServerForAllocation(gsa, distributedComparator)
	assert.Equal(t, ErrNoGameServerReady, err)
	assert.Nil(t, gs)
}

func TestControllerRunSync(t *testing.T) {
	c, m := newFakeController()
	watch := watch.NewFake()

	done := make(chan struct{}, 10)

	m.ExtClient.AddReactor("get", "customresourcedefinitions", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, agtesting.NewEstablishedCRD(), nil
	})
	m.AgonesClient.AddWatchReactor("gameserverallocations", k8stesting.DefaultWatchReactor(watch, nil))
	m.AgonesClient.AddReactor("delete", "gameserverallocations", func(action k8stesting.Action) (bool, runtime.Object, error) {
		da := action.(k8stesting.DeleteAction)
		assert.Equal(t, "default", da.GetNamespace())
		assert.Equal(t, "allocation", da.GetName())

		done <- struct{}{}

		return true, nil, nil
	})

	stop, cancel := agtesting.StartInformers(m)
	defer cancel()

	go func() {
		err := c.Run(1, stop)
		assert.Nil(t, err)
	}()

	gsa := &v1alpha1.GameServerAllocation{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: defaultNs,
			Name:      "allocation",
		},
		Status: v1alpha1.GameServerAllocationStatus{
			State: v1alpha1.GameServerAllocationUnAllocated,
		},
	}

	logrus.Info("adding game server allocation")
	watch.Add(gsa.DeepCopy())

	select {
	case <-done:
	case <-time.After(10 * time.Second):
		assert.Fail(t, "should have got delete operation")
	}

	gsa.ObjectMeta.Name = "default2"
	gsa.Status.State = v1alpha1.GameServerAllocationAllocated
	watch.Add(gsa.DeepCopy())

	select {
	case <-done:
		assert.Fail(t, "should not have got a delete operation")
	case <-time.After(3 * time.Second):
	}
}

func TestControllerSyncDelete(t *testing.T) {
	c, m := newFakeController()

	deleted := false
	m.AgonesClient.AddReactor("delete", "gameserverallocations", func(action k8stesting.Action) (bool, runtime.Object, error) {
		da := action.(k8stesting.DeleteAction)
		assert.Equal(t, "default", da.GetNamespace())
		assert.Equal(t, "allocation", da.GetName())
		deleted = true

		return true, nil, nil
	})

	err := c.syncDelete("default/allocation")
	assert.Nil(t, err)
	assert.True(t, deleted)
}

func defaultFixtures(gsLen int) (*v1alpha1.Fleet, *v1alpha1.GameServerSet, []v1alpha1.GameServer) {
	f := &v1alpha1.Fleet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "fleet-1",
			Namespace: defaultNs,
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
	c := NewController(wh, healthcheck.NewHandler(), &sync.Mutex{}, m.KubeClient, m.KubeInformerFactory, m.ExtClient, m.AgonesClient, m.AgonesInformerFactory)
	c.recorder = m.FakeRecorder
	return c, m
}

func newAdmissionReview(gsa v1alpha1.GameServerAllocation) (admv1beta1.AdmissionReview, error) {
	raw, err := json.Marshal(gsa)
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
			Namespace: defaultNs,
		},
		Response: &admv1beta1.AdmissionResponse{Allowed: true},
	}
	return review, err
}

func applyGameServerPatch(t *testing.T, m agtesting.Mocks, action k8stesting.Action) *v1alpha1.GameServer {
	pa := action.(k8stesting.PatchAction)
	gs, err := m.AgonesInformerFactory.Stable().V1alpha1().GameServers().Lister().GameServers(pa.GetNamespace()).Get(pa.GetName())
	assert.Nil(t, err)
	js, err := json.Marshal(gs)
	assert.Nil(t, err)
	patch, err := applypatch.DecodePatch(pa.GetPatch())
	assert.Nil(t, err)
	newJS, err := patch.Apply(js)
	assert.Nil(t, err)
	// reset it
	gs = &v1alpha1.GameServer{}
	err = json.Unmarshal(newJS, gs)
	assert.Nil(t, err)
	return gs
}
