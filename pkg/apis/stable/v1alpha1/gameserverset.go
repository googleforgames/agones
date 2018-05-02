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
	"reflect"

	"agones.dev/agones/pkg/apis/stable"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// GameServerSetGameServerLabel is the label that the name of the GameServerSet
	// is set on the GameServer the GameServerSet controls
	GameServerSetGameServerLabel = stable.GroupName + "/gameserverset"
)

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// GameServerSet is the data structure a set of GameServers
// This matches philosophically with the relationship between
// Depoyments and ReplicaSets
type GameServerSet struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   GameServerSetSpec   `json:"spec"`
	Status GameServerSetStatus `json:"status"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// GameServerSetList is a list of GameServerSet resources
type GameServerSetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []GameServerSet `json:"items"`
}

// GameServerSetSpec the specification for
type GameServerSetSpec struct {
	// Replicas are the number of GameServers that should be in this set
	Replicas int32 `json:"replicas"`
	// Template the GameServer template to apply for this GameServerSet
	Template GameServerTemplateSpec `json:"template"`
}

// GameServerSetStatus is the status of a GameServerSet
type GameServerSetStatus struct {
	// Replicas the total number of current GameServer replicas
	Replicas int32 `json:"replicas"`
	// ReadyReplicas are the number of Ready GameServer replicas
	ReadyReplicas int32 `json:"readyReplicas"`
	// AllocatedReplicas are the number of Allocated GameServer replicas
	AllocatedReplicas int32 `json:"allocatedReplicas"`
}

// ValidateUpdate validates when updates occur. The argument
// is the new GameServerSet, being passed into the old GameServerSet
func (gsSet *GameServerSet) ValidateUpdate(new *GameServerSet) (bool, []metav1.StatusCause) {
	var causes []metav1.StatusCause
	if !reflect.DeepEqual(gsSet.Spec.Template, new.Spec.Template) {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Field:   "template",
			Message: "template values cannot be updated after creation",
		})
	}

	return len(causes) == 0, causes
}

// GameServer returns a single GameServer derived
// from the GameSever template
func (gsSet *GameServerSet) GameServer() *GameServer {
	gs := &GameServer{
		ObjectMeta: *gsSet.Spec.Template.ObjectMeta.DeepCopy(),
		Spec:       *gsSet.Spec.Template.Spec.DeepCopy(),
	}

	// Switch to GenerateName, so that we always get a Unique name for the GameServer, and there
	// can be no collisions
	gs.ObjectMeta.GenerateName = gsSet.ObjectMeta.Name + "-"
	gs.ObjectMeta.Name = ""
	gs.ObjectMeta.Namespace = gsSet.ObjectMeta.Namespace
	gs.ObjectMeta.ResourceVersion = ""
	gs.ObjectMeta.UID = ""

	ref := metav1.NewControllerRef(gsSet, SchemeGroupVersion.WithKind("GameServerSet"))
	gs.ObjectMeta.OwnerReferences = append(gs.ObjectMeta.OwnerReferences, *ref)

	if gs.ObjectMeta.Labels == nil {
		gs.ObjectMeta.Labels = make(map[string]string, 1)
	}

	gs.ObjectMeta.Labels[GameServerSetGameServerLabel] = gsSet.ObjectMeta.Name

	return gs
}
