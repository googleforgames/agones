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

package v1

import (
	"fmt"
	"testing"

	"agones.dev/agones/pkg/apis"
	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	"agones.dev/agones/pkg/util/runtime"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGameServerAllocationApplyDefaults(t *testing.T) {
	t.Parallel()

	gsa := &GameServerAllocation{}
	gsa.ApplyDefaults()

	assert.Equal(t, apis.Packed, gsa.Spec.Scheduling)

	gsa = &GameServerAllocation{Spec: GameServerAllocationSpec{Scheduling: apis.Distributed}}
	gsa.ApplyDefaults()
	assert.Equal(t, apis.Distributed, gsa.Spec.Scheduling)

	runtime.FeatureTestMutex.Lock()
	defer runtime.FeatureTestMutex.Unlock()
	assert.NoError(t, runtime.ParseFeatures(fmt.Sprintf("%s=true&%s=true", runtime.FeaturePlayerAllocationFilter, runtime.FeatureCountsAndLists)))

	gsa = &GameServerAllocation{}
	gsa.ApplyDefaults()

	assert.Equal(t, agonesv1.GameServerStateReady, *gsa.Spec.Required.GameServerState)
	assert.Equal(t, int64(0), gsa.Spec.Required.Players.MaxAvailable)
	assert.Equal(t, int64(0), gsa.Spec.Required.Players.MinAvailable)
	assert.Equal(t, []Priority(nil), gsa.Spec.Priorities)
	assert.Nil(t, gsa.Spec.Priorities)
}

// nolint // Current lint duplicate threshold will consider this function is a duplication of the function TestGameServerAllocationSpecSelectors
func TestGameServerAllocationSpecPreferredSelectors(t *testing.T) {
	t.Parallel()

	gsas := &GameServerAllocationSpec{
		Preferred: []GameServerSelector{
			{LabelSelector: metav1.LabelSelector{MatchLabels: map[string]string{"check": "blue"}}},
			{LabelSelector: metav1.LabelSelector{MatchLabels: map[string]string{"check": "red"}}},
		},
	}

	require.Len(t, gsas.Preferred, 2)

	gs := &agonesv1.GameServer{ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{}}}

	for _, s := range gsas.Preferred {
		assert.False(t, s.Matches(gs))
	}

	gs.ObjectMeta.Labels["check"] = "blue"
	assert.True(t, gsas.Preferred[0].Matches(gs))
	assert.False(t, gsas.Preferred[1].Matches(gs))

	gs.ObjectMeta.Labels["check"] = "red"
	assert.False(t, gsas.Preferred[0].Matches(gs))
	assert.True(t, gsas.Preferred[1].Matches(gs))
}

// nolint // Current lint duplicate threshold will consider this function is a duplication of the function TestGameServerAllocationSpecPreferredSelectors
func TestGameServerAllocationSpecSelectors(t *testing.T) {
	t.Parallel()

	gsas := &GameServerAllocationSpec{
		Selectors: []GameServerSelector{
			{LabelSelector: metav1.LabelSelector{MatchLabels: map[string]string{"check": "blue"}}},
			{LabelSelector: metav1.LabelSelector{MatchLabels: map[string]string{"check": "red"}}},
		},
	}

	require.Len(t, gsas.Selectors, 2)

	gs := &agonesv1.GameServer{ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{}}}

	for _, s := range gsas.Selectors {
		assert.False(t, s.Matches(gs))
	}

	gs.ObjectMeta.Labels["check"] = "blue"
	assert.True(t, gsas.Selectors[0].Matches(gs))
	assert.False(t, gsas.Selectors[1].Matches(gs))

	gs.ObjectMeta.Labels["check"] = "red"
	assert.False(t, gsas.Selectors[0].Matches(gs))
	assert.True(t, gsas.Selectors[1].Matches(gs))
}

func TestGameServerSelectorApplyDefaults(t *testing.T) {
	t.Parallel()
	runtime.FeatureTestMutex.Lock()
	defer runtime.FeatureTestMutex.Unlock()

	assert.NoError(t, runtime.ParseFeatures(fmt.Sprintf("%s=true&%s=true",
		runtime.FeaturePlayerAllocationFilter,
		runtime.FeatureCountsAndLists)))

	s := &GameServerSelector{}

	// no defaults
	s.ApplyDefaults()
	assert.Equal(t, agonesv1.GameServerStateReady, *s.GameServerState)
	assert.Equal(t, int64(0), s.Players.MinAvailable)
	assert.Equal(t, int64(0), s.Players.MaxAvailable)
	assert.NotNil(t, s.Counters)
	assert.NotNil(t, s.Lists)

	state := agonesv1.GameServerStateAllocated
	// set values
	s = &GameServerSelector{
		GameServerState: &state,
		Players:         &PlayerSelector{MinAvailable: 10, MaxAvailable: 20},
		Counters:        map[string]CounterSelector{"foo": {MinAvailable: 1, MaxAvailable: 10}},
		Lists:           map[string]ListSelector{"bar": {MinAvailable: 2}},
	}
	s.ApplyDefaults()
	assert.Equal(t, state, *s.GameServerState)
	assert.Equal(t, int64(10), s.Players.MinAvailable)
	assert.Equal(t, int64(20), s.Players.MaxAvailable)
	assert.Equal(t, int64(0), s.Counters["foo"].MinCount)
	assert.Equal(t, int64(0), s.Counters["foo"].MaxCount)
	assert.Equal(t, int64(1), s.Counters["foo"].MinAvailable)
	assert.Equal(t, int64(10), s.Counters["foo"].MaxAvailable)
	assert.Equal(t, int64(2), s.Lists["bar"].MinAvailable)
	assert.Equal(t, int64(0), s.Lists["bar"].MaxAvailable)
	assert.Equal(t, "", s.Lists["bar"].ContainsValue)
}

func TestGameServerSelectorValidate(t *testing.T) {
	t.Parallel()

	runtime.FeatureTestMutex.Lock()
	defer runtime.FeatureTestMutex.Unlock()

	assert.NoError(t, runtime.ParseFeatures(fmt.Sprintf("%s=true&%s=true", runtime.FeaturePlayerAllocationFilter, runtime.FeatureCountsAndLists)))

	type expected struct {
		valid    bool
		causeLen int
		fields   []string
	}

	allocated := agonesv1.GameServerStateAllocated
	starting := agonesv1.GameServerStateStarting

	fixtures := map[string]struct {
		selector *GameServerSelector
		expected expected
	}{
		"valid": {
			selector: &GameServerSelector{GameServerState: &allocated, Players: &PlayerSelector{
				MinAvailable: 0,
				MaxAvailable: 10,
			}},
			expected: expected{
				valid:    true,
				causeLen: 0,
			},
		},
		"nil values": {
			selector: &GameServerSelector{},
			expected: expected{
				valid:    true,
				causeLen: 0,
			},
		},
		"invalid state": {
			selector: &GameServerSelector{
				GameServerState: &starting,
			},
			expected: expected{
				valid:    false,
				causeLen: 1,
				fields:   []string{"fieldName"},
			},
		},
		"invalid min value": {
			selector: &GameServerSelector{
				Players: &PlayerSelector{
					MinAvailable: -10,
				},
			},
			expected: expected{
				valid:    false,
				causeLen: 1,
				fields:   []string{"fieldName"},
			},
		},
		"invalid max value": {
			selector: &GameServerSelector{
				Players: &PlayerSelector{
					MinAvailable: -30,
					MaxAvailable: -20,
				},
			},
			expected: expected{
				valid:    false,
				causeLen: 2,
				fields:   []string{"fieldName", "fieldName"},
			},
		},
		"invalid min/max value": {
			selector: &GameServerSelector{
				Players: &PlayerSelector{
					MinAvailable: 10,
					MaxAvailable: 5,
				},
			},
			expected: expected{
				valid:    false,
				causeLen: 1,
				fields:   []string{"fieldName"},
			},
		},
		"invalid label keys": {
			selector: &GameServerSelector{
				LabelSelector: metav1.LabelSelector{
					MatchLabels: map[string]string{"$$$$": "true"},
				},
			},
			expected: expected{
				valid:    false,
				causeLen: 1,
				fields:   []string{"fieldName"},
			},
		},
		"invalid min/max Counter available value": {
			selector: &GameServerSelector{
				Counters: map[string]CounterSelector{
					"counter": {
						MinAvailable: -1,
						MaxAvailable: -1,
					},
				},
			},
			expected: expected{
				valid:    false,
				causeLen: 2,
				fields:   []string{"fieldName", "fieldName"},
			},
		},
		"invalid max less than min Counter available value": {
			selector: &GameServerSelector{
				Counters: map[string]CounterSelector{
					"foo": {
						MinAvailable: 10,
						MaxAvailable: 1,
					},
				},
			},
			expected: expected{
				valid:    false,
				causeLen: 1,
				fields:   []string{"fieldName"},
			},
		},
		"invalid min/max Counter count value": {
			selector: &GameServerSelector{
				Counters: map[string]CounterSelector{
					"counter": {
						MinCount: -1,
						MaxCount: -1,
					},
				},
			},
			expected: expected{
				valid:    false,
				causeLen: 2,
				fields:   []string{"fieldName", "fieldName"},
			},
		},
		"invalid max less than min Counter count value": {
			selector: &GameServerSelector{
				Counters: map[string]CounterSelector{
					"foo": {
						MinCount: 10,
						MaxCount: 1,
					},
				},
			},
			expected: expected{
				valid:    false,
				causeLen: 1,
				fields:   []string{"fieldName"},
			},
		},
		"invalid min/max List value": {
			selector: &GameServerSelector{
				Lists: map[string]ListSelector{
					"list": {
						MinAvailable: -11,
						MaxAvailable: -11,
					},
				},
			},
			expected: expected{
				valid:    false,
				causeLen: 2,
				fields:   []string{"fieldName", "fieldName"},
			},
		},
		"invalid max less than min List value": {
			selector: &GameServerSelector{
				Lists: map[string]ListSelector{
					"list": {
						MinAvailable: 11,
						MaxAvailable: 2,
					},
				},
			},
			expected: expected{
				valid:    false,
				causeLen: 1,
				fields:   []string{"fieldName"},
			},
		},
	}

	for k, v := range fixtures {
		t.Run(k, func(t *testing.T) {
			v.selector.ApplyDefaults()
			causes, valid := v.selector.Validate("fieldName")
			assert.Equal(t, v.expected.valid, valid)
			assert.Len(t, causes, v.expected.causeLen)

			for i := range v.expected.fields {
				assert.Equal(t, v.expected.fields[i], causes[i].Field)
			}
		})
	}
}

func TestGameServerPriorityValidate(t *testing.T) {
	t.Parallel()

	runtime.FeatureTestMutex.Lock()
	defer runtime.FeatureTestMutex.Unlock()
	assert.NoError(t, runtime.ParseFeatures(fmt.Sprintf("%s=true", runtime.FeatureCountsAndLists)))

	type expected struct {
		valid    bool
		causeLen int
		fields   []string
	}

	fixtures := map[string]struct {
		gsa      *GameServerAllocation
		expected expected
	}{
		"valid Counter Ascending": {
			gsa: &GameServerAllocation{
				Spec: GameServerAllocationSpec{Priorities: []Priority{
					{
						PriorityType: "Counter",
						Key:          "Foo",
						Order:        "Ascending",
					},
				},
				},
			},
			expected: expected{
				valid:    true,
				causeLen: 0,
			},
		},
		"valid Counter Descending": {
			gsa: &GameServerAllocation{
				Spec: GameServerAllocationSpec{Priorities: []Priority{
					{
						PriorityType: "Counter",
						Key:          "Bar",
						Order:        "Descending",
					},
				},
				},
			},
			expected: expected{
				valid:    true,
				causeLen: 0,
			},
		},
		"valid Counter empty Order": {
			gsa: &GameServerAllocation{
				Spec: GameServerAllocationSpec{Priorities: []Priority{
					{
						PriorityType: "Counter",
						Key:          "Bar",
						Order:        "",
					},
				},
				},
			},
			expected: expected{
				valid:    true,
				causeLen: 0,
			},
		},
		"invalid counter type and order": {
			gsa: &GameServerAllocation{
				Spec: GameServerAllocationSpec{Priorities: []Priority{
					{
						PriorityType: "counter",
						Key:          "Babar",
						Order:        "descending",
					},
				},
				},
			},
			expected: expected{
				valid:    false,
				causeLen: 2,
			},
		},
		"valid List Ascending": {
			gsa: &GameServerAllocation{
				Spec: GameServerAllocationSpec{Priorities: []Priority{
					{
						PriorityType: "List",
						Key:          "Baz",
						Order:        "Ascending",
					},
				},
				},
			},
			expected: expected{
				valid:    true,
				causeLen: 0,
			},
		},
		"valid List Descending": {
			gsa: &GameServerAllocation{
				Spec: GameServerAllocationSpec{Priorities: []Priority{
					{
						PriorityType: "List",
						Key:          "Blerg",
						Order:        "Descending",
					},
				},
				},
			},
			expected: expected{
				valid:    true,
				causeLen: 0,
			},
		},
		"valid List empty Order": {
			gsa: &GameServerAllocation{
				Spec: GameServerAllocationSpec{Priorities: []Priority{
					{
						PriorityType: "List",
						Key:          "Blerg",
						Order:        "Ascending",
					},
				},
				},
			},
			expected: expected{
				valid:    true,
				causeLen: 0,
			},
		},
		"invalid list type and order": {
			gsa: &GameServerAllocation{
				Spec: GameServerAllocationSpec{Priorities: []Priority{
					{
						PriorityType: "list",
						Key:          "Schmorg",
						Order:        "ascending",
					},
				},
				},
			},
			expected: expected{
				valid:    false,
				causeLen: 2,
			},
		},
	}

	for k, v := range fixtures {
		t.Run(k, func(t *testing.T) {
			v.gsa.ApplyDefaults()
			causes, valid := v.gsa.Validate()
			assert.Equal(t, v.expected.valid, valid)
			assert.Len(t, causes, v.expected.causeLen)

			for i := range v.expected.fields {
				assert.Equal(t, v.expected.fields[i], causes[i].Field)
			}
		})
	}
}

func TestMetaPatchValidate(t *testing.T) {
	t.Parallel()

	// valid
	mp := &MetaPatch{
		Labels:      nil,
		Annotations: nil,
	}
	causes, valid := mp.Validate()
	assert.True(t, valid)
	assert.Empty(t, causes)

	mp.Labels = map[string]string{}
	mp.Annotations = map[string]string{}
	causes, valid = mp.Validate()
	assert.True(t, valid)
	assert.Empty(t, causes)

	mp.Labels["foo"] = "bar"
	mp.Annotations["bar"] = "foo"
	causes, valid = mp.Validate()
	assert.True(t, valid)
	assert.Empty(t, causes)

	// invalid label
	invalid := mp.DeepCopy()
	invalid.Labels["$$$$"] = "no"

	causes, valid = invalid.Validate()
	assert.False(t, valid)
	require.Len(t, causes, 1)
	assert.Equal(t, "metadata.labels", causes[0].Field)

	// invalid annotation
	invalid = mp.DeepCopy()
	invalid.Annotations["$$$$"] = "no"

	causes, valid = invalid.Validate()
	assert.False(t, valid)
	require.Len(t, causes, 1)
	assert.Equal(t, "metadata.annotations", causes[0].Field)

	// invalid both
	invalid.Labels["$$$$"] = "no"
	causes, valid = invalid.Validate()

	assert.False(t, valid)
	require.Len(t, causes, 2)
	assert.Equal(t, "metadata.labels", causes[0].Field)
	assert.Equal(t, "metadata.annotations", causes[1].Field)
}

func TestGameServerSelectorMatches(t *testing.T) {
	t.Parallel()

	runtime.FeatureTestMutex.Lock()
	defer runtime.FeatureTestMutex.Unlock()

	blueSelector := metav1.LabelSelector{
		MatchLabels: map[string]string{"colour": "blue"},
	}

	allocatedState := agonesv1.GameServerStateAllocated
	fixtures := map[string]struct {
		features   string
		selector   *GameServerSelector
		gameServer *agonesv1.GameServer
		matches    bool
	}{
		"no labels, pass": {
			selector:   &GameServerSelector{},
			gameServer: &agonesv1.GameServer{},
			matches:    true,
		},

		"no labels, fail": {
			selector: &GameServerSelector{
				LabelSelector: blueSelector,
			},
			gameServer: &agonesv1.GameServer{},
			matches:    false,
		},
		"single label, match": {
			selector: &GameServerSelector{
				LabelSelector: blueSelector,
			},
			gameServer: &agonesv1.GameServer{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"colour": "blue"},
				},
			},
			matches: true,
		},
		"single label, fail": {
			selector: &GameServerSelector{
				LabelSelector: blueSelector,
			},
			gameServer: &agonesv1.GameServer{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"colour": "purple"},
				},
			},
			matches: false,
		},
		"two labels, pass": {
			selector: &GameServerSelector{
				LabelSelector: metav1.LabelSelector{
					MatchLabels: map[string]string{"colour": "blue", "animal": "frog"},
				},
			},
			gameServer: &agonesv1.GameServer{
				ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"colour": "blue", "animal": "frog"}},
			},
			matches: true,
		},
		"two labels, fail": {
			selector: &GameServerSelector{
				LabelSelector: metav1.LabelSelector{
					MatchLabels: map[string]string{"colour": "blue", "animal": "cat"},
				},
			},
			gameServer: &agonesv1.GameServer{
				ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"colour": "blue", "animal": "frog"}},
			},
			matches: false,
		},
		"state filter, pass": {
			selector: &GameServerSelector{
				GameServerState: &allocatedState,
			},
			gameServer: &agonesv1.GameServer{Status: agonesv1.GameServerStatus{State: allocatedState}},
			matches:    true,
		},
		"state filter, fail": {
			selector: &GameServerSelector{
				GameServerState: &allocatedState,
			},
			gameServer: &agonesv1.GameServer{Status: agonesv1.GameServerStatus{State: agonesv1.GameServerStateReady}},
			matches:    false,
		},
		"player tracking, between, pass": {
			features: string(runtime.FeaturePlayerAllocationFilter) + "=true",
			selector: &GameServerSelector{Players: &PlayerSelector{
				MinAvailable: 10,
				MaxAvailable: 20,
			}},
			gameServer: &agonesv1.GameServer{Status: agonesv1.GameServerStatus{
				Players: &agonesv1.PlayerStatus{
					Count:    20,
					Capacity: 35,
				},
			}},
			matches: true,
		},
		"player tracking, between, fail": {
			features: string(runtime.FeaturePlayerAllocationFilter) + "=true",
			selector: &GameServerSelector{Players: &PlayerSelector{
				MinAvailable: 10,
				MaxAvailable: 20,
			}},
			gameServer: &agonesv1.GameServer{Status: agonesv1.GameServerStatus{
				Players: &agonesv1.PlayerStatus{
					Count:    30,
					Capacity: 35,
				},
			}},
			matches: false,
		},
		"player tracking, max, pass": {
			features: string(runtime.FeaturePlayerAllocationFilter) + "=true",
			selector: &GameServerSelector{Players: &PlayerSelector{
				MinAvailable: 10,
			}},
			gameServer: &agonesv1.GameServer{Status: agonesv1.GameServerStatus{
				Players: &agonesv1.PlayerStatus{
					Count:    20,
					Capacity: 35,
				},
			}},
			matches: true,
		},
		"player tracking, max, fail": {
			features: string(runtime.FeaturePlayerAllocationFilter) + "=true",
			selector: &GameServerSelector{Players: &PlayerSelector{
				MinAvailable: 10,
			}},
			gameServer: &agonesv1.GameServer{Status: agonesv1.GameServerStatus{
				Players: &agonesv1.PlayerStatus{
					Count:    30,
					Capacity: 35,
				},
			}},
			matches: true,
		},
		"combo": {
			features: string(runtime.FeaturePlayerAllocationFilter) + "=true",
			selector: &GameServerSelector{
				LabelSelector: metav1.LabelSelector{
					MatchLabels: map[string]string{"colour": "blue"},
				},
				GameServerState: &allocatedState,
				Players: &PlayerSelector{
					MinAvailable: 10,
					MaxAvailable: 20,
				},
			},
			gameServer: &agonesv1.GameServer{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"colour": "blue"},
				},
				Status: agonesv1.GameServerStatus{
					State: allocatedState,
					Players: &agonesv1.PlayerStatus{
						Count:    5,
						Capacity: 25,
					},
				},
			},
			matches: true,
		},
		"Counter has available capacity": {
			features: string(runtime.FeatureCountsAndLists) + "=true",
			selector: &GameServerSelector{Counters: map[string]CounterSelector{
				"sessions": {
					MinAvailable: 1,
					MaxAvailable: 1000,
				},
			}},
			gameServer: &agonesv1.GameServer{Status: agonesv1.GameServerStatus{
				Counters: map[string]agonesv1.CounterStatus{
					"sessions": {
						Count:    10,
						Capacity: 1000,
					},
				},
			}},
			matches: true,
		},
		"Counter has below minimum available capacity": {
			features: string(runtime.FeatureCountsAndLists) + "=true",
			selector: &GameServerSelector{Counters: map[string]CounterSelector{
				"players": {
					MinAvailable: 100,
					MaxAvailable: 0,
				},
			}},
			gameServer: &agonesv1.GameServer{Status: agonesv1.GameServerStatus{
				Counters: map[string]agonesv1.CounterStatus{
					"players": {
						Count:    999,
						Capacity: 1000,
					},
				},
			}},
			matches: false,
		},
		"Counter has above maximum available capacity": {
			features: string(runtime.FeatureCountsAndLists) + "=true",
			selector: &GameServerSelector{Counters: map[string]CounterSelector{
				"animals": {
					MinAvailable: 1,
					MaxAvailable: 100,
				},
			}},
			gameServer: &agonesv1.GameServer{Status: agonesv1.GameServerStatus{
				Counters: map[string]agonesv1.CounterStatus{
					"animals": {
						Count:    0,
						Capacity: 1000,
					},
				},
			}},
			matches: false,
		},
		"Counter has count in requested range (MaxCount undefined = 0 = unlimited)": {
			features: string(runtime.FeatureCountsAndLists) + "=true",
			selector: &GameServerSelector{Counters: map[string]CounterSelector{
				"games": {
					MinCount: 1,
				},
			}},
			gameServer: &agonesv1.GameServer{Status: agonesv1.GameServerStatus{
				Counters: map[string]agonesv1.CounterStatus{
					"games": {
						Count:    10,
						Capacity: 1000,
					},
				},
			}},
			matches: true,
		},
		"Counter has count below minimum": {
			features: string(runtime.FeatureCountsAndLists) + "=true",
			selector: &GameServerSelector{Counters: map[string]CounterSelector{
				"characters": {
					MinCount: 1,
					MaxCount: 0,
				},
			}},
			gameServer: &agonesv1.GameServer{Status: agonesv1.GameServerStatus{
				Counters: map[string]agonesv1.CounterStatus{
					"characters": {
						Count:    0,
						Capacity: 100,
					},
				},
			}},
			matches: false,
		},
		"Counter has count above maximum": {
			features: string(runtime.FeatureCountsAndLists) + "=true",
			selector: &GameServerSelector{Counters: map[string]CounterSelector{
				"monsters": {
					MinCount: 0,
					MaxCount: 10,
				},
			}},
			gameServer: &agonesv1.GameServer{Status: agonesv1.GameServerStatus{
				Counters: map[string]agonesv1.CounterStatus{
					"monsters": {
						Count:    11,
						Capacity: 100,
					},
				},
			}},
			matches: false,
		},
		"Counter does not exist": {
			features: string(runtime.FeatureCountsAndLists) + "=true",
			selector: &GameServerSelector{Counters: map[string]CounterSelector{
				"dragoons": {
					MinCount: 1,
					MaxCount: 10,
				},
			}},
			gameServer: &agonesv1.GameServer{Status: agonesv1.GameServerStatus{
				Counters: map[string]agonesv1.CounterStatus{
					"dragons": {
						Count:    1,
						Capacity: 100,
					},
				},
			}},
			matches: false,
		},
		"GameServer does not have Counters": {
			features: string(runtime.FeatureCountsAndLists) + "=true",
			selector: &GameServerSelector{Counters: map[string]CounterSelector{
				"dragoons": {
					MinCount: 1,
					MaxCount: 10,
				},
			}},
			gameServer: &agonesv1.GameServer{Status: agonesv1.GameServerStatus{
				Lists: map[string]agonesv1.ListStatus{
					"bazzles": {
						Capacity: 3,
						Values:   []string{"baz1", "baz2", "baz3"},
					},
				},
			}},
			matches: false,
		},
		"List has available capacity": {
			features: string(runtime.FeatureCountsAndLists) + "=true",
			selector: &GameServerSelector{Lists: map[string]ListSelector{
				"lobbies": {
					MinAvailable: 1,
					MaxAvailable: 3,
				},
			}},
			gameServer: &agonesv1.GameServer{Status: agonesv1.GameServerStatus{
				Lists: map[string]agonesv1.ListStatus{
					"lobbies": {
						Capacity: 3,
						Values:   []string{"lobby1", "lobby2"},
					},
				},
			}},
			matches: true,
		},
		"List has below minimum available capacity": {
			features: string(runtime.FeatureCountsAndLists) + "=true",
			selector: &GameServerSelector{Lists: map[string]ListSelector{
				"avatars": {
					MinAvailable: 1,
					MaxAvailable: 1000,
				},
			}},
			gameServer: &agonesv1.GameServer{Status: agonesv1.GameServerStatus{
				Lists: map[string]agonesv1.ListStatus{
					"avatars": {
						Capacity: 3,
						Values:   []string{"avatar1", "avatar2", "avatar3"},
					},
				},
			}},
			matches: false,
		},
		"List has above maximum available capacity": {
			features: string(runtime.FeatureCountsAndLists) + "=true",
			selector: &GameServerSelector{Lists: map[string]ListSelector{
				"things": {
					MinAvailable: 1,
					MaxAvailable: 10,
				},
			}},
			gameServer: &agonesv1.GameServer{Status: agonesv1.GameServerStatus{
				Lists: map[string]agonesv1.ListStatus{
					"things": {
						Capacity: 1000,
						Values:   []string{"thing1", "thing2", "thing3"},
					},
				},
			}},
			matches: false,
		},
		"List does not exist": {
			features: string(runtime.FeatureCountsAndLists) + "=true",
			selector: &GameServerSelector{Lists: map[string]ListSelector{
				"thingamabobs": {
					MinAvailable: 1,
					MaxAvailable: 100,
				},
			}},
			gameServer: &agonesv1.GameServer{Status: agonesv1.GameServerStatus{
				Lists: map[string]agonesv1.ListStatus{
					"thingamajigs": {
						Capacity: 100,
						Values:   []string{"thingamajig1", "thingamajig2"},
					},
				},
			}},
			matches: false,
		},
		"List contains value": {
			features: string(runtime.FeatureCountsAndLists) + "=true",
			selector: &GameServerSelector{Lists: map[string]ListSelector{
				"bazzles": {
					ContainsValue: "baz1",
				},
			}},
			gameServer: &agonesv1.GameServer{Status: agonesv1.GameServerStatus{
				Lists: map[string]agonesv1.ListStatus{
					"bazzles": {
						Capacity: 3,
						Values:   []string{"baz1", "baz2", "baz3"},
					},
				},
			}},
			matches: true,
		},
		"List does not contain value": {
			features: string(runtime.FeatureCountsAndLists) + "=true",
			selector: &GameServerSelector{Lists: map[string]ListSelector{
				"bazzles": {
					ContainsValue: "BAZ1",
				},
			}},
			gameServer: &agonesv1.GameServer{Status: agonesv1.GameServerStatus{
				Lists: map[string]agonesv1.ListStatus{
					"bazzles": {
						Capacity: 3,
						Values:   []string{"baz1", "baz2", "baz3"},
					},
				},
			}},
			matches: false,
		},
		"GameServer does not have Lists": {
			features: string(runtime.FeatureCountsAndLists) + "=true",
			selector: &GameServerSelector{Lists: map[string]ListSelector{
				"bazzles": {
					ContainsValue: "BAZ1",
				},
			}},
			gameServer: &agonesv1.GameServer{Status: agonesv1.GameServerStatus{
				Counters: map[string]agonesv1.CounterStatus{
					"dragons": {
						Count:    1,
						Capacity: 100,
					},
				},
			}},
			matches: false,
		},
	}

	for k, v := range fixtures {
		t.Run(k, func(t *testing.T) {
			if v.features != "" {
				require.NoError(t, runtime.ParseFeatures(v.features))
			}

			match := v.selector.Matches(v.gameServer)
			assert.Equal(t, v.matches, match)
		})
	}
}

// Helper function for creating int64 pointers
func int64Pointer(x int64) *int64 {
	return &x
}

func TestGameServerCounterActions(t *testing.T) {
	t.Parallel()

	runtime.FeatureTestMutex.Lock()
	defer runtime.FeatureTestMutex.Unlock()
	assert.NoError(t, runtime.ParseFeatures(fmt.Sprintf("%s=true", runtime.FeatureCountsAndLists)))

	DECREMENT := "Decrement"
	INCREMENT := "Increment"

	testScenarios := map[string]struct {
		ca      CounterAction
		counter string
		gs      *agonesv1.GameServer
		want    *agonesv1.GameServer
		wantErr bool
	}{
		"update counter capacity": {
			ca: CounterAction{
				Capacity: int64Pointer(0),
			},
			counter: "mages",
			gs: &agonesv1.GameServer{Status: agonesv1.GameServerStatus{
				Counters: map[string]agonesv1.CounterStatus{
					"mages": {
						Count:    1,
						Capacity: 100,
					}}}},
			want: &agonesv1.GameServer{Status: agonesv1.GameServerStatus{
				Counters: map[string]agonesv1.CounterStatus{
					"mages": {
						Count:    1,
						Capacity: 0,
					}}}},
			wantErr: false,
		},
		"fail update counter capacity and count": {
			ca: CounterAction{
				Action:   &INCREMENT,
				Amount:   int64Pointer(10),
				Capacity: int64Pointer(-1),
			},
			counter: "sages",
			gs: &agonesv1.GameServer{Status: agonesv1.GameServerStatus{
				Counters: map[string]agonesv1.CounterStatus{
					"sages": {
						Count:    99,
						Capacity: 100,
					}}}},
			want: &agonesv1.GameServer{Status: agonesv1.GameServerStatus{
				Counters: map[string]agonesv1.CounterStatus{
					"sages": {
						Count:    99,
						Capacity: 100,
					}}}},
			wantErr: true,
		},
		"update counter count": {
			ca: CounterAction{
				Action: &INCREMENT,
				Amount: int64Pointer(10),
			},
			counter: "baddies",
			gs: &agonesv1.GameServer{Status: agonesv1.GameServerStatus{
				Counters: map[string]agonesv1.CounterStatus{
					"baddies": {
						Count:    1,
						Capacity: 100,
					}}}},
			want: &agonesv1.GameServer{Status: agonesv1.GameServerStatus{
				Counters: map[string]agonesv1.CounterStatus{
					"baddies": {
						Count:    11,
						Capacity: 100,
					}}}},
			wantErr: false,
		},
		"update counter count and capacity": {
			ca: CounterAction{
				Action:   &DECREMENT,
				Amount:   int64Pointer(10),
				Capacity: int64Pointer(10),
			},
			counter: "heroes",
			gs: &agonesv1.GameServer{Status: agonesv1.GameServerStatus{
				Counters: map[string]agonesv1.CounterStatus{
					"heroes": {
						Count:    11,
						Capacity: 100,
					}}}},
			want: &agonesv1.GameServer{Status: agonesv1.GameServerStatus{
				Counters: map[string]agonesv1.CounterStatus{
					"heroes": {
						Count:    1,
						Capacity: 10,
					}}}},
			wantErr: false,
		},
	}

	for test, testScenario := range testScenarios {
		t.Run(test, func(t *testing.T) {
			errs := testScenario.ca.CounterActions(testScenario.counter, testScenario.gs)
			if errs != nil {
				assert.True(t, testScenario.wantErr)
			} else {
				assert.False(t, testScenario.wantErr)
			}
			assert.Equal(t, testScenario.want, testScenario.gs)
		})
	}
}

func TestGameServerListActions(t *testing.T) {
	t.Parallel()

	runtime.FeatureTestMutex.Lock()
	defer runtime.FeatureTestMutex.Unlock()
	assert.NoError(t, runtime.ParseFeatures(fmt.Sprintf("%s=true", runtime.FeatureCountsAndLists)))

	testScenarios := map[string]struct {
		la      ListAction
		list    string
		gs      *agonesv1.GameServer
		want    *agonesv1.GameServer
		wantErr bool
	}{
		"update list capacity": {
			la: ListAction{
				Capacity: int64Pointer(0),
			},
			list: "pages",
			gs: &agonesv1.GameServer{Status: agonesv1.GameServerStatus{
				Lists: map[string]agonesv1.ListStatus{
					"pages": {
						Values:   []string{"page1", "page2"},
						Capacity: 100,
					}}}},
			want: &agonesv1.GameServer{Status: agonesv1.GameServerStatus{
				Lists: map[string]agonesv1.ListStatus{
					"pages": {
						Values:   []string{"page1", "page2"},
						Capacity: 0,
					}}}},
			wantErr: false,
		},
		"update list values": {
			la: ListAction{
				AddValues: []string{"sage1", "sage3"},
			},
			list: "sages",
			gs: &agonesv1.GameServer{Status: agonesv1.GameServerStatus{
				Lists: map[string]agonesv1.ListStatus{
					"sages": {
						Values:   []string{"sage1", "sage2"},
						Capacity: 100,
					}}}},
			want: &agonesv1.GameServer{Status: agonesv1.GameServerStatus{
				Lists: map[string]agonesv1.ListStatus{
					"sages": {
						Values:   []string{"sage1", "sage2", "sage3"},
						Capacity: 100,
					}}}},
			wantErr: false,
		},
		"update list values and capacity": {
			la: ListAction{
				AddValues: []string{"magician1", "magician3"},
				Capacity:  int64Pointer(42),
			},
			list: "magicians",
			gs: &agonesv1.GameServer{Status: agonesv1.GameServerStatus{
				Lists: map[string]agonesv1.ListStatus{
					"magicians": {
						Values:   []string{"magician1", "magician2"},
						Capacity: 100,
					}}}},
			want: &agonesv1.GameServer{Status: agonesv1.GameServerStatus{
				Lists: map[string]agonesv1.ListStatus{
					"magicians": {
						Values:   []string{"magician1", "magician2", "magician3"},
						Capacity: 42,
					}}}},
			wantErr: false,
		},
		"update list values and capacity - value add fails": {
			la: ListAction{
				AddValues: []string{"fairy1", "fairy3"},
				Capacity:  int64Pointer(2),
			},
			list: "fairies",
			gs: &agonesv1.GameServer{Status: agonesv1.GameServerStatus{
				Lists: map[string]agonesv1.ListStatus{
					"fairies": {
						Values:   []string{"fairy1", "fairy2"},
						Capacity: 100,
					}}}},
			want: &agonesv1.GameServer{Status: agonesv1.GameServerStatus{
				Lists: map[string]agonesv1.ListStatus{
					"fairies": {
						Values:   []string{"fairy1", "fairy2"},
						Capacity: 2,
					}}}},
			wantErr: true,
		},
	}

	for test, testScenario := range testScenarios {
		t.Run(test, func(t *testing.T) {
			errs := testScenario.la.ListActions(testScenario.list, testScenario.gs)
			if errs != nil {
				assert.True(t, testScenario.wantErr)
			} else {
				assert.False(t, testScenario.wantErr)
			}
			assert.Equal(t, testScenario.want, testScenario.gs)
		})
	}
}

func TestGameServerAllocationValidate(t *testing.T) {
	t.Parallel()

	gsa := &GameServerAllocation{}
	gsa.ApplyDefaults()

	causes, ok := gsa.Validate()
	assert.True(t, ok)
	assert.Empty(t, causes)

	gsa.Spec.Scheduling = "FLERG"

	causes, ok = gsa.Validate()
	assert.False(t, ok)
	assert.Len(t, causes, 1)

	assert.Equal(t, metav1.CauseTypeFieldValueInvalid, causes[0].Type)
	assert.Equal(t, "spec.scheduling", causes[0].Field)

	runtime.FeatureTestMutex.Lock()
	defer runtime.FeatureTestMutex.Unlock()
	assert.NoError(t, runtime.ParseFeatures(fmt.Sprintf("%s=true", runtime.FeaturePlayerAllocationFilter)))

	// invalid player selection
	gsa = &GameServerAllocation{
		Spec: GameServerAllocationSpec{
			Required: GameServerSelector{
				Players: &PlayerSelector{
					MinAvailable: -10,
				},
			},
			Preferred: []GameServerSelector{
				{Players: &PlayerSelector{MaxAvailable: -10}},
			},
			MetaPatch: MetaPatch{
				Labels: map[string]string{"$$$": "foo"},
			},
		},
	}
	gsa.ApplyDefaults()

	causes, ok = gsa.Validate()
	assert.False(t, ok)
	assert.Len(t, causes, 4)
	assert.Equal(t, "spec.required", causes[0].Field)
	assert.Equal(t, "spec.preferred[0]", causes[1].Field)
	assert.Equal(t, "spec.preferred[0]", causes[2].Field)
	assert.Equal(t, "metadata.labels", causes[3].Field)
}

func TestGameServerAllocationConverter(t *testing.T) {
	t.Parallel()

	gsa := &GameServerAllocation{
		Spec: GameServerAllocationSpec{
			Scheduling: "Packed",
			Required: GameServerSelector{
				Players: &PlayerSelector{
					MinAvailable: 5,
					MaxAvailable: 10,
				},
			},
			Preferred: []GameServerSelector{
				{Players: &PlayerSelector{MinAvailable: 10,
					MaxAvailable: 20}},
			},
		},
	}
	gsaExpected := &GameServerAllocation{
		Spec: GameServerAllocationSpec{
			Scheduling: "Packed",
			Required: GameServerSelector{
				Players: &PlayerSelector{
					MinAvailable: 5,
					MaxAvailable: 10,
				},
			},
			Preferred: []GameServerSelector{
				{Players: &PlayerSelector{MinAvailable: 10,
					MaxAvailable: 20}},
			},
			Selectors: []GameServerSelector{
				{Players: &PlayerSelector{MinAvailable: 10,
					MaxAvailable: 20}},
				{Players: &PlayerSelector{
					MinAvailable: 5,
					MaxAvailable: 10}},
			},
		},
	}

	gsa.Converter()
	assert.Equal(t, gsaExpected, gsa)
}
