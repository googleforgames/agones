// Copyright Contributors to Agones a Series of LF Projects, LLC.
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
	"fmt"
	"testing"

	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8stesting "k8s.io/client-go/testing"
)

func TestControllerPortAllocationRejectsOversizedRequest(t *testing.T) {
	c, mocks := newFakeController()
	gs := &agonesv1.GameServer{
		ObjectMeta: metav1.ObjectMeta{Name: "too-many-ports", Namespace: "default"},
		Status:     agonesv1.GameServerStatus{State: agonesv1.GameServerStatePortAllocation},
	}
	// The fake controller's default range is 10 through 20 inclusive.
	for i := 0; i < 12; i++ {
		gs.Spec.Ports = append(gs.Spec.Ports, agonesv1.GameServerPort{
			Name: fmt.Sprintf("port-%d", i), Range: agonesv1.DefaultPortRange, PortPolicy: agonesv1.Dynamic,
		})
	}
	updates := 0
	mocks.AgonesClient.AddReactor("update", "gameservers", func(action k8stesting.Action) (bool, runtime.Object, error) {
		updates++
		updated := action.(k8stesting.UpdateAction).GetObject().(*agonesv1.GameServer)
		assert.Equal(t, agonesv1.GameServerStateError, updated.Status.State)
		for _, port := range updated.Spec.Ports {
			assert.Zero(t, port.HostPort)
		}
		return true, updated, nil
	})
	result, err := c.syncGameServerPortAllocationState(context.Background(), gs)
	require.NoError(t, err)
	assert.Equal(t, agonesv1.GameServerStateError, result.Status.State)
	assert.Equal(t, 1, updates)
	assert.Equal(t, agonesv1.GameServerStatePortAllocation, gs.Status.State)
}
