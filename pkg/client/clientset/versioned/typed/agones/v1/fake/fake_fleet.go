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

// Code generated by client-gen. DO NOT EDIT.

package fake

import (
	"context"
	json "encoding/json"
	"fmt"

	v1 "agones.dev/agones/pkg/apis/agones/v1"
	agonesv1 "agones.dev/agones/pkg/client/applyconfiguration/agones/v1"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeFleets implements FleetInterface
type FakeFleets struct {
	Fake *FakeAgonesV1
	ns   string
}

var fleetsResource = v1.SchemeGroupVersion.WithResource("fleets")

var fleetsKind = v1.SchemeGroupVersion.WithKind("Fleet")

// Get takes name of the fleet, and returns the corresponding fleet object, and an error if there is any.
func (c *FakeFleets) Get(ctx context.Context, name string, options metav1.GetOptions) (result *v1.Fleet, err error) {
	emptyResult := &v1.Fleet{}
	obj, err := c.Fake.
		Invokes(testing.NewGetActionWithOptions(fleetsResource, c.ns, name, options), emptyResult)

	if obj == nil {
		return emptyResult, err
	}
	return obj.(*v1.Fleet), err
}

// List takes label and field selectors, and returns the list of Fleets that match those selectors.
func (c *FakeFleets) List(ctx context.Context, opts metav1.ListOptions) (result *v1.FleetList, err error) {
	emptyResult := &v1.FleetList{}
	obj, err := c.Fake.
		Invokes(testing.NewListActionWithOptions(fleetsResource, fleetsKind, c.ns, opts), emptyResult)

	if obj == nil {
		return emptyResult, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1.FleetList{ListMeta: obj.(*v1.FleetList).ListMeta}
	for _, item := range obj.(*v1.FleetList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested fleets.
func (c *FakeFleets) Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchActionWithOptions(fleetsResource, c.ns, opts))

}

// Create takes the representation of a fleet and creates it.  Returns the server's representation of the fleet, and an error, if there is any.
func (c *FakeFleets) Create(ctx context.Context, fleet *v1.Fleet, opts metav1.CreateOptions) (result *v1.Fleet, err error) {
	emptyResult := &v1.Fleet{}
	obj, err := c.Fake.
		Invokes(testing.NewCreateActionWithOptions(fleetsResource, c.ns, fleet, opts), emptyResult)

	if obj == nil {
		return emptyResult, err
	}
	return obj.(*v1.Fleet), err
}

// Update takes the representation of a fleet and updates it. Returns the server's representation of the fleet, and an error, if there is any.
func (c *FakeFleets) Update(ctx context.Context, fleet *v1.Fleet, opts metav1.UpdateOptions) (result *v1.Fleet, err error) {
	emptyResult := &v1.Fleet{}
	obj, err := c.Fake.
		Invokes(testing.NewUpdateActionWithOptions(fleetsResource, c.ns, fleet, opts), emptyResult)

	if obj == nil {
		return emptyResult, err
	}
	return obj.(*v1.Fleet), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakeFleets) UpdateStatus(ctx context.Context, fleet *v1.Fleet, opts metav1.UpdateOptions) (result *v1.Fleet, err error) {
	emptyResult := &v1.Fleet{}
	obj, err := c.Fake.
		Invokes(testing.NewUpdateSubresourceActionWithOptions(fleetsResource, "status", c.ns, fleet, opts), emptyResult)

	if obj == nil {
		return emptyResult, err
	}
	return obj.(*v1.Fleet), err
}

// Delete takes name of the fleet and deletes it. Returns an error if one occurs.
func (c *FakeFleets) Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteActionWithOptions(fleetsResource, c.ns, name, opts), &v1.Fleet{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeFleets) DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error {
	action := testing.NewDeleteCollectionActionWithOptions(fleetsResource, c.ns, opts, listOpts)

	_, err := c.Fake.Invokes(action, &v1.FleetList{})
	return err
}

// Patch applies the patch and returns the patched fleet.
func (c *FakeFleets) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *v1.Fleet, err error) {
	emptyResult := &v1.Fleet{}
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceActionWithOptions(fleetsResource, c.ns, name, pt, data, opts, subresources...), emptyResult)

	if obj == nil {
		return emptyResult, err
	}
	return obj.(*v1.Fleet), err
}

// Apply takes the given apply declarative configuration, applies it and returns the applied fleet.
func (c *FakeFleets) Apply(ctx context.Context, fleet *agonesv1.FleetApplyConfiguration, opts metav1.ApplyOptions) (result *v1.Fleet, err error) {
	if fleet == nil {
		return nil, fmt.Errorf("fleet provided to Apply must not be nil")
	}
	data, err := json.Marshal(fleet)
	if err != nil {
		return nil, err
	}
	name := fleet.Name
	if name == nil {
		return nil, fmt.Errorf("fleet.Name must be provided to Apply")
	}
	emptyResult := &v1.Fleet{}
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceActionWithOptions(fleetsResource, c.ns, *name, types.ApplyPatchType, data, opts.ToPatchOptions()), emptyResult)

	if obj == nil {
		return emptyResult, err
	}
	return obj.(*v1.Fleet), err
}

// ApplyStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating ApplyStatus().
func (c *FakeFleets) ApplyStatus(ctx context.Context, fleet *agonesv1.FleetApplyConfiguration, opts metav1.ApplyOptions) (result *v1.Fleet, err error) {
	if fleet == nil {
		return nil, fmt.Errorf("fleet provided to Apply must not be nil")
	}
	data, err := json.Marshal(fleet)
	if err != nil {
		return nil, err
	}
	name := fleet.Name
	if name == nil {
		return nil, fmt.Errorf("fleet.Name must be provided to Apply")
	}
	emptyResult := &v1.Fleet{}
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceActionWithOptions(fleetsResource, c.ns, *name, types.ApplyPatchType, data, opts.ToPatchOptions(), "status"), emptyResult)

	if obj == nil {
		return emptyResult, err
	}
	return obj.(*v1.Fleet), err
}

// GetScale takes name of the fleet, and returns the corresponding scale object, and an error if there is any.
func (c *FakeFleets) GetScale(ctx context.Context, fleetName string, options metav1.GetOptions) (result *autoscalingv1.Scale, err error) {
	emptyResult := &autoscalingv1.Scale{}
	obj, err := c.Fake.
		Invokes(testing.NewGetSubresourceActionWithOptions(fleetsResource, c.ns, "scale", fleetName, options), emptyResult)

	if obj == nil {
		return emptyResult, err
	}
	return obj.(*autoscalingv1.Scale), err
}

// UpdateScale takes the representation of a scale and updates it. Returns the server's representation of the scale, and an error, if there is any.
func (c *FakeFleets) UpdateScale(ctx context.Context, fleetName string, scale *autoscalingv1.Scale, opts metav1.UpdateOptions) (result *autoscalingv1.Scale, err error) {
	emptyResult := &autoscalingv1.Scale{}
	obj, err := c.Fake.
		Invokes(testing.NewUpdateSubresourceActionWithOptions(fleetsResource, "scale", c.ns, scale, opts), &autoscalingv1.Scale{})

	if obj == nil {
		return emptyResult, err
	}
	return obj.(*autoscalingv1.Scale), err
}
