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

package gameserversets

import (
	"sort"

	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	listerv1 "agones.dev/agones/pkg/client/listers/agones/v1"
	"agones.dev/agones/pkg/gameservers"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

// sortGameServersByLeastFullNodes sorts the list of gameservers by which gameservers reside on the least full nodes
func sortGameServersByLeastFullNodes(list []*agonesv1.GameServer, count map[string]gameservers.NodeCount) []*agonesv1.GameServer {
	sort.Slice(list, func(i, j int) bool {
		a := list[i]
		b := list[j]

		// not scheduled yet/node deleted, put them first
		ac, ok := count[a.Status.NodeName]
		if !ok {
			return true
		}

		bc, ok := count[b.Status.NodeName]
		if !ok {
			return false
		}

		// if both are in the same node, make sure to delete pre-Ready GameServers first
		if a.Status.NodeName == b.Status.NodeName {
			if a.IsBeforeReady() && b.Status.State == agonesv1.GameServerStateReady {
				return true
			}

			if b.IsBeforeReady() && a.Status.State == agonesv1.GameServerStateReady {
				return false
			}
		}

		return (ac.Allocated + ac.Ready) < (bc.Allocated + bc.Ready)
	})

	return list
}

// sortGameServersByNewFirst sorts by newest gameservers first, and returns them
func sortGameServersByNewFirst(list []*agonesv1.GameServer) []*agonesv1.GameServer {
	sort.Slice(list, func(i, j int) bool {
		a := list[i]
		b := list[j]

		return a.ObjectMeta.CreationTimestamp.Before(&b.ObjectMeta.CreationTimestamp)
	})

	return list
}

// ListGameServersByGameServerSetOwner lists the GameServers for a given GameServerSet
func ListGameServersByGameServerSetOwner(gameServerLister listerv1.GameServerLister,
	gsSet *agonesv1.GameServerSet) ([]*agonesv1.GameServer, error) {
	list, err := gameServerLister.List(labels.SelectorFromSet(labels.Set{agonesv1.GameServerSetGameServerLabel: gsSet.ObjectMeta.Name}))
	if err != nil {
		return list, errors.Wrapf(err, "error listing gameservers for gameserverset %s", gsSet.ObjectMeta.Name)
	}

	var result []*agonesv1.GameServer
	for _, gs := range list {
		if metav1.IsControlledBy(gs, gsSet) {
			result = append(result, gs)
		}
	}

	return result, nil
}
