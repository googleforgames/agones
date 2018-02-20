// Copyright 2017 by the contributors.
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

package healthcheck

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestAsync(t *testing.T) {
	async := Async(func() error {
		time.Sleep(50 * time.Millisecond)
		return nil
	}, 1*time.Millisecond)

	// expect the first call to return ErrNoData since it takes 50ms to return the first time
	assert.EqualError(t, async(), "no data yet")

	// wait for the first run to finish
	time.Sleep(100 * time.Millisecond)

	// make sure the next call returns nil ~immediately
	start := time.Now()
	assert.NoError(t, async())
	assert.WithinDuration(t, time.Now(), start, 1*time.Millisecond,
		"expected async() to return almost immediately")
}

func TestAsyncWithContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	// start an async check that counts how many times it was called
	calls := 0
	AsyncWithContext(ctx, func() error {
		calls++
		time.Sleep(1 * time.Millisecond)
		return nil
	}, 10*time.Millisecond)

	// cancel the context which should stop things mid-flight
	cancel()

	// wait long enough for several runs to have happened
	time.Sleep(100 * time.Millisecond)

	// make sure the check was only executed roughly once
	assert.InDelta(t, calls, 1, 1)
}
