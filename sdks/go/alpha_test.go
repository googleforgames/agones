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

package sdk

import (
	"context"
	"testing"

	"agones.dev/agones/pkg/sdk/alpha"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
)

func TestAlphaGetAndSetPlayerCapacity(t *testing.T) {
	mock := &alphaMock{}
	a := Alpha{
		client: mock,
	}

	err := a.SetPlayerCapacity(15)
	assert.NoError(t, err)
	assert.Equal(t, int64(15), mock.capacity)

	capacity, err := a.GetPlayerCapacity()
	assert.NoError(t, err)
	assert.Equal(t, int64(15), capacity)
}

type alphaMock struct {
	capacity int64
}

func (a *alphaMock) PlayerConnect(ctx context.Context, in *alpha.PlayerID, opts ...grpc.CallOption) (*alpha.Bool, error) {
	panic("implement me")
}

func (a *alphaMock) PlayerDisconnect(ctx context.Context, in *alpha.PlayerID, opts ...grpc.CallOption) (*alpha.Bool, error) {
	panic("implement me")
}

func (a *alphaMock) IsPlayerConnected(ctx context.Context, id *alpha.PlayerID, opts ...grpc.CallOption) (*alpha.Bool, error) {
	panic("implement me")
}

func (a *alphaMock) GetConnectedPlayers(ctx context.Context, id *alpha.Empty, opts ...grpc.CallOption) (*alpha.PlayerIDList, error) {
	panic("implement me")
}

func (a *alphaMock) SetPlayerCapacity(ctx context.Context, in *alpha.Count, opts ...grpc.CallOption) (*alpha.Empty, error) {
	a.capacity = in.Count
	return &alpha.Empty{}, nil
}

func (a *alphaMock) GetPlayerCapacity(ctx context.Context, in *alpha.Empty, opts ...grpc.CallOption) (*alpha.Count, error) {
	return &alpha.Count{Count: a.capacity}, nil
}

func (a *alphaMock) GetPlayerCount(ctx context.Context, in *alpha.Empty, opts ...grpc.CallOption) (*alpha.Count, error) {
	panic("implement me")
}
