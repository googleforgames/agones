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

package sdk

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

func TestSDK(t *testing.T) {
	m := &mock{}
	s := SDK{
		ctx:    context.Background(),
		client: m,
	}
	//gate
	assert.False(t, m.ready)
	assert.False(t, m.shutdown)

	s.Ready()
	assert.True(t, m.ready)
	assert.False(t, m.shutdown)

	s.Shutdown()
	assert.True(t, m.ready)
	assert.True(t, m.shutdown)
}

var _ SDKClient = &mock{}

type mock struct {
	ready    bool
	shutdown bool
}

func (m *mock) Ready(ctx context.Context, e *Empty, opts ...grpc.CallOption) (*Empty, error) {
	m.ready = true
	return e, nil
}

func (m *mock) Shutdown(ctx context.Context, e *Empty, opts ...grpc.CallOption) (*Empty, error) {
	m.shutdown = true
	return e, nil
}
