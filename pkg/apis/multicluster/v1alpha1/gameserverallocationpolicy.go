// Copyright 2019 Google LLC All Rights Reserved.
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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GameServerAllocationPolicySpec defines the desired state of GameServerAllocationPolicy
type GameServerAllocationPolicySpec struct {
	// +kubebuilder:validation:Minimum=0
	Priority int `json:"priority"`
	// +kubebuilder:validation:Minimum=0
	Weight         int                   `json:"weight"`
	ConnectionInfo ClusterConnectionInfo `json:"connectionInfo,omitempty"`
}

// ClusterConnectionInfo defines the connection information for a cluster
type ClusterConnectionInfo struct {
	ClusterName       string `json:"clusterName"`
	APIServerEndpoint string `json:"apiServerEndpoint"`
	SecretName        string `json:"secretName"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// GameServerAllocationPolicy is the Schema for the gameserverallocationpolicies API
// +k8s:openapi-gen=true
type GameServerAllocationPolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec GameServerAllocationPolicySpec `json:"spec,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// GameServerAllocationPolicyList contains a list of GameServerAllocationPolicy
type GameServerAllocationPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []GameServerAllocationPolicy `json:"items"`
}
