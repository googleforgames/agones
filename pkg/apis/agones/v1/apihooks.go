// Copyright 2022 Google LLC All Rights Reserved.
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

package v1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// APIHooks is a subset of the cloudproduct.CloudProduct interface for cloud product hooks specific to this package.
// We use this layering so that cloudproduct can import v1.GameServer (to get e.g. the GameServer type), but
// also allow the cloud product to override behavior of GameServer.Pod(), etc.
type APIHooks interface {
	// ValidateGameServerSpec is called by GameServer.Validate to allow for product specific validation.
	ValidateGameServerSpec(*GameServerSpec) []metav1.StatusCause

	// MutateGameServerPodSpec is called by createGameServerPod to allow for product specific pod mutation.
	MutateGameServerPodSpec(*GameServerSpec, *corev1.PodSpec) error
}

var apiHooks APIHooks = generic{}

// RegisterAPIHooks registers API-specific cloud product hooks. It should only be called by
// the cloudproduct package on initialization.
func RegisterAPIHooks(hooks APIHooks) {
	if hooks == nil {
		hooks = generic{}
	}
	apiHooks = hooks
}

type generic struct{}

func (generic) ValidateGameServerSpec(*GameServerSpec) []metav1.StatusCause    { return nil }
func (generic) MutateGameServerPodSpec(*GameServerSpec, *corev1.PodSpec) error { return nil }
