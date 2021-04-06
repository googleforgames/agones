// Copyright 2018 Google LLC All Rights Reserved.
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
	"agones.dev/agones/pkg"
	"agones.dev/agones/pkg/apis"
	"agones.dev/agones/pkg/apis/agones"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	// FleetNameLabel is the label that the name of the Fleet
	// is set to on GameServerSet and GameServer  the Fleet controls
	FleetNameLabel = agones.GroupName + "/fleet"
)

// +genclient
// +genclient:method=GetScale,verb=get,subresource=scale,result=k8s.io/api/autoscaling/v1.Scale
// +genclient:method=UpdateScale,verb=update,subresource=scale,input=k8s.io/api/autoscaling/v1.Scale,result=k8s.io/api/autoscaling/v1.Scale
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Fleet is the data structure for a Fleet resource
type Fleet struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   FleetSpec   `json:"spec"`
	Status FleetStatus `json:"status"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// FleetList is a list of Fleet resources
type FleetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []Fleet `json:"items"`
}

// FleetSpec is the spec for a Fleet
type FleetSpec struct {
	// Replicas are the number of GameServers that should be in this set. Defaults to 0.
	Replicas int32 `json:"replicas"`
	// Deployment strategy
	Strategy appsv1.DeploymentStrategy `json:"strategy"`
	// Scheduling strategy. Defaults to "Packed".
	Scheduling apis.SchedulingStrategy `json:"scheduling"`
	// Template the GameServer template to apply for this Fleet
	Template GameServerTemplateSpec `json:"template"`
}

// FleetStatus is the status of a Fleet
type FleetStatus struct {
	// Replicas the total number of current GameServer replicas
	Replicas int32 `json:"replicas"`
	// ReadyReplicas are the number of Ready GameServer replicas
	ReadyReplicas int32 `json:"readyReplicas"`
	// ReservedReplicas are the total number of Reserved GameServer replicas in this fleet.
	// Reserved instances won't be deleted on scale down, but won't cause an autoscaler to scale up.
	ReservedReplicas int32 `json:"reservedReplicas"`
	// AllocatedReplicas are the number of Allocated GameServer replicas
	AllocatedReplicas int32 `json:"allocatedReplicas"`
	// [Stage:Alpha]
	// [FeatureFlag:PlayerTracking]
	// Players are the current total player capacity and count for this Fleet
	// +optional
	Players *AggregatedPlayerStatus `json:"players,omitempty"`
}

// GameServerSet returns a single GameServerSet for this Fleet definition
func (f *Fleet) GameServerSet() *GameServerSet {
	gsSet := &GameServerSet{
		ObjectMeta: *f.Spec.Template.ObjectMeta.DeepCopy(),
		Spec: GameServerSetSpec{
			Template:   f.Spec.Template,
			Scheduling: f.Spec.Scheduling,
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

	gsSet.ObjectMeta.Labels[FleetNameLabel] = f.ObjectMeta.Name

	return gsSet
}

// ApplyDefaults applies default values to the Fleet
func (f *Fleet) ApplyDefaults() {
	if f.Spec.Strategy.Type == "" {
		f.Spec.Strategy.Type = appsv1.RollingUpdateDeploymentStrategyType
	}

	if f.Spec.Scheduling == "" {
		f.Spec.Scheduling = apis.Packed
	}

	if f.Spec.Strategy.Type == appsv1.RollingUpdateDeploymentStrategyType {
		if f.Spec.Strategy.RollingUpdate == nil {
			f.Spec.Strategy.RollingUpdate = &appsv1.RollingUpdateDeployment{}
		}

		def := intstr.FromString("25%")
		if f.Spec.Strategy.RollingUpdate.MaxSurge == nil {
			f.Spec.Strategy.RollingUpdate.MaxSurge = &def
		}
		if f.Spec.Strategy.RollingUpdate.MaxUnavailable == nil {
			f.Spec.Strategy.RollingUpdate.MaxUnavailable = &def
		}
	}
	// Add Agones version into Fleet Annotations
	if f.ObjectMeta.Annotations == nil {
		f.ObjectMeta.Annotations = make(map[string]string, 1)
	}
	f.ObjectMeta.Annotations[VersionAnnotation] = pkg.Version
}

// GetGameServerSpec get underlying Gameserver specification
func (f *Fleet) GetGameServerSpec() *GameServerSpec {
	return &f.Spec.Template.Spec
}

func (f *Fleet) validateRollingUpdate(value *intstr.IntOrString, causes *[]metav1.StatusCause, parameter string) {
	r, err := intstr.GetValueFromIntOrPercent(value, 100, true)
	if value.Type == intstr.String {
		if err != nil || r < 1 || r > 99 {
			*causes = append(*causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Field:   parameter,
				Message: parameter + " does not have a valid percentage value (1%-99%)",
			})
		}
	} else {
		if r < 1 {
			*causes = append(*causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Field:   parameter,
				Message: parameter + " does not have a valid integer value (>1)",
			})
		}
	}
}

// Validate validates the Fleet configuration.
// If a Fleet is invalid there will be > 0 values in
// the returned array
func (f *Fleet) Validate() ([]metav1.StatusCause, bool) {
	causes := validateName(f)

	if f.Spec.Strategy.Type == appsv1.RollingUpdateDeploymentStrategyType {
		f.validateRollingUpdate(f.Spec.Strategy.RollingUpdate.MaxUnavailable, &causes, "MaxUnavailable")
		f.validateRollingUpdate(f.Spec.Strategy.RollingUpdate.MaxSurge, &causes, "MaxSurge")
	} else if f.Spec.Strategy.Type != appsv1.RecreateDeploymentStrategyType {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Field:   "Type",
			Message: "Strategy Type should be one of: RollingUpdate, Recreate.",
		})
	}
	// check Gameserver specification in a Fleet
	gsCauses := validateGSSpec(f)
	if len(gsCauses) > 0 {
		causes = append(causes, gsCauses...)
	}
	objMetaCauses := validateObjectMeta(&f.Spec.Template.ObjectMeta)
	if len(objMetaCauses) > 0 {
		causes = append(causes, objMetaCauses...)
	}

	return causes, len(causes) == 0
}

// UpperBoundReplicas returns whichever is smaller,
// the value i, or the f.Spec.Replicas.
func (f *Fleet) UpperBoundReplicas(i int32) int32 {
	if i > f.Spec.Replicas {
		return f.Spec.Replicas
	}
	return i
}

// LowerBoundReplicas returns 0 (the minimum value for
// replicas) if i is < 0
func (f *Fleet) LowerBoundReplicas(i int32) int32 {
	if i < 0 {
		return 0
	}
	return i
}

// SumStatusAllocatedReplicas returns the total number of
// Status.AllocatedReplicas in the list of GameServerSets
func SumStatusAllocatedReplicas(list []*GameServerSet) int32 {
	total := int32(0)
	for _, gsSet := range list {
		total += gsSet.Status.AllocatedReplicas
	}

	return total
}

// SumStatusReplicas returns the total number of
// Status.Replicas in the list of GameServerSets
func SumStatusReplicas(list []*GameServerSet) int32 {
	total := int32(0)
	for _, gsSet := range list {
		total += gsSet.Status.Replicas
	}

	return total
}

// SumSpecReplicas returns the total number of
// Spec.Replicas in the list of GameServerSets
func SumSpecReplicas(list []*GameServerSet) int32 {
	total := int32(0)
	for _, gsSet := range list {
		if gsSet != nil {
			total += gsSet.Spec.Replicas
		}
	}

	return total
}

// GetReadyReplicaCountForGameServerSets returns the total number of
// Status.ReadyReplicas in the list of GameServerSets
func GetReadyReplicaCountForGameServerSets(gss []*GameServerSet) int32 {
	totalReadyReplicas := int32(0)
	for _, gss := range gss {
		if gss != nil {
			totalReadyReplicas += gss.Status.ReadyReplicas
		}
	}
	return totalReadyReplicas
}
