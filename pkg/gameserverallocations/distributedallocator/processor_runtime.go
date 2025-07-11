package distributedallocator

import (
	"context"
	"fmt"

	"agones.dev/agones/pkg/gameserverallocations"
)

type Logger interface {
	Info(args ...interface{})
}

type ProcessorRuntime struct {
	GRPCPort  int
	Logger    Logger
	Allocator *gameserverallocations.Allocator
}

// NewProcessorRuntime creates a ProcessorRuntime
func NewProcessorRuntime(grpcPort int, logger Logger, allocator *gameserverallocations.Allocator) *ProcessorRuntime {
	return &ProcessorRuntime{
		GRPCPort:  grpcPort,
		Logger:    logger,
		Allocator: allocator,
	}
}

// Start launches the gRPC server and returns a stop function for cleanup.
func (rt *ProcessorRuntime) Start(ctx context.Context) (stopFunc func(), err error) {
	stopCh := make(chan struct{})

	rt.Logger.Info("[Processor] Starting Allocator...")
	if err := rt.Allocator.Run(ctx); err != nil {
		rt.Logger.Info("[Processor] Failed to start allocator:", err)
	}
	rt.Logger.Info("[Processor] Allocator Running")

	go func() {
		_ = StartProcessorGRPCServer(ctx, fmt.Sprintf("%d", rt.GRPCPort), rt.Allocator)
		close(stopCh)
	}()
	return func() {
		<-stopCh // Wait for gRPC server to exit
	}, nil
}

func (rt *ProcessorRuntime) Stop() {}

// OnStartedLeading is the callback for when this processor becomes the leader.
func (rt *ProcessorRuntime) OnStartedLeading(ctx context.Context) error {
	stopFunc, err := rt.Start(ctx)
	if err != nil {
		return err
	}
	defer stopFunc()

	rt.Logger.Info("[Processor] Became leader, starting gRPC server and batch processing loop...")
	<-ctx.Done() // Block until leadership is lost
	rt.Logger.Info("Processor leader shutting down.")

	return nil
}

func (rt *ProcessorRuntime) OnStoppedLeading() {
	rt.Stop()

	rt.Logger.Info("Lost leadership, performed cleanup.")
}
