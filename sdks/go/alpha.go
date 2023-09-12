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

	"agones.dev/agones/pkg/sdk/alpha"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

// Alpha is the struct for Alpha SDK functionality.
type Alpha struct {
	client alpha.SDKClient
}

// newAlpha creates a new Alpha SDK with the passed in connection.
func newAlpha(conn *grpc.ClientConn) *Alpha {
	return &Alpha{
		client: alpha.NewSDKClient(conn),
	}
}

// GetPlayerCapacity gets the last player capacity that was set through the SDK.
// If the player capacity is set from outside the SDK, use SDK.GameServer() instead.
func (a *Alpha) GetPlayerCapacity() (int64, error) {
	c, err := a.client.GetPlayerCapacity(context.Background(), &alpha.Empty{})
	return c.GetCount(), errors.Wrap(err, "could not get player capacity")
}

// SetPlayerCapacity changes the player capacity to a new value.
func (a *Alpha) SetPlayerCapacity(capacity int64) error {
	_, err := a.client.SetPlayerCapacity(context.Background(), &alpha.Count{Count: capacity})
	return errors.Wrap(err, "could not set player capacity")
}

// PlayerConnect increases the SDK’s stored player count by one, and appends this playerID to status.players.id.
// Will return true and add the playerID to the list of playerIDs if the playerIDs was not already in the
// list of connected playerIDs.
func (a *Alpha) PlayerConnect(id string) (bool, error) {
	ok, err := a.client.PlayerConnect(context.Background(), &alpha.PlayerID{PlayerID: id})
	return ok.GetBool(), errors.Wrap(err, "could not register connected player")
}

// PlayerDisconnect Decreases the SDK’s stored player count by one, and removes the playerID from status.players.id.
// Will return true and remove the supplied playerID from the list of connected playerIDs if the
// playerID value exists within the list.
func (a *Alpha) PlayerDisconnect(id string) (bool, error) {
	ok, err := a.client.PlayerDisconnect(context.Background(), &alpha.PlayerID{PlayerID: id})
	return ok.GetBool(), errors.Wrap(err, "could not register disconnected player")
}

// GetPlayerCount returns the current player count.
func (a *Alpha) GetPlayerCount() (int64, error) {
	count, err := a.client.GetPlayerCount(context.Background(), &alpha.Empty{})
	return count.GetCount(), errors.Wrap(err, "could not get player count")
}

// IsPlayerConnected returns if the playerID is currently connected to the GameServer.
// This is always accurate, even if the value hasn’t been updated to the GameServer status yet.
func (a *Alpha) IsPlayerConnected(id string) (bool, error) {
	ok, err := a.client.IsPlayerConnected(context.Background(), &alpha.PlayerID{PlayerID: id})
	return ok.GetBool(), errors.Wrap(err, "could not get if player is connected")
}

// GetConnectedPlayers returns the list of the currently connected player ids.
// This is always accurate, even if the value hasn’t been updated to the GameServer status yet.
func (a *Alpha) GetConnectedPlayers() ([]string, error) {
	list, err := a.client.GetConnectedPlayers(context.Background(), &alpha.Empty{})
	return list.GetList(), errors.Wrap(err, "could not list connected players")
}

// GetCounterCount returns the Count for a Counter, given the Counter's key (name).
// Will error if the key was not predefined in the GameServer resource on creation.
func (a *Alpha) GetCounterCount(key string) (int64, error) {
	counter, err := a.client.GetCounter(context.Background(), &alpha.GetCounterRequest{Name: key})
	if err != nil {
		return -1, errors.Wrapf(err, "could not get Counter %s count", key)
	}
	return counter.Count, nil
}

// IncrementCounter increases a counter by the given nonnegative integer amount.
// Will execute the increment operation against the current CRD value. Will max at max(int64).
// Will error if the key was not predefined in the GameServer resource on creation.
// Returns false if the count is at the current capacity (to the latest knowledge of the SDK),
// and no increment will occur.
//
// Note: A potential race condition here is that if count values are set from both the SDK and
// through the K8s API (Allocation or otherwise), since the SDK append operation back to the CRD
// value is batched asynchronous any value incremented past the capacity will be silently truncated.
func (a *Alpha) IncrementCounter(key string, amount int64) (bool, error) {
	if amount < 0 {
		return false, errors.Errorf("CountIncrement amount must be a positive int64, found %d", amount)
	}
	_, err := a.client.UpdateCounter(context.Background(), &alpha.UpdateCounterRequest{
		CounterUpdateRequest: &alpha.CounterUpdateRequest{
			Name:      key,
			CountDiff: amount,
		}})
	if err != nil {
		return false, errors.Wrapf(err, "could not increment Counter %s by amount %d", key, amount)
	}
	return true, err
}

// DecrementCounter decreases the current count by the given nonnegative integer amount.
// The Counter Will not go below 0. Will execute the decrement operation against the current CRD value.
// Returns false if the count is at 0 (to the latest knowledge of the SDK), and no decrement will occur.
func (a *Alpha) DecrementCounter(key string, amount int64) (bool, error) {
	if amount < 0 {
		return false, errors.Errorf("CountDecrement amount must be a positive int64, found %d", amount)
	}
	_, err := a.client.UpdateCounter(context.Background(), &alpha.UpdateCounterRequest{
		CounterUpdateRequest: &alpha.CounterUpdateRequest{
			Name:      key,
			CountDiff: amount * -1,
		}})
	if err != nil {
		return false, errors.Wrapf(err, "could not decrement Counter %s by amount %d", key, amount)
	}
	return true, err
}

// SetCounterCount sets a count to the given value. Use with care, as this will overwrite any previous
// invocations’ value. Cannot be greater than Capacity.
func (a *Alpha) SetCounterCount(key string, amount int64) (bool, error) {
	_, err := a.client.UpdateCounter(context.Background(), &alpha.UpdateCounterRequest{
		CounterUpdateRequest: &alpha.CounterUpdateRequest{
			Name:  key,
			Count: wrapperspb.Int64(amount),
		}})
	if err != nil {
		return false, errors.Wrapf(err, "could not set Counter %s count to amount %d", key, amount)
	}
	return true, err
}

// GetCounterCapacity returns the Capacity for a Counter, given the Counter's key (name).
// Will error if the key was not predefined in the GameServer resource on creation.
func (a *Alpha) GetCounterCapacity(key string) (int64, error) {
	counter, err := a.client.GetCounter(context.Background(), &alpha.GetCounterRequest{Name: key})
	if err != nil {
		return -1, errors.Wrapf(err, "could not get Counter %s capacity", key)
	}
	return counter.Capacity, nil
}

// SetCounterCapacity sets the capacity for a given count. A capacity of 0 is no capacity.
func (a *Alpha) SetCounterCapacity(key string, amount int64) (bool, error) {
	_, err := a.client.UpdateCounter(context.Background(), &alpha.UpdateCounterRequest{
		CounterUpdateRequest: &alpha.CounterUpdateRequest{
			Name:     key,
			Capacity: wrapperspb.Int64(amount),
		}})
	if err != nil {
		return false, errors.Wrapf(err, "could not set Counter %s capacity to amount %d", key, amount)
	}
	return true, err
}
