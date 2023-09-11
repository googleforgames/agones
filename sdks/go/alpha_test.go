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

func TestAlphaGetAndUpdateCounter(t *testing.T) {
	mock := &alphaMock{}
	// Counters must be predefined in the GameServer resource on creation.
	mock.counters = make(map[string]*alpha.Counter)
	sessions := alpha.Counter{
		Name:     "sessions",
		Count:    21,
		Capacity: 42,
	}
	games := alpha.Counter{
		Name:     "games",
		Count:    12,
		Capacity: 24,
	}
	gamers := alpha.Counter{
		Name:     "gamers",
		Count:    263,
		Capacity: 500,
	}
	mock.counters["sessions"] = &alpha.Counter{
		Name:     "sessions",
		Count:    21,
		Capacity: 42,
	}
	mock.counters["games"] = &alpha.Counter{
		Name:     "games",
		Count:    12,
		Capacity: 24,
	}
	mock.counters["gamers"] = &alpha.Counter{
		Name:     "gamers",
		Count:    263,
		Capacity: 500,
	}
	a := Alpha{
		client: mock,
	}

	t.Parallel()

	t.Run("Set Counter and Set Capacity", func(t *testing.T) {
		counter, err := a.GetCounter("sessions")
		assert.NoError(t, err)
		assert.Equal(t, &sessions, &counter)

		wantCapacity := int64(25)
		ok, err := a.SetCounterCapacity("sessions", wantCapacity)
		assert.NoError(t, err)
		assert.True(t, ok)

		capacity, err := a.GetCounterCapacity("sessions")
		assert.NoError(t, err)
		assert.Equal(t, wantCapacity, capacity)

		wantCount := int64(10)
		ok, err = a.SetCounterCount("sessions", wantCount)
		assert.NoError(t, err)
		assert.True(t, ok)

		count, err := a.GetCounterCount("sessions")
		assert.NoError(t, err)
		assert.Equal(t, wantCount, count)
	})

	t.Run("Set Counter and Set Capacity Non Defined Counter", func(t *testing.T) {
		counter, err := a.GetCounter("secessions")
		assert.Error(t, err)
		assert.Empty(t, &counter)

		ok, err := a.SetCounterCapacity("secessions", int64(100))
		assert.Error(t, err)
		assert.False(t, ok)

		ok, err = a.SetCounterCount("secessions", int64(0))
		assert.Error(t, err)
		assert.False(t, ok)
	})

	// nolint:dupl // testing DecrementCounter and IncrementCounter are not duplicates.
	t.Run("Decrement Counter Fails then Success", func(t *testing.T) {
		counter, err := a.GetCounter("games")
		assert.NoError(t, err)
		assert.Equal(t, &games, &counter)

		ok, err := a.DecrementCounter("games", 21)
		assert.Error(t, err)
		assert.False(t, ok)

		count, err := a.GetCounterCount("games")
		assert.NoError(t, err)
		assert.Equal(t, games.Count, count)

		ok, err = a.DecrementCounter("games", -12)
		assert.Error(t, err)
		assert.False(t, ok)

		count, err = a.GetCounterCount("games")
		assert.NoError(t, err)
		assert.Equal(t, games.Count, count)

		ok, err = a.DecrementCounter("games", 12)
		assert.NoError(t, err)
		assert.True(t, ok)

		count, err = a.GetCounterCount("games")
		assert.NoError(t, err)
		assert.Equal(t, int64(0), count)
	})

	// nolint:dupl // testing DecrementCounter and IncrementCounter are not duplicates.
	t.Run("Increment Counter Fails then Success", func(t *testing.T) {
		counter, err := a.GetCounter("gamers")
		assert.NoError(t, err)
		assert.Equal(t, &gamers, &counter)

		ok, err := a.IncrementCounter("gamers", 250)
		assert.Error(t, err)
		assert.False(t, ok)

		count, err := a.GetCounterCount("gamers")
		assert.NoError(t, err)
		assert.Equal(t, gamers.Count, count)

		ok, err = a.IncrementCounter("gamers", -237)
		assert.Error(t, err)
		assert.False(t, ok)

		count, err = a.GetCounterCount("gamers")
		assert.NoError(t, err)
		assert.Equal(t, gamers.Count, count)

		ok, err = a.IncrementCounter("gamers", 237)
		assert.NoError(t, err)
		assert.True(t, ok)

		count, err = a.GetCounterCount("gamers")
		assert.NoError(t, err)
		assert.Equal(t, int64(500), count)
	})

}

type alphaMock struct {
	capacity           int64
	playerCount        int64
	playerConnected    string
	playerDisconnected string
	counters           map[string]*alpha.Counter
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

func (a *alphaMock) GetCounter(ctx context.Context, in *alpha.GetCounterRequest, opts ...grpc.CallOption) (*alpha.Counter, error) {
	if counter, ok := a.counters[in.Name]; ok {
		return counter, nil
	}
	return nil, errors.Errorf("NOT_FOUND. %s Counter not found", in.Name)
}

func (a *alphaMock) UpdateCounter(ctx context.Context, in *alpha.UpdateCounterRequest, opts ...grpc.CallOption) (*alpha.Counter, error) {
	counter, err := a.GetCounter(ctx, &alpha.GetCounterRequest{Name: in.CounterUpdateRequest.Name})
	if err != nil {
		return nil, err
	}

	switch {
	case in.CounterUpdateRequest.CountDiff != 0:
		count := counter.Count + in.CounterUpdateRequest.CountDiff
		if count < 0 || count > counter.Capacity {
			return nil, errors.Errorf("OUT_OF_RANGE. Count must be within range [0,Capacity]. Found Count: %d, Capacity: %d", count, counter.Capacity)
		}
		counter.Count = count
	case in.CounterUpdateRequest.Count != nil:
		countSet := in.CounterUpdateRequest.Count.GetValue()
		if countSet < 0 || countSet > counter.Capacity {
			return nil, errors.Errorf("OUT_OF_RANGE. Count must be within range [0,Capacity]. Found Count: %d, Capacity: %d", countSet, counter.Capacity)
		}
		counter.Count = countSet
	case in.CounterUpdateRequest.Capacity != nil:
		capacity := in.CounterUpdateRequest.Capacity.GetValue()
		if capacity < 0 {
			return nil, errors.Errorf("OUT_OF_RANGE. Capacity must be greater than or equal to 0. Found Capacity: %d", capacity)
		}
		counter.Capacity = capacity
	default:
		return nil, errors.Errorf("INVALID_ARGUMENT. Malformed CounterUpdateRequest: %v",
			in.CounterUpdateRequest)
	}

	a.counters[in.CounterUpdateRequest.Name] = counter
	return a.counters[in.CounterUpdateRequest.Name], nil
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
