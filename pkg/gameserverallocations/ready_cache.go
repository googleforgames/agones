// Copyright 2019 Google LLC All Rights Reserved.
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
	"sort"

	"agones.dev/agones/pkg/apis/agones"
	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	allocationv1 "agones.dev/agones/pkg/apis/allocation/v1"
	getterv1 "agones.dev/agones/pkg/client/clientset/versioned/typed/agones/v1"
	informerv1 "agones.dev/agones/pkg/client/informers/externalversions/agones/v1"
	listerv1 "agones.dev/agones/pkg/client/listers/agones/v1"
	"agones.dev/agones/pkg/gameservers"
	"agones.dev/agones/pkg/util/logfields"
	"agones.dev/agones/pkg/util/runtime"
	"agones.dev/agones/pkg/util/workerqueue"
	"github.com/heptiolabs/healthcheck"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
)

// ReadyGameServerCache handles the gameserver sync operations for cache
type ReadyGameServerCache struct {
	baseLogger       *logrus.Entry
	readyGameServers gameServerCacheEntry
	gameServerGetter getterv1.GameServersGetter
	gameServerLister listerv1.GameServerLister
	gameServerSynced cache.InformerSynced
	workerqueue      *workerqueue.WorkerQueue
	counter          *gameservers.PerNodeCounter
}

// NewReadyGameServerCache creates a new instance of ReadyGameServerCache
func NewReadyGameServerCache(informer informerv1.GameServerInformer, gameServerGetter getterv1.GameServersGetter, counter *gameservers.PerNodeCounter, health healthcheck.Handler) *ReadyGameServerCache {
	c := &ReadyGameServerCache{
		gameServerSynced: informer.Informer().HasSynced,
		gameServerGetter: gameServerGetter,
		gameServerLister: informer.Lister(),
		counter:          counter,
	}

	informer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		UpdateFunc: func(oldObj, newObj interface{}) {
			// only interested in if the old / new state was/is Ready
			oldGs := oldObj.(*agonesv1.GameServer)
			newGs := newObj.(*agonesv1.GameServer)
			key, ok := c.getKey(newGs)
			if !ok {
				return
			}
			if newGs.IsBeingDeleted() {
				c.readyGameServers.Delete(key)
			} else if oldGs.Status.State == agonesv1.GameServerStateReady || newGs.Status.State == agonesv1.GameServerStateReady {
				if newGs.Status.State == agonesv1.GameServerStateReady {
					c.readyGameServers.Store(key, newGs)
				} else {
					c.readyGameServers.Delete(key)
				}
			}
		},
		DeleteFunc: func(obj interface{}) {
			gs, ok := obj.(*agonesv1.GameServer)
			if !ok {
				return
			}
			var key string
			if key, ok = c.getKey(gs); ok {
				c.readyGameServers.Delete(key)
			}
		},
	})

	c.baseLogger = runtime.NewLoggerWithType(c)
	c.workerqueue = workerqueue.NewWorkerQueue(c.SyncGameServers, c.baseLogger, logfields.GameServerKey, agones.GroupName+".GameServerUpdateController")
	health.AddLivenessCheck("gameserverallocation-gameserver-workerqueue", healthcheck.Check(c.workerqueue.Healthy))

	return c
}

func (c *ReadyGameServerCache) loggerForGameServerKey(key string) *logrus.Entry {
	return logfields.AugmentLogEntry(c.baseLogger, logfields.GameServerKey, key)
}

// RemoveFromReadyGameServer removes a gameserver from the list of ready game server list
func (c *ReadyGameServerCache) RemoveFromReadyGameServer(gs *agonesv1.GameServer) error {
	key, _ := cache.MetaNamespaceKeyFunc(gs)
	if ok := c.readyGameServers.Delete(key); !ok {
		return ErrConflictInGameServerSelection
	}
	return nil
}

// Sync waits for cache to sync
func (c *ReadyGameServerCache) Sync(stop <-chan struct{}) error {
	c.baseLogger.Debug("Wait for ReadyGameServerCache cache sync")
	if !cache.WaitForCacheSync(stop, c.gameServerSynced) {
		return errors.New("failed to wait for caches to sync")
	}

	// build the cache
	return c.syncReadyGSServerCache()
}

// Resync enqueues an empty game server to be synced. Using queue helps avoiding multiple threads syncing at the same time.
func (c *ReadyGameServerCache) Resync() {
	// this will trigger syncing of the cache (assuming cache might not be up to date)
	c.workerqueue.EnqueueImmediately(&agonesv1.GameServer{})
}

// Start prepares cache to start
func (c *ReadyGameServerCache) Start(stop <-chan struct{}) error {
	if err := c.Sync(stop); err != nil {
		return err
	}
	// we don't want mutiple workers refresh cache at the same time so one worker will be better.
	// Also we don't expect to have too many failures when allocating
	go c.workerqueue.Run(1, stop)
	return nil
}

// AddToReadyGameServer adds a gameserver to the list of ready game server list
func (c *ReadyGameServerCache) AddToReadyGameServer(gs *agonesv1.GameServer) {
	key, _ := cache.MetaNamespaceKeyFunc(gs)

	c.readyGameServers.Store(key, gs)
}

// getReadyGameServers returns a list of ready game servers
func (c *ReadyGameServerCache) getReadyGameServers() []*agonesv1.GameServer {
	length := c.readyGameServers.Len()
	if length == 0 {
		return nil
	}

	list := make([]*agonesv1.GameServer, 0, length)
	c.readyGameServers.Range(func(_ string, gs *agonesv1.GameServer) bool {
		list = append(list, gs)
		return true
	})
	return list
}

// ListSortedReadyGameServers returns a list of the cache ready gameservers
// sorted by most allocated to least
func (c *ReadyGameServerCache) ListSortedReadyGameServers() []*agonesv1.GameServer {
	list := c.getReadyGameServers()
	if list == nil {
		return []*agonesv1.GameServer{}
	}
	counts := c.counter.Counts()

	sort.Slice(list, func(i, j int) bool {
		gs1 := list[i]
		gs2 := list[j]

		c1, ok := counts[gs1.Status.NodeName]
		if !ok {
			return false
		}

		c2, ok := counts[gs2.Status.NodeName]
		if !ok {
			return true
		}

		if c1.Allocated > c2.Allocated {
			return true
		}
		if c1.Allocated < c2.Allocated {
			return false
		}

		// prefer nodes that have the most Ready gameservers on them - they are most likely to be
		// completely filled and least likely target for scale down.
		if c1.Ready < c2.Ready {
			return false
		}
		if c1.Ready > c2.Ready {
			return true
		}

		// finally sort lexicographically, so we have a stable order
		return gs1.Status.NodeName < gs2.Status.NodeName
	})

	return list
}

// PatchGameServerMetadata patches the input gameserver with allocation meta patch and returns the updated gameserver
func (c *ReadyGameServerCache) PatchGameServerMetadata(fam allocationv1.MetaPatch, gs *agonesv1.GameServer) (*agonesv1.GameServer, error) {
	c.patchMetadata(gs, fam)
	gs.Status.State = agonesv1.GameServerStateAllocated

	return c.gameServerGetter.GameServers(gs.ObjectMeta.Namespace).Update(gs)
}

// patch the labels and annotations of an allocated GameServer with metadata from a GameServerAllocation
func (c *ReadyGameServerCache) patchMetadata(gs *agonesv1.GameServer, fam allocationv1.MetaPatch) {
	// patch ObjectMeta labels
	if fam.Labels != nil {
		if gs.ObjectMeta.Labels == nil {
			gs.ObjectMeta.Labels = make(map[string]string, len(fam.Labels))
		}
		for key, value := range fam.Labels {
			gs.ObjectMeta.Labels[key] = value
		}
	}
	// apply annotations patch
	if fam.Annotations != nil {
		if gs.ObjectMeta.Annotations == nil {
			gs.ObjectMeta.Annotations = make(map[string]string, len(fam.Annotations))
		}
		for key, value := range fam.Annotations {
			gs.ObjectMeta.Annotations[key] = value
		}
	}
}

// SyncGameServers synchronises the GameServers to Gameserver cache. This is called when a failure
// happened during the allocation. This method will sync and make sure the cache is up to date.
func (c *ReadyGameServerCache) SyncGameServers(key string) error {
	c.loggerForGameServerKey(key).Debug("Refreshing Ready Gameserver cache")

	return c.syncReadyGSServerCache()
}

// syncReadyGSServerCache syncs the gameserver cache and updates the local cache for any changes.
func (c *ReadyGameServerCache) syncReadyGSServerCache() error {
	// build the cache
	gsList, err := c.gameServerLister.List(labels.Everything())
	if err != nil {
		return errors.Wrap(err, "could not list GameServers")
	}

	// convert list of current gameservers to map for faster access
	currGameservers := make(map[string]*agonesv1.GameServer)
	for _, gs := range gsList {
		if key, ok := c.getKey(gs); ok {
			currGameservers[key] = gs
		}
	}

	// first remove the gameservers are not in the list anymore
	tobeDeletedGSInCache := make([]string, 0)
	c.readyGameServers.Range(func(key string, gs *agonesv1.GameServer) bool {
		if _, ok := currGameservers[key]; !ok {
			tobeDeletedGSInCache = append(tobeDeletedGSInCache, key)
		}
		return true
	})

	for _, staleGSKey := range tobeDeletedGSInCache {
		c.readyGameServers.Delete(staleGSKey)
	}

	// refresh the cache of possible allocatable GameServers
	for key, gs := range currGameservers {
		if gsCache, ok := c.readyGameServers.Load(key); ok {
			if !(gs.DeletionTimestamp.IsZero() && gs.Status.State == agonesv1.GameServerStateReady) {
				c.readyGameServers.Delete(key)
			} else if gs.ObjectMeta.ResourceVersion != gsCache.ObjectMeta.ResourceVersion {
				c.readyGameServers.Store(key, gs)
			}
		} else if gs.DeletionTimestamp.IsZero() && gs.Status.State == agonesv1.GameServerStateReady {
			c.readyGameServers.Store(key, gs)
		}
	}

	return nil
}

// getKey extract the key of gameserver object
func (c *ReadyGameServerCache) getKey(gs *agonesv1.GameServer) (string, bool) {
	var key string
	ok := true
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(gs); err != nil {
		ok = false
		err = errors.Wrap(err, "Error creating key for object")
		runtime.HandleError(c.baseLogger.WithField("obj", gs), err)
	}
	return key, ok
}
