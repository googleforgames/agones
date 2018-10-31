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

package gameserversets

import (
	"sort"

	"agones.dev/agones/pkg/apis/stable/v1alpha1"
	listerv1alpha1 "agones.dev/agones/pkg/client/listers/stable/v1alpha1"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

// node is just a convenience data structure for
// keeping relevant GameServer information about Nodes
type node struct {
	name  string
	total int64
	ready []*v1alpha1.GameServer
}

// filterGameServersOnLeastFullNodes returns a limited list of GameServers, ordered by the nodes
// they are hosted on, with the least utilised Nodes being prioritised
func filterGameServersOnLeastFullNodes(gsList []*v1alpha1.GameServer, limit int32) []*v1alpha1.GameServer {
	if limit <= 0 {
		return nil
	}

	nodeMap := map[string]*node{}
	var nodeList []*node

	// count up the number of allocated and ready game servers that exist
	// also, since we're already looping through, track all the deletable GameServers
	// per node, so we can use this as a shortlist to delete from
	for _, gs := range gsList {
		if gs.DeletionTimestamp.IsZero() &&
			(gs.Status.State == v1alpha1.Allocated || gs.Status.State == v1alpha1.Ready) {
			_, ok := nodeMap[gs.Status.NodeName]
			if !ok {
				node := &node{name: gs.Status.NodeName}
				nodeMap[gs.Status.NodeName] = node
				nodeList = append(nodeList, node)
			}

			nodeMap[gs.Status.NodeName].total++
			if gs.Status.State == v1alpha1.Ready {
				nodeMap[gs.Status.NodeName].ready = append(nodeMap[gs.Status.NodeName].ready, gs)
			}
		}
	}

	// sort our nodes, least to most
	sort.Slice(nodeList, func(i, j int) bool {
		return nodeList[i].total < nodeList[j].total
	})

	// we need to get Ready GameServer until we equal or pass limit
	result := make([]*v1alpha1.GameServer, 0, limit)

	for _, n := range nodeList {
		result = append(result, n.ready...)

		if int32(len(result)) >= limit {
			return result
		}
	}

	return result
}

// ListGameServersByGameServerSetOwner lists the GameServers for a given GameServerSet
func ListGameServersByGameServerSetOwner(gameServerLister listerv1alpha1.GameServerLister,
	gsSet *v1alpha1.GameServerSet) ([]*v1alpha1.GameServer, error) {
	list, err := gameServerLister.List(labels.SelectorFromSet(labels.Set{v1alpha1.GameServerSetGameServerLabel: gsSet.ObjectMeta.Name}))
	if err != nil {
		return list, errors.Wrapf(err, "error listing gameservers for gameserverset %s", gsSet.ObjectMeta.Name)
	}

	var result []*v1alpha1.GameServer
	for _, gs := range list {
		if metav1.IsControlledBy(gs, gsSet) {
			result = append(result, gs)
		}
	}

	return result, nil
}
