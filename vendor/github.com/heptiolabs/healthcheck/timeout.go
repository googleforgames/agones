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
	"fmt"
	"time"
)

// TimeoutError is the error returned when a Timeout-wrapped Check takes too long
type timeoutError time.Duration

func (e timeoutError) Error() string {
	return fmt.Sprintf("timed out after %s", time.Duration(e).String())
}

// Timeout returns whether this error is a timeout (always true for timeoutError)
func (e timeoutError) Timeout() bool {
	return true
}

// Temporary returns whether this error is temporary (always true for timeoutError)
func (e timeoutError) Temporary() bool {
	return true
}

// Timeout adds a timeout to a Check. If the underlying check takes longer than
// the timeout, it returns an error.
func Timeout(check Check, timeout time.Duration) Check {
	return func() error {
		c := make(chan error, 1)
		go func() { c <- check() }()
		select {
		case err := <-c:
			return err
		case <-time.After(timeout):
			return timeoutError(timeout)
		}
	}
}
