package processor

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	allocationpb "agones.dev/agones/pkg/allocation/go"
)

// handleStream processes incoming messages from the processor stream
// It listens for pull requests and batch responses, dispatching them to appropriate handlers
func (p *processorClient) handleStream(ctx context.Context, stream allocationpb.Processor_StreamBatchesClient) error {
	p.logger.Info("Starting stream message handling")

	// Channel to handle pull requests asynchronously
	pullRequestChan := make(chan struct{}, 20)

	// Start goroutine to handle pull requests without blocking
	go p.pullRequestHandler(ctx, stream, pullRequestChan)

	for {
		select {
		case <-ctx.Done():
			p.logger.Info("Stream handling stopping due to context cancellation")
			return ctx.Err()
		default:
			// Receive message from processor
			msg, err := stream.Recv()
			if err != nil {
				p.logger.WithError(err).Error("Failed to receive message from processor")
				return fmt.Errorf("stream recv error: %w", err)
			}

			// Handle message based on its payload type
			switch payload := msg.GetPayload().(type) {
			case *allocationpb.ProcessorMessage_Pull:
				// Pull request: queue for async handling
				select {
				case pullRequestChan <- struct{}{}:
					p.logger.Debug("Pull request queued successfully")
				default:
					p.logger.Warn("Pull request queue full - dropping request")
				}

			case *allocationpb.ProcessorMessage_BatchResponse:
				// Batch response: handle immediately
				p.handleBatchResponse(payload.BatchResponse)

			default:
				// Unknown message type
				p.logger.WithField("messageType", fmt.Sprintf("%T", payload)).Warn("Received unknown message type from processor")
			}
		}
	}
}

// pullRequestHandler handles pull requests asynchronously without blocking the main stream
// It waits for pull requests on the channel and processes them as they arrive
func (p *processorClient) pullRequestHandler(ctx context.Context, stream allocationpb.Processor_StreamBatchesClient, pullRequestChan <-chan struct{}) {
	p.logger.Debug("Starting async pull request handler")

	for {
		select {
		case <-ctx.Done():
			p.logger.Debug("Pull request handler stopping")
			return

		case <-pullRequestChan:
			p.handlePullRequest(stream)
		}
	}
}

// handlePullRequest responds to pull requests by sending the current batch of allocation requests
// It swaps out the hot batch, resets it for new requests, and sends the ready batch to the processor
func (p *processorClient) handlePullRequest(stream allocationpb.Processor_StreamBatchesClient) {
	// Swap out the hot batch and pending requests
	p.batchMutex.Lock()
	readyBatch := p.hotBatch
	readyRequests := p.pendingRequests

	// Reset hot batch for next requests
	p.hotBatch = &allocationpb.BatchRequest{
		Requests: make([]*allocationpb.RequestWrapper, 0, p.config.MaxBatchSize),
	}
	p.pendingRequests = make([]*pendingRequest, 0, p.config.MaxBatchSize)
	p.batchMutex.Unlock()

	if len(readyRequests) == 0 {
		p.logger.Debug("No requests to send in batch")
		return
	}

	// Send batch to processor
	p.sendBatch(stream, readyBatch, readyRequests)
}

// sendBatch sends a batch of allocation requests to the processor
func (p *processorClient) sendBatch(stream allocationpb.Processor_StreamBatchesClient, batch *allocationpb.BatchRequest, requests []*pendingRequest) {
	batch.BatchId = uuid.NewString()

	// Prepare batch message
	batchMsg := &allocationpb.ProcessorMessage{
		ClientId: p.config.ClientID,
		Payload: &allocationpb.ProcessorMessage_BatchRequest{
			BatchRequest: batch,
		},
	}

	sendStart := time.Now()
	if err := stream.Send(batchMsg); err != nil {
		p.logger.WithError(err).Error("Failed to send batch")

		// Re-add the request to the hot batch and pendingRequests for the next pull
		for _, req := range requests {
			p.batchMutex.Lock()
			p.hotBatch.Requests = append(p.hotBatch.Requests, &allocationpb.RequestWrapper{
				RequestId: req.id,
				Request:   req.request,
			})
			p.pendingRequests = append(p.pendingRequests, req)
			p.batchMutex.Unlock()
		}
		return
	}

	sendDuration := time.Since(sendStart)
	p.logger.WithFields(logrus.Fields{
		"batchID":      batch.BatchId,
		"requestCount": len(requests),
		"sendDuration": sendDuration,
	}).Debug("Batch sent successfully")
}

// handleBatchResponse processes responses from the processor for a batch of requests
// It matches responses to pending requests, sends results/errors, and cleans up processed requests
func (p *processorClient) handleBatchResponse(batchResp *allocationpb.BatchResponse) {
	p.logger.WithFields(logrus.Fields{
		"component":     "processor-client",
		"batchID":       batchResp.BatchId,
		"responseCount": len(batchResp.Responses),
	}).Info("Processing batch response")

	successCount := 0
	errorCount := 0
	notFoundCount := 0

	for _, respWrapper := range batchResp.Responses {
		requestID := respWrapper.RequestId

		// Try to load the pending request for this response
		if reqInterface, exists := p.requestIDMapping.Load(requestID); exists {
			if req, ok := reqInterface.(*pendingRequest); ok {

				// Track if response was processed successfully
				responseProcessed := false

				switch result := respWrapper.Result.(type) {
				case *allocationpb.ResponseWrapper_Response:
					// Success case: send response to caller
					successCount++
					responseProcessed = true

					select {
					case req.response <- result.Response:
						p.logger.WithField("requestID", requestID).Debug("Response sent successfully")
					default:
						p.logger.WithField("requestID", requestID).Warn("Failed to send response - channel full")
						responseProcessed = false
					}

				case *allocationpb.ResponseWrapper_Error:
					// Error case: send error to caller
					errorCount++
					responseProcessed = true

					code := codes.Code(result.Error.Code)
					msg := result.Error.Message

					p.logger.WithFields(logrus.Fields{
						"component": "processor-client",
						"requestID": requestID,
						"batchID":   batchResp.BatchId,
						"errorCode": code,
						"errorMsg":  msg,
					}).Error("Request failed with error from processor")

					select {
					case req.error <- status.Error(code, msg):
						p.logger.WithField("requestID", requestID).Debug("Error sent successfully")
					default:
						p.logger.WithField("requestID", requestID).Warn("Failed to send error - channel full")
						responseProcessed = false
					}

				default:
					// Missing result: treat as internal error
					errorCount++
					responseProcessed = true

					p.logger.WithFields(logrus.Fields{
						"component": "processor-client",
						"requestID": requestID,
						"batchID":   batchResp.BatchId,
					}).Error("Response wrapper has no result")

					select {
					case req.error <- status.Errorf(codes.Internal, "empty response from processor"):
						p.logger.WithField("requestID", requestID).Debug("Error sent successfully")
					default:
						p.logger.WithField("requestID", requestID).Warn("Failed to send error - channel full")
						responseProcessed = false
					}
				}

				// Only delete if response was processed successfully
				if responseProcessed {
					p.requestIDMapping.Delete(requestID)
					p.logger.WithField("requestID", requestID).Debug("Request cleaned up successfully")
				} else {
					p.logger.WithField("requestID", requestID).Warn("Keeping request in map due to failed processing")
				}

			} else {
				// Failed to cast to pendingRequest
				notFoundCount++
				p.logger.WithFields(logrus.Fields{
					"component": "processor-client",
					"requestID": requestID,
					"batchID":   batchResp.BatchId,
				}).Error("Failed to cast request interface to pendingRequest")
			}
		} else {
			// No pending request found for this response
			notFoundCount++
			p.logger.WithFields(logrus.Fields{
				"component": "processor-client",
				"requestID": requestID,
				"batchID":   batchResp.BatchId,
			}).Warn("No pending request found for response - may have timed out")
		}
	}

	// Log summary of batch response processing
	p.logger.WithFields(logrus.Fields{
		"component":     "processor-client",
		"batchID":       batchResp.BatchId,
		"successCount":  successCount,
		"errorCount":    errorCount,
		"notFoundCount": notFoundCount,
		"totalCount":    len(batchResp.Responses),
	}).Info("Batch response processing completed")
}
