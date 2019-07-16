// Copyright 2018 Google LLC All Rights Reserved.
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

package sdkserver

import (
	"fmt"
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
	"agones.dev/agones/pkg/util/logfields"
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
// nolint: maligned
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
	gsLabels           map[string]string
	gsAnnotations      map[string]string
	gsState            stablev1alpha1.GameServerState
	gsUpdateMutex      sync.RWMutex
	gsWaitForSync      sync.WaitGroup
	reserveTimer       *time.Timer
	gsReserveDuration  *time.Duration
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
		gsLabels:           map[string]string{},
		gsAnnotations:      map[string]string{},
		gsUpdateMutex:      sync.RWMutex{},
		gsWaitForSync:      sync.WaitGroup{},
	}

	s.informerFactory = factory
	s.logger = runtime.NewLoggerWithType(s).WithField("gsKey", namespace+"/"+gameServerName)

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

	// we haven't synced yet
	s.gsWaitForSync.Add(1)
	s.workerqueue = workerqueue.NewWorkerQueue(
		s.syncGameServer,
		s.logger,
		logfields.GameServerKey,
		strings.Join([]string{stable.GroupName, s.namespace, s.gameServerName}, "."))

	s.logger.Info("created GameServer sidecar")

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
	if !cache.WaitForCacheSync(stop, s.gameServerSynced) {
		return errors.New("failed to wait for caches to sync")
	}
	// we have the gameserver details now
	s.gsWaitForSync.Done()

	gs, err := s.gameServer()
	if err != nil {
		return err
	}

	// grab configuration details
	s.health = gs.Spec.Health
	s.logger.WithField("health", s.health).Info("setting health configuration")
	s.healthTimeout = time.Duration(gs.Spec.Health.PeriodSeconds) * time.Second
	s.initHealthLastUpdated(time.Duration(gs.Spec.Health.InitialDelaySeconds) * time.Second)

	if gs.Status.State == stablev1alpha1.GameServerStateReserved && gs.Status.ReservedUntil != nil {
		s.gsUpdateMutex.Lock()
		s.resetReserveAfter(context.Background(), time.Until(gs.Status.ReservedUntil.Time))
		s.gsUpdateMutex.Unlock()
	}

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

// syncGameServer synchronises the GameServer with the requested operations.
// The format of the key is {operation}. To prevent old operation data from
// overwriting the new one, the operation data is persisted in SDKServer.
func (s *SDKServer) syncGameServer(key string) error {
	switch Operation(key) {
	case updateState:
		return s.updateState()
	case updateLabel:
		return s.updateLabels()
	case updateAnnotation:
		return s.updateAnnotations()
	}

	return errors.Errorf("could not sync game server key: %s", key)
}

// updateState sets the GameServer Status's state to the one persisted in SDKServer,
// i.e. SDKServer.gsState.
func (s *SDKServer) updateState() error {
	s.logger.WithField("state", s.gsState).Info("Updating state")
	if len(s.gsState) == 0 {
		return errors.Errorf("could not update GameServer %s/%s to empty state", s.namespace, s.gameServerName)
	}

	gameServers := s.gameServerGetter.GameServers(s.namespace)
	gs, err := s.gameServer()
	if err != nil {
		return err
	}

	// If we are currently in shutdown/being deleted, there is no escaping.
	if gs.IsBeingDeleted() {
		s.logger.Info("GameServerState being shutdown. Skipping update.")
		return nil
	}

	// If the state is currently unhealthy, you can't go back to Ready.
	if gs.Status.State == stablev1alpha1.GameServerStateUnhealthy {
		s.logger.Info("GameServerState already unhealthy. Skipping update.")
		return nil
	}

	s.gsUpdateMutex.RLock()
	gs.Status.State = s.gsState

	// If we are setting the Reserved status, check for the duration, and set that too.
	if gs.Status.State == stablev1alpha1.GameServerStateReserved && s.gsReserveDuration != nil {
		n := metav1.NewTime(time.Now().Add(*s.gsReserveDuration))
		gs.Status.ReservedUntil = &n
	} else {
		gs.Status.ReservedUntil = nil
	}
	s.gsUpdateMutex.RUnlock()

	_, err = gameServers.Update(gs)
	if err != nil {
		return errors.Wrapf(err, "could not update GameServer %s/%s to state %s", s.namespace, s.gameServerName, gs.Status.State)
	}

	message := "SDK state change"
	level := corev1.EventTypeNormal
	// post state specific work here
	switch gs.Status.State {
	case stablev1alpha1.GameServerStateUnhealthy:
		level = corev1.EventTypeWarning
	case stablev1alpha1.GameServerStateReserved:
		s.gsUpdateMutex.Lock()
		if s.gsReserveDuration != nil {
			message += fmt.Sprintf(", for %s", s.gsReserveDuration)
			s.resetReserveAfter(context.Background(), *s.gsReserveDuration)
		}
		s.gsUpdateMutex.Unlock()
	}

	s.recorder.Event(gs, level, string(gs.Status.State), message)

	return nil
}

func (s *SDKServer) gameServer() (*stablev1alpha1.GameServer, error) {
	// this ensure that if we get requests for the gameserver before the cache has been synced,
	// they will block here until it's ready
	s.gsWaitForSync.Wait()
	gs, err := s.gameServerLister.GameServers(s.namespace).Get(s.gameServerName)
	return gs, errors.Wrapf(err, "could not retrieve GameServer %s/%s", s.namespace, s.gameServerName)
}

// updateLabels updates the labels on this GameServer to the ones persisted in SDKServer,
// i.e. SDKServer.gsLabels, with the prefix of "stable.agones.dev/sdk-"
func (s *SDKServer) updateLabels() error {
	s.logger.WithField("labels", s.gsLabels).Info("updating label")
	gs, err := s.gameServer()
	if err != nil {
		return err
	}

	gsCopy := gs.DeepCopy()

	s.gsUpdateMutex.RLock()
	if len(s.gsLabels) > 0 && gsCopy.ObjectMeta.Labels == nil {
		gsCopy.ObjectMeta.Labels = map[string]string{}
	}
	for k, v := range s.gsLabels {
		gsCopy.ObjectMeta.Labels[metadataPrefix+k] = v
	}
	s.gsUpdateMutex.RUnlock()

	_, err = s.gameServerGetter.GameServers(s.namespace).Update(gsCopy)
	return err
}

// updateAnnotations updates the Annotations on this GameServer to the ones persisted in SDKServer,
// i.e. SDKServer.gsAnnotations, with the prefix of "stable.agones.dev/sdk-"
func (s *SDKServer) updateAnnotations() error {
	s.logger.WithField("annotations", s.gsAnnotations).Info("updating annotation")
	gs, err := s.gameServer()
	if err != nil {
		return err
	}

	gsCopy := gs.DeepCopy()

	s.gsUpdateMutex.RLock()
	if len(s.gsAnnotations) > 0 && gsCopy.ObjectMeta.Annotations == nil {
		gsCopy.ObjectMeta.Annotations = map[string]string{}
	}
	for k, v := range s.gsAnnotations {
		gsCopy.ObjectMeta.Annotations[metadataPrefix+k] = v
	}
	s.gsUpdateMutex.RUnlock()

	_, err = s.gameServerGetter.GameServers(s.namespace).Update(gsCopy)
	return err
}

// enqueueState enqueue a State change request into the
// workerqueue
func (s *SDKServer) enqueueState(state stablev1alpha1.GameServerState) {
	s.gsUpdateMutex.Lock()
	s.gsState = state
	s.gsUpdateMutex.Unlock()
	s.workerqueue.Enqueue(cache.ExplicitKey(string(updateState)))
}

// Ready enters the RequestReady state change for this GameServer into
// the workqueue so it can be updated
func (s *SDKServer) Ready(ctx context.Context, e *sdk.Empty) (*sdk.Empty, error) {
	s.logger.Info("Received Ready request, adding to queue")
	s.stopReserveTimer()
	s.enqueueState(stablev1alpha1.GameServerStateRequestReady)
	return e, nil
}

// Allocate enters an Allocate state change into the workqueue, so it can be updated
func (s *SDKServer) Allocate(ctx context.Context, e *sdk.Empty) (*sdk.Empty, error) {
	s.stopReserveTimer()
	s.enqueueState(stablev1alpha1.GameServerStateAllocated)
	return e, nil
}

// Shutdown enters the Shutdown state change for this GameServer into
// the workqueue so it can be updated
func (s *SDKServer) Shutdown(ctx context.Context, e *sdk.Empty) (*sdk.Empty, error) {
	s.logger.Info("Received Shutdown request, adding to queue")
	s.stopReserveTimer()
	s.enqueueState(stablev1alpha1.GameServerStateShutdown)
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

	s.gsUpdateMutex.Lock()
	s.gsLabels[kv.Key] = kv.Value
	s.gsUpdateMutex.Unlock()

	s.workerqueue.Enqueue(cache.ExplicitKey(string(updateLabel)))
	return &sdk.Empty{}, nil
}

// SetAnnotation adds the Key/Value to be used to set the annotations with the metadataPrefix to the `GameServer`
// metdata
func (s *SDKServer) SetAnnotation(_ context.Context, kv *sdk.KeyValue) (*sdk.Empty, error) {
	s.logger.WithField("values", kv).Info("Adding SetAnnotation to queue")

	s.gsUpdateMutex.Lock()
	s.gsAnnotations[kv.Key] = kv.Value
	s.gsUpdateMutex.Unlock()

	s.workerqueue.Enqueue(cache.ExplicitKey(string(updateAnnotation)))
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

// Reserve moves this GameServer to the Reserved state for the Duration specified
func (s *SDKServer) Reserve(ctx context.Context, d *sdk.Duration) (*sdk.Empty, error) {
	s.stopReserveTimer()

	e := &sdk.Empty{}

	// 0 is forever.
	if d.Seconds > 0 {
		duration := time.Duration(d.Seconds) * time.Second
		s.gsUpdateMutex.Lock()
		s.gsReserveDuration = &duration
		s.gsUpdateMutex.Unlock()
	}

	s.logger.Info("Received Reserve request, adding to queue")
	s.enqueueState(stablev1alpha1.GameServerStateReserved)

	return e, nil
}

// resetReserveAfter will move the GameServer back to being ready after the specified duration.
// This function should be wrapped in a s.gsUpdateMutex lock when being called.
func (s *SDKServer) resetReserveAfter(ctx context.Context, duration time.Duration) {
	if s.reserveTimer != nil {
		s.reserveTimer.Stop()
	}

	s.reserveTimer = time.AfterFunc(duration, func() {
		if _, err := s.Ready(ctx, &sdk.Empty{}); err != nil {
			s.logger.WithError(errors.WithStack(err)).Error("error returning to Ready after reserved")
		}
	})
}

// stopReserveTimer stops the reserve timer. This is a no-op and safe to call if the timer is nil
func (s *SDKServer) stopReserveTimer() {
	s.gsUpdateMutex.Lock()
	defer s.gsUpdateMutex.Unlock()

	if s.reserveTimer != nil {
		s.reserveTimer.Stop()
	}
	s.gsReserveDuration = nil
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
		s.logger.WithField("gameServerName", s.gameServerName).Info("has failed health check")
		s.enqueueState(stablev1alpha1.GameServerStateUnhealthy)
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
