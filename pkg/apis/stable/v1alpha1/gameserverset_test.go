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
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGameServerSetGameServer(t *testing.T) {
	gsSet := GameServerSet{
		ObjectMeta: v1.ObjectMeta{
			Name:      "test",
			Namespace: "namespace",
			UID:       "1234",
		},
		Spec: GameServerSetSpec{
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

	gs := gsSet.GameServer()
	assert.Equal(t, "", gs.ObjectMeta.Name)
	assert.Equal(t, gsSet.ObjectMeta.Namespace, gs.ObjectMeta.Namespace)
	assert.Equal(t, gsSet.ObjectMeta.Name+"-", gs.ObjectMeta.GenerateName)
	assert.Equal(t, gsSet.ObjectMeta.Name, gs.ObjectMeta.Labels[GameServerSetGameServerLabel])
	assert.Equal(t, gs.Spec, gsSet.Spec.Template.Spec)
	assert.True(t, v1.IsControlledBy(gs, &gsSet))
}

func TestGameServerSetValidateUpdate(t *testing.T) {
	gsSet := GameServerSet{
		ObjectMeta: v1.ObjectMeta{Name: "test"},
		Spec: GameServerSetSpec{
			Replicas: 10,
			Template: GameServerTemplateSpec{
				Spec: GameServerSpec{Ports: []GameServerPort{{ContainerPort: 1234}}},
			},
		},
	}

	ok, causes := gsSet.ValidateUpdate(gsSet.DeepCopy())
	assert.True(t, ok)
	assert.Empty(t, causes)

	newGSS := gsSet.DeepCopy()
	newGSS.Spec.Replicas = 5
	ok, causes = gsSet.ValidateUpdate(newGSS)
	assert.True(t, ok)
	assert.Empty(t, causes)

	newGSS.Spec.Template.Spec.Ports[0].ContainerPort = 321
	ok, causes = gsSet.ValidateUpdate(newGSS)
	assert.False(t, ok)
	assert.Len(t, causes, 1)
	assert.Equal(t, "template", causes[0].Field)
}
