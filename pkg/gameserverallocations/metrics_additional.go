package gameserverallocations

import (
	"context"
	"time"

	mt "agones.dev/agones/pkg/metrics"
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
)

var (
	gameServerAllocationsUpdatesLatency = stats.Float64("gameserver_allocations/update_latency", "The duration of gameserver updates", "s")
	gameServerAllocationsBatchSize      = stats.Int64("gameserver_allocations/batch", "The gameserver allocations batch size", "1")
)

func init() {

	stateViews := []*view.View{
		{
			Name:        "gameserver_allocations_updates_duration_seconds",
			Measure:     gameServerAllocationsUpdatesLatency,
			Description: "The distribution of gameserver allocation update requests latencies.",
			Aggregation: view.Distribution(0, 0.01, 0.025, 0.05, 0.075, 0.1, 0.25, 0.5, 0.75, 1, 2, 3),
			TagKeys:     []tag.Key{keyFleetName, keyClusterName, keyMultiCluster, keyStatus, keySchedulingStrategy},
		},
		{
			Name:        "gameserver_allocations_batch_size",
			Measure:     gameServerAllocationsBatchSize,
			Description: "The count of gameserver allocations in a batch",
			Aggregation: view.Distribution(1, 2, 3, 4, 5, 10, 20, 50, 100),
			TagKeys:     []tag.Key{keyFleetName, keyClusterName, keyMultiCluster, keyStatus, keySchedulingStrategy},
		},
	}

	for _, v := range stateViews {
		if err := view.Register(v); err != nil {
			logger.WithError(err).Error("could not register view")
		}
	}
}

// record the current allocation batch size rate.
func (r *metrics) recordAllocationsBatchSize(ctx context.Context, count int) {
	stats.Record(ctx, gameServerAllocationsBatchSize.M(int64(count)))
}

func (r *metrics) recordAllocationUpdateSuccess(ctx context.Context, duration time.Duration) {
	mt.RecordWithTags(ctx, []tag.Mutator{tag.Upsert(keyStatus, "Success")},
		gameServerAllocationsUpdatesLatency.M(duration.Seconds()))
}

func (r *metrics) recordAllocationUpdateFailure(ctx context.Context, duration time.Duration) {
	mt.RecordWithTags(ctx, []tag.Mutator{tag.Upsert(keyStatus, "Failure")},
		gameServerAllocationsUpdatesLatency.M(duration.Seconds()))
}
