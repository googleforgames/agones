// Copyright 2019 Google Inc. All Rights Reserved.
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
	"testing"

	"agones.dev/agones/pkg/apis/stable/v1alpha1"
	agtesting "agones.dev/agones/pkg/testing"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	k8stesting "k8s.io/client-go/testing"
	"k8s.io/client-go/tools/cache"
)

const (
	defaultNs = "default"
	name1     = "node1"
	name2     = "node2"
)

func TestPerNodeCounterGameServerEvents(t *testing.T) {
	t.Parallel()

	pnc, m := newFakePerNodeCounter()

	watch := watch.NewFake()
	m.AgonesClient.AddWatchReactor("gameservers", k8stesting.DefaultWatchReactor(watch, nil))

	hasSynced := m.AgonesInformerFactory.Stable().V1alpha1().GameServers().Informer().HasSynced
	stop, cancel := agtesting.StartInformers(m)
	defer cancel()

	assert.Empty(t, pnc.Counts())

	gs := &v1alpha1.GameServer{
		ObjectMeta: metav1.ObjectMeta{Name: "gs1", Namespace: defaultNs},
		Status: v1alpha1.GameServerStatus{
			State: v1alpha1.GameServerStateStarting, NodeName: name1}}

	watch.Add(gs.DeepCopy())
	cache.WaitForCacheSync(stop, hasSynced)

	assert.Empty(t, pnc.Counts())

	gs.Status.State = v1alpha1.GameServerStateReady
	watch.Add(gs.DeepCopy())
	cache.WaitForCacheSync(stop, hasSynced)

	counts := pnc.Counts()
	assert.Len(t, counts, 1)
	assert.Equal(t, int64(1), counts[name1].Ready)
	assert.Equal(t, int64(0), counts[name1].Allocated)

	gs.Status.State = v1alpha1.GameServerStateAllocated
	watch.Add(gs.DeepCopy())
	cache.WaitForCacheSync(stop, hasSynced)

	counts = pnc.Counts()
	assert.Len(t, counts, 1)
	assert.Equal(t, int64(0), counts[name1].Ready)
	assert.Equal(t, int64(1), counts[name1].Allocated)

	gs.Status.State = v1alpha1.GameServerStateShutdown
	watch.Add(gs.DeepCopy())
	cache.WaitForCacheSync(stop, hasSynced)

	counts = pnc.Counts()
	assert.Len(t, counts, 1)
	assert.Equal(t, int64(0), counts[name1].Ready)
	assert.Equal(t, int64(0), counts[name1].Allocated)

	gs.ObjectMeta.Name = "gs2"
	gs.Status.State = v1alpha1.GameServerStateReady
	gs.Status.NodeName = name2

	watch.Add(gs.DeepCopy())
	cache.WaitForCacheSync(stop, hasSynced)

	counts = pnc.Counts()
	assert.Len(t, counts, 2)
	assert.Equal(t, int64(0), counts[name1].Ready)
	assert.Equal(t, int64(0), counts[name1].Allocated)
	assert.Equal(t, int64(1), counts[name2].Ready)
	assert.Equal(t, int64(0), counts[name2].Allocated)

	gs.ObjectMeta.Name = "gs3"
	// not likely, but to test the flow
	gs.Status.State = v1alpha1.GameServerStateAllocated
	gs.Status.NodeName = name2

	watch.Add(gs.DeepCopy())
	cache.WaitForCacheSync(stop, hasSynced)

	counts = pnc.Counts()
	assert.Len(t, counts, 2)
	assert.Equal(t, int64(0), counts[name1].Ready)
	assert.Equal(t, int64(0), counts[name1].Allocated)
	assert.Equal(t, int64(1), counts[name2].Ready)
	assert.Equal(t, int64(1), counts[name2].Allocated)
}

func TestPerNodeCounterNodeEvents(t *testing.T) {
	t.Parallel()

	pnc, m := newFakePerNodeCounter()

	gsWatch := watch.NewFake()
	nodeWatch := watch.NewFake()
	m.AgonesClient.AddWatchReactor("gameservers", k8stesting.DefaultWatchReactor(gsWatch, nil))
	m.KubeClient.AddWatchReactor("nodes", k8stesting.DefaultWatchReactor(nodeWatch, nil))

	gsSynced := m.AgonesInformerFactory.Stable().V1alpha1().GameServers().Informer().HasSynced
	nodeSynced := m.KubeInformerFactory.Core().V1().Nodes().Informer().HasSynced

	stop, cancel := agtesting.StartInformers(m)
	defer cancel()

	assert.Empty(t, pnc.Counts())

	gs := &v1alpha1.GameServer{
		ObjectMeta: metav1.ObjectMeta{Name: "gs1", Namespace: defaultNs},
		Status: v1alpha1.GameServerStatus{
			State: v1alpha1.GameServerStateReady, NodeName: name1}}
	node := &corev1.Node{ObjectMeta: metav1.ObjectMeta{Namespace: defaultNs, Name: name1}}

	gsWatch.Add(gs.DeepCopy())
	nodeWatch.Add(node.DeepCopy())
	cache.WaitForCacheSync(stop, gsSynced, nodeSynced)
	assert.Len(t, pnc.Counts(), 1)

	nodeWatch.Delete(node.DeepCopy())
	cache.WaitForCacheSync(stop, nodeSynced)
	assert.Empty(t, pnc.Counts())
}

func TestPerNodeCounterRun(t *testing.T) {
	t.Parallel()
	pnc, m := newFakePerNodeCounter()

	gs1 := &v1alpha1.GameServer{
		ObjectMeta: metav1.ObjectMeta{Name: "gs1", Namespace: defaultNs},
		Status: v1alpha1.GameServerStatus{
			State: v1alpha1.GameServerStateReady, NodeName: name1}}

	gs2 := gs1.DeepCopy()
	gs2.ObjectMeta.Name = "gs2"
	gs2.Status.State = v1alpha1.GameServerStateAllocated

	gs3 := gs1.DeepCopy()
	gs3.ObjectMeta.Name = "gs3"
	gs3.Status.State = v1alpha1.GameServerStateStarting
	gs3.Status.NodeName = name2

	gs4 := gs1.DeepCopy()
	gs4.ObjectMeta.Name = "gs4"
	gs4.Status.State = v1alpha1.GameServerStateAllocated

	m.AgonesClient.AddReactor("list", "gameservers", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, &v1alpha1.GameServerList{Items: []v1alpha1.GameServer{*gs1, *gs2, *gs3, *gs4}}, nil
	})

	stop, cancel := agtesting.StartInformers(m)
	defer cancel()

	err := pnc.Run(0, stop)
	assert.Nil(t, err)

	counts := pnc.Counts()

	assert.Len(t, counts, 2)
	assert.Equal(t, int64(1), counts[name1].Ready)
	assert.Equal(t, int64(2), counts[name1].Allocated)
	assert.Equal(t, int64(0), counts[name2].Ready)
	assert.Equal(t, int64(0), counts[name2].Allocated)
}

// newFakeController returns a controller, backed by the fake Clientset
func newFakePerNodeCounter() (*PerNodeCounter, agtesting.Mocks) {
	m := agtesting.NewMocks()
	c := NewPerNodeCounter(m.KubeInformerFactory, m.AgonesInformerFactory)
	return c, m
}
