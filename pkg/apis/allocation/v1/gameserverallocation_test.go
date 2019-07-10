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
	"testing"

	"agones.dev/agones/pkg/apis"
	"agones.dev/agones/pkg/apis/stable/v1alpha1"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

func TestGameServerAllocationApplyDefaults(t *testing.T) {
	t.Parallel()

	gsa := &GameServerAllocation{}
	gsa.ApplyDefaults()

	assert.Equal(t, apis.Packed, gsa.Spec.Scheduling)

	gsa = &GameServerAllocation{Spec: GameServerAllocationSpec{Scheduling: apis.Distributed}}
	gsa.ApplyDefaults()
	assert.Equal(t, apis.Distributed, gsa.Spec.Scheduling)
}

func TestGameServerAllocationSpecPreferredSelectors(t *testing.T) {
	t.Parallel()

	gsas := &GameServerAllocationSpec{
		Preferred: []metav1.LabelSelector{
			{MatchLabels: map[string]string{"check": "blue"}},
			{MatchLabels: map[string]string{"check": "red"}},
		},
	}

	selectors, err := gsas.PreferredSelectors()
	assert.Nil(t, err)
	assert.Len(t, selectors, 2)

	gs := &v1alpha1.GameServer{ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{}}}

	for _, s := range selectors {
		assert.False(t, s.Matches(labels.Set(gs.ObjectMeta.Labels)))
	}

	gs.ObjectMeta.Labels["check"] = "blue"
	assert.True(t, selectors[0].Matches(labels.Set(gs.ObjectMeta.Labels)))
	assert.False(t, selectors[1].Matches(labels.Set(gs.ObjectMeta.Labels)))

	gs.ObjectMeta.Labels["check"] = "red"
	assert.False(t, selectors[0].Matches(labels.Set(gs.ObjectMeta.Labels)))
	assert.True(t, selectors[1].Matches(labels.Set(gs.ObjectMeta.Labels)))
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
}
