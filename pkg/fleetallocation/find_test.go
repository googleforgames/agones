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

package fleetallocation

import (
	"testing"

	"agones.dev/agones/pkg/apis/stable/v1alpha1"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestFindPackedReadyGameServer(t *testing.T) {
	t.Parallel()

	t.Run("test one", func(t *testing.T) {
		n := metav1.Now()

		gsList := []*v1alpha1.GameServer{
			{ObjectMeta: metav1.ObjectMeta{Name: "gs6", DeletionTimestamp: &n}, Status: v1alpha1.GameServerStatus{NodeName: "node1", State: v1alpha1.Ready}},
			{ObjectMeta: metav1.ObjectMeta{Name: "gs1"}, Status: v1alpha1.GameServerStatus{NodeName: "node1", State: v1alpha1.Ready}},
			{ObjectMeta: metav1.ObjectMeta{Name: "gs2"}, Status: v1alpha1.GameServerStatus{NodeName: "node2", State: v1alpha1.Ready}},
			{ObjectMeta: metav1.ObjectMeta{Name: "gs3"}, Status: v1alpha1.GameServerStatus{NodeName: "node1", State: v1alpha1.Allocated}},
			{ObjectMeta: metav1.ObjectMeta{Name: "gs4"}, Status: v1alpha1.GameServerStatus{NodeName: "node1", State: v1alpha1.Allocated}},
			{ObjectMeta: metav1.ObjectMeta{Name: "gs5"}, Status: v1alpha1.GameServerStatus{NodeName: "node1", State: v1alpha1.Error}},
		}

		gs := findReadyGameServerForAllocation(gsList, packedComparator)
		assert.Equal(t, "node1", gs.Status.NodeName)
		assert.Equal(t, v1alpha1.Ready, gs.Status.State)
		// mock that the first game server is allocated
		gsList[1].Status.State = v1alpha1.Allocated
		gs = findReadyGameServerForAllocation(gsList, packedComparator)
		assert.Equal(t, "node2", gs.Status.NodeName)
		assert.Equal(t, v1alpha1.Ready, gs.Status.State)
		gsList[2].Status.State = v1alpha1.Allocated
		gs = findReadyGameServerForAllocation(gsList, packedComparator)
		assert.Nil(t, gs)
	})

	t.Run("allocation trap", func(t *testing.T) {
		gsList := []*v1alpha1.GameServer{
			{ObjectMeta: metav1.ObjectMeta{Name: "gs1"}, Status: v1alpha1.GameServerStatus{NodeName: "node1", State: v1alpha1.Allocated}},
			{ObjectMeta: metav1.ObjectMeta{Name: "gs2"}, Status: v1alpha1.GameServerStatus{NodeName: "node1", State: v1alpha1.Allocated}},
			{ObjectMeta: metav1.ObjectMeta{Name: "gs3"}, Status: v1alpha1.GameServerStatus{NodeName: "node1", State: v1alpha1.Allocated}},
			{ObjectMeta: metav1.ObjectMeta{Name: "gs4"}, Status: v1alpha1.GameServerStatus{NodeName: "node1", State: v1alpha1.Allocated}},
			{ObjectMeta: metav1.ObjectMeta{Name: "gs5"}, Status: v1alpha1.GameServerStatus{NodeName: "node2", State: v1alpha1.Ready}},
			{ObjectMeta: metav1.ObjectMeta{Name: "gs6"}, Status: v1alpha1.GameServerStatus{NodeName: "node2", State: v1alpha1.Ready}},
			{ObjectMeta: metav1.ObjectMeta{Name: "gs7"}, Status: v1alpha1.GameServerStatus{NodeName: "node2", State: v1alpha1.Ready}},
			{ObjectMeta: metav1.ObjectMeta{Name: "gs8"}, Status: v1alpha1.GameServerStatus{NodeName: "node2", State: v1alpha1.Ready}},
		}

		gs := findReadyGameServerForAllocation(gsList, packedComparator)
		assert.Equal(t, "node2", gs.Status.NodeName)
		assert.Equal(t, v1alpha1.Ready, gs.Status.State)
	})
}

func TestFindDistributedReadyGameServer(t *testing.T) {
	t.Parallel()

	n := metav1.Now()
	gsList := []*v1alpha1.GameServer{
		{ObjectMeta: metav1.ObjectMeta{Name: "gs6", DeletionTimestamp: &n}, Status: v1alpha1.GameServerStatus{NodeName: "node3", State: v1alpha1.Ready}},
		{ObjectMeta: metav1.ObjectMeta{Name: "gs1"}, Status: v1alpha1.GameServerStatus{NodeName: "node1", State: v1alpha1.Ready}},
		{ObjectMeta: metav1.ObjectMeta{Name: "gs2"}, Status: v1alpha1.GameServerStatus{NodeName: "node1", State: v1alpha1.Ready}},
		{ObjectMeta: metav1.ObjectMeta{Name: "gs3"}, Status: v1alpha1.GameServerStatus{NodeName: "node1", State: v1alpha1.Ready}},
		{ObjectMeta: metav1.ObjectMeta{Name: "gs4"}, Status: v1alpha1.GameServerStatus{NodeName: "node1", State: v1alpha1.Error}},
		{ObjectMeta: metav1.ObjectMeta{Name: "gs5"}, Status: v1alpha1.GameServerStatus{NodeName: "node2", State: v1alpha1.Ready}},
		{ObjectMeta: metav1.ObjectMeta{Name: "gs6"}, Status: v1alpha1.GameServerStatus{NodeName: "node2", State: v1alpha1.Ready}},
		{ObjectMeta: metav1.ObjectMeta{Name: "gs7"}, Status: v1alpha1.GameServerStatus{NodeName: "node3", State: v1alpha1.Ready}},
	}

	gs := findReadyGameServerForAllocation(gsList, distributedComparator)
	assert.Equal(t, "node3", gs.Status.NodeName)
	assert.Equal(t, v1alpha1.Ready, gs.Status.State)

	gsList[7].Status.State = v1alpha1.Allocated

	gs = findReadyGameServerForAllocation(gsList, distributedComparator)
	assert.Equal(t, "node2", gs.Status.NodeName)
	assert.Equal(t, v1alpha1.Ready, gs.Status.State)

	gsList[5].Status.State = v1alpha1.Allocated
	assert.Equal(t, "node2", gsList[5].Status.NodeName)

	gs = findReadyGameServerForAllocation(gsList, distributedComparator)
	assert.Equal(t, "node1", gs.Status.NodeName)
	assert.Equal(t, v1alpha1.Ready, gs.Status.State)

	gsList[1].Status.State = v1alpha1.Allocated

	gs = findReadyGameServerForAllocation(gsList, distributedComparator)
	assert.Equal(t, "node2", gs.Status.NodeName)
	assert.Equal(t, v1alpha1.Ready, gs.Status.State)

	gsList[6].Status.State = v1alpha1.Allocated

	gs = findReadyGameServerForAllocation(gsList, distributedComparator)
	assert.Equal(t, "node1", gs.Status.NodeName)
	assert.Equal(t, v1alpha1.Ready, gs.Status.State)

	gsList[2].Status.State = v1alpha1.Allocated

	gs = findReadyGameServerForAllocation(gsList, distributedComparator)
	assert.Equal(t, "node1", gs.Status.NodeName)
	assert.Equal(t, v1alpha1.Ready, gs.Status.State)

	gsList[3].Status.State = v1alpha1.Allocated

	gs = findReadyGameServerForAllocation(gsList, distributedComparator)
	assert.Nil(t, gs)
}
