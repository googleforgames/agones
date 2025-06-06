// Copyright 2024 Google LLC All Rights Reserved.
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

// This code was autogenerated. Do not edit directly.

// Code generated by applyconfiguration-gen. DO NOT EDIT.

package v1

import (
	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	corev1 "k8s.io/api/core/v1"
)

// GameServerPortApplyConfiguration represents a declarative configuration of the GameServerPort type for use
// with apply.
type GameServerPortApplyConfiguration struct {
	Name          *string              `json:"name,omitempty"`
	Range         *string              `json:"range,omitempty"`
	PortPolicy    *agonesv1.PortPolicy `json:"portPolicy,omitempty"`
	Container     *string              `json:"container,omitempty"`
	ContainerPort *int32               `json:"containerPort,omitempty"`
	HostPort      *int32               `json:"hostPort,omitempty"`
	Protocol      *corev1.Protocol     `json:"protocol,omitempty"`
}

// GameServerPortApplyConfiguration constructs a declarative configuration of the GameServerPort type for use with
// apply.
func GameServerPort() *GameServerPortApplyConfiguration {
	return &GameServerPortApplyConfiguration{}
}

// WithName sets the Name field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Name field is set to the value of the last call.
func (b *GameServerPortApplyConfiguration) WithName(value string) *GameServerPortApplyConfiguration {
	b.Name = &value
	return b
}

// WithRange sets the Range field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Range field is set to the value of the last call.
func (b *GameServerPortApplyConfiguration) WithRange(value string) *GameServerPortApplyConfiguration {
	b.Range = &value
	return b
}

// WithPortPolicy sets the PortPolicy field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the PortPolicy field is set to the value of the last call.
func (b *GameServerPortApplyConfiguration) WithPortPolicy(value agonesv1.PortPolicy) *GameServerPortApplyConfiguration {
	b.PortPolicy = &value
	return b
}

// WithContainer sets the Container field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Container field is set to the value of the last call.
func (b *GameServerPortApplyConfiguration) WithContainer(value string) *GameServerPortApplyConfiguration {
	b.Container = &value
	return b
}

// WithContainerPort sets the ContainerPort field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the ContainerPort field is set to the value of the last call.
func (b *GameServerPortApplyConfiguration) WithContainerPort(value int32) *GameServerPortApplyConfiguration {
	b.ContainerPort = &value
	return b
}

// WithHostPort sets the HostPort field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the HostPort field is set to the value of the last call.
func (b *GameServerPortApplyConfiguration) WithHostPort(value int32) *GameServerPortApplyConfiguration {
	b.HostPort = &value
	return b
}

// WithProtocol sets the Protocol field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Protocol field is set to the value of the last call.
func (b *GameServerPortApplyConfiguration) WithProtocol(value corev1.Protocol) *GameServerPortApplyConfiguration {
	b.Protocol = &value
	return b
}
