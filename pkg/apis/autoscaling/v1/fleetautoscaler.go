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
	apimachineryvalidation "k8s.io/apimachinery/pkg/api/validation"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/validation/field"
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
	// [Stage:Beta]
	// [FeatureFlag:CountsAndLists]
	// Counter policy config params. Present only if FleetAutoscalerPolicyType = Counter.
	// +optional
	Counter *CounterPolicy `json:"counter,omitempty"`
	// [Stage:Beta]
	// [FeatureFlag:CountsAndLists]
	// List policy config params. Present only if FleetAutoscalerPolicyType = List.
	// +optional
	List *ListPolicy `json:"list,omitempty"`
	// [Stage:Beta]
	// [FeatureFlag:ChainFleetAutoscaler]
	// Chain policy config params. Present only if FleetAutoscalerPolicyType = Chain.
	// +optional
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
	// [Stage:Beta]
	// [FeatureFlag:CountsAndLists]
	// CounterPolicyType is for Counter based fleet autoscaling
	// nolint:revive // Linter contains comment doesn't start with CounterPolicyType
	CounterPolicyType FleetAutoscalerPolicyType = "Counter"
	// [Stage:Beta]
	// [FeatureFlag:CountsAndLists]
	// ListPolicyType is for List based fleet autoscaling
	// nolint:revive // Linter contains comment doesn't start with ListPolicyType
	ListPolicyType FleetAutoscalerPolicyType = "List"
	// [Stage:Beta]
	// [FeatureFlag:ChainFleetAutoscaler]
	// ChainPolicyType is for Chain based fleet autoscaling
	// nolint:revive // Linter contains comment doesn't start with ChainPolicyType
	ChainPolicyType FleetAutoscalerPolicyType = "Chain"
	// FixedIntervalSyncType is a simple fixed interval based strategy for trigger autoscaling
	FixedIntervalSyncType FleetAutoscalerSyncType = "FixedInterval"

	defaultIntervalSyncSeconds int32 = 30
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

// CounterPolicy controls the desired behavior of the Counter autoscaler policy.
type CounterPolicy struct {
	// Key is the name of the Counter. Required field.
	Key string `json:"key"`

	// MaxCapacity is the maximum aggregate Counter total capacity across the fleet.
	// MaxCapacity must be bigger than both MinCapacity and BufferSize. Required field.
	MaxCapacity int64 `json:"maxCapacity"`

	// MinCapacity is the minimum aggregate Counter total capacity across the fleet.
	// If zero, MinCapacity is ignored.
	// If non zero, MinCapacity must be smaller than MaxCapacity and bigger than BufferSize.
	MinCapacity int64 `json:"minCapacity"`

	// BufferSize is the size of a buffer of counted items that are available in the Fleet (available
	// capacity). Value can be an absolute number (ex: 5) or a percentage of desired gs instances
	// (ex: 5%). An absolute number is calculated from percentage by rounding up.
	// Must be bigger than 0. Required field.
	BufferSize intstr.IntOrString `json:"bufferSize"`
}

// ListPolicy controls the desired behavior of the List autoscaler policy.
type ListPolicy struct {
	// Key is the name of the List. Required field.
	Key string `json:"key"`

	// MaxCapacity is the maximum aggregate List total capacity across the fleet.
	// MaxCapacity must be bigger than both MinCapacity and BufferSize. Required field.
	MaxCapacity int64 `json:"maxCapacity"`

	// MinCapacity is the minimum aggregate List total capacity across the fleet.
	// If zero, it is ignored.
	// If non zero, it must be smaller than MaxCapacity and bigger than BufferSize.
	MinCapacity int64 `json:"minCapacity"`

	// BufferSize is the size of a buffer based on the List capacity that is available over the
	// current aggregate List length in the Fleet (available capacity). It can be specified either
	// as an absolute value (i.e. 5) or percentage format (i.e. 5%).
	// Must be bigger than 0. Required field.
	BufferSize intstr.IntOrString `json:"bufferSize"`
}

type Between struct {
	// Start is the datetime that the policy is eligible to be applied.
	// If not set, the policy is always eligible to be applied
	// as soon as possible. If the datetime is in the past, the policy is
	// immediately eligible to be applied as well.
	// Optional field.
	Start string `json:"start"`

	// End is the datetime that the policy is no longer eligible to be applied.
	// If not set, the policy is always eligible to be applied.
	// after the start time. Optional field. 
	End string `json:"end"`
}

type ActivePeriod struct {
	// StartCron defines when the policy should be applied. 
	// If not set, the policy is always eligible to be applied.
	// This must conform to UNIX cron syntax.
	// Optional field.
	StartCron string `json:"startCron"`

	// Duration is the length of time that the policy is applied. 
	// If not set, the duration is indefinite.
	// A duration string is a possibly signed sequence of decimal numbers, 
	// (e.g. "300ms", "-1.5h" or "2h45m").
	// Valid time units are "ns", "us" (or "µs"), "ms", "s", "m", "h".
	// Optional field.
	Duration string `json:"duration"`
}

type ChainEntry struct {
	// Policy is the name of the policy to be applied.
	// Required field.
	Policy string `json:"policy"`

	// Between defines the time period that the policy is eligible to be applied.
	// Optional field.
	Between Between `json:"between"`

	// ActivePeriod defines the time period that the policy is applied.
	// Optional field.
	ActivePeriod ActivePeriod `json:"activePeriod"`
}

// ChainPolicy controls the desired behavior of the Chain autoscaler policy.
ChainPolicy []ChainEntry
	

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
func (fas *FleetAutoscaler) Validate() field.ErrorList {
	var allErrs field.ErrorList
	switch fas.Spec.Policy.Type {
	case BufferPolicyType:
		allErrs = fas.Spec.Policy.Buffer.ValidateBufferPolicy(field.NewPath("spec", "policy", "buffer"))

	case WebhookPolicyType:
		allErrs = fas.Spec.Policy.Webhook.ValidateWebhookPolicy(field.NewPath("spec", "policy", "webhook"))

	case CounterPolicyType:
		allErrs = fas.Spec.Policy.Counter.ValidateCounterPolicy(field.NewPath("spec", "policy", "counter"))

	case ListPolicyType:
		allErrs = fas.Spec.Policy.List.ValidateListPolicy(field.NewPath("spec", "policy", "list"))
	}

	if fas.Spec.Sync != nil {
		allErrs = append(allErrs, fas.Spec.Sync.FixedInterval.ValidateFixedIntervalSync(field.NewPath("spec", "sync", "fixedInterval"))...)
	}
	return allErrs
}

// ValidateWebhookPolicy validates the FleetAutoscaler Webhook policy settings
func (w *WebhookPolicy) ValidateWebhookPolicy(fldPath *field.Path) field.ErrorList {
	var allErrs field.ErrorList
	if w == nil {
		return append(allErrs, field.Required(fldPath, "webhook policy config params are missing"))
	}
	if w.Service == nil && w.URL == nil {
		allErrs = append(allErrs, field.Required(fldPath, "url should be provided"))
	}
	if w.Service != nil && w.URL != nil {
		allErrs = append(allErrs, field.Duplicate(fldPath.Child("url"), "service and url cannot be used simultaneously"))
	}
	if w.CABundle != nil {
		rootCAs := x509.NewCertPool()
		// Check that CABundle provided is correctly encoded certificate
		if ok := rootCAs.AppendCertsFromPEM(w.CABundle); !ok {
			allErrs = append(allErrs, field.Invalid(fldPath.Child("caBundle"), w.CABundle, "CA Bundle is not valid"))
		}
	}
	if w.URL != nil {
		u, err := url.Parse(*w.URL)
		if err != nil {
			allErrs = append(allErrs, field.Invalid(fldPath.Child("url"), *w.URL, "url is not valid"))
		} else if u.Scheme == "https" && w.CABundle == nil {
			allErrs = append(allErrs, field.Invalid(fldPath.Child("caBundle"), w.CABundle, "CABundle should be provided if HTTPS webhook is used"))
		}

	}
	return allErrs
}

// ValidateBufferPolicy validates the FleetAutoscaler Buffer policy settings
func (b *BufferPolicy) ValidateBufferPolicy(fldPath *field.Path) field.ErrorList {
	var allErrs field.ErrorList
	if b == nil {
		return append(allErrs, field.Required(fldPath, "buffer policy config params are missing"))
	}
	if b.MinReplicas > b.MaxReplicas {
		allErrs = append(allErrs, field.Invalid(fldPath.Child("minReplicas"), b.MinReplicas, "minReplicas should be smaller than maxReplicas"))
	}
	if b.BufferSize.Type == intstr.Int {
		if b.BufferSize.IntValue() <= 0 {
			allErrs = append(allErrs, field.Invalid(fldPath.Child("bufferSize"), b.BufferSize.IntValue(), apimachineryvalidation.IsNegativeErrorMsg))
		}
		if b.MaxReplicas < int32(b.BufferSize.IntValue()) {
			allErrs = append(allErrs, field.Invalid(fldPath.Child("maxReplicas"), b.MaxReplicas, "maxReplicas should be bigger than or equal to bufferSize"))
		}
		if b.MinReplicas != 0 && b.MinReplicas < int32(b.BufferSize.IntValue()) {
			allErrs = append(allErrs, field.Invalid(fldPath.Child("minReplicas"), b.MinReplicas, "minReplicas should be bigger than or equal to bufferSize"))
		}
	} else {
		r, err := intstr.GetScaledValueFromIntOrPercent(&b.BufferSize, 100, true)
		if err != nil || r < 1 || r > 99 {
			allErrs = append(allErrs, field.Invalid(fldPath.Child("bufferSize"), b.BufferSize.String(), "bufferSize should be between 1% and 99%"))
		}
		// When there is no allocated gameservers in a fleet,
		// Fleetautoscaler would reduce size of a fleet to MinReplicas.
		// If we have 0 MinReplicas and 0 Allocated then Fleetautoscaler would set Ready Replicas to 0
		// and we will not be able to raise the number of GS in a Fleet above zero
		if b.MinReplicas < 1 {
			allErrs = append(allErrs, field.Invalid(fldPath.Child("minReplicas"), b.MinReplicas, apimachineryvalidation.IsNegativeErrorMsg))
		}
	}
	return allErrs
}

// ValidateCounterPolicy validates the FleetAutoscaler Counter policy settings.
// Does not validate if a Counter with name CounterPolicy.Key is present in the fleet.
// nolint:dupl  // Linter errors on lines are duplicate of ValidateListPolicy
func (c *CounterPolicy) ValidateCounterPolicy(fldPath *field.Path) field.ErrorList {
	var allErrs field.ErrorList
	if !runtime.FeatureEnabled(runtime.FeatureCountsAndLists) {
		return append(allErrs, field.Forbidden(fldPath, "feature CountsAndLists must be enabled"))
	}

	if c == nil {
		return append(allErrs, field.Required(fldPath, "counter policy config params are missing"))
	}

	if c.MinCapacity > c.MaxCapacity {
		allErrs = append(allErrs, field.Invalid(fldPath.Child("minCapacity"), c.MinCapacity, "minCapacity should be smaller than maxCapacity"))
	}

	if c.BufferSize.Type == intstr.Int {
		if c.BufferSize.IntValue() <= 0 {
			allErrs = append(allErrs, field.Invalid(fldPath.Child("bufferSize"), c.BufferSize.IntValue(), apimachineryvalidation.IsNegativeErrorMsg))
		}
		if c.MaxCapacity < int64(c.BufferSize.IntValue()) {
			allErrs = append(allErrs, field.Invalid(fldPath.Child("maxCapacity"), c.MaxCapacity, "maxCapacity should be bigger than or equal to bufferSize"))
		}
		if c.MinCapacity != 0 && c.MinCapacity < int64(c.BufferSize.IntValue()) {
			allErrs = append(allErrs, field.Invalid(fldPath.Child("minCapacity"), c.MinCapacity, "minCapacity should be bigger than or equal to bufferSize"))
		}
	} else {
		r, err := intstr.GetScaledValueFromIntOrPercent(&c.BufferSize, 100, true)
		if err != nil || r < 1 || r > 99 {
			allErrs = append(allErrs, field.Invalid(fldPath.Child("bufferSize"), c.BufferSize.String(), "bufferSize should be between 1% and 99%"))
		}
		// When bufferSize in percentage format is used, minCapacity should be more than 0.
		if c.MinCapacity < 1 {
			allErrs = append(allErrs, field.Invalid(fldPath.Child("minCapacity"), c.BufferSize.String(), " when bufferSize in percentage format is used, minCapacity should be more than 0"))
		}
	}

	return allErrs
}

// ValidateListPolicy validates the FleetAutoscaler List policy settings.
// Does not validate if a List with name ListPolicy.Key is present in the fleet.
// nolint:dupl  // Linter errors on lines are duplicate of ValidateCounterPolicy
func (l *ListPolicy) ValidateListPolicy(fldPath *field.Path) field.ErrorList {
	var allErrs field.ErrorList
	if !runtime.FeatureEnabled(runtime.FeatureCountsAndLists) {
		return append(allErrs, field.Forbidden(fldPath, "feature CountsAndLists must be enabled"))
	}
	if l == nil {
		return append(allErrs, field.Required(fldPath, "list policy config params are missing"))
	}
	if l.MinCapacity > l.MaxCapacity {
		allErrs = append(allErrs, field.Invalid(fldPath.Child("minCapacity"), l.MinCapacity, "minCapacity should be smaller than maxCapacity"))
	}
	if l.BufferSize.Type == intstr.Int {
		if l.BufferSize.IntValue() <= 0 {
			allErrs = append(allErrs, field.Invalid(fldPath.Child("bufferSize"), l.BufferSize.IntValue(), apimachineryvalidation.IsNegativeErrorMsg))
		}
		if l.MaxCapacity < int64(l.BufferSize.IntValue()) {
			allErrs = append(allErrs, field.Invalid(fldPath.Child("maxCapacity"), l.MaxCapacity, "maxCapacity should be bigger than or equal to bufferSize"))
		}
		if l.MinCapacity != 0 && l.MinCapacity < int64(l.BufferSize.IntValue()) {
			allErrs = append(allErrs, field.Invalid(fldPath.Child("minCapacity"), l.MinCapacity, "minCapacity should be bigger than or equal to bufferSize"))
		}
	} else {
		r, err := intstr.GetScaledValueFromIntOrPercent(&l.BufferSize, 100, true)
		if err != nil || r < 1 || r > 99 {
			allErrs = append(allErrs, field.Invalid(fldPath.Child("bufferSize"), l.BufferSize.String(), "bufferSize should be between 1% and 99%"))
		}
		// When bufferSize in percentage format is used, minCapacity should be more than 0.
		if l.MinCapacity < 1 {
			allErrs = append(allErrs, field.Invalid(fldPath.Child("minCapacity"), l.BufferSize.String(), " when bufferSize in percentage format is used, minCapacity should be more than 0"))
		}
	}
	return allErrs
}

// ValidateFixedIntervalSync validates the FixedIntervalSync settings
func (i *FixedIntervalSync) ValidateFixedIntervalSync(fldPath *field.Path) field.ErrorList {
	var allErrs field.ErrorList
	if i == nil {
		return append(allErrs, field.Required(fldPath, "fixedInterval sync config params are missing"))
	}
	if i.Seconds <= 0 {
		allErrs = append(allErrs, field.Invalid(fldPath.Child("seconds"), i.Seconds, apimachineryvalidation.IsNegativeErrorMsg))
	}
	return allErrs
}

// ApplyDefaults applies default values to the FleetAutoscaler
func (fas *FleetAutoscaler) ApplyDefaults() {
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
