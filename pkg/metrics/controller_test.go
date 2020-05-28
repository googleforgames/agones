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
	"strings"
	"testing"

	"go.opencensus.io/metric/metricdata"
	"go.opencensus.io/metric/metricexport"
	"k8s.io/apimachinery/pkg/util/intstr"

	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	"github.com/stretchr/testify/assert"
)

type metricExporter struct {
	metrics []*metricdata.Metric
}

func (e *metricExporter) ExportMetrics(ctx context.Context, metrics []*metricdata.Metric) error {
	e.metrics = metrics
	return nil
}

func serialize(args []string) string {
	return strings.Join(args, "|")
}

type expectedMetricData struct {
	labels []string
	val    interface{}
}

func assertMetricData(t *testing.T, exporter *metricExporter, metricName string, expected []expectedMetricData) {

	expectedValuesAsMap := make(map[string]expectedMetricData)
	for _, e := range expected {
		expectedValuesAsMap[serialize(e.labels)] = e
	}

	var wantedMetric *metricdata.Metric
	for _, m := range exporter.metrics {
		if m.Descriptor.Name == metricName {
			wantedMetric = m
		}
	}
	assert.NotNil(t, wantedMetric, "No metric found with name: %s", metricName)

	assert.Equal(t, len(expectedValuesAsMap), len(expected), "Multiple entries in 'expected' slice have the exact same labels")
	assert.Equal(t, len(wantedMetric.TimeSeries), len(expectedValuesAsMap), "number of timeseries does not match")
	for _, tsd := range wantedMetric.TimeSeries {
		actualLabelValues := make([]string, len(tsd.LabelValues))
		for i, k := range tsd.LabelValues {
			actualLabelValues[i] = k.Value
		}
		e, ok := expectedValuesAsMap[serialize(actualLabelValues)]
		assert.True(t, ok, "no TimeSeries found with labels: %v", actualLabelValues)
		assert.Equal(t, actualLabelValues, e.labels, "label values don't match")
		assert.Equal(t, len(tsd.Points), 1, "assertMetricDataValues can only handle a single Point in a TimeSeries")
		assert.Equal(t, tsd.Points[0].Value, e.val, "metric: %s, tags: %v, values don't match; got: %v, want: %v", metricName, tsd.LabelValues, tsd.Points[0].Value, e.val)
	}
}

func resetMetrics() {
	unRegisterViews()
	registerViews()
}

func TestControllerGameServerCount(t *testing.T) {
	resetMetrics()
	exporter := &metricExporter{}
	reader := metricexport.NewReader()

	c := newFakeController()
	defer c.close()

	gs1 := gameServerWithFleetAndState("test-fleet", agonesv1.GameServerStateCreating)
	c.gsWatch.Add(gs1)
	gs1 = gs1.DeepCopy()
	gs1.Status.State = agonesv1.GameServerStateReady
	c.gsWatch.Modify(gs1)

	c.run(t)
	c.sync()
	c.collect()

	gs1 = gs1.DeepCopy()
	gs1.Status.State = agonesv1.GameServerStateShutdown
	c.gsWatch.Modify(gs1)
	c.gsWatch.Add(gameServerWithFleetAndState("", agonesv1.GameServerStatePortAllocation))
	c.gsWatch.Add(gameServerWithFleetAndState("", agonesv1.GameServerStatePortAllocation))

	c.run(t)
	c.sync()
	c.collect()
	reader.ReadAndExport(exporter)
	assertMetricData(t, exporter, "gameservers_count", []expectedMetricData{
		{labels: []string{"test-fleet", "default", "Ready"}, val: int64(0)},
		{labels: []string{"test-fleet", "default", "Shutdown"}, val: int64(1)},
		{labels: []string{"none", "default", "PortAllocation"}, val: int64(2)},
	})
}

func TestControllerGameServersTotal(t *testing.T) {
	resetMetrics()
	exporter := &metricExporter{}
	reader := metricexport.NewReader()
	c := newFakeController()
	defer c.close()
	c.run(t)

	// deleted gs should not be counted
	gs := gameServerWithFleetAndState("deleted", agonesv1.GameServerStateCreating)
	c.gsWatch.Add(gs)
	c.gsWatch.Delete(gs)

	generateGsEvents(16, agonesv1.GameServerStateCreating, "test", c.gsWatch)
	generateGsEvents(15, agonesv1.GameServerStateScheduled, "test", c.gsWatch)
	generateGsEvents(10, agonesv1.GameServerStateStarting, "test", c.gsWatch)
	generateGsEvents(1, agonesv1.GameServerStateUnhealthy, "test", c.gsWatch)
	generateGsEvents(19, agonesv1.GameServerStateCreating, "", c.gsWatch)
	generateGsEvents(18, agonesv1.GameServerStateScheduled, "", c.gsWatch)
	generateGsEvents(16, agonesv1.GameServerStateStarting, "", c.gsWatch)
	generateGsEvents(1, agonesv1.GameServerStateUnhealthy, "", c.gsWatch)

	c.sync()
	reader.ReadAndExport(exporter)
	assertMetricData(t, exporter, "gameservers_total", []expectedMetricData{
		{labels: []string{"test", "default", "Creating"}, val: int64(16)},
		{labels: []string{"test", "default", "Scheduled"}, val: int64(15)},
		{labels: []string{"test", "default", "Starting"}, val: int64(10)},
		{labels: []string{"test", "default", "Unhealthy"}, val: int64(1)},
		{labels: []string{"none", "default", "Creating"}, val: int64(19)},
		{labels: []string{"none", "default", "Scheduled"}, val: int64(18)},
		{labels: []string{"none", "default", "Starting"}, val: int64(16)},
		{labels: []string{"none", "default", "Unhealthy"}, val: int64(1)},
	})
}

func TestControllerFleetReplicasCount(t *testing.T) {

	resetMetrics()
	exporter := &metricExporter{}
	reader := metricexport.NewReader()
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

	reader.ReadAndExport(exporter)
	assertMetricData(t, exporter, "fleets_replicas_count", []expectedMetricData{
		{labels: []string{"fleet-deleted", "default", "allocated"}, val: int64(0)},
		{labels: []string{"fleet-deleted", "default", "desired"}, val: int64(0)},
		{labels: []string{"fleet-deleted", "default", "ready"}, val: int64(0)},
		{labels: []string{"fleet-deleted", "default", "total"}, val: int64(0)},
		{labels: []string{"fleet-test", "default", "allocated"}, val: int64(2)},
		{labels: []string{"fleet-test", "default", "desired"}, val: int64(5)},
		{labels: []string{"fleet-test", "default", "ready"}, val: int64(1)},
		{labels: []string{"fleet-test", "default", "total"}, val: int64(8)},
	})
}

func TestControllerFleetAutoScalerState(t *testing.T) {
	resetMetrics()
	exporter := &metricExporter{}
	reader := metricexport.NewReader()
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

	reader.ReadAndExport(exporter)
	assertMetricData(t, exporter, "fleet_autoscalers_able_to_scale", []expectedMetricData{
		{labels: []string{"first-fleet", "name-switch", "default"}, val: int64(0)},
		{labels: []string{"second-fleet", "name-switch", "default"}, val: int64(1)},
		{labels: []string{"deleted-fleet", "deleted", "default"}, val: int64(0)},
	})
	assertMetricData(t, exporter, "fleet_autoscalers_buffer_limits", []expectedMetricData{
		{labels: []string{"first-fleet", "name-switch", "default", "max"}, val: int64(50)},
		{labels: []string{"first-fleet", "name-switch", "default", "min"}, val: int64(10)},
		{labels: []string{"second-fleet", "name-switch", "default", "max"}, val: int64(50)},
		{labels: []string{"second-fleet", "name-switch", "default", "min"}, val: int64(10)},
		{labels: []string{"deleted-fleet", "deleted", "default", "max"}, val: int64(150)},
		{labels: []string{"deleted-fleet", "deleted", "default", "min"}, val: int64(15)},
	})
	assertMetricData(t, exporter, "fleet_autoscalers_buffer_size", []expectedMetricData{
		{labels: []string{"first-fleet", "name-switch", "default", "count"}, val: int64(10)},
		{labels: []string{"second-fleet", "name-switch", "default", "count"}, val: int64(10)},
		{labels: []string{"deleted-fleet", "deleted", "default", "percentage"}, val: int64(50)},
	})
	assertMetricData(t, exporter, "fleet_autoscalers_current_replicas_count", []expectedMetricData{
		{labels: []string{"first-fleet", "name-switch", "default"}, val: int64(0)},
		{labels: []string{"second-fleet", "name-switch", "default"}, val: int64(20)},
		{labels: []string{"deleted-fleet", "deleted", "default"}, val: int64(0)},
	})
	assertMetricData(t, exporter, "fleet_autoscalers_desired_replicas_count", []expectedMetricData{
		{labels: []string{"first-fleet", "name-switch", "default"}, val: int64(0)},
		{labels: []string{"second-fleet", "name-switch", "default"}, val: int64(10)},
		{labels: []string{"deleted-fleet", "deleted", "default"}, val: int64(0)},
	})
	assertMetricData(t, exporter, "fleet_autoscalers_limited", []expectedMetricData{
		{labels: []string{"first-fleet", "name-switch", "default"}, val: int64(0)},
		{labels: []string{"second-fleet", "name-switch", "default"}, val: int64(1)},
		{labels: []string{"deleted-fleet", "deleted", "default"}, val: int64(0)},
	})
}

func TestControllerGameServersNodeState(t *testing.T) {
	resetMetrics()
	c := newFakeController()
	defer c.close()
	c.nodeWatch.Add(nodeWithName("node1"))
	c.nodeWatch.Add(nodeWithName("node2"))
	c.nodeWatch.Add(nodeWithName("node3"))
	c.gsWatch.Add(gameServerWithNode("node1"))
	c.gsWatch.Add(gameServerWithNode("node2"))
	c.gsWatch.Add(gameServerWithNode("node2"))

	c.run(t)
	c.sync()

	exporter := &metricExporter{}
	reader := metricexport.NewReader()
	reader.ReadAndExport(exporter)
	assertMetricData(t, exporter, "gameservers_node_count", []expectedMetricData{
		{labels: []string{}, val: &metricdata.Distribution{
			Count:                 3,
			Sum:                   3,
			SumOfSquaredDeviation: 2,
			BucketOptions:         &metricdata.BucketOptions{Bounds: []float64{0.00001, 1.00001, 2.00001, 3.00001, 4.00001, 5.00001, 6.00001, 7.00001, 8.00001, 9.00001, 10.00001, 11.00001, 12.00001, 13.00001, 14.00001, 15.00001, 16.00001, 32.00001, 40.00001, 50.00001, 60.00001, 70.00001, 80.00001, 90.00001, 100.00001, 110.00001, 120.00001}},
			Buckets:               []metricdata.Bucket{{Count: 1}, {Count: 1}, {Count: 1}, {Count: 0}, {Count: 0}, {Count: 0}, {Count: 0}, {Count: 0}, {Count: 0}, {Count: 0}, {Count: 0}, {Count: 0}, {Count: 0}, {Count: 0}, {Count: 0}, {Count: 0}, {Count: 0}, {Count: 0}, {Count: 0}, {Count: 0}, {Count: 0}, {Count: 0}, {Count: 0}, {Count: 0}, {Count: 0}, {Count: 0}, {Count: 0}, {Count: 0}}}},
	})
	assertMetricData(t, exporter, "nodes_count", []expectedMetricData{
		{labels: []string{"true"}, val: int64(1)},
		{labels: []string{"false"}, val: int64(2)},
	})
}
