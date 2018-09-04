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
import "k8s.io/apimachinery/pkg/util/intstr"

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// FleetAutoscaler is the data structure for a FleetAutoscaler resource
type FleetAutoscaler struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   FleetAutoscalerSpec   `json:"spec"`
	Status FleetAutoscalerStatus `json:"status"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// FleetAutoscalerList is a list of Fleet Scaler resources
type FleetAutoscalerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []FleetAutoscaler `json:"items"`
}

// FleetAutoscalerSpec is the spec for a Fleet Scaler
type FleetAutoscalerSpec struct {
	FleetName string `json:"fleetName"`

	// Autoscaling policy
	Policy FleetAutoscalerPolicy `json:"policy"`
}

// FleetAutoscalerPolicy describes how to scale a fleet
type FleetAutoscalerPolicy struct {
	// Type of autoscaling policy.
	Type FleetAutoscalerPolicyType `json:"type"`

	// Buffer policy config params. Present only if FleetAutoscalerPolicyType = Buffer.
	// +optional
	Buffer *BufferPolicy `json:"buffer,omitempty"`
}

type FleetAutoscalerPolicyType string

const (
	// Kill all existing pods before creating new ones.
	BufferPolicyType FleetAutoscalerPolicyType = "Buffer"
)

// BufferPolicy controls the desired behavior of the buffer policy.
type BufferPolicy struct {
	// MaxReplicas is the maximum amount of replicas that the fleet may have.
	// It must be bigger than both MinReplicas and BufferSize
	MaxReplicas int32 `json:"maxReplicas"`

	// MinReplicas is the minimum amount of replicas that the fleet must have
	// If zero, it is ignored.
	// If non zero, it must be smaller than MaxReplicas and bigger than BufferSize
	MinReplicas int32 `json:"minReplicas"`

	// BufferSize defines how many replicas the autoscaler tries to have ready all the time
	// Value can be an absolute number (ex: 5) or a percentage of desired gs instances (ex: 15%)
	// Absolute number is calculated from percentage by rounding up.
	// Example: when this is set to 20%, the autoscaler will make sure that 20%
	//   of the fleet's game server replicas are ready. When this is set to 20,
	//   the autoscaler will make sure that there are 20 available game servers
	// Must be bigger than 0
	// Note: by "ready" we understand in this case "non-allocated"; this is done to ensure robustness
	//       and computation stability in different edge case (fleet just created, not enough
	//       capacity in the cluster etc)
	BufferSize intstr.IntOrString `json:"bufferSize"`
}

// FleetAutoscalerStatus defines the current status of a FleetAutoscaler
type FleetAutoscalerStatus struct {
	// CurrentReplicas is the current number of gameserver replicas
	// of the fleet managed by this autoscaler, as last seen by the autoscaler
	CurrentReplicas int32 `json:"currentReplicas"`

	// DesiredReplicas is the desired number of gameserver replicas
	// of the fleet managed by this autoscaler, as last calculated by the autoscaler
	DesiredReplicas int32 `json:"desiredReplicas"`

	// lastScaleTime is the last time the FleetAutoscaler scaled the attached fleet,
	// +optional
	LastScaleTime *metav1.Time `json:"lastScaleTime"`

	// AbleToScale indicates that we can access the target fleet
	AbleToScale bool `json:"ableToScale"`

	// ScalingLimited indicates that the calculated scale would be above or below the range
	// defined by MinReplicas and MaxReplicas, and has thus been capped.
	ScalingLimited bool `json:"scalingLimited"`
}

// ValidateUpdate validates when an update occurs
func (fas *FleetAutoscaler) ValidateUpdate(new *FleetAutoscaler, causes []metav1.StatusCause) []metav1.StatusCause {
	if fas.Spec.FleetName != new.Spec.FleetName {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Field:   "fleetName",
			Message: "fleetName cannot be updated",
		})
	}

	return new.ValidateAutoScalingSettings(causes)
}

//ValidateAutoScalingSettings validates the FleetAutoscaler scaling settings
func (fas *FleetAutoscaler) ValidateAutoScalingSettings(causes []metav1.StatusCause) []metav1.StatusCause {
	if fas.Spec.Policy.Type == BufferPolicyType {
		causes = fas.Spec.Policy.Buffer.ValidateAutoScalingBufferPolicy(causes)
	}
	return causes
}

//ValidateAutoScalingSettings validates the FleetAutoscaler Buffer policy settings
func (b *BufferPolicy) ValidateAutoScalingBufferPolicy(causes []metav1.StatusCause) []metav1.StatusCause {
	if b == nil {
		return append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Field:   "buffer",
			Message: "Buffer policy config params are missing",
		})
	}
	if b.MinReplicas > b.MaxReplicas {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Field:   "minReplicas",
			Message: "minReplicas is bigger than maxReplicas",
		})
	}
	if b.BufferSize.Type == intstr.Int {
		if b.BufferSize.IntValue() <= 0 {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Field:   "bufferSize",
				Message: "bufferSize must be bigger than 0",
			})
		}
		if b.MaxReplicas < int32(b.BufferSize.IntValue()) {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Field:   "maxReplicas",
				Message: "maxReplicas must be bigger than bufferSize",
			})
		}
		if b.MinReplicas != 0 && b.MinReplicas < int32(b.BufferSize.IntValue()) {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Field:   "minReplicas",
				Message: "minReplicas is smaller than bufferSize",
			})
		}
	} else {
		r, err := intstr.GetValueFromIntOrPercent(&b.BufferSize, 100, true)
		if err != nil || r < 1 || r > 99 {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Field:   "bufferSize",
				Message: "bufferSize does not have a valid percentage value (1%-99%)",
			})
		}
	}
	return causes
}
