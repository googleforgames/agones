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
	"testing"
	"time"

	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	"agones.dev/agones/pkg/gameservers"
	agtesting "agones.dev/agones/pkg/testing"
	utilruntime "agones.dev/agones/pkg/util/runtime"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8stesting "k8s.io/client-go/testing"
)

func TestSortGameServersByLeastFullNodes(t *testing.T) {
	t.Parallel()

	utilruntime.FeatureTestMutex.Lock()
	defer utilruntime.FeatureTestMutex.Unlock()

	require.NoError(t, utilruntime.ParseFeatures(string(utilruntime.FeatureCountsAndLists)+"=true"))

	nc := map[string]gameservers.NodeCount{
		"n1": {Ready: 1, Allocated: 0},
		"n2": {Ready: 0, Allocated: 2},
		"n3": {Ready: 2, Allocated: 2},
		"n4": {Ready: 2, Allocated: 2},
	}

	list := []*agonesv1.GameServer{
		{ObjectMeta: metav1.ObjectMeta{Name: "g1"}, Status: agonesv1.GameServerStatus{NodeName: "n2", State: agonesv1.GameServerStateReady}},
		{ObjectMeta: metav1.ObjectMeta{Name: "g2"}, Status: agonesv1.GameServerStatus{NodeName: "", State: agonesv1.GameServerStateReady}},
		{ObjectMeta: metav1.ObjectMeta{Name: "g3"}, Status: agonesv1.GameServerStatus{NodeName: "n1", State: agonesv1.GameServerStateReady}},
		{ObjectMeta: metav1.ObjectMeta{Name: "g4"}, Status: agonesv1.GameServerStatus{NodeName: "n2", State: agonesv1.GameServerStateCreating}},
		{ObjectMeta: metav1.ObjectMeta{Name: "g5"}, Status: agonesv1.GameServerStatus{NodeName: "n3", State: agonesv1.GameServerStateReady}},
		{ObjectMeta: metav1.ObjectMeta{Name: "g6"}, Status: agonesv1.GameServerStatus{NodeName: "n4", State: agonesv1.GameServerStateReady}},
		{ObjectMeta: metav1.ObjectMeta{Name: "g7"}, Status: agonesv1.GameServerStatus{NodeName: "n3", State: agonesv1.GameServerStateReady}},
		{ObjectMeta: metav1.ObjectMeta{Name: "g8"}, Status: agonesv1.GameServerStatus{NodeName: "n4", State: agonesv1.GameServerStateReady}},
		{ObjectMeta: metav1.ObjectMeta{Name: "g9"}, Status: agonesv1.GameServerStatus{
			NodeName: "n3",
			State:    agonesv1.GameServerStateAllocated,
			Counters: map[string]agonesv1.CounterStatus{
				"foo": {
					Count:    0,
					Capacity: 100,
				}}}},
		{ObjectMeta: metav1.ObjectMeta{Name: "g10"}, Status: agonesv1.GameServerStatus{
			NodeName: "n3",
			State:    agonesv1.GameServerStateAllocated,
			Counters: map[string]agonesv1.CounterStatus{
				"foo": {
					Count:    99,
					Capacity: 100,
				}}}},
		{ObjectMeta: metav1.ObjectMeta{Name: "g11"}, Status: agonesv1.GameServerStatus{
			NodeName: "n4",
			State:    agonesv1.GameServerStateAllocated,
			Counters: map[string]agonesv1.CounterStatus{
				"foo": {
					Count:    0,
					Capacity: 90,
				}}}},
		{ObjectMeta: metav1.ObjectMeta{Name: "g12"}, Status: agonesv1.GameServerStatus{
			NodeName: "n4",
			State:    agonesv1.GameServerStateAllocated,
			Counters: map[string]agonesv1.CounterStatus{
				"foo": {
					Count:    89,
					Capacity: 90,
				}}}},
	}

	priorities := []agonesv1.Priority{{
		PriorityType: "Counter",
		Key:          "foo",
		Order:        "Descending",
	}}

	result := sortGameServersByLeastFullNodes(list, nc, priorities)

	require.Len(t, result, len(list))
	assert.Equal(t, "g2", result[0].ObjectMeta.Name)
	assert.Equal(t, "g3", result[1].ObjectMeta.Name)
	assert.Equal(t, "g4", result[2].ObjectMeta.Name)
	assert.Equal(t, "g1", result[3].ObjectMeta.Name)
	assert.Equal(t, "g10", result[4].ObjectMeta.Name)
	assert.Equal(t, "g9", result[5].ObjectMeta.Name)
	assert.Equal(t, "g12", result[6].ObjectMeta.Name)
	assert.Equal(t, "g11", result[7].ObjectMeta.Name)
	assert.Equal(t, "g5", result[8].ObjectMeta.Name)
	assert.Equal(t, "g7", result[9].ObjectMeta.Name)
	assert.Equal(t, "g6", result[10].ObjectMeta.Name)
	assert.Equal(t, "g8", result[11].ObjectMeta.Name)

	priorities = []agonesv1.Priority{{
		PriorityType: "Counter",
		Key:          "foo",
		Order:        "Ascending",
	}}

	result = sortGameServersByLeastFullNodes(list, nc, priorities)

	require.Len(t, result, len(list))
	assert.Equal(t, "g2", result[0].ObjectMeta.Name)
	assert.Equal(t, "g3", result[1].ObjectMeta.Name)
	assert.Equal(t, "g4", result[2].ObjectMeta.Name)
	assert.Equal(t, "g1", result[3].ObjectMeta.Name)
	assert.Equal(t, "g11", result[4].ObjectMeta.Name)
	assert.Equal(t, "g12", result[5].ObjectMeta.Name)
	assert.Equal(t, "g9", result[6].ObjectMeta.Name)
	assert.Equal(t, "g10", result[7].ObjectMeta.Name)
	assert.Equal(t, "g5", result[8].ObjectMeta.Name)
	assert.Equal(t, "g7", result[9].ObjectMeta.Name)
	assert.Equal(t, "g6", result[10].ObjectMeta.Name)
	assert.Equal(t, "g8", result[11].ObjectMeta.Name)

	priorities = []agonesv1.Priority{{
		PriorityType: "Counter",
		Key:          "foo",
		Order:        "",
	}}
	// Result should be same as for Ascending
	result = sortGameServersByLeastFullNodes(list, nc, priorities)

	require.Len(t, result, len(list))
	assert.Equal(t, "g2", result[0].ObjectMeta.Name)
	assert.Equal(t, "g3", result[1].ObjectMeta.Name)
	assert.Equal(t, "g4", result[2].ObjectMeta.Name)
	assert.Equal(t, "g1", result[3].ObjectMeta.Name)
	assert.Equal(t, "g11", result[4].ObjectMeta.Name)
	assert.Equal(t, "g12", result[5].ObjectMeta.Name)
	assert.Equal(t, "g9", result[6].ObjectMeta.Name)
	assert.Equal(t, "g10", result[7].ObjectMeta.Name)
	assert.Equal(t, "g5", result[8].ObjectMeta.Name)
	assert.Equal(t, "g7", result[9].ObjectMeta.Name)
	assert.Equal(t, "g6", result[10].ObjectMeta.Name)
	assert.Equal(t, "g8", result[11].ObjectMeta.Name)
}

func TestSortGameServersByNewFirst(t *testing.T) {
	now := metav1.Now()

	list := []*agonesv1.GameServer{
		{ObjectMeta: metav1.ObjectMeta{Name: "g1", CreationTimestamp: metav1.Time{Time: now.Add(10 * time.Second)}}},
		{ObjectMeta: metav1.ObjectMeta{Name: "g2", CreationTimestamp: now}},
		{ObjectMeta: metav1.ObjectMeta{Name: "g3", CreationTimestamp: metav1.Time{Time: now.Add(30 * time.Second)}}},
	}
	l := len(list)

	result := sortGameServersByNewFirst(list)
	require.Len(t, result, l)
	assert.Equal(t, "g2", result[0].ObjectMeta.Name)
	assert.Equal(t, "g1", result[1].ObjectMeta.Name)
	assert.Equal(t, "g3", result[2].ObjectMeta.Name)
}

func TestListGameServersByGameServerSetOwner(t *testing.T) {
	t.Parallel()

	gsSet := &agonesv1.GameServerSet{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "test", UID: "1234"},
		Spec: agonesv1.GameServerSetSpec{
			Replicas: 10,
			Template: agonesv1.GameServerTemplateSpec{},
		},
	}

	gs1 := gsSet.GameServer()
	gs1.ObjectMeta.Name = "test-1"
	gs2 := gsSet.GameServer()
	assert.True(t, metav1.IsControlledBy(gs2, gsSet))

	gs2.ObjectMeta.Name = "test-2"
	gs3 := agonesv1.GameServer{ObjectMeta: metav1.ObjectMeta{Name: "not-included"}}
	gs4 := gsSet.GameServer()
	gs4.ObjectMeta.OwnerReferences = nil

	m := agtesting.NewMocks()
	m.AgonesClient.AddReactor("list", "gameservers", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, &agonesv1.GameServerList{Items: []agonesv1.GameServer{*gs1, *gs2, gs3, *gs4}}, nil
	})

	gameServers := m.AgonesInformerFactory.Agones().V1().GameServers()
	_, cancel := agtesting.StartInformers(m, gameServers.Informer().HasSynced)
	defer cancel()

	list, err := ListGameServersByGameServerSetOwner(gameServers.Lister(), gsSet)
	require.NoError(t, err)

	// sort of stable ordering
	sort.SliceStable(list, func(i, j int) bool {
		return list[i].ObjectMeta.Name < list[j].ObjectMeta.Name
	})
	assert.Equal(t, []*agonesv1.GameServer{gs1, gs2}, list)
}
