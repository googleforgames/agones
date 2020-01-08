// Copyright 2020 Google LLC All Rights Reserved.
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
	"testing"

	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestIsGameServerPod(t *testing.T) {
	t.Parallel()

	t.Run("it is a game server pod", func(t *testing.T) {
		gs := &agonesv1.GameServer{ObjectMeta: metav1.ObjectMeta{Name: "gameserver", UID: "1234"}, Spec: newSingleContainerSpec()}
		gs.ApplyDefaults()
		pod, err := gs.Pod()
		assert.Nil(t, err)

		assert.True(t, isGameServerPod(pod))
	})

	t.Run("it is not a game server pod", func(t *testing.T) {
		pod := &corev1.Pod{}
		assert.False(t, isGameServerPod(pod))
	})
}

func TestAddress(t *testing.T) {
	t.Parallel()

	fixture := map[string]struct {
		node            *corev1.Node
		expectedAddress string
	}{
		"node with external ip": {
			node:            &corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: nodeFixtureName}, Status: corev1.NodeStatus{Addresses: []corev1.NodeAddress{{Address: "12.12.12.12", Type: corev1.NodeExternalIP}}}},
			expectedAddress: "12.12.12.12",
		},
		"node with an internal ip": {
			node:            &corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: nodeFixtureName}, Status: corev1.NodeStatus{Addresses: []corev1.NodeAddress{{Address: "11.11.11.11", Type: corev1.NodeInternalIP}}}},
			expectedAddress: "11.11.11.11",
		},
		"node with internal and external ip": {
			node: &corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: nodeFixtureName},
				Status: corev1.NodeStatus{Addresses: []corev1.NodeAddress{
					{Address: "9.9.9.8", Type: corev1.NodeExternalIP},
					{Address: "12.12.12.12", Type: corev1.NodeInternalIP},
				}}},
			expectedAddress: "9.9.9.8",
		},
	}

	dummyGS := &agonesv1.GameServer{}
	dummyGS.Name = "some-gs"

	for name, fixture := range fixture {
		t.Run(name, func(t *testing.T) {
			addr, err := address(fixture.node)
			assert.Nil(t, err)
			assert.Equal(t, fixture.expectedAddress, addr)
		})
	}
}

func TestApplyGameServerAddressAndPort(t *testing.T) {
	t.Parallel()

	gsFixture := &agonesv1.GameServer{ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "default"},
		Spec: newSingleContainerSpec(), Status: agonesv1.GameServerStatus{State: agonesv1.GameServerStateRequestReady}}
	gsFixture.ApplyDefaults()
	node := &corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: nodeFixtureName}, Status: corev1.NodeStatus{Addresses: []corev1.NodeAddress{{Address: ipFixture, Type: corev1.NodeExternalIP}}}}
	pod, err := gsFixture.Pod()
	assert.Nil(t, err)
	pod.Spec.NodeName = node.ObjectMeta.Name

	gs, err := applyGameServerAddressAndPort(gsFixture, node, pod)
	assert.Nil(t, err)
	assert.Equal(t, gs.Spec.Ports[0].HostPort, gs.Status.Ports[0].Port)
	assert.Equal(t, ipFixture, gs.Status.Address)
	assert.Equal(t, node.ObjectMeta.Name, gs.Status.NodeName)
}
