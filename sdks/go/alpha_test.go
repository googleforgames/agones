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
		count, err := a.GetCounterCount("sessions")
		assert.NoError(t, err)
		assert.Equal(t, sessions.Count, count)

		capacity, err := a.GetCounterCapacity("sessions")
		assert.NoError(t, err)
		assert.Equal(t, sessions.Capacity, capacity)

		wantCapacity := int64(25)
		ok, err := a.SetCounterCapacity("sessions", wantCapacity)
		assert.NoError(t, err)
		assert.True(t, ok)

		capacity, err = a.GetCounterCapacity("sessions")
		assert.NoError(t, err)
		assert.Equal(t, wantCapacity, capacity)

		wantCount := int64(10)
		ok, err = a.SetCounterCount("sessions", wantCount)
		assert.NoError(t, err)
		assert.True(t, ok)

		count, err = a.GetCounterCount("sessions")
		assert.NoError(t, err)
		assert.Equal(t, wantCount, count)
	})

	t.Run("Get and Set Non-Defined Counter", func(t *testing.T) {
		_, err := a.GetCounterCount("secessions")
		assert.Error(t, err)

		_, err = a.GetCounterCapacity("secessions")
		assert.Error(t, err)

		ok, err := a.SetCounterCapacity("secessions", int64(100))
		assert.Error(t, err)
		assert.False(t, ok)

		ok, err = a.SetCounterCount("secessions", int64(0))
		assert.Error(t, err)
		assert.False(t, ok)
	})

	// nolint:dupl // testing DecrementCounter and IncrementCounter are not duplicates.
	t.Run("Decrement Counter Fails then Success", func(t *testing.T) {
		count, err := a.GetCounterCount("games")
		assert.NoError(t, err)
		assert.Equal(t, games.Count, count)

		ok, err := a.DecrementCounter("games", 21)
		assert.Error(t, err)
		assert.False(t, ok)

		count, err = a.GetCounterCount("games")
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
		count, err := a.GetCounterCount("gamers")
		assert.NoError(t, err)
		assert.Equal(t, gamers.Count, count)

		ok, err := a.IncrementCounter("gamers", 250)
		assert.Error(t, err)
		assert.False(t, ok)

		count, err = a.GetCounterCount("gamers")
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

func TestAlphaGetAndUpdateList(t *testing.T) {
	mock := &alphaMock{}
	// Lists must be predefined in the GameServer resource on creation.
	mock.lists = make(map[string]*alpha.List)

	foo := alpha.List{
		Name:     "foo",
		Values:   []string{},
		Capacity: 2,
	}
	bar := alpha.List{
		Name:     "bar",
		Values:   []string{"abc", "def"},
		Capacity: 5,
	}
	baz := alpha.List{
		Name:     "baz",
		Values:   []string{"123", "456", "789"},
		Capacity: 5,
	}

	mock.lists["foo"] = &alpha.List{
		Name:     "foo",
		Values:   []string{},
		Capacity: 2,
	}
	mock.lists["bar"] = &alpha.List{
		Name:     "bar",
		Values:   []string{"abc", "def"},
		Capacity: 5,
	}
	mock.lists["baz"] = &alpha.List{
		Name:     "baz",
		Values:   []string{"123", "456", "789"},
		Capacity: 5,
	}

	a := Alpha{
		client: mock,
	}

	t.Parallel()

	t.Run("Get and Set List Capacity", func(t *testing.T) {
		capacity, err := a.GetListCapacity("foo")
		assert.NoError(t, err)
		assert.Equal(t, foo.Capacity, capacity)

		wantCapacity := int64(5)
		ok, err := a.SetListCapacity("foo", wantCapacity)
		assert.NoError(t, err)
		assert.True(t, ok)

		capacity, err = a.GetListCapacity("foo")
		assert.NoError(t, err)
		assert.Equal(t, wantCapacity, capacity)
	})

	t.Run("Get List Length, Get List Values, ListContains, and Append List Value", func(t *testing.T) {
		length, err := a.GetListLength("bar")
		assert.NoError(t, err)
		assert.Equal(t, len(bar.Values), length)

		values, err := a.GetListValues("bar")
		assert.NoError(t, err)
		assert.Equal(t, bar.Values, values)

		ok, err := a.AppendListValue("bar", "ghi")
		assert.NoError(t, err)
		assert.True(t, ok)

		length, err = a.GetListLength("bar")
		assert.NoError(t, err)
		assert.Equal(t, len(bar.Values)+1, length)

		wantValues := []string{"abc", "def", "ghi"}
		values, err = a.GetListValues("bar")
		assert.NoError(t, err)
		assert.Equal(t, wantValues, values)

		contains, err := a.ListContains("bar", "ghi")
		assert.NoError(t, err)
		assert.True(t, contains)
	})

	t.Run("Get List Length, Get List Values, ListContains, and Delete List Value", func(t *testing.T) {
		length, err := a.GetListLength("baz")
		assert.NoError(t, err)
		assert.Equal(t, len(baz.Values), length)

		values, err := a.GetListValues("baz")
		assert.NoError(t, err)
		assert.Equal(t, baz.Values, values)

		ok, err := a.DeleteListValue("baz", "456")
		assert.NoError(t, err)
		assert.True(t, ok)

		length, err = a.GetListLength("baz")
		assert.NoError(t, err)
		assert.Equal(t, len(baz.Values)-1, length)

		wantValues := []string{"123", "789"}
		values, err = a.GetListValues("baz")
		assert.NoError(t, err)
		assert.Equal(t, wantValues, values)

		contains, err := a.ListContains("baz", "456")
		assert.NoError(t, err)
		assert.False(t, contains)
	})

}

type alphaMock struct {
	capacity           int64
	playerCount        int64
	playerConnected    string
	playerDisconnected string
	counters           map[string]*alpha.Counter
	lists              map[string]*alpha.List
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
	return nil, errors.Errorf("counter not found: %s", in.Name)
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
			return nil, errors.Errorf("out of range. Count must be within range [0,Capacity]. Found Count: %d, Capacity: %d", count, counter.Capacity)
		}
		counter.Count = count
	case in.CounterUpdateRequest.Count != nil:
		countSet := in.CounterUpdateRequest.Count.GetValue()
		if countSet < 0 || countSet > counter.Capacity {
			return nil, errors.Errorf("out of range. Count must be within range [0,Capacity]. Found Count: %d, Capacity: %d", countSet, counter.Capacity)
		}
		counter.Count = countSet
	case in.CounterUpdateRequest.Capacity != nil:
		capacity := in.CounterUpdateRequest.Capacity.GetValue()
		if capacity < 0 {
			return nil, errors.Errorf("out of range. Capacity must be greater than or equal to 0. Found Capacity: %d", capacity)
		}
		counter.Capacity = capacity
	default:
		return nil, errors.Errorf("invalid argument. Malformed CounterUpdateRequest: %v",
			in.CounterUpdateRequest)
	}

	a.counters[in.CounterUpdateRequest.Name] = counter
	return a.counters[in.CounterUpdateRequest.Name], nil
}

// GetList returns the list of alphaMock. Note: unlike the SDK Server, this does not return
// a list with any pending batched changes applied.
func (a *alphaMock) GetList(ctx context.Context, in *alpha.GetListRequest, opts ...grpc.CallOption) (*alpha.List, error) {
	if in == nil {
		return nil, errors.Errorf("GetListRequest cannot be nil")
	}
	if list, ok := a.lists[in.Name]; ok {
		return list, nil
	}
	return nil, errors.Errorf("list not found: %s", in.Name)
}

// Note: unlike the SDK Server, UpdateList does not batch changes and instead updates the list
// directly.
func (a *alphaMock) UpdateList(ctx context.Context, in *alpha.UpdateListRequest, opts ...grpc.CallOption) (*alpha.List, error) {
	if in == nil {
		return nil, errors.Errorf("UpdateListRequest cannot be nil")
	}
	list, ok := a.lists[in.List.Name]
	if !ok {
		return nil, errors.Errorf("list not found: %s", in.List.Name)
	}
	if in.List.Capacity < 0 || in.List.Capacity > 1000 {
		return nil, errors.Errorf("out of range. Capacity must be within range [0,1000]. Found Capacity: %d", in.List.Capacity)
	}
	list.Capacity = in.List.Capacity
	if len(list.Values) > int(list.Capacity) {
		list.Values = append([]string{}, list.Values[:list.Capacity]...)
	}
	a.lists[in.List.Name] = list
	return &alpha.List{}, nil
}

// Note: unlike the SDK Server, AddListValue does not batch changes and instead updates the list
// directly.
func (a *alphaMock) AddListValue(ctx context.Context, in *alpha.AddListValueRequest, opts ...grpc.CallOption) (*alpha.List, error) {
	if in == nil {
		return nil, errors.Errorf("AddListValueRequest cannot be nil")
	}
	list, ok := a.lists[in.Name]
	if !ok {
		return nil, errors.Errorf("list not found: %s", in.Name)
	}
	if int(list.Capacity) <= len(list.Values) {
		return nil, errors.Errorf("out of range. No available capacity. Current Capacity: %d, List Size: %d", list.Capacity, len(list.Values))
	}
	for _, val := range list.Values {
		if in.Value == val {
			return nil, errors.Errorf("already exists. Value: %s already in List: %s", in.Value, in.Name)
		}
	}
	list.Values = append(list.Values, in.Value)
	a.lists[in.Name] = list
	return &alpha.List{}, nil
}

// Note: unlike the SDK Server, RemoveListValue does not batch changes and instead updates the list
// directly.
func (a *alphaMock) RemoveListValue(ctx context.Context, in *alpha.RemoveListValueRequest, opts ...grpc.CallOption) (*alpha.List, error) {
	if in == nil {
		return nil, errors.Errorf("RemoveListValueRequest cannot be nil")
	}
	list, ok := a.lists[in.Name]
	if !ok {
		return nil, errors.Errorf("list not found: %s", in.Name)
	}
	for i, val := range list.Values {
		if in.Value != val {
			continue
		}
		list.Values = append(list.Values[:i], list.Values[i+1:]...)
		a.lists[in.Name] = list
		return &alpha.List{}, nil
	}
	return nil, errors.Errorf("not found. Value: %s not found in List: %s", in.Value, in.Name)
}
