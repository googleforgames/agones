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
	"context"
	"testing"
	"time"

	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	"agones.dev/agones/pkg/cloudproduct"
	agtesting "agones.dev/agones/pkg/testing"
	"github.com/heptiolabs/healthcheck"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	k8stesting "k8s.io/client-go/testing"
	"k8s.io/client-go/tools/cache"
)

func TestMigrationControllerIsRunningGameServer(t *testing.T) {
	fixtures := map[string]struct {
		setup    func(*corev1.Pod) *corev1.Pod
		expected bool
	}{
		"not a gameserver pod": {
			setup: func(pod *corev1.Pod) *corev1.Pod {
				return &corev1.Pod{}
			},
			expected: false,
		},
		"game server pod": {
			setup: func(pod *corev1.Pod) *corev1.Pod {
				return pod
			},
			expected: false,
		},
		"game server pod that has a nodename": {
			setup: func(pod *corev1.Pod) *corev1.Pod {
				pod.Spec.NodeName = "x"
				return pod
			},
			expected: true,
		},
		"game server pod that has a node, but is being deleted": {
			setup: func(pod *corev1.Pod) *corev1.Pod {
				pod.Spec.NodeName = "z"
				now := metav1.Now()
				pod.ObjectMeta.DeletionTimestamp = &now
				return pod
			},
			expected: false,
		},
	}

	for k, v := range fixtures {
		t.Run(k, func(t *testing.T) {
			gs := &agonesv1.GameServer{ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "default"},
				Spec: newSingleContainerSpec()}
			gs.ApplyDefaults()

			gsPod, err := gs.Pod()
			require.NoError(t, err)

			pod := v.setup(gsPod)
			assert.Equal(t, v.expected, isActiveGameServerWithNode(pod))
		})
	}
}

func TestMigrationControllerSyncGameServer(t *testing.T) {
	t.Parallel()

	ipChangeFixture := "99.99.99.99"
	nodeNameChangeFixture := "nodeChange"

	type expected struct {
		updated     bool
		updateTests func(t *testing.T, gs *agonesv1.GameServer)
		postTests   func(t *testing.T, mocks agtesting.Mocks)
	}
	fixtures := map[string]struct {
		setup    func(*agonesv1.GameServer, *corev1.Pod, *corev1.Node) (*agonesv1.GameServer, *corev1.Pod, *corev1.Node)
		expected expected
	}{
		"no change, gameserver nodeName not yet set": {
			setup: func(gs *agonesv1.GameServer, pod *corev1.Pod, node *corev1.Node) (*agonesv1.GameServer, *corev1.Pod, *corev1.Node) {
				return gs, pod, node
			},
			expected: expected{
				updated:     false,
				updateTests: func(t *testing.T, gs *agonesv1.GameServer) {},
				postTests: func(t *testing.T, m agtesting.Mocks) {
					agtesting.AssertNoEvent(t, m.FakeRecorder.Events)
				},
			},
		},
		"no change, with same address": {
			setup: func(gs *agonesv1.GameServer, pod *corev1.Pod, node *corev1.Node) (*agonesv1.GameServer, *corev1.Pod, *corev1.Node) {
				gs.Status.NodeName = nodeFixtureName
				gs.Status.Address = ipFixture
				return gs, pod, node
			},
			expected: expected{
				updated:     false,
				updateTests: func(t *testing.T, gs *agonesv1.GameServer) {},
				postTests: func(t *testing.T, m agtesting.Mocks) {
					agtesting.AssertNoEvent(t, m.FakeRecorder.Events)
				},
			},
		},
		"change before ready, ip only change": {
			setup: func(gs *agonesv1.GameServer, pod *corev1.Pod, node *corev1.Node) (*agonesv1.GameServer, *corev1.Pod, *corev1.Node) {
				gs.Status.NodeName = nodeFixtureName
				gs.Status.Address = ipFixture
				gs.Status.State = agonesv1.GameServerStateScheduled
				node.Status.Addresses[0].Address = ipChangeFixture
				return gs, pod, node
			},
			expected: expected{
				updated: true,
				updateTests: func(t *testing.T, gs *agonesv1.GameServer) {
					assert.Equal(t, ipChangeFixture, gs.Status.Address)
				},
				postTests: func(t *testing.T, m agtesting.Mocks) {
					agtesting.AssertEventContains(t, m.FakeRecorder.Events, "Warning Scheduled Address updated due to Node migration")
				},
			},
		},
		"change before ready, full node change": {
			setup: func(gs *agonesv1.GameServer, pod *corev1.Pod, node *corev1.Node) (*agonesv1.GameServer, *corev1.Pod, *corev1.Node) {
				gs.Status.NodeName = nodeFixtureName
				gs.Status.Address = ipFixture
				gs.Status.State = agonesv1.GameServerStateScheduled

				// full node name change
				pod.Spec.NodeName = nodeNameChangeFixture
				node.ObjectMeta.Name = nodeNameChangeFixture
				node.Status.Addresses[0].Address = ipChangeFixture
				return gs, pod, node
			},
			expected: expected{
				updated: true,
				updateTests: func(t *testing.T, gs *agonesv1.GameServer) {
					assert.Equal(t, ipChangeFixture, gs.Status.Address)
					assert.Equal(t, nodeNameChangeFixture, gs.Status.NodeName)
				},
				postTests: func(t *testing.T, m agtesting.Mocks) {
					agtesting.AssertEventContains(t, m.FakeRecorder.Events, "Warning Scheduled Address updated due to Node migration")
				},
			},
		},
		"change after ready": {
			setup: func(gs *agonesv1.GameServer, pod *corev1.Pod, node *corev1.Node) (*agonesv1.GameServer, *corev1.Pod, *corev1.Node) {
				gs.Status.NodeName = nodeFixtureName
				gs.Status.Address = ipFixture
				gs.Status.State = agonesv1.GameServerStateAllocated

				// full node name change
				pod.Spec.NodeName = nodeNameChangeFixture
				node.ObjectMeta.Name = nodeNameChangeFixture
				node.Status.Addresses[0].Address = ipChangeFixture

				return gs, pod, node
			},
			expected: expected{
				updated: true,
				updateTests: func(t *testing.T, gs *agonesv1.GameServer) {
					assert.Equal(t, agonesv1.GameServerStateUnhealthy, gs.Status.State)
				},
				postTests: func(t *testing.T, m agtesting.Mocks) {
					agtesting.AssertEventContains(t, m.FakeRecorder.Events, "Warning Unhealthy Node migration occurred")
				},
			},
		},
	}

	for k, v := range fixtures {
		t.Run(k, func(t *testing.T) {
			m := agtesting.NewMocks()
			c := NewMigrationController(healthcheck.NewHandler(), m.KubeClient, m.AgonesClient, m.KubeInformerFactory, m.AgonesInformerFactory,
				cloudproduct.MustNewGeneric(context.Background()))
			c.recorder = m.FakeRecorder

			gs := &agonesv1.GameServer{ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "default"},
				Spec: newSingleContainerSpec(), Status: agonesv1.GameServerStatus{}}
			gs.ApplyDefaults()

			pod, err := gs.Pod()
			require.NoError(t, err)
			pod.Spec.NodeName = nodeFixtureName

			node := &corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: nodeFixtureName},
				Status: corev1.NodeStatus{Addresses: []corev1.NodeAddress{{Address: ipFixture, Type: corev1.NodeExternalIP}}}}

			gs, pod, node = v.setup(gs, pod, node)

			// populate
			m.AgonesClient.AddReactor("list", "gameservers", func(action k8stesting.Action) (bool, runtime.Object, error) {
				return true, &agonesv1.GameServerList{Items: []agonesv1.GameServer{*gs}}, nil
			})
			m.KubeClient.AddReactor("list", "nodes", func(action k8stesting.Action) (bool, runtime.Object, error) {
				return true,
					&corev1.NodeList{Items: []corev1.Node{*node}}, nil
			})
			m.KubeClient.AddReactor("list", "pods", func(action k8stesting.Action) (bool, runtime.Object, error) {
				return true, &corev1.PodList{Items: []corev1.Pod{*pod}}, nil
			})

			// check values
			updated := false
			m.AgonesClient.AddReactor("update", "gameservers", func(action k8stesting.Action) (bool, runtime.Object, error) {
				updated = true
				ua := action.(k8stesting.UpdateAction)
				gs := ua.GetObject().(*agonesv1.GameServer)
				v.expected.updateTests(t, gs)
				return true, gs, nil
			})

			ctx, cancel := agtesting.StartInformers(m, c.nodeSynced, c.gameServerSynced, c.podSynced)
			defer cancel()

			err = c.syncGameServer(ctx, "default/test")
			assert.NoError(t, err)
			assert.Equal(t, v.expected.updated, updated)
			v.expected.postTests(t, m)
		})
	}
}

func TestMigrationControllerRun(t *testing.T) {
	m := agtesting.NewMocks()
	c := NewMigrationController(healthcheck.NewHandler(), m.KubeClient, m.AgonesClient, m.KubeInformerFactory, m.AgonesInformerFactory,
		cloudproduct.MustNewGeneric(context.Background()))
	gs := &agonesv1.GameServer{ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "default"},
		Spec: newSingleContainerSpec(), Status: agonesv1.GameServerStatus{}}
	gs.ApplyDefaults()
	gsPod, err := gs.Pod()
	require.NoError(t, err)
	gsPod.Spec.NodeName = nodeFixtureName

	received := make(chan string)
	h := func(_ context.Context, name string) error {
		assert.Equal(t, "default/test", name)
		received <- name
		return nil
	}

	podWatch := watch.NewFake()
	m.KubeClient.AddWatchReactor("pods", k8stesting.DefaultWatchReactor(podWatch, nil))

	c.workerqueue.SyncHandler = h

	ctx, cancel := agtesting.StartInformers(m, c.gameServerSynced)
	defer cancel()

	go func() {
		err := c.Run(ctx)
		assert.Nil(t, err, "Run should not error")
	}()

	noChange := func() {
		assert.True(t, cache.WaitForCacheSync(ctx.Done(), c.podSynced))
		select {
		case <-received:
			assert.Fail(t, "should not be updated")
		default:
		}
	}

	result := func() {
		select {
		case res := <-received:
			assert.Equal(t, "default/test", res)
		case <-time.After(2 * time.Second):
			assert.Fail(t, "did not receive queue")
		}
	}

	// initial pod
	podWatch.Add(gsPod.DeepCopy())
	result()

	gsPod.Spec.NodeName += "+changed"
	podWatch.Modify(gsPod.DeepCopy())
	result()

	// deleted pod
	now := metav1.Now()
	gsPod.ObjectMeta.DeletionTimestamp = &now
	podWatch.Modify(gsPod.DeepCopy())
	noChange()

	// non gameserver pod
	pod := corev1.Pod{ObjectMeta: metav1.ObjectMeta{
		Name:      "test2",
		Namespace: "default",
	}}
	pod.Spec.NodeName = nodeFixtureName
	podWatch.Add(pod.DeepCopy())
	noChange()

	pod.Spec.NodeName += "+changed"
	podWatch.Modify(pod.DeepCopy())
	noChange()
}
