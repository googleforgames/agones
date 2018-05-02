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
	"agones.dev/agones/pkg/apis/stable"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// FleetGameServerSetLabel is the label that the name of the Fleet
	// is set to on the GameServerSet the Fleet controls
	FleetGameServerSetLabel = stable.GroupName + "/fleet"
)

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Fleet is the data structure for a gameserver resource
type Fleet struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   FleetSpec   `json:"spec"`
	Status FleetStatus `json:"status"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// FleetList is a list of GameServer resources
type FleetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []Fleet `json:"items"`
}

// FleetSpec is the spec for a Fleet
type FleetSpec struct {
	// Replicas are the number of GameServers that should be in this set
	Replicas int32 `json:"replicas"`
	// Template the GameServer template to apply for this Fleet
	Template GameServerTemplateSpec `json:"template"`
}

// FleetStatus is the status of a GameServerSet
type FleetStatus struct {
	// Replicas the total number of current GameServer replicas
	Replicas int32 `json:"replicas"`
	// ReadyReplicas are the number of Ready GameServer replicas
	ReadyReplicas int32 `json:"readyReplicas"`
	// AllocatedReplicas are the number of Allocated GameServer replicas
	AllocatedReplicas int32 `json:"allocatedReplicas"`
}

// GameServerSet returns a single GameServerSet for this Fleet definition
func (f *Fleet) GameServerSet() *GameServerSet {
	gsSet := &GameServerSet{
		ObjectMeta: *f.Spec.Template.ObjectMeta.DeepCopy(),
		Spec: GameServerSetSpec{
			Replicas: f.Spec.Replicas,
			Template: f.Spec.Template,
		},
	}

	// Switch to GenerateName, so that we always get a Unique name for the GameServerSet, and there
	// can be no collisions
	gsSet.ObjectMeta.GenerateName = f.ObjectMeta.Name + "-"
	gsSet.ObjectMeta.Name = ""
	gsSet.ObjectMeta.Namespace = f.ObjectMeta.Namespace
	gsSet.ObjectMeta.ResourceVersion = ""
	gsSet.ObjectMeta.UID = ""

	ref := metav1.NewControllerRef(f, SchemeGroupVersion.WithKind("Fleet"))
	gsSet.ObjectMeta.OwnerReferences = append(gsSet.ObjectMeta.OwnerReferences, *ref)

	if gsSet.ObjectMeta.Labels == nil {
		gsSet.ObjectMeta.Labels = make(map[string]string, 1)
	}

	gsSet.ObjectMeta.Labels[FleetGameServerSetLabel] = f.ObjectMeta.Name

	return gsSet
}
