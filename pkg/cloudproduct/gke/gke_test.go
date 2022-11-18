// Copyright 2022 Google LLC All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package gke

import (
	"testing"

	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestSyncPodPortsToGameServer(t *testing.T) {
	assignmentAnnotation := map[string]string{hostPortAssignmentAnnotation: `{"min":7000,"max":8000,"portsAssigned":{"7001":7737,"7002":7738}}`}
	badAnnotation := map[string]string{hostPortAssignmentAnnotation: `good luck parsing this as JSON`}
	for name, tc := range map[string]struct {
		gs      *agonesv1.GameServer
		pod     *corev1.Pod
		wantGS  *agonesv1.GameServer
		wantErr bool
	}{
		"no ports => no change": {
			gs:     &agonesv1.GameServer{},
			pod:    testPod(nil),
			wantGS: &agonesv1.GameServer{},
		},
		"no annotation => no change": {
			gs:     testGameServer([]int32{7777}, nil),
			pod:    testPod(nil),
			wantGS: testGameServer([]int32{7777}, nil),
		},
		"annotation => ports mapped": {
			gs:     testGameServer([]int32{7002, 7001, 7002}, nil),
			pod:    testPod(assignmentAnnotation),
			wantGS: testGameServer([]int32{7738, 7737, 7738}, nil),
		},
		"annotation, but ports already assigned => ports mapped": {
			gs:     testGameServer([]int32{7001, 7002}, []int32{7001, 7002}),
			pod:    testPod(assignmentAnnotation),
			wantGS: testGameServer([]int32{7001, 7002}, []int32{7001, 7002}),
		},
		"bad annotation": {
			gs:      testGameServer([]int32{7002, 7001, 7002}, nil),
			pod:     testPod(badAnnotation),
			wantErr: true,
		},
	} {
		t.Run(name, func(t *testing.T) {
			oldPod := tc.pod.DeepCopy()
			err := (&gkeAutopilot{}).SyncPodPortsToGameServer(tc.gs, tc.pod)
			if tc.wantErr {
				assert.NotNil(t, err)
				return
			}
			if assert.NoError(t, err) {
				require.Equal(t, tc.wantGS, tc.gs)
				require.Equal(t, oldPod, tc.pod)
			}
		})
	}
}

func TestValidateGameServer(t *testing.T) {
	for name, tc := range map[string]struct {
		ports []agonesv1.GameServerPort
		want  []metav1.StatusCause
	}{
		"no ports => validated": {},
		"good ports => validated": {
			ports: []agonesv1.GameServerPort{
				{
					Name:          "some-tcpudp",
					PortPolicy:    agonesv1.Dynamic,
					ContainerPort: 4321,
					Protocol:      agonesv1.ProtocolTCPUDP,
				},
				{
					Name:          "awesome-udp",
					PortPolicy:    agonesv1.Dynamic,
					ContainerPort: 1234,
					Protocol:      corev1.ProtocolUDP,
				},
				{
					Name:          "awesome-tcp",
					PortPolicy:    agonesv1.Dynamic,
					ContainerPort: 1234,
					Protocol:      corev1.ProtocolTCP,
				},
			},
		},
		"bad policy => fails validation": {
			ports: []agonesv1.GameServerPort{
				{
					Name:          "best-tcpudp",
					PortPolicy:    agonesv1.Dynamic,
					ContainerPort: 4321,
					Protocol:      agonesv1.ProtocolTCPUDP,
				},
				{
					Name:          "bad-udp",
					PortPolicy:    agonesv1.Static,
					ContainerPort: 1234,
					Protocol:      corev1.ProtocolUDP,
				},
				{
					Name:          "another-bad-udp",
					PortPolicy:    agonesv1.Static,
					ContainerPort: 1234,
					Protocol:      corev1.ProtocolUDP,
				},
			},
			want: []metav1.StatusCause{
				{
					Type:    "FieldValueInvalid",
					Message: "PortPolicy must be Dynamic on GKE Autopilot",
					Field:   "bad-udp.portPolicy",
				},
				{
					Type:    "FieldValueInvalid",
					Message: "PortPolicy must be Dynamic on GKE Autopilot",
					Field:   "another-bad-udp.portPolicy",
				},
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			causes := (&gkeAutopilot{}).ValidateGameServer(&agonesv1.GameServer{Spec: agonesv1.GameServerSpec{Ports: tc.ports}})
			require.Equal(t, tc.want, causes)
		})
	}
}

func TestAutopilotPortAllocator(t *testing.T) {
	for name, tc := range map[string]struct {
		ports          []agonesv1.GameServerPort
		wantPorts      []agonesv1.GameServerPort
		wantAnnotation bool
	}{
		"no ports => no change": {},
		"ports => assigned and annotated": {
			ports: []agonesv1.GameServerPort{
				{
					Name:          "some-tcpudp",
					PortPolicy:    agonesv1.Dynamic,
					ContainerPort: 4321,
					Protocol:      agonesv1.ProtocolTCPUDP,
				},
				{
					Name:          "awesome-udp",
					PortPolicy:    agonesv1.Dynamic,
					ContainerPort: 1234,
					Protocol:      corev1.ProtocolUDP,
				},
				{
					Name:          "awesome-tcp",
					PortPolicy:    agonesv1.Dynamic,
					ContainerPort: 1234,
					Protocol:      corev1.ProtocolTCP,
				},
				{
					Name:          "another-tcpudp",
					PortPolicy:    agonesv1.Dynamic,
					ContainerPort: 5678,
					Protocol:      agonesv1.ProtocolTCPUDP,
				},
			},
			wantPorts: []agonesv1.GameServerPort{
				{
					Name:          "some-tcpudp-tcp",
					PortPolicy:    agonesv1.Dynamic,
					ContainerPort: 4321,
					HostPort:      1,
					Protocol:      corev1.ProtocolTCP,
				},
				{
					Name:          "some-tcpudp-udp",
					PortPolicy:    agonesv1.Dynamic,
					ContainerPort: 4321,
					HostPort:      1,
					Protocol:      corev1.ProtocolUDP,
				},
				{
					Name:          "awesome-udp",
					PortPolicy:    agonesv1.Dynamic,
					ContainerPort: 1234,
					HostPort:      2,
					Protocol:      corev1.ProtocolUDP,
				},
				{
					Name:          "awesome-tcp",
					PortPolicy:    agonesv1.Dynamic,
					ContainerPort: 1234,
					HostPort:      3,
					Protocol:      corev1.ProtocolTCP,
				},
				{
					Name:          "another-tcpudp-tcp",
					PortPolicy:    agonesv1.Dynamic,
					ContainerPort: 5678,
					HostPort:      4,
					Protocol:      corev1.ProtocolTCP,
				},
				{
					Name:          "another-tcpudp-udp",
					PortPolicy:    agonesv1.Dynamic,
					ContainerPort: 5678,
					HostPort:      4,
					Protocol:      corev1.ProtocolUDP,
				},
			},
			wantAnnotation: true,
		},
		"bad policy => no change (should be rejected by webhooks previously)": {
			ports: []agonesv1.GameServerPort{
				{
					Name:          "awesome-udp",
					PortPolicy:    agonesv1.Static,
					ContainerPort: 1234,
					Protocol:      corev1.ProtocolUDP,
				},
			},
			wantPorts: []agonesv1.GameServerPort{
				{
					Name:          "awesome-udp",
					PortPolicy:    agonesv1.Static,
					ContainerPort: 1234,
					Protocol:      corev1.ProtocolUDP,
				},
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			gs := (&autopilotPortAllocator{minPort: 8000, maxPort: 9000}).Allocate(&agonesv1.GameServer{Spec: agonesv1.GameServerSpec{Ports: tc.ports}})
			wantGS := &agonesv1.GameServer{Spec: agonesv1.GameServerSpec{Ports: tc.wantPorts}}
			if tc.wantAnnotation {
				wantGS.Spec.Template = corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{"autopilot.gke.io/host-port-assignment": `{"min":8000,"max":9000}`},
					},
				}
			}
			require.Equal(t, wantGS, gs)
		})
	}
}

func testPod(annotations map[string]string) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "best-game-server",
			Namespace:   "best-game",
			Annotations: annotations,
		},
		TypeMeta: metav1.TypeMeta{Kind: "Pod"},
	}
}

func testGameServer(portSpecIn []int32, portStatusIn []int32) *agonesv1.GameServer {
	var portSpec []agonesv1.GameServerPort
	for _, port := range portSpecIn {
		portSpec = append(portSpec, agonesv1.GameServerPort{HostPort: port})
	}
	var portStatus []agonesv1.GameServerStatusPort
	for _, port := range portStatusIn {
		portStatus = append(portStatus, agonesv1.GameServerStatusPort{Port: port})
	}
	return &agonesv1.GameServer{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "best-game-server",
			Namespace: "best-game",
		},
		Spec: agonesv1.GameServerSpec{
			Ports: portSpec,
		},
		Status: agonesv1.GameServerStatus{
			Ports: portStatus,
		},
	}
}
