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
	"net"
	"testing"
	"time"
)

func TestTimeout(t *testing.T) {
	tooSlow := Timeout(func() error {
		time.Sleep(10 * time.Millisecond)
		return nil
	}, 1*time.Millisecond)
	err := tooSlow()
	if _, isTimeoutError := err.(timeoutError); !isTimeoutError {
		t.Errorf("expected a TimeoutError, got %v", err)
	}

	if netErr, ok := err.(net.Error); !ok || !netErr.Timeout() {
		t.Errorf("expected Timeout() to be true, got %v", err)
	}

	if netErr, ok := err.(net.Error); !ok || !netErr.Temporary() {
		t.Errorf("expected Temporary() to be true, got %v", err)
	}

	notTooSlow := Timeout(func() error {
		time.Sleep(1 * time.Millisecond)
		return nil
	}, 10*time.Millisecond)
	if err := notTooSlow(); err != nil {
		t.Errorf("expected success, got %v", err)
	}
}
