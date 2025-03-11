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

// Code generated by applyconfiguration-gen. DO NOT EDIT.

package v1

import (
	v1 "k8s.io/api/admissionregistration/v1"
)

// WebhookPolicyApplyConfiguration represents a declarative configuration of the WebhookPolicy type for use
// with apply.
type WebhookPolicyApplyConfiguration struct {
	URL      *string              `json:"url,omitempty"`
	Service  *v1.ServiceReference `json:"service,omitempty"`
	CABundle []byte               `json:"caBundle,omitempty"`
}

// WebhookPolicyApplyConfiguration constructs a declarative configuration of the WebhookPolicy type for use with
// apply.
func WebhookPolicy() *WebhookPolicyApplyConfiguration {
	return &WebhookPolicyApplyConfiguration{}
}

// WithURL sets the URL field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the URL field is set to the value of the last call.
func (b *WebhookPolicyApplyConfiguration) WithURL(value string) *WebhookPolicyApplyConfiguration {
	b.URL = &value
	return b
}

// WithService sets the Service field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Service field is set to the value of the last call.
func (b *WebhookPolicyApplyConfiguration) WithService(value v1.ServiceReference) *WebhookPolicyApplyConfiguration {
	b.Service = &value
	return b
}

// WithCABundle adds the given value to the CABundle field in the declarative configuration
// and returns the receiver, so that objects can be build by chaining "With" function invocations.
// If called multiple times, values provided by each call will be appended to the CABundle field.
func (b *WebhookPolicyApplyConfiguration) WithCABundle(values ...byte) *WebhookPolicyApplyConfiguration {
	for i := range values {
		b.CABundle = append(b.CABundle, values[i])
	}
	return b
}
