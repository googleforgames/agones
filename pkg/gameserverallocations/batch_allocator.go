package gameserverallocations

import (
	"context"
	goErrors "errors"
	"time"

	"agones.dev/agones/pkg/apis"
	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	allocationv1 "agones.dev/agones/pkg/apis/allocation/v1"
	"agones.dev/agones/pkg/util/runtime"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// batchResponse is an async list of responses for matching requests
type batchResponses struct {
	counterErrors error
	listErrors    error
	responses     []response
}

// batchAllocationUpdateWorkers tries to update each newly allocated gs with the last state. If
// the update fails because of a version conflict, all allocations that were applied onto a gs
// will receive an error, thus being available for retries. If the update succeeds, all allocations
// that were applied onto a gs will succeed, and the gs with the updated state will be added
// back to the cache.
func (c *Allocator) batchAllocationUpdateWorkers(ctx context.Context, workerCount int) chan<- batchResponses {
	metrics := c.newMetrics(ctx)
	batchUpdateQueue := make(chan batchResponses)

	for i := 0; i < workerCount; i++ {
		go func() {
			for {
				select {
				case batchRes := <-batchUpdateQueue:
					if len(batchRes.responses) > 0 {
						// The last response contains the latest gs state
						lastGsState := batchRes.responses[len(batchRes.responses)-1].gs

						requestStartTime := time.Now()

						// Try to update with the latest gs state
						updatedGs, updateErr := c.gameServerGetter.GameServers(lastGsState.ObjectMeta.Namespace).Update(ctx, lastGsState, metav1.UpdateOptions{})
						if updateErr != nil {
							metrics.recordAllocationUpdateFailure(ctx, time.Since(requestStartTime))

							if !k8serrors.IsConflict(errors.Cause(updateErr)) {
								// since we could not allocate, we should put it back
								// but not if it's a conflict, as the cache is no longer up to date, and
								// we should wait for it to get updated with fresh info.
								c.allocationCache.AddGameServer(updatedGs)
							}
							updateErr = errors.Wrap(updateErr, "error updating allocated gameserver")
						} else {
							metrics.recordAllocationUpdateSuccess(ctx, time.Since(requestStartTime))

							// Add the server back as soon as possible and not wait for the informer to update the cache
							c.allocationCache.AddGameServer(updatedGs)

							// If successful Update record any Counter or List action errors as a warning
							if batchRes.counterErrors != nil {
								c.recorder.Event(updatedGs, corev1.EventTypeWarning, "CounterActionError", batchRes.counterErrors.Error())
							}
							if batchRes.listErrors != nil {
								c.recorder.Event(updatedGs, corev1.EventTypeWarning, "ListActionError", batchRes.listErrors.Error())
							}
							c.recorder.Event(updatedGs, corev1.EventTypeNormal, string(updatedGs.Status.State), "Allocated")
						}

						// Forward all responses with their appropriate gs state and update error
						for _, res := range batchRes.responses {
							res.err = updateErr
							res.request.response <- res
						}
					}
				case <-ctx.Done():
					return
				}
			}
		}()
	}

	return batchUpdateQueue
}

// ListenAndBatchAllocate is a blocking function that runs in a loop
// looking at c.pendingRequests for batches of requests that are coming through.
// The difference between this and the original ListenAndAllocate is that this will
// apply the allocation to the local gs (still removing it from the cache) and continue with
// the next allocation from the batch. When the batch is done, the update workers will try to
// update each newly allocated gs with the last state.
func (c *Allocator) ListenAndBatchAllocate(ctx context.Context, updateWorkerCount int) {
	// setup workers for batch allocation updates
	batchUpdateQueue := c.batchAllocationUpdateWorkers(ctx, updateWorkerCount)

	var list []*agonesv1.GameServer
	var sortKey uint64
	requestCount := 0

	metrics := c.newMetrics(ctx)
	batchResponsesPerGs := make(map[string]batchResponses)

	flush := func() {
		if requestCount > 0 {
			metrics.recordAllocationsBatchSize(ctx, requestCount)
		}

		for _, batchResponses := range batchResponsesPerGs {
			batchUpdateQueue <- batchResponses
		}
		batchResponsesPerGs = make(map[string]batchResponses)

		list = nil
		requestCount = 0
	}

	checkSortKey := func(gsa *allocationv1.GameServerAllocation) {
		if runtime.FeatureEnabled(runtime.FeatureCountsAndLists) {
			// SortKey returns the sorting values (list of Priorities) as a determinstic key.
			// In case gsa.Spec.Priorities is nil this will still return a sortKey.
			// In case of error this will return 0 for the sortKey.
			newSortKey, err := gsa.SortKey()
			if err != nil {
				c.baseLogger.WithError(err).Warn("error getting sortKey for GameServerAllocationSpec", err)
			}
			// Set sortKey if this is the first request, or the previous request errored on creating a sortKey.
			if sortKey == uint64(0) {
				sortKey = newSortKey
			}

			if newSortKey != sortKey {
				sortKey = newSortKey
				flush()
			}
		}
	}

	checkRefreshList := func(gsa *allocationv1.GameServerAllocation) {
		// refresh the list after every 100 allocations made in a single batch
		if requestCount >= maxBatchBeforeRefresh {
			flush()
		}
		requestCount++

		checkSortKey(gsa)

		// Sort list if necessary
		if list == nil {
			if !runtime.FeatureEnabled(runtime.FeatureCountsAndLists) || gsa.Spec.Scheduling == apis.Packed {
				list = c.allocationCache.ListSortedGameServers(gsa)
			} else {
				// If FeatureCountsAndLists and Scheduling == Distributed, sort game servers by Priorities
				list = c.allocationCache.ListSortedGameServersPriorities(gsa)
			}
		}
	}

	for {
		select {
		case req := <-c.pendingRequests:
			checkRefreshList(req.gsa)

			gs, index, err := findGameServerForAllocation(req.gsa, list)
			if err != nil {
				req.response <- response{request: req, gs: nil, err: err}
				continue
			}

			// if the gs has not been already allocated in this batch, remove it from the cache,
			// but keep it in the list for the next allocation
			existingBatch, alreadyAllocated := batchResponsesPerGs[string(gs.UID)]
			if !alreadyAllocated {
				if removeErr := c.allocationCache.RemoveGameServer(gs); removeErr != nil {
					// this seems unlikely, but lets handle it just in case
					removeErr = errors.Wrap(removeErr, "error removing gameserver from cache")
					req.response <- response{request: req, gs: nil, err: removeErr}

					// remove the game server because it is problematic
					list = append(list[:index], list[index+1:]...)
					continue
				}
			}

			// apply the allocation to the gs in the list (not in cache anymore)
			applyError, counterErrors, listErrors := c.applyAllocationToLocalGameServer(req.gsa.Spec.MetaPatch, gs, req.gsa)
			if applyError == nil {
				if alreadyAllocated {
					existingBatch.responses = append(existingBatch.responses, response{request: req, gs: gs.DeepCopy(), err: nil})
					existingBatch.counterErrors = goErrors.Join(existingBatch.counterErrors, counterErrors)
					existingBatch.listErrors = goErrors.Join(existingBatch.listErrors, listErrors)
					batchResponsesPerGs[string(gs.UID)] = existingBatch
				} else { // first time we see this gs in this batch
					batchResponsesPerGs[string(gs.UID)] = batchResponses{
						responses:     []response{{request: req, gs: gs.DeepCopy(), err: nil}},
						counterErrors: counterErrors,
						listErrors:    listErrors,
					}
				}
			} else {
				req.response <- response{request: req, gs: nil, err: applyError}
			}
		case <-ctx.Done():
			flush()
			return
		default:
			flush()

			// If nothing is found in c.pendingRequests, we move to
			// default: which will wait for c.batchWaitTime, to allow for some requests to backup in c.pendingRequests,
			// providing us with a batch of Allocation requests in that channel

			// Once we have 1 or more requests in c.pendingRequests (which is buffered to 100), we can start the batch process.
			time.Sleep(c.batchWaitTime)
		}
	}
}

// applyAllocationToLocalGameServer patches the inputted GameServer with the allocation metadata changes, and updates it to the Allocated State.
// Returns the encountered errors.
func (c *Allocator) applyAllocationToLocalGameServer(mp allocationv1.MetaPatch, gs *agonesv1.GameServer, gsa *allocationv1.GameServerAllocation) (error, error, error) {
	// add last allocated, so it always gets updated, even if it is already Allocated
	ts, err := time.Now().MarshalText()
	if err != nil {
		return err, nil, nil
	}
	if gs.ObjectMeta.Annotations == nil {
		gs.ObjectMeta.Annotations = make(map[string]string, 1)
	}
	gs.ObjectMeta.Annotations[LastAllocatedAnnotationKey] = string(ts)
	gs.Status.State = agonesv1.GameServerStateAllocated

	// patch ObjectMeta labels
	if mp.Labels != nil {
		if gs.ObjectMeta.Labels == nil {
			gs.ObjectMeta.Labels = make(map[string]string, len(mp.Labels))
		}
		for key, value := range mp.Labels {
			gs.ObjectMeta.Labels[key] = value
		}
	}

	if gs.ObjectMeta.Annotations == nil {
		gs.ObjectMeta.Annotations = make(map[string]string, len(mp.Annotations))
	}
	// apply annotations patch
	for key, value := range mp.Annotations {
		gs.ObjectMeta.Annotations[key] = value
	}

	// perfom any Counter or List actions
	var counterErrors error
	var listErrors error
	if runtime.FeatureEnabled(runtime.FeatureCountsAndLists) {
		if gsa.Spec.Counters != nil {
			for counter, ca := range gsa.Spec.Counters {
				counterErrors = goErrors.Join(counterErrors, ca.CounterActions(counter, gs))
			}
		}
		if gsa.Spec.Lists != nil {
			for list, la := range gsa.Spec.Lists {
				listErrors = goErrors.Join(listErrors, la.ListActions(list, gs))
			}
		}
	}

	return nil, counterErrors, listErrors
}
