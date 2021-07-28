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
	assert.NoError(t, runtime.ParseFeatures(fmt.Sprintf("%s=true&%s=true", runtime.FeaturePlayerAllocationFilter, runtime.FeatureStateAllocationFilter)))

	gsa = &GameServerAllocation{}
	gsa.ApplyDefaults()

	assert.Equal(t, agonesv1.GameServerStateReady, *gsa.Spec.Required.GameServerState)
	assert.Equal(t, int64(0), gsa.Spec.Required.Players.MaxAvailable)
	assert.Equal(t, int64(0), gsa.Spec.Required.Players.MinAvailable)
}

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

func TestGameServerSelectorApplyDefaults(t *testing.T) {
	t.Parallel()
	runtime.FeatureTestMutex.Lock()
	defer runtime.FeatureTestMutex.Unlock()

	assert.NoError(t, runtime.ParseFeatures(fmt.Sprintf("%s=true&%s=true", runtime.FeaturePlayerAllocationFilter, runtime.FeatureStateAllocationFilter)))

	s := &GameServerSelector{}

	// no defaults
	s.ApplyDefaults()
	assert.Equal(t, agonesv1.GameServerStateReady, *s.GameServerState)
	assert.Equal(t, int64(0), s.Players.MinAvailable)
	assert.Equal(t, int64(0), s.Players.MaxAvailable)

	state := agonesv1.GameServerStateAllocated
	// set values
	s = &GameServerSelector{
		GameServerState: &state,
		Players:         &PlayerSelector{MinAvailable: 10, MaxAvailable: 20},
	}
	s.ApplyDefaults()
	assert.Equal(t, state, *s.GameServerState)
	assert.Equal(t, int64(10), s.Players.MinAvailable)
	assert.Equal(t, int64(20), s.Players.MaxAvailable)
}

func TestGameServerSelectorValidate(t *testing.T) {
	t.Parallel()

	runtime.FeatureTestMutex.Lock()
	defer runtime.FeatureTestMutex.Unlock()

	assert.NoError(t, runtime.ParseFeatures(fmt.Sprintf("%s=true&%s=true", runtime.FeaturePlayerAllocationFilter, runtime.FeatureStateAllocationFilter)))

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
			features: string(runtime.FeatureStateAllocationFilter) + "=true",
			selector: &GameServerSelector{
				GameServerState: &allocatedState,
			},
			gameServer: &agonesv1.GameServer{Status: agonesv1.GameServerStatus{State: allocatedState}},
			matches:    true,
		},
		"state filter, fail": {
			features: string(runtime.FeatureStateAllocationFilter) + "=true",
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
			features: string(runtime.FeaturePlayerAllocationFilter) + "=true&" + string(runtime.FeatureStateAllocationFilter) + "=true",
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
	assert.NoError(t, runtime.ParseFeatures(fmt.Sprintf("%s=true&%s=true", runtime.FeaturePlayerAllocationFilter, runtime.FeatureStateAllocationFilter)))

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
		},
	}
	gsa.ApplyDefaults()

	causes, ok = gsa.Validate()
	assert.False(t, ok)
	assert.Len(t, causes, 3)
	assert.Equal(t, "spec.required", causes[0].Field)
	assert.Equal(t, "spec.preferred[0]", causes[1].Field)
	assert.Equal(t, "spec.preferred[0]", causes[2].Field)
}
