// Copyright 2019 Google LLC All Rights Reserved.
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
	"crypto/x509"
	"net/url"

	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	"agones.dev/agones/pkg/util/runtime"
	admregv1 "k8s.io/api/admissionregistration/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// +genclient
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
	// [Stage:Beta]
	// [FeatureFlag:CustomFasSyncInterval]
	// Sync defines when FleetAutoscalers runs autoscaling
	// +optional
	Sync *FleetAutoscalerSync `json:"sync,omitempty"`
}

// FleetAutoscalerPolicy describes how to scale a fleet
type FleetAutoscalerPolicy struct {
	// Type of autoscaling policy.
	Type FleetAutoscalerPolicyType `json:"type"`

	// Buffer policy config params. Present only if FleetAutoscalerPolicyType = Buffer.
	// +optional
	Buffer *BufferPolicy `json:"buffer,omitempty"`
	// Webhook policy config params. Present only if FleetAutoscalerPolicyType = Webhook.
	// +optional
	Webhook *WebhookPolicy `json:"webhook,omitempty"`
}

// FleetAutoscalerPolicyType is the policy for autoscaling
// for a given Fleet
type FleetAutoscalerPolicyType string

// FleetAutoscalerSync describes when to sync a fleet
type FleetAutoscalerSync struct {
	// Type of autoscaling sync.
	Type FleetAutoscalerSyncType `json:"type"`

	// FixedInterval config params. Present only if FleetAutoscalerSyncType = FixedInterval.
	// +optional
	FixedInterval FixedIntervalSync `json:"fixedInterval"`
}

// FleetAutoscalerSyncType is the sync strategy for a given Fleet
type FleetAutoscalerSyncType string

const (
	// BufferPolicyType FleetAutoscalerPolicyType is a simple buffering strategy for Ready
	// GameServers
	BufferPolicyType FleetAutoscalerPolicyType = "Buffer"
	// WebhookPolicyType is a simple webhook strategy used for horizontal fleet scaling
	// GameServers
	WebhookPolicyType FleetAutoscalerPolicyType = "Webhook"
	// FixedIntervalSyncType is a simple fixed interval based strategy for trigger autoscaling
	FixedIntervalSyncType FleetAutoscalerSyncType = "FixedInterval"

	defaultIntervalSyncSeconds = 30
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

// WebhookPolicy controls the desired behavior of the webhook policy.
// It contains the description of the webhook autoscaler service
// used to form url which is accessible inside the cluster
type WebhookPolicy admregv1.WebhookClientConfig

// FixedIntervalSync controls the desired behavior of the fixed interval based sync.
type FixedIntervalSync struct {
	// Seconds defines how often we run fleet autoscaling in seconds
	Seconds int32 `json:"seconds"`
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

// FleetAutoscaleRequest defines the request to webhook autoscaler endpoint
type FleetAutoscaleRequest struct {
	// UID is an identifier for the individual request/response. It allows us to distinguish instances of requests which are
	// otherwise identical (parallel requests, requests when earlier requests did not modify etc)
	// The UID is meant to track the round trip (request/response) between the Autoscaler and the WebHook, not the user request.
	// It is suitable for correlating log entries between the webhook and apiserver, for either auditing or debugging.
	UID types.UID `json:"uid"`
	// Name is the name of the Fleet being scaled
	Name string `json:"name"`
	// Namespace is the namespace associated with the request (if any).
	Namespace string `json:"namespace"`
	// The Fleet's status values
	Status agonesv1.FleetStatus `json:"status"`
}

// FleetAutoscaleResponse defines the response of webhook autoscaler endpoint
type FleetAutoscaleResponse struct {
	// UID is an identifier for the individual request/response.
	// This should be copied over from the corresponding FleetAutoscaleRequest.
	UID types.UID `json:"uid"`
	// Set to false if no scaling should occur to the Fleet
	Scale bool `json:"scale"`
	// The targeted replica count
	Replicas int32 `json:"replicas"`
}

// FleetAutoscaleReview is passed to the webhook with a populated Request value,
// and then returned with a populated Response.
type FleetAutoscaleReview struct {
	Request  *FleetAutoscaleRequest  `json:"request"`
	Response *FleetAutoscaleResponse `json:"response"`
}

// Validate validates the FleetAutoscaler scaling settings
func (fas *FleetAutoscaler) Validate(causes []metav1.StatusCause) []metav1.StatusCause {
	switch fas.Spec.Policy.Type {
	case BufferPolicyType:
		causes = fas.Spec.Policy.Buffer.ValidateBufferPolicy(causes)

	case WebhookPolicyType:
		causes = fas.Spec.Policy.Webhook.ValidateWebhookPolicy(causes)
	}

	if runtime.FeatureEnabled(runtime.FeatureCustomFasSyncInterval) && fas.Spec.Sync != nil {
		causes = fas.Spec.Sync.FixedInterval.ValidateFixedIntervalSync(causes)
	}
	return causes
}

// ValidateWebhookPolicy validates the FleetAutoscaler Webhook policy settings
func (w *WebhookPolicy) ValidateWebhookPolicy(causes []metav1.StatusCause) []metav1.StatusCause {
	if w == nil {
		return append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Field:   "webhook",
			Message: "webhook policy config params are missing",
		})
	}
	if w.Service == nil && w.URL == nil {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueNotFound,
			Field:   "url",
			Message: "url should be provided",
		})
	}
	if w.Service != nil && w.URL != nil {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueNotFound,
			Field:   "url",
			Message: "service and url cannot be used simultaneously",
		})
	}
	if w.CABundle != nil {
		rootCAs := x509.NewCertPool()
		// Check that CABundle provided is correctly encoded certificate
		if ok := rootCAs.AppendCertsFromPEM(w.CABundle); !ok {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Field:   "caBundle",
				Message: "CA Bundle is not valid",
			})
		}
	}
	if w.URL != nil {
		u, err := url.Parse(*w.URL)
		if err != nil {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Field:   "url",
				Message: "url is not valid",
			})
		} else if u.Scheme == "https" {
			if w.CABundle == nil {
				causes = append(causes, metav1.StatusCause{
					Type:    metav1.CauseTypeFieldValueNotFound,
					Field:   "caBundle",
					Message: "CABundle should be provided if HTTPS webhook is used",
				})
			}
		}

	}
	return causes
}

// ValidateBufferPolicy validates the FleetAutoscaler Buffer policy settings
func (b *BufferPolicy) ValidateBufferPolicy(causes []metav1.StatusCause) []metav1.StatusCause {
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
		// When there is no allocated gameservers in a fleet,
		// Fleetautoscaler would reduce size of a fleet to MinReplicas.
		// If we have 0 MinReplicas and 0 Allocated then Fleetautoscaler would set Ready Replicas to 0
		// and we will not be able to raise the number of GS in a Fleet above zero
		if b.MinReplicas < 1 {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Field:   "minReplicas",
				Message: "minReplicas should be above 0 when used with percentage value bufferSize",
			})
		}
	}
	return causes
}

// ValidateFixedIntervalSync validates the FixedIntervalSync settings
func (i *FixedIntervalSync) ValidateFixedIntervalSync(causes []metav1.StatusCause) []metav1.StatusCause {
	if i == nil {
		return append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Field:   "fixedInterval",
			Message: "fixedInterval config params are missing",
		})
	}
	if i.Seconds <= 0 {
		return append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Field:   "seconds",
			Message: "seconds should be bigger than 0",
		})
	}
	return causes
}

// ApplyDefaults applies default values to the FleetAutoscaler
func (fas *FleetAutoscaler) ApplyDefaults() {
	if runtime.FeatureEnabled(runtime.FeatureCustomFasSyncInterval) {
		if fas.Spec.Sync == nil {
			fas.Spec.Sync = &FleetAutoscalerSync{}
		}
		if fas.Spec.Sync.Type == "" {
			fas.Spec.Sync.Type = FixedIntervalSyncType
		}
		if fas.Spec.Sync.FixedInterval.Seconds == 0 {
			fas.Spec.Sync.FixedInterval.Seconds = defaultIntervalSyncSeconds
		}
	}
}
