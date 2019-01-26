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
	"strings"
	"testing"

	"agones.dev/agones/pkg/apis/stable/v1alpha1"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func TestControllerGameServerCount(t *testing.T) {

	registry := prometheus.NewRegistry()
	_, err := RegisterPrometheusExporter(registry)
	assert.Nil(t, err)

	c := newFakeController()
	defer c.close()

	gs1 := gameServer("test-fleet", v1alpha1.GameServerStateCreating)
	c.gsWatch.Add(gs1)
	gs1 = gs1.DeepCopy()
	gs1.Status.State = v1alpha1.GameServerStateReady
	c.gsWatch.Modify(gs1)

	c.sync()
	c.collect()
	report()

	gs1 = gs1.DeepCopy()
	gs1.Status.State = v1alpha1.GameServerStateShutdown
	c.gsWatch.Modify(gs1)
	c.gsWatch.Add(gameServer("", v1alpha1.GameServerStatePortAllocation))
	c.gsWatch.Add(gameServer("", v1alpha1.GameServerStatePortAllocation))

	c.sync()
	c.collect()
	report()

	assert.Nil(t, testutil.GatherAndCompare(registry, strings.NewReader(gsCountExpected), "agones_gameservers_count"))
}

func TestControllerFleetAllocationCount(t *testing.T) {

	registry := prometheus.NewRegistry()
	_, err := RegisterPrometheusExporter(registry)
	assert.Nil(t, err)

	c := newFakeController()
	defer c.close()

	fa1 := fleetAllocation("deleted-fleet")
	c.faWatch.Add(fa1)
	c.faWatch.Add(fleetAllocation("test-fleet"))
	c.faWatch.Add(fleetAllocation("test-fleet"))
	c.faWatch.Add(fleetAllocation("test-fleet2"))

	c.sync()
	c.collect()
	report()

	c.faWatch.Delete(fa1)
	c.faWatch.Add(fleetAllocation("test-fleet"))
	c.faWatch.Add(fleetAllocation("test-fleet2"))

	c.sync()
	c.collect()
	report()

	assert.Nil(t, testutil.GatherAndCompare(registry, strings.NewReader(faCountExpected), "agones_fleet_allocations_count"))
}

func TestControllerFleetAllocationTotal(t *testing.T) {

	registry := prometheus.NewRegistry()
	_, err := RegisterPrometheusExporter(registry)
	assert.Nil(t, err)

	c := newFakeController()
	defer c.close()
	c.run(t)
	// non allocated should not be counted
	fa1 := fleetAllocation("unallocated")
	fa1.Status.GameServer = nil
	c.faWatch.Add(fa1)
	c.faWatch.Delete(fa1)

	for i := 0; i < 3; i++ {
		fa := fleetAllocation("test")
		// only fleet allocation that were not allocated to a gameserver are collected
		// this way we avoid counting multiple update events.
		fa.Status.GameServer = nil
		c.faWatch.Add(fa)
		faUpdated := fa.DeepCopy()
		faUpdated.Status.GameServer = gameServer("test", v1alpha1.GameServerStateAllocated)
		c.faWatch.Modify(faUpdated)
		// make sure we count only one event
		c.faWatch.Modify(faUpdated)
	}
	for i := 0; i < 2; i++ {
		fa := fleetAllocation("test2")
		fa.Status.GameServer = nil
		c.faWatch.Add(fa)
		faUpdated := fa.DeepCopy()
		faUpdated.Status.GameServer = gameServer("test2", v1alpha1.GameServerStateAllocated)
		c.faWatch.Modify(faUpdated)
	}
	c.sync()
	report()

	assert.Nil(t, testutil.GatherAndCompare(registry, strings.NewReader(faTotalExpected), "agones_fleet_allocations_total"))
}

func TestControllerGameServersTotal(t *testing.T) {

	registry := prometheus.NewRegistry()
	_, err := RegisterPrometheusExporter(registry)
	assert.Nil(t, err)

	c := newFakeController()
	defer c.close()
	c.run(t)

	// deleted gs should not be counted
	gs := gameServer("deleted", v1alpha1.GameServerStateCreating)
	c.gsWatch.Add(gs)
	c.gsWatch.Delete(gs)

	generateGsEvents(16, v1alpha1.GameServerStateCreating, "test", c.gsWatch)
	generateGsEvents(15, v1alpha1.GameServerStateScheduled, "test", c.gsWatch)
	generateGsEvents(10, v1alpha1.GameServerStateStarting, "test", c.gsWatch)
	generateGsEvents(1, v1alpha1.GameServerStateUnhealthy, "test", c.gsWatch)
	generateGsEvents(19, v1alpha1.GameServerStateCreating, "", c.gsWatch)
	generateGsEvents(18, v1alpha1.GameServerStateScheduled, "", c.gsWatch)
	generateGsEvents(16, v1alpha1.GameServerStateStarting, "", c.gsWatch)
	generateGsEvents(1, v1alpha1.GameServerStateUnhealthy, "", c.gsWatch)

	c.sync()
	report()

	assert.Nil(t, testutil.GatherAndCompare(registry, strings.NewReader(gsTotalExpected), "agones_gameservers_total"))
}

func TestControllerFleetReplicasCount(t *testing.T) {

	registry := prometheus.NewRegistry()
	_, err := RegisterPrometheusExporter(registry)
	assert.Nil(t, err)

	c := newFakeController()
	defer c.close()
	c.run(t)

	f := fleet("fleet-test", 8, 2, 5, 1)
	fd := fleet("fleet-deleted", 100, 100, 100, 100)
	c.fleetWatch.Add(f)
	f = f.DeepCopy()
	f.Status.ReadyReplicas = 1
	f.Spec.Replicas = 5
	c.fleetWatch.Modify(f)
	c.fleetWatch.Add(fd)
	c.fleetWatch.Delete(fd)

	c.sync()
	report()

	assert.Nil(t, testutil.GatherAndCompare(registry, strings.NewReader(fleetReplicasCountExpected), "agones_fleets_replicas_count"))
}

func TestControllerFleetAutoScalerState(t *testing.T) {
	registry := prometheus.NewRegistry()
	_, err := RegisterPrometheusExporter(registry)
	assert.Nil(t, err)

	c := newFakeController()
	defer c.close()
	c.run(t)

	// testing fleet name change
	fasFleetNameChange := fleetAutoScaler("first-fleet", "name-switch")
	c.fasWatch.Add(fasFleetNameChange)
	fasFleetNameChange = fasFleetNameChange.DeepCopy()
	fasFleetNameChange.Spec.Policy.Buffer.BufferSize = intstr.FromInt(10)
	fasFleetNameChange.Spec.Policy.Buffer.MaxReplicas = 50
	fasFleetNameChange.Spec.Policy.Buffer.MinReplicas = 10
	fasFleetNameChange.Status.CurrentReplicas = 20
	fasFleetNameChange.Status.DesiredReplicas = 10
	fasFleetNameChange.Status.ScalingLimited = true
	c.fasWatch.Modify(fasFleetNameChange)
	fasFleetNameChange = fasFleetNameChange.DeepCopy()
	fasFleetNameChange.Spec.FleetName = "second-fleet"
	c.fasWatch.Modify(fasFleetNameChange)
	// testing deletion
	fasDeleted := fleetAutoScaler("deleted-fleet", "deleted")
	fasDeleted.Spec.Policy.Buffer.BufferSize = intstr.FromString("50%")
	fasDeleted.Spec.Policy.Buffer.MaxReplicas = 150
	fasDeleted.Spec.Policy.Buffer.MinReplicas = 15
	c.fasWatch.Add(fasDeleted)
	c.fasWatch.Delete(fasDeleted)

	c.sync()
	report()

	assert.Nil(t, testutil.GatherAndCompare(registry, strings.NewReader(fasStateExpected),
		"agones_fleet_autoscalers_able_to_scale", "agones_fleet_autoscalers_buffer_limits", "agones_fleet_autoscalers_buffer_size",
		"agones_fleet_autoscalers_current_replicas_count", "agones_fleet_autoscalers_desired_replicas_count", "agones_fleet_autoscalers_limited"))

}
