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

package v1alpha1

import (
	"time"

	v1alpha1 "agones.dev/agones/pkg/apis/multicluster/v1alpha1"
	scheme "agones.dev/agones/pkg/client/clientset/versioned/scheme"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// GameServerAllocationPoliciesGetter has a method to return a GameServerAllocationPolicyInterface.
// A group's client should implement this interface.
type GameServerAllocationPoliciesGetter interface {
	GameServerAllocationPolicies(namespace string) GameServerAllocationPolicyInterface
}

// GameServerAllocationPolicyInterface has methods to work with GameServerAllocationPolicy resources.
type GameServerAllocationPolicyInterface interface {
	Create(*v1alpha1.GameServerAllocationPolicy) (*v1alpha1.GameServerAllocationPolicy, error)
	Update(*v1alpha1.GameServerAllocationPolicy) (*v1alpha1.GameServerAllocationPolicy, error)
	Delete(name string, options *v1.DeleteOptions) error
	DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error
	Get(name string, options v1.GetOptions) (*v1alpha1.GameServerAllocationPolicy, error)
	List(opts v1.ListOptions) (*v1alpha1.GameServerAllocationPolicyList, error)
	Watch(opts v1.ListOptions) (watch.Interface, error)
	Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.GameServerAllocationPolicy, err error)
	GameServerAllocationPolicyExpansion
}

// gameServerAllocationPolicies implements GameServerAllocationPolicyInterface
type gameServerAllocationPolicies struct {
	client rest.Interface
	ns     string
}

// newGameServerAllocationPolicies returns a GameServerAllocationPolicies
func newGameServerAllocationPolicies(c *MulticlusterV1alpha1Client, namespace string) *gameServerAllocationPolicies {
	return &gameServerAllocationPolicies{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the gameServerAllocationPolicy, and returns the corresponding gameServerAllocationPolicy object, and an error if there is any.
func (c *gameServerAllocationPolicies) Get(name string, options v1.GetOptions) (result *v1alpha1.GameServerAllocationPolicy, err error) {
	result = &v1alpha1.GameServerAllocationPolicy{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("gameserverallocationpolicies").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of GameServerAllocationPolicies that match those selectors.
func (c *gameServerAllocationPolicies) List(opts v1.ListOptions) (result *v1alpha1.GameServerAllocationPolicyList, err error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result = &v1alpha1.GameServerAllocationPolicyList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("gameserverallocationpolicies").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Do().
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested gameServerAllocationPolicies.
func (c *gameServerAllocationPolicies) Watch(opts v1.ListOptions) (watch.Interface, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("gameserverallocationpolicies").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Watch()
}

// Create takes the representation of a gameServerAllocationPolicy and creates it.  Returns the server's representation of the gameServerAllocationPolicy, and an error, if there is any.
func (c *gameServerAllocationPolicies) Create(gameServerAllocationPolicy *v1alpha1.GameServerAllocationPolicy) (result *v1alpha1.GameServerAllocationPolicy, err error) {
	result = &v1alpha1.GameServerAllocationPolicy{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("gameserverallocationpolicies").
		Body(gameServerAllocationPolicy).
		Do().
		Into(result)
	return
}

// Update takes the representation of a gameServerAllocationPolicy and updates it. Returns the server's representation of the gameServerAllocationPolicy, and an error, if there is any.
func (c *gameServerAllocationPolicies) Update(gameServerAllocationPolicy *v1alpha1.GameServerAllocationPolicy) (result *v1alpha1.GameServerAllocationPolicy, err error) {
	result = &v1alpha1.GameServerAllocationPolicy{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("gameserverallocationpolicies").
		Name(gameServerAllocationPolicy.Name).
		Body(gameServerAllocationPolicy).
		Do().
		Into(result)
	return
}

// Delete takes name of the gameServerAllocationPolicy and deletes it. Returns an error if one occurs.
func (c *gameServerAllocationPolicies) Delete(name string, options *v1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("gameserverallocationpolicies").
		Name(name).
		Body(options).
		Do().
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *gameServerAllocationPolicies) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	var timeout time.Duration
	if listOptions.TimeoutSeconds != nil {
		timeout = time.Duration(*listOptions.TimeoutSeconds) * time.Second
	}
	return c.client.Delete().
		Namespace(c.ns).
		Resource("gameserverallocationpolicies").
		VersionedParams(&listOptions, scheme.ParameterCodec).
		Timeout(timeout).
		Body(options).
		Do().
		Error()
}

// Patch applies the patch and returns the patched gameServerAllocationPolicy.
func (c *gameServerAllocationPolicies) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.GameServerAllocationPolicy, err error) {
	result = &v1alpha1.GameServerAllocationPolicy{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("gameserverallocationpolicies").
		SubResource(subresources...).
		Name(name).
		Body(data).
		Do().
		Into(result)
	return
}
