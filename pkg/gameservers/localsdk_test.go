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

package gameservers

import (
	"sync"
	"testing"
	"time"

	"agones.dev/agones/pkg/sdk"
	"github.com/stretchr/testify/assert"
	"golang.org/x/net/context"
)

func TestLocal(t *testing.T) {
	ctx := context.Background()
	e := &sdk.Empty{}
	l := NewLocalSDKServer()

	_, err := l.Ready(ctx, e)
	assert.Nil(t, err, "Ready should not error")

	_, err = l.Shutdown(ctx, e)
	assert.Nil(t, err, "Shutdown should not error")

	wg := sync.WaitGroup{}
	wg.Add(1)
	stream := newEmptyMockStream()

	go func() {
		err = l.Health(stream)
		assert.Nil(t, err)
		wg.Done()
	}()

	stream.msgs <- &sdk.Empty{}
	close(stream.msgs)

	wg.Wait()

	gs, err := l.GetGameServer(ctx, e)
	assert.Nil(t, err)

	assert.Equal(t, fixture, gs)
}

func TestLocalSDKServerSetLabel(t *testing.T) {
	ctx := context.Background()
	e := &sdk.Empty{}
	l := NewLocalSDKServer()
	kv := &sdk.KeyValue{Key: "foo", Value: "bar"}

	stream := newGameServerMockStream()
	go func() {
		err := l.WatchGameServer(e, stream)
		assert.Nil(t, err)
	}()

	_, err := l.SetLabel(ctx, kv)
	assert.Nil(t, err)

	gs, err := l.GetGameServer(ctx, e)
	assert.Nil(t, err)
	assert.Equal(t, gs.ObjectMeta.Labels[metadataPrefix+"foo"], "bar")

	select {
	case msg := <-stream.msgs:
		assert.Equal(t, msg.ObjectMeta.Labels[metadataPrefix+"foo"], "bar")
	case <-time.After(2 * l.watchPeriod):
		assert.FailNow(t, "timeout on receiving messages")
	}
}

func TestLocalSDKServerSetAnnotation(t *testing.T) {
	ctx := context.Background()
	e := &sdk.Empty{}
	l := NewLocalSDKServer()
	kv := &sdk.KeyValue{Key: "bar", Value: "foo"}

	stream := newGameServerMockStream()
	go func() {
		err := l.WatchGameServer(e, stream)
		assert.Nil(t, err)
	}()

	_, err := l.SetAnnotation(ctx, kv)
	assert.Nil(t, err)

	gs, err := l.GetGameServer(ctx, e)
	assert.Nil(t, err)
	assert.Equal(t, gs.ObjectMeta.Annotations[metadataPrefix+"bar"], "foo")

	select {
	case msg := <-stream.msgs:
		assert.Equal(t, msg.ObjectMeta.Annotations[metadataPrefix+"bar"], "foo")
	case <-time.After(2 * l.watchPeriod):
		assert.FailNow(t, "timeout on receiving messages")
	}
}

func TestLocalSDKServerWatchGameServer(t *testing.T) {
	t.Parallel()

	e := &sdk.Empty{}
	l := NewLocalSDKServer()
	l.watchPeriod = time.Second

	stream := newGameServerMockStream()
	go func() {
		err := l.WatchGameServer(e, stream)
		assert.Nil(t, err)
	}()

	for i := 0; i < 3; i++ {
		select {
		case msg := <-stream.msgs:
			assert.Equal(t, fixture, msg)
		case <-time.After(2 * l.watchPeriod):
			assert.FailNow(t, "timeout on receiving messages")
		}
	}
}
