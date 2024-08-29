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

package gameservers

import (
	"context"
	"time"

	listerv1 "agones.dev/agones/pkg/client/listers/agones/v1"
	mt "agones.dev/agones/pkg/metrics"
	"agones.dev/agones/pkg/util/runtime"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
)

var (
	logger = runtime.NewLoggerWithSource("metrics")

	keyFleetName          = mt.MustTagKey("fleet_name")
	keyClusterName        = mt.MustTagKey("cluster_name")
	keyMultiCluster       = mt.MustTagKey("is_multicluster")
	keyStatus             = mt.MustTagKey("status")
	keySchedulingStrategy = mt.MustTagKey("scheduling_strategy")

	gameServerUpdateErrorTotal = stats.Int64("gameserver_updates/errors", "The errors of gameserver updates", "1")
)

func init() {

	stateViews := []*view.View{
		{
			Name:        "gameserver_updates_errors_total",
			Measure:     gameServerUpdateErrorTotal,
			Description: "The count of gameserver update errors",
			Aggregation: view.Distribution(1, 2, 3, 4, 5, 6, 7, 8),
			TagKeys:     []tag.Key{keyFleetName, keyClusterName, keyMultiCluster, keyStatus, keySchedulingStrategy},
		},
	}

	for _, v := range stateViews {
		if err := view.Register(v); err != nil {
			logger.WithError(err).Error("could not register view")
		}
	}
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

// record the current allocation retry rate.
func (r *metrics) recordUpdateError(ctx context.Context, retryCount int64, errorType string) {
	mt.RecordWithTags(ctx, []tag.Mutator{tag.Upsert(keyStatus, "Error in  "+errorType)},
		gameServerUpdateErrorTotal.M(retryCount))
}
