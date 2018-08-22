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
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"agones.dev/agones/pkg/apis/stable"
	stablev1alpha1 "agones.dev/agones/pkg/apis/stable/v1alpha1"
	"agones.dev/agones/pkg/client/clientset/versioned"
	typedv1alpha1 "agones.dev/agones/pkg/client/clientset/versioned/typed/stable/v1alpha1"
	"agones.dev/agones/pkg/client/informers/externalversions"
	"agones.dev/agones/pkg/client/listers/stable/v1alpha1"
	"agones.dev/agones/pkg/sdk"
	"agones.dev/agones/pkg/util/runtime"
	"agones.dev/agones/pkg/util/workerqueue"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/context"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/util/clock"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
)

var _ sdk.SDKServer = &SDKServer{}

// SDKServer is a gRPC server, that is meant to be a sidecar
// for a GameServer that will update the game server status on SDK requests
type SDKServer struct {
	logger             *logrus.Entry
	gameServerName     string
	namespace          string
	informerFactory    externalversions.SharedInformerFactory
	gameServerGetter   typedv1alpha1.GameServersGetter
	gameServerLister   v1alpha1.GameServerLister
	gameServerSynced   cache.InformerSynced
	server             *http.Server
	clock              clock.Clock
	health             stablev1alpha1.Health
	healthTimeout      time.Duration
	healthMutex        sync.RWMutex
	healthLastUpdated  time.Time
	healthFailureCount int32
	workerqueue        *workerqueue.WorkerQueue
	streamMutex        sync.RWMutex
	connectedStreams   []sdk.SDK_WatchGameServerServer
	stop               <-chan struct{}
	recorder           record.EventRecorder
}

// NewSDKServer creates a SDKServer that sets up an
// InClusterConfig for Kubernetes
func NewSDKServer(gameServerName, namespace string, kubeClient kubernetes.Interface,
	agonesClient versioned.Interface) (*SDKServer, error) {
	mux := http.NewServeMux()

	// limit the informer to only working with the gameserver that the sdk is attached to
	factory := externalversions.NewFilteredSharedInformerFactory(agonesClient, 30*time.Second, namespace, func(opts *metav1.ListOptions) {
		s1 := fields.OneTermEqualSelector("metadata.name", gameServerName)
		opts.FieldSelector = s1.String()
	})
	gameServers := factory.Stable().V1alpha1().GameServers()

	s := &SDKServer{
		gameServerName:   gameServerName,
		namespace:        namespace,
		gameServerGetter: agonesClient.StableV1alpha1(),
		gameServerLister: gameServers.Lister(),
		gameServerSynced: gameServers.Informer().HasSynced,
		server: &http.Server{
			Addr:    ":8080",
			Handler: mux,
		},
		clock:              clock.RealClock{},
		healthMutex:        sync.RWMutex{},
		healthFailureCount: 0,
		streamMutex:        sync.RWMutex{},
	}

	s.informerFactory = factory
	s.logger = runtime.NewLoggerWithType(s)

	gameServers.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		UpdateFunc: func(_, newObj interface{}) {
			gs := newObj.(*stablev1alpha1.GameServer)
			s.sendGameServerUpdate(gs)
		},
	})

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

	s.workerqueue = workerqueue.NewWorkerQueue(
		func(key string) error {
			return s.updateState(stablev1alpha1.State(key))
		},
		s.logger,
		strings.Join([]string{stable.GroupName, s.namespace, s.gameServerName}, "."))

	s.logger.WithField("gameServerName", s.gameServerName).WithField("namespace", s.namespace).Info("created GameServer sidecar")

	return s, nil
}

// initHealthLastUpdated adds the initial delay to now, then it will always be after `now`
// until the delay passes
func (s *SDKServer) initHealthLastUpdated(healthInitialDelay time.Duration) {
	s.healthLastUpdated = s.clock.Now().UTC().Add(healthInitialDelay)
}

// Run processes the rate limited queue.
// Will block until stop is closed
func (s *SDKServer) Run(stop <-chan struct{}) error {
	s.informerFactory.Start(stop)
	cache.WaitForCacheSync(stop, s.gameServerSynced)

	gs, err := s.gameServerLister.GameServers(s.namespace).Get(s.gameServerName)
	if err != nil {
		return errors.Wrapf(err, "error retrieving gameserver %s/%s", s.namespace, s.gameServerName)
	}

	// grab configuration details
	s.health = gs.Spec.Health
	s.logger.WithField("health", s.health).Info("setting health configuration")
	s.healthTimeout = time.Duration(gs.Spec.Health.PeriodSeconds) * time.Second
	s.initHealthLastUpdated(time.Duration(gs.Spec.Health.InitialDelaySeconds) * time.Second)

	// start health checking running
	if !s.health.Disabled {
		s.logger.Info("Starting GameServer health checking")
		go wait.Until(s.runHealth, s.healthTimeout, stop)
	}

	// then start the http endpoints
	s.logger.Info("Starting SDKServer http health check...")
	go func() {
		if err := s.server.ListenAndServe(); err != nil {
			if err == http.ErrServerClosed {
				s.logger.WithError(err).Info("health check: http server closed")
			} else {
				err = errors.Wrap(err, "Could not listen on :8080")
				runtime.HandleError(s.logger.WithError(err), err)
			}
		}
	}()
	defer s.server.Close() // nolint: errcheck

	// need this for streaming gRPC commands
	s.stop = stop
	s.workerqueue.Run(1, stop)
	return nil
}

// updateState sets the GameServer Status's state to the state
// that has been passed through
func (s *SDKServer) updateState(state stablev1alpha1.State) error {
	s.logger.WithField("state", state).Info("Updating state")
	gameServers := s.gameServerGetter.GameServers(s.namespace)
	gs, err := s.gameServerLister.GameServers(s.namespace).Get(s.gameServerName)
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
	s.workerqueue.Enqueue(cache.ExplicitKey(stablev1alpha1.RequestReady))
	return e, nil
}

// Shutdown enters the Shutdown state change for this GameServer into
// the workqueue so it can be updated
func (s *SDKServer) Shutdown(ctx context.Context, e *sdk.Empty) (*sdk.Empty, error) {
	s.logger.Info("Received Shutdown request, adding to queue")
	s.workerqueue.Enqueue(cache.ExplicitKey(stablev1alpha1.Shutdown))
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

// GetGameServer returns the current GameServer configuration and state from the backing GameServer CRD
func (s *SDKServer) GetGameServer(context.Context, *sdk.Empty) (*sdk.GameServer, error) {
	s.logger.Info("Received GetGameServer request")
	gs, err := s.gameServerLister.GameServers(s.namespace).Get(s.gameServerName)
	if err != nil {
		return nil, errors.Wrapf(err, "error retrieving gameserver %s/%s", s.namespace, s.gameServerName)
	}

	return s.convert(gs), nil
}

// WatchGameServer sends events through the stream when changes occur to the
// backing GameServer configuration / status
func (s *SDKServer) WatchGameServer(_ *sdk.Empty, stream sdk.SDK_WatchGameServerServer) error {
	s.logger.Info("Received WatchGameServer request, adding stream to connectedStreams")
	s.streamMutex.Lock()
	s.connectedStreams = append(s.connectedStreams, stream)
	s.streamMutex.Unlock()
	// don't exit until we shutdown, because that will close the stream
	<-s.stop
	return nil
}

// sendGameServerUpdate sends a watch game server event
func (s *SDKServer) sendGameServerUpdate(gs *stablev1alpha1.GameServer) {
	s.logger.Info("Sending GameServer Event to connectedStreams")

	s.streamMutex.RLock()
	defer s.streamMutex.RUnlock()

	for _, stream := range s.connectedStreams {
		err := stream.Send(s.convert(gs))
		// We essentially ignoring any disconnected streams.
		// I think this is fine, as disconnections shouldn't actually happen.
		// but we should log them, just in case they do happen, and we can track it
		if err != nil {
			s.logger.WithError(errors.WithStack(err)).
				Error("error sending game server update event")
		}
	}
}

// convert converts a K8s GameServer object, into a gRPC SDK GameServer object
func (s *SDKServer) convert(gs *stablev1alpha1.GameServer) *sdk.GameServer {
	meta := gs.ObjectMeta
	status := gs.Status
	health := gs.Spec.Health
	result := &sdk.GameServer{
		ObjectMeta: &sdk.GameServer_ObjectMeta{
			Name:              meta.Name,
			Namespace:         meta.Namespace,
			Uid:               string(meta.UID),
			ResourceVersion:   meta.ResourceVersion,
			Generation:        meta.Generation,
			CreationTimestamp: meta.CreationTimestamp.Unix(),
			Annotations:       meta.Annotations,
			Labels:            meta.Labels,
		},
		Spec: &sdk.GameServer_Spec{
			Health: &sdk.GameServer_Spec_Health{
				Disabled:            health.Disabled,
				PeriodSeconds:       health.PeriodSeconds,
				FailureThreshold:    health.FailureThreshold,
				InitialDelaySeconds: health.InitialDelaySeconds,
			},
		},
		Status: &sdk.GameServer_Status{
			State:   string(status.State),
			Address: status.Address,
		},
	}
	if meta.DeletionTimestamp != nil {
		result.ObjectMeta.DeletionTimestamp = meta.DeletionTimestamp.Unix()
	}

	// loop around and add all the ports
	for _, p := range status.Ports {
		grpcPort := &sdk.GameServer_Status_Port{
			Name: p.Name,
			Port: p.Port,
		}
		result.Status.Ports = append(result.Status.Ports, grpcPort)
	}

	return result
}

// runHealth actively checks the health, and if not
// healthy will push the Unhealthy state into the queue so
// it can be updated
func (s *SDKServer) runHealth() {
	s.checkHealth()
	if !s.healthy() {
		s.logger.WithField("gameServerName", s.gameServerName).Info("being marked as not healthy")
		s.workerqueue.Enqueue(cache.ExplicitKey(stablev1alpha1.Unhealthy))
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
	if s.health.Disabled {
		return true
	}

	s.healthMutex.RLock()
	defer s.healthMutex.RUnlock()
	return s.healthFailureCount < s.health.FailureThreshold
}
