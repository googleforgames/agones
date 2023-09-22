// Copyright 2023 Google LLC All Rights Reserved.
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

// Code generated by client-gen. DO NOT EDIT.

package fake

import (
	"context"

	v1 "agones.dev/agones/pkg/apis/allocation/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	testing "k8s.io/client-go/testing"
)

// FakeGameServerAllocations implements GameServerAllocationInterface
type FakeGameServerAllocations struct {
	Fake *FakeAllocationV1
	ns   string
}

var gameserverallocationsResource = schema.GroupVersionResource{Group: "allocation.agones.dev", Version: "v1", Resource: "gameserverallocations"}

var gameserverallocationsKind = schema.GroupVersionKind{Group: "allocation.agones.dev", Version: "v1", Kind: "GameServerAllocation"}

// Create takes the representation of a gameServerAllocation and creates it.  Returns the server's representation of the gameServerAllocation, and an error, if there is any.
func (c *FakeGameServerAllocations) Create(ctx context.Context, gameServerAllocation *v1.GameServerAllocation, opts metav1.CreateOptions) (result *v1.GameServerAllocation, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(gameserverallocationsResource, c.ns, gameServerAllocation), &v1.GameServerAllocation{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1.GameServerAllocation), err
}
