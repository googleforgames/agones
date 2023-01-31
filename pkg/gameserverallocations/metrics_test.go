// Copyright 2022 Google LLC All Rights Reserved.
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
	gameserverv1 "agones.dev/agones/pkg/client/listers/agones/v1"
	"agones.dev/agones/pkg/util/runtime"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

type mockGameServerLister struct {
	gameServerNamespaceLister mockGameServerNamespaceLister
	gameServersCalled         bool
}

type mockGameServerNamespaceLister struct {
	gameServer *agonesv1.GameServer
}

func (s *mockGameServerLister) List(selector labels.Selector) (ret []*agonesv1.GameServer, err error) {
	return ret, nil
}

func (s *mockGameServerLister) GameServers(namespace string) gameserverv1.GameServerNamespaceLister {
	s.gameServersCalled = true
	return s.gameServerNamespaceLister
}

func (s mockGameServerNamespaceLister) Get(name string) (*agonesv1.GameServer, error) {
	return s.gameServer, nil
}

func (s mockGameServerNamespaceLister) List(selector labels.Selector) (ret []*agonesv1.GameServer, err error) {
	return ret, nil
}

func TestSetResponse(t *testing.T) {
	subtests := []struct {
		name           string
		gameServer     *agonesv1.GameServer
		err            error
		allocation     *allocationv1.GameServerAllocation
		expectedState  allocationv1.GameServerAllocationState
		expectedCalled bool
	}{
		{
			name: "Try to get gs from local cluster for local allocation",
			gameServer: &agonesv1.GameServer{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{agonesv1.FleetNameLabel: "fleetName"},
				},
			},
			allocation: &allocationv1.GameServerAllocation{
				Status: allocationv1.GameServerAllocationStatus{
					State:          allocationv1.GameServerAllocationAllocated,
					GameServerName: "gameServerName",
					Source:         "local",
				},
			},
			expectedCalled: true,
		},
		{
			name: "Do not try to get gs from local cluster for remote allocation",
			gameServer: &agonesv1.GameServer{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{agonesv1.FleetNameLabel: "fleetName"},
				},
			},
			allocation: &allocationv1.GameServerAllocation{
				Status: allocationv1.GameServerAllocationStatus{
					State:          allocationv1.GameServerAllocationAllocated,
					GameServerName: "gameServerName",
					Source:         "33.188.237.156:443",
				},
			},
			expectedCalled: false,
		},
	}

	for _, subtest := range subtests {
		gsl := mockGameServerLister{
			gameServerNamespaceLister: mockGameServerNamespaceLister{
				gameServer: subtest.gameServer,
			},
		}

		metrics := metrics{
			ctx:              context.Background(),
			gameServerLister: &gsl,
			logger:           runtime.NewLoggerWithSource("metrics_test"),
			start:            time.Now(),
		}

		t.Run(subtest.name, func(t *testing.T) {
			metrics.setResponse(subtest.allocation)
			assert.Equal(t, subtest.expectedCalled, gsl.gameServersCalled)
		})
	}
}
