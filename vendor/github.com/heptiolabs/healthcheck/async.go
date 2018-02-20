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
	"errors"
	"time"
)

// ErrNoData is returned if the first call of an Async() wrapped Check has not
// yet returned.
var ErrNoData = errors.New("no data yet")

// Async converts a Check into an asynchronous check that runs in a background
// goroutine at a fixed interval. The check is called at a fixed rate, not with
// a fixed delay between invocations. If your check takes longer than the
// interval to execute, the next execution will happen immediately.
//
// Note: if you need to clean up the background goroutine, use AsyncWithContext().
func Async(check Check, interval time.Duration) Check {
	return AsyncWithContext(context.Background(), check, interval)
}

// AsyncWithContext converts a Check into an asynchronous check that runs in a
// background goroutine at a fixed interval. The check is called at a fixed
// rate, not with a fixed delay between invocations. If your check takes longer
// than the interval to execute, the next execution will happen immediately.
//
// Note: if you don't need to cancel execution (because this runs forever), use Async()
func AsyncWithContext(ctx context.Context, check Check, interval time.Duration) Check {
	// create a chan that will buffer the most recent check result
	result := make(chan error, 1)

	// fill it with ErrNoData so we'll start in an initially failing state
	// (we don't want to be ready/live until we've actually executed the check
	// once, but that might be slow).
	result <- ErrNoData

	// make a wrapper that runs the check, and swaps out the current head of
	// the channel with the latest result
	update := func() {
		err := check()
		<-result
		result <- err
	}

	// spawn a background goroutine to run the check
	go func() {
		// call once right away (time.Tick() doesn't always tick immediately
		// but we want an initial result as soon as possible)
		update()

		// loop forever or until the context is canceled
		ticker := time.Tick(interval)
		for {
			select {
			case <-ticker:
				update()
			case <-ctx.Done():
				return
			}
		}
	}()

	// return a Check function that closes over our result and mutex
	return func() error {
		// peek at the head of the channel, then put it back
		err := <-result
		result <- err
		return err
	}
}
