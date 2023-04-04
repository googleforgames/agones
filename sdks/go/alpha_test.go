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
	"github.com/pkg/errors"
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

	playerID := "one"
	ok, err := a.PlayerConnect(playerID)
	assert.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, playerID, mock.playerConnected)

	count, err := a.GetPlayerCount()
	assert.NoError(t, err)
	assert.Equal(t, int64(1), count)

	ok, err = a.PlayerDisconnect(playerID)
	assert.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, playerID, mock.playerDisconnected)

	// Put the player back in.
	ok, err = a.PlayerConnect(playerID)
	assert.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, int64(1), count)

	ok, err = a.IsPlayerConnected(playerID)
	assert.NoError(t, err)
	assert.True(t, ok, "Player should be connected")

	ok, err = a.IsPlayerConnected("false")
	assert.NoError(t, err)
	assert.False(t, ok, "Player should not be connected")

	list, err := a.GetConnectedPlayers()
	assert.NoError(t, err)
	assert.Equal(t, []string{playerID}, list)
}

type alphaMock struct {
	capacity           int64
	playerCount        int64
	playerConnected    string
	playerDisconnected string
}

func (a *alphaMock) PlayerConnect(ctx context.Context, id *alpha.PlayerID, opts ...grpc.CallOption) (*alpha.Bool, error) {
	a.playerConnected = id.PlayerID
	a.playerCount++
	return &alpha.Bool{Bool: true}, nil
}

func (a *alphaMock) PlayerDisconnect(ctx context.Context, id *alpha.PlayerID, opts ...grpc.CallOption) (*alpha.Bool, error) {
	a.playerDisconnected = id.PlayerID
	a.playerCount--
	return &alpha.Bool{Bool: true}, nil
}

func (a *alphaMock) IsPlayerConnected(ctx context.Context, id *alpha.PlayerID, opts ...grpc.CallOption) (*alpha.Bool, error) {
	return &alpha.Bool{Bool: id.PlayerID == a.playerConnected}, nil
}

func (a *alphaMock) GetConnectedPlayers(ctx context.Context, in *alpha.Empty, opts ...grpc.CallOption) (*alpha.PlayerIDList, error) {
	return &alpha.PlayerIDList{List: []string{a.playerConnected}}, nil
}

func (a *alphaMock) SetPlayerCapacity(ctx context.Context, in *alpha.Count, opts ...grpc.CallOption) (*alpha.Empty, error) {
	a.capacity = in.Count
	return &alpha.Empty{}, nil
}

func (a *alphaMock) GetPlayerCapacity(ctx context.Context, in *alpha.Empty, opts ...grpc.CallOption) (*alpha.Count, error) {
	return &alpha.Count{Count: a.capacity}, nil
}

func (a *alphaMock) GetPlayerCount(ctx context.Context, in *alpha.Empty, opts ...grpc.CallOption) (*alpha.Count, error) {
	return &alpha.Count{Count: a.playerCount}, nil
}

// GetCounter to be implemented
func (a *alphaMock) GetCounter(ctx context.Context, in *alpha.GetCounterRequest, opts ...grpc.CallOption) (*alpha.Counter, error) {
	// TODO(#2716): Implement me!
	return nil, errors.Errorf("Unimplemented -- GetCounter coming soon")
}

// UpdateCounter to be implemented
func (a *alphaMock) UpdateCounter(ctx context.Context, in *alpha.UpdateCounterRequest, opts ...grpc.CallOption) (*alpha.Counter, error) {
	// TODO(#2716): Implement me!
	return nil, errors.Errorf("Unimplemented -- UpdateCounter coming soon")
}

// GetList to be implemented
func (a *alphaMock) GetList(ctx context.Context, in *alpha.GetListRequest, opts ...grpc.CallOption) (*alpha.List, error) {
	// TODO(#2716): Implement me!
	return nil, errors.Errorf("Unimplemented -- GetList coming soon")
}

// UpdateList to be implemented
func (a *alphaMock) UpdateList(ctx context.Context, in *alpha.UpdateListRequest, opts ...grpc.CallOption) (*alpha.List, error) {
	// TODO(#2716): Implement me!
	return nil, errors.Errorf("Unimplemented -- UpdateList coming soon")
}

// AddListValue to be implemented
func (a *alphaMock) AddListValue(ctx context.Context, in *alpha.AddListValueRequest, opts ...grpc.CallOption) (*alpha.List, error) {
	// TODO(#2716): Implement me!
	return nil, errors.Errorf("Unimplemented -- AddListValue coming soon")
}

// RemoveListValue to be implemented
func (a *alphaMock) RemoveListValue(ctx context.Context, in *alpha.RemoveListValueRequest, opts ...grpc.CallOption) (*alpha.List, error) {
	// TODO(#2716): Implement me!
	return nil, errors.Errorf("Unimplemented -- RemoveListValue coming soon")
}
