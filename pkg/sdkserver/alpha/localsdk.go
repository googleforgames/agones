// Copyright 2020 Google LLC All Rights Reserved.
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

package alpha

import (
	"context"

	"agones.dev/agones/pkg/sdk/alpha"
)

var _ alpha.SDKServer = LocalSDKServer{}

// LocalSDKServer is the local sdk server implementation
// for alpha features.
type LocalSDKServer struct{}

// NewLocalSDKServer is a constructor for the alpha local SDK Server.
func NewLocalSDKServer() *LocalSDKServer {
	return &LocalSDKServer{}
}

// Close tears down all the things.
func (l *LocalSDKServer) Close() {
	// placeholder in case things need to be shut down
}

// PlayerConnect should be called when a player connects.
func (l LocalSDKServer) PlayerConnect(ctx context.Context, id *alpha.PlayerId) (*alpha.Empty, error) {
	panic("implement me")
}

// PlayerDisconnect should be called when a player disconnects.
func (l LocalSDKServer) PlayerDisconnect(ctx context.Context, id *alpha.PlayerId) (*alpha.Empty, error) {
	panic("implement me")
}

// SetPlayerCapacity to change the game server's player capacity.
func (l LocalSDKServer) SetPlayerCapacity(ctx context.Context, count *alpha.Count) (*alpha.Empty, error) {
	panic("implement me")
}

// GetPlayerCapacity returns the current player capacity.
func (l LocalSDKServer) GetPlayerCapacity(ctx context.Context, _ *alpha.Empty) (*alpha.Count, error) {
	panic("implement me")
}

// GetPlayerCount returns the current player count.
func (l LocalSDKServer) GetPlayerCount(ctx context.Context, _ *alpha.Empty) (*alpha.Count, error) {
	panic("implement me")
}
