package processor

import (
	allocationpb "agones.dev/agones/pkg/allocation/go"
)

// pendingRequest represents a request waiting for processing.
type pendingRequest struct {
	// id is the unique identifier for this request
	id string

	// request is the original allocation request data
	request *allocationpb.AllocationRequest

	// response is the channel to receive the allocation response
	response chan *allocationpb.AllocationResponse

	// error is the channel to receive an error if processing fails
	error chan error
}
