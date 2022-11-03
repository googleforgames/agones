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
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
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
				if diff := cmp.Diff(tc.wantGS, tc.gs); diff != "" {
					t.Errorf("GameServer diff (-want +got):\n%s", diff)
				}
				if diff := cmp.Diff(oldPod, tc.pod); diff != "" {
					t.Errorf("Pod was modified (-old +new):\n%s", diff)
				}
			}
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
