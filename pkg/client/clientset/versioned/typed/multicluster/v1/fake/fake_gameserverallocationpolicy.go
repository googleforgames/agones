// Copyright 2022 Google LLC All Rights Reserved.
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

	multiclusterv1 "agones.dev/agones/pkg/apis/multicluster/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeGameServerAllocationPolicies implements GameServerAllocationPolicyInterface
type FakeGameServerAllocationPolicies struct {
	Fake *FakeMulticlusterV1
	ns   string
}

var gameserverallocationpoliciesResource = schema.GroupVersionResource{Group: "multicluster.agones.dev", Version: "v1", Resource: "gameserverallocationpolicies"}

var gameserverallocationpoliciesKind = schema.GroupVersionKind{Group: "multicluster.agones.dev", Version: "v1", Kind: "GameServerAllocationPolicy"}

// Get takes name of the gameServerAllocationPolicy, and returns the corresponding gameServerAllocationPolicy object, and an error if there is any.
func (c *FakeGameServerAllocationPolicies) Get(ctx context.Context, name string, options v1.GetOptions) (result *multiclusterv1.GameServerAllocationPolicy, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(gameserverallocationpoliciesResource, c.ns, name), &multiclusterv1.GameServerAllocationPolicy{})

	if obj == nil {
		return nil, err
	}
	return obj.(*multiclusterv1.GameServerAllocationPolicy), err
}

// List takes label and field selectors, and returns the list of GameServerAllocationPolicies that match those selectors.
func (c *FakeGameServerAllocationPolicies) List(ctx context.Context, opts v1.ListOptions) (result *multiclusterv1.GameServerAllocationPolicyList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(gameserverallocationpoliciesResource, gameserverallocationpoliciesKind, c.ns, opts), &multiclusterv1.GameServerAllocationPolicyList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &multiclusterv1.GameServerAllocationPolicyList{ListMeta: obj.(*multiclusterv1.GameServerAllocationPolicyList).ListMeta}
	for _, item := range obj.(*multiclusterv1.GameServerAllocationPolicyList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested gameServerAllocationPolicies.
func (c *FakeGameServerAllocationPolicies) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(gameserverallocationpoliciesResource, c.ns, opts))

}

// Create takes the representation of a gameServerAllocationPolicy and creates it.  Returns the server's representation of the gameServerAllocationPolicy, and an error, if there is any.
func (c *FakeGameServerAllocationPolicies) Create(ctx context.Context, gameServerAllocationPolicy *multiclusterv1.GameServerAllocationPolicy, opts v1.CreateOptions) (result *multiclusterv1.GameServerAllocationPolicy, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(gameserverallocationpoliciesResource, c.ns, gameServerAllocationPolicy), &multiclusterv1.GameServerAllocationPolicy{})

	if obj == nil {
		return nil, err
	}
	return obj.(*multiclusterv1.GameServerAllocationPolicy), err
}

// Update takes the representation of a gameServerAllocationPolicy and updates it. Returns the server's representation of the gameServerAllocationPolicy, and an error, if there is any.
func (c *FakeGameServerAllocationPolicies) Update(ctx context.Context, gameServerAllocationPolicy *multiclusterv1.GameServerAllocationPolicy, opts v1.UpdateOptions) (result *multiclusterv1.GameServerAllocationPolicy, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(gameserverallocationpoliciesResource, c.ns, gameServerAllocationPolicy), &multiclusterv1.GameServerAllocationPolicy{})

	if obj == nil {
		return nil, err
	}
	return obj.(*multiclusterv1.GameServerAllocationPolicy), err
}

// Delete takes name of the gameServerAllocationPolicy and deletes it. Returns an error if one occurs.
func (c *FakeGameServerAllocationPolicies) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteActionWithOptions(gameserverallocationpoliciesResource, c.ns, name, opts), &multiclusterv1.GameServerAllocationPolicy{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeGameServerAllocationPolicies) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(gameserverallocationpoliciesResource, c.ns, listOpts)

	_, err := c.Fake.Invokes(action, &multiclusterv1.GameServerAllocationPolicyList{})
	return err
}

// Patch applies the patch and returns the patched gameServerAllocationPolicy.
func (c *FakeGameServerAllocationPolicies) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *multiclusterv1.GameServerAllocationPolicy, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(gameserverallocationpoliciesResource, c.ns, name, pt, data, subresources...), &multiclusterv1.GameServerAllocationPolicy{})

	if obj == nil {
		return nil, err
	}
	return obj.(*multiclusterv1.GameServerAllocationPolicy), err
}
