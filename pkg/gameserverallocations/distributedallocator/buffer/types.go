package buffer

import allocationpb "agones.dev/agones/pkg/allocation/go"

// PendingRequest represents a single allocation request with response and error channels.
type PendingRequest struct {
	Req    *allocationpb.AllocationRequest
	RespCh chan *allocationpb.AllocationResponse
	ErrCh  chan error
}

// BatchRequest pairs allocation requests with a response channel.
type BatchRequest struct {
	Requests []*allocationpb.AllocationRequest
	Response chan *allocationpb.BatchResponse
}
