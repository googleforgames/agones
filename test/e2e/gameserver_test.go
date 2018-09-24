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

package e2e

import (
	"fmt"
	"testing"
	"time"

	"agones.dev/agones/pkg/apis/stable/v1alpha1"
	e2eframework "agones.dev/agones/test/e2e/framework"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

func TestCreateConnect(t *testing.T) {
	t.Parallel()
	gs := defaultGameServer()
	readyGs, err := framework.CreateGameServerAndWaitUntilReady(defaultNs, gs)

	if err != nil {
		t.Fatalf("Could not get a GameServer ready: %v", err)
	}
	assert.Equal(t, len(readyGs.Status.Ports), 1)
	assert.NotEmpty(t, readyGs.Status.Ports[0].Port)
	assert.NotEmpty(t, readyGs.Status.Address)
	assert.NotEmpty(t, readyGs.Status.NodeName)
	assert.Equal(t, readyGs.Status.State, v1alpha1.Ready)

	reply, err := e2eframework.PingGameServer("Hello World !", fmt.Sprintf("%s:%d", readyGs.Status.Address,
		readyGs.Status.Ports[0].Port))

	if err != nil {
		t.Fatalf("Could ping GameServer: %v", err)
	}

	assert.Equal(t, reply, "ACK: Hello World !\n")
}

// nolint:dupl
func TestSDKSetLabel(t *testing.T) {
	t.Parallel()
	gs := defaultGameServer()
	readyGs, err := framework.CreateGameServerAndWaitUntilReady(defaultNs, gs)
	if err != nil {
		t.Fatalf("Could not get a GameServer ready: %v", err)
	}

	assert.Equal(t, readyGs.Status.State, v1alpha1.Ready)
	reply, err := e2eframework.PingGameServer("LABEL", fmt.Sprintf("%s:%d", readyGs.Status.Address,
		readyGs.Status.Ports[0].Port))

	if err != nil {
		t.Fatalf("Could ping GameServer: %v", err)
	}

	assert.Equal(t, "ACK: LABEL\n", reply)

	// the label is set in a queue, so it may take a moment
	err = wait.PollImmediate(time.Second, 10*time.Second, func() (bool, error) {
		gs, err = framework.AgonesClient.StableV1alpha1().GameServers(defaultNs).Get(readyGs.ObjectMeta.Name, metav1.GetOptions{})
		if err != nil {
			return true, err
		}
		return gs.ObjectMeta.Labels != nil, nil
	})

	assert.Nil(t, err)
	assert.NotEmpty(t, gs.ObjectMeta.Labels["stable.agones.dev/sdk-timestamp"])
}

// nolint:dupl
func TestSDKSetAnnotation(t *testing.T) {
	t.Parallel()
	gs := defaultGameServer()
	readyGs, err := framework.CreateGameServerAndWaitUntilReady(defaultNs, gs)
	if err != nil {
		t.Fatalf("Could not get a GameServer ready: %v", err)
	}

	assert.Equal(t, readyGs.Status.State, v1alpha1.Ready)
	reply, err := e2eframework.PingGameServer("ANNOTATION", fmt.Sprintf("%s:%d", readyGs.Status.Address,
		readyGs.Status.Ports[0].Port))

	if err != nil {
		t.Fatalf("Could ping GameServer: %v", err)
	}

	assert.Equal(t, "ACK: ANNOTATION\n", reply)

	// the label is set in a queue, so it may take a moment
	err = wait.PollImmediate(time.Second, 10*time.Second, func() (bool, error) {
		gs, err = framework.AgonesClient.StableV1alpha1().GameServers(defaultNs).Get(readyGs.ObjectMeta.Name, metav1.GetOptions{})
		if err != nil {
			return true, err
		}
		return gs.ObjectMeta.Annotations != nil, nil
	})

	assert.Nil(t, err)
	assert.NotEmpty(t, gs.ObjectMeta.Annotations["stable.agones.dev/sdk-timestamp"])
}

func defaultGameServer() *v1alpha1.GameServer {
	gs := &v1alpha1.GameServer{ObjectMeta: metav1.ObjectMeta{GenerateName: "udp-server", Namespace: defaultNs},
		Spec: v1alpha1.GameServerSpec{
			Container: "udp-server",
			Ports: []v1alpha1.GameServerPort{{
				ContainerPort: 7654,
				Name:          "gameport",
				PortPolicy:    v1alpha1.Dynamic,
				Protocol:      corev1.ProtocolUDP,
			}},
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Name:            "udp-server",
						Image:           framework.GameServerImage,
						ImagePullPolicy: corev1.PullIfNotPresent}},
				},
			},
		},
	}

	if framework.PullSecret != "" {
		gs.Spec.Template.Spec.ImagePullSecrets = []corev1.LocalObjectReference{{
			Name: framework.PullSecret}}
	}

	return gs
}
