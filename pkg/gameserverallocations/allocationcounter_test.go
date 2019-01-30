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
	"testing"

	"k8s.io/apimachinery/pkg/runtime"

	"agones.dev/agones/pkg/apis/stable/v1alpha1"
	agtesting "agones.dev/agones/pkg/testing"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	k8stesting "k8s.io/client-go/testing"
	"k8s.io/client-go/tools/cache"
)

func TestAllocationCounterGameServerEvents(t *testing.T) {
	t.Parallel()

	ac, m := newFakeAllocationCounter()

	watch := watch.NewFake()
	m.AgonesClient.AddWatchReactor("gameservers", k8stesting.DefaultWatchReactor(watch, nil))

	hasSynced := m.AgonesInformerFactory.Stable().V1alpha1().GameServers().Informer().HasSynced
	stop, cancel := agtesting.StartInformers(m)
	defer cancel()

	assert.Empty(t, ac.Counts())

	gs := &v1alpha1.GameServer{
		ObjectMeta: metav1.ObjectMeta{Name: "gs1", Namespace: defaultNs},
		Status: v1alpha1.GameServerStatus{
			State: v1alpha1.GameServerStateStarting, NodeName: n1}}

	watch.Add(gs.DeepCopy())
	cache.WaitForCacheSync(stop, hasSynced)

	assert.Empty(t, ac.Counts())

	gs.Status.State = v1alpha1.GameServerStateReady
	watch.Add(gs.DeepCopy())
	cache.WaitForCacheSync(stop, hasSynced)

	counts := ac.Counts()
	assert.Len(t, counts, 1)
	assert.Equal(t, int64(1), counts[n1].ready)
	assert.Equal(t, int64(0), counts[n1].allocated)

	gs.Status.State = v1alpha1.GameServerStateAllocated
	watch.Add(gs.DeepCopy())
	cache.WaitForCacheSync(stop, hasSynced)

	counts = ac.Counts()
	assert.Len(t, counts, 1)
	assert.Equal(t, int64(0), counts[n1].ready)
	assert.Equal(t, int64(1), counts[n1].allocated)

	gs.Status.State = v1alpha1.GameServerStateShutdown
	watch.Add(gs.DeepCopy())
	cache.WaitForCacheSync(stop, hasSynced)

	counts = ac.Counts()
	assert.Len(t, counts, 1)
	assert.Equal(t, int64(0), counts[n1].ready)
	assert.Equal(t, int64(0), counts[n1].allocated)

	gs.ObjectMeta.Name = "gs2"
	gs.Status.State = v1alpha1.GameServerStateReady
	gs.Status.NodeName = n2

	watch.Add(gs.DeepCopy())
	cache.WaitForCacheSync(stop, hasSynced)

	counts = ac.Counts()
	assert.Len(t, counts, 2)
	assert.Equal(t, int64(0), counts[n1].ready)
	assert.Equal(t, int64(0), counts[n1].allocated)
	assert.Equal(t, int64(1), counts[n2].ready)
	assert.Equal(t, int64(0), counts[n2].allocated)

	gs.ObjectMeta.Name = "gs3"
	// not likely, but to test the flow
	gs.Status.State = v1alpha1.GameServerStateAllocated
	gs.Status.NodeName = n2

	watch.Add(gs.DeepCopy())
	cache.WaitForCacheSync(stop, hasSynced)

	counts = ac.Counts()
	assert.Len(t, counts, 2)
	assert.Equal(t, int64(0), counts[n1].ready)
	assert.Equal(t, int64(0), counts[n1].allocated)
	assert.Equal(t, int64(1), counts[n2].ready)
	assert.Equal(t, int64(1), counts[n2].allocated)
}

func TestAllocationCountNodeEvents(t *testing.T) {
	t.Parallel()

	ac, m := newFakeAllocationCounter()

	gsWatch := watch.NewFake()
	nodeWatch := watch.NewFake()
	m.AgonesClient.AddWatchReactor("gameservers", k8stesting.DefaultWatchReactor(gsWatch, nil))
	m.KubeClient.AddWatchReactor("nodes", k8stesting.DefaultWatchReactor(nodeWatch, nil))

	gsSynced := m.AgonesInformerFactory.Stable().V1alpha1().GameServers().Informer().HasSynced
	nodeSynced := m.KubeInformationFactory.Core().V1().Nodes().Informer().HasSynced

	stop, cancel := agtesting.StartInformers(m)
	defer cancel()

	assert.Empty(t, ac.Counts())

	gs := &v1alpha1.GameServer{
		ObjectMeta: metav1.ObjectMeta{Name: "gs1", Namespace: defaultNs},
		Status: v1alpha1.GameServerStatus{
			State: v1alpha1.GameServerStateReady, NodeName: n1}}
	node := &corev1.Node{ObjectMeta: metav1.ObjectMeta{Namespace: defaultNs, Name: n1}}

	gsWatch.Add(gs.DeepCopy())
	nodeWatch.Add(node.DeepCopy())
	cache.WaitForCacheSync(stop, gsSynced, nodeSynced)
	assert.Len(t, ac.Counts(), 1)

	nodeWatch.Delete(node.DeepCopy())
	cache.WaitForCacheSync(stop, nodeSynced)
	assert.Empty(t, ac.Counts())
}

func TestAllocationCounterRun(t *testing.T) {
	t.Parallel()
	ac, m := newFakeAllocationCounter()

	gs1 := &v1alpha1.GameServer{
		ObjectMeta: metav1.ObjectMeta{Name: "gs1", Namespace: defaultNs},
		Status: v1alpha1.GameServerStatus{
			State: v1alpha1.GameServerStateReady, NodeName: n1}}

	gs2 := gs1.DeepCopy()
	gs2.ObjectMeta.Name = "gs2"
	gs2.Status.State = v1alpha1.GameServerStateAllocated

	gs3 := gs1.DeepCopy()
	gs3.ObjectMeta.Name = "gs3"
	gs3.Status.State = v1alpha1.GameServerStateStarting
	gs3.Status.NodeName = n2

	gs4 := gs1.DeepCopy()
	gs4.ObjectMeta.Name = "gs4"
	gs4.Status.State = v1alpha1.GameServerStateAllocated

	m.AgonesClient.AddReactor("list", "gameservers", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, &v1alpha1.GameServerList{Items: []v1alpha1.GameServer{*gs1, *gs2, *gs3, *gs4}}, nil
	})

	stop, cancel := agtesting.StartInformers(m)
	defer cancel()

	err := ac.Run(stop)
	assert.Nil(t, err)

	counts := ac.Counts()

	assert.Len(t, counts, 2)
	assert.Equal(t, int64(1), counts[n1].ready)
	assert.Equal(t, int64(2), counts[n1].allocated)
	assert.Equal(t, int64(0), counts[n2].ready)
	assert.Equal(t, int64(0), counts[n2].allocated)
}

// newFakeController returns a controller, backed by the fake Clientset
func newFakeAllocationCounter() (*AllocationCounter, agtesting.Mocks) {
	m := agtesting.NewMocks()
	c := NewAllocationCounter(m.KubeInformationFactory, m.AgonesInformerFactory)
	return c, m
}
