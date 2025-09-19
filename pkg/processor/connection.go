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

package processor

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health/grpc_health_v1"

	allocationpb "agones.dev/agones/pkg/allocation/go"
)

// connectAndRun handles the full connection lifecycle to the processor service
// It establishes a connection, creates a stream, registers the client, and then
// delegates to handleStream to process messages until an error or cancellation
func (p *processorClient) connectAndRun(ctx context.Context) error {
	// Connect to the processor
	conn, err := p.connect(ctx)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer func() { _ = conn.Close() }()

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

// connect attempts to connect to the processor service with health checks
// Returns a healthy gRPC connection or an error
func (p *processorClient) connect(ctx context.Context) (*grpc.ClientConn, error) {
	p.logger.Info("attempting connection")

	conn, err := grpc.NewClient(p.config.ProcessorAddress,
		grpc.WithTransportCredentials(insecure.NewCredentials()))

	if err != nil {
		p.logger.WithError(err).Error("connection failed")
		return nil, err
	}

	// Perform a health check on the connection
	if err := p.healthCheck(ctx, conn); err != nil {
		p.logger.WithError(err).Error("health check failed")
		_ = conn.Close()
		return nil, err
	}

	p.logger.Info("successfully connected to processor")
	return conn, nil
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
