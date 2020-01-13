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

	"agones.dev/agones/pkg/apis"
	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

const (
	// GameServerAllocationAllocated is allocation successful
	GameServerAllocationAllocated GameServerAllocationState = "Allocated"
	// GameServerAllocationUnAllocated when the allocation is unsuccessful
	GameServerAllocationUnAllocated GameServerAllocationState = "UnAllocated"
	// GameServerAllocationContention when the allocation is unsuccessful
	// because of contention
	GameServerAllocationContention GameServerAllocationState = "Contention"
)

// GameServerAllocationState is the Allocation state
type GameServerAllocationState string

// +genclient
// +genclient:onlyVerbs=create
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// GameServerAllocation is the data structure for allocating against a set of
// GameServers, defined `required` and `preferred` selectors
type GameServerAllocation struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              GameServerAllocationSpec   `json:"spec"`
	Status            GameServerAllocationStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// GameServerAllocationList is a list of GameServer Allocation resources
type GameServerAllocationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []GameServerAllocation `json:"items"`
}

// GameServerAllocationSpec is the spec for a GameServerAllocation
type GameServerAllocationSpec struct {
	// MultiClusterPolicySelector if specified, multi-cluster policies are applied.
	// Otherwise, allocation will happen locally.
	MultiClusterSetting MultiClusterSetting `json:"multiClusterSetting,omitempty"`

	// Required The required allocation. Defaults to all GameServers.
	Required metav1.LabelSelector `json:"required,omitempty"`

	// Preferred ordered list of preferred allocations out of the `required` set.
	// If the first selector is not matched,
	// the selection attempts the second selector, and so on.
	Preferred []metav1.LabelSelector `json:"preferred,omitempty"`

	// Scheduling strategy. Defaults to "Packed".
	Scheduling apis.SchedulingStrategy `json:"scheduling"`

	// MetaPatch is optional custom metadata that is added to the game server at allocation
	// You can use this to tell the server necessary session data
	MetaPatch MetaPatch `json:"metadata,omitempty"`
}

// MultiClusterSetting specifies settings for multi-cluster allocation.
type MultiClusterSetting struct {
	Enabled        bool                 `json:"enabled,omitempty"`
	PolicySelector metav1.LabelSelector `json:"policySelector,omitempty"`
}

// MetaPatch is the metadata used to patch the GameServer metadata on allocation
type MetaPatch struct {
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

// PreferredSelectors converts all the preferred label selectors into an array of
// labels.Selectors. This is useful as they all have `Match()` functions!
func (gsas *GameServerAllocationSpec) PreferredSelectors() ([]labels.Selector, error) {
	list := make([]labels.Selector, len(gsas.Preferred))

	var err error
	for i, p := range gsas.Preferred {
		p := p
		list[i], err = metav1.LabelSelectorAsSelector(&p)
		if err != nil {
			break
		}
	}

	return list, errors.WithStack(err)
}

// GameServerAllocationStatus is the status for an GameServerAllocation resource
type GameServerAllocationStatus struct {
	// GameServerState is the current state of an GameServerAllocation, e.g. Allocated, or UnAllocated
	State          GameServerAllocationState       `json:"state"`
	GameServerName string                          `json:"gameServerName"`
	Ports          []agonesv1.GameServerStatusPort `json:"ports,omitempty"`
	Address        string                          `json:"address,omitempty"`
	NodeName       string                          `json:"nodeName,omitempty"`
}

// ApplyDefaults applies the default values to this GameServerAllocation
func (gsa *GameServerAllocation) ApplyDefaults() {
	if gsa.Spec.Scheduling == "" {
		gsa.Spec.Scheduling = apis.Packed
	}
}

// Validate validation for the GameServerAllocation
func (gsa *GameServerAllocation) Validate() ([]metav1.StatusCause, bool) {
	var causes []metav1.StatusCause

	valid := false
	for _, v := range []apis.SchedulingStrategy{apis.Packed, apis.Distributed} {
		if gsa.Spec.Scheduling == v {
			valid = true
		}
	}
	if !valid {
		causes = append(causes, metav1.StatusCause{Type: metav1.CauseTypeFieldValueInvalid,
			Field:   "spec.scheduling",
			Message: fmt.Sprintf("Invalid value: %s, value must be either Packed or Distributed", gsa.Spec.Scheduling)})
	}

	return causes, len(causes) == 0
}
