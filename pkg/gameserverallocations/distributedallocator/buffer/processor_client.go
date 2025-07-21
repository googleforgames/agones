package buffer

import (
	"context"

	allocationpb "agones.dev/agones/pkg/allocation/go"
	"google.golang.org/grpc"
)

// PullAndDispatchBatches connects to a remote processor and streams batches.
func PullAndDispatchBatches(
	ctx context.Context,
	processorAddr string,
	clientID string,
	batchSource <-chan *BatchRequest,
) error {
	conn, err := grpc.Dial(processorAddr, grpc.WithInsecure())
	if err != nil {
		return err
	}
	defer conn.Close()

	client := allocationpb.NewProcessorClient(conn)
	stream, err := client.StreamBatches(ctx)
	if err != nil {
		return err
	}

	// Send only the client_id as the first message (no payload)
	initMsg := &allocationpb.ProcessorMessage{
		ClientId: clientID,
	}
	if err := stream.Send(initMsg); err != nil {
		return err
	}

	for {
		resp, err := stream.Recv()
		if err != nil {
			return err
		}

		switch resp.Payload.(type) {
		case *allocationpb.ProcessorMessage_Pull:
			var batchReq *BatchRequest
			select {
			case batchReq = <-batchSource:
			default:
				batchReq = nil
			}
			if batchReq != nil && len(batchReq.Requests) > 0 {
				batchMsg := &allocationpb.ProcessorMessage{
					ClientId: clientID,
					Payload: &allocationpb.ProcessorMessage_Batch{
						Batch: &allocationpb.Batch{Requests: batchReq.Requests},
					},
				}
				if err := stream.Send(batchMsg); err != nil {
					return err
				}
				// Wait for BatchResponse before continuing
				resp, err := stream.Recv()
				if err != nil {
					return err
				}
				if batchResp, ok := resp.Payload.(*allocationpb.ProcessorMessage_BatchResponse); ok {
					batchReq.Response <- batchResp.BatchResponse
				}
			}
		case *allocationpb.ProcessorMessage_BatchResponse:
			// Handle unexpected BatchResponse if needed
		default:
			// Handle other message types if needed
		}
	}
}
