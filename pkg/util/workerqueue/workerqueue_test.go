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

package workerqueue

import (
	"testing"
	"time"

	"io/ioutil"
	"net/http"
	"net/http/httptest"

	"github.com/heptiolabs/healthcheck"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
)

func TestWorkerQueueRun(t *testing.T) {
	t.Parallel()

	received := make(chan string)
	defer close(received)

	syncHandler := func(name string) error {
		assert.Equal(t, "default/test", name)
		received <- name
		return nil
	}

	wq := NewWorkerQueue(syncHandler, logrus.WithField("source", "test"), "test")
	stop := make(chan struct{})
	defer close(stop)

	go wq.Run(1, stop)

	// no change, should be no value
	select {
	case <-received:
		assert.Fail(t, "should not have received value")
	case <-time.After(1 * time.Second):
	}

	wq.Enqueue(cache.ExplicitKey("default/test"))

	select {
	case <-received:
	case <-time.After(5 * time.Second):
		assert.Fail(t, "should have received value")
	}
}

func TestWorkerQueueHealthy(t *testing.T) {
	t.Parallel()

	done := make(chan struct{})
	handler := func(string) error {
		<-done
		return nil
	}
	wq := NewWorkerQueue(handler, logrus.WithField("source", "test"), "test")
	wq.Enqueue(cache.ExplicitKey("default/test"))

	stop := make(chan struct{})
	go wq.Run(1, stop)

	// Yield to the scheduler to ensure the worker queue goroutine can run.
	err := wait.Poll(100*time.Millisecond, 3*time.Second, func() (done bool, err error) {
		if (wq.RunCount() == 1) && wq.Healthy() == nil {
			return true, nil
		}

		return false, nil
	})
	assert.Nil(t, err)

	close(done) // Ensure the handler no longer blocks.
	close(stop) // Stop the worker queue.

	// Yield to the scheduler again to ensure the worker queue goroutine can
	// finish.
	err = wait.Poll(100*time.Millisecond, 3*time.Second, func() (done bool, err error) {
		if (wq.RunCount() == 0) && wq.Healthy() != nil {
			return true, nil
		}

		return false, nil
	})
	assert.Nil(t, err)
}

func TestWorkQueueHealthCheck(t *testing.T) {
	t.Parallel()

	health := healthcheck.NewHandler()
	handler := func(string) error {
		return nil
	}
	wq := NewWorkerQueue(handler, logrus.WithField("source", "test"), "test")
	health.AddLivenessCheck("test", wq.Healthy)

	server := httptest.NewServer(health)
	defer server.Close()

	stop := make(chan struct{})
	go wq.Run(1, stop)

	url := server.URL + "/live"

	f := func(t *testing.T, url string, status int) {

		// sometimes the http server takes a bit to start up
		err := wait.PollImmediate(time.Second, 5*time.Second, func() (bool, error) {
			resp, err := http.Get(url)
			assert.Nil(t, err)
			defer resp.Body.Close() // nolint: errcheck

			if status != resp.StatusCode {
				return false, nil
			}

			body, err := ioutil.ReadAll(resp.Body)
			assert.Nil(t, err)
			assert.Equal(t, status, resp.StatusCode)
			assert.Equal(t, []byte("{}\n"), body)

			return true, nil
		})

		assert.Nil(t, err)
	}

	f(t, url, http.StatusOK)

	close(stop)
	// closing can take a short while
	err := wait.PollImmediate(time.Second, 5*time.Second, func() (bool, error) {
		rc := wq.RunCount()
		logrus.WithField("runcount", rc).Info("Checking run count")
		return rc == 0, nil
	})
	assert.Nil(t, err)

	// gate
	assert.Error(t, wq.Healthy())
	f(t, url, http.StatusServiceUnavailable)
}
