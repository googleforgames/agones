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
	http "net/http"

	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	scheme "agones.dev/agones/pkg/client/clientset/versioned/scheme"
	rest "k8s.io/client-go/rest"
)

type AgonesV1Interface interface {
	RESTClient() rest.Interface
	FleetsGetter
	GameServersGetter
	GameServerSetsGetter
}

// AgonesV1Client is used to interact with features provided by the agones.dev group.
type AgonesV1Client struct {
	restClient rest.Interface
}

func (c *AgonesV1Client) Fleets(namespace string) FleetInterface {
	return newFleets(c, namespace)
}

func (c *AgonesV1Client) GameServers(namespace string) GameServerInterface {
	return newGameServers(c, namespace)
}

func (c *AgonesV1Client) GameServerSets(namespace string) GameServerSetInterface {
	return newGameServerSets(c, namespace)
}

// NewForConfig creates a new AgonesV1Client for the given config.
// NewForConfig is equivalent to NewForConfigAndClient(c, httpClient),
// where httpClient was generated with rest.HTTPClientFor(c).
func NewForConfig(c *rest.Config) (*AgonesV1Client, error) {
	config := *c
	if err := setConfigDefaults(&config); err != nil {
		return nil, err
	}
	httpClient, err := rest.HTTPClientFor(&config)
	if err != nil {
		return nil, err
	}
	return NewForConfigAndClient(&config, httpClient)
}

// NewForConfigAndClient creates a new AgonesV1Client for the given config and http client.
// Note the http client provided takes precedence over the configured transport values.
func NewForConfigAndClient(c *rest.Config, h *http.Client) (*AgonesV1Client, error) {
	config := *c
	if err := setConfigDefaults(&config); err != nil {
		return nil, err
	}
	client, err := rest.RESTClientForConfigAndClient(&config, h)
	if err != nil {
		return nil, err
	}
	return &AgonesV1Client{client}, nil
}

// NewForConfigOrDie creates a new AgonesV1Client for the given config and
// panics if there is an error in the config.
func NewForConfigOrDie(c *rest.Config) *AgonesV1Client {
	client, err := NewForConfig(c)
	if err != nil {
		panic(err)
	}
	return client
}

// New creates a new AgonesV1Client for the given RESTClient.
func New(c rest.Interface) *AgonesV1Client {
	return &AgonesV1Client{c}
}

func setConfigDefaults(config *rest.Config) error {
	gv := agonesv1.SchemeGroupVersion
	config.GroupVersion = &gv
	config.APIPath = "/apis"
	config.NegotiatedSerializer = rest.CodecFactoryForGeneratedClient(scheme.Scheme, scheme.Codecs).WithoutConversion()

	if config.UserAgent == "" {
		config.UserAgent = rest.DefaultKubernetesUserAgent()
	}

	return nil
}

// RESTClient returns a RESTClient that is used to communicate
// with API server by this client implementation.
func (c *AgonesV1Client) RESTClient() rest.Interface {
	if c == nil {
		return nil
	}
	return c.restClient
}
