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
	"net/http"
	"strconv"
	"sync"
	"testing"
	"time"

	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	"agones.dev/agones/pkg/sdk"
	"agones.dev/agones/pkg/sdk/alpha"
	agtesting "agones.dev/agones/pkg/testing"
	agruntime "agones.dev/agones/pkg/util/runtime"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/context"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/clock"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"
	k8stesting "k8s.io/client-go/testing"
	"k8s.io/client-go/tools/cache"
)

func TestSidecarRun(t *testing.T) {
	t.Parallel()

	type expected struct {
		state       agonesv1.GameServerState
		labels      map[string]string
		annotations map[string]string
		recordings  []string
	}

	fixtures := map[string]struct {
		f        func(*SDKServer, context.Context)
		expected expected
	}{
		"ready": {
			f: func(sc *SDKServer, ctx context.Context) {
				sc.Ready(ctx, &sdk.Empty{}) // nolint: errcheck
			},
			expected: expected{
				state:      agonesv1.GameServerStateRequestReady,
				recordings: []string{"Normal " + string(agonesv1.GameServerStateRequestReady)},
			},
		},
		"shutdown": {
			f: func(sc *SDKServer, ctx context.Context) {
				sc.Shutdown(ctx, &sdk.Empty{}) // nolint: errcheck
			},
			expected: expected{
				state:      agonesv1.GameServerStateShutdown,
				recordings: []string{"Normal " + string(agonesv1.GameServerStateShutdown)},
			},
		},
		"unhealthy": {
			f: func(sc *SDKServer, ctx context.Context) {
				// we have a 1 second timeout
				time.Sleep(2 * time.Second)
			},
			expected: expected{
				state:      agonesv1.GameServerStateUnhealthy,
				recordings: []string{"Warning " + string(agonesv1.GameServerStateUnhealthy)},
			},
		},
		"label": {
			f: func(sc *SDKServer, ctx context.Context) {
				_, err := sc.SetLabel(ctx, &sdk.KeyValue{Key: "foo", Value: "value-foo"})
				assert.Nil(t, err)
				_, err = sc.SetLabel(ctx, &sdk.KeyValue{Key: "bar", Value: "value-bar"})
				assert.Nil(t, err)
			},
			expected: expected{
				labels: map[string]string{
					metadataPrefix + "foo": "value-foo",
					metadataPrefix + "bar": "value-bar"},
			},
		},
		"annotation": {
			f: func(sc *SDKServer, ctx context.Context) {
				_, err := sc.SetAnnotation(ctx, &sdk.KeyValue{Key: "test-1", Value: "annotation-1"})
				assert.Nil(t, err)
				_, err = sc.SetAnnotation(ctx, &sdk.KeyValue{Key: "test-2", Value: "annotation-2"})
				assert.Nil(t, err)
			},
			expected: expected{
				annotations: map[string]string{
					metadataPrefix + "test-1": "annotation-1",
					metadataPrefix + "test-2": "annotation-2"},
			},
		},
		"allocated": {
			f: func(sc *SDKServer, ctx context.Context) {
				_, err := sc.Allocate(ctx, &sdk.Empty{})
				assert.NoError(t, err)
			},
			expected: expected{
				state:      agonesv1.GameServerStateAllocated,
				recordings: []string{string(agonesv1.GameServerStateAllocated)},
			},
		},
		"reserved": {
			f: func(sc *SDKServer, ctx context.Context) {
				_, err := sc.Reserve(ctx, &sdk.Duration{Seconds: 10})
				assert.NoError(t, err)
			},
			expected: expected{
				state:      agonesv1.GameServerStateReserved,
				recordings: []string{string(agonesv1.GameServerStateReserved)},
			},
		},
	}

	for k, v := range fixtures {
		t.Run(k, func(t *testing.T) {
			m := agtesting.NewMocks()
			done := make(chan bool)

			m.AgonesClient.AddReactor("list", "gameservers", func(action k8stesting.Action) (bool, runtime.Object, error) {
				gs := agonesv1.GameServer{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test", Namespace: "default",
					},
					Spec: agonesv1.GameServerSpec{
						Health: agonesv1.Health{Disabled: false, FailureThreshold: 1, PeriodSeconds: 1, InitialDelaySeconds: 0},
					},
					Status: agonesv1.GameServerStatus{
						State: agonesv1.GameServerStateStarting,
					},
				}
				gs.ApplyDefaults()
				return true, &agonesv1.GameServerList{Items: []agonesv1.GameServer{gs}}, nil
			})
			m.AgonesClient.AddReactor("update", "gameservers", func(action k8stesting.Action) (bool, runtime.Object, error) {
				defer close(done)
				ua := action.(k8stesting.UpdateAction)
				gs := ua.GetObject().(*agonesv1.GameServer)

				if v.expected.state != "" {
					assert.Equal(t, v.expected.state, gs.Status.State)
				}

				for label, value := range v.expected.labels {
					assert.Equal(t, value, gs.ObjectMeta.Labels[label])
				}
				for ann, value := range v.expected.annotations {
					assert.Equal(t, value, gs.ObjectMeta.Annotations[ann])
				}

				return true, gs, nil
			})

			sc, err := NewSDKServer("test", "default", m.KubeClient, m.AgonesClient)
			stop := make(chan struct{})
			defer close(stop)
			sc.informerFactory.Start(stop)
			assert.True(t, cache.WaitForCacheSync(stop, sc.gameServerSynced))

			assert.Nil(t, err)
			sc.recorder = m.FakeRecorder

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			wg := sync.WaitGroup{}
			wg.Add(1)

			go func() {
				err := sc.Run(ctx.Done())
				assert.Nil(t, err)
				wg.Done()
			}()
			v.f(sc, ctx)

			select {
			case <-done:
			case <-time.After(10 * time.Second):
				assert.Fail(t, "Timeout on Run")
			}

			logrus.Info("attempting to find event recording")
			for _, str := range v.expected.recordings {
				agtesting.AssertEventContains(t, m.FakeRecorder.Events, str)
			}

			cancel()
			wg.Wait()
		})
	}
}

func TestSDKServerSyncGameServer(t *testing.T) {
	t.Parallel()

	type expected struct {
		state       agonesv1.GameServerState
		labels      map[string]string
		annotations map[string]string
	}

	type scData struct {
		gsState       agonesv1.GameServerState
		gsLabels      map[string]string
		gsAnnotations map[string]string
	}

	fixtures := map[string]struct {
		expected expected
		key      string
		scData   scData
	}{
		"ready": {
			key: string(updateState),
			scData: scData{
				gsState: agonesv1.GameServerStateReady,
			},
			expected: expected{
				state: agonesv1.GameServerStateReady,
			},
		},
		"label": {
			key: string(updateLabel),
			scData: scData{
				gsLabels: map[string]string{"foo": "bar"},
			},
			expected: expected{
				labels: map[string]string{metadataPrefix + "foo": "bar"},
			},
		},
		"annotation": {
			key: string(updateAnnotation),
			scData: scData{
				gsAnnotations: map[string]string{"test": "annotation"},
			},
			expected: expected{
				annotations: map[string]string{metadataPrefix + "test": "annotation"},
			},
		},
	}

	for k, v := range fixtures {
		t.Run(k, func(t *testing.T) {
			m := agtesting.NewMocks()
			sc, err := defaultSidecar(m)
			assert.Nil(t, err)

			sc.gsState = v.scData.gsState
			sc.gsLabels = v.scData.gsLabels
			sc.gsAnnotations = v.scData.gsAnnotations

			updated := false

			m.AgonesClient.AddReactor("list", "gameservers", func(action k8stesting.Action) (bool, runtime.Object, error) {
				gs := agonesv1.GameServer{ObjectMeta: metav1.ObjectMeta{
					UID:  "1234",
					Name: sc.gameServerName, Namespace: sc.namespace,
					Labels: map[string]string{}, Annotations: map[string]string{}},
				}
				return true, &agonesv1.GameServerList{Items: []agonesv1.GameServer{gs}}, nil
			})
			m.AgonesClient.AddReactor("update", "gameservers", func(action k8stesting.Action) (bool, runtime.Object, error) {
				updated = true
				ua := action.(k8stesting.UpdateAction)
				gs := ua.GetObject().(*agonesv1.GameServer)

				if v.expected.state != "" {
					assert.Equal(t, v.expected.state, gs.Status.State)
				}
				for label, value := range v.expected.labels {
					assert.Equal(t, value, gs.ObjectMeta.Labels[label])
				}
				for ann, value := range v.expected.annotations {
					assert.Equal(t, value, gs.ObjectMeta.Annotations[ann])
				}

				return true, gs, nil
			})

			stop := make(chan struct{})
			defer close(stop)
			sc.informerFactory.Start(stop)
			assert.True(t, cache.WaitForCacheSync(stop, sc.gameServerSynced))
			sc.gsWaitForSync.Done()

			err = sc.syncGameServer(v.key)
			assert.Nil(t, err)
			assert.True(t, updated, "should have updated")
		})
	}
}

func TestSidecarUpdateState(t *testing.T) {
	t.Parallel()

	fixtures := map[string]struct {
		f func(gs *agonesv1.GameServer)
	}{
		"unhealthy": {
			f: func(gs *agonesv1.GameServer) {
				gs.Status.State = agonesv1.GameServerStateUnhealthy
			},
		},
		"shutdown": {
			f: func(gs *agonesv1.GameServer) {
				gs.Status.State = agonesv1.GameServerStateShutdown
			},
		},
		"deleted": {
			f: func(gs *agonesv1.GameServer) {
				now := metav1.Now()
				gs.ObjectMeta.DeletionTimestamp = &now
			},
		},
	}

	for k, v := range fixtures {
		t.Run(k, func(t *testing.T) {
			m := agtesting.NewMocks()
			sc, err := defaultSidecar(m)
			require.NoError(t, err)
			sc.gsState = agonesv1.GameServerStateReady

			updated := false

			m.AgonesClient.AddReactor("list", "gameservers", func(action k8stesting.Action) (bool, runtime.Object, error) {
				gs := agonesv1.GameServer{
					ObjectMeta: metav1.ObjectMeta{Name: sc.gameServerName, Namespace: sc.namespace},
					Status:     agonesv1.GameServerStatus{},
				}

				// apply mutation
				v.f(&gs)

				return true, &agonesv1.GameServerList{Items: []agonesv1.GameServer{gs}}, nil
			})
			m.AgonesClient.AddReactor("update", "gameservers", func(action k8stesting.Action) (bool, runtime.Object, error) {
				updated = true
				return true, nil, nil
			})

			stop := make(chan struct{})
			defer close(stop)
			sc.informerFactory.Start(stop)
			assert.True(t, cache.WaitForCacheSync(stop, sc.gameServerSynced))
			sc.gsWaitForSync.Done()

			err = sc.updateState()
			assert.Nil(t, err)
			assert.False(t, updated)
		})
	}
}

func TestSidecarHealthLastUpdated(t *testing.T) {
	t.Parallel()
	now := time.Now().UTC()
	m := agtesting.NewMocks()

	sc, err := defaultSidecar(m)
	require.NoError(t, err)

	sc.health = agonesv1.Health{Disabled: false}
	fc := clock.NewFakeClock(now)
	sc.clock = fc

	stream := newEmptyMockStream()

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		err := sc.Health(stream)
		assert.Nil(t, err)
		wg.Done()
	}()

	// Test once with a single message
	fc.Step(3 * time.Second)
	stream.msgs <- &sdk.Empty{}

	err = waitForMessage(sc)
	assert.Nil(t, err)
	sc.healthMutex.RLock()
	assert.Equal(t, sc.clock.Now().UTC().String(), sc.healthLastUpdated.String())
	sc.healthMutex.RUnlock()

	// Test again, since the value has been set, that it is re-set
	fc.Step(3 * time.Second)
	stream.msgs <- &sdk.Empty{}
	err = waitForMessage(sc)
	assert.Nil(t, err)
	sc.healthMutex.RLock()
	assert.Equal(t, sc.clock.Now().UTC().String(), sc.healthLastUpdated.String())
	sc.healthMutex.RUnlock()

	// make sure closing doesn't change the time
	fc.Step(3 * time.Second)
	close(stream.msgs)
	assert.NotEqual(t, sc.clock.Now().UTC().String(), sc.healthLastUpdated.String())

	wg.Wait()
}

func TestSidecarUnhealthyMessage(t *testing.T) {
	t.Parallel()

	m := agtesting.NewMocks()
	sc, err := NewSDKServer("test", "default", m.KubeClient, m.AgonesClient)
	require.NoError(t, err)

	m.AgonesClient.AddReactor("list", "gameservers", func(action k8stesting.Action) (bool, runtime.Object, error) {
		gs := agonesv1.GameServer{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test", Namespace: "default",
			},
			Spec: agonesv1.GameServerSpec{},
			Status: agonesv1.GameServerStatus{
				State: agonesv1.GameServerStateStarting,
			},
		}
		gs.ApplyDefaults()
		return true, &agonesv1.GameServerList{Items: []agonesv1.GameServer{gs}}, nil
	})
	m.AgonesClient.AddReactor("update", "gameservers", func(action k8stesting.Action) (bool, runtime.Object, error) {
		ua := action.(k8stesting.UpdateAction)
		gs := ua.GetObject().(*agonesv1.GameServer)
		return true, gs, nil
	})

	stop := make(chan struct{})
	defer close(stop)

	sc.informerFactory.Start(stop)
	assert.True(t, cache.WaitForCacheSync(stop, sc.gameServerSynced))

	sc.recorder = m.FakeRecorder

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		err := sc.Run(ctx.Done())
		assert.Nil(t, err)
	}()

	// manually push through an unhealthy state change
	sc.enqueueState(agonesv1.GameServerStateUnhealthy)
	agtesting.AssertEventContains(t, m.FakeRecorder.Events, "Health check failure")
}

func TestSidecarHealthy(t *testing.T) {
	t.Parallel()

	m := agtesting.NewMocks()
	sc, err := defaultSidecar(m)
	require.NoError(t, err)

	// manually set the values
	sc.health = agonesv1.Health{FailureThreshold: 1}
	sc.healthTimeout = 5 * time.Second
	sc.initHealthLastUpdated(0 * time.Second)

	now := time.Now().UTC()
	fc := clock.NewFakeClock(now)
	sc.clock = fc

	stream := newEmptyMockStream()

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		err := sc.Health(stream)
		assert.Nil(t, err)
		wg.Done()
	}()

	fixtures := map[string]struct {
		timeAdd         time.Duration
		disabled        bool
		expectedHealthy bool
	}{
		"disabled, under timeout": {disabled: true, timeAdd: time.Second, expectedHealthy: true},
		"disabled, over timeout":  {disabled: true, timeAdd: 15 * time.Second, expectedHealthy: true},
		"enabled, under timeout":  {disabled: false, timeAdd: time.Second, expectedHealthy: true},
		"enabled, over timeout":   {disabled: false, timeAdd: 15 * time.Second, expectedHealthy: false},
	}

	for k, v := range fixtures {
		t.Run(k, func(t *testing.T) {
			logrus.WithField("test", k).Infof("Test Running")
			sc.health.Disabled = v.disabled
			fc.SetTime(time.Now().UTC())
			stream.msgs <- &sdk.Empty{}
			err = waitForMessage(sc)
			assert.Nil(t, err)

			fc.Step(v.timeAdd)
			sc.checkHealth()
			assert.Equal(t, v.expectedHealthy, sc.healthy())
		})
	}

	t.Run("initial delay", func(t *testing.T) {
		sc.health.Disabled = false
		fc.SetTime(time.Now().UTC())
		sc.initHealthLastUpdated(0)
		sc.healthFailureCount = 0
		sc.checkHealth()
		assert.True(t, sc.healthy())

		sc.initHealthLastUpdated(10 * time.Second)
		sc.checkHealth()
		assert.True(t, sc.healthy())
		fc.Step(9 * time.Second)
		sc.checkHealth()
		assert.True(t, sc.healthy())

		fc.Step(10 * time.Second)
		sc.checkHealth()
		assert.False(t, sc.healthy())
	})

	t.Run("health failure threshold", func(t *testing.T) {
		sc.health.Disabled = false
		sc.health.FailureThreshold = 3
		fc.SetTime(time.Now().UTC())
		sc.initHealthLastUpdated(0)
		sc.healthFailureCount = 0

		sc.checkHealth()
		assert.True(t, sc.healthy())
		assert.Equal(t, int32(0), sc.healthFailureCount)

		fc.Step(10 * time.Second)
		sc.checkHealth()
		assert.True(t, sc.healthy())
		sc.checkHealth()
		assert.True(t, sc.healthy())
		sc.checkHealth()
		assert.False(t, sc.healthy())

		stream.msgs <- &sdk.Empty{}
		err = waitForMessage(sc)
		assert.Nil(t, err)
		fc.Step(10 * time.Second)
		assert.True(t, sc.healthy())
	})

	close(stream.msgs)
	wg.Wait()
}

func TestSidecarHTTPHealthCheck(t *testing.T) {
	m := agtesting.NewMocks()
	sc, err := NewSDKServer("test", "default", m.KubeClient, m.AgonesClient)
	require.NoError(t, err)

	now := time.Now().Add(time.Hour).UTC()
	fc := clock.NewFakeClock(now)
	// now we control time - so slow machines won't fail anymore
	sc.clock = fc

	m.AgonesClient.AddReactor("list", "gameservers", func(action k8stesting.Action) (bool, runtime.Object, error) {
		gs := agonesv1.GameServer{
			ObjectMeta: metav1.ObjectMeta{Name: sc.gameServerName, Namespace: sc.namespace},
			Spec: agonesv1.GameServerSpec{
				Health: agonesv1.Health{Disabled: false, FailureThreshold: 1, PeriodSeconds: 1, InitialDelaySeconds: 0},
			},
		}

		return true, &agonesv1.GameServerList{Items: []agonesv1.GameServer{gs}}, nil
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	wg := sync.WaitGroup{}
	wg.Add(1)

	step := 2 * time.Second

	go func() {
		err := sc.Run(ctx.Done())
		assert.Nil(t, err)
		// gate
		assert.Equal(t, 1*time.Second, sc.healthTimeout)
		wg.Done()
	}()

	testHTTPHealth(t, "http://localhost:8080/healthz", "ok", http.StatusOK)
	testHTTPHealth(t, "http://localhost:8080/gshealthz", "ok", http.StatusOK)

	assert.Equal(t, now, sc.healthLastUpdated)

	fc.Step(step)
	time.Sleep(step)
	assert.False(t, sc.healthy())
	testHTTPHealth(t, "http://localhost:8080/gshealthz", "", http.StatusInternalServerError)
	cancel()
	wg.Wait() // wait for go routine test results.
}

func TestSDKServerGetGameServer(t *testing.T) {
	t.Parallel()

	fixture := &agonesv1.GameServer{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "default",
		},
		Status: agonesv1.GameServerStatus{
			State: agonesv1.GameServerStateReady,
		},
	}

	m := agtesting.NewMocks()
	m.AgonesClient.AddReactor("list", "gameservers", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, &agonesv1.GameServerList{Items: []agonesv1.GameServer{*fixture}}, nil
	})

	stop := make(chan struct{})
	defer close(stop)

	sc, err := defaultSidecar(m)
	require.NoError(t, err)

	sc.informerFactory.Start(stop)
	assert.True(t, cache.WaitForCacheSync(stop, sc.gameServerSynced))
	sc.gsWaitForSync.Done()

	result, err := sc.GetGameServer(context.Background(), &sdk.Empty{})
	require.NoError(t, err)
	assert.Equal(t, fixture.ObjectMeta.Name, result.ObjectMeta.Name)
	assert.Equal(t, fixture.ObjectMeta.Namespace, result.ObjectMeta.Namespace)
	assert.Equal(t, string(fixture.Status.State), result.Status.State)
}

func TestSDKServerWatchGameServer(t *testing.T) {
	t.Parallel()

	agruntime.FeatureTestMutex.Lock()
	defer agruntime.FeatureTestMutex.Unlock()
	err := agruntime.ParseFeatures(string(agruntime.FeatureSDKWatchSendOnExecute) + "=false")
	require.NoError(t, err, "Can not parse FeatureSDKWatchSendOnExecute")

	m := agtesting.NewMocks()
	sc, err := defaultSidecar(m)
	require.NoError(t, err)
	assert.Empty(t, sc.connectedStreams)

	stream := newGameServerMockStream()
	asyncWatchGameServer(t, sc, stream)
	assert.Nil(t, waitConnectedStreamCount(sc, 1))
	assert.Equal(t, stream, sc.connectedStreams[0])

	stream = newGameServerMockStream()
	asyncWatchGameServer(t, sc, stream)
	assert.Nil(t, waitConnectedStreamCount(sc, 2))
	assert.Len(t, sc.connectedStreams, 2)
	assert.Equal(t, stream, sc.connectedStreams[1])
}

func TestSDKServerWatchGameServerFeatureSDKWatchSendOnExecute(t *testing.T) {
	t.Parallel()

	agruntime.FeatureTestMutex.Lock()
	defer agruntime.FeatureTestMutex.Unlock()

	fixture := &agonesv1.GameServer{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "default",
		},
		Status: agonesv1.GameServerStatus{
			State: agonesv1.GameServerStateReady,
		},
	}

	m := agtesting.NewMocks()
	fakeWatch := watch.NewFake()
	m.AgonesClient.AddWatchReactor("gameservers", k8stesting.DefaultWatchReactor(fakeWatch, nil))

	err := agruntime.ParseFeatures(string(agruntime.FeatureSDKWatchSendOnExecute) + "=true")
	require.NoError(t, err, "Can not parse FeatureSDKWatchSendOnExecute")

	sc, err := defaultSidecar(m)
	require.NoError(t, err)
	assert.Empty(t, sc.connectedStreams)

	stop := make(chan struct{})
	defer close(stop)
	sc.informerFactory.Start(stop)

	fakeWatch.Add(fixture.DeepCopy())
	assert.True(t, cache.WaitForCacheSync(stop, sc.gameServerSynced))
	sc.gsWaitForSync.Done()

	stream := newGameServerMockStream()
	asyncWatchGameServer(t, sc, stream)

	assert.Nil(t, waitConnectedStreamCount(sc, 1))
	assert.Equal(t, stream, sc.connectedStreams[0])

	// modify for 2nd event in watch stream
	fixture.Status.State = agonesv1.GameServerStateAllocated
	fakeWatch.Modify(fixture.DeepCopy())

	totalSendCalls := 0
	running := true
	for running {
		select {
		case gs := <-stream.msgs:
			assert.Equal(t, fixture.ObjectMeta.Name, gs.ObjectMeta.Name)
			totalSendCalls++
			switch totalSendCalls {
			case 1:
				assert.Equal(t, string(agonesv1.GameServerStateReady), gs.Status.State)
			case 2:
				assert.Equal(t, string(agonesv1.GameServerStateAllocated), gs.Status.State)
			}
			// we shouldn't get more than 2, but let's put an upper bound on this
			// just in case we suddenly get way more than we expect.
			if totalSendCalls > 10 {
				assert.FailNow(t, "We should have only received two events. Got over 10 instead.")
			}
		case <-time.After(5 * time.Second):
			// we can't `break` out of the loop, hence we need `running`.
			running = false
		}
	}

	// if SDKWatchSendOnExecute feature is turned on, there are two stream.Send() calls should happen:
	// one in sendGameServerUpdate, another one in WatchGameServer.
	assert.Equal(t, 2, totalSendCalls)
}

func TestSDKServerSendGameServerUpdate(t *testing.T) {
	t.Parallel()

	agruntime.FeatureTestMutex.Lock()
	defer agruntime.FeatureTestMutex.Unlock()
	err := agruntime.ParseFeatures(string(agruntime.FeatureSDKWatchSendOnExecute) + "=false")
	require.NoError(t, err, "Can not parse FeatureSDKWatchSendOnExecute")

	m := agtesting.NewMocks()
	sc, err := defaultSidecar(m)
	require.NoError(t, err)
	assert.Empty(t, sc.connectedStreams)

	stop := make(chan struct{})
	defer close(stop)
	sc.stop = stop

	stream := newGameServerMockStream()
	asyncWatchGameServer(t, sc, stream)
	assert.Nil(t, waitConnectedStreamCount(sc, 1))

	fixture := &agonesv1.GameServer{ObjectMeta: metav1.ObjectMeta{Name: "test-server"}}

	sc.sendGameServerUpdate(fixture)

	var sdkGS *sdk.GameServer
	select {
	case sdkGS = <-stream.msgs:
	case <-time.After(3 * time.Second):
		assert.Fail(t, "Event stream should not have timed out")
	}

	assert.Equal(t, fixture.ObjectMeta.Name, sdkGS.ObjectMeta.Name)
}

func TestSDKServerUpdateEventHandler(t *testing.T) {
	t.Parallel()

	// Acquire lock in order to be sure that
	// no other parallel test turn on FeatureSDKWatchSendOnExecute
	agruntime.FeatureTestMutex.Lock()
	defer agruntime.FeatureTestMutex.Unlock()
	err := agruntime.ParseFeatures(string(agruntime.FeatureSDKWatchSendOnExecute) + "=false")
	require.NoError(t, err, "Can not parse FeatureSDKWatchSendOnExecute")

	m := agtesting.NewMocks()
	fakeWatch := watch.NewFake()
	m.AgonesClient.AddWatchReactor("gameservers", k8stesting.DefaultWatchReactor(fakeWatch, nil))

	stop := make(chan struct{})
	defer close(stop)

	sc, err := defaultSidecar(m)
	require.NoError(t, err)

	sc.informerFactory.Start(stop)
	assert.True(t, cache.WaitForCacheSync(stop, sc.gameServerSynced))
	sc.gsWaitForSync.Done()

	stream := newGameServerMockStream()
	asyncWatchGameServer(t, sc, stream)
	assert.Nil(t, waitConnectedStreamCount(sc, 1))

	fixture := &agonesv1.GameServer{ObjectMeta: metav1.ObjectMeta{Name: "test-server", Namespace: "default"},
		Spec: agonesv1.GameServerSpec{},
	}

	// need to add it before it can be modified
	fakeWatch.Add(fixture.DeepCopy())
	fakeWatch.Modify(fixture.DeepCopy())

	var sdkGS *sdk.GameServer
	select {
	case sdkGS = <-stream.msgs:
	case <-time.After(3 * time.Second):
		assert.Fail(t, "Event stream should not have timed out")
	}

	assert.NotNil(t, sdkGS)
	assert.Equal(t, fixture.ObjectMeta.Name, sdkGS.ObjectMeta.Name)
}

func TestSDKServerReserveTimeoutOnRun(t *testing.T) {
	t.Parallel()
	m := agtesting.NewMocks()

	updated := make(chan agonesv1.GameServerStatus, 1)

	m.AgonesClient.AddReactor("list", "gameservers", func(action k8stesting.Action) (bool, runtime.Object, error) {
		n := metav1.NewTime(metav1.Now().Add(time.Second))

		gs := agonesv1.GameServer{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test", Namespace: "default",
			},
			Status: agonesv1.GameServerStatus{
				State:         agonesv1.GameServerStateReserved,
				ReservedUntil: &n,
			},
		}
		gs.ApplyDefaults()
		return true, &agonesv1.GameServerList{Items: []agonesv1.GameServer{gs}}, nil
	})
	m.AgonesClient.AddReactor("update", "gameservers", func(action k8stesting.Action) (bool, runtime.Object, error) {
		ua := action.(k8stesting.UpdateAction)
		gs := ua.GetObject().(*agonesv1.GameServer)

		updated <- gs.Status

		return true, gs, nil
	})

	sc, err := defaultSidecar(m)
	require.NoError(t, err)

	stop := make(chan struct{})
	sc.informerFactory.Start(stop)
	assert.True(t, cache.WaitForCacheSync(stop, sc.gameServerSynced))

	wg := sync.WaitGroup{}
	wg.Add(1)

	go func() {
		err = sc.Run(stop)
		assert.Nil(t, err)
		wg.Done()
	}()

	select {
	case status := <-updated:
		assert.Equal(t, agonesv1.GameServerStateRequestReady, status.State)
		assert.Nil(t, status.ReservedUntil)
	case <-time.After(5 * time.Second):
		assert.Fail(t, "should have been an update")
	}

	close(stop)
	wg.Wait()
}

func TestSDKServerReserveTimeout(t *testing.T) {
	t.Parallel()
	m := agtesting.NewMocks()

	state := make(chan agonesv1.GameServerStatus, 100)
	defer close(state)

	m.AgonesClient.AddReactor("list", "gameservers", func(action k8stesting.Action) (bool, runtime.Object, error) {
		gs := agonesv1.GameServer{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test", Namespace: "default",
			},
			Spec: agonesv1.GameServerSpec{Health: agonesv1.Health{Disabled: true}},
		}
		gs.ApplyDefaults()
		return true, &agonesv1.GameServerList{Items: []agonesv1.GameServer{gs}}, nil
	})

	m.AgonesClient.AddReactor("update", "gameservers", func(action k8stesting.Action) (bool, runtime.Object, error) {
		ua := action.(k8stesting.UpdateAction)
		gs := ua.GetObject().(*agonesv1.GameServer)

		state <- gs.Status

		return true, gs, nil
	})

	sc, err := defaultSidecar(m)

	assert.NoError(t, err)
	stop := make(chan struct{})
	sc.informerFactory.Start(stop)
	assert.True(t, cache.WaitForCacheSync(stop, sc.gameServerSynced))

	wg := sync.WaitGroup{}
	wg.Add(1)

	go func() {
		err = sc.Run(stop)
		assert.Nil(t, err)
		wg.Done()
	}()

	assertStateChange := func(expected agonesv1.GameServerState, timeout time.Duration, additional func(status agonesv1.GameServerStatus)) {
		select {
		case current := <-state:
			assert.Equal(t, expected, current.State)
			additional(current)
		case <-time.After(timeout):
			assert.Fail(t, "should have gone to Reserved by now")
		}
	}
	assertReservedUntilDuration := func(d time.Duration) func(status agonesv1.GameServerStatus) {
		return func(status agonesv1.GameServerStatus) {
			assert.Equal(t, time.Now().Add(d).Round(time.Second), status.ReservedUntil.Time.Round(time.Second))
		}
	}
	assertReservedUntilNil := func(status agonesv1.GameServerStatus) {
		assert.Nil(t, status.ReservedUntil)
	}

	_, err = sc.Reserve(context.Background(), &sdk.Duration{Seconds: 3})
	assert.NoError(t, err)
	assertStateChange(agonesv1.GameServerStateReserved, 2*time.Second, assertReservedUntilDuration(3*time.Second))

	// Wait for the game server to go back to being Ready.
	assertStateChange(agonesv1.GameServerStateRequestReady, 4*time.Second, func(status agonesv1.GameServerStatus) {
		assert.Nil(t, status.ReservedUntil)
	})

	// Test that a 0 second input into Reserved, never will go back to Ready
	_, err = sc.Reserve(context.Background(), &sdk.Duration{Seconds: 0})
	assert.NoError(t, err)
	assertStateChange(agonesv1.GameServerStateReserved, 2*time.Second, assertReservedUntilNil)
	assert.False(t, sc.reserveTimer.Stop())

	// Test that a negative input into Reserved, is the same as a 0 input
	_, err = sc.Reserve(context.Background(), &sdk.Duration{Seconds: -100})
	assert.NoError(t, err)
	assertStateChange(agonesv1.GameServerStateReserved, 2*time.Second, assertReservedUntilNil)
	assert.False(t, sc.reserveTimer.Stop())

	// Test that the timer to move Reserved->Ready is reset when requesting another state.

	// Test the return to a Ready state.
	_, err = sc.Reserve(context.Background(), &sdk.Duration{Seconds: 3})
	assert.NoError(t, err)
	assertStateChange(agonesv1.GameServerStateReserved, 2*time.Second, assertReservedUntilDuration(3*time.Second))

	_, err = sc.Ready(context.Background(), &sdk.Empty{})
	assert.NoError(t, err)
	assertStateChange(agonesv1.GameServerStateRequestReady, 2*time.Second, assertReservedUntilNil)
	assert.False(t, sc.reserveTimer.Stop())

	// Test Allocated resets the timer on Reserved->Ready
	_, err = sc.Reserve(context.Background(), &sdk.Duration{Seconds: 3})
	assert.NoError(t, err)
	assertStateChange(agonesv1.GameServerStateReserved, 2*time.Second, assertReservedUntilDuration(3*time.Second))

	_, err = sc.Allocate(context.Background(), &sdk.Empty{})
	assert.NoError(t, err)
	assertStateChange(agonesv1.GameServerStateAllocated, 2*time.Second, assertReservedUntilNil)
	assert.False(t, sc.reserveTimer.Stop())

	// Test Shutdown resets the timer on Reserved->Ready
	_, err = sc.Reserve(context.Background(), &sdk.Duration{Seconds: 3})
	assert.NoError(t, err)
	assertStateChange(agonesv1.GameServerStateReserved, 2*time.Second, assertReservedUntilDuration(3*time.Second))

	_, err = sc.Shutdown(context.Background(), &sdk.Empty{})
	assert.NoError(t, err)
	assertStateChange(agonesv1.GameServerStateShutdown, 2*time.Second, assertReservedUntilNil)
	assert.False(t, sc.reserveTimer.Stop())

	close(stop)
	wg.Wait()
}

func TestSDKServerPlayerCapacity(t *testing.T) {
	t.Parallel()
	agruntime.FeatureTestMutex.Lock()
	defer agruntime.FeatureTestMutex.Unlock()

	err := agruntime.ParseFeatures(string(agruntime.FeaturePlayerTracking) + "=true")
	require.NoError(t, err, "Can not parse FeaturePlayerTracking feature")

	m := agtesting.NewMocks()
	stop := make(chan struct{})
	defer close(stop)

	sc, err := defaultSidecar(m)
	require.NoError(t, err)

	m.AgonesClient.AddReactor("list", "gameservers", func(action k8stesting.Action) (bool, runtime.Object, error) {
		gs := agonesv1.GameServer{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test", Namespace: "default",
			},
			Spec: agonesv1.GameServerSpec{
				SdkServer: agonesv1.SdkServer{
					LogLevel: "Debug",
				},
				Players: &agonesv1.PlayersSpec{
					InitialCapacity: 10,
				},
			},
		}
		gs.ApplyDefaults()
		return true, &agonesv1.GameServerList{Items: []agonesv1.GameServer{gs}}, nil
	})

	updated := make(chan int64, 10)
	m.AgonesClient.AddReactor("update", "gameservers", func(action k8stesting.Action) (bool, runtime.Object, error) {
		ua := action.(k8stesting.UpdateAction)
		gs := ua.GetObject().(*agonesv1.GameServer)
		updated <- gs.Status.Players.Capacity
		return true, gs, nil
	})

	sc.informerFactory.Start(stop)
	assert.True(t, cache.WaitForCacheSync(stop, sc.gameServerSynced))

	go func() {
		err = sc.Run(stop)
		assert.NoError(t, err)
	}()

	// check initial value comes through

	// async, so check after a period
	err = wait.Poll(time.Second, 10*time.Second, func() (bool, error) {
		count, err := sc.GetPlayerCapacity(context.Background(), &alpha.Empty{})
		return count.Count == 10, err
	})
	assert.NoError(t, err)

	// on update from the SDK, the value is available from GetPlayerCapacity
	_, err = sc.SetPlayerCapacity(context.Background(), &alpha.Count{Count: 20})
	assert.NoError(t, err)

	count, err := sc.GetPlayerCapacity(context.Background(), &alpha.Empty{})
	require.NoError(t, err)
	assert.Equal(t, int64(20), count.Count)

	// on an update, confirm that the update hits the K8s api
	select {
	case value := <-updated:
		assert.Equal(t, int64(20), value)
	case <-time.After(time.Minute):
		assert.Fail(t, "Should have been updated")
	}

	agtesting.AssertEventContains(t, m.FakeRecorder.Events, "PlayerCapacity Set to 20")
}

func TestSDKServerPlayerConnectAndDisconnect(t *testing.T) {
	t.Parallel()
	agruntime.FeatureTestMutex.Lock()
	defer agruntime.FeatureTestMutex.Unlock()

	err := agruntime.ParseFeatures(string(agruntime.FeaturePlayerTracking) + "=true")
	require.NoError(t, err, "Can not parse FeaturePlayerTracking feature")

	m := agtesting.NewMocks()
	stop := make(chan struct{})
	defer close(stop)

	sc, err := defaultSidecar(m)
	require.NoError(t, err)

	capacity := int64(3)
	m.AgonesClient.AddReactor("list", "gameservers", func(action k8stesting.Action) (bool, runtime.Object, error) {
		gs := agonesv1.GameServer{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test", Namespace: "default",
			},
			Spec: agonesv1.GameServerSpec{
				SdkServer: agonesv1.SdkServer{
					LogLevel: "Debug",
				},
				// this is here to give us a reference, so we know when sc.Run() has completed.
				Players: &agonesv1.PlayersSpec{
					InitialCapacity: capacity,
				},
			},
		}
		gs.ApplyDefaults()
		return true, &agonesv1.GameServerList{Items: []agonesv1.GameServer{gs}}, nil
	})
	updated := make(chan *agonesv1.PlayerStatus, 10)
	m.AgonesClient.AddReactor("update", "gameservers", func(action k8stesting.Action) (bool, runtime.Object, error) {
		ua := action.(k8stesting.UpdateAction)
		gs := ua.GetObject().(*agonesv1.GameServer)
		updated <- gs.Status.Players
		return true, gs, nil
	})

	sc.informerFactory.Start(stop)
	assert.True(t, cache.WaitForCacheSync(stop, sc.gameServerSynced))

	go func() {
		err = sc.Run(stop)
		assert.NoError(t, err)
	}()

	// check initial value comes through
	// async, so check after a period
	e := &alpha.Empty{}
	err = wait.Poll(time.Second, 10*time.Second, func() (bool, error) {
		count, err := sc.GetPlayerCapacity(context.Background(), e)
		return count.Count == capacity, err
	})
	assert.NoError(t, err)

	count, err := sc.GetPlayerCount(context.Background(), e)
	require.NoError(t, err)
	assert.Equal(t, int64(0), count.Count)

	list, err := sc.GetConnectedPlayers(context.Background(), e)
	require.NoError(t, err)
	assert.Empty(t, list.List)

	ok, err := sc.IsPlayerConnected(context.Background(), &alpha.PlayerID{PlayerID: "1"})
	require.NoError(t, err)
	assert.False(t, ok.Bool, "no player connected yet")

	// sdk value should always be correct, even if we send more than one update per second.
	for i := int64(0); i < capacity; i++ {
		token := strconv.FormatInt(i, 10)
		id := &alpha.PlayerID{PlayerID: token}
		ok, err := sc.PlayerConnect(context.Background(), id)
		require.NoError(t, err)
		assert.True(t, ok.Bool, "Player "+token+" should not yet be connected")

		ok, err = sc.IsPlayerConnected(context.Background(), id)
		require.NoError(t, err)
		assert.True(t, ok.Bool, "Player "+token+" should be connected")
	}
	count, err = sc.GetPlayerCount(context.Background(), e)
	require.NoError(t, err)
	assert.Equal(t, capacity, count.Count)

	list, err = sc.GetConnectedPlayers(context.Background(), e)
	require.NoError(t, err)
	assert.Equal(t, []string{"0", "1", "2"}, list.List)

	// on an update, confirm that the update hits the K8s api, only once
	select {
	case value := <-updated:
		assert.Equal(t, capacity, value.Count)
		assert.Equal(t, []string{"0", "1", "2"}, value.IDs)
	case <-time.After(5 * time.Second):
		assert.Fail(t, "Should have been updated")
	}
	agtesting.AssertEventContains(t, m.FakeRecorder.Events, "PlayerCount Set to 3")

	// confirm there was only one update
	select {
	case <-updated:
		assert.Fail(t, "There should be only one update for the player connections")
	case <-time.After(2 * time.Second):
	}

	// should return an error if we try and add another, since we're at capacity
	nopePlayer := &alpha.PlayerID{PlayerID: "nope"}
	_, err = sc.PlayerConnect(context.Background(), nopePlayer)
	assert.EqualError(t, err, "players are already at capacity")

	// sdk value should always be correct, even if we send more than one update per second.
	// let's leave one player behind
	for i := int64(0); i < capacity-1; i++ {
		token := strconv.FormatInt(i, 10)
		id := &alpha.PlayerID{PlayerID: token}
		ok, err := sc.PlayerDisconnect(context.Background(), id)
		require.NoError(t, err)
		assert.Truef(t, ok.Bool, "Player %s should be disconnected", token)

		ok, err = sc.IsPlayerConnected(context.Background(), id)
		require.NoError(t, err)
		assert.Falsef(t, ok.Bool, "Player %s should be connected", token)
	}
	count, err = sc.GetPlayerCount(context.Background(), e)
	require.NoError(t, err)
	assert.Equal(t, int64(1), count.Count)

	list, err = sc.GetConnectedPlayers(context.Background(), e)
	require.NoError(t, err)
	assert.Equal(t, []string{"2"}, list.List)

	// on an update, confirm that the update hits the K8s api, only once
	select {
	case value := <-updated:
		assert.Equal(t, int64(1), value.Count)
		assert.Equal(t, []string{"2"}, value.IDs)
	case <-time.After(5 * time.Second):
		assert.Fail(t, "Should have been updated")
	}
	agtesting.AssertEventContains(t, m.FakeRecorder.Events, "PlayerCount Set to 1")

	// confirm there was only one update
	select {
	case <-updated:
		assert.Fail(t, "There should be only one update for the player disconnections")
	case <-time.After(2 * time.Second):
	}

	// last player is still there
	ok, err = sc.IsPlayerConnected(context.Background(), &alpha.PlayerID{PlayerID: "2"})
	require.NoError(t, err)
	assert.True(t, ok.Bool, "Player 2 should be connected")

	// finally, check idempotency of connect and disconnect
	id := &alpha.PlayerID{PlayerID: "2"} // only one left behind
	ok, err = sc.PlayerConnect(context.Background(), id)
	require.NoError(t, err)
	assert.False(t, ok.Bool, "Player 2 should already be connected")
	count, err = sc.GetPlayerCount(context.Background(), e)
	require.NoError(t, err)
	assert.Equal(t, int64(1), count.Count)

	// no longer there.
	id.PlayerID = "0"
	ok, err = sc.PlayerDisconnect(context.Background(), id)
	require.NoError(t, err)
	assert.False(t, ok.Bool, "Player 2 should already be disconnected")
	count, err = sc.GetPlayerCount(context.Background(), e)
	require.NoError(t, err)
	assert.Equal(t, int64(1), count.Count)

	agtesting.AssertNoEvent(t, m.FakeRecorder.Events)

	list, err = sc.GetConnectedPlayers(context.Background(), e)
	require.NoError(t, err)
	assert.Equal(t, []string{"2"}, list.List)
}

func defaultSidecar(m agtesting.Mocks) (*SDKServer, error) {
	server, err := NewSDKServer("test", "default", m.KubeClient, m.AgonesClient)
	if err != nil {
		return server, err
	}

	server.recorder = m.FakeRecorder
	return server, err
}

func waitForMessage(sc *SDKServer) error {
	return wait.PollImmediate(time.Second, 5*time.Second, func() (done bool, err error) {
		sc.healthMutex.RLock()
		defer sc.healthMutex.RUnlock()
		return sc.clock.Now().UTC() == sc.healthLastUpdated, nil
	})
}

func waitConnectedStreamCount(sc *SDKServer, count int) error {
	return wait.PollImmediate(1*time.Second, 10*time.Second, func() (bool, error) {
		sc.streamMutex.RLock()
		defer sc.streamMutex.RUnlock()
		return len(sc.connectedStreams) == count, nil
	})
}

func asyncWatchGameServer(t *testing.T, sc *SDKServer, stream sdk.SDK_WatchGameServerServer) {
	// Note that new FeatureSDKWatchSendOnExecute feature gate
	// uses getGameServer() function and therefore WatchGameServer()
	// would block if gsWaitForSync is not Done().
	go func() {
		err := sc.WatchGameServer(&sdk.Empty{}, stream)
		assert.Nil(t, err)
	}()
}
