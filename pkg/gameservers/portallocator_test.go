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

package gameservers

import (
	"fmt"
	"testing"

	"sync"

	"github.com/agonio/agon/pkg/apis/stable/v1alpha1"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	k8stesting "k8s.io/client-go/testing"
	"k8s.io/client-go/tools/cache"
)

var (
	n1 = corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: "node1"}}
	n2 = corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: "node2"}}
	n3 = corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: "node3"}}
)

func TestPortAllocatorAllocate(t *testing.T) {
	t.Parallel()

	t.Run("ports are all allocated", func(t *testing.T) {
		m := newMocks()
		pa := NewPortAllocator(10, 20, m.kubeInformationFactory, m.agonInformerFactory)
		nodeWatch := watch.NewFake()
		m.kubeClient.AddWatchReactor("nodes", k8stesting.DefaultWatchReactor(nodeWatch, nil))

		stop, cancel := startInformers(m)
		defer cancel()

		// Make sure the add's don't corrupt the sync
		nodeWatch.Add(&n1)
		nodeWatch.Add(&n2)

		err := pa.Run(stop)
		assert.Nil(t, err)

		// two nodes
		for x := 0; x < 2; x++ {
			// ports between 10 and 20
			for i := 10; i <= 20; i++ {
				var p int32
				p, err = pa.Allocate()
				assert.True(t, 10 <= p && p <= 20, "%v is not between 10 and 20", p)
				assert.Nil(t, err)
			}
		}

		// now we should have none left
		_, err = pa.Allocate()
		assert.Equal(t, ErrPortNotFound, err)
	})

	t.Run("ports are unique in a node", func(t *testing.T) {
		m := newMocks()
		pa := NewPortAllocator(10, 20, m.kubeInformationFactory, m.agonInformerFactory)

		m.kubeClient.AddReactor("list", "nodes", func(action k8stesting.Action) (bool, runtime.Object, error) {
			nl := &corev1.NodeList{Items: []corev1.Node{n1}}
			return true, nl, nil
		})
		stop, cancel := startInformers(m)
		defer cancel()
		err := pa.Run(stop)
		assert.Nil(t, err)
		var ports []int32
		for i := 10; i <= 20; i++ {
			p, err := pa.Allocate()
			assert.Nil(t, err)
			assert.NotContains(t, ports, p)
			ports = append(ports, p)
		}
	})
}

func TestPortAllocatorMultithreadAllocate(t *testing.T) {
	m := newMocks()
	pa := NewPortAllocator(10, 110, m.kubeInformationFactory, m.agonInformerFactory)

	m.kubeClient.AddReactor("list", "nodes", func(action k8stesting.Action) (bool, runtime.Object, error) {
		nl := &corev1.NodeList{Items: []corev1.Node{n1, n2}}
		return true, nl, nil
	})
	stop, cancel := startInformers(m)
	defer cancel()
	err := pa.Run(stop)
	assert.Nil(t, err)
	wg := sync.WaitGroup{}

	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func(i int) {
			for x := 0; x < 10; x++ {
				logrus.WithField("x", x).WithField("i", i).Info("allocating!")
				_, err := pa.Allocate()
				assert.Nil(t, err)
			}
			wg.Done()
		}(i)
	}

	wg.Wait()
}

func TestPortAllocatorSyncPortAllocations(t *testing.T) {
	t.Parallel()

	m := newMocks()
	pa := NewPortAllocator(10, 20, m.kubeInformationFactory, m.agonInformerFactory)

	m.kubeClient.AddReactor("list", "nodes", func(action k8stesting.Action) (bool, runtime.Object, error) {
		nl := &corev1.NodeList{Items: []corev1.Node{n1, n2, n3}}
		return true, nl, nil
	})

	m.agonClient.AddReactor("list", "gameservers", func(action k8stesting.Action) (bool, runtime.Object, error) {
		gs1 := v1alpha1.GameServer{ObjectMeta: metav1.ObjectMeta{Name: "gs1"}, Spec: v1alpha1.GameServerSpec{HostPort: 10},
			Status: v1alpha1.GameServerStatus{State: v1alpha1.Ready, Port: 10, NodeName: n1.ObjectMeta.Name}}
		gs2 := v1alpha1.GameServer{ObjectMeta: metav1.ObjectMeta{Name: "gs2"}, Spec: v1alpha1.GameServerSpec{HostPort: 10},
			Status: v1alpha1.GameServerStatus{State: v1alpha1.Ready, Port: 10, NodeName: n2.ObjectMeta.Name}}
		gs3 := v1alpha1.GameServer{ObjectMeta: metav1.ObjectMeta{Name: "gs3"}, Spec: v1alpha1.GameServerSpec{HostPort: 11},
			Status: v1alpha1.GameServerStatus{State: v1alpha1.Ready, Port: 11, NodeName: n3.ObjectMeta.Name}}
		gs4 := v1alpha1.GameServer{ObjectMeta: metav1.ObjectMeta{Name: "gs4"}, Spec: v1alpha1.GameServerSpec{HostPort: 12},
			Status: v1alpha1.GameServerStatus{State: v1alpha1.Creating}}
		gs5 := v1alpha1.GameServer{ObjectMeta: metav1.ObjectMeta{Name: "gs5"}, Spec: v1alpha1.GameServerSpec{HostPort: 12},
			Status: v1alpha1.GameServerStatus{State: v1alpha1.Creating}}
		gsl := &v1alpha1.GameServerList{Items: []v1alpha1.GameServer{gs1, gs2, gs3, gs4, gs5}}
		return true, gsl, nil
	})

	stop, cancel := startInformers(m)
	defer cancel()
	err := pa.syncPortAllocations(stop)
	assert.Nil(t, err)
	assert.Len(t, pa.portAllocations, 3)

	// count the number of allocated ports,
	assert.Equal(t, 2, countAllocatedPorts(pa, 10))
	assert.Equal(t, 1, countAllocatedPorts(pa, 11))
	assert.Equal(t, 2, countAllocatedPorts(pa, 12))

	count := 0
	for i := int32(10); i <= 20; i++ {
		count += countAllocatedPorts(pa, i)
	}
	assert.Equal(t, 5, count)
}

func TestPortAllocatorSyncDeleteGameServer(t *testing.T) {
	t.Parallel()

	m := newMocks()
	fakeWatch := watch.NewFake()
	m.agonClient.AddWatchReactor("gameservers", k8stesting.DefaultWatchReactor(fakeWatch, nil))

	gs1Fixture := &v1alpha1.GameServer{ObjectMeta: metav1.ObjectMeta{Name: "gs4"}, Spec: v1alpha1.GameServerSpec{HostPort: 10}}
	gs2Fixture := gs1Fixture.DeepCopy()
	gs2Fixture.ObjectMeta.Name = "gs5"

	pa := NewPortAllocator(10, 20, m.kubeInformationFactory, m.agonInformerFactory)

	m.kubeClient.AddReactor("list", "nodes", func(action k8stesting.Action) (bool, runtime.Object, error) {
		nl := &corev1.NodeList{Items: []corev1.Node{n1, n2, n3}}
		return true, nl, nil
	})

	m.agonClient.AddReactor("list", "gameservers", func(action k8stesting.Action) (bool, runtime.Object, error) {
		gs1 := v1alpha1.GameServer{ObjectMeta: metav1.ObjectMeta{Name: "gs1"}, Spec: v1alpha1.GameServerSpec{HostPort: 10},
			Status: v1alpha1.GameServerStatus{State: v1alpha1.Ready, Port: 10, NodeName: n1.ObjectMeta.Name}}
		gs2 := v1alpha1.GameServer{ObjectMeta: metav1.ObjectMeta{Name: "gs2"}, Spec: v1alpha1.GameServerSpec{HostPort: 11},
			Status: v1alpha1.GameServerStatus{State: v1alpha1.Ready, Port: 11, NodeName: n1.ObjectMeta.Name}}
		gs3 := v1alpha1.GameServer{ObjectMeta: metav1.ObjectMeta{Name: "gs3"}, Spec: v1alpha1.GameServerSpec{HostPort: 10},
			Status: v1alpha1.GameServerStatus{State: v1alpha1.Ready, Port: 10, NodeName: n2.ObjectMeta.Name}}

		gsl := &v1alpha1.GameServerList{Items: []v1alpha1.GameServer{gs1, gs2, gs3}}
		return true, gsl, nil
	})

	stop, cancel := startInformers(m)
	defer cancel()

	// this should do nothing, as it's before pa.Created is called
	fakeWatch.Add(gs2Fixture.DeepCopy())
	fakeWatch.Delete(gs2Fixture.DeepCopy())

	err := pa.Run(stop)
	assert.Nil(t, err)

	nonGSPod := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "notagameserver"}}
	fakeWatch.Add(gs1Fixture.DeepCopy())
	fakeWatch.Add(nonGSPod.DeepCopy())
	assert.True(t, cache.WaitForCacheSync(stop, pa.gameServerSynced))

	// gate
	pa.mutex.RLock() // reading mutable state, so read lock
	assert.Equal(t, 2, countAllocatedPorts(pa, 10))
	assert.Equal(t, 1, countAllocatedPorts(pa, 11))
	pa.mutex.RUnlock()

	fakeWatch.Delete(gs1Fixture.DeepCopy())
	assert.True(t, cache.WaitForCacheSync(stop, pa.gameServerSynced))

	pa.mutex.RLock() // reading mutable state, so read lock
	assert.Equal(t, 1, countAllocatedPorts(pa, 10))
	assert.Equal(t, 1, countAllocatedPorts(pa, 11))
	pa.mutex.RUnlock()

	// delete the non gameserver pod, all should be the same
	fakeWatch.Delete(nonGSPod.DeepCopy())
	assert.True(t, cache.WaitForCacheSync(stop, pa.gameServerSynced))
	pa.mutex.RLock() // reading mutable state, so read lock
	assert.Equal(t, 1, countAllocatedPorts(pa, 10))
	assert.Equal(t, 1, countAllocatedPorts(pa, 11))
	pa.mutex.RUnlock()
}

func TestPortAllocatorNodeEvents(t *testing.T) {
	m := newMocks()
	pa := NewPortAllocator(10, 20, m.kubeInformationFactory, m.agonInformerFactory)
	nodeWatch := watch.NewFake()
	gsWatch := watch.NewFake()
	m.kubeClient.AddWatchReactor("nodes", k8stesting.DefaultWatchReactor(nodeWatch, nil))
	m.agonClient.AddWatchReactor("gameservers", k8stesting.DefaultWatchReactor(gsWatch, nil))

	stop, cancel := startInformers(m)
	defer cancel()

	// Make sure the add's don't corrupt the sync
	nodeWatch.Add(&n1)
	nodeWatch.Add(&n2)

	err := pa.Run(stop)
	assert.Nil(t, err)

	// add a game server
	port, err := pa.Allocate()
	assert.Nil(t, err)
	gs := v1alpha1.GameServer{ObjectMeta: metav1.ObjectMeta{Name: "gs1"}, Spec: v1alpha1.GameServerSpec{HostPort: port}}
	gsWatch.Add(&gs)

	pa.mutex.RLock()
	assert.Len(t, pa.portAllocations, 2)
	assert.Equal(t, 1, countAllocatedPorts(pa, port))
	pa.mutex.RUnlock()

	// add the n3 node
	logrus.Info("adding n3")
	nodeWatch.Add(&n3)
	assert.True(t, cache.WaitForCacheSync(stop, pa.nodeSynced))

	pa.mutex.RLock()
	assert.Len(t, pa.portAllocations, 3)
	assert.Equal(t, 1, countAllocatedPorts(pa, port))
	pa.mutex.RUnlock()

	// mark the node as unscheduled
	logrus.Info("unscheduling n3")
	copy := n3.DeepCopy()
	copy.Spec.Unschedulable = true
	assert.True(t, copy.Spec.Unschedulable)
	nodeWatch.Modify(copy)
	assert.True(t, cache.WaitForCacheSync(stop, pa.nodeSynced))
	pa.mutex.RLock()
	assert.Len(t, pa.portAllocations, 2)
	assert.Equal(t, 1, countAllocatedPorts(pa, port))
	pa.mutex.RUnlock()

	// schedule the n3 node again
	logrus.Info("scheduling n3")
	copy = n3.DeepCopy()
	copy.Spec.Unschedulable = false
	nodeWatch.Modify(copy)
	assert.True(t, cache.WaitForCacheSync(stop, pa.nodeSynced))
	pa.mutex.RLock()
	assert.Len(t, pa.portAllocations, 3)
	assert.Equal(t, 1, countAllocatedPorts(pa, port))
	pa.mutex.RUnlock()

	// delete the n3 node
	logrus.Info("deleting n3")
	nodeWatch.Delete(n3.DeepCopy())
	assert.True(t, cache.WaitForCacheSync(stop, pa.nodeSynced))
	pa.mutex.RLock()
	assert.Len(t, pa.portAllocations, 2)
	assert.Equal(t, 1, countAllocatedPorts(pa, port))
	pa.mutex.RUnlock()
}

func TestNodePortAllocation(t *testing.T) {
	t.Parallel()

	m := newMocks()
	pa := NewPortAllocator(10, 20, m.kubeInformationFactory, m.agonInformerFactory)
	nodes := []corev1.Node{n1, n2, n3}
	m.kubeClient.AddReactor("list", "nodes", func(action k8stesting.Action) (bool, runtime.Object, error) {
		nl := &corev1.NodeList{Items: nodes}
		return true, nl, nil
	})
	result := pa.nodePortAllocation([]*corev1.Node{&n1, &n2, &n3})
	assert.Len(t, result, 3)
	for _, n := range nodes {
		ports, ok := result[n.ObjectMeta.Name]
		assert.True(t, ok, "Should have a port allocation for %s", n.ObjectMeta.Name)
		assert.Len(t, ports, 11)
		for _, v := range ports {
			assert.False(t, v)
		}
	}
}

func TestTakePortAllocation(t *testing.T) {
	t.Parallel()

	fixture := []portAllocation{{1: false, 2: false}, {1: false, 2: false}, {1: false, 3: false}}
	result := setPortAllocation(2, fixture, true)
	assert.True(t, result[0][2])

	for i, row := range fixture {
		for p, taken := range row {
			if i != 0 && p != 2 {
				assert.False(t, taken, fmt.Sprintf("row %d and port %d should be false", i, p))
			}
		}
	}
}

// countAllocatedPorts counts how many of a given port have been
// allocated across nodes
func countAllocatedPorts(pa *PortAllocator, p int32) int {
	count := 0
	for _, node := range pa.portAllocations {
		if node[p] {
			count++
		}
	}
	return count
}
