// Copyright 2018 Google LLC All Rights Reserved.
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

package sdkserver

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"sync"
	"testing"
	"time"

	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	"agones.dev/agones/pkg/sdk"
	"agones.dev/agones/pkg/sdk/alpha"
	"agones.dev/agones/pkg/util/runtime"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

func TestLocal(t *testing.T) {
	ctx := context.Background()
	e := &sdk.Empty{}
	l, err := NewLocalSDKServer("")
	assert.Nil(t, err)

	_, err = l.Ready(ctx, e)
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

	assert.Equal(t, defaultGs, gs)
}

func TestLocalSDKWithTestMode(t *testing.T) {
	l, err := NewLocalSDKServer("")
	assert.NoError(t, err, "Should be able to create local SDK server")
	a := []string{"ready", "allocate", "setlabel", "setannotation", "gameserver", "health", "shutdown", "watch"}
	b := []string{"ready", "health", "ready", "watch", "allocate", "gameserver", "setlabel", "setannotation", "health", "health", "shutdown"}
	assert.True(t, l.EqualSets(a, a))
	assert.True(t, l.EqualSets(a, b))
	assert.True(t, l.EqualSets(b, a))
	assert.True(t, l.EqualSets(b, b))
	a[0] = "rady"
	assert.False(t, l.EqualSets(a, b))
	assert.False(t, l.EqualSets(b, a))
	a[0] = "ready"
	b[1] = "halth"
	assert.False(t, l.EqualSets(a, b))
	assert.False(t, l.EqualSets(b, a))
}

func TestLocalSDKWithGameServer(t *testing.T) {
	ctx := context.Background()
	e := &sdk.Empty{}

	fixture := &agonesv1.GameServer{ObjectMeta: metav1.ObjectMeta{Name: "stuff"}}
	path, err := gsToTmpFile(fixture.DeepCopy())
	assert.Nil(t, err)

	l, err := NewLocalSDKServer(path)
	assert.Nil(t, err)

	gs, err := l.GetGameServer(ctx, e)
	assert.Nil(t, err)

	assert.Equal(t, fixture.ObjectMeta.Name, gs.ObjectMeta.Name)
}

// nolint:dupl
func TestLocalSDKServerSetLabel(t *testing.T) {
	t.Parallel()

	fixtures := map[string]struct {
		gs *agonesv1.GameServer
	}{
		"default": {
			gs: nil,
		},
		"no labels": {
			gs: &agonesv1.GameServer{ObjectMeta: metav1.ObjectMeta{Name: "empty"}},
		},
		"empty": {
			gs: &agonesv1.GameServer{},
		},
	}

	for k, v := range fixtures {
		t.Run(k, func(t *testing.T) {
			ctx := context.Background()
			e := &sdk.Empty{}
			path, err := gsToTmpFile(v.gs)
			assert.Nil(t, err)

			l, err := NewLocalSDKServer(path)
			assert.Nil(t, err)
			kv := &sdk.KeyValue{Key: "foo", Value: "bar"}

			stream := newGameServerMockStream()
			wg := sync.WaitGroup{}
			wg.Add(1)
			go func() {
				defer wg.Done()
				err := l.WatchGameServer(e, stream)
				assert.Nil(t, err)
			}()

			// make sure length of l.updateObservers is at least 1
			err = wait.PollImmediate(time.Second, 10*time.Second, func() (bool, error) {
				ret := false
				l.updateObservers.Range(func(_, _ interface{}) bool {
					ret = true
					return false
				})

				return ret, nil
			})
			assert.Nil(t, err)

			_, err = l.SetLabel(ctx, kv)
			assert.Nil(t, err)

			gs, err := l.GetGameServer(ctx, e)
			assert.Nil(t, err)
			assert.Equal(t, gs.ObjectMeta.Labels[metadataPrefix+"foo"], "bar")

			select {
			case msg := <-stream.msgs:
				assert.Equal(t, msg.ObjectMeta.Labels[metadataPrefix+"foo"], "bar")
			case <-time.After(10 * time.Second):
				assert.Fail(t, "timeout on receiving messages")
			}

			l.Close()
			wg.Wait()
		})
	}
}

// nolint:dupl
func TestLocalSDKServerSetAnnotation(t *testing.T) {
	t.Parallel()

	fixtures := map[string]struct {
		gs *agonesv1.GameServer
	}{
		"default": {
			gs: nil,
		},
		"no annotation": {
			gs: &agonesv1.GameServer{ObjectMeta: metav1.ObjectMeta{Name: "empty"}},
		},
		"empty": {
			gs: &agonesv1.GameServer{},
		},
	}

	for k, v := range fixtures {
		t.Run(k, func(t *testing.T) {
			ctx := context.Background()
			e := &sdk.Empty{}
			path, err := gsToTmpFile(v.gs)
			assert.Nil(t, err)

			l, err := NewLocalSDKServer(path)
			assert.Nil(t, err)

			kv := &sdk.KeyValue{Key: "bar", Value: "foo"}

			stream := newGameServerMockStream()
			wg := sync.WaitGroup{}
			wg.Add(1)
			go func() {
				defer wg.Done()
				err := l.WatchGameServer(e, stream)
				assert.Nil(t, err)
			}()

			// make sure length of l.updateObservers is at least 1
			err = wait.PollImmediate(time.Second, 10*time.Second, func() (bool, error) {
				ret := false
				l.updateObservers.Range(func(_, _ interface{}) bool {
					ret = true
					return false
				})

				return ret, nil
			})
			assert.Nil(t, err)

			_, err = l.SetAnnotation(ctx, kv)
			assert.Nil(t, err)

			gs, err := l.GetGameServer(ctx, e)
			assert.Nil(t, err)
			assert.Equal(t, gs.ObjectMeta.Annotations[metadataPrefix+"bar"], "foo")

			select {
			case msg := <-stream.msgs:
				assert.Equal(t, msg.ObjectMeta.Annotations[metadataPrefix+"bar"], "foo")
			case <-time.After(10 * time.Second):
				assert.FailNow(t, "timeout on receiving messages")
			}

			l.Close()
			wg.Wait()
		})
	}
}

func TestLocalSDKServerWatchGameServer(t *testing.T) {
	t.Parallel()

	fixture := &agonesv1.GameServer{ObjectMeta: metav1.ObjectMeta{Name: "stuff"}}
	path, err := gsToTmpFile(fixture)
	assert.Nil(t, err)

	e := &sdk.Empty{}
	l, err := NewLocalSDKServer(path)
	assert.Nil(t, err)

	stream := newGameServerMockStream()
	go func() {
		err := l.WatchGameServer(e, stream)
		assert.Nil(t, err)
	}()

	select {
	case <-stream.msgs:
		assert.Fail(t, "should not get a message")
	case <-time.After(time.Second):
	}

	fixture.ObjectMeta.Annotations = map[string]string{"foo": "bar"}
	j, err := json.Marshal(fixture)
	assert.Nil(t, err)

	err = ioutil.WriteFile(path, j, os.ModeDevice)
	assert.Nil(t, err)

	select {
	case msg := <-stream.msgs:
		assert.Equal(t, "bar", msg.ObjectMeta.Annotations["foo"])
	case <-time.After(10 * time.Second):
		assert.Fail(t, "timeout getting watch")
	}
}

func TestLocalSDKServerPlayerCapacity(t *testing.T) {
	t.Parallel()

	runtime.FeatureTestMutex.Lock()
	defer runtime.FeatureTestMutex.Unlock()
	assert.NoError(t, runtime.ParseFeatures(string(runtime.FeaturePlayerTracking)+"=true"))

	fixture := &agonesv1.GameServer{ObjectMeta: metav1.ObjectMeta{Name: "stuff"}}

	e := &alpha.Empty{}
	path, err := gsToTmpFile(fixture)
	assert.NoError(t, err)
	l, err := NewLocalSDKServer(path)
	assert.Nil(t, err)

	stream := newGameServerMockStream()
	go func() {
		err := l.WatchGameServer(&sdk.Empty{}, stream)
		assert.Nil(t, err)
	}()

	// wait for watching to begin
	err = wait.Poll(time.Second, 10*time.Second, func() (bool, error) {
		found := false
		l.updateObservers.Range(func(_, _ interface{}) bool {
			found = true
			return false
		})
		return found, nil
	})
	assert.NoError(t, err)

	c, err := l.GetPlayerCapacity(context.Background(), e)
	assert.NoError(t, err)
	assert.Equal(t, int64(0), c.Count)

	_, err = l.SetPlayerCapacity(context.Background(), &alpha.Count{Count: 10})
	assert.NoError(t, err)

	select {
	case msg := <-stream.msgs:
		assert.Equal(t, int64(10), msg.Status.Players.Capacity)
	case <-time.After(10 * time.Second):
		assert.Fail(t, "timeout getting watch")
	}

	c, err = l.GetPlayerCapacity(context.Background(), e)
	assert.NoError(t, err)
	assert.Equal(t, int64(10), c.Count)

	gs, err := l.GetGameServer(context.Background(), &sdk.Empty{})
	assert.NoError(t, err)
	assert.Equal(t, int64(10), gs.Status.Players.Capacity)
}

func gsToTmpFile(gs *agonesv1.GameServer) (string, error) {
	file, err := ioutil.TempFile(os.TempDir(), "gameserver-")
	if err != nil {
		return file.Name(), err
	}

	err = json.NewEncoder(file).Encode(gs)
	return file.Name(), err
}

// TestLocalSDKServerStateUpdates verify that SDK functions changes the state of the
// GameServer object
func TestLocalSDKServerStateUpdates(t *testing.T) {
	t.Parallel()
	l, err := NewLocalSDKServer("")
	assert.Nil(t, err)

	ctx := context.Background()
	e := &sdk.Empty{}
	_, err = l.Ready(ctx, e)
	assert.Nil(t, err)

	gs, err := l.GetGameServer(ctx, e)
	assert.Nil(t, err)
	assert.Equal(t, gs.Status.State, string(agonesv1.GameServerStateReady))

	seconds := &sdk.Duration{Seconds: 2}
	_, err = l.Reserve(ctx, seconds)
	assert.Nil(t, err)

	gs, err = l.GetGameServer(ctx, e)
	assert.Nil(t, err)
	assert.Equal(t, gs.Status.State, string(agonesv1.GameServerStateReserved))

	_, err = l.Allocate(ctx, e)
	assert.Nil(t, err)

	gs, err = l.GetGameServer(ctx, e)
	assert.Nil(t, err)
	assert.Equal(t, gs.Status.State, string(agonesv1.GameServerStateAllocated))

	_, err = l.Shutdown(ctx, e)
	assert.Nil(t, err)

	gs, err = l.GetGameServer(ctx, e)
	assert.Nil(t, err)
	assert.Equal(t, gs.Status.State, string(agonesv1.GameServerStateShutdown))
}

// TestSDKConformanceFunctionality - run a number of record requests in parallel
func TestSDKConformanceFunctionality(t *testing.T) {
	t.Parallel()

	l, err := NewLocalSDKServer("")
	assert.Nil(t, err)
	l.testMode = true
	l.recordRequest("")
	l.gs = &sdk.GameServer{ObjectMeta: &sdk.GameServer_ObjectMeta{Name: "empty"}}
	exampleUID := "052fb0f4-3d50-11e5-b066-42010af0d7b6"
	// field which is tested
	setAnnotation := "setannotation"
	l.gs.ObjectMeta.Uid = exampleUID

	expected := []string{}
	expected = append(expected, "", setAnnotation)

	wg := sync.WaitGroup{}
	for i := 0; i < 20; i++ {
		wg.Add(1)
		str := fmt.Sprintf("%d", i)
		expected = append(expected, str)

		go func() {
			l.recordRequest(str)
			l.recordRequestWithValue(setAnnotation, exampleUID, "UID")
			wg.Done()
		}()
	}
	wg.Wait()

	l.SetExpectedSequence(expected)
	b := l.EqualSets(l.expectedSequence, l.requestSequence)
	assert.True(t, b, "we should receive strings from all go routines %v %v", l.expectedSequence, l.requestSequence)
}
