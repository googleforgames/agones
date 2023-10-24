// Copyright 2023 Google LLC All Rights Reserved.
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

package allocator

import (
	"context"
	"testing"
	"time"

	pb "agones.dev/agones/pkg/allocation/go"
	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	helper "agones.dev/agones/test/e2e/allochelper"
	e2e "agones.dev/agones/test/e2e/framework"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	retryInterval = 5 * time.Second
	retryTimeout  = 45 * time.Second
)

func TestAllocatorAfterDeleteReplica(t *testing.T) {
	ctx := context.Background()

	var list *v1.PodList

	dep, err := framework.KubeClient.AppsV1().Deployments("agones-system").Get(ctx, "agones-allocator", metav1.GetOptions{})
	require.NoError(t, err, "Failed to get replicas")
	replicaCnt := int(*(dep.Spec.Replicas))
	logrus.Infof("Replica count config is %d", replicaCnt)

	// poll and wait until all allocator pods are running
	_ = wait.PollUntilContextTimeout(context.Background(), retryInterval, retryTimeout, true, func(ctx context.Context) (done bool, err error) {
		list, err = helper.GetAgonesAllocatorPods(ctx, framework)
		if err != nil {
			return true, err
		}

		if len(list.Items) != replicaCnt {
			return false, nil
		}

		for _, allocpod := range list.Items {
			podstatus := string(allocpod.Status.Phase)
			logrus.Infof("Allocator Pod %s, has status of %s", allocpod.ObjectMeta.Name, podstatus)
			if podstatus != "Running" {
				return false, nil
			}
		}

		return true, nil
	})

	// create fleet
	flt, err := helper.CreateFleet(ctx, framework.Namespace, framework)
	if !assert.Nil(t, err) {
		return
	}
	framework.AssertFleetCondition(t, flt, e2e.FleetReadyCount(flt.Spec.Replicas))

	var response *pb.AllocationResponse
	request := &pb.AllocationRequest{
		Namespace:                    framework.Namespace,
		PreferredGameServerSelectors: []*pb.GameServerSelector{{MatchLabels: map[string]string{agonesv1.FleetNameLabel: flt.ObjectMeta.Name}}},
		Scheduling:                   pb.AllocationRequest_Packed,
		Metadata:                     &pb.MetaPatch{Labels: map[string]string{"gslabel": "allocatedbytest"}},
	}

	// delete all of the allocators except 1
	for _, pod := range list.Items[1:] {
		err = helper.DeleteAgonesAllocatorPod(ctx, pod.ObjectMeta.Name, framework)
		require.NoError(t, err, "Could not delete allocator pod")
	}

	grpcClient, err := helper.GetAllocatorClient(ctx, t, framework)
	require.NoError(t, err, "Could not initialize rpc client")

	// Wait and keep making calls till we know the draining time has passed
	_ = wait.PollUntilContextTimeout(context.Background(), retryInterval, retryTimeout, true, func(ctx context.Context) (bool, error) {
		response, err = grpcClient.Allocate(context.Background(), request)
		logrus.Info(response)
		helper.ValidateAllocatorResponse(t, response)
		require.NoError(t, err, "Failed grpc allocation request")
		err = helper.DeleteAgonesPod(ctx, response.GameServerName, framework.Namespace, framework)
		require.NoError(t, err, "Failed to delete game server pod %s", response.GameServerName)
		return false, nil
	})
}
