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
	"fmt"
	"net/http"
	"time"

	"github.com/agonio/agon/gameservers/sidecar/sdk"
	"github.com/agonio/agon/pkg/apis/stable"
	stablev1alpha1 "github.com/agonio/agon/pkg/apis/stable/v1alpha1"
	"github.com/agonio/agon/pkg/client/clientset/versioned"
	typedv1alpha1 "github.com/agonio/agon/pkg/client/clientset/versioned/typed/stable/v1alpha1"
	"github.com/agonio/agon/pkg/util/runtime"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/context"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/workqueue"
)

var _ sdk.SDKServer = &Sidecar{}

// Sidecar GameServer sidecar implementation that will update the
// game server status on SDK request
type Sidecar struct {
	gameServerName   string
	namespace        string
	gameServerGetter typedv1alpha1.GameServersGetter
	queue            workqueue.RateLimitingInterface
	server           *http.Server
}

// NewSidecar creates a Sidecar that sets up an
// InClusterConfig for Kubernetes
func NewSidecar(gameServerName, namespace string, agonClient versioned.Interface) (*Sidecar, error) {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write([]byte("ok"))
		if err != nil {
			logrus.WithError(err).Error("could not send ok response on healthz")
			w.WriteHeader(http.StatusInternalServerError)
		}
	})

	s := &Sidecar{
		gameServerName:   gameServerName,
		namespace:        namespace,
		gameServerGetter: agonClient.StableV1alpha1(),
		server: &http.Server{
			Addr:    ":8080",
			Handler: mux,
		},
	}

	s.queue = s.newWorkQueue()

	logrus.WithField("gameServerNameEnv", s.gameServerName).WithField("namespace", s.namespace).Info("created GameServer sidecar")

	return s, nil
}

func (s *Sidecar) newWorkQueue() workqueue.RateLimitingInterface {
	return workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(),
		fmt.Sprintf("%s/%s/%s", stable.GroupName, s.namespace, s.gameServerName))
}

// Run processes the rate limited queue.
// Will block until stop is closed
func (s *Sidecar) Run(stop <-chan struct{}) {
	defer s.queue.ShutDown()

	logrus.Info("Starting health check...")
	go func() {
		if err := s.server.ListenAndServe(); err != nil {
			if err == http.ErrServerClosed {
				logrus.WithError(err).Info("health check: http server closed")
			} else {
				err := errors.Wrap(err, "Could not listen on :8080")
				runtime.HandleError(logrus.WithError(err), err)
			}
		}
	}()
	defer s.server.Close() // nolint: errcheck

	logrus.Info("Starting worker")
	wait.Until(s.runWorker, time.Second, stop)
	<-stop
	logrus.Info("Shut down workers")
}

// runWorker is a long-running function that will continually call the
// processNextWorkItem function in order to read and process a message on the
// workqueue.
func (s *Sidecar) runWorker() {
	for s.processNextWorkItem() {
	}
}

func (s *Sidecar) processNextWorkItem() bool {
	obj, quit := s.queue.Get()
	if quit {
		return false
	}
	defer s.queue.Done(obj)

	logrus.WithField("obj", obj).Info("Processing obj")

	var state stablev1alpha1.State
	var ok bool
	if state, ok = obj.(stablev1alpha1.State); !ok {
		runtime.HandleError(logrus.WithField("obj", obj), errors.Errorf("expected State in queue, but got %T", obj))
		// this is a bad entry, we don't want to reprocess
		s.queue.Forget(obj)
		return true
	}

	if err := s.updateState(state); err != nil {
		// we don't forget here, because we want this to be retried via the queue
		runtime.HandleError(logrus.WithField("obj", obj), err)
		s.queue.AddRateLimited(obj)
		return true
	}

	s.queue.Forget(obj)
	return true
}

// updateState sets the GameServer Status's state to the state
// that has been passed through
func (s *Sidecar) updateState(state stablev1alpha1.State) error {
	logrus.WithField("state", state).Info("Updating state")
	gameServers := s.gameServerGetter.GameServers(s.namespace)
	gs, err := gameServers.Get(s.gameServerName, metav1.GetOptions{})
	if err != nil {
		return errors.Wrapf(err, "could not retrieve GameServer %s/%s", s.namespace, s.gameServerName)
	}
	gs.Status.State = state
	_, err = gameServers.Update(gs)

	return errors.Wrapf(err, "could not update GameServer %s/%s to state %s", s.namespace, s.gameServerName, state)
}

// Ready enters the RequestReady state change for this GameServer into
// the workqueue so it can be updated
func (s *Sidecar) Ready(ctx context.Context, e *sdk.Empty) (*sdk.Empty, error) {
	logrus.Info("Received Ready request, adding to queue")
	s.queue.AddRateLimited(stablev1alpha1.RequestReady)
	return e, nil
}

// Shutdown enters the Shutdown state change for this GameServer into
// the workqueue so it can be updated
func (s *Sidecar) Shutdown(ctx context.Context, e *sdk.Empty) (*sdk.Empty, error) {
	logrus.Info("Received Shutdown request, adding to queue")
	s.queue.AddRateLimited(stablev1alpha1.Shutdown)
	return e, nil
}
