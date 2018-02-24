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

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/tools/cache"
)

func TestWorkerQueueRun(t *testing.T) {
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
