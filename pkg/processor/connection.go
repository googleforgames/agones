package processor

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/health/grpc_health_v1"

	allocationpb "agones.dev/agones/pkg/allocation/go"
)

const (
	// maxConnectionAttempts defines the maximum number of connection attempts before giving up
	maxConnectionAttempts = 10
)

// connectAndRun handles the full connection lifecycle to the processor service
// It establishes a connection, creates a stream, registers the client, and then
// delegates to handleStream to process messages until an error or cancellation
func (p *processorClient) connectAndRun(ctx context.Context) error {
	// Connect to the processor with retry logic
	conn, err := p.connectWithRetry(ctx)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer conn.Close()

	// Create a new processor client from the connection
	client := allocationpb.NewProcessorClient(conn)

	// Open a streaming RPC to the processor
	stream, err := client.StreamBatches(ctx)
	if err != nil {
		return fmt.Errorf("failed to create stream: %w", err)
	}

	// Register this client instance with the processor
	if err := p.registerClient(stream); err != nil {
		return fmt.Errorf("failed to register: %w", err)
	}

	p.logger.Info("connected to processor")

	// Handle the stream until an error occurs or the context is cancelled
	return p.handleStream(ctx, stream)
}

// connectWithRetry attempts to connect to the processor service with health checks and retry logic
// It will retry up to maxConnectionAttempts, waiting ReconnectInterval between attempts
// Returns a healthy gRPC connection or an error if all attempts failed
func (p *processorClient) connectWithRetry(ctx context.Context) (*grpc.ClientConn, error) {
	for attempt := 1; attempt <= maxConnectionAttempts; attempt++ {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		p.logger.WithField("attempt", attempt).Debug("attempting connection")

		// Dial the processor with a timeout
		dialCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		conn, err := grpc.DialContext(dialCtx, p.config.ProcessorAddress,
			grpc.WithInsecure(),
			grpc.WithBlock())
		cancel()

		if err != nil {
			p.logger.WithError(err).Warnf("connection attempt %d failed", attempt)
			// Wait before retrying, unless this was the last attempt
			if attempt < maxConnectionAttempts {
				select {
				case <-ctx.Done():
					return nil, ctx.Err()
				case <-time.After(p.config.ReconnectInterval):
				}
			}
			continue
		}

		// Perform a health check on the connection
		if err := p.healthCheck(ctx, conn); err != nil {
			p.logger.WithError(err).Warnf("health check failed on attempt %d", attempt)
			conn.Close()
			// Wait before retrying, unless this was the last attempt
			if attempt < maxConnectionAttempts {
				select {
				case <-ctx.Done():
					return nil, ctx.Err()
				case <-time.After(p.config.ReconnectInterval):
				}
			}
			continue
		}

		p.logger.Info("successfully connected to processor")
		return conn, nil
	}

	return nil, fmt.Errorf("failed to connect after %d attempts", maxConnectionAttempts)
}

// healthCheck verifies that the processor service is healthy and serving requests
// Returns an error if the health check fails or the service is not in SERVING state
func (p *processorClient) healthCheck(ctx context.Context, conn *grpc.ClientConn) error {
	healthClient := grpc_health_v1.NewHealthClient(conn)

	// Set a timeout for the health check RPC
	healthCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	resp, err := healthClient.Check(healthCtx, &grpc_health_v1.HealthCheckRequest{
		Service: "processor",
	})
	if err != nil {
		return err
	}

	if resp.Status != grpc_health_v1.HealthCheckResponse_SERVING {
		return fmt.Errorf("processor not serving: %v", resp.Status)
	}

	return nil
}

// registerClient sends a registration message to the processor over the stream
// This identifies the client instance to the processor service
func (p *processorClient) registerClient(stream allocationpb.Processor_StreamBatchesClient) error {
	p.logger.WithField("clientID", p.config.ClientID).Info("Registering client with processor")

	registerMsg := &allocationpb.ProcessorMessage{
		ClientId: p.config.ClientID,
	}

	// Send the registration message
	err := stream.Send(registerMsg)
	if err != nil {
		p.logger.WithField("clientID", p.config.ClientID).WithError(err).Error("Failed to register client")
		return err
	}

	return nil
}
