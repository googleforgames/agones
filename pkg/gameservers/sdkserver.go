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
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"agones.dev/agones/pkg/apis/stable"
	stablev1alpha1 "agones.dev/agones/pkg/apis/stable/v1alpha1"
	"agones.dev/agones/pkg/client/clientset/versioned"
	typedv1alpha1 "agones.dev/agones/pkg/client/clientset/versioned/typed/stable/v1alpha1"
	"agones.dev/agones/pkg/sdk"
	"agones.dev/agones/pkg/util/runtime"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/context"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/clock"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
)

var _ sdk.SDKServer = &SDKServer{}

// SDKServer is a gRPC server, that is meant to be a sidecar
// for a GameServer that will update the game server status on SDK requests
type SDKServer struct {
	logger                 *logrus.Entry
	gameServerName         string
	namespace              string
	gameServerGetter       typedv1alpha1.GameServersGetter
	queue                  workqueue.RateLimitingInterface
	server                 *http.Server
	clock                  clock.Clock
	healthDisabled         bool
	healthTimeout          time.Duration
	healthFailureThreshold int64
	healthMutex            sync.RWMutex
	healthLastUpdated      time.Time
	healthFailureCount     int64
	recorder               record.EventRecorder
}

// NewSDKServer creates a SDKServer that sets up an
// InClusterConfig for Kubernetes
func NewSDKServer(gameServerName, namespace string,
	healthDisabled bool, healthTimeout time.Duration, healthFailureThreshold int64, healthInitialDelay time.Duration,
	kubeClient kubernetes.Interface,
	agonesClient versioned.Interface) (*SDKServer, error) {
	mux := http.NewServeMux()

	s := &SDKServer{
		gameServerName:   gameServerName,
		namespace:        namespace,
		gameServerGetter: agonesClient.StableV1alpha1(),
		server: &http.Server{
			Addr:    ":8080",
			Handler: mux,
		},
		clock:                  clock.RealClock{},
		healthDisabled:         healthDisabled,
		healthFailureThreshold: healthFailureThreshold,
		healthTimeout:          healthTimeout,
		healthMutex:            sync.RWMutex{},
		healthFailureCount:     0,
	}

	s.logger = runtime.NewLoggerWithType(s)

	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(s.logger.Infof)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: kubeClient.CoreV1().Events("")})
	s.recorder = eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: "gameserver-sidecar"})

	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write([]byte("ok"))
		if err != nil {
			s.logger.WithError(err).Error("could not send ok response on healthz")
			w.WriteHeader(http.StatusInternalServerError)
		}
	})
	mux.HandleFunc("/gshealthz", func(w http.ResponseWriter, r *http.Request) {
		if s.healthy() {
			_, err := w.Write([]byte("ok"))
			if err != nil {
				s.logger.WithError(err).Error("could not send ok response on gshealthz")
				w.WriteHeader(http.StatusInternalServerError)
			}
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
	})

	s.initHealthLastUpdated(healthInitialDelay)
	s.queue = s.newWorkQueue()

	s.logger.WithField("gameServerName", s.gameServerName).WithField("namespace", s.namespace).Info("created GameServer sidecar")

	return s, nil
}

// initHealthLastUpdated adds the initial delay to now, then it will always be after `now`
// until the delay passes
func (s *SDKServer) initHealthLastUpdated(healthInitialDelay time.Duration) {
	s.healthLastUpdated = s.clock.Now().UTC().Add(healthInitialDelay)
}

func (s *SDKServer) newWorkQueue() workqueue.RateLimitingInterface {
	return workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(),
		fmt.Sprintf("%s/%s/%s", stable.GroupName, s.namespace, s.gameServerName))
}

// Run processes the rate limited queue.
// Will block until stop is closed
func (s *SDKServer) Run(stop <-chan struct{}) {
	defer s.queue.ShutDown()

	s.logger.Info("Starting SDKServer http health check...")
	go func() {
		if err := s.server.ListenAndServe(); err != nil {
			if err == http.ErrServerClosed {
				s.logger.WithError(err).Info("health check: http server closed")
			} else {
				err := errors.Wrap(err, "Could not listen on :8080")
				runtime.HandleError(s.logger.WithError(err), err)
			}
		}
	}()
	defer s.server.Close() // nolint: errcheck

	if !s.healthDisabled {
		s.logger.Info("Starting GameServer health checking")
		go wait.Until(s.runHealth, s.healthTimeout, stop)
	}

	s.logger.Info("Starting worker")
	go wait.Until(s.runWorker, time.Second, stop)
	<-stop
	s.logger.Info("Shut down workers and health checking")
}

// runWorker is a long-running function that will continually call the
// processNextWorkItem function in order to read and process a message on the
// workqueue.
func (s *SDKServer) runWorker() {
	for s.processNextWorkItem() {
	}
}

func (s *SDKServer) processNextWorkItem() bool {
	obj, quit := s.queue.Get()
	if quit {
		return false
	}
	defer s.queue.Done(obj)

	s.logger.WithField("obj", obj).Info("Processing obj")

	var state stablev1alpha1.State
	var ok bool
	if state, ok = obj.(stablev1alpha1.State); !ok {
		runtime.HandleError(s.logger.WithField("obj", obj), errors.Errorf("expected State in queue, but got %T", obj))
		// this is a bad entry, we don't want to reprocess
		s.queue.Forget(obj)
		return true
	}

	if err := s.updateState(state); err != nil {
		// we don't forget here, because we want this to be retried via the queue
		runtime.HandleError(s.logger.WithField("obj", obj), err)
		s.queue.AddRateLimited(obj)
		return true
	}

	s.queue.Forget(obj)
	return true
}

// updateState sets the GameServer Status's state to the state
// that has been passed through
func (s *SDKServer) updateState(state stablev1alpha1.State) error {
	s.logger.WithField("state", state).Info("Updating state")
	gameServers := s.gameServerGetter.GameServers(s.namespace)
	gs, err := gameServers.Get(s.gameServerName, metav1.GetOptions{})
	if err != nil {
		return errors.Wrapf(err, "could not retrieve GameServer %s/%s", s.namespace, s.gameServerName)
	}

	// if the state is currently unhealthy, you can't go back to Ready
	if gs.Status.State == stablev1alpha1.Unhealthy {
		s.logger.Info("State already unhealthy. Skipping update.")
		return nil
	}

	gs.Status.State = state
	_, err = gameServers.Update(gs)

	// state specific work here
	if gs.Status.State == stablev1alpha1.Unhealthy {
		s.recorder.Event(gs, corev1.EventTypeWarning, string(gs.Status.State), "No longer healthy")
	}

	return errors.Wrapf(err, "could not update GameServer %s/%s to state %s", s.namespace, s.gameServerName, state)
}

// Ready enters the RequestReady state change for this GameServer into
// the workqueue so it can be updated
func (s *SDKServer) Ready(ctx context.Context, e *sdk.Empty) (*sdk.Empty, error) {
	s.logger.Info("Received Ready request, adding to queue")
	s.queue.AddRateLimited(stablev1alpha1.RequestReady)
	return e, nil
}

// Shutdown enters the Shutdown state change for this GameServer into
// the workqueue so it can be updated
func (s *SDKServer) Shutdown(ctx context.Context, e *sdk.Empty) (*sdk.Empty, error) {
	s.logger.Info("Received Shutdown request, adding to queue")
	s.queue.AddRateLimited(stablev1alpha1.Shutdown)
	return e, nil
}

// Health receives each health ping, and tracks the last time the health
// check was received, to track if a GameServer is healthy
func (s *SDKServer) Health(stream sdk.SDK_HealthServer) error {
	for {
		_, err := stream.Recv()
		if err == io.EOF {
			s.logger.Info("Health stream closed.")
			return stream.SendAndClose(&sdk.Empty{})
		}
		if err != nil {
			return errors.Wrap(err, "Error with Health check")
		}
		s.logger.Info("Health Ping Received")
		s.touchHealthLastUpdated()
	}
}

// runHealth actively checks the health, and if not
// healthy will push the Unhealthy state into the queue so
// it can be updated
func (s *SDKServer) runHealth() {
	s.checkHealth()
	if !s.healthy() {
		s.logger.WithField("gameServerName", s.gameServerName).Info("being marked as not healthy")
		s.queue.AddRateLimited(stablev1alpha1.Unhealthy)
	}
}

// touchHealthLastUpdated sets the healthLastUpdated
// value to now in UTC
func (s *SDKServer) touchHealthLastUpdated() {
	s.healthMutex.Lock()
	defer s.healthMutex.Unlock()
	s.healthLastUpdated = s.clock.Now().UTC()
	s.healthFailureCount = 0
}

// checkHealth checks the healthLastUpdated value
// and if it is outside the timeout value, logger and
// count a failure
func (s *SDKServer) checkHealth() {
	timeout := s.healthLastUpdated.Add(s.healthTimeout)
	if timeout.Before(s.clock.Now().UTC()) {
		s.healthMutex.Lock()
		defer s.healthMutex.Unlock()
		s.healthFailureCount++
		s.logger.WithField("failureCount", s.healthFailureCount).Infof("GameServer Health Check failed")
	}
}

// healthy returns if the GameServer is
// currently healthy or not based on the configured
// failure count vs failure threshold
func (s *SDKServer) healthy() bool {
	if s.healthDisabled {
		return true
	}

	s.healthMutex.RLock()
	defer s.healthMutex.RUnlock()
	return s.healthFailureCount < s.healthFailureThreshold
}
