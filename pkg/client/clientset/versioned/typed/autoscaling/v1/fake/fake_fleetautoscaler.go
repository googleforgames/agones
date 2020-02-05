// Copyright 2020 Google LLC All Rights Reserved.
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
	autoscalingv1 "agones.dev/agones/pkg/apis/autoscaling/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeFleetAutoscalers implements FleetAutoscalerInterface
type FakeFleetAutoscalers struct {
	Fake *FakeAutoscalingV1
	ns   string
}

var fleetautoscalersResource = schema.GroupVersionResource{Group: "autoscaling.agones.dev", Version: "v1", Resource: "fleetautoscalers"}

var fleetautoscalersKind = schema.GroupVersionKind{Group: "autoscaling.agones.dev", Version: "v1", Kind: "FleetAutoscaler"}

// Get takes name of the fleetAutoscaler, and returns the corresponding fleetAutoscaler object, and an error if there is any.
func (c *FakeFleetAutoscalers) Get(name string, options v1.GetOptions) (result *autoscalingv1.FleetAutoscaler, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(fleetautoscalersResource, c.ns, name), &autoscalingv1.FleetAutoscaler{})

	if obj == nil {
		return nil, err
	}
	return obj.(*autoscalingv1.FleetAutoscaler), err
}

// List takes label and field selectors, and returns the list of FleetAutoscalers that match those selectors.
func (c *FakeFleetAutoscalers) List(opts v1.ListOptions) (result *autoscalingv1.FleetAutoscalerList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(fleetautoscalersResource, fleetautoscalersKind, c.ns, opts), &autoscalingv1.FleetAutoscalerList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &autoscalingv1.FleetAutoscalerList{ListMeta: obj.(*autoscalingv1.FleetAutoscalerList).ListMeta}
	for _, item := range obj.(*autoscalingv1.FleetAutoscalerList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested fleetAutoscalers.
func (c *FakeFleetAutoscalers) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(fleetautoscalersResource, c.ns, opts))

}

// Create takes the representation of a fleetAutoscaler and creates it.  Returns the server's representation of the fleetAutoscaler, and an error, if there is any.
func (c *FakeFleetAutoscalers) Create(fleetAutoscaler *autoscalingv1.FleetAutoscaler) (result *autoscalingv1.FleetAutoscaler, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(fleetautoscalersResource, c.ns, fleetAutoscaler), &autoscalingv1.FleetAutoscaler{})

	if obj == nil {
		return nil, err
	}
	return obj.(*autoscalingv1.FleetAutoscaler), err
}

// Update takes the representation of a fleetAutoscaler and updates it. Returns the server's representation of the fleetAutoscaler, and an error, if there is any.
func (c *FakeFleetAutoscalers) Update(fleetAutoscaler *autoscalingv1.FleetAutoscaler) (result *autoscalingv1.FleetAutoscaler, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(fleetautoscalersResource, c.ns, fleetAutoscaler), &autoscalingv1.FleetAutoscaler{})

	if obj == nil {
		return nil, err
	}
	return obj.(*autoscalingv1.FleetAutoscaler), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakeFleetAutoscalers) UpdateStatus(fleetAutoscaler *autoscalingv1.FleetAutoscaler) (*autoscalingv1.FleetAutoscaler, error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateSubresourceAction(fleetautoscalersResource, "status", c.ns, fleetAutoscaler), &autoscalingv1.FleetAutoscaler{})

	if obj == nil {
		return nil, err
	}
	return obj.(*autoscalingv1.FleetAutoscaler), err
}

// Delete takes name of the fleetAutoscaler and deletes it. Returns an error if one occurs.
func (c *FakeFleetAutoscalers) Delete(name string, options *v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteAction(fleetautoscalersResource, c.ns, name), &autoscalingv1.FleetAutoscaler{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeFleetAutoscalers) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(fleetautoscalersResource, c.ns, listOptions)

	_, err := c.Fake.Invokes(action, &autoscalingv1.FleetAutoscalerList{})
	return err
}

// Patch applies the patch and returns the patched fleetAutoscaler.
func (c *FakeFleetAutoscalers) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *autoscalingv1.FleetAutoscaler, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(fleetautoscalersResource, c.ns, name, pt, data, subresources...), &autoscalingv1.FleetAutoscaler{})

	if obj == nil {
		return nil, err
	}
	return obj.(*autoscalingv1.FleetAutoscaler), err
}
