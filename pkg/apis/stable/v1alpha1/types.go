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
	"github.com/agonio/agon/pkg/apis/stable"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// Creating is when the Pod for the GameServer is being created,
	// but they have yet to register themselves yet as Ready
	Creating State = "Creating"
	// Starting is for when the Pods for the GameServer are being
	// created but have yet to register themselves as Ready
	Starting State = "Starting"
	// RequestReady is when the GameServer has declared that it is ready
	RequestReady State = "RequestReady"
	// Ready is when a GameServer is ready to take connections
	// from Game clients
	Ready State = "Ready"
	// Shutdown is when the GameServer has shutdown and everything needs to be
	// deleted from the cluster
	Shutdown State = "Shutdown"
	// Error is when something has gone with the Gameserver and
	// it cannot be resolved
	Error State = "Error"

	// Static PortPolicy means that the user defines the hostPort to be used
	// in the configuration.
	Static PortPolicy = "static"

	// GameServerPodLabel is the label that the name of the GameServer
	// is set on the Pod the GameServer controls
	GameServerPodLabel = stable.GroupName + "/gameserver"
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
	State   State  `json:"state"`
	Port    int32  `json:"port"`
	Address string `json:"address"`
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
		gs.Status.State = Creating
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

// Pod creates a new Pod from the PodTemplateSpec
// attached for the
func (gs *GameServer) Pod(sidecars ...corev1.Container) (*corev1.Pod, error) {
	pod := &corev1.Pod{
		ObjectMeta: *gs.Spec.Template.ObjectMeta.DeepCopy(),
		Spec:       *gs.Spec.Template.Spec.DeepCopy(),
	}
	// Switch to GenerateName, so that we always get a Unique name for the Pod, and there
	// can be no collisions
	pod.ObjectMeta.GenerateName = gs.ObjectMeta.Name + "-"
	pod.ObjectMeta.Name = ""
	// Pods for GameServers need to stay in the same namespace
	pod.ObjectMeta.Namespace = gs.ObjectMeta.Namespace
	// Make sure these are blank, just in case
	pod.ResourceVersion = ""
	pod.UID = ""
	if pod.ObjectMeta.Labels == nil {
		pod.ObjectMeta.Labels = make(map[string]string, 1)
	}
	pod.ObjectMeta.Labels[stable.GroupName+"/role"] = "gameserver"
	// store the GameServer name as a label, for easy lookup later on
	pod.ObjectMeta.Labels[GameServerPodLabel] = gs.ObjectMeta.Name
	ref := metav1.NewControllerRef(gs, SchemeGroupVersion.WithKind("GameServer"))
	pod.ObjectMeta.OwnerReferences = append(pod.ObjectMeta.OwnerReferences, *ref)

	i, gsContainer, err := gs.FindGameServerContainer()
	// this shouldn't happen, but if it does.
	if err != nil {
		return pod, err
	}

	cp := corev1.ContainerPort{
		ContainerPort: gs.Spec.ContainerPort,
		HostPort:      gs.Spec.HostPort,
		Protocol:      gs.Spec.Protocol,
	}
	gsContainer.Ports = append(gsContainer.Ports, cp)
	pod.Spec.Containers[i] = gsContainer

	pod.Spec.Containers = append(pod.Spec.Containers, sidecars...)
	return pod, nil
}
