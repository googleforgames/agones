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

package metrics

import (
	"context"
	"testing"

	"agones.dev/agones/pkg/apis/stable/v1alpha1"
	agtesting "agones.dev/agones/pkg/testing"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/apimachinery/pkg/watch"
	k8stesting "k8s.io/client-go/testing"
	"k8s.io/client-go/tools/cache"
)

// newFakeController returns a controller, backed by the fake Clientset
func newFakeController() *fakeController {
	m := agtesting.NewMocks()
	c := NewController(m.KubeClient, m.AgonesClient, m.AgonesInformerFactory)
	gsWatch := watch.NewFake()
	faWatch := watch.NewFake()
	fasWatch := watch.NewFake()
	fleetWatch := watch.NewFake()

	m.AgonesClient.AddWatchReactor("gameservers", k8stesting.DefaultWatchReactor(gsWatch, nil))
	m.AgonesClient.AddWatchReactor("fleetallocations", k8stesting.DefaultWatchReactor(faWatch, nil))
	m.AgonesClient.AddWatchReactor("fleetautoscalers", k8stesting.DefaultWatchReactor(fasWatch, nil))
	m.AgonesClient.AddWatchReactor("fleets", k8stesting.DefaultWatchReactor(fleetWatch, nil))

	stop, cancel := agtesting.StartInformers(m, c.gameServerSynced, c.faSynced,
		c.fleetSynced, c.fasSynced)

	return &fakeController{
		Controller: c,
		Mocks:      m,
		gsWatch:    gsWatch,
		faWatch:    faWatch,
		fasWatch:   fasWatch,
		fleetWatch: fleetWatch,
		cancel:     cancel,
		stop:       stop,
	}
}

func (c *fakeController) close() {
	if c.cancel != nil {
		c.cancel()
	}
}

// hacky: unregistering views force view collections
// so to not have to wait for the reporting period to hit we can
// unregister and register again
func report() {
	unRegisterViews()
	registerViews()
}

func (c *fakeController) run(t *testing.T) {
	go func() {
		err := c.Controller.Run(1, c.stop)
		assert.Nil(t, err)
	}()
	c.sync()
}

func (c *fakeController) sync() {
	cache.WaitForCacheSync(c.stop, c.gameServerSynced, c.fleetSynced,
		c.fasSynced, c.faSynced)
}

type fakeController struct {
	*Controller
	agtesting.Mocks
	gsWatch    *watch.FakeWatcher
	faWatch    *watch.FakeWatcher
	fasWatch   *watch.FakeWatcher
	fleetWatch *watch.FakeWatcher
	stop       <-chan struct{}
	cancel     context.CancelFunc
}

func gameServer(fleetName string, state v1alpha1.GameServerState) *v1alpha1.GameServer {
	lbs := map[string]string{}
	if fleetName != "" {
		lbs[v1alpha1.FleetNameLabel] = fleetName
	}
	gs := &v1alpha1.GameServer{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rand.String(10),
			Namespace: "default",
			UID:       uuid.NewUUID(),
			Labels:    lbs,
		},
		Status: v1alpha1.GameServerStatus{
			State: state,
		},
	}
	return gs
}

func generateGsEvents(count int, state v1alpha1.GameServerState, fleetName string, fakew *watch.FakeWatcher) {
	for i := 0; i < count; i++ {
		gs := gameServer(fleetName, v1alpha1.GameServerState(""))
		fakew.Add(gs)
		gsUpdated := gs.DeepCopy()
		gsUpdated.Status.State = state
		fakew.Modify(gsUpdated)
		// make sure we count only one event
		fakew.Modify(gsUpdated)
	}
}

func fleetAllocation(fleetName string) *v1alpha1.FleetAllocation {
	return &v1alpha1.FleetAllocation{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rand.String(10),
			Namespace: "default",
			UID:       uuid.NewUUID(),
		},
		Spec: v1alpha1.FleetAllocationSpec{
			FleetName: fleetName,
		},
		Status: v1alpha1.FleetAllocationStatus{
			GameServer: gameServer(fleetName, v1alpha1.GameServerStateAllocated),
		},
	}
}

func fleet(fleetName string, total, allocated, ready, desired int32) *v1alpha1.Fleet {
	return &v1alpha1.Fleet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fleetName,
			Namespace: "default",
			UID:       uuid.NewUUID(),
		},
		Spec: v1alpha1.FleetSpec{
			Replicas: desired,
		},
		Status: v1alpha1.FleetStatus{
			AllocatedReplicas: allocated,
			ReadyReplicas:     ready,
			Replicas:          total,
		},
	}
}

func fleetAutoScaler(fleetName string, fasName string) *v1alpha1.FleetAutoscaler {
	return &v1alpha1.FleetAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fasName,
			Namespace: "default",
			UID:       uuid.NewUUID(),
		},
		Spec: v1alpha1.FleetAutoscalerSpec{
			FleetName: fleetName,
			Policy: v1alpha1.FleetAutoscalerPolicy{
				Type: v1alpha1.BufferPolicyType,
				Buffer: &v1alpha1.BufferPolicy{
					MaxReplicas: 30,
					MinReplicas: 10,
					BufferSize:  intstr.FromInt(11),
				},
			},
		},
		Status: v1alpha1.FleetAutoscalerStatus{
			AbleToScale:     true,
			ScalingLimited:  false,
			CurrentReplicas: 10,
			DesiredReplicas: 20,
		},
	}
}

var gsCountExpected = `# HELP agones_gameservers_count The number of gameservers
# TYPE agones_gameservers_count gauge
agones_gameservers_count{fleet_name="test-fleet",type="Ready"} 0
agones_gameservers_count{fleet_name="test-fleet",type="Shutdown"} 1
agones_gameservers_count{fleet_name="none",type="PortAllocation"} 2
`
var faCountExpected = `# HELP agones_fleet_allocations_count The number of fleet allocations
# TYPE agones_fleet_allocations_count gauge
agones_fleet_allocations_count{fleet_name="test-fleet"} 3
agones_fleet_allocations_count{fleet_name="test-fleet2"} 2
agones_fleet_allocations_count{fleet_name="deleted-fleet"} 0
`

var faTotalExpected = `# HELP agones_fleet_allocations_total The total of fleet allocations
# TYPE agones_fleet_allocations_total counter
agones_fleet_allocations_total{fleet_name="test2"} 2
agones_fleet_allocations_total{fleet_name="test"} 3
`

var gsTotalExpected = `# HELP agones_gameservers_total The total of gameservers
# TYPE agones_gameservers_total counter
agones_gameservers_total{fleet_name="test",type="Creating"} 16
agones_gameservers_total{fleet_name="test",type="Scheduled"} 15
agones_gameservers_total{fleet_name="test",type="Starting"} 10
agones_gameservers_total{fleet_name="test",type="Unhealthy"} 1
agones_gameservers_total{fleet_name="none",type="Creating"} 19
agones_gameservers_total{fleet_name="none",type="Scheduled"} 18
agones_gameservers_total{fleet_name="none",type="Starting"} 16
agones_gameservers_total{fleet_name="none",type="Unhealthy"} 1
`

var fleetReplicasCountExpected = `# HELP agones_fleets_replicas_count The number of replicas per fleet
# TYPE agones_fleets_replicas_count gauge
agones_fleets_replicas_count{name="fleet-deleted",type="allocated"} 0
agones_fleets_replicas_count{name="fleet-deleted",type="desired"} 0
agones_fleets_replicas_count{name="fleet-deleted",type="ready"} 0
agones_fleets_replicas_count{name="fleet-deleted",type="total"} 0
agones_fleets_replicas_count{name="fleet-test",type="allocated"} 2
agones_fleets_replicas_count{name="fleet-test",type="desired"} 5
agones_fleets_replicas_count{name="fleet-test",type="ready"} 1
agones_fleets_replicas_count{name="fleet-test",type="total"} 8
`

var fasStateExpected = `# HELP agones_fleet_autoscalers_able_to_scale The fleet autoscaler can access the fleet to scale
# TYPE agones_fleet_autoscalers_able_to_scale gauge
agones_fleet_autoscalers_able_to_scale{fleet_name="first-fleet",name="name-switch"} 0
agones_fleet_autoscalers_able_to_scale{fleet_name="second-fleet",name="name-switch"} 1
agones_fleet_autoscalers_able_to_scale{fleet_name="deleted-fleet",name="deleted"} 0
# HELP agones_fleet_autoscalers_buffer_limits The limits of buffer based fleet autoscalers
# TYPE agones_fleet_autoscalers_buffer_limits gauge
agones_fleet_autoscalers_buffer_limits{fleet_name="first-fleet",name="name-switch",type="max"} 50
agones_fleet_autoscalers_buffer_limits{fleet_name="first-fleet",name="name-switch",type="min"} 10
agones_fleet_autoscalers_buffer_limits{fleet_name="second-fleet",name="name-switch",type="max"} 50
agones_fleet_autoscalers_buffer_limits{fleet_name="second-fleet",name="name-switch",type="min"} 10
agones_fleet_autoscalers_buffer_limits{fleet_name="deleted-fleet",name="deleted",type="max"} 150
agones_fleet_autoscalers_buffer_limits{fleet_name="deleted-fleet",name="deleted",type="min"} 15
# HELP agones_fleet_autoscalers_buffer_size The buffer size of fleet autoscalers
# TYPE agones_fleet_autoscalers_buffer_size gauge
agones_fleet_autoscalers_buffer_size{fleet_name="first-fleet",name="name-switch",type="count"} 10
agones_fleet_autoscalers_buffer_size{fleet_name="second-fleet",name="name-switch",type="count"} 10
agones_fleet_autoscalers_buffer_size{fleet_name="deleted-fleet",name="deleted",type="percentage"} 50
# HELP agones_fleet_autoscalers_current_replicas_count The current replicas count as seen by autoscalers
# TYPE agones_fleet_autoscalers_current_replicas_count gauge
agones_fleet_autoscalers_current_replicas_count{fleet_name="first-fleet",name="name-switch"} 0
agones_fleet_autoscalers_current_replicas_count{fleet_name="second-fleet",name="name-switch"} 20
agones_fleet_autoscalers_current_replicas_count{fleet_name="deleted-fleet",name="deleted"} 0
# HELP agones_fleet_autoscalers_desired_replicas_count The desired replicas count as seen by autoscalers
# TYPE agones_fleet_autoscalers_desired_replicas_count gauge
agones_fleet_autoscalers_desired_replicas_count{fleet_name="first-fleet",name="name-switch"} 0
agones_fleet_autoscalers_desired_replicas_count{fleet_name="second-fleet",name="name-switch"} 10
agones_fleet_autoscalers_desired_replicas_count{fleet_name="deleted-fleet",name="deleted"} 0
# HELP agones_fleet_autoscalers_limited The fleet autoscaler is capped
# TYPE agones_fleet_autoscalers_limited gauge
agones_fleet_autoscalers_limited{fleet_name="first-fleet",name="name-switch"} 0
agones_fleet_autoscalers_limited{fleet_name="second-fleet",name="name-switch"} 1
agones_fleet_autoscalers_limited{fleet_name="deleted-fleet",name="deleted"} 0
`
