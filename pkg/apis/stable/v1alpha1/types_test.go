// Copyright 2017 Google Inc. All Rights Reserved.
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
	"k8s.io/api/core/v1"
)

func TestGameServerFindGameServerContainer(t *testing.T) {
	t.Parallel()

	fixture := v1.Container{Name: "mycontainer", Image: "foo/mycontainer"}
	gs := &GameServer{
		Spec: GameServerSpec{
			Container: "mycontainer",
			Template: v1.PodTemplateSpec{
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						fixture,
						{Name: "notmycontainer", Image: "foo/notmycontainer"},
					},
				},
			},
		},
	}

	i, container, err := gs.FindGameServerContainer()
	assert.Nil(t, err)
	assert.Equal(t, fixture, container)
	container.Ports = append(container.Ports, v1.ContainerPort{HostPort: 1234})
	gs.Spec.Template.Spec.Containers[i] = container
	assert.Equal(t, gs.Spec.Template.Spec.Containers[0], container)
}

func TestGameServerApplyDefaults(t *testing.T) {
	t.Parallel()

	data := map[string]struct {
		gameServer        GameServer
		expectedContainer string
		expectedProtocol  v1.Protocol
		expectedState     State
	}{
		"set basic defaults on a very simple gameserver": {
			gameServer: GameServer{
				Spec: GameServerSpec{
					Template: v1.PodTemplateSpec{
						Spec: v1.PodSpec{Containers: []v1.Container{{Name: "testing", Image: "testing/image"}}}}},
			},
			expectedContainer: "testing",
			expectedProtocol:  "UDP",
			expectedState:     CreatingState,
		},
		"defaults are already set": {
			gameServer: GameServer{
				Spec: GameServerSpec{
					Container: "testing2", Protocol: "TCP",
					Template: v1.PodTemplateSpec{
						Spec: v1.PodSpec{Containers: []v1.Container{
							{Name: "testing", Image: "testing/image"},
							{Name: "testing2", Image: "testing/image2"}}},
					},
				},
				Status: GameServerStatus{State: "TestState"}},
			expectedContainer: "testing2",
			expectedProtocol:  "TCP",
			expectedState:     "TestState",
		},
	}

	for name, test := range data {
		t.Run(name, func(t *testing.T) {
			test.gameServer.ApplyDefaults()

			spec := test.gameServer.Spec
			assert.Equal(t, test.expectedContainer, spec.Container)
			assert.Equal(t, test.expectedProtocol, spec.Protocol)
			assert.Equal(t, test.expectedState, test.gameServer.Status.State)
		})
	}
}
