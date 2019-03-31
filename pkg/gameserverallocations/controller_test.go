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

package gameserverallocations

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"agones.dev/agones/pkg/apis"
	"agones.dev/agones/pkg/apis/allocation/v1alpha1"
	stablev1alpha1 "agones.dev/agones/pkg/apis/stable/v1alpha1"
	"agones.dev/agones/pkg/gameservers"
	agtesting "agones.dev/agones/pkg/testing"
	"agones.dev/agones/pkg/util/apiserver"
	"github.com/heptiolabs/healthcheck"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"
	k8stesting "k8s.io/client-go/testing"
	"k8s.io/client-go/tools/cache"
)

const (
	defaultNs = "default"
	n1        = "node1"
	n2        = "node2"
)

func TestControllerAllocationHandler(t *testing.T) {
	t.Parallel()

	t.Run("successful allocation", func(t *testing.T) {
		f, _, gsList := defaultFixtures(3)

		gsa := &v1alpha1.GameServerAllocation{
			Spec: v1alpha1.GameServerAllocationSpec{
				Required: metav1.LabelSelector{MatchLabels: map[string]string{stablev1alpha1.FleetNameLabel: f.ObjectMeta.Name}},
			}}

		c, m := newFakeController()
		gsWatch := watch.NewFake()
		m.AgonesClient.AddWatchReactor("gameservers", k8stesting.DefaultWatchReactor(gsWatch, nil))
		m.AgonesClient.AddReactor("list", "gameservers", func(action k8stesting.Action) (bool, k8sruntime.Object, error) {
			return true, &stablev1alpha1.GameServerList{Items: gsList}, nil
		})
		m.AgonesClient.AddReactor("update", "gameservers", func(action k8stesting.Action) (bool, k8sruntime.Object, error) {
			ua := action.(k8stesting.UpdateAction)
			gs := ua.GetObject().(*stablev1alpha1.GameServer)
			gsWatch.Modify(gs)
			return true, gs, nil
		})

		stop, cancel := agtesting.StartInformers(m)
		defer cancel()

		// This call initializes the cache
		err := c.syncReadyGSServerCache()
		assert.Nil(t, err)

		err = c.counter.Run(0, stop)
		assert.Nil(t, err)

		test := func(gsa *v1alpha1.GameServerAllocation, expectedState v1alpha1.GameServerAllocationState) {
			buf := bytes.NewBuffer(nil)
			err := json.NewEncoder(buf).Encode(gsa)
			assert.NoError(t, err)
			r, err := http.NewRequest(http.MethodPost, "/", buf)
			r.Header.Set("Content-Type", k8sruntime.ContentTypeJSON)
			assert.NoError(t, err)
			rec := httptest.NewRecorder()
			err = c.allocationHandler(rec, r, "default")
			assert.NoError(t, err)
			ret := &v1alpha1.GameServerAllocation{}
			err = json.Unmarshal(rec.Body.Bytes(), ret)
			assert.NoError(t, err)

			assert.Equal(t, gsa.Spec.Required, ret.Spec.Required)
			assert.True(t, expectedState == ret.Status.State, "Failed: %s vs %s", expectedState, ret.Status.State)
		}

		test(gsa.DeepCopy(), v1alpha1.GameServerAllocationAllocated)
		test(gsa.DeepCopy(), v1alpha1.GameServerAllocationAllocated)
		test(gsa.DeepCopy(), v1alpha1.GameServerAllocationAllocated)
		test(gsa.DeepCopy(), v1alpha1.GameServerAllocationUnAllocated)
	})

	t.Run("method not allowed", func(t *testing.T) {
		c, _ := newFakeController()
		r, err := http.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		assert.NoError(t, err)

		err = c.allocationHandler(rec, r, "default")
		assert.NoError(t, err)

		assert.Equal(t, http.StatusMethodNotAllowed, rec.Code)
	})

	t.Run("invalid gameserverallocation", func(t *testing.T) {
		c, _ := newFakeController()
		gsa := &v1alpha1.GameServerAllocation{
			Spec: v1alpha1.GameServerAllocationSpec{
				Scheduling: "wrong",
			}}
		buf := bytes.NewBuffer(nil)
		err := json.NewEncoder(buf).Encode(gsa)
		assert.NoError(t, err)
		r, err := http.NewRequest(http.MethodPost, "/", buf)
		r.Header.Set("Content-Type", k8sruntime.ContentTypeJSON)
		assert.NoError(t, err)
		rec := httptest.NewRecorder()
		err = c.allocationHandler(rec, r, "default")
		assert.NoError(t, err)

		assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)

		s := &metav1.Status{}
		err = json.NewDecoder(rec.Body).Decode(s)
		assert.NoError(t, err)

		assert.Equal(t, metav1.StatusReasonInvalid, s.Reason)
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

	m.AgonesClient.AddReactor("list", "gameservers", func(action k8stesting.Action) (bool, k8sruntime.Object, error) {
		return true, &stablev1alpha1.GameServerList{Items: gsList}, nil
	})

	updated := false
	gsWatch := watch.NewFake()
	m.AgonesClient.AddWatchReactor("gameservers", k8stesting.DefaultWatchReactor(gsWatch, nil))
	m.AgonesClient.AddReactor("update", "gameservers", func(action k8stesting.Action) (bool, k8sruntime.Object, error) {
		ua := action.(k8stesting.UpdateAction)
		gs := ua.GetObject().(*stablev1alpha1.GameServer)

		updated = true
		assert.Equal(t, stablev1alpha1.GameServerStateAllocated, gs.Status.State)
		gsWatch.Modify(gs)

		return true, gs, nil
	})

	stop, cancel := agtesting.StartInformers(m, m.AgonesInformerFactory.Stable().V1alpha1().GameServers().Informer().HasSynced)
	defer cancel()

	// This call initializes the cache
	err := c.syncReadyGSServerCache()
	assert.Nil(t, err)

	err = c.counter.Run(0, stop)
	assert.Nil(t, err)

	gsa := v1alpha1.GameServerAllocation{ObjectMeta: metav1.ObjectMeta{Name: "gsa-1"},
		Spec: v1alpha1.GameServerAllocationSpec{
			Required:  metav1.LabelSelector{MatchLabels: map[string]string{stablev1alpha1.FleetNameLabel: f.ObjectMeta.Name}},
			MetaPatch: fam,
		}}
	gsa.ApplyDefaults()

	gs, err := c.allocate(&gsa)
	assert.Nil(t, err)
	assert.Equal(t, stablev1alpha1.GameServerStateAllocated, gs.Status.State)
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
	assert.Equal(t, stablev1alpha1.GameServerStateAllocated, gs.Status.State)
	assert.True(t, updated)

	updated = false
	gs, err = c.allocate(&gsa)
	assert.Nil(t, err)
	assert.Equal(t, stablev1alpha1.GameServerStateAllocated, gs.Status.State)
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

		m.AgonesClient.AddReactor("list", "gameservers", func(action k8stesting.Action) (bool, k8sruntime.Object, error) {
			return true, &stablev1alpha1.GameServerList{Items: gsList}, nil
		})

		gsWatch := watch.NewFake()
		m.AgonesClient.AddWatchReactor("gameservers", k8stesting.DefaultWatchReactor(gsWatch, nil))
		m.AgonesClient.AddReactor("update", "gameservers", func(action k8stesting.Action) (bool, k8sruntime.Object, error) {
			ua := action.(k8stesting.UpdateAction)
			gs := ua.GetObject().(*stablev1alpha1.GameServer)
			gsWatch.Modify(gs)

			return true, gs, nil
		})

		stop, cancel := agtesting.StartInformers(m)
		defer cancel()

		// This call initializes the cache
		err := c.syncReadyGSServerCache()
		assert.Nil(t, err)

		err = c.counter.Run(0, stop)
		assert.Nil(t, err)

		gas := &v1alpha1.GameServerAllocation{ObjectMeta: metav1.ObjectMeta{Name: "fa-1"},
			Spec: v1alpha1.GameServerAllocationSpec{
				Required: metav1.LabelSelector{MatchLabels: map[string]string{stablev1alpha1.FleetNameLabel: f.ObjectMeta.Name}},
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
		gas.Spec.Scheduling = apis.Distributed
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
		gsList := []stablev1alpha1.GameServer{
			{ObjectMeta: metav1.ObjectMeta{Name: "gs6", Namespace: defaultNs, Labels: labels, DeletionTimestamp: &n}, Status: stablev1alpha1.GameServerStatus{NodeName: "node1", State: stablev1alpha1.GameServerStateReady}},
			{ObjectMeta: metav1.ObjectMeta{Name: "gs1", Namespace: defaultNs, Labels: labels}, Status: stablev1alpha1.GameServerStatus{NodeName: "node1", State: stablev1alpha1.GameServerStateReady}},
			{ObjectMeta: metav1.ObjectMeta{Name: "gs2", Namespace: defaultNs, Labels: labels}, Status: stablev1alpha1.GameServerStatus{NodeName: "node2", State: stablev1alpha1.GameServerStateReady}},
			{ObjectMeta: metav1.ObjectMeta{Name: "gs3", Namespace: defaultNs, Labels: labels}, Status: stablev1alpha1.GameServerStatus{NodeName: "node1", State: stablev1alpha1.GameServerStateAllocated}},
			{ObjectMeta: metav1.ObjectMeta{Name: "gs4", Namespace: defaultNs, Labels: labels}, Status: stablev1alpha1.GameServerStatus{NodeName: "node1", State: stablev1alpha1.GameServerStateAllocated}},
			{ObjectMeta: metav1.ObjectMeta{Name: "gs5", Namespace: defaultNs, Labels: labels}, Status: stablev1alpha1.GameServerStatus{NodeName: "node1", State: stablev1alpha1.GameServerStateError}},
		}

		m.AgonesClient.AddReactor("list", "gameservers", func(action k8stesting.Action) (bool, k8sruntime.Object, error) {
			return true, &stablev1alpha1.GameServerList{Items: gsList}, nil
		})

		stop, cancel := agtesting.StartInformers(m, hasSync)
		defer cancel()

		// This call initializes the cache
		err := c.syncReadyGSServerCache()
		assert.Nil(t, err)

		err = c.counter.Run(0, stop)
		assert.Nil(t, err)

		gs, err := c.findReadyGameServerForAllocation(gsa, packedComparator)
		assert.Nil(t, err)
		assert.Equal(t, "node1", gs.Status.NodeName)
		assert.Equal(t, stablev1alpha1.GameServerStateReady, gs.Status.State)

		// mock that the first game server is allocated
		change := gsList[1].DeepCopy()
		change.Status.State = stablev1alpha1.GameServerStateAllocated
		watch.Modify(change)
		assert.True(t, cache.WaitForCacheSync(stop, hasSync))

		gs, err = c.findReadyGameServerForAllocation(gsa, packedComparator)
		assert.Nil(t, err)
		assert.Equal(t, "node2", gs.Status.NodeName)
		assert.Equal(t, stablev1alpha1.GameServerStateReady, gs.Status.State)

		gsList[2].Status.State = stablev1alpha1.GameServerStateAllocated
		change = gsList[2].DeepCopy()
		change.Status.State = stablev1alpha1.GameServerStateAllocated
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
		gsList := []stablev1alpha1.GameServer{
			{ObjectMeta: metav1.ObjectMeta{Name: "gs1", Namespace: defaultNs, Labels: prefLabels}, Status: stablev1alpha1.GameServerStatus{NodeName: "node1", State: stablev1alpha1.GameServerStateReady}},
			{ObjectMeta: metav1.ObjectMeta{Name: "gs2", Namespace: defaultNs, Labels: labels}, Status: stablev1alpha1.GameServerStatus{NodeName: "node2", State: stablev1alpha1.GameServerStateReady}},
			{ObjectMeta: metav1.ObjectMeta{Name: "gs3", Namespace: defaultNs, Labels: labels}, Status: stablev1alpha1.GameServerStatus{NodeName: "node1", State: stablev1alpha1.GameServerStateReady}},
			{ObjectMeta: metav1.ObjectMeta{Name: "gs4", Namespace: defaultNs, Labels: prefLabels}, Status: stablev1alpha1.GameServerStatus{NodeName: "node2", State: stablev1alpha1.GameServerStateReady}},
			{ObjectMeta: metav1.ObjectMeta{Name: "gs5", Namespace: defaultNs, Labels: labels}, Status: stablev1alpha1.GameServerStatus{NodeName: "node1", State: stablev1alpha1.GameServerStateReady}},
			{ObjectMeta: metav1.ObjectMeta{Name: "gs6", Namespace: defaultNs, Labels: labels}, Status: stablev1alpha1.GameServerStatus{NodeName: "node1", State: stablev1alpha1.GameServerStateReady}},
		}

		m.AgonesClient.AddReactor("list", "gameservers", func(action k8stesting.Action) (bool, k8sruntime.Object, error) {
			return true, &stablev1alpha1.GameServerList{Items: gsList}, nil
		})

		prefGsa := gsa.DeepCopy()
		prefGsa.Spec.Preferred = append(prefGsa.Spec.Preferred, metav1.LabelSelector{
			MatchLabels: map[string]string{"preferred": "true"},
		})

		stop, cancel := agtesting.StartInformers(m, hasSync)
		defer cancel()

		// This call initializes the cache
		err := c.syncReadyGSServerCache()
		assert.Nil(t, err)

		err = c.counter.Run(0, stop)
		assert.Nil(t, err)

		gs, err := c.findReadyGameServerForAllocation(prefGsa, packedComparator)
		assert.Nil(t, err)
		assert.Equal(t, "node1", gs.Status.NodeName)
		assert.Equal(t, "gs1", gs.ObjectMeta.Name)
		assert.Equal(t, stablev1alpha1.GameServerStateReady, gs.Status.State)

		change := gsList[0].DeepCopy()
		change.Status.State = stablev1alpha1.GameServerStateAllocated
		watch.Modify(change)
		assert.True(t, cache.WaitForCacheSync(stop, hasSync))

		gs, err = c.findReadyGameServerForAllocation(prefGsa, packedComparator)
		assert.Nil(t, err)
		assert.Equal(t, "node2", gs.Status.NodeName)
		assert.Equal(t, "gs4", gs.ObjectMeta.Name)
		assert.Equal(t, stablev1alpha1.GameServerStateReady, gs.Status.State)

		change = gsList[3].DeepCopy()
		change.Status.State = stablev1alpha1.GameServerStateAllocated
		watch.Modify(change)
		assert.True(t, cache.WaitForCacheSync(stop, hasSync))

		gs, err = c.findReadyGameServerForAllocation(prefGsa, packedComparator)
		assert.Nil(t, err)
		assert.Equal(t, "node1", gs.Status.NodeName)
		assert.Equal(t, stablev1alpha1.GameServerStateReady, gs.Status.State)
	})

	t.Run("allocation trap", func(t *testing.T) {
		c, m := newFakeController()
		hasSync := m.AgonesInformerFactory.Stable().V1alpha1().GameServers().Informer().HasSynced

		gsList := []stablev1alpha1.GameServer{
			{ObjectMeta: metav1.ObjectMeta{Name: "gs1", Labels: labels},
				Status: stablev1alpha1.GameServerStatus{NodeName: "node1", State: stablev1alpha1.GameServerStateAllocated}},
			{ObjectMeta: metav1.ObjectMeta{Name: "gs2", Labels: labels},
				Status: stablev1alpha1.GameServerStatus{NodeName: "node1", State: stablev1alpha1.GameServerStateAllocated}},
			{ObjectMeta: metav1.ObjectMeta{Name: "gs3", Labels: labels},
				Status: stablev1alpha1.GameServerStatus{NodeName: "node1", State: stablev1alpha1.GameServerStateAllocated}},
			{ObjectMeta: metav1.ObjectMeta{Name: "gs4", Labels: labels},
				Status: stablev1alpha1.GameServerStatus{NodeName: "node1", State: stablev1alpha1.GameServerStateAllocated}},
			{ObjectMeta: metav1.ObjectMeta{Name: "gs5", Labels: labels},
				Status: stablev1alpha1.GameServerStatus{NodeName: "node2", State: stablev1alpha1.GameServerStateReady}},
			{ObjectMeta: metav1.ObjectMeta{Name: "gs6", Labels: labels},
				Status: stablev1alpha1.GameServerStatus{NodeName: "node2", State: stablev1alpha1.GameServerStateReady}},
			{ObjectMeta: metav1.ObjectMeta{Name: "gs7", Labels: labels},
				Status: stablev1alpha1.GameServerStatus{NodeName: "node2", State: stablev1alpha1.GameServerStateReady}},
			{ObjectMeta: metav1.ObjectMeta{Name: "gs8", Labels: labels},
				Status: stablev1alpha1.GameServerStatus{NodeName: "node2", State: stablev1alpha1.GameServerStateReady}},
		}

		m.AgonesClient.AddReactor("list", "gameservers", func(action k8stesting.Action) (bool, k8sruntime.Object, error) {
			return true, &stablev1alpha1.GameServerList{Items: gsList}, nil
		})

		stop, cancel := agtesting.StartInformers(m, hasSync)
		defer cancel()

		// This call initializes the cache
		err := c.syncReadyGSServerCache()
		assert.Nil(t, err)

		err = c.counter.Run(0, stop)
		assert.Nil(t, err)

		gs, err := c.findReadyGameServerForAllocation(gsa, packedComparator)
		assert.Nil(t, err)
		assert.Equal(t, "node2", gs.Status.NodeName)
		assert.Equal(t, stablev1alpha1.GameServerStateReady, gs.Status.State)
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

	gsList := []stablev1alpha1.GameServer{
		{ObjectMeta: metav1.ObjectMeta{Name: "gs1", Namespace: defaultNs, Labels: labels},
			Status: stablev1alpha1.GameServerStatus{NodeName: "node1", State: stablev1alpha1.GameServerStateReady}},
		{ObjectMeta: metav1.ObjectMeta{Name: "gs2", Namespace: defaultNs, Labels: labels},
			Status: stablev1alpha1.GameServerStatus{NodeName: "node1", State: stablev1alpha1.GameServerStateReady}},
		{ObjectMeta: metav1.ObjectMeta{Name: "gs3", Namespace: defaultNs, Labels: labels},
			Status: stablev1alpha1.GameServerStatus{NodeName: "node1", State: stablev1alpha1.GameServerStateReady}},
		{ObjectMeta: metav1.ObjectMeta{Name: "gs4", Namespace: defaultNs, Labels: labels},
			Status: stablev1alpha1.GameServerStatus{NodeName: "node1", State: stablev1alpha1.GameServerStateError}},
		{ObjectMeta: metav1.ObjectMeta{Name: "gs5", Namespace: defaultNs, Labels: labels},
			Status: stablev1alpha1.GameServerStatus{NodeName: "node2", State: stablev1alpha1.GameServerStateReady}},
		{ObjectMeta: metav1.ObjectMeta{Name: "gs6", Namespace: defaultNs, Labels: labels},
			Status: stablev1alpha1.GameServerStatus{NodeName: "node2", State: stablev1alpha1.GameServerStateReady}},
		{ObjectMeta: metav1.ObjectMeta{Name: "gs7", Namespace: defaultNs, Labels: labels},
			Status: stablev1alpha1.GameServerStatus{NodeName: "node3", State: stablev1alpha1.GameServerStateReady}},
	}

	m.AgonesClient.AddReactor("list", "gameservers", func(action k8stesting.Action) (bool, k8sruntime.Object, error) {
		return true, &stablev1alpha1.GameServerList{Items: gsList}, nil
	})

	stop, cancel := agtesting.StartInformers(m, hasSync)
	defer cancel()

	// This call initializes the cache
	err := c.syncReadyGSServerCache()
	assert.Nil(t, err)

	err = c.counter.Run(0, stop)
	assert.Nil(t, err)

	gs, err := c.findReadyGameServerForAllocation(gsa, distributedComparator)
	assert.Nil(t, err)
	assert.Equal(t, "node3", gs.Status.NodeName)
	assert.Equal(t, stablev1alpha1.GameServerStateReady, gs.Status.State)

	change := gsList[6].DeepCopy()
	change.Status.State = stablev1alpha1.GameServerStateAllocated
	watch.Modify(change)
	assert.True(t, cache.WaitForCacheSync(stop, hasSync))

	gs, err = c.findReadyGameServerForAllocation(gsa, distributedComparator)
	assert.Nil(t, err)
	assert.Equal(t, "node2", gs.Status.NodeName)
	assert.Equal(t, stablev1alpha1.GameServerStateReady, gs.Status.State)

	change = gsList[4].DeepCopy()
	change.Status.State = stablev1alpha1.GameServerStateAllocated
	watch.Modify(change)
	assert.True(t, cache.WaitForCacheSync(stop, hasSync))

	gs, err = c.findReadyGameServerForAllocation(gsa, distributedComparator)
	assert.Nil(t, err)
	assert.Equal(t, "node1", gs.Status.NodeName)
	assert.Equal(t, stablev1alpha1.GameServerStateReady, gs.Status.State)

	change = gsList[0].DeepCopy()
	change.Status.State = stablev1alpha1.GameServerStateAllocated
	watch.Modify(change)
	assert.True(t, cache.WaitForCacheSync(stop, hasSync))

	gs, err = c.findReadyGameServerForAllocation(gsa, distributedComparator)
	assert.Nil(t, err)
	assert.Equal(t, "node2", gs.Status.NodeName)
	assert.Equal(t, stablev1alpha1.GameServerStateReady, gs.Status.State)

	change = gsList[5].DeepCopy()
	change.Status.State = stablev1alpha1.GameServerStateAllocated
	watch.Modify(change)
	assert.True(t, cache.WaitForCacheSync(stop, hasSync))

	gs, err = c.findReadyGameServerForAllocation(gsa, distributedComparator)
	assert.Nil(t, err)
	assert.Equal(t, "node1", gs.Status.NodeName)
	assert.Equal(t, stablev1alpha1.GameServerStateReady, gs.Status.State)

	change = gsList[1].DeepCopy()
	change.Status.State = stablev1alpha1.GameServerStateAllocated
	watch.Modify(change)
	assert.True(t, cache.WaitForCacheSync(stop, hasSync))

	gs, err = c.findReadyGameServerForAllocation(gsa, distributedComparator)
	assert.Nil(t, err)
	assert.Equal(t, "node1", gs.Status.NodeName)
	assert.Equal(t, stablev1alpha1.GameServerStateReady, gs.Status.State)

	change = gsList[2].DeepCopy()
	change.Status.State = stablev1alpha1.GameServerStateAllocated
	watch.Modify(change)
	assert.True(t, cache.WaitForCacheSync(stop, hasSync))

	gs, err = c.findReadyGameServerForAllocation(gsa, distributedComparator)
	assert.Equal(t, ErrNoGameServerReady, err)
	assert.Nil(t, gs)
}

func TestAllocationApiResource(t *testing.T) {
	t.Parallel()

	_, m := newFakeController()
	ts := httptest.NewServer(m.Mux)
	defer ts.Close()

	client := ts.Client()

	resp, err := client.Get(ts.URL + "/apis/" + v1alpha1.SchemeGroupVersion.String())
	if !assert.Nil(t, err) {
		assert.FailNow(t, err.Error())
	}
	defer resp.Body.Close() // nolint: errcheck

	list := &metav1.APIResourceList{}
	err = json.NewDecoder(resp.Body).Decode(list)
	assert.Nil(t, err)

	assert.Len(t, list.APIResources, 1)
	assert.Equal(t, "gameserverallocation", list.APIResources[0].SingularName)
}

func TestControllerRunCacheSync(t *testing.T) {
	c, m := newFakeController()
	watch := watch.NewFake()

	m.AgonesClient.AddWatchReactor("gameservers", k8stesting.DefaultWatchReactor(watch, nil))

	stop, cancel := agtesting.StartInformers(m)
	defer cancel()

	assertCacheEntries := func(expected int) {
		count := 0
		err := wait.PollImmediate(time.Second, 5*time.Second, func() (done bool, err error) {
			count = 0
			c.readyGameServers.Range(func(key string, gs *stablev1alpha1.GameServer) bool {
				count++
				return true
			})

			return count == expected, nil
		})

		assert.NoError(t, err, fmt.Sprintf("Should be %d values", expected))
	}

	go func() {
		err := c.Run(1, stop)
		assert.Nil(t, err)
	}()

	gs := stablev1alpha1.GameServer{
		ObjectMeta: metav1.ObjectMeta{Name: "gs1", Namespace: "default"},
		Status:     stablev1alpha1.GameServerStatus{State: stablev1alpha1.GameServerStateStarting},
	}

	logrus.Info("adding ready game server")
	watch.Add(gs.DeepCopy())

	assertCacheEntries(0)

	gs.Status.State = stablev1alpha1.GameServerStateReady
	watch.Modify(gs.DeepCopy())

	assertCacheEntries(1)

	// try again, should be no change
	gs.Status.State = stablev1alpha1.GameServerStateReady
	watch.Modify(gs.DeepCopy())

	assertCacheEntries(1)

	// now move it to Shutdown
	gs.Status.State = stablev1alpha1.GameServerStateShutdown
	watch.Modify(gs.DeepCopy())

	assertCacheEntries(0)
}

func TestGetRandomlySelectedGS(t *testing.T) {
	c, _ := newFakeController()
	c.topNGameServerCount = 5
	gsa := &v1alpha1.GameServerAllocation{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: defaultNs,
			Name:      "allocation",
		},
		Status: v1alpha1.GameServerAllocationStatus{
			State: v1alpha1.GameServerAllocationUnAllocated,
		},
	}

	_, _, gsList := defaultFixtures(10)

	selectedGS := c.getRandomlySelectedGS(gsa, gsList)
	assert.NotNil(t, "selectedGS can't be nil", selectedGS)
	for i := 1; i <= 5; i++ {
		expectedName := "gs" + strconv.Itoa(i)
		assert.NotEqual(t, expectedName, selectedGS.ObjectMeta.Name)
	}

	_, _, gsList = defaultFixtures(5)

	selectedGS = c.getRandomlySelectedGS(gsa, gsList)
	assert.NotNil(t, "selectedGS can't be nil", selectedGS)

	_, _, gsList = defaultFixtures(1)

	selectedGS = c.getRandomlySelectedGS(gsa, gsList)
	assert.NotNil(t, "selectedGS can't be nil", selectedGS)
	assert.Equal(t, "gs1", selectedGS.ObjectMeta.Name)
}

func defaultFixtures(gsLen int) (*stablev1alpha1.Fleet, *stablev1alpha1.GameServerSet, []stablev1alpha1.GameServer) {
	f := &stablev1alpha1.Fleet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "fleet-1",
			Namespace: defaultNs,
			UID:       "1234",
		},
		Spec: stablev1alpha1.FleetSpec{
			Replicas: 5,
			Template: stablev1alpha1.GameServerTemplateSpec{},
		},
	}
	f.ApplyDefaults()
	gsSet := f.GameServerSet()
	gsSet.ObjectMeta.Name = "gsSet1"
	var gsList []stablev1alpha1.GameServer
	for i := 1; i <= gsLen; i++ {
		gs := gsSet.GameServer()
		gs.ObjectMeta.Name = "gs" + strconv.Itoa(i)
		gs.Status.State = stablev1alpha1.GameServerStateReady
		gsList = append(gsList, *gs)
	}
	return f, gsSet, gsList
}

// newFakeController returns a controller, backed by the fake Clientset
func newFakeController() (*Controller, agtesting.Mocks) {
	m := agtesting.NewMocks()
	m.Mux = http.NewServeMux()
	counter := gameservers.NewPerNodeCounter(m.KubeInformerFactory, m.AgonesInformerFactory)
	api := apiserver.NewAPIServer(m.Mux)
	c := NewController(api, healthcheck.NewHandler(), counter, 1, m.KubeClient, m.AgonesClient, m.AgonesInformerFactory)
	c.recorder = m.FakeRecorder
	return c, m
}
