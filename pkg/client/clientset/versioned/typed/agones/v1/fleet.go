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

package v1

import (
	"context"

	v1 "agones.dev/agones/pkg/apis/agones/v1"
	agonesv1 "agones.dev/agones/pkg/client/applyconfiguration/agones/v1"
	scheme "agones.dev/agones/pkg/client/clientset/versioned/scheme"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	gentype "k8s.io/client-go/gentype"
)

// FleetsGetter has a method to return a FleetInterface.
// A group's client should implement this interface.
type FleetsGetter interface {
	Fleets(namespace string) FleetInterface
}

// FleetInterface has methods to work with Fleet resources.
type FleetInterface interface {
	Create(ctx context.Context, fleet *v1.Fleet, opts metav1.CreateOptions) (*v1.Fleet, error)
	Update(ctx context.Context, fleet *v1.Fleet, opts metav1.UpdateOptions) (*v1.Fleet, error)
	// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
	UpdateStatus(ctx context.Context, fleet *v1.Fleet, opts metav1.UpdateOptions) (*v1.Fleet, error)
	Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error
	Get(ctx context.Context, name string, opts metav1.GetOptions) (*v1.Fleet, error)
	List(ctx context.Context, opts metav1.ListOptions) (*v1.FleetList, error)
	Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *v1.Fleet, err error)
	Apply(ctx context.Context, fleet *agonesv1.FleetApplyConfiguration, opts metav1.ApplyOptions) (result *v1.Fleet, err error)
	// Add a +genclient:noStatus comment above the type to avoid generating ApplyStatus().
	ApplyStatus(ctx context.Context, fleet *agonesv1.FleetApplyConfiguration, opts metav1.ApplyOptions) (result *v1.Fleet, err error)
	GetScale(ctx context.Context, fleetName string, options metav1.GetOptions) (*autoscalingv1.Scale, error)
	UpdateScale(ctx context.Context, fleetName string, scale *autoscalingv1.Scale, opts metav1.UpdateOptions) (*autoscalingv1.Scale, error)

	FleetExpansion
}

// fleets implements FleetInterface
type fleets struct {
	*gentype.ClientWithListAndApply[*v1.Fleet, *v1.FleetList, *agonesv1.FleetApplyConfiguration]
}

// newFleets returns a Fleets
func newFleets(c *AgonesV1Client, namespace string) *fleets {
	return &fleets{
		gentype.NewClientWithListAndApply[*v1.Fleet, *v1.FleetList, *agonesv1.FleetApplyConfiguration](
			"fleets",
			c.RESTClient(),
			scheme.ParameterCodec,
			namespace,
			func() *v1.Fleet { return &v1.Fleet{} },
			func() *v1.FleetList { return &v1.FleetList{} }),
	}
}

// GetScale takes name of the fleet, and returns the corresponding autoscalingv1.Scale object, and an error if there is any.
func (c *fleets) GetScale(ctx context.Context, fleetName string, options metav1.GetOptions) (result *autoscalingv1.Scale, err error) {
	result = &autoscalingv1.Scale{}
	err = c.GetClient().Get().
		Namespace(c.GetNamespace()).
		Resource("fleets").
		Name(fleetName).
		SubResource("scale").
		VersionedParams(&options, scheme.ParameterCodec).
		Do(ctx).
		Into(result)
	return
}

// UpdateScale takes the top resource name and the representation of a scale and updates it. Returns the server's representation of the scale, and an error, if there is any.
func (c *fleets) UpdateScale(ctx context.Context, fleetName string, scale *autoscalingv1.Scale, opts metav1.UpdateOptions) (result *autoscalingv1.Scale, err error) {
	result = &autoscalingv1.Scale{}
	err = c.GetClient().Put().
		Namespace(c.GetNamespace()).
		Resource("fleets").
		Name(fleetName).
		SubResource("scale").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(scale).
		Do(ctx).
		Into(result)
	return
}
