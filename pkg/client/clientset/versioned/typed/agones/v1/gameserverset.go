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

package v1

import (
	"context"
	"time"

	v1 "agones.dev/agones/pkg/apis/agones/v1"
	scheme "agones.dev/agones/pkg/client/clientset/versioned/scheme"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
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
	UpdateStatus(ctx context.Context, gameServerSet *v1.GameServerSet, opts metav1.UpdateOptions) (*v1.GameServerSet, error)
	Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error
	Get(ctx context.Context, name string, opts metav1.GetOptions) (*v1.GameServerSet, error)
	List(ctx context.Context, opts metav1.ListOptions) (*v1.GameServerSetList, error)
	Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *v1.GameServerSet, err error)
	GetScale(ctx context.Context, gameServerSetName string, options metav1.GetOptions) (*autoscalingv1.Scale, error)
	UpdateScale(ctx context.Context, gameServerSetName string, scale *autoscalingv1.Scale, opts metav1.UpdateOptions) (*autoscalingv1.Scale, error)

	GameServerSetExpansion
}

// gameServerSets implements GameServerSetInterface
type gameServerSets struct {
	client rest.Interface
	ns     string
}

// newGameServerSets returns a GameServerSets
func newGameServerSets(c *AgonesV1Client, namespace string) *gameServerSets {
	return &gameServerSets{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the gameServerSet, and returns the corresponding gameServerSet object, and an error if there is any.
func (c *gameServerSets) Get(ctx context.Context, name string, options metav1.GetOptions) (result *v1.GameServerSet, err error) {
	result = &v1.GameServerSet{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("gameserversets").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do(ctx).
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of GameServerSets that match those selectors.
func (c *gameServerSets) List(ctx context.Context, opts metav1.ListOptions) (result *v1.GameServerSetList, err error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result = &v1.GameServerSetList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("gameserversets").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Do(ctx).
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested gameServerSets.
func (c *gameServerSets) Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("gameserversets").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Watch(ctx)
}

// Create takes the representation of a gameServerSet and creates it.  Returns the server's representation of the gameServerSet, and an error, if there is any.
func (c *gameServerSets) Create(ctx context.Context, gameServerSet *v1.GameServerSet, opts metav1.CreateOptions) (result *v1.GameServerSet, err error) {
	result = &v1.GameServerSet{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("gameserversets").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(gameServerSet).
		Do(ctx).
		Into(result)
	return
}

// Update takes the representation of a gameServerSet and updates it. Returns the server's representation of the gameServerSet, and an error, if there is any.
func (c *gameServerSets) Update(ctx context.Context, gameServerSet *v1.GameServerSet, opts metav1.UpdateOptions) (result *v1.GameServerSet, err error) {
	result = &v1.GameServerSet{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("gameserversets").
		Name(gameServerSet.Name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(gameServerSet).
		Do(ctx).
		Into(result)
	return
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *gameServerSets) UpdateStatus(ctx context.Context, gameServerSet *v1.GameServerSet, opts metav1.UpdateOptions) (result *v1.GameServerSet, err error) {
	result = &v1.GameServerSet{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("gameserversets").
		Name(gameServerSet.Name).
		SubResource("status").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(gameServerSet).
		Do(ctx).
		Into(result)
	return
}

// Delete takes name of the gameServerSet and deletes it. Returns an error if one occurs.
func (c *gameServerSets) Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("gameserversets").
		Name(name).
		Body(&opts).
		Do(ctx).
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *gameServerSets) DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error {
	var timeout time.Duration
	if listOpts.TimeoutSeconds != nil {
		timeout = time.Duration(*listOpts.TimeoutSeconds) * time.Second
	}
	return c.client.Delete().
		Namespace(c.ns).
		Resource("gameserversets").
		VersionedParams(&listOpts, scheme.ParameterCodec).
		Timeout(timeout).
		Body(&opts).
		Do(ctx).
		Error()
}

// Patch applies the patch and returns the patched gameServerSet.
func (c *gameServerSets) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *v1.GameServerSet, err error) {
	result = &v1.GameServerSet{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("gameserversets").
		Name(name).
		SubResource(subresources...).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}

// GetScale takes name of the gameServerSet, and returns the corresponding autoscalingv1.Scale object, and an error if there is any.
func (c *gameServerSets) GetScale(ctx context.Context, gameServerSetName string, options metav1.GetOptions) (result *autoscalingv1.Scale, err error) {
	result = &autoscalingv1.Scale{}
	err = c.client.Get().
		Namespace(c.ns).
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
	err = c.client.Put().
		Namespace(c.ns).
		Resource("gameserversets").
		Name(gameServerSetName).
		SubResource("scale").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(scale).
		Do(ctx).
		Into(result)
	return
}
