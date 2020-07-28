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

package gameservers

import (
	"errors"
	"testing"
	"time"

	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	agtesting "agones.dev/agones/pkg/testing"
	"github.com/heptiolabs/healthcheck"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"
	k8stesting "k8s.io/client-go/testing"
)

func TestHealthControllerFailedContainer(t *testing.T) {
	t.Parallel()

	m := agtesting.NewMocks()
	hc := NewHealthController(healthcheck.NewHandler(), m.KubeClient, m.AgonesClient, m.KubeInformerFactory, m.AgonesInformerFactory)

	gs := agonesv1.GameServer{ObjectMeta: metav1.ObjectMeta{Name: "test"}, Spec: newSingleContainerSpec()}
	gs.ApplyDefaults()

	pod, err := gs.Pod()
	require.NoError(t, err)
	pod.Status = corev1.PodStatus{ContainerStatuses: []corev1.ContainerStatus{{Name: gs.Spec.Container,
		State: corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{}}}}}

	assert.True(t, hc.failedContainer(pod))
	pod2 := pod.DeepCopy()

	pod.Status.ContainerStatuses[0].State.Terminated = nil
	assert.False(t, hc.failedContainer(pod))

	pod.Status.ContainerStatuses[0].LastTerminationState.Terminated = &corev1.ContainerStateTerminated{}
	assert.True(t, hc.failedContainer(pod))

	pod2.Status.ContainerStatuses[0].Name = "Not a matching name"
	assert.False(t, hc.failedContainer(pod2))
}

func TestHealthUnschedulableWithNoFreePorts(t *testing.T) {
	t.Parallel()

	m := agtesting.NewMocks()
	hc := NewHealthController(healthcheck.NewHandler(), m.KubeClient, m.AgonesClient, m.KubeInformerFactory, m.AgonesInformerFactory)

	gs := agonesv1.GameServer{ObjectMeta: metav1.ObjectMeta{Name: "test"}, Spec: newSingleContainerSpec()}
	gs.ApplyDefaults()

	pod, err := gs.Pod()
	require.NoError(t, err)

	pod.Status.Conditions = []corev1.PodCondition{
		{Type: corev1.PodScheduled, Reason: corev1.PodReasonUnschedulable,
			Message: "0/4 nodes are available: 4 node(s) didn't have free ports for the requestedpod ports."},
	}
	assert.True(t, hc.unschedulableWithNoFreePorts(pod))

	pod.Status.Conditions[0].Message = "not a real reason"
	assert.False(t, hc.unschedulableWithNoFreePorts(pod))
}

func TestHealthControllerSkipUnhealthy(t *testing.T) {
	t.Parallel()

	fixtures := map[string]struct {
		setup    func(*agonesv1.GameServer, *corev1.Pod)
		expected bool
	}{
		"scheduled and terminated container": {
			setup: func(gs *agonesv1.GameServer, pod *corev1.Pod) {
				gs.Status.State = agonesv1.GameServerStateScheduled
				pod.Status.ContainerStatuses = []corev1.ContainerStatus{{
					Name:  gs.Spec.Container,
					State: corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{}},
				}}
			},
			expected: true,
		},
		"after ready and terminated container": {
			setup: func(gs *agonesv1.GameServer, pod *corev1.Pod) {
				gs.Status.State = agonesv1.GameServerStateReady
				pod.Status.ContainerStatuses = []corev1.ContainerStatus{{
					Name:  gs.Spec.Container,
					State: corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{}},
				}}
			},
			expected: false,
		},
		"before ready, with no terminated container": {
			setup: func(gs *agonesv1.GameServer, pod *corev1.Pod) {
				gs.Status.State = agonesv1.GameServerStateScheduled
			},
			expected: false,
		},
		"after ready, with no terminated container": {
			setup: func(gs *agonesv1.GameServer, pod *corev1.Pod) {
				gs.Status.State = agonesv1.GameServerStateAllocated
			},
			expected: false,
		},
		"before ready, with a LastTerminated container": {
			setup: func(gs *agonesv1.GameServer, pod *corev1.Pod) {
				gs.Status.State = agonesv1.GameServerStateScheduled
				pod.Status.ContainerStatuses = []corev1.ContainerStatus{{
					Name:                 gs.Spec.Container,
					LastTerminationState: corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{}},
				}}
			},
			expected: true,
		},
		"after ready, with a LastTerminated container, not matching": {
			setup: func(gs *agonesv1.GameServer, pod *corev1.Pod) {
				gs.Status.State = agonesv1.GameServerStateReady
				gs.Annotations[agonesv1.GameServerReadyContainerIDAnnotation] = "4321"
				pod.Status.ContainerStatuses = []corev1.ContainerStatus{{
					ContainerID:          "1234",
					Name:                 gs.Spec.Container,
					LastTerminationState: corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{}},
				}}
			},
			expected: false,
		},
		"after ready, with a LastTerminated container, matching": {
			setup: func(gs *agonesv1.GameServer, pod *corev1.Pod) {
				gs.Status.State = agonesv1.GameServerStateReserved
				gs.Annotations[agonesv1.GameServerReadyContainerIDAnnotation] = "1234"
				pod.Status.ContainerStatuses = []corev1.ContainerStatus{{
					ContainerID:          "1234",
					Name:                 gs.Spec.Container,
					LastTerminationState: corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{}},
				}}
			},
			expected: true,
		},
		"pod is missing!": {
			setup: func(server *agonesv1.GameServer, pod *corev1.Pod) {
				pod.ObjectMeta.Name = "missing"
			},
			expected: false,
		},
	}

	for k, v := range fixtures {
		t.Run(k, func(t *testing.T) {
			m := agtesting.NewMocks()
			hc := NewHealthController(healthcheck.NewHandler(), m.KubeClient, m.AgonesClient, m.KubeInformerFactory, m.AgonesInformerFactory)
			gs := &agonesv1.GameServer{ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: defaultNs}, Spec: newSingleContainerSpec()}
			gs.ApplyDefaults()
			pod, err := gs.Pod()
			assert.NoError(t, err)

			v.setup(gs, pod)

			m.KubeClient.AddReactor("list", "pods", func(action k8stesting.Action) (bool, runtime.Object, error) {
				return true, &corev1.PodList{Items: []corev1.Pod{*pod}}, nil
			})

			_, cancel := agtesting.StartInformers(m, hc.podSynced)
			defer cancel()

			result, err := hc.skipUnhealthy(gs)
			assert.NoError(t, err)
			assert.Equal(t, v.expected, result)
		})
	}
}

func TestHealthControllerSyncGameServer(t *testing.T) {
	t.Parallel()

	type expected struct {
		updated bool
	}
	fixtures := map[string]struct {
		state     agonesv1.GameServerState
		podStatus *corev1.PodStatus
		expected  expected
	}{
		"started": {
			state: agonesv1.GameServerStateStarting,
			expected: expected{
				updated: true,
			},
		},
		"shutdown": {
			state: agonesv1.GameServerStateShutdown,
			expected: expected{
				updated: false,
			},
		},
		"unhealthy": {
			state: agonesv1.GameServerStateUnhealthy,
			expected: expected{
				updated: false,
			},
		},
		"ready": {
			state: agonesv1.GameServerStateReady,
			expected: expected{
				updated: true,
			},
		},
		"allocated": {
			state: agonesv1.GameServerStateAllocated,
			expected: expected{
				updated: true,
			},
		},
		"container failed before ready": {
			state: agonesv1.GameServerStateStarting,
			podStatus: &corev1.PodStatus{ContainerStatuses: []corev1.ContainerStatus{
				{Name: "container", State: corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{}}}}},
			expected: expected{updated: false},
		},
		"container failed after ready": {
			state: agonesv1.GameServerStateAllocated,
			podStatus: &corev1.PodStatus{ContainerStatuses: []corev1.ContainerStatus{
				{Name: "container", State: corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{}}}}},
			expected: expected{updated: true},
		},
	}

	for name, test := range fixtures {
		t.Run(name, func(t *testing.T) {
			m := agtesting.NewMocks()
			hc := NewHealthController(healthcheck.NewHandler(), m.KubeClient, m.AgonesClient, m.KubeInformerFactory, m.AgonesInformerFactory)
			hc.recorder = m.FakeRecorder

			gs := agonesv1.GameServer{ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "test"}, Spec: newSingleContainerSpec(),
				Status: agonesv1.GameServerStatus{State: test.state}}
			gs.ApplyDefaults()

			got := false
			updated := false
			m.KubeClient.AddReactor("list", "pods", func(action k8stesting.Action) (bool, runtime.Object, error) {
				list := &corev1.PodList{Items: []corev1.Pod{}}
				if test.podStatus != nil {
					pod, err := gs.Pod()
					assert.NoError(t, err)
					pod.Status = *test.podStatus
					list.Items = append(list.Items, *pod)
				}
				return true, list, nil
			})
			m.AgonesClient.AddReactor("list", "gameservers", func(action k8stesting.Action) (bool, runtime.Object, error) {
				got = true
				return true, &agonesv1.GameServerList{Items: []agonesv1.GameServer{gs}}, nil
			})
			m.AgonesClient.AddReactor("update", "gameservers", func(action k8stesting.Action) (bool, runtime.Object, error) {
				updated = true
				ua := action.(k8stesting.UpdateAction)
				gsObj := ua.GetObject().(*agonesv1.GameServer)
				assert.Equal(t, agonesv1.GameServerStateUnhealthy, gsObj.Status.State)
				return true, gsObj, nil
			})

			_, cancel := agtesting.StartInformers(m, hc.gameServerSynced, hc.podSynced)
			defer cancel()

			err := hc.syncGameServer("default/test")
			assert.Nil(t, err, err)
			assert.True(t, got, "GameServers Should be got!")

			assert.Equal(t, test.expected.updated, updated, "updated test")
		})
	}
}

func TestHealthControllerSyncGameServerUpdateFailed(t *testing.T) {
	t.Parallel()

	m := agtesting.NewMocks()
	hc := NewHealthController(healthcheck.NewHandler(), m.KubeClient, m.AgonesClient, m.KubeInformerFactory, m.AgonesInformerFactory)
	hc.recorder = m.FakeRecorder

	gs := agonesv1.GameServer{ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "test"}, Spec: newSingleContainerSpec(),
		Status: agonesv1.GameServerStatus{State: agonesv1.GameServerStateAllocated}}
	gs.ApplyDefaults()

	m.KubeClient.AddReactor("list", "pods", func(action k8stesting.Action) (bool, runtime.Object, error) {
		list := &corev1.PodList{Items: []corev1.Pod{}}
		return true, list, nil
	})
	m.AgonesClient.AddReactor("list", "gameservers", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, &agonesv1.GameServerList{Items: []agonesv1.GameServer{gs}}, nil
	})
	m.AgonesClient.AddReactor("update", "gameservers", func(action k8stesting.Action) (bool, runtime.Object, error) {
		ua := action.(k8stesting.UpdateAction)
		gsObj := ua.GetObject().(*agonesv1.GameServer)
		assert.Equal(t, agonesv1.GameServerStateUnhealthy, gsObj.Status.State)
		return true, gsObj, errors.New("update-err")
	})

	_, cancel := agtesting.StartInformers(m, hc.gameServerSynced, hc.podSynced)
	defer cancel()

	err := hc.syncGameServer("default/test")

	if assert.Error(t, err) {
		assert.Equal(t, "error updating GameServer test/default to unhealthy: update-err", err.Error())
	}
}

func TestHealthControllerRun(t *testing.T) {
	t.Parallel()

	m := agtesting.NewMocks()
	hc := NewHealthController(healthcheck.NewHandler(), m.KubeClient, m.AgonesClient, m.KubeInformerFactory, m.AgonesInformerFactory)
	hc.recorder = m.FakeRecorder

	gsWatch := watch.NewFake()
	m.AgonesClient.AddWatchReactor("gameservers", k8stesting.DefaultWatchReactor(gsWatch, nil))

	podWatch := watch.NewFake()
	m.KubeClient.AddWatchReactor("pods", k8stesting.DefaultWatchReactor(podWatch, nil))

	updated := make(chan bool)
	defer close(updated)
	m.AgonesClient.AddReactor("update", "gameservers", func(action k8stesting.Action) (bool, runtime.Object, error) {
		defer func() {
			updated <- true
		}()
		ua := action.(k8stesting.UpdateAction)
		gsObj := ua.GetObject().(*agonesv1.GameServer)
		assert.Equal(t, agonesv1.GameServerStateUnhealthy, gsObj.Status.State)
		return true, gsObj, nil
	})

	gs := &agonesv1.GameServer{ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "test"}, Spec: newSingleContainerSpec(),
		Status: agonesv1.GameServerStatus{State: agonesv1.GameServerStateReady}}
	gs.ApplyDefaults()
	pod, err := gs.Pod()
	require.NoError(t, err)

	stop, cancel := agtesting.StartInformers(m)
	defer cancel()

	gsWatch.Add(gs.DeepCopy())
	podWatch.Add(pod.DeepCopy())

	go hc.Run(stop) // nolint: errcheck
	err = wait.PollImmediate(time.Second, 10*time.Second, func() (bool, error) {
		return hc.workerqueue.RunCount() == 1, nil
	})
	assert.NoError(t, err)

	pod.Status.ContainerStatuses = []corev1.ContainerStatus{{Name: gs.Spec.Container, State: corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{}}}}
	// gate
	assert.True(t, hc.failedContainer(pod))
	assert.False(t, hc.unschedulableWithNoFreePorts(pod))

	podWatch.Modify(pod.DeepCopy())

	select {
	case <-updated:
	case <-time.After(10 * time.Second):
		assert.FailNow(t, "timeout on GameServer update")
	}

	agtesting.AssertEventContains(t, m.FakeRecorder.Events, string(agonesv1.GameServerStateUnhealthy))

	pod.Status.ContainerStatuses = nil
	pod.Status.Conditions = []corev1.PodCondition{
		{Type: corev1.PodScheduled, Reason: corev1.PodReasonUnschedulable,
			Message: "0/4 nodes are available: 4 node(s) didn't have free ports for the requestedpod ports."},
	}
	// gate
	assert.True(t, hc.unschedulableWithNoFreePorts(pod))
	assert.False(t, hc.failedContainer(pod))

	podWatch.Modify(pod.DeepCopy())

	select {
	case <-updated:
	case <-time.After(10 * time.Second):
		assert.FailNow(t, "timeout on GameServer update")
	}

	agtesting.AssertEventContains(t, m.FakeRecorder.Events, string(agonesv1.GameServerStateUnhealthy))

	podWatch.Delete(pod.DeepCopy())
	select {
	case <-updated:
	case <-time.After(10 * time.Second):
		assert.FailNow(t, "timeout on GameServer update")
	}

	agtesting.AssertEventContains(t, m.FakeRecorder.Events, string(agonesv1.GameServerStateUnhealthy))
}
