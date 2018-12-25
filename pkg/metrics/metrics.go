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
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
)

var (
	keyName      = mustTagKey("name")
	keyFleetName = mustTagKey("fleet_name")
	keyType      = mustTagKey("type")

	fleetsReplicasCountStats = stats.Int64("fleets/replicas_count", "The count of replicas per fleet", "1")
	fleetsReplicasCountView  = &view.View{
		Name:        "fleets_replicas_count",
		Measure:     fleetsReplicasCountStats,
		Description: "The number of replicas per fleet",
		Aggregation: view.LastValue(),
		TagKeys:     []tag.Key{keyName, keyType},
	}

	fasBufferLimitsCountStats = stats.Int64("fas/buffer_limits", "The buffer limits of autoscalers", "1")
	fasBufferLimitsCountView  = &view.View{
		Name:        "fleet_autoscalers_buffer_limits",
		Measure:     fasBufferLimitsCountStats,
		Description: "The limits of buffer based fleet autoscalers",
		Aggregation: view.LastValue(),
		TagKeys:     []tag.Key{keyName, keyType, keyFleetName},
	}

	fasBufferSizeStats = stats.Int64("fas/buffer_size", "The buffer size value of autoscalers", "1")
	fasBufferSizeView  = &view.View{
		Name:        "fleet_autoscalers_buffer_size",
		Measure:     fasBufferSizeStats,
		Description: "The buffer size of fleet autoscalers",
		Aggregation: view.LastValue(),
		TagKeys:     []tag.Key{keyName, keyType, keyFleetName},
	}

	fasCurrentReplicasStats = stats.Int64("fas/current_replicas_count", "The current replicas cout as seen by autoscalers", "1")
	fasCurrentReplicasView  = &view.View{
		Name:        "fleet_autoscalers_current_replicas_count",
		Measure:     fasCurrentReplicasStats,
		Description: "The current replicas count as seen by autoscalers",
		Aggregation: view.LastValue(),
		TagKeys:     []tag.Key{keyName, keyFleetName},
	}

	fasDesiredReplicasStats = stats.Int64("fas/desired_replicas_count", "The desired replicas cout as seen by autoscalers", "1")
	fasDesiredReplicasView  = &view.View{
		Name:        "fleet_autoscalers_desired_replicas_count",
		Measure:     fasDesiredReplicasStats,
		Description: "The desired replicas count as seen by autoscalers",
		Aggregation: view.LastValue(),
		TagKeys:     []tag.Key{keyName, keyFleetName},
	}

	fasAbleToScaleStats = stats.Int64("fas/able_to_scale", "The fleet autoscaler can access the fleet to scale (0 indicates false, 1 indicates true)", "1")
	fasAbleToScaleView  = &view.View{
		Name:        "fleet_autoscalers_able_to_scale",
		Measure:     fasAbleToScaleStats,
		Description: "The fleet autoscaler can access the fleet to scale",
		Aggregation: view.LastValue(),
		TagKeys:     []tag.Key{keyName, keyFleetName},
	}

	fasLimitedStats = stats.Int64("fas/limited", "The fleet autoscaler is capped (0 indicates false, 1 indicates true)", "1")
	fasLimitedView  = &view.View{
		Name:        "fleet_autoscalers_limited",
		Measure:     fasLimitedStats,
		Description: "The fleet autoscaler is capped",
		Aggregation: view.LastValue(),
		TagKeys:     []tag.Key{keyName, keyFleetName},
	}

	gameServerCountStats = stats.Int64("gameservers/count", "The count of gameservers", "1")
	gameServersCountView = &view.View{
		Name:        "gameservers_count",
		Measure:     gameServerCountStats,
		Description: "The number of gameservers",
		Aggregation: view.LastValue(),
		TagKeys:     []tag.Key{keyType, keyFleetName},
	}

	fleetAllocationCountStats = stats.Int64("fleet_allocations/count", "The count of fleet allocations", "1")
	fleetAllocationCountView  = &view.View{
		Name:        "fleet_allocations_count",
		Measure:     fleetAllocationCountStats,
		Description: "The number of fleet allocations",
		Aggregation: view.LastValue(),
		TagKeys:     []tag.Key{keyFleetName},
	}

	fleetAllocationTotalStats = stats.Int64("fleet_allocations/total", "The total of fleet allocations", "1")
	fleetAllocationTotalView  = &view.View{
		Name:        "fleet_allocations_total",
		Measure:     fleetAllocationTotalStats,
		Description: "The total of fleet allocations",
		Aggregation: view.Count(),
		TagKeys:     []tag.Key{keyFleetName},
	}

	gameServerTotalStats = stats.Int64("gameservers/total", "The total of gameservers", "1")
	gameServersTotalView = &view.View{
		Name:        "gameservers_total",
		Measure:     gameServerTotalStats,
		Description: "The total of gameservers",
		Aggregation: view.Count(),
		TagKeys:     []tag.Key{keyType, keyFleetName},
	}

	views = []*view.View{fleetsReplicasCountView, gameServersCountView, gameServersTotalView,
		fasBufferSizeView, fasBufferLimitsCountView, fasCurrentReplicasView, fasDesiredReplicasView,
		fasAbleToScaleView, fasLimitedView, fleetAllocationCountView, fleetAllocationTotalView}
)

func mustTagKey(key string) tag.Key {
	t, err := tag.NewKey(key)
	if err != nil {
		panic(err)
	}
	return t
}
