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

// GameServerSetsGetter has a method to return a GameServerSetInterface.
// A group's client should implement this interface.
type GameServerSetsGetter interface {
	GameServerSets(namespace string) GameServerSetInterface
}

// GameServerSetInterface has methods to work with GameServerSet resources.
type GameServerSetInterface interface {
	Create(ctx context.Context, gameServerSet *v1.GameServerSet, opts metav1.CreateOptions) (*v1.GameServerSet, error)
	Update(ctx context.Context, gameServerSet *v1.GameServerSet, opts metav1.UpdateOptions) (*v1.GameServerSet, error)
	// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
	UpdateStatus(ctx context.Context, gameServerSet *v1.GameServerSet, opts metav1.UpdateOptions) (*v1.GameServerSet, error)
	Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error
	Get(ctx context.Context, name string, opts metav1.GetOptions) (*v1.GameServerSet, error)
	List(ctx context.Context, opts metav1.ListOptions) (*v1.GameServerSetList, error)
	Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *v1.GameServerSet, err error)
	Apply(ctx context.Context, gameServerSet *agonesv1.GameServerSetApplyConfiguration, opts metav1.ApplyOptions) (result *v1.GameServerSet, err error)
	// Add a +genclient:noStatus comment above the type to avoid generating ApplyStatus().
	ApplyStatus(ctx context.Context, gameServerSet *agonesv1.GameServerSetApplyConfiguration, opts metav1.ApplyOptions) (result *v1.GameServerSet, err error)
	GetScale(ctx context.Context, gameServerSetName string, options metav1.GetOptions) (*autoscalingv1.Scale, error)
	UpdateScale(ctx context.Context, gameServerSetName string, scale *autoscalingv1.Scale, opts metav1.UpdateOptions) (*autoscalingv1.Scale, error)

	GameServerSetExpansion
}

// gameServerSets implements GameServerSetInterface
type gameServerSets struct {
	*gentype.ClientWithListAndApply[*v1.GameServerSet, *v1.GameServerSetList, *agonesv1.GameServerSetApplyConfiguration]
}

// newGameServerSets returns a GameServerSets
func newGameServerSets(c *AgonesV1Client, namespace string) *gameServerSets {
	return &gameServerSets{
		gentype.NewClientWithListAndApply[*v1.GameServerSet, *v1.GameServerSetList, *agonesv1.GameServerSetApplyConfiguration](
			"gameserversets",
			c.RESTClient(),
			scheme.ParameterCodec,
			namespace,
			func() *v1.GameServerSet { return &v1.GameServerSet{} },
			func() *v1.GameServerSetList { return &v1.GameServerSetList{} }),
	}
}

// GetScale takes name of the gameServerSet, and returns the corresponding autoscalingv1.Scale object, and an error if there is any.
func (c *gameServerSets) GetScale(ctx context.Context, gameServerSetName string, options metav1.GetOptions) (result *autoscalingv1.Scale, err error) {
	result = &autoscalingv1.Scale{}
	err = c.GetClient().Get().
		Namespace(c.GetNamespace()).
		Resource("gameserversets").
		Name(gameServerSetName).
		SubResource("scale").
		VersionedParams(&options, scheme.ParameterCodec).
		Do(ctx).
		Into(result)
	return
}

// UpdateScale takes the top resource name and the representation of a scale and updates it. Returns the server's representation of the scale, and an error, if there is any.
func (c *gameServerSets) UpdateScale(ctx context.Context, gameServerSetName string, scale *autoscalingv1.Scale, opts metav1.UpdateOptions) (result *autoscalingv1.Scale, err error) {
	result = &autoscalingv1.Scale{}
	err = c.GetClient().Put().
		Namespace(c.GetNamespace()).
		Resource("gameserversets").
		Name(gameServerSetName).
		SubResource("scale").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(scale).
		Do(ctx).
		Into(result)
	return
}
