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
	"agones.dev/agones/pkg/apis/stable"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

const (
	// PortAllocation is for when a dynamically allocating GameServer
	// is being created, an open port needs to be allocated
	PortAllocation State = "PortAllocation"
	// Creating is before the Pod for the GameServer is being created
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
	// Unhealthy is when the GameServer has failed its health checks
	Unhealthy State = "Unhealthy"
	// Allocated is when the GameServer has been allocated to a session
	Allocated State = "Allocated"

	// Static PortPolicy means that the user defines the hostPort to be used
	// in the configuration.
	Static PortPolicy = "static"
	// Dynamic PortPolicy means that the system will choose an open
	// port for the GameServer in question
	Dynamic PortPolicy = "dynamic"

	// RoleLabel is the label in which the Agones role is specified.
	// Pods from a GameServer will have the value "gameserver"
	RoleLabel = stable.GroupName + "/role"
	// GameServerLabelRole is the GameServer label value for RoleLabel
	GameServerLabelRole = "gameserver"
	// GameServerPodLabel is the label that the name of the GameServer
	// is set on the Pod the GameServer controls
	GameServerPodLabel = stable.GroupName + "/gameserver"
	// GameServerContainerAnnotation is the annotation that stores
	// which container is the container that runs the dedicated game server
	GameServerContainerAnnotation = stable.GroupName + "/container"
	// SidecarServiceAccountName is the default service account for managing access to get/update GameServers
	SidecarServiceAccountName = "agones-sdk"
)

var (
	// GameServerRolePodSelector is the selector to get all GameServer Pods
	GameServerRolePodSelector = labels.SelectorFromSet(labels.Set{RoleLabel: GameServerLabelRole})
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

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// GameServerList is a list of GameServer resources
type GameServerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []GameServer `json:"items"`
}

// GameServerTemplateSpec is a template for GameServers
type GameServerTemplateSpec struct {
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              GameServerSpec `json:"spec"`
}

// GameServerSpec is the spec for a GameServer resource
type GameServerSpec struct {
	// Container specifies which Pod container is the game server. Only required if there is more than one
	// container defined
	Container string `json:"container,omitempty"`
	// PortPolicy defines the policy for how the HostPort is populated.
	// Dynamic port will allocate a HostPort within the selected MIN_PORT and MAX_PORT range passed to the controller
	// at installation time.
	// When `static` is the policy specified, `HostPort` is required, to specify the port that game clients will
	// connect to
	PortPolicy PortPolicy `json:"portPolicy,omitempty"`
	// ContainerPort is the port that is being opened on the game server process
	ContainerPort int32 `json:"containerPort"`
	// HostPort the port exposed on the host for clients to connect to
	HostPort int32 `json:"hostPort,omitempty"`
	// Protocol is the network protocol being used. Defaults to UDP. TCP is the only other option
	Protocol corev1.Protocol `json:"protocol,omitempty"`
	// Health configures health checking
	Health Health `json:"health,omitempty"`
	// Template describes the Pod that will be created for the GameServer
	Template corev1.PodTemplateSpec `json:"template"`
}

// State is the state for the GameServer
type State string

// PortPolicy is the port policy for the GameServer
type PortPolicy string

// Health configures health checking on the GameServer
type Health struct {
	// Disabled is whether health checking is disabled or not
	Disabled bool `json:"disabled,omitempty"`
	// PeriodSeconds is the number of seconds each health ping has to occur in
	PeriodSeconds int32 `json:"periodSeconds,omitempty"`
	// FailureThreshold how many failures in a row constitutes unhealthy
	FailureThreshold int32 `json:"failureThreshold,omitempty"`
	// InitialDelaySeconds initial delay before checking health
	InitialDelaySeconds int32 `json:"initialDelaySeconds,omitempty"`
}

// GameServerStatus is the status for a GameServer resource
type GameServerStatus struct {
	// State is the current state of a GameServer, e.g. Creating, Starting, Ready, etc
	State    State  `json:"state"`
	Port     int32  `json:"port"`
	Address  string `json:"address"`
	NodeName string `json:"nodeName"`
}

// ApplyDefaults applies default values to the GameServer if they are not already populated
func (gs *GameServer) ApplyDefaults() {
	gs.ObjectMeta.Finalizers = append(gs.ObjectMeta.Finalizers, stable.GroupName)

	if len(gs.Spec.Template.Spec.Containers) == 1 {
		gs.Spec.Container = gs.Spec.Template.Spec.Containers[0].Name
	}

	// basic spec
	if gs.Spec.PortPolicy == "" {
		gs.Spec.PortPolicy = Dynamic
	}

	if gs.Spec.Protocol == "" {
		gs.Spec.Protocol = "UDP"
	}

	if gs.Status.State == "" {
		if gs.Spec.PortPolicy == Dynamic {
			gs.Status.State = PortAllocation
		} else {
			gs.Status.State = Creating
		}
	}

	// health
	if !gs.Spec.Health.Disabled {
		if gs.Spec.Health.PeriodSeconds <= 0 {
			gs.Spec.Health.PeriodSeconds = 5
		}
		if gs.Spec.Health.FailureThreshold <= 0 {
			gs.Spec.Health.FailureThreshold = 3
		}
		if gs.Spec.Health.InitialDelaySeconds <= 0 {
			gs.Spec.Health.InitialDelaySeconds = 5
		}
	}
}

// Validate validates the GameServer configuration.
// If a GameServer is invalid there will be > 0 values in
// the returned array
func (gs *GameServer) Validate() (bool, []metav1.StatusCause) {
	var causes []metav1.StatusCause

	// make sure a name is specified when there is multiple containers in the pod.
	if len(gs.Spec.Container) == 0 && len(gs.Spec.Template.Spec.Containers) > 1 {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Field:   "container",
			Message: "Container is required when using multiple containers in the pod template",
		})
	}

	// no host port when using dynamic PortPolicy
	if gs.Spec.HostPort > 0 && gs.Spec.PortPolicy == Dynamic {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Field:   "hostPort",
			Message: "HostPort cannot be specified with a Dynamic PortPolicy",
		})
	}

	// make sure the container value points to a valid container
	_, _, err := gs.FindGameServerContainer()
	if err != nil {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Field:   "container",
			Message: err.Error(),
		})
	}

	return len(causes) == 0, causes
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
// attached to the GameServer resource
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
	if pod.Spec.ServiceAccountName == "" {
		pod.Spec.ServiceAccountName = SidecarServiceAccountName
	}
	pod.UID = ""
	if pod.ObjectMeta.Labels == nil {
		pod.ObjectMeta.Labels = make(map[string]string, 2)
	}
	if pod.ObjectMeta.Annotations == nil {
		pod.ObjectMeta.Annotations = make(map[string]string, 1)
	}
	pod.ObjectMeta.Labels[RoleLabel] = GameServerLabelRole
	// store the GameServer name as a label, for easy lookup later on
	pod.ObjectMeta.Labels[GameServerPodLabel] = gs.ObjectMeta.Name
	// store the GameServer container as an annotation, to make lookup at a Pod level easier
	pod.ObjectMeta.Annotations[GameServerContainerAnnotation] = gs.Spec.Container

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
