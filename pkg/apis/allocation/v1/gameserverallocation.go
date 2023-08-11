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
	"errors"
	"fmt"

	"agones.dev/agones/pkg/apis"
	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	"agones.dev/agones/pkg/util/runtime"
	hashstructure "github.com/mitchellh/hashstructure/v2"
	corev1 "k8s.io/api/core/v1"
	apivalidation "k8s.io/apimachinery/pkg/api/validation"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	metav1validation "k8s.io/apimachinery/pkg/apis/meta/v1/validation"
	"k8s.io/apimachinery/pkg/labels"
	field "k8s.io/apimachinery/pkg/util/validation/field"
)

const (
	// GameServerAllocationAllocated is allocation successful
	GameServerAllocationAllocated GameServerAllocationState = "Allocated"
	// GameServerAllocationUnAllocated when the allocation is unsuccessful
	GameServerAllocationUnAllocated GameServerAllocationState = "UnAllocated"
	// GameServerAllocationContention when the allocation is unsuccessful
	// because of contention
	GameServerAllocationContention GameServerAllocationState = "Contention"
)

// GameServerAllocationState is the Allocation state
type GameServerAllocationState string

// +genclient
// +genclient:onlyVerbs=create
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// GameServerAllocation is the data structure for allocating against a set of
// GameServers, defined `selectors` selectors
type GameServerAllocation struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              GameServerAllocationSpec   `json:"spec"`
	Status            GameServerAllocationStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// GameServerAllocationList is a list of GameServer Allocation resources
type GameServerAllocationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []GameServerAllocation `json:"items"`
}

// GameServerAllocationSpec is the spec for a GameServerAllocation
type GameServerAllocationSpec struct {
	// MultiClusterPolicySelector if specified, multi-cluster policies are applied.
	// Otherwise, allocation will happen locally.
	MultiClusterSetting MultiClusterSetting `json:"multiClusterSetting,omitempty" hash:"ignore"`

	// Deprecated: use field Selectors instead. If Selectors is set, this field is ignored.
	// Required is the GameServer selector from which to choose GameServers from.
	// Defaults to all GameServers.
	Required GameServerSelector `json:"required,omitempty" hash:"ignore"`

	// Deprecated: use field Selectors instead. If Selectors is set, this field is ignored.
	// Preferred is an ordered list of preferred GameServer selectors
	// that are optional to be fulfilled, but will be searched before the `required` selector.
	// If the first selector is not matched, the selection attempts the second selector, and so on.
	// If any of the preferred selectors are matched, the required selector is not considered.
	// This is useful for things like smoke testing of new game servers.
	Preferred []GameServerSelector `json:"preferred,omitempty" hash:"ignore"`

	// (Alpha, CountsAndLists feature flag) The first Priority on the array of Priorities is the most
	// important for sorting. The allocator will use the first priority for sorting GameServers by
	// available Capacity in the Selector set. Acts as a tie-breaker after sorting the game servers
	// by State and Strategy Packed. Impacts which GameServer is checked first.
	// +optional
	Priorities []agonesv1.Priority `json:"priorities,omitempty"`

	// Ordered list of GameServer label selectors.
	// If the first selector is not matched, the selection attempts the second selector, and so on.
	// This is useful for things like smoke testing of new game servers.
	// Note: This field can only be set if neither Required or Preferred is set.
	Selectors []GameServerSelector `json:"selectors,omitempty" hash:"ignore"`

	// Scheduling strategy. Defaults to "Packed".
	Scheduling apis.SchedulingStrategy `json:"scheduling"`

	// MetaPatch is optional custom metadata that is added to the game server at allocation
	// You can use this to tell the server necessary session data
	MetaPatch MetaPatch `json:"metadata,omitempty" hash:"ignore"`

	// (Alpha, CountsAndLists feature flag) Counters and Lists provide a set of actions to perform
	// on Counters and Lists during allocation.
	// +optional
	Counters map[string]CounterAction `json:"counters,omitempty" hash:"ignore"`
	Lists    map[string]ListAction    `json:"lists,omitempty" hash:"ignore"`
}

// GameServerSelector contains all the filter options for selecting
// a GameServer for allocation.
type GameServerSelector struct {
	// See: https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/
	metav1.LabelSelector `json:",inline"`
	// GameServerState specifies which State is the filter to be used when attempting to retrieve a GameServer
	// via Allocation. Defaults to "Ready". The only other option is "Allocated", which can be used in conjunction with
	// label/annotation/player selectors to retrieve an already Allocated GameServer.
	GameServerState *agonesv1.GameServerState `json:"gameServerState,omitempty"`
	// [Stage:Alpha]
	// [FeatureFlag:PlayerAllocationFilter]
	// +optional
	// Players provides a filter on minimum and maximum values for player capacity when retrieving a GameServer
	// through Allocation. Defaults to no limits.
	Players *PlayerSelector `json:"players,omitempty"`
	// (Alpha, CountsAndLists feature flag) Counters provides filters on minimum and maximum values
	// for a Counter's count and available capacity when retrieving a GameServer through Allocation.
	// Defaults to no limits.
	// +optional
	Counters map[string]CounterSelector `json:"counters,omitempty"`
	// (Alpha, CountsAndLists feature flag) Lists provides filters on minimum and maximum values
	// for List capacity, and for the existence of a value in a List, when retrieving a GameServer
	// through Allocation. Defaults to no limits.
	// +optional
	Lists map[string]ListSelector `json:"lists,omitempty"`
}

// PlayerSelector is the filter options for a GameServer based on player counts
type PlayerSelector struct {
	MinAvailable int64 `json:"minAvailable,omitempty"`
	MaxAvailable int64 `json:"maxAvailable,omitempty"`
}

// CounterSelector is the filter options for a GameServer based on the count and/or available capacity.
// 0 for MaxCount or MaxAvailable means unlimited maximum. Default for all fields: 0
type CounterSelector struct {
	MinCount     int64 `json:"minCount"`
	MaxCount     int64 `json:"maxCount"`
	MinAvailable int64 `json:"minAvailable"`
	MaxAvailable int64 `json:"maxAvailable"`
}

// ListSelector is the filter options for a GameServer based on List available capacity and/or the
// existence of a value in a List.
// 0 for MaxAvailable means unlimited maximum. Default for integer fields: 0
// "" for ContainsValue means ignore field. Default for string field: ""
type ListSelector struct {
	ContainsValue string `json:"containsValue"`
	MinAvailable  int64  `json:"minAvailable"`
	MaxAvailable  int64  `json:"maxAvailable"`
}

// CounterAction is an optional action that can be performed on a Counter at allocation.
// Action: "Increment" or "Decrement" the Counter's Count (optional). Must also define the Amount.
// Amount: The amount to increment or decrement the Count (optional). Must be a positive integer.
// Capacity: Update the maximum capacity of the Counter to this number (optional). Min 0, Max int64.
type CounterAction struct {
	Action   *string `json:"action,omitempty"`
	Amount   *int64  `json:"amount,omitempty"`
	Capacity *int64  `json:"capacity,omitempty"`
}

// ListAction is an optional action that can be performed on a List at allocation.
// AddValues: Append values to a List's Values array (optional). Any duplicate values will be ignored.
// Capacity: Update the maximum capacity of the Counter to this number (optional). Min 0, Max 1000.
type ListAction struct {
	AddValues []string `json:"addValues,omitempty"`
	Capacity  *int64   `json:"capacity,omitempty"`
}

// ApplyDefaults applies default values
func (s *GameServerSelector) ApplyDefaults() {
	if s.GameServerState == nil {
		state := agonesv1.GameServerStateReady
		s.GameServerState = &state
	}

	if runtime.FeatureEnabled(runtime.FeaturePlayerAllocationFilter) {
		if s.Players == nil {
			s.Players = &PlayerSelector{}
		}
	}

	if runtime.FeatureEnabled(runtime.FeatureCountsAndLists) {
		if s.Counters == nil {
			s.Counters = make(map[string]CounterSelector)
		}
		if s.Lists == nil {
			s.Lists = make(map[string]ListSelector)
		}
	}
}

// Matches checks to see if a GameServer matches a given GameServerSelector's criteria.
// Will panic if the `GameServerSelector` has not passed `Validate()`.
func (s *GameServerSelector) Matches(gs *agonesv1.GameServer) bool {

	// Assume at this point, this has already been run through Validate(), and it can be converted.
	// We end up running LabelSelectorAsSelector twice for each allocation, but if we store the results of this
	// function within the GameServerSelector, we can't fuzz the GameServerAllocation as reflect.DeepEqual
	// will fail due to the unexported field.
	selector, err := metav1.LabelSelectorAsSelector(&s.LabelSelector)
	if err != nil {
		panic("GameServerSelector.Validate() has not been called before calling GameServerSelector.Matches(...)")
	}

	// first check labels
	if !selector.Matches(labels.Set(gs.ObjectMeta.Labels)) {
		return false
	}

	// then if state is being checked, check state
	if s.GameServerState != nil && gs.Status.State != *s.GameServerState {
		return false
	}

	// then if player count is being checked, check that
	if runtime.FeatureEnabled(runtime.FeaturePlayerAllocationFilter) {
		// 0 is unlimited number of players
		if s.Players != nil && gs.Status.Players != nil && s.Players.MaxAvailable != 0 {
			available := gs.Status.Players.Capacity - gs.Status.Players.Count
			if !(available >= s.Players.MinAvailable && available <= s.Players.MaxAvailable) {
				return false
			}
		}
	}

	if runtime.FeatureEnabled(runtime.FeatureCountsAndLists) {
		// Only check for matches if there are CounterSelectors or ListSelectors
		if (s.Counters != nil) && (len(s.Counters) != 0) {
			if !(s.matchCounters(gs)) {
				return false
			}
		}
		if (s.Lists != nil) && (len(s.Lists) != 0) {
			if !(s.matchLists(gs)) {
				return false
			}
		}
	}

	return true
}

// matchCounters returns true if there is a match for the CounterSelector in the GameServerStatus
func (s *GameServerSelector) matchCounters(gs *agonesv1.GameServer) bool {
	if gs.Status.Counters == nil {
		return false
	}
	for counter, counterSelector := range s.Counters {
		// If the Counter Selector does not exist in GameServerStatus, return false.
		counterStatus, ok := gs.Status.Counters[counter]
		if !ok {
			return false
		}
		// 0 means undefined (unlimited) for MaxAvailable.
		available := counterStatus.Capacity - counterStatus.Count
		if available < counterSelector.MinAvailable ||
			(counterSelector.MaxAvailable != 0 && available > counterSelector.MaxAvailable) {
			return false
		}
		// 0 means undefined (unlimited) for MaxCount.
		if counterStatus.Count < counterSelector.MinCount ||
			(counterSelector.MaxCount != 0 && counterStatus.Count > counterSelector.MaxCount) {
			return false
		}
	}
	return true
}

// CounterActions attempts to peform any actions from the CounterAction on the GameServer Counter.
// Returns the errors of any actions that could not be performed.
func (ca *CounterAction) CounterActions(counter string, gs *agonesv1.GameServer) error {
	var errs error
	if ca.Capacity != nil {
		capErr := gs.UpdateCounterCapacity(counter, *ca.Capacity)
		if capErr != nil {
			errs = errors.Join(errs, capErr)
		}
	}
	if ca.Action != nil && ca.Amount != nil {
		cntErr := gs.UpdateCount(counter, *ca.Action, *ca.Amount)
		if cntErr != nil {
			errs = errors.Join(errs, cntErr)
		}
	}
	return errs
}

// ListActions attempts to peform any actions from the ListAction on the GameServer List.
// Returns a string list of any actions that could not be performed.
func (la *ListAction) ListActions(list string, gs *agonesv1.GameServer) error {
	var errs error
	if la.Capacity != nil {
		capErr := gs.UpdateListCapacity(list, *la.Capacity)
		if capErr != nil {
			errs = errors.Join(errs, capErr)
		}
	}
	if la.AddValues != nil && len(la.AddValues) > 0 {
		cntErr := gs.AppendListValues(list, la.AddValues)
		if cntErr != nil {
			errs = errors.Join(errs, cntErr)
		}
	}
	return errs
}

// matchLists returns true if there is a match for the ListSelector in the GameServerStatus
func (s *GameServerSelector) matchLists(gs *agonesv1.GameServer) bool {
	if gs.Status.Lists == nil {
		return false
	}
	for list, listSelector := range s.Lists {
		// If the List Selector does not exist in GameServerStatus, return false.
		listStatus, ok := gs.Status.Lists[list]
		if !ok {
			return false
		}
		// Match List based on capacity
		available := listStatus.Capacity - int64(len(listStatus.Values))
		// 0 means undefined (unlimited) for MaxAvailable.
		if available < listSelector.MinAvailable ||
			(listSelector.MaxAvailable != 0 && available > listSelector.MaxAvailable) {
			return false
		}
		// Check if List contains ContainsValue (if a value has been specified)
		if listSelector.ContainsValue != "" {
			valueExists := false
			for _, value := range listStatus.Values {
				if value == listSelector.ContainsValue {
					valueExists = true
					break
				}
			}
			if !valueExists {
				return false
			}
		}
	}
	return true
}

// Validate validates that the selection fields have valid values
func (s *GameServerSelector) Validate(fldPath *field.Path) field.ErrorList {
	var allErrs field.ErrorList

	_, err := metav1.LabelSelectorAsSelector(&s.LabelSelector)
	if err != nil {
		allErrs = append(allErrs, field.Invalid(fldPath.Child("labelSelector"), s.LabelSelector, fmt.Sprintf("Error converting label selector: %s", err)))
	}

	if s.GameServerState != nil && !(*s.GameServerState == agonesv1.GameServerStateAllocated || *s.GameServerState == agonesv1.GameServerStateReady) {
		allErrs = append(allErrs, field.Invalid(fldPath.Child("gameServerState"), *s.GameServerState, "GameServerState must be either Allocated or Ready"))
	}

	if runtime.FeatureEnabled(runtime.FeaturePlayerAllocationFilter) && s.Players != nil {
		if s.Players.MinAvailable < 0 {
			allErrs = append(allErrs, field.Invalid(fldPath.Child("players").Child("minAvailable"), s.Players.MinAvailable, apivalidation.IsNegativeErrorMsg))
		}

		if s.Players.MaxAvailable < 0 {
			allErrs = append(allErrs, field.Invalid(fldPath.Child("players").Child("maxAvailable"), s.Players.MaxAvailable, apivalidation.IsNegativeErrorMsg))
		}

		if s.Players.MinAvailable > s.Players.MaxAvailable {
			allErrs = append(allErrs, field.Invalid(fldPath.Child("players").Child("minAvailable"), s.Players.MinAvailable, "minAvailable cannot be greater than maxAvailable"))
		}
	}

	if runtime.FeatureEnabled(runtime.FeatureCountsAndLists) {
		if s.Counters != nil {
			allErrs = append(allErrs, validateCounters(s.Counters, fldPath.Child("counters"))...)
		}
		if s.Lists != nil {
			allErrs = append(allErrs, validateLists(s.Lists, fldPath.Child("lists"))...)
		}
	} else {
		if s.Counters != nil {
			allErrs = append(allErrs, field.Forbidden(fldPath.Child("counters"), "Feature CountsAndLists must be enabled"))
		}
		if s.Lists != nil {
			allErrs = append(allErrs, field.Forbidden(fldPath.Child("lists"), "Feature CountsAndLists must be enabled"))
		}
	}

	return allErrs
}

// validateCounters validates that the selection field has valid values for CounterSelectors
func validateCounters(counters map[string]CounterSelector, fldPath *field.Path) field.ErrorList {
	var allErrs field.ErrorList
	for key, counterSelector := range counters {
		keyPath := fldPath.Key(key)
		if counterSelector.MinCount < 0 {
			allErrs = append(allErrs, field.Invalid(keyPath.Child("minCount"), counterSelector.MinCount, apivalidation.IsNegativeErrorMsg))
		}
		if counterSelector.MaxCount < 0 {
			allErrs = append(allErrs, field.Invalid(keyPath.Child("maxCount"), counterSelector.MaxCount, apivalidation.IsNegativeErrorMsg))
		}
		if (counterSelector.MaxCount < counterSelector.MinCount) && (counterSelector.MaxCount != 0) {
			allErrs = append(allErrs, field.Invalid(keyPath, counterSelector.MaxCount, fmt.Sprintf("maxCount must zero or greater than minCount %d", counterSelector.MinCount)))
		}
		if counterSelector.MinAvailable < 0 {
			allErrs = append(allErrs, field.Invalid(keyPath.Child("minAvailable"), counterSelector.MinAvailable, apivalidation.IsNegativeErrorMsg))
		}
		if counterSelector.MaxAvailable < 0 {
			allErrs = append(allErrs, field.Invalid(keyPath.Child("maxAvailable"), counterSelector.MaxAvailable, apivalidation.IsNegativeErrorMsg))
		}
		if (counterSelector.MaxAvailable < counterSelector.MinAvailable) && (counterSelector.MaxAvailable != 0) {
			allErrs = append(allErrs, field.Invalid(keyPath, counterSelector.MaxAvailable, fmt.Sprintf("maxAvailable must zero or greater than minAvailable %d", counterSelector.MinAvailable)))
		}
	}

	return allErrs
}

// validateLists validates that the selection field has valid values for ListSelectors
func validateLists(lists map[string]ListSelector, fldPath *field.Path) field.ErrorList {
	var allErrs field.ErrorList
	for key, listSelector := range lists {
		keyPath := fldPath.Key(key)
		if listSelector.MinAvailable < 0 {
			allErrs = append(allErrs, field.Invalid(keyPath.Child("minAvailable"), listSelector.MinAvailable, apivalidation.IsNegativeErrorMsg))
		}
		if listSelector.MaxAvailable < 0 {
			allErrs = append(allErrs, field.Invalid(keyPath.Child("maxAvailable"), listSelector.MaxAvailable, apivalidation.IsNegativeErrorMsg))
		}
		if (listSelector.MaxAvailable < listSelector.MinAvailable) && (listSelector.MaxAvailable != 0) {
			allErrs = append(allErrs, field.Invalid(keyPath, listSelector.MaxAvailable, fmt.Sprintf("maxAvailable must zero or greater than minAvailable %d", listSelector.MinAvailable)))
		}
	}

	return allErrs
}

// MultiClusterSetting specifies settings for multi-cluster allocation.
type MultiClusterSetting struct {
	Enabled        bool                 `json:"enabled,omitempty"`
	PolicySelector metav1.LabelSelector `json:"policySelector,omitempty"`
}

// MetaPatch is the metadata used to patch the GameServer metadata on allocation
type MetaPatch struct {
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

// Validate returns if the labels and/or annotations that are to be applied to a `GameServer` post
// allocation are valid.
func (mp *MetaPatch) Validate(fldPath *field.Path) field.ErrorList {
	allErrs := metav1validation.ValidateLabels(mp.Labels, fldPath.Child("labels"))
	allErrs = append(allErrs, apivalidation.ValidateAnnotations(mp.Annotations, fldPath.Child("annotations"))...)
	return allErrs
}

// GameServerAllocationStatus is the status for an GameServerAllocation resource
type GameServerAllocationStatus struct {
	// GameServerState is the current state of an GameServerAllocation, e.g. Allocated, or UnAllocated
	State          GameServerAllocationState       `json:"state"`
	GameServerName string                          `json:"gameServerName"`
	Ports          []agonesv1.GameServerStatusPort `json:"ports,omitempty"`
	Address        string                          `json:"address,omitempty"`
	Addresses      []corev1.NodeAddress            `json:"addresses,omitempty"`
	NodeName       string                          `json:"nodeName,omitempty"`
	// If the allocation is from a remote cluster, Source is the endpoint of the remote agones-allocator.
	// Otherwise, Source is "local"
	Source   string              `json:"source"`
	Metadata *GameServerMetadata `json:"metadata,omitempty"`
}

// GameServerMetadata is the metadata from the allocated game server at allocation time
type GameServerMetadata struct {
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

// ApplyDefaults applies the default values to this GameServerAllocation
func (gsa *GameServerAllocation) ApplyDefaults() {
	if gsa.Spec.Scheduling == "" {
		gsa.Spec.Scheduling = apis.Packed
	}

	if len(gsa.Spec.Selectors) == 0 {
		gsa.Spec.Required.ApplyDefaults()

		for i := range gsa.Spec.Preferred {
			gsa.Spec.Preferred[i].ApplyDefaults()
		}
	} else {
		for i := range gsa.Spec.Selectors {
			gsa.Spec.Selectors[i].ApplyDefaults()
		}
	}
}

// Validate validation for the GameServerAllocation
// Validate should be called before attempting to Match any of the GameServer selectors.
func (gsa *GameServerAllocation) Validate() field.ErrorList {
	var allErrs field.ErrorList
	specPath := field.NewPath("spec")
	if gsa.Spec.Scheduling != apis.Packed && gsa.Spec.Scheduling != apis.Distributed {
		allErrs = append(allErrs, field.NotSupported(specPath.Child("scheduling"), string(gsa.Spec.Scheduling), []string{string(apis.Packed), string(apis.Distributed)}))
	}

	allErrs = append(allErrs, gsa.Spec.Required.Validate(specPath.Child("required"))...)
	for i := range gsa.Spec.Preferred {
		allErrs = append(allErrs, gsa.Spec.Preferred[i].Validate(specPath.Child("preferred").Index(i))...)
	}
	for i := range gsa.Spec.Selectors {
		allErrs = append(allErrs, gsa.Spec.Selectors[i].Validate(specPath.Child("selectors").Index(i))...)
	}

	if !runtime.FeatureEnabled(runtime.FeatureCountsAndLists) && (gsa.Spec.Priorities != nil) {
		allErrs = append(allErrs, field.Forbidden(specPath.Child("priorities"), "Feature CountsAndLists must be enabled if Priorities is specified"))
	}

	allErrs = append(allErrs, gsa.Spec.MetaPatch.Validate(specPath.Child("metadata"))...)
	return allErrs
}

// Converter converts game server allocation required and preferred fields to selectors field.
func (gsa *GameServerAllocation) Converter() {
	if len(gsa.Spec.Selectors) == 0 {
		var selectors []GameServerSelector
		selectors = append(selectors, gsa.Spec.Preferred...)
		selectors = append(selectors, gsa.Spec.Required)
		gsa.Spec.Selectors = selectors
	}
}

// SortKey generates and returns the hash of the GameServerAllocationSpec []Priority and Scheduling.
// Note: The hash:"ignore" in GameServerAllocationSpec means that these fields will not be considered
// in hashing. The hash is used for determining when GameServerAllocations have equal or different
// []Priority and Scheduling.
func (gsa *GameServerAllocation) SortKey() (uint64, error) {
	hash, err := hashstructure.Hash(gsa.Spec, hashstructure.FormatV2, nil)
	if err != nil {
		return 0, err
	}
	return hash, nil
}
