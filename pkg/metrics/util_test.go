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

package metrics

import (
	"context"
	"reflect"
	"testing"

	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	autoscalingv1 "agones.dev/agones/pkg/apis/autoscaling/v1"
	agtesting "agones.dev/agones/pkg/testing"
	"contrib.go.opencensus.io/exporter/stackdriver"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
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
	c := NewController(m.KubeClient, m.AgonesClient, m.KubeInformerFactory, m.AgonesInformerFactory)
	gsWatch := watch.NewFake()
	fasWatch := watch.NewFake()
	fleetWatch := watch.NewFake()
	nodeWatch := watch.NewFake()

	m.AgonesClient.AddWatchReactor("gameservers", k8stesting.DefaultWatchReactor(gsWatch, nil))
	m.AgonesClient.AddWatchReactor("fleetautoscalers", k8stesting.DefaultWatchReactor(fasWatch, nil))
	m.AgonesClient.AddWatchReactor("fleets", k8stesting.DefaultWatchReactor(fleetWatch, nil))
	m.KubeClient.AddWatchReactor("nodes", k8stesting.DefaultWatchReactor(nodeWatch, nil))

	stop, cancel := agtesting.StartInformers(m, c.gameServerSynced, c.fleetSynced, c.fasSynced, c.nodeSynced)

	return &fakeController{
		Controller: c,
		Mocks:      m,
		gsWatch:    gsWatch,
		fasWatch:   fasWatch,
		fleetWatch: fleetWatch,
		nodeWatch:  nodeWatch,
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
	cache.WaitForCacheSync(c.stop, c.gameServerSynced, c.fleetSynced, c.fasSynced)
}

type fakeController struct {
	*Controller
	agtesting.Mocks
	gsWatch    *watch.FakeWatcher
	fasWatch   *watch.FakeWatcher
	fleetWatch *watch.FakeWatcher
	nodeWatch  *watch.FakeWatcher
	stop       <-chan struct{}
	cancel     context.CancelFunc
}

func nodeWithName(name string) *v1.Node {
	return &v1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			UID:  uuid.NewUUID(),
		},
	}
}

func gameServerWithNode(nodeName string) *agonesv1.GameServer {
	gs := gameServerWithFleetAndState("fleet", agonesv1.GameServerStateReady)
	gs.Status.NodeName = nodeName
	return gs
}

func gameServerWithFleetAndState(fleetName string, state agonesv1.GameServerState) *agonesv1.GameServer {
	lbs := map[string]string{}
	if fleetName != "" {
		lbs[agonesv1.FleetNameLabel] = fleetName
	}
	gs := &agonesv1.GameServer{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rand.String(10),
			Namespace: "default",
			UID:       uuid.NewUUID(),
			Labels:    lbs,
		},
		Status: agonesv1.GameServerStatus{
			State: state,
		},
	}
	return gs
}

func generateGsEvents(count int, state agonesv1.GameServerState, fleetName string, fakew *watch.FakeWatcher) {
	for i := 0; i < count; i++ {
		gs := gameServerWithFleetAndState(fleetName, agonesv1.GameServerState(""))
		fakew.Add(gs)
		gsUpdated := gs.DeepCopy()
		gsUpdated.Status.State = state
		fakew.Modify(gsUpdated)
		// make sure we count only one event
		fakew.Modify(gsUpdated)
	}
}

func fleet(fleetName string, total, allocated, ready, desired int32) *agonesv1.Fleet {
	return &agonesv1.Fleet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fleetName,
			Namespace: "default",
			UID:       uuid.NewUUID(),
		},
		Spec: agonesv1.FleetSpec{
			Replicas: desired,
		},
		Status: agonesv1.FleetStatus{
			AllocatedReplicas: allocated,
			ReadyReplicas:     ready,
			Replicas:          total,
		},
	}
}

func fleetAutoScaler(fleetName string, fasName string) *autoscalingv1.FleetAutoscaler {
	return &autoscalingv1.FleetAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fasName,
			Namespace: "default",
			UID:       uuid.NewUUID(),
		},
		Spec: autoscalingv1.FleetAutoscalerSpec{
			FleetName: fleetName,
			Policy: autoscalingv1.FleetAutoscalerPolicy{
				Type: autoscalingv1.BufferPolicyType,
				Buffer: &autoscalingv1.BufferPolicy{
					MaxReplicas: 30,
					MinReplicas: 10,
					BufferSize:  intstr.FromInt(11),
				},
			},
		},
		Status: autoscalingv1.FleetAutoscalerStatus{
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

var nodeCountExpected = `# HELP agones_gameservers_node_count The count of gameservers per node in the cluster
# TYPE agones_gameservers_node_count histogram
agones_gameservers_node_count_bucket{le="1e-05"} 1
agones_gameservers_node_count_bucket{le="1.00001"} 2
agones_gameservers_node_count_bucket{le="2.00001"} 3
agones_gameservers_node_count_bucket{le="3.00001"} 3
agones_gameservers_node_count_bucket{le="4.00001"} 3
agones_gameservers_node_count_bucket{le="5.00001"} 3
agones_gameservers_node_count_bucket{le="6.00001"} 3
agones_gameservers_node_count_bucket{le="7.00001"} 3
agones_gameservers_node_count_bucket{le="8.00001"} 3
agones_gameservers_node_count_bucket{le="9.00001"} 3
agones_gameservers_node_count_bucket{le="10.00001"} 3
agones_gameservers_node_count_bucket{le="11.00001"} 3
agones_gameservers_node_count_bucket{le="12.00001"} 3
agones_gameservers_node_count_bucket{le="13.00001"} 3
agones_gameservers_node_count_bucket{le="14.00001"} 3
agones_gameservers_node_count_bucket{le="15.00001"} 3
agones_gameservers_node_count_bucket{le="16.00001"} 3
agones_gameservers_node_count_bucket{le="32.00001"} 3
agones_gameservers_node_count_bucket{le="40.00001"} 3
agones_gameservers_node_count_bucket{le="50.00001"} 3
agones_gameservers_node_count_bucket{le="60.00001"} 3
agones_gameservers_node_count_bucket{le="70.00001"} 3
agones_gameservers_node_count_bucket{le="80.00001"} 3
agones_gameservers_node_count_bucket{le="90.00001"} 3
agones_gameservers_node_count_bucket{le="100.00001"} 3
agones_gameservers_node_count_bucket{le="110.00001"} 3
agones_gameservers_node_count_bucket{le="120.00001"} 3
agones_gameservers_node_count_bucket{le="+Inf"} 3
agones_gameservers_node_count_sum 3
agones_gameservers_node_count_count 3
# HELP agones_nodes_count The count of nodes in the cluster
# TYPE agones_nodes_count gauge
agones_nodes_count{empty="false"} 2
agones_nodes_count{empty="true"} 1
`

func Test_parseLabels(t *testing.T) {
	tests := []struct {
		input   string
		want    *stackdriver.Labels
		wantErr bool
	}{
		{
			"",
			labelsFromMap(nil),
			false,
		},
		{
			"a=b",
			labelsFromMap(map[string]string{"a": "b"}),
			false,
		},
		{
			"a=b,",
			nil,
			true,
		},
		{
			"a=b,c",
			nil,
			true,
		},
		{
			"a=b,c=d",
			labelsFromMap(map[string]string{"a": "b", "c": "d"}),
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := parseLabels(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseLabels() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseLabels() = %v, want %v", got, tt.want)
			}
		})
	}
}

func labelsFromMap(m map[string]string) *stackdriver.Labels {
	res := &stackdriver.Labels{}
	for k, v := range m {
		res.Set(k, v, "")
	}
	return res
}
