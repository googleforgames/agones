package buffer

import (
	"context"
	"fmt"
	"time"

	allocationpb "agones.dev/agones/pkg/allocation/go"
)

// BatchSourceFromPendingRequests batches incoming PendingRequests into BatchRequests.
func BatchSourceFromPendingRequests(
	ctx context.Context,
	requestBuffer <-chan *PendingRequest,
	batchTimeout time.Duration,
	maxBatchSize int,
) <-chan *BatchRequest {
	out := make(chan *BatchRequest)
	go func() {
		defer close(out)
		for {
			var batch []*allocationpb.AllocationRequest
			var pendingReqs []*PendingRequest

			timer := time.NewTimer(batchTimeout)
			defer timer.Stop()

			// Block for the first request or context cancel
			select {
			case <-ctx.Done():
				return
			case req, ok := <-requestBuffer:
				if !ok {
					return
				}
				batch = append(batch, req.Req)
				pendingReqs = append(pendingReqs, req)
			}

		collectLoop:
			for len(batch) < maxBatchSize {
				select {
				case <-ctx.Done():
					return
				case req, ok := <-requestBuffer:
					if !ok {
						break collectLoop
					}
					batch = append(batch, req.Req)
					pendingReqs = append(pendingReqs, req)
				case <-timer.C:
					break collectLoop
				}
			}

			batchRespCh := make(chan *allocationpb.BatchResponse, 1)
			out <- &BatchRequest{
				Requests: batch,
				Response: batchRespCh,
			}

			// Dispatch batch response to each PendingRequest
			go func(reqs []*PendingRequest, respCh chan *allocationpb.BatchResponse) {
				resp := <-respCh
				if resp == nil || len(resp.Responses) != len(reqs) {
					for _, pr := range reqs {
						pr.ErrCh <- fmt.Errorf("batch response mismatch or nil")
					}
					return
				}
				for i, pr := range reqs {
					pr.RespCh <- resp.Responses[i]
				}
			}(pendingReqs, batchRespCh)
		}
	}()

	return out
}
