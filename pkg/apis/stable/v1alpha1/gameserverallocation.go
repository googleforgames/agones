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

package v1alpha1

import (
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

const (
	// GameServerAllocationAllocated is allocation successful
	GameServerAllocationAllocated GameServerAllocationState = "Allocated"
	// GameServerAllocationUnAllocated when the allocation is unsuccessful
	GameServerAllocationUnAllocated GameServerAllocationState = "UnAllocated"
)

// GameServerAllocationState is the Allocation state
type GameServerAllocationState string

// +genclient
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
	// Required The required allocation. Defaults to all GameServers.
	Required metav1.LabelSelector `json:"required,omitempty"`

	// Preferred ordered list of preferred allocations out of the `required` set.
	// If the first selector is not matched,
	// the selection attempts the second selector, and so on.
	Preferred []metav1.LabelSelector `json:"preferred,omitempty"`

	// Scheduling strategy. Defaults to "Packed".
	Scheduling SchedulingStrategy `json:"scheduling"`

	// MetaPatch is optional custom metadata that is added to the game server at allocation
	// You can use this to tell the server necessary session data
	MetaPatch MetaPatch `json:"metadata,omitempty"`
}

// PreferredSelectors converts all the the preferred label selectors into an array of
// labels.Selectors. This is useful as they all have `Match()` functions!
func (gsas *GameServerAllocationSpec) PreferredSelectors() ([]labels.Selector, error) {
	list := make([]labels.Selector, len(gsas.Preferred))

	var err error
	for i, p := range gsas.Preferred {
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
	State          GameServerAllocationState `json:"state"`
	GameServerName string                    `json:"gameServerName"`
	Ports          []GameServerStatusPort    `json:"ports,omitempty"`
	Address        string                    `json:"address,omitempty"`
	NodeName       string                    `json:"nodeName,omitempty"`
}

// ApplyDefaults applies the default values to this GameServerAllocation
func (gsa *GameServerAllocation) ApplyDefaults() {
	if gsa.Spec.Scheduling == "" {
		gsa.Spec.Scheduling = Packed
	}
}

// ValidateUpdate validates when an update occurs
func (gsa *GameServerAllocation) ValidateUpdate(new *GameServerAllocation) ([]metav1.StatusCause, bool) {
	var causes []metav1.StatusCause

	if !equality.Semantic.DeepEqual(gsa.Spec, new.Spec) {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Field:   "spec",
			Message: "spec cannot be updated",
		})
	}

	return causes, len(causes) == 0
}
