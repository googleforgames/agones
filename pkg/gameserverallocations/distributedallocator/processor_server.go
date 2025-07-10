package distributedallocator

import (
	"context"
	"log"
	"net"
	"sync"
	"time"

	"google.golang.org/grpc"

	"agones.dev/agones/pkg/allocation/converters"
	allocationpb "agones.dev/agones/pkg/allocation/go"
	allocationv1 "agones.dev/agones/pkg/apis/allocation/v1"
	"agones.dev/agones/pkg/gameserverallocations"
)

// processorServer implements the Processor gRPC service.
type processorServer struct {
	allocationpb.UnimplementedProcessorServer
	batchAllocator *gameserverallocations.Allocator
	mu             sync.Mutex
}

// StreamBatches handles streaming batch allocation requests.
func (s *processorServer) StreamBatches(stream allocationpb.Processor_StreamBatchesServer) error {
	var clientID string
	pullTicker := time.NewTicker(200 * time.Millisecond)
	defer pullTicker.Stop()
	done := make(chan struct{})

	// Goroutine: send PullRequest every 200ms
	go func() {
		for {
			select {
			case <-pullTicker.C:
				pullMsg := &allocationpb.ProcessorMessage{
					ClientId: clientID,
					Payload: &allocationpb.ProcessorMessage_Pull{
						Pull: &allocationpb.PullRequest{Message: "pull"},
					},
				}
				_ = stream.Send(pullMsg)
			case <-done:
				return
			}
		}
	}()

	for {
		msg, err := stream.Recv()
		if err != nil {
			log.Printf("[Processor] Error receiving from stream: %v", err)
			close(done)
			return err
		}
		log.Printf("[Processor] Received message from allocator: clientID=%s, payloadType=%T", msg.GetClientId(), msg.GetPayload())

		if clientID == "" {
			clientID = msg.GetClientId()
			log.Printf("[Processor] Set clientID to %s", clientID)
			if clientID == "" {
				log.Printf("[Processor] Received empty clientID, closing stream")
				close(done)
				return nil
			}
		}

		switch payload := msg.GetPayload().(type) {
		case *allocationpb.ProcessorMessage_Batch:
			batch := payload.Batch
			log.Printf("[Processor] Received BatchRequest with %d requests from clientID=%s", len(batch.Requests), clientID)
			responses := make([]*allocationpb.AllocationResponse, len(batch.Requests))
			errors := make([]string, len(batch.Requests))
			for i, req := range batch.Requests {
				log.Printf("[Processor] Processing allocation request %d/%d for clientID=%s", i+1, len(batch.Requests), clientID)
				gsa := converters.ConvertAllocationRequestToGSA(req)
				gsa.ApplyDefaults()

				defer func() {
					if r := recover(); r != nil {
						log.Printf("[Processor] Panic in Allocate for request %d: %v", i+1, r)
					}
				}()

				log.Printf("[Processor] Calling Allocate for request %d", i+1)
				resultObj, err := s.batchAllocator.Allocate(stream.Context(), gsa)
				log.Printf("[Processor] Allocate returned for request %d (err=%v)", i+1, err)

				if err != nil {
					log.Printf("[Processor] Allocation error for request %d: %v", i+1, err)
					errors[i] = err.Error()
					continue
				}
				resp, _ := converters.ConvertGSAToAllocationResponse(resultObj.(*allocationv1.GameServerAllocation), 0)
				responses[i] = resp
			}
			respMsg := &allocationpb.ProcessorMessage{
				ClientId: clientID,
				Payload: &allocationpb.ProcessorMessage_BatchResponse{
					BatchResponse: &allocationpb.BatchResponse{
						Responses: responses,
						Errors:    errors,
					},
				},
			}
			log.Printf("[Processor] Sending BatchResponse to clientID=%s", clientID)
			if err := stream.Send(respMsg); err != nil {
				log.Printf("[Processor] Error sending BatchResponse: %v", err)
				close(done)
				return err
			}
		}
	}
}

// StartProcessorGRPCServer starts the processor gRPC server.
func StartProcessorGRPCServer(ctx context.Context, grpcPort string, batchAllocator *gameserverallocations.Allocator) error {
	log.Printf("[Processor] Starting gRPC server on :%s", grpcPort)

	ln, err := net.Listen("tcp", ":"+grpcPort)
	if err != nil {
		return err
	}

	s := grpc.NewServer()
	allocationpb.RegisterProcessorServer(s, &processorServer{
		batchAllocator: batchAllocator,
	})

	go func() {
		<-ctx.Done()
		log.Println("[Processor] Shutting down gRPC server...")
		s.GracefulStop()
		ln.Close()
	}()

	return s.Serve(ln)
}
