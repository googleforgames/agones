// Copyright 2018 Google Inc. All Rights Reserved.
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

package main

import (
	"net"
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/util/clock"
)

type mockAddr struct {
	addr string
}

func (m *mockAddr) Network() string {
	return m.addr
}

func (m *mockAddr) String() string {
	return m.addr
}

func TestUDPServerVisit(t *testing.T) {
	t.Parallel()

	fc := clock.NewFakeClock(time.Now())
	u, err := defaultFixture(fc)
	assert.Nil(t, err)
	defer u.close()

	// gate
	assert.Empty(t, u.limits)

	m := &mockAddr{addr: "[::1]:52998"}

	u.rateLimitedEchoResponse([]byte{}, m)

	// gate
	assert.NotEmpty(t, u.limits)
	assert.Len(t, u.limits, 1)
	assert.Equal(t, fc.Now(), u.limits[m.Network()].stamp)

	fc.Step(30 * time.Second)

	u.rateLimitedEchoResponse([]byte{}, m)
	assert.Len(t, u.limits, 1)
	assert.Equal(t, fc.Now(), u.limits[m.Network()].stamp)

	m = &mockAddr{addr: "[::1]:52999"}
	u.rateLimitedEchoResponse([]byte{}, m)
	assert.Len(t, u.limits, 2)
	assert.Equal(t, fc.Now(), u.limits[m.Network()].stamp)
}

func TestUDPServerCleanup(t *testing.T) {
	t.Parallel()

	fc := clock.NewFakeClock(time.Now())
	u, err := defaultFixture(fc)
	assert.Nil(t, err)
	defer u.close()

	// gate
	assert.Empty(t, u.limits)

	m := &mockAddr{addr: "[::1]:52998"}
	u.rateLimitedEchoResponse([]byte{}, m)

	// gate
	assert.NotEmpty(t, u.limits)

	assert.Equal(t, u.clock.Now(), u.limits[m.String()].stamp)
	fc.Step(10 * time.Second)
	u.cleanUp()
	assert.NotEmpty(t, u.limits)

	fc.Step(time.Minute)
	u.cleanUp()
	assert.Empty(t, u.limits)
}

func TestUDPServerHealth(t *testing.T) {
	t.Parallel()

	fc := clock.NewFakeClock(time.Now())
	u, err := defaultFixture(fc)
	assert.Nil(t, err)
	defer u.close()

	assert.Error(t, u.Health())

	stop := make(chan struct{})
	u.run(stop)

	assert.Nil(t, u.Health())

	close(stop)

	err = wait.PollImmediate(time.Second, 5*time.Second, func() (done bool, err error) {
		return u.Health() != nil, nil
	})

	assert.Nil(t, err)
}

func defaultFixture(clock clock.Clock) (*udpServer, error) {
	u := newUDPServer(5)
	u.clock = clock
	var err error
	u.conn, err = net.ListenPacket("udp", ":0")
	return u, err
}
