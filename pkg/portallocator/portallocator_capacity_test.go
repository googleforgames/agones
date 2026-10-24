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

package portallocator

import (
	"fmt"
	"testing"

	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	agtesting "agones.dev/agones/pkg/testing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPortRangeAllocatorRejectsOversizedRequest(t *testing.T) {
	for _, policy := range []agonesv1.PortPolicy{agonesv1.Dynamic, agonesv1.Passthrough} {
		t.Run(string(policy), func(t *testing.T) {
			m := agtesting.NewMocks()
			pa := newRangeAllocator(agonesv1.DefaultPortRange, 10, 12, m.KubeInformerFactory, m.AgonesInformerFactory)
			pa.portAllocations = []portAllocation{pa.newPortAllocation()}
			gs := dynamicGameServerFixture()
			gs.Spec.Ports = nil
			for i := 0; i < 4; i++ {
				gs.Spec.Ports = append(gs.Spec.Ports, agonesv1.GameServerPort{
					Name: fmt.Sprintf("port-%d", i), Range: agonesv1.DefaultPortRange,
					PortPolicy: policy, Protocol: agonesv1.ProtocolTCPUDP,
				})
			}
			original := gs.DeepCopy()
			allocated, err := pa.Allocate(gs)
			require.ErrorContains(t, err, "requests 4 ports")
			assert.Nil(t, allocated)
			assert.Equal(t, original, gs)
			assert.Len(t, pa.portAllocations, 1)
			assert.Zero(t, countTotalAllocatedPorts(pa))
			assert.Empty(t, pa.gameServerRegistry)

			gs.Spec.Ports = gs.Spec.Ports[:3]
			allocated, err = pa.Allocate(gs)
			require.NoError(t, err)
			require.Len(t, allocated.Spec.Ports, 6)
			assert.Equal(t, 3, countTotalAllocatedPorts(pa))
			pa.DeAllocate(allocated)
			assert.Zero(t, countTotalAllocatedPorts(pa))
		})
	}
}

func TestPortAllocatorReleasesEarlierRangesOnFailure(t *testing.T) {
	m := agtesting.NewMocks()
	first := newRangeAllocator(agonesv1.DefaultPortRange, 10, 12, m.KubeInformerFactory, m.AgonesInformerFactory)
	second := newRangeAllocator("other", 20, 22, m.KubeInformerFactory, m.AgonesInformerFactory)
	pa := &portAllocator{allocators: []*portRangeAllocator{first, second}}
	gs := dynamicGameServerFixture()
	for i := 0; i < 4; i++ {
		gs.Spec.Ports = append(gs.Spec.Ports, agonesv1.GameServerPort{
			Name: fmt.Sprintf("other-%d", i), Range: "other", PortPolicy: agonesv1.Dynamic,
		})
	}
	allocated, err := pa.Allocate(gs)
	require.ErrorContains(t, err, "range other with capacity 3")
	assert.Nil(t, allocated)
	assert.Zero(t, countTotalAllocatedPorts(first))
	assert.Empty(t, first.gameServerRegistry)
	assert.Empty(t, second.gameServerRegistry)
	assert.Empty(t, second.portAllocations)
}
