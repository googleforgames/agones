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

package e2e

import (
	"testing"

	"agones.dev/agones/pkg/apis/stable/v1alpha1"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestCreateFleetAndAllocate(t *testing.T) {
	t.Parallel()

	flt, err := framework.AgonesClient.StableV1alpha1().Fleets(defaultNs).Create(defaultFleet())
	assert.Nil(t, err)

	err = framework.WaitForFleetReady(flt)
	assert.Nil(t, err, "fleet not ready")

	fa := &v1alpha1.FleetAllocation{
		ObjectMeta: metav1.ObjectMeta{GenerateName: "allocatioon-", Namespace: defaultNs},
		Spec: v1alpha1.FleetAllocationSpec{
			FleetName: flt.ObjectMeta.Name,
		},
	}

	fa, err = framework.AgonesClient.StableV1alpha1().FleetAllocations(defaultNs).Create(fa)
	assert.Nil(t, err)
	assert.Equal(t, v1alpha1.Allocated, fa.Status.GameServer.Status.State)
}

// defaultFleet returns a default fleet configuration
func defaultFleet() *v1alpha1.Fleet {
	gs := defaultGameServer()

	return &v1alpha1.Fleet{
		ObjectMeta: metav1.ObjectMeta{GenerateName: "simple-fleet-", Namespace: defaultNs},
		Spec: v1alpha1.FleetSpec{
			Replicas: 3,
			Template: v1alpha1.GameServerTemplateSpec{
				Spec: gs.Spec,
			},
		},
	}
}
