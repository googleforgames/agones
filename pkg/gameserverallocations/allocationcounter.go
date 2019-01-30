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
	"sync"

	"agones.dev/agones/pkg/apis/stable/v1alpha1"
	"agones.dev/agones/pkg/client/informers/externalversions"
	listerv1alpha1 "agones.dev/agones/pkg/client/listers/stable/v1alpha1"
	"agones.dev/agones/pkg/util/runtime"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/tools/cache"
)

// AllocationCounter counts how many Allocated and
// Ready GameServers currently exist on each node.
// This is useful for scheduling allocations on the
// right Nodes.
type AllocationCounter struct {
	logger           *logrus.Entry
	gameServerSynced cache.InformerSynced
	gameServerLister listerv1alpha1.GameServerLister
	countMutex       sync.RWMutex
	counts           map[string]*NodeCount
}

// NodeCount is just a convenience data structure for
// keeping relevant GameServer counts about Nodes
type NodeCount struct {
	ready     int64
	allocated int64
}

// NewAllocationCounter returns a new AllocationCounter
func NewAllocationCounter(
	kubeInformerFactory informers.SharedInformerFactory,
	agonesInformerFactory externalversions.SharedInformerFactory) *AllocationCounter {

	gameServers := agonesInformerFactory.Stable().V1alpha1().GameServers()
	gsInformer := gameServers.Informer()

	ac := &AllocationCounter{
		gameServerSynced: gsInformer.HasSynced,
		gameServerLister: gameServers.Lister(),
		countMutex:       sync.RWMutex{},
		counts:           map[string]*NodeCount{},
	}

	ac.logger = runtime.NewLoggerWithType(ac)

	gsInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			gs := obj.(*v1alpha1.GameServer)

			switch gs.Status.State {
			case v1alpha1.GameServerStateReady:
				ac.inc(gs, 1, 0)
			case v1alpha1.GameServerStateAllocated:
				ac.inc(gs, 0, 1)
			}
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			oldGS := oldObj.(*v1alpha1.GameServer)
			newGS := newObj.(*v1alpha1.GameServer)

			var ready int64
			var allocated int64

			if oldGS.Status.State == v1alpha1.GameServerStateReady && newGS.Status.State != v1alpha1.GameServerStateReady {
				ready = -1
			} else if newGS.Status.State == v1alpha1.GameServerStateReady && oldGS.Status.State != v1alpha1.GameServerStateReady {
				ready = 1
			}

			if oldGS.Status.State == v1alpha1.GameServerStateAllocated && newGS.Status.State != v1alpha1.GameServerStateAllocated {
				allocated = -1
			} else if newGS.Status.State == v1alpha1.GameServerStateAllocated && oldGS.Status.State != v1alpha1.GameServerStateAllocated {
				allocated = 1
			}

			ac.inc(newGS, ready, allocated)
		},
		DeleteFunc: func(obj interface{}) {
			gs, ok := obj.(*v1alpha1.GameServer)
			if !ok {
				return
			}

			switch gs.Status.State {
			case v1alpha1.GameServerStateReady:
				ac.inc(gs, -1, 0)
			case v1alpha1.GameServerStateAllocated:
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
func (ac *AllocationCounter) Run(stop <-chan struct{}) error {
	ac.countMutex.Lock()
	defer ac.countMutex.Unlock()

	ac.logger.Info("Running")

	if !cache.WaitForCacheSync(stop, ac.gameServerSynced) {
		return errors.New("failed to wait for caches to sync")
	}

	gsList, err := ac.gameServerLister.List(labels.Everything())
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
		case v1alpha1.GameServerStateReady:
			counts[gs.Status.NodeName].ready++
		case v1alpha1.GameServerStateAllocated:
			counts[gs.Status.NodeName].allocated++
		}
	}

	ac.counts = counts
	return nil
}

// Counts returns the NodeCount map in a thread safe way
func (ac *AllocationCounter) Counts() map[string]NodeCount {
	ac.countMutex.RLock()
	defer ac.countMutex.RUnlock()

	result := make(map[string]NodeCount, len(ac.counts))

	// return a copy, so it's thread safe
	for k, v := range ac.counts {
		result[k] = *v
	}

	return result
}

func (ac *AllocationCounter) inc(gs *v1alpha1.GameServer, ready, allocated int64) {
	ac.countMutex.Lock()
	defer ac.countMutex.Unlock()

	_, ok := ac.counts[gs.Status.NodeName]
	if !ok {
		ac.counts[gs.Status.NodeName] = &NodeCount{}
	}

	ac.counts[gs.Status.NodeName].allocated += allocated
	ac.counts[gs.Status.NodeName].ready += ready

	// just in case
	if ac.counts[gs.Status.NodeName].allocated < 0 {
		ac.counts[gs.Status.NodeName].allocated = 0
	}

	if ac.counts[gs.Status.NodeName].ready < 0 {
		ac.counts[gs.Status.NodeName].ready = 0
	}
}
