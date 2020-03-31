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
)

// Alpha is the struct for Alpha SDK functionality
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
	return c.Count, errors.Wrap(err, "could not get player capacity")
}

// SetPlayerCapacity changes the player capacity to a new value
func (a *Alpha) SetPlayerCapacity(capacity int64) error {
	_, err := a.client.SetPlayerCapacity(context.Background(), &alpha.Count{Count: capacity})
	return errors.Wrap(err, "could not set player capacity")
}
