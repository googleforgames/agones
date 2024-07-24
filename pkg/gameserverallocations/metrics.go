// Copyright 2019 Google LLC All Rights Reserved.
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

/*
package gameserverallocations

import (
	"context"
	"strconv"
	"time"

	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	allocationv1 "agones.dev/agones/pkg/apis/allocation/v1"
	listerv1 "agones.dev/agones/pkg/client/listers/agones/v1"
	mt "agones.dev/agones/pkg/metrics"
	"agones.dev/agones/pkg/util/runtime"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
)

var (
	logger = runtime.NewLoggerWithSource("metrics")

	keyFleetName          = mt.MustTagKey("fleet_name")
	keyClusterName        = mt.MustTagKey("cluster_name")
	keyMultiCluster       = mt.MustTagKey("is_multicluster")
	keyStatus             = mt.MustTagKey("status")
	keySchedulingStrategy = mt.MustTagKey("scheduling_strategy")
	keyRetryCount         = mt.MustTagKey("retry_count")

	gameServerAllocationsLatency   = stats.Float64("gameserver_allocations/latency", "The duration of gameserver allocations", "s")
	gameServerAllocationsErrorRate = stats.Int64("gameserver_allocations/errors", "The errors of gameserver allocations", "1")
	gameServerAllocationsTest      = stats.Int64("gameserver_allocations/test", "The test of gameserver allocations", "1")
)

func init() {

	stateViews := []*view.View{
		{
			Name:        "gameserver_allocations_duration_seconds",
			Measure:     gameServerAllocationsLatency,
			Description: "The distribution of gameserver allocation requests latencies.",
			Aggregation: view.Distribution(0, 0.01, 0.025, 0.05, 0.075, 0.1, 0.25, 0.5, 0.75, 1, 2, 3),
			TagKeys:     []tag.Key{keyFleetName, keyClusterName, keyMultiCluster, keyStatus, keySchedulingStrategy},
		},
		{
			Name:        "gameserver_allocations_error_rate",
			Measure:     gameServerAllocationsErrorRate,
			Description: "The distribution of gameserver allocation errors",
			Aggregation: view.Count(),
			TagKeys:     []tag.Key{keyFleetName, keyClusterName, keyMultiCluster, keyStatus, keySchedulingStrategy, keyRetryCount},
		},
		{
			Name:        "gameserver_allocations_testttt",
			Measure:     gameServerAllocationsTest,
			Description: "The distribution of gameserver allocation test",
			Aggregation: view.Count(),
			TagKeys:     []tag.Key{keyFleetName, keyClusterName, keyMultiCluster, keyStatus, keySchedulingStrategy, keyRetryCount},
		},
	}
	//runtime.Must(view.Register(stateViews))
	for _, v := range stateViews {
		if err := view.Register(v); err != nil {
			logger.WithError(err).Error("could not register view")
		}
	}

	/*runtime.Must(view.Register(&view.View{
		Name:        "gameserver_allocations_duration_seconds",
		Measure:     gameServerAllocationsLatency,
		Description: "The distribution of gameserver allocation requests latencies.",
		Aggregation: view.Distribution(0, 0.01, 0.025, 0.05, 0.075, 0.1, 0.25, 0.5, 0.75, 1, 2, 3),
		TagKeys:     []tag.Key{keyFleetName, keyClusterName, keyMultiCluster, keyStatus, keySchedulingStrategy},
	}))
}

// default set of tags for latency metric
var latencyTags = []tag.Mutator{
	tag.Insert(keyMultiCluster, "none"),
	tag.Insert(keyClusterName, "none"),
	tag.Insert(keySchedulingStrategy, "none"),
	tag.Insert(keyFleetName, "none"),
	tag.Insert(keyStatus, "none"),
	tag.Insert(keyRetryCount, "none"),
}

type metrics struct {
	ctx              context.Context
	gameServerLister listerv1.GameServerLister
	logger           *logrus.Entry
	start            time.Time
}

// mutate the current set of metric tags
func (r *metrics) mutate(m ...tag.Mutator) {
	var err error
	r.ctx, err = tag.New(r.ctx, m...)
	if err != nil {
		r.logger.WithError(err).Warn("failed to mutate request context.")
	}
}

// setStatus set the latency status tag.
func (r *metrics) setStatus(status string) {
	r.mutate(tag.Update(keyStatus, status))
}

// setError set the latency status tag as error.
func (r *metrics) setError(errorType string) {
	r.mutate(tag.Update(keyStatus, errorType))
}

// setRetry set the latency status tag as error.
func (r *metrics) setRetry(retryCount string) {
	r.mutate(tag.Update(keyRetryCount, retryCount))
}

// setRequest set request metric tags.
func (r *metrics) setRequest(in *allocationv1.GameServerAllocation) {
	tags := []tag.Mutator{
		tag.Update(keySchedulingStrategy, string(in.Spec.Scheduling)),
	}

	tags = append(tags, tag.Update(keyMultiCluster, strconv.FormatBool(in.Spec.MultiClusterSetting.Enabled)))
	r.mutate(tags...)
}

// setResponse set response metric tags.
func (r *metrics) setResponse(o k8sruntime.Object) {
	out, ok := o.(*allocationv1.GameServerAllocation)
	if out == nil || !ok {
		return
	}
	r.setStatus(string(out.Status.State))
	var tags []tag.Mutator
	// sets the fleet name tag if possible
	if out.Status.State == allocationv1.GameServerAllocationAllocated && out.Status.Source == localAllocationSource {
		gs, err := r.gameServerLister.GameServers(out.Namespace).Get(out.Status.GameServerName)
		if err != nil {
			r.logger.WithError(err).Warnf("failed to get gameserver:%s namespace:%s", out.Status.GameServerName, out.Namespace)
			return
		}
		fleetName := gs.Labels[agonesv1.FleetNameLabel]
		if fleetName != "" {
			tags = append(tags, tag.Update(keyFleetName, fleetName))
		}
	}
	r.mutate(tags...)
}

// record the current allocation latency.
func (r *metrics) record() {
	stats.Record(r.ctx, gameServerAllocationsLatency.M(time.Since(r.start).Seconds()))
}
*/
// record the current allocation error rate.
// We only get "Conflict" error types which correspond to status code 409 which maps
// to StatusReasonAlreadyExists error as shown below
/*
From vendor/k8s.io/apimachinery/pkg/apis/meta/v1/types.go
	// StatusReasonAlreadyExists means the resource you are creating already exists.
	// Details (optional):
	//   "kind" string - the kind attribute of the conflicting resource
	//   "id"   string - the identifier of the conflicting resource
	// Status code 409
	StatusReasonAlreadyExists StatusReason = "AlreadyExists"

	// StatusReasonConflict means the requested operation cannot be completed
	// due to a conflict in the operation. The client may need to alter the
	// request. Each resource may define custom details that indicate the
	// nature of the conflict.
	// Status code 409
	StatusReasonConflict StatusReason = "Conflict"

	func (r *metrics) recordAllocationErrorRate() {
	stats.Record(r.ctx, gameServerAllocationsErrorRate.M(int64(1)))
	stats.Record(r.ctx, gameServerAllocationsTest.M(int64(1)))

}
*/

// Copyright 2019 Google LLC All Rights Reserved.
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
	"strconv"
	"time"

	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	allocationv1 "agones.dev/agones/pkg/apis/allocation/v1"
	listerv1 "agones.dev/agones/pkg/client/listers/agones/v1"
	mt "agones.dev/agones/pkg/metrics"
	"agones.dev/agones/pkg/util/runtime"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
)

var (
	keyFleetName          = mt.MustTagKey("fleet_name")
	keyClusterName        = mt.MustTagKey("cluster_name")
	keyMultiCluster       = mt.MustTagKey("is_multicluster")
	keyStatus             = mt.MustTagKey("status")
	keySchedulingStrategy = mt.MustTagKey("scheduling_strategy")

	gameServerAllocationsLatency = stats.Float64("gameserver_allocations/latency", "The duration of gameserver allocations", "s")
)

func init() {
	runtime.Must(view.Register(&view.View{
		Name:        "gameserver_allocations_duration_seconds",
		Measure:     gameServerAllocationsLatency,
		Description: "The distribution of gameserver allocation requests latencies.",
		Aggregation: view.Distribution(0, 0.01, 0.025, 0.05, 0.075, 0.1, 0.25, 0.5, 0.75, 1, 2, 3),
		TagKeys:     []tag.Key{keyFleetName, keyClusterName, keyMultiCluster, keyStatus, keySchedulingStrategy},
	}))
}

// default set of tags for latency metric
var latencyTags = []tag.Mutator{
	tag.Insert(keyMultiCluster, "none"),
	tag.Insert(keyClusterName, "none"),
	tag.Insert(keySchedulingStrategy, "none"),
	tag.Insert(keyFleetName, "none"),
	tag.Insert(keyStatus, "none"),
}

type metrics struct {
	ctx              context.Context
	gameServerLister listerv1.GameServerLister
	logger           *logrus.Entry
	start            time.Time
}

// mutate the current set of metric tags
func (r *metrics) mutate(m ...tag.Mutator) {
	var err error
	r.ctx, err = tag.New(r.ctx, m...)
	if err != nil {
		r.logger.WithError(err).Warn("failed to mutate request context.")
	}
}

// setStatus set the latency status tag.
func (r *metrics) setStatus(status string) {
	r.mutate(tag.Update(keyStatus, status))
}

// setError set the latency status tag as error.
func (r *metrics) setError() {
	r.mutate(tag.Update(keyStatus, "error"))
}

// setRequest set request metric tags.
func (r *metrics) setRequest(in *allocationv1.GameServerAllocation) {
	tags := []tag.Mutator{
		tag.Update(keySchedulingStrategy, string(in.Spec.Scheduling)),
	}

	tags = append(tags, tag.Update(keyMultiCluster, strconv.FormatBool(in.Spec.MultiClusterSetting.Enabled)))
	r.mutate(tags...)
}

// setResponse set response metric tags.
func (r *metrics) setResponse(o k8sruntime.Object) {
	out, ok := o.(*allocationv1.GameServerAllocation)
	if out == nil || !ok {
		return
	}
	r.setStatus(string(out.Status.State))
	var tags []tag.Mutator
	// sets the fleet name tag if possible
	if out.Status.State == allocationv1.GameServerAllocationAllocated && out.Status.Source == localAllocationSource {
		gs, err := r.gameServerLister.GameServers(out.Namespace).Get(out.Status.GameServerName)
		if err != nil {
			r.logger.WithError(err).Warnf("failed to get gameserver:%s namespace:%s", out.Status.GameServerName, out.Namespace)
			return
		}
		fleetName := gs.Labels[agonesv1.FleetNameLabel]
		if fleetName != "" {
			tags = append(tags, tag.Update(keyFleetName, fleetName))
		}
	}
	r.mutate(tags...)
}

// record the current allocation latency.
func (r *metrics) record() {
	stats.Record(r.ctx, gameServerAllocationsLatency.M(time.Since(r.start).Seconds()))
}
