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
	listersv1alpha1 "agones.dev/agones/pkg/client/listers/stable/v1alpha1"
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
	k8sv1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
)

// Operation is a synchronisation action
type Operation string

const (
	updateState      Operation = "updateState"
	updateLabel      Operation = "updateLabel"
	updateAnnotation Operation = "updateAnnotation"
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
	gameServerLister   listersv1alpha1.GameServerLister
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
	eventBroadcaster.StartRecordingToSink(&k8sv1.EventSinkImpl{Interface: kubeClient.CoreV1().Events("")})
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
		s.syncGameServer,
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

// syncGameServer synchronises the GameServer with the
// requested operations
// takes a key in the format of {operation}/{data}
func (s *SDKServer) syncGameServer(key string) error {
	op := strings.Split(key, "/")
	rest := op[1:]

	switch Operation(op[0]) {
	case updateState:
		return s.syncState(rest)
	case updateLabel:
		return s.syncLabel(rest)
	case updateAnnotation:
		return s.syncAnnotation(rest)
	}

	return errors.Errorf("could not sync game server key: %s", key)
}

// syncState converts the string array into values for updateState
func (s *SDKServer) syncState(rest []string) error {
	if len(rest) == 0 {
		return errors.New("could not sync state, as not state provided")
	}

	return s.updateState(stablev1alpha1.State(rest[0]))
}

// updateState sets the GameServer Status's state to the state
// that has been passed through
func (s *SDKServer) updateState(state stablev1alpha1.State) error {
	s.logger.WithField("state", state).Info("Updating state")
	gameServers := s.gameServerGetter.GameServers(s.namespace)
	gs, err := s.gameServer()
	if err != nil {
		return err
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

func (s *SDKServer) gameServer() (*stablev1alpha1.GameServer, error) {
	gs, err := s.gameServerLister.GameServers(s.namespace).Get(s.gameServerName)
	return gs, errors.Wrapf(err, "could not retrieve GameServer %s/%s", s.namespace, s.gameServerName)
}

// syncLabel converts the string array values into values for
// updateLabel
func (s *SDKServer) syncLabel(rest []string) error {
	if len(rest) < 2 {
		return errors.Errorf("could not sync label: %#v", rest)
	}

	return s.updateLabel(rest[0], rest[1])
}

// updateLabel updates the label on this GameServer, with the prefix of
// "stable.agones.dev/sdk-"
func (s *SDKServer) updateLabel(key, value string) error {
	s.logger.WithField("key", key).WithField("value", value).Info("updating label")
	gs, err := s.gameServer()
	if err != nil {
		return err
	}

	gsCopy := gs.DeepCopy()
	if gsCopy.ObjectMeta.Labels == nil {
		gsCopy.ObjectMeta.Labels = map[string]string{}
	}
	gsCopy.ObjectMeta.Labels[metadataPrefix+key] = value

	_, err = s.gameServerGetter.GameServers(s.namespace).Update(gsCopy)
	return err
}

// syncAnnotation converts the string array values into values for
// updateAnnotation
func (s *SDKServer) syncAnnotation(rest []string) error {
	if len(rest) < 2 {
		return errors.Errorf("could not sync annotation: %#v", rest)
	}

	return s.updateAnnotation(rest[0], rest[1])
}

// updateAnnotation updates the Annotation on this GameServer, with the prefix of
// "stable.agones.dev/sdk-"
func (s *SDKServer) updateAnnotation(key, value string) error {
	gs, err := s.gameServer()
	if err != nil {
		return err
	}

	gsCopy := gs.DeepCopy()
	if gsCopy.ObjectMeta.Annotations == nil {
		gsCopy.ObjectMeta.Annotations = map[string]string{}
	}
	gsCopy.ObjectMeta.Annotations[metadataPrefix+key] = value

	_, err = s.gameServerGetter.GameServers(s.namespace).Update(gsCopy)
	return err
}

// enqueueState enqueue a State change request into the
// workerqueue
func (s *SDKServer) enqueueState(state stablev1alpha1.State) {
	key := string(updateState) + "/" + string(state)
	s.workerqueue.Enqueue(cache.ExplicitKey(key))
}

// Ready enters the RequestReady state change for this GameServer into
// the workqueue so it can be updated
func (s *SDKServer) Ready(ctx context.Context, e *sdk.Empty) (*sdk.Empty, error) {
	s.logger.Info("Received Ready request, adding to queue")
	s.enqueueState(stablev1alpha1.RequestReady)
	return e, nil
}

// Shutdown enters the Shutdown state change for this GameServer into
// the workqueue so it can be updated
func (s *SDKServer) Shutdown(ctx context.Context, e *sdk.Empty) (*sdk.Empty, error) {
	s.logger.Info("Received Shutdown request, adding to queue")
	s.enqueueState(stablev1alpha1.Shutdown)
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

// SetLabel adds the Key/Value to be used to set the label with the metadataPrefix to the `GameServer`
// metdata
func (s *SDKServer) SetLabel(_ context.Context, kv *sdk.KeyValue) (*sdk.Empty, error) {
	s.logger.WithField("values", kv).Info("Adding SetLabel to queue")
	key := string(updateLabel) + "/" + kv.Key + "/" + kv.Value
	s.workerqueue.Enqueue(cache.ExplicitKey(key))
	return &sdk.Empty{}, nil
}

// SetAnnotation adds the Key/Value to be used to set the annotations with the metadataPrefix to the `GameServer`
// metdata
func (s *SDKServer) SetAnnotation(_ context.Context, kv *sdk.KeyValue) (*sdk.Empty, error) {
	s.logger.WithField("values", kv).Info("Adding SetLabel to queue")
	key := string(updateAnnotation) + "/" + kv.Key + "/" + kv.Value
	s.workerqueue.Enqueue(cache.ExplicitKey(key))
	return &sdk.Empty{}, nil
}

// GetGameServer returns the current GameServer configuration and state from the backing GameServer CRD
func (s *SDKServer) GetGameServer(context.Context, *sdk.Empty) (*sdk.GameServer, error) {
	s.logger.Info("Received GetGameServer request")
	gs, err := s.gameServer()
	if err != nil {
		return nil, err
	}

	return convert(gs), nil
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
		err := stream.Send(convert(gs))
		// We essentially ignoring any disconnected streams.
		// I think this is fine, as disconnections shouldn't actually happen.
		// but we should log them, just in case they do happen, and we can track it
		if err != nil {
			s.logger.WithError(errors.WithStack(err)).
				Error("error sending game server update event")
		}
	}
}

// runHealth actively checks the health, and if not
// healthy will push the Unhealthy state into the queue so
// it can be updated
func (s *SDKServer) runHealth() {
	s.checkHealth()
	if !s.healthy() {
		s.logger.WithField("gameServerName", s.gameServerName).Info("being marked as not healthy")
		s.enqueueState(stablev1alpha1.Unhealthy)
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
