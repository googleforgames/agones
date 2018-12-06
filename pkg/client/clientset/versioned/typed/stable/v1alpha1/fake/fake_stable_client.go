// Copyright 2018 Google Inc. All Rights Reserved.
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
	v1alpha1 "agones.dev/agones/pkg/client/clientset/versioned/typed/stable/v1alpha1"
	rest "k8s.io/client-go/rest"
	testing "k8s.io/client-go/testing"
)

type FakeStableV1alpha1 struct {
	*testing.Fake
}

func (c *FakeStableV1alpha1) Fleets(namespace string) v1alpha1.FleetInterface {
	return &FakeFleets{c, namespace}
}

func (c *FakeStableV1alpha1) FleetAllocations(namespace string) v1alpha1.FleetAllocationInterface {
	return &FakeFleetAllocations{c, namespace}
}

func (c *FakeStableV1alpha1) FleetAutoscalers(namespace string) v1alpha1.FleetAutoscalerInterface {
	return &FakeFleetAutoscalers{c, namespace}
}

func (c *FakeStableV1alpha1) GameServers(namespace string) v1alpha1.GameServerInterface {
	return &FakeGameServers{c, namespace}
}

func (c *FakeStableV1alpha1) GameServerAllocations(namespace string) v1alpha1.GameServerAllocationInterface {
	return &FakeGameServerAllocations{c, namespace}
}

func (c *FakeStableV1alpha1) GameServerSets(namespace string) v1alpha1.GameServerSetInterface {
	return &FakeGameServerSets{c, namespace}
}

// RESTClient returns a RESTClient that is used to communicate
// with API server by this client implementation.
func (c *FakeStableV1alpha1) RESTClient() rest.Interface {
	var ret *rest.RESTClient
	return ret
}
