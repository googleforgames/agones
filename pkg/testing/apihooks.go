// Copyright 2023 Google LLC All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package testing

import (
	"agones.dev/agones/pkg/apis"
	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// FakeAPIHooks is a no-op, fake implementation of APIHooks
type FakeAPIHooks struct {
}

var _ agonesv1.APIHooks = FakeAPIHooks{}

// ValidateGameServerSpec is called by GameServer.Validate to allow for product specific validation.
func (f FakeAPIHooks) ValidateGameServerSpec(_ *agonesv1.GameServerSpec) []metav1.StatusCause {
	return nil
}

// ValidateScheduling is called by Fleet and GameServerSet Validate() to allow for product specific validation of scheduling strategy.
func (f FakeAPIHooks) ValidateScheduling(_ apis.SchedulingStrategy) []metav1.StatusCause {
	return nil
}

// MutateGameServerPodSpec is called by createGameServerPod to allow for product specific pod mutation.
func (f FakeAPIHooks) MutateGameServerPodSpec(_ *agonesv1.GameServerSpec, podSpec *corev1.PodSpec) error {
	return nil
}

// SetEviction is called by gs.Pod to enforce GameServer.Status.Eviction.
func (f FakeAPIHooks) SetEviction(_ *agonesv1.Eviction, pod *corev1.Pod) error {
	return nil
}
