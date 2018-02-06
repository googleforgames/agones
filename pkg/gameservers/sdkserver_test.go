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
	"net/http"
	"sync"
	"testing"
	"time"

	"agones.dev/agones/pkg/apis/stable/v1alpha1"
	"agones.dev/agones/pkg/sdk"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"golang.org/x/net/context"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/clock"
	"k8s.io/apimachinery/pkg/util/wait"
	k8stesting "k8s.io/client-go/testing"
)

func TestSidecarRun(t *testing.T) {
	fixtures := map[string]struct {
		state      v1alpha1.State
		f          func(*SDKServer, context.Context)
		recordings []string
	}{
		"ready": {
			state: v1alpha1.RequestReady,
			f: func(sc *SDKServer, ctx context.Context) {
				sc.Ready(ctx, &sdk.Empty{})
			},
		},
		"shutdown": {
			state: v1alpha1.Shutdown,
			f: func(sc *SDKServer, ctx context.Context) {
				sc.Shutdown(ctx, &sdk.Empty{})
			},
		},
		"unhealthy": {
			state: v1alpha1.Unhealthy,
			f: func(sc *SDKServer, ctx context.Context) {
				// we have a 1 second timeout
				time.Sleep(2 * time.Second)
			},
			recordings: []string{string(v1alpha1.Unhealthy)},
		},
	}

	for k, v := range fixtures {
		t.Run(k, func(t *testing.T) {
			m := newMocks()
			done := make(chan bool)

			m.agonesClient.AddReactor("get", "gameservers", func(action k8stesting.Action) (bool, runtime.Object, error) {
				gs := &v1alpha1.GameServer{
					Status: v1alpha1.GameServerStatus{
						State: v1alpha1.Starting,
					},
				}
				return true, gs, nil
			})
			m.agonesClient.AddReactor("update", "gameservers", func(action k8stesting.Action) (bool, runtime.Object, error) {
				defer close(done)
				ua := action.(k8stesting.UpdateAction)
				gs := ua.GetObject().(*v1alpha1.GameServer)

				assert.Equal(t, v.state, gs.Status.State)

				return true, gs, nil
			})

			sc, err := NewSDKServer("test", "default",
				false, time.Second, 1, 0, m.kubeClient, m.agonesClient)
			sc.recorder = m.fakeRecorder

			assert.Nil(t, err)

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			go sc.Run(ctx.Done())
			v.f(sc, ctx)
			timeout := time.After(10 * time.Second)

			select {
			case <-done:
			case <-timeout:
				assert.Fail(t, "Timeout on Run")
			}

			for _, str := range v.recordings {
				assert.Contains(t, <-m.fakeRecorder.Events, str)
			}
		})
	}
}

func TestSidecarUpdateState(t *testing.T) {
	t.Parallel()

	t.Run("ignore state change when unhealthy", func(t *testing.T) {
		m := newMocks()
		sc, err := defaultSidecar(m)
		assert.Nil(t, err)

		updated := false

		m.agonesClient.AddReactor("get", "gameservers", func(action k8stesting.Action) (bool, runtime.Object, error) {
			gs := &v1alpha1.GameServer{
				Status: v1alpha1.GameServerStatus{
					State: v1alpha1.Unhealthy,
				},
			}
			return true, gs, nil
		})
		m.agonesClient.AddReactor("update", "gameservers", func(action k8stesting.Action) (bool, runtime.Object, error) {
			updated = true
			return true, nil, nil
		})

		err = sc.updateState(v1alpha1.Ready)
		assert.Nil(t, err)
		assert.False(t, updated)
	})
}

func TestSidecarHealthLastUpdated(t *testing.T) {
	t.Parallel()
	now := time.Now().UTC()
	m := newMocks()

	sc, err := defaultSidecar(m)
	assert.Nil(t, err)
	sc.healthDisabled = false
	fc := clock.NewFakeClock(now)
	sc.clock = fc

	stream := newMockStream()

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		err := sc.Health(stream) // nolint: vetshadow
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

func TestSidecarHealthy(t *testing.T) {
	t.Parallel()

	m := newMocks()
	sc, err := defaultSidecar(m)
	assert.Nil(t, err)

	now := time.Now().UTC()
	fc := clock.NewFakeClock(now)
	sc.clock = fc

	stream := newMockStream()

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		err := sc.Health(stream) // nolint: vetshadow
		assert.Nil(t, err)
		wg.Done()
	}()

	fixtures := map[string]struct {
		disabled        bool
		timeAdd         time.Duration
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
			sc.healthDisabled = v.disabled
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
		sc.healthDisabled = false
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
		sc.healthDisabled = false
		sc.healthFailureThreshold = 3
		fc.SetTime(time.Now().UTC())
		sc.initHealthLastUpdated(0)
		sc.healthFailureCount = 0

		sc.checkHealth()
		assert.True(t, sc.healthy())
		assert.Equal(t, int64(0), sc.healthFailureCount)

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
	m := newMocks()
	sc, err := NewSDKServer("test", "default",
		false, 1*time.Second, 1, 0, m.kubeClient, m.agonesClient)
	assert.Nil(t, err)
	now := time.Now().Add(time.Hour).UTC()
	fc := clock.NewFakeClock(now)
	// now we control time - so slow machines won't fail anymore
	sc.clock = fc
	sc.healthLastUpdated = now
	sc.healthFailureCount = 0

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go sc.Run(ctx.Done())

	testHTTPHealth(t, "http://localhost:8080/healthz", "ok", http.StatusOK)
	testHTTPHealth(t, "http://localhost:8080/gshealthz", "ok", http.StatusOK)
	step := 2 * time.Second
	fc.Step(step)
	time.Sleep(step)
	testHTTPHealth(t, "http://localhost:8080/gshealthz", "", http.StatusInternalServerError)
}

func defaultSidecar(mocks mocks) (*SDKServer, error) {
	return NewSDKServer("test", "default",
		true, 5*time.Second, 1, 0, mocks.kubeClient, mocks.agonesClient)
}

func waitForMessage(sc *SDKServer) error {
	return wait.PollImmediate(time.Second, 5*time.Second, func() (done bool, err error) {
		sc.healthMutex.RLock()
		defer sc.healthMutex.RUnlock()
		return sc.clock.Now().UTC() == sc.healthLastUpdated, nil
	})
}
