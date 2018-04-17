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

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// FleetAllocation is the data structure for allocating against a Fleet
type FleetAllocation struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   FleetAllocationSpec   `json:"spec"`
	Status FleetAllocationStatus `json:"status"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// FleetAllocationList is a list of Fleet Allocation resources
type FleetAllocationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []FleetAllocation `json:"items"`
}

// FleetAllocationSpec is the spec for a Fleet
// Allocation
type FleetAllocationSpec struct {
	FleetName string `json:"fleetName"`
}

// FleetAllocationStatus will contain the
// `GameServer` that has been allocated from
// a Fleet
type FleetAllocationStatus struct {
	GameServer *GameServer
}

// ValidateUpdate validates when an update occurs
func (fa *FleetAllocation) ValidateUpdate(new *FleetAllocation) (bool, []metav1.StatusCause) {
	var causes []metav1.StatusCause

	if fa.Spec.FleetName != new.Spec.FleetName {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Field:   "fleetName",
			Message: "fleetName cannot be updated",
		})
	}

	return len(causes) == 0, causes
}
