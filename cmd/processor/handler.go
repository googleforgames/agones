// Copyright 2025 Google LLC All Rights Reserved.
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

// Processor
package main

import (
	"context"
	"fmt"
	"sync"
	"time"

	"agones.dev/agones/pkg/allocation/converters"
	allocationpb "agones.dev/agones/pkg/allocation/go"
	allocationv1 "agones.dev/agones/pkg/apis/allocation/v1"
	"agones.dev/agones/pkg/client/clientset/versioned"
	"agones.dev/agones/pkg/client/informers/externalversions"
	"agones.dev/agones/pkg/gameserverallocations"
	"agones.dev/agones/pkg/gameservers"

	"github.com/heptiolabs/healthcheck"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/plugin/ocgrpc"
	rpcstatus "google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/status"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
)

type allocationResult struct {
	response *allocationpb.AllocationResponse
	error    *rpcstatus.Status
}

type processorHandler struct {
	allocationpb.UnimplementedProcessorServer
	allocator                 *gameserverallocations.Allocator
	mu                        sync.RWMutex
	clients                   map[string]allocationpb.Processor_StreamBatchesServer
	grpcUnallocatedStatusCode codes.Code
	pullInterval              time.Duration
	ctx                       context.Context
	cancel                    context.CancelFunc
}

func newServiceHandler(ctx context.Context, kubeClient kubernetes.Interface, agonesClient versioned.Interface,
	health healthcheck.Handler, config processorConfig, grpcUnallocatedStatusCode codes.Code) *processorHandler {
	defaultResync := 30 * time.Second
	agonesInformerFactory := externalversions.NewSharedInformerFactory(agonesClient, defaultResync)
	kubeInformerFactory := informers.NewSharedInformerFactory(kubeClient, defaultResync)
	gsCounter := gameservers.NewPerNodeCounter(kubeInformerFactory, agonesInformerFactory)

	allocator := gameserverallocations.NewAllocator(
		agonesInformerFactory.Multicluster().V1().GameServerAllocationPolicies(),
		kubeInformerFactory.Core().V1().Secrets(),
		agonesClient.AgonesV1(),
		kubeClient,
		gameserverallocations.NewAllocationCache(agonesInformerFactory.Agones().V1().GameServers(), gsCounter, health),
		config.remoteAllocationTimeout,
		config.totalRemoteAllocationTimeout,
		config.allocationBatchWaitTime)
	batchCtx, cancel := context.WithCancel(ctx)
	h := processorHandler{
		allocator:                 allocator,
		ctx:                       batchCtx,
		cancel:                    cancel,
		grpcUnallocatedStatusCode: grpcUnallocatedStatusCode,
		pullInterval:              config.pullInterval,
	}

	kubeInformerFactory.Start(ctx.Done())
	agonesInformerFactory.Start(ctx.Done())

	if err := allocator.Run(ctx); err != nil {
		logger.WithError(err).Fatal("starting allocator failed.")
	}

	return &h
}

func (h *processorHandler) StreamBatches(stream allocationpb.Processor_StreamBatchesServer) error {
	var clientID string

	// Wait for first message to get clientID
	msg, err := stream.Recv()
	if err != nil {
		logger.WithError(err).Debug("Stream receive error on connect")
		return err
	}
	clientID = msg.GetClientId()
	if clientID == "" {
		logger.Warn("Received empty clientID, closing stream")
		return nil
	}
	h.addClient(clientID, stream)
	defer h.removeClient(clientID)
	logger.WithField("clientID", clientID).Debug("Client registered")

	// Main loop: handle incoming messages
	for {
		msg, err := stream.Recv()
		if err != nil {
			logger.WithError(err).Debug("Stream receive error")
			return err
		}

		payload := msg.GetPayload()
		if payload == nil {
			logger.WithField("clientID", clientID).Warn("Received message with nil payload")
			continue
		}

		if batchPayload, ok := payload.(*allocationpb.ProcessorMessage_BatchRequest); ok {
			batchStart := time.Now()

			batchRequest := batchPayload.BatchRequest
			batchID := batchRequest.GetBatchId()
			requestWrappers := batchRequest.GetRequests()

			logger.WithFields(logrus.Fields{
				"clientID":     clientID,
				"batchID":      batchID,
				"requestCount": len(requestWrappers),
			}).Debug("Received batch request")

			// Extract request IDs for logging
			requestIDs := make([]string, len(requestWrappers))
			for i, wrapper := range requestWrappers {
				requestIDs[i] = wrapper.GetRequestId()
			}

			response := h.submitBatch(batchID, requestWrappers)

			processingTime := time.Since(batchStart)
			avgPerRequest := time.Duration(0)
			if len(requestWrappers) > 0 {
				avgPerRequest = processingTime / time.Duration(len(requestWrappers))
			}

			logger.WithFields(logrus.Fields{
				"clientID":       clientID,
				"batchID":        batchID,
				"batchSize":      len(requestWrappers),
				"processingTime": processingTime,
				"avgPerRequest":  avgPerRequest,
			}).Info("Batch processing completed")

			// Count successful and failed responses for logging
			successCount := 0
			errorCount := 0
			for _, respWrapper := range response.Responses {
				switch respWrapper.Result.(type) {
				case *allocationpb.ResponseWrapper_Response:
					successCount++
				case *allocationpb.ResponseWrapper_Error:
					errorCount++
				}
			}

			respMsg := &allocationpb.ProcessorMessage{
				ClientId: clientID,
				Payload: &allocationpb.ProcessorMessage_BatchResponse{
					BatchResponse: response,
				},
			}

			if err := stream.Send(respMsg); err != nil {
				logger.WithFields(logrus.Fields{
					"clientID": clientID,
					"batchID":  batchID,
				}).WithError(err).Error("Failed to send response")
				return err
			}
		}
	}
}

// Start the pullRequest ticker
func (h *processorHandler) startPullRequestTicker() {
	go func() {
		ticker := time.NewTicker(h.pullInterval)
		defer ticker.Stop()

		for {
			select {
			case <-h.ctx.Done():
				return
			case <-ticker.C:
				h.mu.RLock()
				for clientID, stream := range h.clients {
					pullMsg := &allocationpb.ProcessorMessage{
						ClientId: clientID,
						Payload: &allocationpb.ProcessorMessage_Pull{
							Pull: &allocationpb.PullRequest{Message: "pull"},
						},
					}
					go func(id string, s allocationpb.Processor_StreamBatchesServer) {
						if err := s.Send(pullMsg); err != nil {
							logger.WithFields(logrus.Fields{
								"clientID": id,
								"error":    err,
							}).Warn("Failed to send pull request, removing client")
							h.removeClient(id)
						}
					}(clientID, stream)
				}
				h.mu.RUnlock()
			}
		}
	}()
}

func (h *processorHandler) processAllocationsConcurrently(requestWrappers []*allocationpb.RequestWrapper) []allocationResult {
	results := make([]allocationResult, len(requestWrappers))
	var wg sync.WaitGroup

	for i, reqWrapper := range requestWrappers {
		wg.Add(1)
		go func(index int, requestWrapper *allocationpb.RequestWrapper) {
			defer wg.Done()
			results[index] = h.processAllocation(requestWrapper.Request)
		}(i, reqWrapper)
	}

	wg.Wait()

	return results
}

func (h *processorHandler) processAllocation(req *allocationpb.AllocationRequest) allocationResult {
	gsa := converters.ConvertAllocationRequestToGSA(req)
	gsa.ApplyDefaults()

	resultObj, err := h.allocator.Allocate(h.ctx, gsa)
	if err != nil {
		return allocationResult{
			error: &rpcstatus.Status{
				Code:    int32(codes.Internal),
				Message: fmt.Sprintf("Allocator error: %v", err),
			},
		}
	}

	if s, ok := resultObj.(*metav1.Status); ok {
		return allocationResult{
			error: &rpcstatus.Status{
				Code:    int32(grpcCodeFromHTTPStatus(int(s.Code))),
				Message: s.Message,
			},
		}
	}

	allocatedGsa, ok := resultObj.(*allocationv1.GameServerAllocation)
	if !ok {
		msg := fmt.Sprintf("internal server error - Bad GSA format %v", resultObj)
		return allocationResult{
			error: &rpcstatus.Status{
				Code:    int32(codes.Internal),
				Message: msg,
			},
		}
	}

	response, err := converters.ConvertGSAToAllocationResponse(allocatedGsa, h.grpcUnallocatedStatusCode)
	if err != nil {
		grpcStatus, ok := status.FromError(err)
		code := h.grpcUnallocatedStatusCode
		msg := err.Error()
		if ok {
			code = grpcStatus.Code()
			msg = grpcStatus.Message()
		}
		return allocationResult{
			error: &rpcstatus.Status{
				Code:    int32(code),
				Message: msg,
			},
		}
	}
	return allocationResult{response: response}
}

func (h *processorHandler) submitBatch(batchID string, requestWrappers []*allocationpb.RequestWrapper) *allocationpb.BatchResponse {
	results := h.processAllocationsConcurrently(requestWrappers)
	responseWrappers := make([]*allocationpb.ResponseWrapper, len(requestWrappers))

	for i, result := range results {
		wrapper := &allocationpb.ResponseWrapper{
			RequestId: requestWrappers[i].RequestId,
		}

		if result.error != nil {
			wrapper.Result = &allocationpb.ResponseWrapper_Error{
				Error: result.error,
			}
		} else {
			wrapper.Result = &allocationpb.ResponseWrapper_Response{
				Response: result.response,
			}
		}
		responseWrappers[i] = wrapper
	}

	return &allocationpb.BatchResponse{
		BatchId:   batchID,
		Responses: responseWrappers,
	}
}

// getGRPCServerOptions returns a list of GRPC server options to use when
// only serving gRPC requests.
// Current options are TLS certs and opencensus stats handler.
func (h *processorHandler) getGRPCServerOptions() []grpc.ServerOption {
	opts := []grpc.ServerOption{
		grpc.StatsHandler(&ocgrpc.ServerHandler{}),

		// Optimized for minikube networking
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
			MinTime:             10 * time.Second,
			PermitWithoutStream: true,
		}),
		grpc.KeepaliveParams(keepalive.ServerParameters{
			MaxConnectionIdle: 1 * time.Minute,
			Timeout:           30 * time.Second,
			Time:              30 * time.Second,
		}),

		// Buffer sizes optimized for batch processing
		grpc.WriteBufferSize(64 * 1024),
		grpc.ReadBufferSize(64 * 1024),
		grpc.InitialWindowSize(128 * 1024),
		grpc.InitialConnWindowSize(128 * 1024),

		// Message size limits
		grpc.MaxRecvMsgSize(4 * 1024 * 1024),
		grpc.MaxSendMsgSize(4 * 1024 * 1024),
	}

	return opts
}

// Add these methods for client management
func (h *processorHandler) addClient(clientID string, stream allocationpb.Processor_StreamBatchesServer) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.clients == nil {
		h.clients = make(map[string]allocationpb.Processor_StreamBatchesServer)
	}
	h.clients[clientID] = stream
}

func (h *processorHandler) removeClient(clientID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.clients, clientID)
}
