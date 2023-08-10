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

	"agones.dev/agones/pkg/apis"
	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	listerv1 "agones.dev/agones/pkg/client/listers/agones/v1"
	"agones.dev/agones/pkg/gameservers"
	"agones.dev/agones/pkg/util/logfields"
	"agones.dev/agones/pkg/util/runtime"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

func loggerForGameServerSetKey(log *logrus.Entry, key string) *logrus.Entry {
	return logfields.AugmentLogEntry(log, logfields.GameServerSetKey, key)
}

func loggerForGameServerSet(log *logrus.Entry, gsSet *agonesv1.GameServerSet) *logrus.Entry {
	gsSetName := "NilGameServerSet"
	if gsSet != nil {
		gsSetName = gsSet.Namespace + "/" + gsSet.Name
	}
	return loggerForGameServerSetKey(log, gsSetName).WithField("gss", gsSet)
}

// SortGameServersByStrategy will sort by least full nodes when Packed, and newest first when Distributed
func SortGameServersByStrategy(strategy apis.SchedulingStrategy, list []*agonesv1.GameServer, counts map[string]gameservers.NodeCount, priorities []agonesv1.Priority) []*agonesv1.GameServer {
	if strategy == apis.Packed {
		return sortGameServersByPackedStrategy(list, counts, priorities)
	}
	return sortGameServersByDistributedStrategy(list, priorities)
}

// sortGameServersByPackedStrategy sorts the list of gameservers by which gameservers reside on the least full nodes
// Performs a tie-breaking sort if nodes are equally full on CountsAndLists Priorities.
func sortGameServersByPackedStrategy(list []*agonesv1.GameServer, count map[string]gameservers.NodeCount, priorities []agonesv1.Priority) []*agonesv1.GameServer {
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

		if a.Status.NodeName == b.Status.NodeName {
			if a.IsBeforeReady() && b.Status.State == agonesv1.GameServerStateReady {
				return true
			}

			if b.IsBeforeReady() && a.Status.State == agonesv1.GameServerStateReady {
				return false
			}
		}

		// Check Node total count
		acTotal, bcTotal := ac.Allocated+ac.Ready, bc.Allocated+bc.Ready
		if acTotal < bcTotal {
			return true
		}
		if acTotal > bcTotal {
			return false
		}

		// if the Nodes have the same count and CountsAndLists flag is enabled, sort based on CountsAndLists Priorities.
		if runtime.FeatureEnabled(runtime.FeatureCountsAndLists) {
			for _, priority := range priorities {
				res := compareGameServerCapacity(&priority, a, b)
				switch priority.Order {
				case agonesv1.GameServerPriorityAscending:
					if res == -1 {
						return true
					}
					if res == 1 {
						return false
					}
				case agonesv1.GameServerPriorityDescending:
					if res == -1 {
						return false
					}
					if res == 1 {
						return true
					}
				}
			}
		}

		// if the gameservers are on the same node, then sort lexigraphically for a stable sort
		if a.Status.NodeName == b.Status.NodeName {
			return a.GetObjectMeta().GetName() < b.GetObjectMeta().GetName()
		}
		// if both Nodes have the same count, one node is emptied first (packed scheduling behavior)
		return a.Status.NodeName < b.Status.NodeName
	})

	return list
}

// sortGameServersByDistributedStrategy sorts by newest gameservers first.
// If FeatureCountsAndLists is enabled, sort by Priority first, then tie break with newest gameservers.
func sortGameServersByDistributedStrategy(list []*agonesv1.GameServer, priorities []agonesv1.Priority) []*agonesv1.GameServer {
	sort.Slice(list, func(i, j int) bool {
		a := list[i]
		b := list[j]

		if runtime.FeatureEnabled(runtime.FeatureCountsAndLists) {
			for _, priority := range priorities {
				res := compareGameServerCapacity(&priority, a, b)
				switch priority.Order {
				case agonesv1.GameServerPriorityAscending:
					if res == -1 {
						return true
					}
					if res == 1 {
						return false
					}
				case agonesv1.GameServerPriorityDescending:
					if res == -1 {
						return false
					}
					if res == 1 {
						return true
					}
				}
			}
		}

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

// compareGameServerCapacity compares two game servers based on a CountsAndLists Priority. First
// compares by Capacity, and then compares by Count or length of the List Values if Capacity is equal.
// NOTE: This does NOT sort by available capacity.
// Returns -1 if gs1 < gs2; 1 if gs1 > gs2; 0 if gs1 == gs2; 0 if neither gamer server has the Priority.
// If only one game server has the Priority, prefer that server. I.e. nil < gsX when Priority
// Order is Descending (3, 2, 1, 0, nil), and nil > gsX when Order is Ascending (0, 1, 2, 3, nil).
func compareGameServerCapacity(p *agonesv1.Priority, gs1, gs2 *agonesv1.GameServer) int {
	var gs1ok, gs2ok bool
	switch p.Type {
	case agonesv1.GameServerPriorityCounter:
		// Check if both game servers contain the Counter.
		counter1, ok1 := gs1.Status.Counters[p.Key]
		counter2, ok2 := gs2.Status.Counters[p.Key]
		// If both game servers have the Counter
		if ok1 && ok2 {
			if counter1.Capacity < counter2.Capacity {
				return -1
			}
			if counter1.Capacity > counter2.Capacity {
				return 1
			}
			if counter1.Capacity == counter2.Capacity {
				if counter1.Count < counter2.Count {
					return -1
				}
				if counter1.Count > counter2.Count {
					return 1
				}
				return 0
			}
		}
		gs1ok = ok1
		gs2ok = ok2
	case agonesv1.GameServerPriorityList:
		// Check if both game servers contain the List.
		list1, ok1 := gs1.Status.Lists[p.Key]
		list2, ok2 := gs2.Status.Lists[p.Key]
		// If both game servers have the List
		if ok1 && ok2 {
			if list1.Capacity < list2.Capacity {
				return -1
			}
			if list1.Capacity > list2.Capacity {
				return 1
			}
			if list1.Capacity == list2.Capacity {
				if len(list1.Values) < len(list2.Values) {
					return -1
				}
				if len(list1.Values) > len(list2.Values) {
					return 1
				}
				return 0
			}
		}
		gs1ok = ok1
		gs2ok = ok2
	}
	// If only one game server has the Priority, prefer that server. I.e. nil < gsX when Order is
	// Descending (3, 2, 1, 0, nil), and nil > gsX when Order is Ascending (0, 1, 2, 3, nil).
	// No Order specified "" is the same as Ascending.
	if (gs1ok && p.Order == agonesv1.GameServerPriorityDescending) ||
		(gs2ok && p.Order == agonesv1.GameServerPriorityAscending) {
		return 1
	}
	if (gs1ok && p.Order == agonesv1.GameServerPriorityAscending) ||
		(gs2ok && p.Order == agonesv1.GameServerPriorityDescending) {
		return -1
	}
	// If neither game server has the Priority
	return 0
}
