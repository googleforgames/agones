// Copyright 2017 Google Inc. All Rights Reserved.
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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// CreatingState is when the Pod for the GameServer is being created,
	// but they have yet to register themselves yet as Ready
	CreatingState State = "Creating"
	// StartingState is for when the Pods for the GameServer are being
	// created but have yet to register themselves as Ready
	StartingState State = "Starting"
	// ReadyState is when a GameServer is ready to take connections
	// from Game clients
	ReadyState State = "Ready"

	// Error state is when something has gone with the Gameserver and
	// it cannot be resolved
	ErrorState State = "Error"

	// StaticPortPolicy is the PortPolicy is defined in the configuration
	StaticPortPolicy PortPolicy = "static"
)

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// GameServer is the data structure for a gameserver resource
type GameServer struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   GameServerSpec   `json:"spec"`
	Status GameServerStatus `json:"status"`
}

// GameServerSpec is the spec for a GameServer resource
type GameServerSpec struct {
	// Container specifies which Pod container is the game server. Only required if there is more than once
	// container defined
	Container string `json:"container,omitempty"`
	// PortPolicy defined the policy for how the HostPort is populated.
	// `static` PortPolicy is the only current option. Dynamic port allocated will come in future releases.
	// When `static` is the policy specified, `HostPort` is required, to specify the port that game clients will
	// connect to
	PortPolicy PortPolicy `json:"PortPolicy,omitempty"`
	// ContainerPort is the port that is being opened on the game server process
	ContainerPort int32 `json:"containerPort"`
	// HostPort the port exposed on the host for clients to connect to
	HostPort int32 `json:"hostPort,omitempty"`
	// Protocoal is the network protocol being used. Defaults to UDP. TCP is the only other option
	Protocol corev1.Protocol `json:"protocol,omitempty"`
	// Template describes the Pod that will be created for the GameServer
	Template corev1.PodTemplateSpec `json:"template"`
}

// State is the state for the GameServer
type State string

// PortPolicy is the port policy for the GameServer
type PortPolicy string

// GameServerStatus is the status for a GameServer resource
type GameServerStatus struct {
	// The current state of a GameServer, e.g. Creating, Starting, Ready, etc
	State State `json:"state"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// GameServerList is a list of GameServer resources
type GameServerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []GameServer `json:"items"`
}

// ApplyDefaults applies default values to the GameServer if they are not already populated
func (gs *GameServer) ApplyDefaults() {
	if len(gs.Spec.Template.Spec.Containers) == 1 {
		gs.Spec.Container = gs.Spec.Template.Spec.Containers[0].Name
	}

	if gs.Spec.Protocol == "" {
		gs.Spec.Protocol = "UDP"
	}

	if gs.Status.State == "" {
		gs.Status.State = CreatingState
	}
}

// FindGameServerContainer returns the container that is specified in
// spec.gameServer.container. Returns the index and the value.
// Returns an error if not found
func (gs *GameServer) FindGameServerContainer() (int, corev1.Container, error) {
	for i, c := range gs.Spec.Template.Spec.Containers {
		if c.Name == gs.Spec.Container {
			return i, c, nil
		}
	}

	return -1, corev1.Container{}, errors.Errorf("Could not find a container named %s", gs.Spec.Container)
}
