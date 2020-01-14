// Copyright 2017 Google LLC All Rights Reserved.
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
	"encoding/json"
	"fmt"
	"net"

	"github.com/mattbaird/jsonpatch"

	"agones.dev/agones/pkg"
	"agones.dev/agones/pkg/apis"
	"agones.dev/agones/pkg/apis/agones"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

const (
	// GameServerStatePortAllocation is for when a dynamically allocating GameServer
	// is being created, an open port needs to be allocated
	GameServerStatePortAllocation GameServerState = "PortAllocation"
	// GameServerStateCreating is before the Pod for the GameServer is being created
	GameServerStateCreating GameServerState = "Creating"
	// GameServerStateStarting is for when the Pods for the GameServer are being
	// created but are not yet Scheduled
	GameServerStateStarting GameServerState = "Starting"
	// GameServerStateScheduled is for when we have determined that the Pod has been
	// scheduled in the cluster -- basically, we have a NodeName
	GameServerStateScheduled GameServerState = "Scheduled"
	// GameServerStateRequestReady is when the GameServer has declared that it is ready
	GameServerStateRequestReady GameServerState = "RequestReady"
	// GameServerStateReady is when a GameServer is ready to take connections
	// from Game clients
	GameServerStateReady GameServerState = "Ready"
	// GameServerStateShutdown is when the GameServer has shutdown and everything needs to be
	// deleted from the cluster
	GameServerStateShutdown GameServerState = "Shutdown"
	// GameServerStateError is when something has gone wrong with the Gameserver and
	// it cannot be resolved
	GameServerStateError GameServerState = "Error"
	// GameServerStateUnhealthy is when the GameServer has failed its health checks
	GameServerStateUnhealthy GameServerState = "Unhealthy"
	// GameServerStateReserved is for when a GameServer is reserved and therefore can be allocated but not removed
	GameServerStateReserved GameServerState = "Reserved"
	// GameServerStateAllocated is when the GameServer has been allocated to a session
	GameServerStateAllocated GameServerState = "Allocated"

	// Static PortPolicy means that the user defines the hostPort to be used
	// in the configuration.
	Static PortPolicy = "Static"
	// Dynamic PortPolicy means that the system will choose an open
	// port for the GameServer in question
	Dynamic PortPolicy = "Dynamic"
	// Passthrough dynamically sets the `containerPort` to the same value as the dynamically selected hostPort.
	// This will mean that users will need to lookup what port has been opened through the server side SDK.
	Passthrough PortPolicy = "Passthrough"

	// SdkServerLogLevelInfo will cause the SDK server to output all messages except for debug messages.
	SdkServerLogLevelInfo SdkServerLogLevel = "Info"
	// SdkServerLogLevelDebug will cause the SDK server to output all messages including debug messages.
	SdkServerLogLevelDebug SdkServerLogLevel = "Debug"
	// SdkServerLogLevelError will cause the SDK server to only output error messages.
	SdkServerLogLevelError SdkServerLogLevel = "Error"

	// RoleLabel is the label in which the Agones role is specified.
	// Pods from a GameServer will have the value "gameserver"
	RoleLabel = agones.GroupName + "/role"
	// GameServerLabelRole is the GameServer label value for RoleLabel
	GameServerLabelRole = "gameserver"
	// GameServerPodLabel is the label that the name of the GameServer
	// is set on the Pod the GameServer controls
	GameServerPodLabel = agones.GroupName + "/gameserver"
	// GameServerContainerAnnotation is the annotation that stores
	// which container is the container that runs the dedicated game server
	GameServerContainerAnnotation = agones.GroupName + "/container"
	// DevAddressAnnotation is an annotation to indicate that a GameServer hosted outside of Agones.
	// A locally hosted GameServer is not managed by Agones it is just simply registered.
	DevAddressAnnotation = "agones.dev/dev-address"
	// GameServerReadyContainerIDAnnotation is an annotation that is set on the GameServer
	// becomes ready, so we can track when restarts should occur and when a GameServer
	// should be moved to Unhealthy.
	GameServerReadyContainerIDAnnotation = agones.GroupName + "/ready-container-id"
)

var (
	// GameServerRolePodSelector is the selector to get all GameServer Pods
	GameServerRolePodSelector = labels.SelectorFromSet(labels.Set{RoleLabel: GameServerLabelRole})
)

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// GameServer is the data structure for a GameServer resource.
// It is worth noting that while there is a `GameServerStatus` Status entry for the `GameServer`, it is not
// defined as a subresource - unlike `Fleet` and other Agones resources.
// This is so that we can retain the ability to change multiple aspects of a `GameServer` in a single atomic operation,
// which is particularly useful for operations such as allocation.
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
	// Ports are the array of ports that can be exposed via the game server
	Ports []GameServerPort `json:"ports"`
	// Health configures health checking
	Health Health `json:"health,omitempty"`
	// Scheduling strategy. Defaults to "Packed"
	Scheduling apis.SchedulingStrategy `json:"scheduling,omitempty"`
	// SdkServer specifies parameters for the Agones SDK Server sidecar container
	SdkServer SdkServer `json:"sdkServer,omitempty"`
	// Template describes the Pod that will be created for the GameServer
	Template corev1.PodTemplateSpec `json:"template"`
}

// GameServerState is the state for the GameServer
type GameServerState string

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

// GameServerPort defines a set of Ports that
// are to be exposed via the GameServer
type GameServerPort struct {
	// Name is the descriptive name of the port
	Name string `json:"name,omitempty"`
	// PortPolicy defines the policy for how the HostPort is populated.
	// Dynamic port will allocate a HostPort within the selected MIN_PORT and MAX_PORT range passed to the controller
	// at installation time.
	// When `Static` portPolicy is specified, `HostPort` is required, to specify the port that game clients will
	// connect to
	PortPolicy PortPolicy `json:"portPolicy,omitempty"`
	// ContainerPort is the port that is being opened on the game server process
	ContainerPort int32 `json:"containerPort,omitempty"`
	// HostPort the port exposed on the host for clients to connect to
	HostPort int32 `json:"hostPort,omitempty"`
	// Protocol is the network protocol being used. Defaults to UDP. TCP is the only other option
	Protocol corev1.Protocol `json:"protocol,omitempty"`
}

// SdkServerLogLevel is the log level for SDK server (sidecar) logs
type SdkServerLogLevel string

// SdkServer specifies parameters for the Agones SDK Server sidecar container
type SdkServer struct {
	// LogLevel for SDK server (sidecar) logs. Defaults to "Info"
	LogLevel SdkServerLogLevel `json:"logLevel,omitempty"`
	// GRPCPort is the port on which the SDK Server binds the gRPC server to accept incoming connections
	GRPCPort int32 `json:"grpcPort,omitempty"`
	// HTTPPort is the port on which the SDK Server binds the HTTP gRPC gateway server to accept incoming connections
	HTTPPort int32 `json:"httpPort,omitempty"`
}

// GameServerStatus is the status for a GameServer resource
type GameServerStatus struct {
	// GameServerState is the current state of a GameServer, e.g. Creating, Starting, Ready, etc
	State         GameServerState        `json:"state"`
	Ports         []GameServerStatusPort `json:"ports"`
	Address       string                 `json:"address"`
	NodeName      string                 `json:"nodeName"`
	ReservedUntil *metav1.Time           `json:"reservedUntil"`
}

// GameServerStatusPort shows the port that was allocated to a
// GameServer.
type GameServerStatusPort struct {
	Name string `json:"name,omitempty"`
	Port int32  `json:"port"`
}

// ApplyDefaults applies default values to the GameServer if they are not already populated
func (gs *GameServer) ApplyDefaults() {
	// VersionAnnotation is the annotation that stores
	// the version of sdk which runs in a sidecar
	if gs.ObjectMeta.Annotations == nil {
		gs.ObjectMeta.Annotations = map[string]string{}
	}
	gs.ObjectMeta.Annotations[VersionAnnotation] = pkg.Version
	gs.ObjectMeta.Finalizers = append(gs.ObjectMeta.Finalizers, agones.GroupName)

	gs.Spec.ApplyDefaults()
	gs.applyStateDefaults()
}

// ApplyDefaults applies default values to the GameServerSpec if they are not already populated
func (gss *GameServerSpec) ApplyDefaults() {
	gss.applyContainerDefaults()
	gss.applyPortDefaults()
	gss.applyHealthDefaults()
	gss.applySchedulingDefaults()
	gss.applySdkServerDefaults()
}

// applySdkServerDefaults applies the default log level ("Info") for the sidecar
func (gss *GameServerSpec) applySdkServerDefaults() {
	if gss.SdkServer.LogLevel == "" {
		gss.SdkServer.LogLevel = SdkServerLogLevelInfo
	}
	if gss.SdkServer.GRPCPort == 0 {
		gss.SdkServer.GRPCPort = 9357
	}
	if gss.SdkServer.HTTPPort == 0 {
		gss.SdkServer.HTTPPort = 9358
	}
}

// applyContainerDefaults applies the container defaults
func (gss *GameServerSpec) applyContainerDefaults() {
	if len(gss.Template.Spec.Containers) == 1 {
		gss.Container = gss.Template.Spec.Containers[0].Name
	}
}

// applyHealthDefaults applies health checking defaults
func (gss *GameServerSpec) applyHealthDefaults() {
	if !gss.Health.Disabled {
		if gss.Health.PeriodSeconds <= 0 {
			gss.Health.PeriodSeconds = 5
		}
		if gss.Health.FailureThreshold <= 0 {
			gss.Health.FailureThreshold = 3
		}
		if gss.Health.InitialDelaySeconds <= 0 {
			gss.Health.InitialDelaySeconds = 5
		}
	}
}

// applyStateDefaults applies state defaults
func (gs *GameServer) applyStateDefaults() {
	if gs.Status.State == "" {
		gs.Status.State = GameServerStateCreating
		// applyStateDefaults() should be called after applyPortDefaults()
		if gs.HasPortPolicy(Dynamic) || gs.HasPortPolicy(Passthrough) {
			gs.Status.State = GameServerStatePortAllocation
		}
	}
}

// applyPortDefaults applies default values for all ports
func (gss *GameServerSpec) applyPortDefaults() {
	for i, p := range gss.Ports {
		// basic spec
		if p.PortPolicy == "" {
			gss.Ports[i].PortPolicy = Dynamic
		}

		if p.Protocol == "" {
			gss.Ports[i].Protocol = "UDP"
		}
	}
}

func (gss *GameServerSpec) applySchedulingDefaults() {
	if gss.Scheduling == "" {
		gss.Scheduling = apis.Packed
	}
}

// Validate validates the GameServerSpec configuration.
// devAddress is a specific IP address used for local Gameservers, for fleets "" is used
// If a GameServer Spec is invalid there will be > 0 values in
// the returned array
func (gss *GameServerSpec) Validate(devAddress string) ([]metav1.StatusCause, bool) {
	var causes []metav1.StatusCause
	if devAddress != "" {
		// verify that the value is a valid IP address.
		if net.ParseIP(devAddress) == nil {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Field:   fmt.Sprintf("annotations.%s", DevAddressAnnotation),
				Message: fmt.Sprintf("Value '%s' of annotation '%s' must be a valid IP address.", DevAddressAnnotation, devAddress),
			})
		}

		for _, p := range gss.Ports {
			if p.HostPort == 0 {
				causes = append(causes, metav1.StatusCause{
					Type:    metav1.CauseTypeFieldValueRequired,
					Field:   fmt.Sprintf("%s.hostPort", p.Name),
					Message: fmt.Sprintf("HostPort is required if GameServer is annotated with %s", DevAddressAnnotation),
				})
			}
			if p.PortPolicy != Static {
				causes = append(causes, metav1.StatusCause{
					Type:    metav1.CauseTypeFieldValueRequired,
					Field:   fmt.Sprintf("%s.portPolicy", p.Name),
					Message: fmt.Sprint(ErrPortPolicyStatic),
				})
			}
		}
	} else {
		// make sure a name is specified when there is multiple containers in the pod.
		if gss.Container == "" && len(gss.Template.Spec.Containers) > 1 {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Field:   "container",
				Message: ErrContainerRequired,
			})
		}

		// no host port when using dynamic PortPolicy
		for _, p := range gss.Ports {
			if p.PortPolicy == Dynamic || p.PortPolicy == Static {
				if p.ContainerPort <= 0 {
					causes = append(causes, metav1.StatusCause{
						Type:    metav1.CauseTypeFieldValueInvalid,
						Field:   fmt.Sprintf("%s.containerPort", p.Name),
						Message: ErrContainerPortRequired,
					})
				}
			}

			if p.PortPolicy == Passthrough && p.ContainerPort > 0 {
				causes = append(causes, metav1.StatusCause{
					Type:    metav1.CauseTypeFieldValueInvalid,
					Field:   fmt.Sprintf("%s.containerPort", p.Name),
					Message: ErrContainerPortPassthrough,
				})
			}

			if p.HostPort > 0 && (p.PortPolicy == Dynamic || p.PortPolicy == Passthrough) {
				causes = append(causes, metav1.StatusCause{
					Type:    metav1.CauseTypeFieldValueInvalid,
					Field:   fmt.Sprintf("%s.hostPort", p.Name),
					Message: ErrHostPortDynamic,
				})
			}
		}

		// make sure the container value points to a valid container
		_, _, err := gss.FindGameServerContainer()
		if err != nil {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Field:   "container",
				Message: err.Error(),
			})
		}
	}
	objMetaCauses := validateObjectMeta(&gss.Template.ObjectMeta)
	if len(objMetaCauses) > 0 {
		causes = append(causes, objMetaCauses...)
	}

	return causes, len(causes) == 0
}

// Validate validates the GameServer configuration.
// If a GameServer is invalid there will be > 0 values in
// the returned array
func (gs *GameServer) Validate() ([]metav1.StatusCause, bool) {
	causes := validateName(gs)

	// make sure the host port is specified if this is a development server
	devAddress, _ := gs.GetDevAddress()
	gssCauses, _ := gs.Spec.Validate(devAddress)
	causes = append(causes, gssCauses...)
	return causes, len(causes) == 0
}

// GetDevAddress returns the address for game server.
func (gs *GameServer) GetDevAddress() (string, bool) {
	devAddress, hasDevAddress := gs.ObjectMeta.Annotations[DevAddressAnnotation]
	return devAddress, hasDevAddress
}

// IsDeletable returns false if the server is currently allocated/reserved and is not already in the
// process of being deleted
func (gs *GameServer) IsDeletable() bool {
	if gs.Status.State == GameServerStateAllocated || gs.Status.State == GameServerStateReserved {
		return !gs.ObjectMeta.DeletionTimestamp.IsZero()
	}

	return true
}

// IsBeingDeleted returns true if the server is in the process of being deleted.
func (gs *GameServer) IsBeingDeleted() bool {
	return !gs.ObjectMeta.DeletionTimestamp.IsZero() || gs.Status.State == GameServerStateShutdown
}

// IsBeforeReady returns true if the GameServer Status has yet to move to or past the Ready
// state in its lifecycle, such as Allocated or Reserved, or any of the Error/Unhealthy states
func (gs *GameServer) IsBeforeReady() bool {
	switch gs.Status.State {
	case GameServerStatePortAllocation:
		return true
	case GameServerStateCreating:
		return true
	case GameServerStateStarting:
		return true
	case GameServerStateScheduled:
		return true
	case GameServerStateRequestReady:
		return true
	}

	return false
}

// FindGameServerContainer returns the container that is specified in
// gameServer.Spec.Container. Returns the index and the value.
// Returns an error if not found
func (gss *GameServerSpec) FindGameServerContainer() (int, corev1.Container, error) {
	for i, c := range gss.Template.Spec.Containers {
		if c.Name == gss.Container {
			return i, c, nil
		}
	}

	return -1, corev1.Container{}, errors.Errorf("Could not find a container named %s", gss.Container)
}

// FindGameServerContainer returns the container that is specified in
// gameServer.Spec.Container. Returns the index and the value.
// Returns an error if not found
func (gs *GameServer) FindGameServerContainer() (int, corev1.Container, error) {
	return gs.Spec.FindGameServerContainer()
}

// ApplyToPodGameServerContainer applies func(v1.Container) to the pod's gameserver container
func (gs *GameServer) ApplyToPodGameServerContainer(pod *corev1.Pod, f func(corev1.Container) corev1.Container) *corev1.Pod {
	for i, c := range pod.Spec.Containers {
		if c.Name == gs.Spec.Container {
			c = f(c)
			pod.Spec.Containers[i] = c
			break
		}
	}

	return pod
}

// Pod creates a new Pod from the PodTemplateSpec
// attached to the GameServer resource
func (gs *GameServer) Pod(sidecars ...corev1.Container) (*corev1.Pod, error) {
	pod := &corev1.Pod{
		ObjectMeta: *gs.Spec.Template.ObjectMeta.DeepCopy(),
		Spec:       *gs.Spec.Template.Spec.DeepCopy(),
	}

	gs.podObjectMeta(pod)

	i, gsContainer, err := gs.FindGameServerContainer()
	// this shouldn't happen, but if it does.
	if err != nil {
		return pod, err
	}

	for _, p := range gs.Spec.Ports {
		cp := corev1.ContainerPort{
			ContainerPort: p.ContainerPort,
			HostPort:      p.HostPort,
			Protocol:      p.Protocol,
		}
		gsContainer.Ports = append(gsContainer.Ports, cp)
	}
	pod.Spec.Containers[i] = gsContainer

	pod.Spec.Containers = append(pod.Spec.Containers, sidecars...)

	gs.podScheduling(pod)

	return pod, nil
}

// podObjectMeta configures the pod ObjectMeta details
func (gs *GameServer) podObjectMeta(pod *corev1.Pod) {
	pod.ObjectMeta.GenerateName = ""
	// Pods inherit the name of their gameserver. It's safe since there's
	// a guarantee that pod won't outlive its parent.
	pod.ObjectMeta.Name = gs.ObjectMeta.Name
	// Pods for GameServers need to stay in the same namespace
	pod.ObjectMeta.Namespace = gs.ObjectMeta.Namespace
	// Make sure these are blank, just in case
	pod.ObjectMeta.ResourceVersion = ""
	pod.ObjectMeta.UID = ""
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

	if gs.Spec.Scheduling == apis.Packed {
		// This means that the autoscaler cannot remove the Node that this Pod is on.
		// (and evict the Pod in the process)
		pod.ObjectMeta.Annotations["cluster-autoscaler.kubernetes.io/safe-to-evict"] = "false"
	}

	// Add Agones version into Pod Annotations
	pod.ObjectMeta.Annotations[VersionAnnotation] = pkg.Version
	if gs.ObjectMeta.Annotations == nil {
		gs.ObjectMeta.Annotations = make(map[string]string, 1)
	}
}

// podScheduling applies the Fleet scheduling strategy to the passed in Pod
// this sets the a PreferredDuringSchedulingIgnoredDuringExecution for GameServer
// pods to a host topology. Basically doing a half decent job of packing GameServer
// pods together.
func (gs *GameServer) podScheduling(pod *corev1.Pod) {
	if gs.Spec.Scheduling == apis.Packed {
		if pod.Spec.Affinity == nil {
			pod.Spec.Affinity = &corev1.Affinity{}
		}
		if pod.Spec.Affinity.PodAffinity == nil {
			pod.Spec.Affinity.PodAffinity = &corev1.PodAffinity{}
		}

		wpat := corev1.WeightedPodAffinityTerm{
			Weight: 100,
			PodAffinityTerm: corev1.PodAffinityTerm{
				TopologyKey:   "kubernetes.io/hostname",
				LabelSelector: &metav1.LabelSelector{MatchLabels: map[string]string{RoleLabel: GameServerLabelRole}},
			},
		}

		pod.Spec.Affinity.PodAffinity.PreferredDuringSchedulingIgnoredDuringExecution = append(pod.Spec.Affinity.PodAffinity.PreferredDuringSchedulingIgnoredDuringExecution, wpat)
	}
}

// DisableServiceAccount disables the service account for the gameserver container
func (gs *GameServer) DisableServiceAccount(pod *corev1.Pod) {
	// gameservers don't get access to the k8s api.
	emptyVol := corev1.Volume{Name: "empty", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}}
	pod.Spec.Volumes = append(pod.Spec.Volumes, emptyVol)
	mount := corev1.VolumeMount{MountPath: "/var/run/secrets/kubernetes.io/serviceaccount", Name: emptyVol.Name, ReadOnly: true}

	gs.ApplyToPodGameServerContainer(pod, func(c corev1.Container) corev1.Container {
		c.VolumeMounts = append(c.VolumeMounts, mount)

		return c
	})
}

// HasPortPolicy checks if there is a port with a given
// PortPolicy
func (gs *GameServer) HasPortPolicy(policy PortPolicy) bool {
	for _, p := range gs.Spec.Ports {
		if p.PortPolicy == policy {
			return true
		}
	}
	return false
}

// Status returns a GameServerSatusPort for this GameServerPort
func (p GameServerPort) Status() GameServerStatusPort {
	return GameServerStatusPort{Name: p.Name, Port: p.HostPort}
}

// CountPorts returns the number of
// ports that match condition function
func (gs *GameServer) CountPorts(f func(policy PortPolicy) bool) int {
	count := 0
	for _, p := range gs.Spec.Ports {
		if f(p.PortPolicy) {
			count++
		}
	}
	return count
}

// Patch creates a JSONPatch to move the current GameServer
// to the passed in delta GameServer
func (gs *GameServer) Patch(delta *GameServer) ([]byte, error) {
	var result []byte

	oldJSON, err := json.Marshal(gs)
	if err != nil {
		return result, errors.Wrapf(err, "error marshalling to json current GameServer %s", gs.ObjectMeta.Name)
	}

	newJSON, err := json.Marshal(delta)
	if err != nil {
		return result, errors.Wrapf(err, "error marshalling to json delta GameServer %s", delta.ObjectMeta.Name)
	}

	patch, err := jsonpatch.CreatePatch(oldJSON, newJSON)
	if err != nil {
		return result, errors.Wrapf(err, "error creating patch for GameServer %s", gs.ObjectMeta.Name)
	}

	result, err = json.Marshal(patch)
	return result, errors.Wrapf(err, "error creating json for patch for GameServer %s", gs.ObjectMeta.Name)
}

// IsUnhealthy returns true if the GameServer is Unhealthy or in Error state
func (gs *GameServer) IsUnhealthy() bool {
	state := gs.Status.State
	return state == GameServerStateUnhealthy || state == GameServerStateError
}
