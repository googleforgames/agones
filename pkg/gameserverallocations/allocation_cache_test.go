// Copyright 2021 Google LLC All Rights Reserved.
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
	"fmt"
	"testing"
	"time"

	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	"agones.dev/agones/pkg/gameservers"
	agtesting "agones.dev/agones/pkg/testing"
	"agones.dev/agones/pkg/util/runtime"
	"github.com/heptiolabs/healthcheck"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"
	k8stesting "k8s.io/client-go/testing"
)

func TestAllocationCacheListSortedGameServers(t *testing.T) {
	t.Parallel()
	runtime.FeatureTestMutex.Lock()
	defer runtime.FeatureTestMutex.Unlock()

	gs1 := agonesv1.GameServer{ObjectMeta: metav1.ObjectMeta{Name: "gs1", Namespace: defaultNs, UID: "1"}, Status: agonesv1.GameServerStatus{NodeName: "node1", State: agonesv1.GameServerStateReady}}
	gs2 := agonesv1.GameServer{ObjectMeta: metav1.ObjectMeta{Name: "gs2", Namespace: defaultNs, UID: "2"}, Status: agonesv1.GameServerStatus{NodeName: "node2", State: agonesv1.GameServerStateReady}}
	gs3 := agonesv1.GameServer{ObjectMeta: metav1.ObjectMeta{Name: "gs3", Namespace: defaultNs, UID: "3"}, Status: agonesv1.GameServerStatus{NodeName: "node1", State: agonesv1.GameServerStateAllocated}}
	gs4 := agonesv1.GameServer{ObjectMeta: metav1.ObjectMeta{Name: "gs4", Namespace: defaultNs, UID: "4"}, Status: agonesv1.GameServerStatus{NodeName: "node2", State: agonesv1.GameServerStateReady}}

	gs5 := agonesv1.GameServer{ObjectMeta: metav1.ObjectMeta{Name: "gs5", Namespace: defaultNs, UID: "5"},
		Status: agonesv1.GameServerStatus{NodeName: "node1", State: agonesv1.GameServerStateReady, Players: &agonesv1.PlayerStatus{
			Count:    0,
			Capacity: 10,
		}},
	}

	gs6 := agonesv1.GameServer{ObjectMeta: metav1.ObjectMeta{Name: "gs6", Namespace: defaultNs, UID: "6"},
		Status: agonesv1.GameServerStatus{NodeName: "node1", State: agonesv1.GameServerStateReady, Players: &agonesv1.PlayerStatus{
			Count:    2,
			Capacity: 10,
		}},
	}

	fixtures := map[string]struct {
		list     []agonesv1.GameServer
		test     func(*testing.T, []*agonesv1.GameServer)
		features string
	}{
		"most allocated": {
			// node1: 1 ready, 1 allocated, node2: 1 ready
			list: []agonesv1.GameServer{gs1, gs2, gs3},
			test: func(t *testing.T, list []*agonesv1.GameServer) {
				assert.Len(t, list, 2)
				if !assert.Equal(t, []*agonesv1.GameServer{&gs1, &gs2}, list) {
					for _, gs := range list {
						logrus.WithField("name", gs.Name).Info("game server")
					}
				}
			},
			features: fmt.Sprintf("%s=false", runtime.FeatureStateAllocationFilter),
		},
		"allocated first (StateAllocationFilter)": {
			list:     []agonesv1.GameServer{gs1, gs2, gs3},
			features: fmt.Sprintf("%s=true", runtime.FeatureStateAllocationFilter),
			test: func(t *testing.T, list []*agonesv1.GameServer) {
				assert.Equal(t, []*agonesv1.GameServer{&gs3, &gs1, &gs2}, list)
			},
		},
		"nil player status (PlayerAllocationFilter)": {
			list:     []agonesv1.GameServer{gs1, gs2, gs4},
			features: fmt.Sprintf("%s=true", runtime.FeaturePlayerAllocationFilter),
			test: func(t *testing.T, list []*agonesv1.GameServer) {
				require.Len(t, list, 3)
				// first two items can come in any order
				assert.ElementsMatchf(t, []*agonesv1.GameServer{&gs2, &gs4}, list[:2], "GameServer Names")
				assert.Equal(t, &gs1, list[2])
			},
		},
		"least player capacity (PlayerAllocationFilter)": {
			list:     []agonesv1.GameServer{gs5, gs6},
			features: fmt.Sprintf("%s=true", runtime.FeaturePlayerAllocationFilter),
			test: func(t *testing.T, list []*agonesv1.GameServer) {
				assert.Equal(t, []*agonesv1.GameServer{&gs6, &gs5}, list)
			},
		},
		"list ready": {
			// node1: 1 ready, node2: 2 ready
			list: []agonesv1.GameServer{gs1, gs2, gs4},
			test: func(t *testing.T, list []*agonesv1.GameServer) {
				assert.Len(t, list, 3)
				// first two items can come in any order
				assert.ElementsMatchf(t, []*agonesv1.GameServer{&gs2, &gs4}, list[:2], "GameServer Names")
				assert.Equal(t, &gs1, list[2])
			},
		},
		"lexicographical (node name)": {
			list: []agonesv1.GameServer{gs2, gs1},
			test: func(t *testing.T, list []*agonesv1.GameServer) {
				assert.Len(t, list, 2)
				if !assert.Equal(t, []*agonesv1.GameServer{&gs1, &gs2}, list) {
					for _, gs := range list {
						logrus.WithField("name", gs.Name).Info("game server")
					}
				}
			},
		},
	}

	for k, v := range fixtures {
		k := k
		v := v
		t.Run(k, func(t *testing.T) {
			// deliberately not resetting the Feature state, to catch any possible unknown regressions with the
			// new feature flags
			if v.features != "" {
				require.NoError(t, runtime.ParseFeatures(v.features))
			}

			cache, m := newFakeAllocationCache()

			gsList := v.list

			m.AgonesClient.AddReactor("list", "gameservers", func(action k8stesting.Action) (bool, k8sruntime.Object, error) {
				return true, &agonesv1.GameServerList{Items: gsList}, nil
			})

			ctx, cancel := agtesting.StartInformers(m, cache.gameServerSynced)
			defer cancel()

			// This call initializes the cache
			err := cache.syncCache()
			assert.Nil(t, err)

			err = cache.counter.Run(ctx, 0)
			assert.Nil(t, err)

			list := cache.ListSortedGameServers()

			v.test(t, list)
		})
	}
}

func TestAllocatorRunCacheSyncFeatureStateAllocationFilter(t *testing.T) {
	t.Parallel()

	// TODO(markmandel): When this feature gets promoted to stable, replace test TestAllocatorRunCacheSync below with this test.
	runtime.FeatureTestMutex.Lock()
	defer runtime.FeatureTestMutex.Unlock()
	require.NoError(t, runtime.ParseFeatures(string(runtime.FeatureStateAllocationFilter)+"=true"))

	cache, m := newFakeAllocationCache()
	gsWatch := watch.NewFake()

	m.AgonesClient.AddWatchReactor("gameservers", k8stesting.DefaultWatchReactor(gsWatch, nil))

	ctx, cancel := agtesting.StartInformers(m, cache.gameServerSynced)
	defer cancel()

	assertCacheEntries := func(expected int) {
		count := 0
		err := wait.PollImmediate(time.Second, 5*time.Second, func() (done bool, err error) {
			count = 0
			cache.cache.Range(func(key string, gs *agonesv1.GameServer) bool {
				count++
				return true
			})

			return count == expected, nil
		})

		assert.NoError(t, err, fmt.Sprintf("Should be %d values", expected))
	}

	go func() {
		err := cache.Run(ctx)
		assert.Nil(t, err)
	}()

	gs := agonesv1.GameServer{
		ObjectMeta: metav1.ObjectMeta{Name: "gs1", Namespace: "default"},
		Status:     agonesv1.GameServerStatus{State: agonesv1.GameServerStateStarting},
	}

	logrus.Info("adding ready game server")
	gsWatch.Add(gs.DeepCopy())

	assertCacheEntries(0)

	gs.Status.State = agonesv1.GameServerStateReady
	gsWatch.Modify(gs.DeepCopy())

	assertCacheEntries(1)

	// try again, should be no change
	gs.Status.State = agonesv1.GameServerStateReady
	gsWatch.Modify(gs.DeepCopy())

	assertCacheEntries(1)

	// now move it to Shutdown
	gs.Status.State = agonesv1.GameServerStateShutdown
	gsWatch.Modify(gs.DeepCopy())
	assertCacheEntries(0)

	// add it back in as Allocated
	gs.Status.State = agonesv1.GameServerStateAllocated
	gsWatch.Modify(gs.DeepCopy())
	assertCacheEntries(1)

	// now move it to Shutdown
	gs.Status.State = agonesv1.GameServerStateShutdown
	gsWatch.Modify(gs.DeepCopy())
	assertCacheEntries(0)

	// add back in ready gameserver
	gs.Status.State = agonesv1.GameServerStateReady
	gsWatch.Modify(gs.DeepCopy())
	assertCacheEntries(1)

	// update with deletion timestamp
	n := metav1.Now()
	deletedCopy := gs.DeepCopy()
	deletedCopy.ObjectMeta.DeletionTimestamp = &n
	gsWatch.Modify(deletedCopy)
	assertCacheEntries(0)

	// add back in ready gameserver
	gs.Status.State = agonesv1.GameServerStateReady
	gsWatch.Modify(gs.DeepCopy())
	assertCacheEntries(1)

	// now actually delete it
	gsWatch.Delete(gs.DeepCopy())
	assertCacheEntries(0)
}

func TestAllocatorRunCacheSync(t *testing.T) {
	t.Parallel()

	cache, m := newFakeAllocationCache()
	gsWatch := watch.NewFake()

	m.AgonesClient.AddWatchReactor("gameservers", k8stesting.DefaultWatchReactor(gsWatch, nil))

	ctx, cancel := agtesting.StartInformers(m, cache.gameServerSynced)
	defer cancel()

	assertCacheEntries := func(expected int) {
		count := 0
		err := wait.PollImmediate(time.Second, 5*time.Second, func() (done bool, err error) {
			count = 0
			cache.cache.Range(func(key string, gs *agonesv1.GameServer) bool {
				count++
				return true
			})

			return count == expected, nil
		})

		assert.NoError(t, err, fmt.Sprintf("Should be %d values", expected))
	}

	go func() {
		err := cache.Run(ctx)
		assert.Nil(t, err)
	}()

	gs := agonesv1.GameServer{
		ObjectMeta: metav1.ObjectMeta{Name: "gs1", Namespace: "default"},
		Status:     agonesv1.GameServerStatus{State: agonesv1.GameServerStateStarting},
	}

	logrus.Info("adding ready game server")
	gsWatch.Add(gs.DeepCopy())

	assertCacheEntries(0)

	gs.Status.State = agonesv1.GameServerStateReady
	gsWatch.Modify(gs.DeepCopy())

	assertCacheEntries(1)

	// try again, should be no change
	gs.Status.State = agonesv1.GameServerStateReady
	gsWatch.Modify(gs.DeepCopy())

	assertCacheEntries(1)

	// now move it to Shutdown
	gs.Status.State = agonesv1.GameServerStateShutdown
	gsWatch.Modify(gs.DeepCopy())

	assertCacheEntries(0)

	// add back in ready gameserver
	gs.Status.State = agonesv1.GameServerStateReady
	gsWatch.Modify(gs.DeepCopy())
	assertCacheEntries(1)

	// update with deletion timestamp
	n := metav1.Now()
	deletedCopy := gs.DeepCopy()
	deletedCopy.ObjectMeta.DeletionTimestamp = &n
	gsWatch.Modify(deletedCopy)
	assertCacheEntries(0)

	// add back in ready gameserver
	gs.Status.State = agonesv1.GameServerStateReady
	gsWatch.Modify(gs.DeepCopy())
	assertCacheEntries(1)

	// now actually delete it
	gsWatch.Delete(gs.DeepCopy())
	assertCacheEntries(0)
}

func newFakeAllocationCache() (*AllocationCache, agtesting.Mocks) {
	m := agtesting.NewMocks()
	cache := NewAllocationCache(m.AgonesInformerFactory.Agones().V1().GameServers(), gameservers.NewPerNodeCounter(m.KubeInformerFactory, m.AgonesInformerFactory), healthcheck.NewHandler())
	return cache, m
}
