// Copyright 2017 Google Inc. All Rights Reserved.
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

package main

import (
	"testing"
	"time"

	"github.com/agonio/agon/gameservers/sidecar/sdk"
	"github.com/agonio/agon/pkg/apis/stable/v1alpha1"
	"github.com/agonio/agon/pkg/client/clientset/versioned/fake"
	"github.com/stretchr/testify/assert"
	"golang.org/x/net/context"
	"k8s.io/apimachinery/pkg/runtime"
	k8stesting "k8s.io/client-go/testing"
)

func TestSidecarRun(t *testing.T) {
	t.Parallel()

	fixtures := map[string]struct {
		state v1alpha1.State
		f     func(*Sidecar, context.Context)
	}{
		"ready": {
			state: v1alpha1.RequestReady,
			f: func(sc *Sidecar, ctx context.Context) {
				sc.Ready(ctx, &sdk.Empty{})
			},
		},
		"shutdown": {
			state: v1alpha1.Shutdown,
			f: func(sc *Sidecar, ctx context.Context) {
				sc.Shutdown(ctx, &sdk.Empty{})
			},
		},
	}

	for k, v := range fixtures {
		t.Run(k, func(t *testing.T) {
			agonClient := &fake.Clientset{}
			done := make(chan bool)

			agonClient.AddReactor("get", "gameservers", func(action k8stesting.Action) (bool, runtime.Object, error) {
				gs := &v1alpha1.GameServer{
					Status: v1alpha1.GameServerStatus{
						State: v1alpha1.Starting,
					},
				}
				return true, gs, nil
			})
			agonClient.AddReactor("update", "gameservers", func(action k8stesting.Action) (bool, runtime.Object, error) {
				defer close(done)
				ua := action.(k8stesting.UpdateAction)
				gs := ua.GetObject().(*v1alpha1.GameServer)

				assert.Equal(t, v.state, gs.Status.State)

				return true, gs, nil
			})

			sc := &Sidecar{
				gameServerName:   "test",
				namespace:        "default",
				gameServerGetter: agonClient.StableV1alpha1(),
			}
			sc.queue = sc.newWorkQueue()

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
		})
	}
}
