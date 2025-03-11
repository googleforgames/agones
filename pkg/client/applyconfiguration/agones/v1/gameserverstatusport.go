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

// GameServerStatusPortApplyConfiguration represents a declarative configuration of the GameServerStatusPort type for use
// with apply.
type GameServerStatusPortApplyConfiguration struct {
	Name *string `json:"name,omitempty"`
	Port *int32  `json:"port,omitempty"`
}

// GameServerStatusPortApplyConfiguration constructs a declarative configuration of the GameServerStatusPort type for use with
// apply.
func GameServerStatusPort() *GameServerStatusPortApplyConfiguration {
	return &GameServerStatusPortApplyConfiguration{}
}

// WithName sets the Name field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Name field is set to the value of the last call.
func (b *GameServerStatusPortApplyConfiguration) WithName(value string) *GameServerStatusPortApplyConfiguration {
	b.Name = &value
	return b
}

// WithPort sets the Port field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Port field is set to the value of the last call.
func (b *GameServerStatusPortApplyConfiguration) WithPort(value int32) *GameServerStatusPortApplyConfiguration {
	b.Port = &value
	return b
}
