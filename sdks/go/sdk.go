// Copyright 2017 Google LLC All Rights Reserved.
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
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"agones.dev/agones/pkg/sdk"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
)

// GameServerCallback is a function definition to be called
// when a GameServer CRD has been changed
type GameServerCallback func(gs *sdk.GameServer)

// SDK is an instance of the Agones SDK
type SDK struct {
	client sdk.SDKClient
	ctx    context.Context
	health sdk.SDK_HealthClient
	alpha  *Alpha
}

// NewSDK starts a new SDK instance, and connects to
// localhost on port 9357. Blocks until connection and handshake are made.
// Times out after 30 seconds.
func NewSDK() (*SDK, error) {
	p := os.Getenv("AGONES_SDK_GRPC_PORT")
	if p == "" {
		p = "9357"
	}
	addr := fmt.Sprintf("localhost:%s", p)
	s := &SDK{
		ctx: context.Background(),
	}
	// block for at least 30 seconds
	ctx, cancel := context.WithTimeout(s.ctx, 30*time.Second)
	defer cancel()
	conn, err := grpc.DialContext(ctx, addr, grpc.WithBlock(), grpc.WithInsecure())
	if err != nil {
		return s, errors.Wrapf(err, "could not connect to %s", addr)
	}
	s.client = sdk.NewSDKClient(conn)
	s.health, err = s.client.Health(s.ctx)
	s.alpha = newAlpha(conn)
	return s, errors.Wrap(err, "could not set up health check")
}

// Alpha returns the Alpha SDK
func (s *SDK) Alpha() *Alpha {
	return s.alpha
}

// Ready marks the Game Server as ready to
// receive connections
func (s *SDK) Ready() error {
	_, err := s.client.Ready(s.ctx, &sdk.Empty{})
	return errors.Wrap(err, "could not send Ready message")
}

// Allocate self marks this gameserver as Allocated.
func (s *SDK) Allocate() error {
	_, err := s.client.Allocate(s.ctx, &sdk.Empty{})
	return errors.Wrap(err, "could not mark self as Allocated")
}

// Shutdown marks the Game Server as ready to
// shutdown
func (s *SDK) Shutdown() error {
	_, err := s.client.Shutdown(s.ctx, &sdk.Empty{})
	return errors.Wrapf(err, "could not send Shutdown message")
}

// Reserve marks the Game Server as Reserved for a given duration, at which point
// it will return the GameServer to a Ready state.
// Do note, the smallest unit available in the time.Duration argument is a second.
func (s *SDK) Reserve(d time.Duration) error {
	_, err := s.client.Reserve(s.ctx, &sdk.Duration{Seconds: int64(d.Seconds())})
	return errors.Wrap(err, "could not send Reserve message")
}

// Health sends a ping to the health
// check to indicate that this server is healthy
func (s *SDK) Health() error {
	return errors.Wrap(s.health.Send(&sdk.Empty{}), "could not send Health ping")
}

// SetLabel sets a metadata label on the `GameServer` with the prefix
// stable.agones.dev/sdk-
func (s *SDK) SetLabel(key, value string) error {
	kv := &sdk.KeyValue{Key: key, Value: value}
	_, err := s.client.SetLabel(s.ctx, kv)
	return errors.Wrap(err, "could not set label")
}

// SetAnnotation sets a metadata annotation on the `GameServer` with the prefix
// stable.agones.dev/sdk-
func (s *SDK) SetAnnotation(key, value string) error {
	kv := &sdk.KeyValue{Key: key, Value: value}
	_, err := s.client.SetAnnotation(s.ctx, kv)
	return errors.Wrap(err, "could not set annotation")
}

// GameServer retrieve the GameServer details
func (s *SDK) GameServer() (*sdk.GameServer, error) {
	gs, err := s.client.GetGameServer(s.ctx, &sdk.Empty{})
	return gs, errors.Wrap(err, "could not retrieve gameserver")
}

// WatchGameServer asynchronously calls the given GameServerCallback with the current GameServer
// configuration when the backing GameServer configuration is updated.
// This function can be called multiple times to add more than one GameServerCallback.
func (s *SDK) WatchGameServer(f GameServerCallback) error {
	stream, err := s.client.WatchGameServer(s.ctx, &sdk.Empty{})
	if err != nil {
		return errors.Wrap(err, "could not watch gameserver")
	}

	go func() {
		for {
			var gs *sdk.GameServer
			gs, err = stream.Recv()
			if err != nil {
				if err == io.EOF {
					_, _ = fmt.Fprintln(os.Stderr, "gameserver event stream EOF received")
					return
				}
				_, _ = fmt.Fprintf(os.Stderr, "error watching GameServer: %s\n", err.Error())
				// This is to wait for the reconnection, and not peg the CPU at 100%
				time.Sleep(time.Second)
				continue
			}
			f(gs)
		}
	}()
	return nil
}
