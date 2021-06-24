// Copyright 2021 Google LLC All Rights Reserved.
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

package gameserverallocations

import (
	"context"
	"testing"
	"time"

	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	allocationv1 "agones.dev/agones/pkg/apis/allocation/v1"
	"agones.dev/agones/pkg/gameservers"
	agtesting "agones.dev/agones/pkg/testing"
	"github.com/heptiolabs/healthcheck"
	"github.com/stretchr/testify/assert"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	k8stesting "k8s.io/client-go/testing"
)

func TestAllocatorApplyAllocationToGameServer(t *testing.T) {
	t.Parallel()
	m := agtesting.NewMocks()
	ctx := context.Background()

	m.AgonesClient.AddReactor("update", "gameservers", func(action k8stesting.Action) (bool, k8sruntime.Object, error) {
		ua := action.(k8stesting.UpdateAction)
		gs := ua.GetObject().(*agonesv1.GameServer)
		return true, gs, nil
	})

	allocator := NewAllocator(m.AgonesInformerFactory.Multicluster().V1().GameServerAllocationPolicies(),
		m.KubeInformerFactory.Core().V1().Secrets(),
		m.AgonesClient.AgonesV1(), m.KubeClient,
		NewAllocationCache(m.AgonesInformerFactory.Agones().V1().GameServers(), gameservers.NewPerNodeCounter(m.KubeInformerFactory, m.AgonesInformerFactory), healthcheck.NewHandler()),
		time.Second, 5*time.Second,
	)

	gs, err := allocator.applyAllocationToGameServer(ctx, allocationv1.MetaPatch{}, &agonesv1.GameServer{})
	assert.NoError(t, err)
	assert.Equal(t, agonesv1.GameServerStateAllocated, gs.Status.State)
	assert.NotNil(t, gs.ObjectMeta.Annotations["agones.dev/last-allocated"])

	gs, err = allocator.applyAllocationToGameServer(ctx, allocationv1.MetaPatch{Labels: map[string]string{"foo": "bar"}}, &agonesv1.GameServer{})
	assert.NoError(t, err)
	assert.Equal(t, agonesv1.GameServerStateAllocated, gs.Status.State)
	assert.Equal(t, "bar", gs.ObjectMeta.Labels["foo"])
	assert.NotNil(t, gs.ObjectMeta.Annotations["agones.dev/last-allocated"])

	gs, err = allocator.applyAllocationToGameServer(ctx,
		allocationv1.MetaPatch{Labels: map[string]string{"foo": "bar"}, Annotations: map[string]string{"bar": "foo"}},
		&agonesv1.GameServer{})
	assert.NoError(t, err)
	assert.Equal(t, agonesv1.GameServerStateAllocated, gs.Status.State)
	assert.Equal(t, "bar", gs.ObjectMeta.Labels["foo"])
	assert.Equal(t, "foo", gs.ObjectMeta.Annotations["bar"])
	assert.NotNil(t, gs.ObjectMeta.Annotations[lastAllocatedAnnotationKey])
}
