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
	"os"
	"strconv"
	"sync"
	"testing"
	"time"

	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	"agones.dev/agones/pkg/sdk"
	"agones.dev/agones/pkg/sdk/alpha"
	"agones.dev/agones/pkg/util/runtime"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

func TestLocal(t *testing.T) {
	ctx := context.Background()
	e := &sdk.Empty{}
	l, err := NewLocalSDKServer("", "")
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

	defaultGameServer := defaultGs()
	// do this to adjust for any time differences.
	// we only care about all the other values to be compared.
	defaultGameServer.ObjectMeta.CreationTimestamp = gs.GetObjectMeta().CreationTimestamp

	assert.Equal(t, defaultGameServer.GetObjectMeta(), gs.GetObjectMeta())
	assert.Equal(t, defaultGameServer.GetSpec(), gs.GetSpec())
	gsStatus := defaultGameServer.GetStatus()
	gsStatus.State = "Shutdown"
	assert.Equal(t, gsStatus, gs.GetStatus())
}

func TestLocalSDKWithTestMode(t *testing.T) {
	l, err := NewLocalSDKServer("", "")
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

	l, err := NewLocalSDKServer(path, "")
	assert.Nil(t, err)

	gs, err := l.GetGameServer(ctx, e)
	assert.Nil(t, err)

	assert.Equal(t, fixture.ObjectMeta.Name, gs.ObjectMeta.Name)
}

// nolint:dupl
func TestLocalSDKWithLogLevel(t *testing.T) {
	ctx := context.Background()
	e := &sdk.Empty{}

	fixture := &agonesv1.GameServer{
		ObjectMeta: metav1.ObjectMeta{Name: "stuff"},
		Spec: agonesv1.GameServerSpec{
			SdkServer: agonesv1.SdkServer{LogLevel: "debug"},
		},
	}
	path, err := gsToTmpFile(fixture.DeepCopy())
	assert.Nil(t, err)

	l, err := NewLocalSDKServer(path, "test")
	assert.Nil(t, err)

	_, err = l.GetGameServer(ctx, e)
	assert.Nil(t, err)

	// Check if the LocalSDKServer's logger.LogLevel equal fixture's
	assert.Equal(t, string(fixture.Spec.SdkServer.LogLevel), l.logger.Logger.Level.String())
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
		// pin variables here, see scopelint for details
		k := k
		v := v
		t.Run(k, func(t *testing.T) {
			ctx := context.Background()
			e := &sdk.Empty{}
			path, err := gsToTmpFile(v.gs)
			assert.Nil(t, err)

			l, err := NewLocalSDKServer(path, "")
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
			assertInitialWatchUpdate(t, stream)

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

			assertWatchUpdate(t, stream, "bar", func(gs *sdk.GameServer) interface{} {
				return gs.ObjectMeta.Labels[metadataPrefix+"foo"]
			})

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

			l, err := NewLocalSDKServer(path, "")
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
			assertInitialWatchUpdate(t, stream)

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

			assertWatchUpdate(t, stream, "foo", func(gs *sdk.GameServer) interface{} {
				return gs.ObjectMeta.Annotations[metadataPrefix+"bar"]
			})

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
	l, err := NewLocalSDKServer(path, "")
	assert.Nil(t, err)

	stream := newGameServerMockStream()
	go func() {
		err := l.WatchGameServer(e, stream)
		assert.Nil(t, err)
	}()
	assertInitialWatchUpdate(t, stream)

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

	assertNoWatchUpdate(t, stream)
	fixture.ObjectMeta.Annotations = map[string]string{"foo": "bar"}
	j, err := json.Marshal(fixture)
	assert.Nil(t, err)

	err = os.WriteFile(path, j, os.ModeDevice)
	assert.Nil(t, err)

	assertWatchUpdate(t, stream, "bar", func(gs *sdk.GameServer) interface{} {
		return gs.ObjectMeta.Annotations["foo"]
	})
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
	l, err := NewLocalSDKServer(path, "")
	assert.Nil(t, err)

	stream := newGameServerMockStream()
	go func() {
		err := l.WatchGameServer(&sdk.Empty{}, stream)
		assert.Nil(t, err)
	}()
	assertInitialWatchUpdate(t, stream)

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

func TestLocalSDKServerPlayerConnectAndDisconnectWithoutPlayerTracking(t *testing.T) {
	t.Parallel()
	runtime.FeatureTestMutex.Lock()
	defer runtime.FeatureTestMutex.Unlock()

	assert.NoError(t, runtime.ParseFeatures(string(runtime.FeaturePlayerTracking)+"=false"))

	l, err := NewLocalSDKServer("", "")
	assert.Nil(t, err)

	e := &alpha.Empty{}
	capacity, err := l.GetPlayerCapacity(context.Background(), e)
	assert.Nil(t, capacity)
	assert.Error(t, err)

	count, err := l.GetPlayerCount(context.Background(), e)
	assert.Error(t, err)
	assert.Nil(t, count)

	list, err := l.GetConnectedPlayers(context.Background(), e)
	assert.Error(t, err)
	assert.Nil(t, list)

	id := &alpha.PlayerID{PlayerID: "test-player"}

	ok, err := l.PlayerConnect(context.Background(), id)
	assert.Error(t, err)
	assert.False(t, ok.Bool)

	ok, err = l.IsPlayerConnected(context.Background(), id)
	assert.Error(t, err)
	assert.False(t, ok.Bool)

	ok, err = l.PlayerDisconnect(context.Background(), id)
	assert.Error(t, err)
	assert.False(t, ok.Bool)
}

func TestLocalSDKServerPlayerConnectAndDisconnect(t *testing.T) {
	t.Parallel()

	runtime.FeatureTestMutex.Lock()
	defer runtime.FeatureTestMutex.Unlock()
	assert.NoError(t, runtime.ParseFeatures(string(runtime.FeaturePlayerTracking)+"=true"))

	gs := func() *agonesv1.GameServer {
		return &agonesv1.GameServer{
			ObjectMeta: metav1.ObjectMeta{Name: "stuff"},
			Status: agonesv1.GameServerStatus{
				Players: &agonesv1.PlayerStatus{
					Capacity: 1,
				},
			}}
	}

	e := &alpha.Empty{}

	fixtures := map[string]struct {
		testMode bool
		gs       *agonesv1.GameServer
		useFile  bool
	}{
		"test mode on, gs with Status.Players": {
			testMode: true,
			gs:       gs(),
			useFile:  true,
		},
		"test mode off, gs with Status.Players": {
			testMode: false,
			gs:       gs(),
			useFile:  true,
		},
		"test mode on, gs without Status.Players": {
			testMode: true,
			useFile:  true,
		},
		"test mode off, gs without Status.Players": {
			testMode: false,
			useFile:  true,
		},
		"test mode on, no filePath": {
			testMode: true,
			useFile:  false,
		},
		"test mode off, no filePath": {
			testMode: false,
			useFile:  false,
		},
	}

	for k, v := range fixtures {
		// pin variables here, see https://github.com/kyoh86/scopelint for the details
		k := k
		v := v
		t.Run(k, func(t *testing.T) {
			var l *LocalSDKServer
			var err error
			if v.useFile {
				path, pathErr := gsToTmpFile(v.gs)
				assert.NoError(t, pathErr)
				l, err = NewLocalSDKServer(path, "")
			} else {
				l, err = NewLocalSDKServer("", "")
			}
			assert.Nil(t, err)
			l.SetTestMode(v.testMode)

			stream := newGameServerMockStream()
			go func() {
				err := l.WatchGameServer(&sdk.Empty{}, stream)
				assert.Nil(t, err)
			}()
			assertInitialWatchUpdate(t, stream)

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

			if !v.useFile || v.gs == nil {
				_, err := l.SetPlayerCapacity(context.Background(), &alpha.Count{
					Count: 1,
				})
				assert.NoError(t, err)
				expected := &sdk.GameServer_Status_PlayerStatus{
					Capacity: 1,
				}
				assertWatchUpdate(t, stream, expected, func(gs *sdk.GameServer) interface{} {
					return gs.Status.Players
				})
			}

			id := &alpha.PlayerID{PlayerID: "one"}
			ok, err := l.IsPlayerConnected(context.Background(), id)
			assert.NoError(t, err)
			if assert.NotNil(t, ok) {
				assert.False(t, ok.Bool, "player should not be connected")
			}

			count, err := l.GetPlayerCount(context.Background(), e)
			assert.NoError(t, err)
			assert.Equal(t, int64(0), count.Count)

			list, err := l.GetConnectedPlayers(context.Background(), e)
			assert.NoError(t, err)
			assert.Empty(t, list.List)

			// connect a player
			ok, err = l.PlayerConnect(context.Background(), id)
			assert.NoError(t, err)
			assert.True(t, ok.Bool, "Player should not exist yet")

			count, err = l.GetPlayerCount(context.Background(), e)
			assert.NoError(t, err)
			assert.Equal(t, int64(1), count.Count)

			expected := &sdk.GameServer_Status_PlayerStatus{
				Count:    1,
				Capacity: 1,
				Ids:      []string{id.PlayerID},
			}
			assertWatchUpdate(t, stream, expected, func(gs *sdk.GameServer) interface{} {
				return gs.Status.Players
			})

			ok, err = l.IsPlayerConnected(context.Background(), id)
			assert.NoError(t, err)
			assert.True(t, ok.Bool, "player should be connected")

			list, err = l.GetConnectedPlayers(context.Background(), e)
			assert.NoError(t, err)
			assert.Equal(t, []string{id.PlayerID}, list.List)

			// add same player
			ok, err = l.PlayerConnect(context.Background(), id)
			assert.NoError(t, err)
			assert.False(t, ok.Bool, "Player already exists")

			count, err = l.GetPlayerCount(context.Background(), e)
			assert.NoError(t, err)
			assert.Equal(t, int64(1), count.Count)
			assertNoWatchUpdate(t, stream)

			list, err = l.GetConnectedPlayers(context.Background(), e)
			assert.NoError(t, err)
			assert.Equal(t, []string{id.PlayerID}, list.List)

			// should return an error if we try to add another, since we're at capacity
			nopePlayer := &alpha.PlayerID{PlayerID: "nope"}
			_, err = l.PlayerConnect(context.Background(), nopePlayer)
			assert.EqualError(t, err, "Players are already at capacity")

			ok, err = l.IsPlayerConnected(context.Background(), nopePlayer)
			assert.NoError(t, err)
			assert.False(t, ok.Bool)

			// disconnect a player
			ok, err = l.PlayerDisconnect(context.Background(), id)
			assert.NoError(t, err)
			assert.True(t, ok.Bool, "Player should be removed")
			count, err = l.GetPlayerCount(context.Background(), e)
			assert.NoError(t, err)
			assert.Equal(t, int64(0), count.Count)

			expected = &sdk.GameServer_Status_PlayerStatus{
				Count:    0,
				Capacity: 1,
				Ids:      []string{},
			}
			assertWatchUpdate(t, stream, expected, func(gs *sdk.GameServer) interface{} {
				return gs.Status.Players
			})

			list, err = l.GetConnectedPlayers(context.Background(), e)
			assert.NoError(t, err)
			assert.Empty(t, list.List)

			// remove same player
			ok, err = l.PlayerDisconnect(context.Background(), id)
			assert.NoError(t, err)
			assert.False(t, ok.Bool, "Player already be gone")
			count, err = l.GetPlayerCount(context.Background(), e)
			assert.NoError(t, err)
			assert.Equal(t, int64(0), count.Count)
			assertNoWatchUpdate(t, stream)
		})
	}
}

func TestLocalSDKServerGetCounter(t *testing.T) {
	t.Parallel()

	runtime.FeatureTestMutex.Lock()
	defer runtime.FeatureTestMutex.Unlock()
	assert.NoError(t, runtime.ParseFeatures(string(runtime.FeatureCountsAndLists)+"=true"))

	name := "sessions"
	count := int64(1)
	capacity := int64(100)
	counters := map[string]agonesv1.CounterStatus{
		name: {Count: count, Capacity: capacity},
	}
	fixture := &agonesv1.GameServer{
		ObjectMeta: metav1.ObjectMeta{Name: "stuff"},
		Status: agonesv1.GameServerStatus{
			Counters: counters,
		},
	}

	path, err := gsToTmpFile(fixture)
	assert.NoError(t, err)
	l, err := NewLocalSDKServer(path, "")
	assert.Nil(t, err)

	stream := newGameServerMockStream()
	go func() {
		err := l.WatchGameServer(&sdk.Empty{}, stream)
		assert.Nil(t, err)
	}()
	assertInitialWatchUpdate(t, stream)

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

	want := &alpha.Counter{Name: name, Count: count, Capacity: capacity}
	got, err := l.GetCounter(context.Background(), &alpha.GetCounterRequest{Name: name})
	assert.NoError(t, err)
	if diff := cmp.Diff(want, got, protocmp.Transform()); diff != "" {
		t.Errorf("unexpected difference:\n%v", diff)
	}
}

func TestLocalSDKServerUpdateCounter(t *testing.T) {
	t.Parallel()

	runtime.FeatureTestMutex.Lock()
	defer runtime.FeatureTestMutex.Unlock()
	assert.NoError(t, runtime.ParseFeatures(string(runtime.FeatureCountsAndLists)+"=true"))

	counterName := "sessions"
	counters := map[string]agonesv1.CounterStatus{
		counterName: {Count: 1, Capacity: 100},
	}
	fixture := &agonesv1.GameServer{
		ObjectMeta: metav1.ObjectMeta{Name: "stuff"},
		Status: agonesv1.GameServerStatus{
			Counters: counters,
		},
	}

	path, err := gsToTmpFile(fixture)
	assert.NoError(t, err)
	l, err := NewLocalSDKServer(path, "")
	assert.Nil(t, err)

	stream := newGameServerMockStream()
	go func() {
		err := l.WatchGameServer(&sdk.Empty{}, stream)
		assert.Nil(t, err)
	}()
	assertInitialWatchUpdate(t, stream)

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

	// Check UpdateCounter only updates fields in the FieldMask, and that the default value for Count
	// is applied (since not specified in the request Counter object).
	got, err := l.UpdateCounter(context.Background(), &alpha.UpdateCounterRequest{
		Counter: &alpha.Counter{
			Name:     counterName,
			Capacity: int64(999),
		},
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"count"}},
	})
	want := &alpha.Counter{
		Name:     counterName,
		Count:    int64(0),
		Capacity: int64(100),
	}
	assert.NoError(t, err)
	if diff := cmp.Diff(want, got, protocmp.Transform()); diff != "" {
		t.Errorf("unexpected difference:\n%v", diff)
	}

	// Confirm updated Counter has been properly written to the GameServer
	got, err = l.GetCounter(context.Background(), &alpha.GetCounterRequest{Name: counterName})
	assert.NoError(t, err)
	if diff := cmp.Diff(want, got, protocmp.Transform()); diff != "" {
		t.Errorf("unexpected difference:\n%v", diff)
	}
}

func TestLocalSDKServerGetList(t *testing.T) {
	t.Parallel()

	runtime.FeatureTestMutex.Lock()
	defer runtime.FeatureTestMutex.Unlock()
	assert.NoError(t, runtime.ParseFeatures(string(runtime.FeatureCountsAndLists)+"=true"))

	name := "games"
	capacity := int64(100)
	values := []string{"game1", "game2"}
	lists := map[string]agonesv1.ListStatus{
		name: {Capacity: capacity, Values: values},
	}
	fixture := &agonesv1.GameServer{
		ObjectMeta: metav1.ObjectMeta{Name: "stuff"},
		Status: agonesv1.GameServerStatus{
			Lists: lists,
		},
	}

	path, err := gsToTmpFile(fixture)
	assert.NoError(t, err)
	l, err := NewLocalSDKServer(path, "")
	assert.Nil(t, err)

	stream := newGameServerMockStream()
	go func() {
		err := l.WatchGameServer(&sdk.Empty{}, stream)
		assert.Nil(t, err)
	}()
	assertInitialWatchUpdate(t, stream)

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

	want := &alpha.List{Name: name, Capacity: capacity, Values: values}
	got, err := l.GetList(context.Background(), &alpha.GetListRequest{Name: name})
	assert.NoError(t, err)
	if diff := cmp.Diff(want, got, protocmp.Transform()); diff != "" {
		t.Errorf("unexpected difference:\n%v", diff)
	}
}

func TestLocalSDKServerUpdateList(t *testing.T) {
	t.Parallel()

	runtime.FeatureTestMutex.Lock()
	defer runtime.FeatureTestMutex.Unlock()
	assert.NoError(t, runtime.ParseFeatures(string(runtime.FeatureCountsAndLists)+"=true"))

	name := "games"
	lists := map[string]agonesv1.ListStatus{
		name: {Capacity: 100, Values: []string{"game1", "game2"}},
	}
	fixture := &agonesv1.GameServer{
		ObjectMeta: metav1.ObjectMeta{Name: "stuff"},
		Status: agonesv1.GameServerStatus{
			Lists: lists,
		},
	}

	path, err := gsToTmpFile(fixture)
	assert.NoError(t, err)
	l, err := NewLocalSDKServer(path, "")
	assert.Nil(t, err)

	stream := newGameServerMockStream()
	go func() {
		err := l.WatchGameServer(&sdk.Empty{}, stream)
		assert.Nil(t, err)
	}()
	assertInitialWatchUpdate(t, stream)

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

	// Check UpdateList only overwrites fields in the FieldMask, and that the default value for "values"
	// is applied (since not specified in the request List object).
	got, err := l.UpdateList(context.Background(), &alpha.UpdateListRequest{
		List: &alpha.List{
			Name:     name,
			Capacity: int64(10),
		},
		UpdateMask: &fieldmaskpb.FieldMask{
			Paths: []string{"values"},
		},
	})
	want := &alpha.List{
		Name:     name,
		Capacity: int64(100),
		Values:   []string{},
	}
	assert.NoError(t, err)
	if diff := cmp.Diff(want, got, protocmp.Transform()); diff != "" {
		t.Errorf("unexpected difference:\n%v", diff)
	}

	// Confirm updated List has been properly written to the GameServer
	got, err = l.GetList(context.Background(), &alpha.GetListRequest{Name: name})
	assert.NoError(t, err)
	if diff := cmp.Diff(want, got, protocmp.Transform()); diff != "" {
		t.Errorf("unexpected difference:\n%v", diff)
	}
}

func TestLocalSDKServerAddListValue(t *testing.T) {
	t.Parallel()

	runtime.FeatureTestMutex.Lock()
	defer runtime.FeatureTestMutex.Unlock()
	assert.NoError(t, runtime.ParseFeatures(string(runtime.FeatureCountsAndLists)+"=true"))

	name := "lemmings"
	capacity := int64(100)
	lists := map[string]agonesv1.ListStatus{
		name: {Capacity: capacity, Values: []string{"lemming1", "lemming2"}},
	}
	fixture := &agonesv1.GameServer{
		ObjectMeta: metav1.ObjectMeta{Name: "stuff"},
		Status: agonesv1.GameServerStatus{
			Lists: lists,
		},
	}

	path, err := gsToTmpFile(fixture)
	assert.NoError(t, err)
	l, err := NewLocalSDKServer(path, "")
	assert.Nil(t, err)

	stream := newGameServerMockStream()
	go func() {
		err := l.WatchGameServer(&sdk.Empty{}, stream)
		assert.Nil(t, err)
	}()
	assertInitialWatchUpdate(t, stream)

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

	// Add new unique value to list
	got, err := l.AddListValue(context.Background(), &alpha.AddListValueRequest{Name: name, Value: "lemming3"})
	want := &alpha.List{Name: name, Capacity: capacity, Values: []string{"lemming1", "lemming2", "lemming3"}}
	assert.NoError(t, err)
	if diff := cmp.Diff(want, got, protocmp.Transform()); diff != "" {
		t.Errorf("unexpected difference:\n%v", diff)
	}

	// Confirm updated List has been properly written to the GameServer
	got, err = l.GetList(context.Background(), &alpha.GetListRequest{Name: name})
	assert.NoError(t, err)
	if diff := cmp.Diff(want, got, protocmp.Transform()); diff != "" {
		t.Errorf("unexpected difference:\n%v", diff)
	}
}

func TestLocalSDKServerRemoveListValue(t *testing.T) {
	t.Parallel()

	runtime.FeatureTestMutex.Lock()
	defer runtime.FeatureTestMutex.Unlock()
	assert.NoError(t, runtime.ParseFeatures(string(runtime.FeatureCountsAndLists)+"=true"))

	name := "players"
	capacity := int64(100)
	lists := map[string]agonesv1.ListStatus{
		name: {Capacity: capacity, Values: []string{"player1", "player2"}},
	}
	fixture := &agonesv1.GameServer{
		ObjectMeta: metav1.ObjectMeta{Name: "stuff"},
		Status: agonesv1.GameServerStatus{
			Lists: lists,
		},
	}

	path, err := gsToTmpFile(fixture)
	assert.NoError(t, err)
	l, err := NewLocalSDKServer(path, "")
	assert.Nil(t, err)

	stream := newGameServerMockStream()
	go func() {
		err := l.WatchGameServer(&sdk.Empty{}, stream)
		assert.Nil(t, err)
	}()
	assertInitialWatchUpdate(t, stream)

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

	// Remove existing value form list
	got, err := l.RemoveListValue(context.Background(), &alpha.RemoveListValueRequest{Name: name, Value: "player2"})
	want := &alpha.List{Name: name, Capacity: capacity, Values: []string{"player1"}}
	assert.NoError(t, err)
	if diff := cmp.Diff(want, got, protocmp.Transform()); diff != "" {
		t.Errorf("unexpected difference:\n%v", diff)
	}

	// Confirm updated List has been properly written to the GameServer
	got, err = l.GetList(context.Background(), &alpha.GetListRequest{Name: name})
	assert.NoError(t, err)
	if diff := cmp.Diff(want, got, protocmp.Transform()); diff != "" {
		t.Errorf("unexpected difference:\n%v", diff)
	}
}

// TestLocalSDKServerStateUpdates verify that SDK functions changes the state of the
// GameServer object
func TestLocalSDKServerStateUpdates(t *testing.T) {
	t.Parallel()
	l, err := NewLocalSDKServer("", "")
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

	l, err := NewLocalSDKServer("", "")
	assert.Nil(t, err)
	l.testMode = true
	l.recordRequest("")
	l.gs = &sdk.GameServer{ObjectMeta: &sdk.GameServer_ObjectMeta{Name: "empty"}}
	exampleUID := "052fb0f4-3d50-11e5-b066-42010af0d7b6"
	// field which is tested
	setAnnotation := "setannotation"
	l.gs.ObjectMeta.Uid = exampleUID

	var expected []string
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

func TestAlphaSDKConformanceFunctionality(t *testing.T) {
	t.Parallel()
	lStable, err := NewLocalSDKServer("", "")
	assert.Nil(t, err)
	v := int64(0)
	lStable.recordRequestWithValue("setplayercapacity", strconv.FormatInt(v, 10), "PlayerCapacity")
	lStable.recordRequestWithValue("isplayerconnected", "", "PlayerIDs")

	runtime.FeatureTestMutex.Lock()
	defer runtime.FeatureTestMutex.Unlock()

	assert.NoError(t, runtime.ParseFeatures(string(runtime.FeaturePlayerTracking)+"=true"))
	l, err := NewLocalSDKServer("", "")
	assert.Nil(t, err)
	l.testMode = true
	l.recordRequestWithValue("setplayercapacity", strconv.FormatInt(v, 10), "PlayerCapacity")
	l.recordRequestWithValue("isplayerconnected", "", "PlayerIDs")

}

func gsToTmpFile(gs *agonesv1.GameServer) (string, error) {
	file, err := os.CreateTemp(os.TempDir(), "gameserver-")
	if err != nil {
		return file.Name(), err
	}

	err = json.NewEncoder(file).Encode(gs)
	return file.Name(), err
}

// assertWatchUpdate checks the values of an update message when a GameServer value has been changed
func assertWatchUpdate(t *testing.T, stream *gameServerMockStream, expected interface{}, actual func(gs *sdk.GameServer) interface{}) {
	select {
	case msg := <-stream.msgs:
		assert.Equal(t, expected, actual(msg))
	case <-time.After(20 * time.Second):
		assert.Fail(t, "timeout on receiving messages")
	}
}

// assertNoWatchUpdate checks that no update message has been sent for changes to the GameServer
func assertNoWatchUpdate(t *testing.T, stream *gameServerMockStream) {
	select {
	case <-stream.msgs:
		assert.Fail(t, "should not get a message")
	case <-time.After(time.Second):
	}
}

// assertInitialWatchUpdate checks that the initial GameServer state is sent immediately after WatchGameServer
func assertInitialWatchUpdate(t *testing.T, stream *gameServerMockStream) {
	select {
	case <-stream.msgs:
	case <-time.After(time.Second):
		assert.Fail(t, "timeout on receiving initial message")
	}
}
