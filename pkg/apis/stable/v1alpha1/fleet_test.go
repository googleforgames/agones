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

package v1alpha1

import (
	"testing"

	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestFleetGameServerSetGameServer(t *testing.T) {
	f := Fleet{
		ObjectMeta: v1.ObjectMeta{
			Name:      "test",
			Namespace: "namespace",
			UID:       "1234",
		},
		Spec: FleetSpec{
			Replicas: 10,
			Template: GameServerTemplateSpec{
				Spec: GameServerSpec{
					Ports: []GameServerPort{{ContainerPort: 1234}},
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{{Name: "container", Image: "myimage"}},
						},
					},
				},
			},
		},
	}

	gsSet := f.GameServerSet()
	assert.Equal(t, "", gsSet.ObjectMeta.Name)
	assert.Equal(t, f.ObjectMeta.Namespace, gsSet.ObjectMeta.Namespace)
	assert.Equal(t, f.ObjectMeta.Name+"-", gsSet.ObjectMeta.GenerateName)
	assert.Equal(t, f.ObjectMeta.Name, gsSet.ObjectMeta.Labels[FleetGameServerSetLabel])
	assert.Equal(t, int32(0), gsSet.Spec.Replicas)
	assert.Equal(t, f.Spec.Template, gsSet.Spec.Template)
	assert.True(t, v1.IsControlledBy(gsSet, &f))
}

func TestFleetApplyDefaults(t *testing.T) {
	f := &Fleet{}

	// gate
	assert.EqualValues(t, "", f.Spec.Strategy.Type)

	f.ApplyDefaults()
	assert.Equal(t, appsv1.RollingUpdateDeploymentStrategyType, f.Spec.Strategy.Type)
	assert.Equal(t, "25%", f.Spec.Strategy.RollingUpdate.MaxUnavailable.String())
	assert.Equal(t, "25%", f.Spec.Strategy.RollingUpdate.MaxSurge.String())
}

func TestFleetUpperBoundReplicas(t *testing.T) {
	f := &Fleet{Spec: FleetSpec{Replicas: 10}}

	assert.Equal(t, int32(10), f.UpperBoundReplicas(12))
	assert.Equal(t, int32(10), f.UpperBoundReplicas(10))
	assert.Equal(t, int32(5), f.UpperBoundReplicas(5))
}

func TestFleetLowerBoundReplicas(t *testing.T) {
	f := &Fleet{Spec: FleetSpec{Replicas: 10}}

	assert.Equal(t, int32(5), f.LowerBoundReplicas(5))
	assert.Equal(t, int32(0), f.LowerBoundReplicas(0))
	assert.Equal(t, int32(0), f.LowerBoundReplicas(-5))
}

func TestSumStatusAllocatedReplicas(t *testing.T) {
	f := Fleet{}
	gsSet1 := f.GameServerSet()
	gsSet1.Status.AllocatedReplicas = 2

	gsSet2 := f.GameServerSet()
	gsSet2.Status.AllocatedReplicas = 3

	assert.Equal(t, int32(5), SumStatusAllocatedReplicas([]*GameServerSet{gsSet1, gsSet2}))
}

func TestSumStatusReplicas(t *testing.T) {
	fixture := []*GameServerSet{
		{Status: GameServerSetStatus{Replicas: 10}},
		{Status: GameServerSetStatus{Replicas: 15}},
		{Status: GameServerSetStatus{Replicas: 5}},
	}

	assert.Equal(t, int32(30), SumStatusReplicas(fixture))
}
