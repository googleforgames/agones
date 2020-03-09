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

package gameservers

import (
	"sync"

	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	"agones.dev/agones/pkg/client/informers/externalversions"
	listerv1 "agones.dev/agones/pkg/client/listers/agones/v1"
	"agones.dev/agones/pkg/util/runtime"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/tools/cache"
)

// PerNodeCounter counts how many Allocated and
// Ready GameServers currently exist on each node.
// This is useful for scheduling allocations, fleet management
// mostly under a Packed strategy
type PerNodeCounter struct {
	logger           *logrus.Entry
	gameServerSynced cache.InformerSynced
	gameServerLister listerv1.GameServerLister
	countMutex       sync.RWMutex
	counts           map[string]*NodeCount
}

// NodeCount is just a convenience data structure for
// keeping relevant GameServer counts about Nodes
type NodeCount struct {
	// Ready is ready count
	Ready int64
	// Allocated is allocated out
	Allocated int64
}

// NewPerNodeCounter returns a new PerNodeCounter
func NewPerNodeCounter(
	kubeInformerFactory informers.SharedInformerFactory,
	agonesInformerFactory externalversions.SharedInformerFactory) *PerNodeCounter {

	gameServers := agonesInformerFactory.Agones().V1().GameServers()
	gsInformer := gameServers.Informer()

	ac := &PerNodeCounter{
		gameServerSynced: gsInformer.HasSynced,
		gameServerLister: gameServers.Lister(),
		countMutex:       sync.RWMutex{},
		counts:           map[string]*NodeCount{},
	}

	ac.logger = runtime.NewLoggerWithType(ac)

	gsInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			gs := obj.(*agonesv1.GameServer)

			switch gs.Status.State {
			case agonesv1.GameServerStateReady:
				ac.inc(gs, 1, 0)
			case agonesv1.GameServerStateAllocated:
				ac.inc(gs, 0, 1)
			}
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			oldGS := oldObj.(*agonesv1.GameServer)
			newGS := newObj.(*agonesv1.GameServer)

			var ready int64
			var allocated int64

			if oldGS.Status.State == agonesv1.GameServerStateReady && newGS.Status.State != agonesv1.GameServerStateReady {
				ready = -1
			} else if newGS.Status.State == agonesv1.GameServerStateReady && oldGS.Status.State != agonesv1.GameServerStateReady {
				ready = 1
			}

			if oldGS.Status.State == agonesv1.GameServerStateAllocated && newGS.Status.State != agonesv1.GameServerStateAllocated {
				allocated = -1
			} else if newGS.Status.State == agonesv1.GameServerStateAllocated && oldGS.Status.State != agonesv1.GameServerStateAllocated {
				allocated = 1
			}

			ac.inc(newGS, ready, allocated)
		},
		DeleteFunc: func(obj interface{}) {
			gs, ok := obj.(*agonesv1.GameServer)
			if !ok {
				return
			}

			switch gs.Status.State {
			case agonesv1.GameServerStateReady:
				ac.inc(gs, -1, 0)
			case agonesv1.GameServerStateAllocated:
				ac.inc(gs, 0, -1)
			}
		},
	})

	// remove the record when the node is deleted
	kubeInformerFactory.Core().V1().Nodes().Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		DeleteFunc: func(obj interface{}) {
			node, ok := obj.(*corev1.Node)
			if !ok {
				return
			}

			ac.countMutex.Lock()
			defer ac.countMutex.Unlock()

			delete(ac.counts, node.ObjectMeta.Name)
		},
	})

	return ac
}

// Run sets up the current state GameServer counts across nodes
// non blocking Run function.
func (pnc *PerNodeCounter) Run(_ int, stop <-chan struct{}) error {
	pnc.countMutex.Lock()
	defer pnc.countMutex.Unlock()

	pnc.logger.Debug("Running")

	if !cache.WaitForCacheSync(stop, pnc.gameServerSynced) {
		return errors.New("failed to wait for caches to sync")
	}

	gsList, err := pnc.gameServerLister.List(labels.Everything())
	if err != nil {
		return errors.Wrap(err, "error attempting to list all GameServers")
	}

	counts := map[string]*NodeCount{}
	for _, gs := range gsList {
		_, ok := counts[gs.Status.NodeName]
		if !ok {
			counts[gs.Status.NodeName] = &NodeCount{}
		}

		switch gs.Status.State {
		case agonesv1.GameServerStateReady:
			counts[gs.Status.NodeName].Ready++
		case agonesv1.GameServerStateAllocated:
			counts[gs.Status.NodeName].Allocated++
		}
	}

	pnc.counts = counts
	return nil
}

// Counts returns the NodeCount map in a thread safe way
func (pnc *PerNodeCounter) Counts() map[string]NodeCount {
	pnc.countMutex.RLock()
	defer pnc.countMutex.RUnlock()

	result := make(map[string]NodeCount, len(pnc.counts))

	// return a copy, so it's thread safe
	for k, v := range pnc.counts {
		result[k] = *v
	}

	return result
}

func (pnc *PerNodeCounter) inc(gs *agonesv1.GameServer, ready, allocated int64) {
	pnc.countMutex.Lock()
	defer pnc.countMutex.Unlock()

	_, ok := pnc.counts[gs.Status.NodeName]
	if !ok {
		pnc.counts[gs.Status.NodeName] = &NodeCount{}
	}

	pnc.counts[gs.Status.NodeName].Allocated += allocated
	pnc.counts[gs.Status.NodeName].Ready += ready

	// just in case
	if pnc.counts[gs.Status.NodeName].Allocated < 0 {
		pnc.counts[gs.Status.NodeName].Allocated = 0
	}

	if pnc.counts[gs.Status.NodeName].Ready < 0 {
		pnc.counts[gs.Status.NodeName].Ready = 0
	}
}
