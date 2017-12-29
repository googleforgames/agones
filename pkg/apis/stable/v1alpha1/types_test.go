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

	"github.com/agonio/agon/pkg/apis/stable"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGameServerFindGameServerContainer(t *testing.T) {
	t.Parallel()

	fixture := corev1.Container{Name: "mycontainer", Image: "foo/mycontainer"}
	gs := &GameServer{
		Spec: GameServerSpec{
			Container: "mycontainer",
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
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
	container.Ports = append(container.Ports, corev1.ContainerPort{HostPort: 1234})
	gs.Spec.Template.Spec.Containers[i] = container
	assert.Equal(t, gs.Spec.Template.Spec.Containers[0], container)
}

func TestGameServerApplyDefaults(t *testing.T) {
	t.Parallel()

	data := map[string]struct {
		gameServer        GameServer
		expectedContainer string
		expectedProtocol  corev1.Protocol
		expectedState     State
	}{
		"set basic defaults on a very simple gameserver": {
			gameServer: GameServer{
				Spec: GameServerSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "testing", Image: "testing/image"}}}}},
			},
			expectedContainer: "testing",
			expectedProtocol:  "UDP",
			expectedState:     Creating,
		},
		"defaults are already set": {
			gameServer: GameServer{
				Spec: GameServerSpec{
					Container: "testing2", Protocol: "TCP",
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{Containers: []corev1.Container{
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
			assert.Contains(t, test.gameServer.ObjectMeta.Finalizers, stable.GroupName)
			assert.Equal(t, test.expectedContainer, spec.Container)
			assert.Equal(t, test.expectedProtocol, spec.Protocol)
			assert.Equal(t, test.expectedState, test.gameServer.Status.State)
		})
	}
}

func TestGameServerPod(t *testing.T) {
	fixture := &GameServer{ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "default", UID: "1234"},
		Spec: GameServerSpec{
			ContainerPort: 7777,
			HostPort:      9999,
			PortPolicy:    Static,
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{Name: "container", Image: "container/image"}},
				},
			},
		}, Status: GameServerStatus{State: Creating}}
	fixture.ApplyDefaults()

	pod, err := fixture.Pod()
	assert.Nil(t, err, "Pod should not return an error")
	assert.Equal(t, fixture.ObjectMeta.Name+"-", pod.ObjectMeta.GenerateName)
	assert.Equal(t, fixture.ObjectMeta.Namespace, pod.ObjectMeta.Namespace)
	assert.Equal(t, "gameserver", pod.ObjectMeta.Labels[stable.GroupName+"/role"])
	assert.Equal(t, fixture.ObjectMeta.Name, pod.ObjectMeta.Labels[GameServerPodLabel])
	assert.True(t, metav1.IsControlledBy(pod, fixture))
	assert.Equal(t, fixture.Spec.HostPort, pod.Spec.Containers[0].Ports[0].HostPort)
	assert.Equal(t, fixture.Spec.ContainerPort, pod.Spec.Containers[0].Ports[0].ContainerPort)
	assert.Equal(t, corev1.Protocol("UDP"), pod.Spec.Containers[0].Ports[0].Protocol)
	logrus.SetFormatter(&logrus.JSONFormatter{})
	assert.True(t, metav1.IsControlledBy(pod, fixture))

	sidecar := corev1.Container{Name: "sidecar", Image: "container/sidecar"}
	pod, err = fixture.Pod(sidecar)
	assert.Nil(t, err, "Pod should not return an error")
	assert.Equal(t, fixture.ObjectMeta.Name+"-", pod.ObjectMeta.GenerateName)
	assert.Len(t, pod.Spec.Containers, 2, "Should have two containers")
	assert.Equal(t, "container", pod.Spec.Containers[0].Name)
	assert.Equal(t, "sidecar", pod.Spec.Containers[1].Name)
	assert.True(t, metav1.IsControlledBy(pod, fixture))
}
