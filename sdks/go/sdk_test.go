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
	"google.golang.org/grpc/metadata"
)

func TestSDK(t *testing.T) {
	sm := &sdkMock{
		hm: &healthMock{},
	}
	s := SDK{
		ctx:    context.Background(),
		client: sm,
		health: sm.hm,
	}

	//gate
	assert.False(t, sm.ready)
	assert.False(t, sm.shutdown)
	assert.False(t, sm.hm.healthy)

	s.Ready()
	assert.True(t, sm.ready)
	assert.False(t, sm.shutdown)

	s.Health()
	assert.True(t, sm.hm.healthy)

	s.Shutdown()
	assert.True(t, sm.ready)
	assert.True(t, sm.shutdown)
}

var _ SDKClient = &sdkMock{}
var _ SDK_HealthClient = &healthMock{}

type sdkMock struct {
	ready    bool
	shutdown bool
	hm       *healthMock
}

func (m *sdkMock) Ready(ctx context.Context, e *Empty, opts ...grpc.CallOption) (*Empty, error) {
	m.ready = true
	return e, nil
}

func (m *sdkMock) Shutdown(ctx context.Context, e *Empty, opts ...grpc.CallOption) (*Empty, error) {
	m.shutdown = true
	return e, nil
}

func (m *sdkMock) Health(ctx context.Context, opts ...grpc.CallOption) (SDK_HealthClient, error) {
	return m.hm, nil
}

type healthMock struct {
	healthy bool
}

func (h *healthMock) Send(*Empty) error {
	h.healthy = true
	return nil
}

func (h *healthMock) CloseAndRecv() (*Empty, error) {
	panic("implement me")
}

func (h *healthMock) Header() (metadata.MD, error) {
	panic("implement me")
}

func (h *healthMock) Trailer() metadata.MD {
	panic("implement me")
}

func (h *healthMock) CloseSend() error {
	panic("implement me")
}

func (h *healthMock) Context() context.Context {
	panic("implement me")
}

func (h *healthMock) SendMsg(m interface{}) error {
	panic("implement me")
}

func (h *healthMock) RecvMsg(m interface{}) error {
	panic("implement me")
}
