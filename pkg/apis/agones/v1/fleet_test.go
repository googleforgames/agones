// Copyright 2018 Google LLC All Rights Reserved.
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

package v1

import (
	"strings"
	"testing"

	"agones.dev/agones/pkg/apis"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/validation"
)

func TestFleetGameServerSetGameServer(t *testing.T) {
	f := Fleet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "namespace",
			UID:       "1234",
		},
		Spec: FleetSpec{
			Replicas:   10,
			Scheduling: apis.Packed,
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
	assert.Equal(t, f.ObjectMeta.Name, gsSet.ObjectMeta.Labels[FleetNameLabel])
	assert.Equal(t, int32(0), gsSet.Spec.Replicas)
	assert.Equal(t, f.Spec.Scheduling, gsSet.Spec.Scheduling)
	assert.Equal(t, f.Spec.Template, gsSet.Spec.Template)
	assert.True(t, metav1.IsControlledBy(gsSet, &f))
}

func TestFleetApplyDefaults(t *testing.T) {
	f := &Fleet{}

	// gate
	assert.EqualValues(t, "", f.Spec.Strategy.Type)
	assert.EqualValues(t, "", f.Spec.Scheduling)
	assert.EqualValues(t, 0, f.Spec.Replicas)

	f.ApplyDefaults()
	assert.Equal(t, appsv1.RollingUpdateDeploymentStrategyType, f.Spec.Strategy.Type)
	assert.Equal(t, "25%", f.Spec.Strategy.RollingUpdate.MaxUnavailable.String())
	assert.Equal(t, "25%", f.Spec.Strategy.RollingUpdate.MaxSurge.String())
	assert.Equal(t, apis.Packed, f.Spec.Scheduling)
	assert.Equal(t, int32(0), f.Spec.Replicas)
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

func TestFleetGameserverSpec(t *testing.T) {
	f := defaultFleet()
	f.ApplyDefaults()
	causes, ok := f.Validate()
	assert.True(t, ok)
	assert.Len(t, causes, 0)

	f.Spec.Template.Spec.Template =
		corev1.PodTemplateSpec{
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{{Name: "container", Image: "myimage"}, {Name: "container2", Image: "myimage"}},
			},
		}
	causes, ok = f.Validate()

	assert.False(t, ok)
	assert.Len(t, causes, 1)
	assert.Equal(t, "container", causes[0].Field)

	f.Spec.Template.Spec.Container = "testing"
	causes, ok = f.Validate()

	assert.False(t, ok)
	assert.Len(t, causes, 1)
	assert.Equal(t, "Could not find a container named testing", causes[0].Message)

	f.Spec.Template.Spec.Container = "container"
	causes, ok = f.Validate()
	assert.True(t, ok)
	assert.Len(t, causes, 0)

	// Verify RollingUpdate parameters validation
	percent := intstr.FromString("0%")
	f.Spec.Strategy.RollingUpdate.MaxUnavailable = &percent
	f.Spec.Strategy.RollingUpdate.MaxSurge = &percent
	causes, ok = f.Validate()
	assert.False(t, ok)
	assert.Len(t, causes, 2)

	intParam := intstr.FromInt(0)
	f.Spec.Strategy.RollingUpdate.MaxUnavailable = &intParam
	f.Spec.Strategy.RollingUpdate.MaxSurge = &intParam
	causes, ok = f.Validate()
	assert.False(t, ok)
	assert.Len(t, causes, 2)

	percent = intstr.FromString("2a")
	f.Spec.Strategy.RollingUpdate.MaxUnavailable = &percent
	f.Spec.Strategy.RollingUpdate.MaxSurge = &percent
	causes, ok = f.Validate()
	assert.False(t, ok)
	assert.Len(t, causes, 2)

	longName := strings.Repeat("f", validation.LabelValueMaxLength+1)
	f = defaultFleet()
	f.ApplyDefaults()
	f.Spec.Template.ObjectMeta.Labels = make(map[string]string)
	f.Spec.Template.ObjectMeta.Labels["label"] = longName
	causes, ok = f.Validate()
	assert.False(t, ok)
	assert.Len(t, causes, 1)

	f = defaultFleet()
	f.ApplyDefaults()
	f.Spec.Template.Spec.Template.ObjectMeta.Labels = make(map[string]string)
	f.Spec.Template.Spec.Template.ObjectMeta.Labels["label"] = longName
	causes, ok = f.Validate()
	assert.False(t, ok)
	assert.Len(t, causes, 1)

	// Annotations test
	f = defaultFleet()
	f.ApplyDefaults()
	f.Spec.Template.Spec.Template.ObjectMeta.Annotations = make(map[string]string)
	f.Spec.Template.Spec.Template.ObjectMeta.Annotations[longName] = ""
	causes, ok = f.Validate()
	assert.False(t, ok)
	assert.Len(t, causes, 1)

	// Strategy Type validation test
	f = defaultFleet()
	f.ApplyDefaults()
	f.Spec.Strategy.Type = appsv1.DeploymentStrategyType("")
	causes, ok = f.Validate()
	assert.False(t, ok)
	assert.Len(t, causes, 1)
}

func TestFleetName(t *testing.T) {
	f := defaultFleet()
	f.ApplyDefaults()

	longName := strings.Repeat("f", validation.LabelValueMaxLength+1)
	f.Name = longName
	causes, ok := f.Validate()
	assert.False(t, ok)
	assert.Len(t, causes, 1)
	assert.Equal(t, "Name", causes[0].Field)

	f.Name = ""
	f.GenerateName = longName
	causes, ok = f.Validate()
	assert.True(t, ok)
	assert.Len(t, causes, 0)
}

func TestSumStatusReplicas(t *testing.T) {
	fixture := []*GameServerSet{
		{Status: GameServerSetStatus{Replicas: 10}},
		{Status: GameServerSetStatus{Replicas: 15}},
		{Status: GameServerSetStatus{Replicas: 5}},
	}

	assert.Equal(t, int32(30), SumStatusReplicas(fixture))
}

func defaultFleet() *Fleet {
	gs := GameServer{
		Spec: GameServerSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "testing", Image: "testing/image"}}}}},
	}
	return &Fleet{
		ObjectMeta: metav1.ObjectMeta{GenerateName: "simple-fleet-", Namespace: "defaultNs"},
		Spec: FleetSpec{
			Replicas: 2,
			Template: GameServerTemplateSpec{
				Spec: gs.Spec,
			},
		},
	}
}
