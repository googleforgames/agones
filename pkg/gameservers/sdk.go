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

package gameservers

import (
	"agones.dev/agones/pkg/apis/stable/v1alpha1"
	"agones.dev/agones/pkg/sdk"
)

const (
	// metadataPrefix prefix for labels and annotations
	metadataPrefix = "stable.agones.dev/sdk-"
)

// convert converts a K8s GameServer object, into a gRPC SDK GameServer object
func convert(gs *v1alpha1.GameServer) *sdk.GameServer {
	meta := gs.ObjectMeta
	status := gs.Status
	health := gs.Spec.Health
	result := &sdk.GameServer{
		ObjectMeta: &sdk.GameServer_ObjectMeta{
			Name:              meta.Name,
			Namespace:         meta.Namespace,
			Uid:               string(meta.UID),
			ResourceVersion:   meta.ResourceVersion,
			Generation:        meta.Generation,
			CreationTimestamp: meta.CreationTimestamp.Unix(),
			Annotations:       meta.Annotations,
			Labels:            meta.Labels,
		},
		Spec: &sdk.GameServer_Spec{
			Health: &sdk.GameServer_Spec_Health{
				Disabled:            health.Disabled,
				PeriodSeconds:       health.PeriodSeconds,
				FailureThreshold:    health.FailureThreshold,
				InitialDelaySeconds: health.InitialDelaySeconds,
			},
		},
		Status: &sdk.GameServer_Status{
			State:   string(status.State),
			Address: status.Address,
		},
	}
	if meta.DeletionTimestamp != nil {
		result.ObjectMeta.DeletionTimestamp = meta.DeletionTimestamp.Unix()
	}

	// loop around and add all the ports
	for _, p := range status.Ports {
		grpcPort := &sdk.GameServer_Status_Port{
			Name: p.Name,
			Port: p.Port,
		}
		result.Status.Ports = append(result.Status.Ports, grpcPort)
	}

	return result
}
