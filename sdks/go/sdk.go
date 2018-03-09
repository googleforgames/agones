// Copyright 2017 Google Inc. All Rights Reserved.
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

// Package sdk is the Go game server sdk
package sdk

import (
	"fmt"
	"time"

	"agones.dev/agones/pkg/sdk"
	"github.com/pkg/errors"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

const port = 59357

// SDK is an instance of the Agones SDK
type SDK struct {
	client sdk.SDKClient
	ctx    context.Context
	health sdk.SDK_HealthClient
}

// NewSDK starts a new SDK instance, and connects to
// localhost on port 59357. Blocks until connection and handshake are made.
// Times out after 30 seconds.
func NewSDK() (*SDK, error) {
	addr := fmt.Sprintf("localhost:%d", port)
	s := &SDK{ctx: context.Background()}
	// block for at least 30 seconds
	ctx, cancel := context.WithTimeout(s.ctx, 30*time.Second)
	defer cancel()
	conn, err := grpc.DialContext(ctx, addr, grpc.WithBlock(), grpc.WithInsecure())
	if err != nil {
		return s, errors.Wrapf(err, "could not connect to %s", addr)
	}
	s.client = sdk.NewSDKClient(conn)
	s.health, err = s.client.Health(s.ctx)
	return s, errors.Wrap(err, "could not set up health check")
}

// Ready marks the Game Server as ready to
// receive connections
func (s *SDK) Ready() error {
	_, err := s.client.Ready(s.ctx, &sdk.Empty{})
	return errors.Wrap(err, "could not send Ready message")
}

// Shutdown marks the Game Server as ready to
// shutdown
func (s *SDK) Shutdown() error {
	_, err := s.client.Shutdown(s.ctx, &sdk.Empty{})
	return errors.Wrapf(err, "could not send Shutdown message")
}

// Health sends a ping to the health
// check to indicate that this server is healthy
func (s *SDK) Health() error {
	return errors.Wrap(s.health.Send(&sdk.Empty{}), "could not send Health ping")
}
