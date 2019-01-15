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
	"strconv"
	"sync"
	"time"

	stablev1alpha1 "agones.dev/agones/pkg/apis/stable/v1alpha1"
	"agones.dev/agones/pkg/client/clientset/versioned"
	"agones.dev/agones/pkg/client/informers/externalversions"
	listerv1alpha1 "agones.dev/agones/pkg/client/listers/stable/v1alpha1"
	"agones.dev/agones/pkg/util/runtime"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/stats"
	"go.opencensus.io/tag"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

var (
	// MetricResyncPeriod is the interval to re-synchronize metrics based on indexed cache.
	MetricResyncPeriod = time.Second * 1
)

func init() {
	registerViews()
}

// Controller is a metrics controller collecting Agones state metrics
type Controller struct {
	logger           *logrus.Entry
	gameServerLister listerv1alpha1.GameServerLister
	faLister         listerv1alpha1.FleetAllocationLister
	gameServerSynced cache.InformerSynced
	fleetSynced      cache.InformerSynced
	fasSynced        cache.InformerSynced
	faSynced         cache.InformerSynced
	lock             sync.Mutex
	gsCount          GameServerCount
	faCount          map[string]int64
}

// NewController returns a new metrics controller
func NewController(
	kubeClient kubernetes.Interface,
	agonesClient versioned.Interface,
	agonesInformerFactory externalversions.SharedInformerFactory) *Controller {

	gameServer := agonesInformerFactory.Stable().V1alpha1().GameServers()
	gsInformer := gameServer.Informer()

	fa := agonesInformerFactory.Stable().V1alpha1().FleetAllocations()
	faInformer := fa.Informer()
	fleets := agonesInformerFactory.Stable().V1alpha1().Fleets()
	fInformer := fleets.Informer()
	fas := agonesInformerFactory.Stable().V1alpha1().FleetAutoscalers()
	fasInformer := fas.Informer()

	c := &Controller{
		gameServerLister: gameServer.Lister(),
		gameServerSynced: gsInformer.HasSynced,
		faLister:         fa.Lister(),
		fleetSynced:      fInformer.HasSynced,
		fasSynced:        fasInformer.HasSynced,
		faSynced:         faInformer.HasSynced,
		gsCount:          GameServerCount{},
		faCount:          map[string]int64{},
	}

	c.logger = runtime.NewLoggerWithType(c)

	fInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: c.recordFleetChanges,
		UpdateFunc: func(old, new interface{}) {
			c.recordFleetChanges(new)
		},
		DeleteFunc: c.recordFleetDeletion,
	})

	fasInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(added interface{}) {
			c.recordFleetAutoScalerChanges(nil, added)
		},
		UpdateFunc: c.recordFleetAutoScalerChanges,
		DeleteFunc: c.recordFleetAutoScalerDeletion,
	})

	faInformer.AddEventHandlerWithResyncPeriod(cache.ResourceEventHandlerFuncs{
		UpdateFunc: c.recordFleetAllocationChanges,
	}, 0)

	gsInformer.AddEventHandlerWithResyncPeriod(cache.ResourceEventHandlerFuncs{
		UpdateFunc: c.recordGameServerStatusChanges,
	}, 0)

	return c
}

func (c *Controller) recordFleetAutoScalerChanges(old, new interface{}) {

	fas, ok := new.(*stablev1alpha1.FleetAutoscaler)
	if !ok {
		return
	}

	// we looking for fleet name changes if that happens we need to reset
	// metrics for the old fas.
	if old != nil {
		if oldFas, ok := old.(*stablev1alpha1.FleetAutoscaler); ok &&
			oldFas.Spec.FleetName != fas.Spec.FleetName {
			c.recordFleetAutoScalerDeletion(old)
		}
	}

	// fleet autoscaler has been deleted last value should be 0
	if fas.DeletionTimestamp != nil {
		c.recordFleetAutoScalerDeletion(fas)
		return
	}

	ctx, _ := tag.New(context.Background(), tag.Upsert(keyName, fas.Name),
		tag.Upsert(keyFleetName, fas.Spec.FleetName))

	ableToScale := 0
	limited := 0
	if fas.Status.AbleToScale {
		ableToScale = 1
	}
	if fas.Status.ScalingLimited {
		limited = 1
	}
	// recording status
	stats.Record(ctx,
		fasCurrentReplicasStats.M(int64(fas.Status.CurrentReplicas)),
		fasDesiredReplicasStats.M(int64(fas.Status.DesiredReplicas)),
		fasAbleToScaleStats.M(int64(ableToScale)),
		fasLimitedStats.M(int64(limited)))

	// recording buffer policy
	if fas.Spec.Policy.Buffer != nil {
		// recording limits
		recordWithTags(ctx, []tag.Mutator{tag.Upsert(keyType, "max")},
			fasBufferLimitsCountStats.M(int64(fas.Spec.Policy.Buffer.MaxReplicas)))
		recordWithTags(ctx, []tag.Mutator{tag.Upsert(keyType, "min")},
			fasBufferLimitsCountStats.M(int64(fas.Spec.Policy.Buffer.MinReplicas)))

		// recording size
		if fas.Spec.Policy.Buffer.BufferSize.Type == intstr.String {
			// as percentage
			sizeString := fas.Spec.Policy.Buffer.BufferSize.StrVal
			if sizeString != "" {
				if size, err := strconv.Atoi(sizeString[:len(sizeString)-1]); err == nil {
					recordWithTags(ctx, []tag.Mutator{tag.Upsert(keyType, "percentage")},
						fasBufferSizeStats.M(int64(size)))
				}
			}
		} else {
			// as count
			recordWithTags(ctx, []tag.Mutator{tag.Upsert(keyType, "count")},
				fasBufferSizeStats.M(int64(fas.Spec.Policy.Buffer.BufferSize.IntVal)))
		}
	}
}

func (c *Controller) recordFleetAutoScalerDeletion(obj interface{}) {
	fas, ok := obj.(*stablev1alpha1.FleetAutoscaler)
	if !ok {
		return
	}
	ctx, _ := tag.New(context.Background(), tag.Upsert(keyName, fas.Name),
		tag.Upsert(keyFleetName, fas.Spec.FleetName))

	// recording status
	stats.Record(ctx,
		fasCurrentReplicasStats.M(int64(0)),
		fasDesiredReplicasStats.M(int64(0)),
		fasAbleToScaleStats.M(int64(0)),
		fasLimitedStats.M(int64(0)))
}

func (c *Controller) recordFleetChanges(obj interface{}) {
	f, ok := obj.(*stablev1alpha1.Fleet)
	if !ok {
		return
	}

	// fleet has been deleted last value should be 0
	if f.DeletionTimestamp != nil {
		c.recordFleetDeletion(f)
		return
	}

	c.recordFleetReplicas(f.Name, f.Status.Replicas, f.Status.AllocatedReplicas,
		f.Status.ReadyReplicas, f.Spec.Replicas)
}

func (c *Controller) recordFleetDeletion(obj interface{}) {
	f, ok := obj.(*stablev1alpha1.Fleet)
	if !ok {
		return
	}

	c.recordFleetReplicas(f.Name, 0, 0, 0, 0)
}

func (c *Controller) recordFleetReplicas(fleetName string, total, allocated, ready, desired int32) {

	ctx, _ := tag.New(context.Background(), tag.Upsert(keyName, fleetName))

	recordWithTags(ctx, []tag.Mutator{tag.Upsert(keyType, "total")},
		fleetsReplicasCountStats.M(int64(total)))
	recordWithTags(ctx, []tag.Mutator{tag.Upsert(keyType, "allocated")},
		fleetsReplicasCountStats.M(int64(allocated)))
	recordWithTags(ctx, []tag.Mutator{tag.Upsert(keyType, "ready")},
		fleetsReplicasCountStats.M(int64(ready)))
	recordWithTags(ctx, []tag.Mutator{tag.Upsert(keyType, "desired")},
		fleetsReplicasCountStats.M(int64(desired)))
}

// recordGameServerStatusChanged records gameserver status changes, however since it's based
// on cache events some events might collapsed and not appear, for example transition state
// like creating, port allocation, could be skipped.
// This is still very useful for final state, like READY, ERROR and since this is a counter
// (as opposed to gauge) you can aggregate using a rate, let's say how many gameserver are failing
// per second.
// Addition to the cache are not handled, otherwise resync would make metrics inaccurate by doubling
// current gameservers states.
func (c *Controller) recordGameServerStatusChanges(old, new interface{}) {
	newGs, ok := new.(*stablev1alpha1.GameServer)
	if !ok {
		return
	}
	oldGs, ok := old.(*stablev1alpha1.GameServer)
	if !ok {
		return
	}
	if newGs.Status.State != oldGs.Status.State {
		fleetName := newGs.Labels[stablev1alpha1.FleetNameLabel]
		if fleetName == "" {
			fleetName = "none"
		}
		recordWithTags(context.Background(), []tag.Mutator{tag.Upsert(keyType, string(newGs.Status.State)),
			tag.Upsert(keyFleetName, fleetName)}, gameServerTotalStats.M(1))
	}
}

// record fleet allocations total by watching cache changes.
func (c *Controller) recordFleetAllocationChanges(old, new interface{}) {
	newFa, ok := new.(*stablev1alpha1.FleetAllocation)
	if !ok {
		return
	}
	oldFa, ok := old.(*stablev1alpha1.FleetAllocation)
	if !ok {
		return
	}
	// fleet allocations are added without gameserver allocated
	// but then get modified on successful allocation with their gameserver
	if oldFa.Status.GameServer == nil && newFa.Status.GameServer != nil {
		recordWithTags(context.Background(), []tag.Mutator{tag.Upsert(keyFleetName, newFa.Spec.FleetName)},
			fleetAllocationTotalStats.M(1))
	}
}

// Run the Metrics controller. Will block until stop is closed.
// Collect metrics via cache changes and parse the cache periodically to record resource counts.
func (c *Controller) Run(workers int, stop <-chan struct{}) error {
	c.logger.Info("Wait for cache sync")
	if !cache.WaitForCacheSync(stop, c.gameServerSynced, c.fleetSynced,
		c.fasSynced, c.faSynced) {
		return errors.New("failed to wait for caches to sync")
	}
	wait.Until(c.collect, MetricResyncPeriod, stop)
	return nil
}

// collect all metrics that are not event-based.
// this is fired periodically.
func (c *Controller) collect() {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.collectGameServerCounts()
	c.collectFleetAllocationCounts()
}

// collects fleet allocations count by going through our informer cache
func (c *Controller) collectFleetAllocationCounts() {
	//reset fleet allocations count per fleet name
	for fleetName := range c.faCount {
		c.faCount[fleetName] = 0
	}

	fleetAllocations, err := c.faLister.List(labels.Everything())
	if err != nil {
		c.logger.WithError(err).Warn("failed listing fleet allocations")
	}

	for _, fa := range fleetAllocations {
		c.faCount[fa.Spec.FleetName]++
	}

	for fleetName, count := range c.faCount {
		recordWithTags(context.Background(), []tag.Mutator{tag.Insert(keyFleetName, fleetName)},
			fleetAllocationCountStats.M(count))
	}
}

// collects gameservers count by going through our informer cache
// this not meant to be called concurrently
func (c *Controller) collectGameServerCounts() {

	gameservers, err := c.gameServerLister.List(labels.Everything())
	if err != nil {
		c.logger.WithError(err).Warn("failed listing gameservers")
	}

	if err := c.gsCount.record(gameservers); err != nil {
		c.logger.WithError(err).Warn("error while recoding stats")
	}
}
