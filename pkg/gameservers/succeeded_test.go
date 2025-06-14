// Copyright 2025 Google LLC All Rights Reserved.
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
	"time"

	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	agtesting "agones.dev/agones/pkg/testing"
	"github.com/heptiolabs/healthcheck"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8stesting "k8s.io/client-go/testing"
)

func TestSucceededControllerSyncGameServer(t *testing.T) {
	type expected struct {
		updated     bool
		updateTests func(t *testing.T, gs *agonesv1.GameServer)
		postTests   func(t *testing.T, mocks agtesting.Mocks)
	}
	fixtures := map[string]struct {
		setup    func(*agonesv1.GameServer, *corev1.Pod) (*agonesv1.GameServer, *corev1.Pod)
		expected expected
	}{
		"pod exists but not in Succeeded state": {
			setup: func(gs *agonesv1.GameServer, pod *corev1.Pod) (*agonesv1.GameServer, *corev1.Pod) {
				return gs, pod
			},
			expected: expected{
				updated:     false,
				updateTests: func(_ *testing.T, _ *agonesv1.GameServer) {},
				postTests:   func(_ *testing.T, _ agtesting.Mocks) {},
			},
		},
		"pod exists and is in Succeeded state": {
			setup: func(gs *agonesv1.GameServer, pod *corev1.Pod) (*agonesv1.GameServer, *corev1.Pod) {
				pod.Status.Phase = corev1.PodSucceeded
				return gs, pod
			},
			expected: expected{
				updated: true,
				updateTests: func(t *testing.T, gs *agonesv1.GameServer) {
					require.Equal(t, agonesv1.GameServerStateShutdown, gs.Status.State)
				},
				postTests: func(t *testing.T, m agtesting.Mocks) {
					agtesting.AssertEventContains(t, m.FakeRecorder.Events, "Normal Shutdown Pod is in Succeeded state")
				},
			},
		},
		"pod doesn't exist": {
			setup: func(gs *agonesv1.GameServer, _ *corev1.Pod) (*agonesv1.GameServer, *corev1.Pod) {
				return gs, nil
			},
			expected: expected{
				updated:     false,
				updateTests: func(_ *testing.T, _ *agonesv1.GameServer) {},
				postTests:   func(_ *testing.T, _ agtesting.Mocks) {},
			},
		},
		"game server not found": {
			setup: func(_ *agonesv1.GameServer, _ *corev1.Pod) (*agonesv1.GameServer, *corev1.Pod) {
				return nil, nil
			},
			expected: expected{
				updated:     false,
				updateTests: func(_ *testing.T, _ *agonesv1.GameServer) {},
				postTests:   func(_ *testing.T, _ agtesting.Mocks) {},
			},
		},
		"game server is being deleted": {
			setup: func(gs *agonesv1.GameServer, pod *corev1.Pod) (*agonesv1.GameServer, *corev1.Pod) {
				now := metav1.Now()
				gs.ObjectMeta.DeletionTimestamp = &now
				pod.Status.Phase = corev1.PodSucceeded
				return gs, pod
			},
			expected: expected{
				updated:     false,
				updateTests: func(_ *testing.T, _ *agonesv1.GameServer) {},
				postTests:   func(_ *testing.T, _ agtesting.Mocks) {},
			},
		},
		"game server is already in Shutdown state": {
			setup: func(gs *agonesv1.GameServer, pod *corev1.Pod) (*agonesv1.GameServer, *corev1.Pod) {
				gs.Status.State = agonesv1.GameServerStateShutdown
				pod.Status.Phase = corev1.PodSucceeded
				return gs, pod
			},
			expected: expected{
				updated:     false,
				updateTests: func(_ *testing.T, _ *agonesv1.GameServer) {},
				postTests:   func(_ *testing.T, _ agtesting.Mocks) {},
			},
		},
		"game server is in Error state": {
			setup: func(gs *agonesv1.GameServer, pod *corev1.Pod) (*agonesv1.GameServer, *corev1.Pod) {
				gs.Status.State = agonesv1.GameServerStateError
				pod.Status.Phase = corev1.PodSucceeded
				return gs, pod
			},
			expected: expected{
				updated:     false,
				updateTests: func(_ *testing.T, _ *agonesv1.GameServer) {},
				postTests:   func(_ *testing.T, _ agtesting.Mocks) {},
			},
		},
		"game server is in Unhealthy state": {
			setup: func(gs *agonesv1.GameServer, pod *corev1.Pod) (*agonesv1.GameServer, *corev1.Pod) {
				gs.Status.State = agonesv1.GameServerStateUnhealthy
				pod.Status.Phase = corev1.PodSucceeded
				return gs, pod
			},
			expected: expected{
				updated:     false,
				updateTests: func(_ *testing.T, _ *agonesv1.GameServer) {},
				postTests:   func(_ *testing.T, _ agtesting.Mocks) {},
			},
		},
		"pod is not a gameserver pod": {
			setup: func(gs *agonesv1.GameServer, _ *corev1.Pod) (*agonesv1.GameServer, *corev1.Pod) {
				pod := &corev1.Pod{ObjectMeta: gs.ObjectMeta}
				pod.Status.Phase = corev1.PodSucceeded
				return gs, pod
			},
			expected: expected{
				updated:     false,
				updateTests: func(_ *testing.T, _ *agonesv1.GameServer) {},
				postTests:   func(_ *testing.T, _ agtesting.Mocks) {},
			},
		},
		"pod is in terminating state": {
			setup: func(gs *agonesv1.GameServer, pod *corev1.Pod) (*agonesv1.GameServer, *corev1.Pod) {
				now := metav1.Now()
				pod.ObjectMeta.DeletionTimestamp = &now
				pod.Status.Phase = corev1.PodSucceeded
				return gs, pod
			},
			expected: expected{
				updated:     false,
				updateTests: func(_ *testing.T, _ *agonesv1.GameServer) {},
				postTests:   func(_ *testing.T, _ agtesting.Mocks) {},
			},
		},
	}

	for k, v := range fixtures {
		t.Run(k, func(t *testing.T) {
			m := agtesting.NewMocks()
			c := NewSucceededController(healthcheck.NewHandler(), m.KubeClient, m.AgonesClient, m.KubeInformerFactory, m.AgonesInformerFactory)
			c.recorder = m.FakeRecorder

			gs := &agonesv1.GameServer{ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "default"},
				Spec: newSingleContainerSpec(), Status: agonesv1.GameServerStatus{}}
			gs.ApplyDefaults()

			pod, err := gs.Pod(agtesting.FakeAPIHooks{})
			require.NoError(t, err)

			gs, pod = v.setup(gs, pod)
			m.AgonesClient.AddReactor("list", "gameservers", func(_ k8stesting.Action) (bool, runtime.Object, error) {
				if gs != nil {
					return true, &agonesv1.GameServerList{Items: []agonesv1.GameServer{*gs}}, nil
				}
				return true, &agonesv1.GameServerList{Items: []agonesv1.GameServer{}}, nil
			})
			m.KubeClient.AddReactor("list", "pods", func(_ k8stesting.Action) (bool, runtime.Object, error) {
				if pod != nil {
					return true, &corev1.PodList{Items: []corev1.Pod{*pod}}, nil
				}
				return true, &corev1.PodList{Items: []corev1.Pod{}}, nil
			})

			updated := false
			m.AgonesClient.AddReactor("update", "gameservers", func(action k8stesting.Action) (bool, runtime.Object, error) {
				updated = true
				ua := action.(k8stesting.UpdateAction)
				gs := ua.GetObject().(*agonesv1.GameServer)
				v.expected.updateTests(t, gs)
				return true, gs, nil
			})

			// Use context explicitly to avoid unused import warning
			ctx, cancel := agtesting.StartInformers(m, c.gameServerSynced, c.podSynced)
			defer cancel()

			err = c.syncGameServer(ctx, "default/test")
			require.NoError(t, err)
			require.Equal(t, v.expected.updated, updated)
			v.expected.postTests(t, m)
		})
	}
}

func TestSucceededControllerRun(t *testing.T) {
	m := agtesting.NewMocks()
	c := NewSucceededController(healthcheck.NewHandler(), m.KubeClient, m.AgonesClient, m.KubeInformerFactory, m.AgonesInformerFactory)

	gs := &agonesv1.GameServer{ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "default"},
		Spec: newSingleContainerSpec(), Status: agonesv1.GameServerStatus{}}
	gs.ApplyDefaults()

	pod, err := gs.Pod(agtesting.FakeAPIHooks{})
	require.NoError(t, err)
	pod.Status.Phase = corev1.PodSucceeded

	m.AgonesClient.AddReactor("list", "gameservers", func(_ k8stesting.Action) (bool, runtime.Object, error) {
		return true, &agonesv1.GameServerList{Items: []agonesv1.GameServer{*gs}}, nil
	})
	m.KubeClient.AddReactor("list", "pods", func(_ k8stesting.Action) (bool, runtime.Object, error) {
		return true, &corev1.PodList{Items: []corev1.Pod{*pod}}, nil
	})

	updated := make(chan bool, 10)
	m.AgonesClient.AddReactor("update", "gameservers", func(action k8stesting.Action) (bool, runtime.Object, error) {
		ua := action.(k8stesting.UpdateAction)
		gs := ua.GetObject().(*agonesv1.GameServer)
		updated <- gs.Status.State == agonesv1.GameServerStateShutdown
		return true, gs, nil
	})

	ctx, cancel := agtesting.StartInformers(m, c.gameServerSynced, c.podSynced)
	defer cancel()

	go func() {
		err := c.Run(ctx, 1)
		require.NoError(t, err)
	}()

	select {
	case <-time.After(5 * time.Second):
		require.FailNow(t, "timeout waiting for GameServer to be marked as Shutdown")
	case value := <-updated:
		require.True(t, value)
	}
}
