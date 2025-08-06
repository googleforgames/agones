package processor

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	allocationpb "agones.dev/agones/pkg/allocation/go"
)

// ProcessorClient interface for allocation operations
// Provides methods to run the processor client and perform allocation requests
type ProcessorClient interface {
	// Run starts the processor client
	Run(ctx context.Context) error

	// Allocate performs a batch allocation request
	Allocate(ctx context.Context, req *allocationpb.AllocationRequest) (*allocationpb.AllocationResponse, error)
}

// processorClient implements ProcessorClient interface
type processorClient struct {
	config Config
	logger logrus.FieldLogger

	// requestIDMapping is a map to correlate request IDs to pendingRequest objects for response handling
	requestIDMapping sync.Map

	// hotBatch holds the current batch of allocation requests that have been converted to protobuf format
	// It accumulates requests until a pull request is received to be directly sent to the processor
	// After sending, hotBatch is reset and starts collecting new requests for the next batch
	hotBatch *allocationpb.BatchRequest

	// pendingRequests holds the list of allocation requests currently in the batch
	// Each pendingRequest tracks the original request, its unique ID, and channels for response and error
	// This slice is used to correlate responses from the processor back to the original caller
	pendingRequests []*pendingRequest

	batchMutex sync.RWMutex
}

// NewProcessorClient creates a new processor client
func NewProcessorClient(config Config, logger logrus.FieldLogger) ProcessorClient {
	if len(config.ClientID) == 0 {
		config.ClientID = uuid.New().String()
	}

	return &processorClient{
		config: config,
		logger: logger,
		hotBatch: &allocationpb.BatchRequest{
			Requests: make([]*allocationpb.RequestWrapper, 0, config.MaxBatchSize),
		},
		pendingRequests: make([]*pendingRequest, 0, config.MaxBatchSize),
	}
}

// Run starts the processor client and manages the connection lifecycle
// It will retry connecting to the processor service until the context is cancelled
func (p *processorClient) Run(ctx context.Context) error {
	p.logger.Info("starting processor client")

	// Main connection loop with retry
	for {
		select {
		case <-ctx.Done():
			p.logger.Info("processor client stopping")
			return ctx.Err()
		default:
			if err := p.connectAndRun(ctx); err != nil {
				if ctx.Err() != nil {
					return ctx.Err()
				}
				p.logger.WithError(err).Error("connection failed, retrying")

				// Wait before retrying connection
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(p.config.ReconnectInterval):
				}
			}
		}
	}
}

// Allocate performs an allocation request by batching it and waiting for a response or error
func (p *processorClient) Allocate(ctx context.Context, req *allocationpb.AllocationRequest) (*allocationpb.AllocationResponse, error) {
	requestID := generateRequestID()

	// Create a pendingRequest to track this allocation request and its response/error
	pendingReq := &pendingRequest{
		id:       requestID,
		request:  req,
		response: make(chan *allocationpb.AllocationResponse, 1),
		error:    make(chan error, 1),
	}
	p.requestIDMapping.Store(requestID, pendingReq)

	// Wrap the request for batching.
	wrapper := &allocationpb.RequestWrapper{
		RequestId: requestID,
		Request:   req,
	}

	// Safely add the request to the current batch and pending list using a scoped lock
	batchAdded := false
	func() {
		p.batchMutex.Lock()
		defer p.batchMutex.Unlock()

		p.hotBatch.Requests = append(p.hotBatch.Requests, wrapper)
		p.pendingRequests = append(p.pendingRequests, pendingReq)
		batchAdded = true
	}()

	if !batchAdded {
		p.requestIDMapping.Delete(requestID)
		return nil, status.Errorf(codes.Internal, "failed to add request to batch")
	}

	// Wait for response, error, cancellation, or timeout
	timeout := p.config.AllocationTimeout

	select {
	case response := <-pendingReq.response:
		p.logger.WithField("requestID", requestID).Debug("Received successful response")
		return response, nil

	case err := <-pendingReq.error:
		p.logger.WithField("requestID", requestID).WithError(err).Debug("Received error response")
		return nil, err

	case <-ctx.Done():
		p.requestIDMapping.Delete(requestID)
		p.logger.WithField("requestID", requestID).Debug("Request cancelled by context")
		return nil, ctx.Err()

	case <-time.After(timeout):
		p.requestIDMapping.Delete(requestID)
		p.logger.WithField("requestID", requestID).Error("Timeout waiting for processor response")
		return nil, status.Errorf(codes.DeadlineExceeded, "allocation timeout after %v", timeout)
	}
}

// generateRequestID generates a unique request ID
func generateRequestID() string {
	return fmt.Sprintf("%d-%d", time.Now().UnixNano(), rand.Int63())
}
