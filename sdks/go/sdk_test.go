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

	"agones.dev/agones/pkg/sdk"
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

	err := s.Ready()
	assert.Nil(t, err)
	assert.True(t, sm.ready)
	assert.False(t, sm.shutdown)

	err = s.Health()
	assert.Nil(t, err)
	assert.True(t, sm.hm.healthy)

	err = s.Shutdown()
	assert.Nil(t, err)
	assert.True(t, sm.ready)
	assert.True(t, sm.shutdown)

	gs, err := s.GameServer()
	assert.Nil(t, err)
	assert.NotNil(t, gs)
}

var _ sdk.SDKClient = &sdkMock{}
var _ sdk.SDK_HealthClient = &healthMock{}

type sdkMock struct {
	ready    bool
	shutdown bool
	hm       *healthMock
}

func (m *sdkMock) GetGameServer(ctx context.Context, in *sdk.Empty, opts ...grpc.CallOption) (*sdk.GameServer, error) {
	return &sdk.GameServer{}, nil
}

func (m *sdkMock) Ready(ctx context.Context, e *sdk.Empty, opts ...grpc.CallOption) (*sdk.Empty, error) {
	m.ready = true
	return e, nil
}

func (m *sdkMock) Shutdown(ctx context.Context, e *sdk.Empty, opts ...grpc.CallOption) (*sdk.Empty, error) {
	m.shutdown = true
	return e, nil
}

func (m *sdkMock) Health(ctx context.Context, opts ...grpc.CallOption) (sdk.SDK_HealthClient, error) {
	return m.hm, nil
}

type healthMock struct {
	healthy bool
}

func (h *healthMock) Send(*sdk.Empty) error {
	h.healthy = true
	return nil
}

func (h *healthMock) CloseAndRecv() (*sdk.Empty, error) {
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
