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
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"agones.dev/agones/pkg/apis/agones"
	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	"agones.dev/agones/pkg/client/clientset/versioned"
	typedv1 "agones.dev/agones/pkg/client/clientset/versioned/typed/agones/v1"
	"agones.dev/agones/pkg/client/informers/externalversions"
	listersv1 "agones.dev/agones/pkg/client/listers/agones/v1"
	"agones.dev/agones/pkg/gameserverallocations"
	"agones.dev/agones/pkg/sdk"
	"agones.dev/agones/pkg/sdk/alpha"
	"agones.dev/agones/pkg/sdk/beta"
	"agones.dev/agones/pkg/util/logfields"
	"agones.dev/agones/pkg/util/runtime"
	"agones.dev/agones/pkg/util/workerqueue"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	k8sv1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/utils/clock"
)

// Operation is a synchronisation action
type Operation string

const (
	updateState            Operation     = "updateState"
	updateLabel            Operation     = "updateLabel"
	updateAnnotation       Operation     = "updateAnnotation"
	updatePlayerCapacity   Operation     = "updatePlayerCapacity"
	updateConnectedPlayers Operation     = "updateConnectedPlayers"
	updateCounters         Operation     = "updateCounters"
	updatePeriod           time.Duration = time.Second
)

var (
	_ sdk.SDKServer   = &SDKServer{}
	_ alpha.SDKServer = &SDKServer{}
	_ beta.SDKServer  = &SDKServer{}
)

type counterUpdateRequest struct {
	// Capacity of the Counter as set by capacitySet.
	capacitySet *int64
	// Count of the Counter as set by countSet.
	countSet *int64
	// Tracks the sum of CountIncrement, CountDecrement, and/or CountSet requests from the client SDK.
	diff int64
	// Counter as retreived from the GameServer
	counter agonesv1.CounterStatus
}

// SDKServer is a gRPC server, that is meant to be a sidecar
// for a GameServer that will update the game server status on SDK requests
//
//nolint:govet // ignore fieldalignment, singleton
type SDKServer struct {
	logger              *logrus.Entry
	gameServerName      string
	namespace           string
	informerFactory     externalversions.SharedInformerFactory
	gameServerGetter    typedv1.GameServersGetter
	gameServerLister    listersv1.GameServerLister
	gameServerSynced    cache.InformerSynced
	connected           bool
	server              *http.Server
	clock               clock.Clock
	health              agonesv1.Health
	healthTimeout       time.Duration
	healthMutex         sync.RWMutex
	healthLastUpdated   time.Time
	healthFailureCount  int32
	healthChecksRunning sync.Once
	workerqueue         *workerqueue.WorkerQueue
	streamMutex         sync.RWMutex
	connectedStreams    []sdk.SDK_WatchGameServerServer
	ctx                 context.Context
	recorder            record.EventRecorder
	gsLabels            map[string]string
	gsAnnotations       map[string]string
	gsState             agonesv1.GameServerState
	gsStateChannel      chan agonesv1.GameServerState
	gsUpdateMutex       sync.RWMutex
	gsWaitForSync       sync.WaitGroup
	reserveTimer        *time.Timer
	gsReserveDuration   *time.Duration
	gsPlayerCapacity    int64
	gsConnectedPlayers  []string
	gsCounterUpdates    map[string]counterUpdateRequest
	gsCopy              *agonesv1.GameServer
}

// NewSDKServer creates a SDKServer that sets up an
// InClusterConfig for Kubernetes
func NewSDKServer(gameServerName, namespace string, kubeClient kubernetes.Interface,
	agonesClient versioned.Interface) (*SDKServer, error) {
	mux := http.NewServeMux()

	// limit the informer to only working with the gameserver that the sdk is attached to
	tweakListOptions := func(opts *metav1.ListOptions) {
		s1 := fields.OneTermEqualSelector("metadata.name", gameServerName)
		opts.FieldSelector = s1.String()
	}
	factory := externalversions.NewSharedInformerFactoryWithOptions(agonesClient, 30*time.Second, externalversions.WithNamespace(namespace), externalversions.WithTweakListOptions(tweakListOptions))
	gameServers := factory.Agones().V1().GameServers()

	s := &SDKServer{
		gameServerName:   gameServerName,
		namespace:        namespace,
		gameServerGetter: agonesClient.AgonesV1(),
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
		gsConnectedPlayers: []string{},
		gsStateChannel:     make(chan agonesv1.GameServerState, 2),
	}

	if runtime.FeatureEnabled(runtime.FeatureCountsAndLists) {
		// Once FeatureCountsAndLists is in GA, move this into SDKServer creation above.
		s.gsCounterUpdates = map[string]counterUpdateRequest{}
	}

	s.informerFactory = factory
	s.logger = runtime.NewLoggerWithType(s).WithField("gsKey", namespace+"/"+gameServerName)

	_, _ = gameServers.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		UpdateFunc: func(_, newObj interface{}) {
			gs := newObj.(*agonesv1.GameServer)
			s.sendGameServerUpdate(gs)
		},
	})

	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(s.logger.Debugf)
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
		s.ensureHealthChecksRunning()
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
		strings.Join([]string{agones.GroupName, s.namespace, s.gameServerName}, "."))

	s.logger.Info("Created GameServer sidecar")

	return s, nil
}

// Run processes the rate limited queue.
// Will block until stop is closed
func (s *SDKServer) Run(ctx context.Context) error {
	s.informerFactory.Start(ctx.Done())
	if !cache.WaitForCacheSync(ctx.Done(), s.gameServerSynced) {
		return errors.New("failed to wait for caches to sync")
	}

	// need this for streaming gRPC commands
	s.ctx = ctx
	// we have the gameserver details now
	s.gsWaitForSync.Done()

	gs, err := s.gameServer()
	if err != nil {
		return err
	}

	logLevel := agonesv1.SdkServerLogLevelInfo
	// grab configuration details
	if gs.Spec.SdkServer.LogLevel != "" {
		logLevel = gs.Spec.SdkServer.LogLevel
	}
	s.logger.WithField("logLevel", logLevel).Debug("Setting LogLevel configuration")
	level, err := logrus.ParseLevel(strings.ToLower(string(logLevel)))
	if err == nil {
		s.logger.Logger.SetLevel(level)
	} else {
		s.logger.WithError(err).Warn("Specified wrong Logging.SdkServer. Setting default loglevel - Info")
		s.logger.Logger.SetLevel(logrus.InfoLevel)
	}

	s.health = gs.Spec.Health
	s.logger.WithField("health", s.health).Debug("Setting health configuration")
	s.healthTimeout = time.Duration(gs.Spec.Health.PeriodSeconds) * time.Second
	s.touchHealthLastUpdated()

	if gs.Status.State == agonesv1.GameServerStateReserved && gs.Status.ReservedUntil != nil {
		s.gsUpdateMutex.Lock()
		s.resetReserveAfter(context.Background(), time.Until(gs.Status.ReservedUntil.Time))
		s.gsUpdateMutex.Unlock()
	}

	// populate player tracking values
	if runtime.FeatureEnabled(runtime.FeaturePlayerTracking) {
		s.gsUpdateMutex.Lock()
		if gs.Status.Players != nil {
			s.gsPlayerCapacity = gs.Status.Players.Capacity
			s.gsConnectedPlayers = gs.Status.Players.IDs
		}
		s.gsUpdateMutex.Unlock()
	}

	// then start the http endpoints
	s.logger.Debug("Starting SDKServer http health check...")
	go func() {
		if err := s.server.ListenAndServe(); err != nil {
			if err == http.ErrServerClosed {
				s.logger.WithError(err).Error("Health check: http server closed")
			} else {
				err = errors.Wrap(err, "Could not listen on :8080")
				runtime.HandleError(s.logger.WithError(err), err)
			}
		}
	}()
	defer s.server.Close() // nolint: errcheck

	s.workerqueue.Run(ctx, 1)
	return nil
}

// WaitForConnection attempts a GameServer GET every 3s until the client responds.
// This is a workaround for the informer hanging indefinitely on first LIST due
// to a flaky network to the Kubernetes service endpoint.
func (s *SDKServer) WaitForConnection(ctx context.Context) error {
	// In normal operaiton, waitForConnection is called exactly once in Run().
	// In unit tests, waitForConnection() can be called before Run() to ensure
	// that connected is true when Run() is called, otherwise the List() below
	// may race with a test that changes a mock. (Despite the fact that we drop
	// the data on the ground, the Go race detector will pereceive a data race.)
	if s.connected {
		return nil
	}

	try := 0
	return wait.PollImmediateInfiniteWithContext(ctx, 4*time.Second, func(ctx context.Context) (bool, error) {
		ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
		defer cancel()

		// Specifically use gameServerGetter since it's the raw client (gameServerLister is the informer).
		// We use List here to avoid needing permission to Get().
		_, err := s.gameServerGetter.GameServers(s.namespace).List(ctx, metav1.ListOptions{
			FieldSelector: fields.OneTermEqualSelector("metadata.name", s.gameServerName).String(),
		})
		if err != nil {
			s.logger.WithField("try", try).WithError(err).Info("Connection to Kubernetes service failed")
			try++
			return false, nil
		}
		s.logger.WithField("try", try).Info("Connection to Kubernetes service established")
		s.connected = true
		return true, nil
	})
}

// syncGameServer synchronises the GameServer with the requested operations.
// The format of the key is {operation}. To prevent old operation data from
// overwriting the new one, the operation data is persisted in SDKServer.
func (s *SDKServer) syncGameServer(ctx context.Context, key string) error {
	switch Operation(key) {
	case updateState:
		return s.updateState(ctx)
	case updateLabel:
		return s.updateLabels(ctx)
	case updateAnnotation:
		return s.updateAnnotations(ctx)
	case updatePlayerCapacity:
		return s.updatePlayerCapacity(ctx)
	case updateConnectedPlayers:
		return s.updateConnectedPlayers(ctx)
	case updateCounters:
		return s.updateCounter(ctx)
	}

	return errors.Errorf("could not sync game server key: %s", key)
}

// updateState sets the GameServer Status's state to the one persisted in SDKServer,
// i.e. SDKServer.gsState.
func (s *SDKServer) updateState(ctx context.Context) error {
	s.gsUpdateMutex.RLock()
	s.logger.WithField("state", s.gsState).Debug("Updating state")
	if len(s.gsState) == 0 {
		s.gsUpdateMutex.RUnlock()
		return errors.Errorf("could not update GameServer %s/%s to empty state", s.namespace, s.gameServerName)
	}
	s.gsUpdateMutex.RUnlock()

	gameServers := s.gameServerGetter.GameServers(s.namespace)
	gs, err := s.gameServer()
	if err != nil {
		return err
	}

	// If we are currently in shutdown/being deleted, there is no escaping.
	if gs.IsBeingDeleted() {
		s.logger.Debug("GameServerState being shutdown. Skipping update.")

		// Explicitly update gsStateChannel if current state is Shutdown since sendGameServerUpdate will not triggered.
		if s.gsState == agonesv1.GameServerStateShutdown && gs.Status.State != agonesv1.GameServerStateShutdown {
			go func() {
				s.gsStateChannel <- agonesv1.GameServerStateShutdown
			}()
		}

		return nil
	}

	// If the state is currently unhealthy, you can't go back to Ready.
	if gs.Status.State == agonesv1.GameServerStateUnhealthy {
		s.logger.Debug("GameServerState already unhealthy. Skipping update.")
		return nil
	}

	s.gsUpdateMutex.RLock()
	gsCopy := gs.DeepCopy()
	gsCopy.Status.State = s.gsState

	// If we are setting the Reserved status, check for the duration, and set that too.
	if gsCopy.Status.State == agonesv1.GameServerStateReserved && s.gsReserveDuration != nil {
		n := metav1.NewTime(time.Now().Add(*s.gsReserveDuration))
		gsCopy.Status.ReservedUntil = &n
	} else {
		gsCopy.Status.ReservedUntil = nil
	}
	s.gsUpdateMutex.RUnlock()

	// If we are setting the Allocated status, set the last-allocated annotation as well.
	if gsCopy.Status.State == agonesv1.GameServerStateAllocated {
		ts, err := s.clock.Now().MarshalText()
		if err != nil {
			return err
		}
		if gsCopy.ObjectMeta.Annotations == nil {
			gsCopy.ObjectMeta.Annotations = map[string]string{}
		}
		gsCopy.ObjectMeta.Annotations[gameserverallocations.LastAllocatedAnnotationKey] = string(ts)
	}

	gs, err = gameServers.Update(ctx, gsCopy, metav1.UpdateOptions{})
	if err != nil {
		return errors.Wrapf(err, "could not update GameServer %s/%s to state %s", s.namespace, s.gameServerName, gs.Status.State)
	}

	message := "SDK state change"
	level := corev1.EventTypeNormal
	// post state specific work here
	switch gs.Status.State {
	case agonesv1.GameServerStateUnhealthy:
		level = corev1.EventTypeWarning
		message = "Health check failure"
	case agonesv1.GameServerStateReserved:
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

// Gets the GameServer from the cache, or from the local SDKServer if that version is more recent.
func (s *SDKServer) gameServer() (*agonesv1.GameServer, error) {
	// this ensure that if we get requests for the gameserver before the cache has been synced,
	// they will block here until it's ready
	s.gsWaitForSync.Wait()
	gs, err := s.gameServerLister.GameServers(s.namespace).Get(s.gameServerName)
	if err != nil {
		return gs, errors.Wrapf(err, "could not retrieve GameServer %s/%s", s.namespace, s.gameServerName)
	}
	s.gsUpdateMutex.RLock()
	defer s.gsUpdateMutex.RUnlock()
	if s.gsCopy != nil && gs.ObjectMeta.Generation < s.gsCopy.Generation {
		return s.gsCopy, nil
	}
	return gs, nil
}

// updateLabels updates the labels on this GameServer to the ones persisted in SDKServer,
// i.e. SDKServer.gsLabels, with the prefix of "agones.dev/sdk-"
func (s *SDKServer) updateLabels(ctx context.Context) error {
	s.logger.WithField("labels", s.gsLabels).Debug("Updating label")
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

	_, err = s.gameServerGetter.GameServers(s.namespace).Update(ctx, gsCopy, metav1.UpdateOptions{})
	return err
}

// updateAnnotations updates the Annotations on this GameServer to the ones persisted in SDKServer,
// i.e. SDKServer.gsAnnotations, with the prefix of "agones.dev/sdk-"
func (s *SDKServer) updateAnnotations(ctx context.Context) error {
	s.logger.WithField("annotations", s.gsAnnotations).Debug("Updating annotation")
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

	_, err = s.gameServerGetter.GameServers(s.namespace).Update(ctx, gsCopy, metav1.UpdateOptions{})
	return err
}

// enqueueState enqueue a State change request into the
// workerqueue
func (s *SDKServer) enqueueState(state agonesv1.GameServerState) {
	s.gsUpdateMutex.Lock()
	// Update cached state, but prevent transitions out of `Unhealthy` by the SDK.
	if s.gsState != agonesv1.GameServerStateUnhealthy {
		s.gsState = state
	}
	s.gsUpdateMutex.Unlock()
	s.workerqueue.Enqueue(cache.ExplicitKey(string(updateState)))
}

// Ready enters the RequestReady state change for this GameServer into
// the workqueue so it can be updated
func (s *SDKServer) Ready(ctx context.Context, e *sdk.Empty) (*sdk.Empty, error) {
	s.logger.Debug("Received Ready request, adding to queue")
	s.stopReserveTimer()
	s.enqueueState(agonesv1.GameServerStateRequestReady)
	return e, nil
}

// Allocate enters an Allocate state change into the workqueue, so it can be updated
func (s *SDKServer) Allocate(ctx context.Context, e *sdk.Empty) (*sdk.Empty, error) {
	s.stopReserveTimer()
	s.enqueueState(agonesv1.GameServerStateAllocated)
	return e, nil
}

// Shutdown enters the Shutdown state change for this GameServer into
// the workqueue so it can be updated. If gracefulTermination feature is enabled,
// Shutdown will block on GameServer being shutdown.
func (s *SDKServer) Shutdown(ctx context.Context, e *sdk.Empty) (*sdk.Empty, error) {
	s.logger.Debug("Received Shutdown request, adding to queue")
	s.stopReserveTimer()
	s.enqueueState(agonesv1.GameServerStateShutdown)

	return e, nil
}

// Health receives each health ping, and tracks the last time the health
// check was received, to track if a GameServer is healthy
func (s *SDKServer) Health(stream sdk.SDK_HealthServer) error {
	for {
		_, err := stream.Recv()
		if err == io.EOF {
			s.logger.Debug("Health stream closed.")
			return stream.SendAndClose(&sdk.Empty{})
		}
		if err != nil {
			return errors.Wrap(err, "Error with Health check")
		}
		s.logger.Debug("Health Ping Received")
		s.touchHealthLastUpdated()
	}
}

// SetLabel adds the Key/Value to be used to set the label with the metadataPrefix to the `GameServer`
// metdata
func (s *SDKServer) SetLabel(_ context.Context, kv *sdk.KeyValue) (*sdk.Empty, error) {
	s.logger.WithField("values", kv).Debug("Adding SetLabel to queue")

	s.gsUpdateMutex.Lock()
	s.gsLabels[kv.Key] = kv.Value
	s.gsUpdateMutex.Unlock()

	s.workerqueue.Enqueue(cache.ExplicitKey(string(updateLabel)))
	return &sdk.Empty{}, nil
}

// SetAnnotation adds the Key/Value to be used to set the annotations with the metadataPrefix to the `GameServer`
// metdata
func (s *SDKServer) SetAnnotation(_ context.Context, kv *sdk.KeyValue) (*sdk.Empty, error) {
	s.logger.WithField("values", kv).Debug("Adding SetAnnotation to queue")

	s.gsUpdateMutex.Lock()
	s.gsAnnotations[kv.Key] = kv.Value
	s.gsUpdateMutex.Unlock()

	s.workerqueue.Enqueue(cache.ExplicitKey(string(updateAnnotation)))
	return &sdk.Empty{}, nil
}

// GetGameServer returns the current GameServer configuration and state from the backing GameServer CRD
func (s *SDKServer) GetGameServer(context.Context, *sdk.Empty) (*sdk.GameServer, error) {
	s.logger.Debug("Received GetGameServer request")
	gs, err := s.gameServer()
	if err != nil {
		return nil, err
	}
	return convert(gs), nil
}

// WatchGameServer sends events through the stream when changes occur to the
// backing GameServer configuration / status
func (s *SDKServer) WatchGameServer(_ *sdk.Empty, stream sdk.SDK_WatchGameServerServer) error {
	s.logger.Debug("Received WatchGameServer request, adding stream to connectedStreams")

	gs, err := s.GetGameServer(context.Background(), &sdk.Empty{})
	if err != nil {
		return err
	}

	if err := stream.Send(gs); err != nil {
		return err
	}

	s.streamMutex.Lock()
	s.connectedStreams = append(s.connectedStreams, stream)
	s.streamMutex.Unlock()
	// don't exit until we shutdown, because that will close the stream
	<-s.ctx.Done()
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

	s.logger.Debug("Received Reserve request, adding to queue")
	s.enqueueState(agonesv1.GameServerStateReserved)

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

// PlayerConnect should be called when a player connects.
// [Stage:Alpha]
// [FeatureFlag:PlayerTracking]
func (s *SDKServer) PlayerConnect(ctx context.Context, id *alpha.PlayerID) (*alpha.Bool, error) {
	if !runtime.FeatureEnabled(runtime.FeaturePlayerTracking) {
		return &alpha.Bool{Bool: false}, errors.Errorf("%s not enabled", runtime.FeaturePlayerTracking)
	}
	s.logger.WithField("playerID", id.PlayerID).Debug("Player Connected")

	s.gsUpdateMutex.Lock()
	defer s.gsUpdateMutex.Unlock()

	// the player is already connected, return false.
	for _, playerID := range s.gsConnectedPlayers {
		if playerID == id.PlayerID {
			return &alpha.Bool{Bool: false}, nil
		}
	}

	if int64(len(s.gsConnectedPlayers)) >= s.gsPlayerCapacity {
		return &alpha.Bool{Bool: false}, errors.New("players are already at capacity")
	}

	// let's retain the original order, as it should be a smaller patch on data change
	s.gsConnectedPlayers = append(s.gsConnectedPlayers, id.PlayerID)
	s.workerqueue.EnqueueAfter(cache.ExplicitKey(string(updateConnectedPlayers)), updatePeriod)

	return &alpha.Bool{Bool: true}, nil
}

// PlayerDisconnect should be called when a player disconnects.
// [Stage:Alpha]
// [FeatureFlag:PlayerTracking]
func (s *SDKServer) PlayerDisconnect(ctx context.Context, id *alpha.PlayerID) (*alpha.Bool, error) {
	if !runtime.FeatureEnabled(runtime.FeaturePlayerTracking) {
		return &alpha.Bool{Bool: false}, errors.Errorf("%s not enabled", runtime.FeaturePlayerTracking)
	}
	s.logger.WithField("playerID", id.PlayerID).Debug("Player Disconnected")

	s.gsUpdateMutex.Lock()
	defer s.gsUpdateMutex.Unlock()

	found := -1
	for i, playerID := range s.gsConnectedPlayers {
		if playerID == id.PlayerID {
			found = i
			break
		}
	}
	if found == -1 {
		return &alpha.Bool{Bool: false}, nil
	}

	// let's retain the original order, as it should be a smaller patch on data change
	s.gsConnectedPlayers = append(s.gsConnectedPlayers[:found], s.gsConnectedPlayers[found+1:]...)
	s.workerqueue.EnqueueAfter(cache.ExplicitKey(string(updateConnectedPlayers)), updatePeriod)

	return &alpha.Bool{Bool: true}, nil
}

// IsPlayerConnected returns if the playerID is currently connected to the GameServer.
// This is always accurate, even if the value hasn’t been updated to the GameServer status yet.
// [Stage:Alpha]
// [FeatureFlag:PlayerTracking]
func (s *SDKServer) IsPlayerConnected(ctx context.Context, id *alpha.PlayerID) (*alpha.Bool, error) {
	if !runtime.FeatureEnabled(runtime.FeaturePlayerTracking) {
		return &alpha.Bool{Bool: false}, errors.Errorf("%s not enabled", runtime.FeaturePlayerTracking)
	}
	s.gsUpdateMutex.RLock()
	defer s.gsUpdateMutex.RUnlock()

	result := &alpha.Bool{Bool: false}

	for _, playerID := range s.gsConnectedPlayers {
		if playerID == id.PlayerID {
			result.Bool = true
			break
		}
	}

	return result, nil
}

// GetConnectedPlayers returns the list of the currently connected player ids.
// This is always accurate, even if the value hasn’t been updated to the GameServer status yet.
// [Stage:Alpha]
// [FeatureFlag:PlayerTracking]
func (s *SDKServer) GetConnectedPlayers(c context.Context, empty *alpha.Empty) (*alpha.PlayerIDList, error) {
	if !runtime.FeatureEnabled(runtime.FeaturePlayerTracking) {
		return nil, errors.Errorf("%s not enabled", runtime.FeaturePlayerTracking)
	}
	s.gsUpdateMutex.RLock()
	defer s.gsUpdateMutex.RUnlock()

	return &alpha.PlayerIDList{List: s.gsConnectedPlayers}, nil
}

// GetPlayerCount returns the current player count.
// [Stage:Alpha]
// [FeatureFlag:PlayerTracking]
func (s *SDKServer) GetPlayerCount(ctx context.Context, _ *alpha.Empty) (*alpha.Count, error) {
	if !runtime.FeatureEnabled(runtime.FeaturePlayerTracking) {
		return nil, errors.Errorf("%s not enabled", runtime.FeaturePlayerTracking)
	}
	s.gsUpdateMutex.RLock()
	defer s.gsUpdateMutex.RUnlock()
	return &alpha.Count{Count: int64(len(s.gsConnectedPlayers))}, nil
}

// SetPlayerCapacity to change the game server's player capacity.
// [Stage:Alpha]
// [FeatureFlag:PlayerTracking]
func (s *SDKServer) SetPlayerCapacity(ctx context.Context, count *alpha.Count) (*alpha.Empty, error) {
	if !runtime.FeatureEnabled(runtime.FeaturePlayerTracking) {
		return nil, errors.Errorf("%s not enabled", runtime.FeaturePlayerTracking)
	}
	s.gsUpdateMutex.Lock()
	s.gsPlayerCapacity = count.Count
	s.gsUpdateMutex.Unlock()
	s.workerqueue.Enqueue(cache.ExplicitKey(string(updatePlayerCapacity)))

	return &alpha.Empty{}, nil
}

// GetPlayerCapacity returns the current player capacity, as set by SDK.SetPlayerCapacity()
// [Stage:Alpha]
// [FeatureFlag:PlayerTracking]
func (s *SDKServer) GetPlayerCapacity(ctx context.Context, _ *alpha.Empty) (*alpha.Count, error) {
	if !runtime.FeatureEnabled(runtime.FeaturePlayerTracking) {
		return nil, errors.Errorf("%s not enabled", runtime.FeaturePlayerTracking)
	}
	s.gsUpdateMutex.RLock()
	defer s.gsUpdateMutex.RUnlock()
	return &alpha.Count{Count: s.gsPlayerCapacity}, nil
}

// GetCounter returns a Counter. Returns NOT_FOUND if the counter does not exist.
// [Stage:Alpha]
// [FeatureFlag:CountsAndLists]
func (s *SDKServer) GetCounter(ctx context.Context, in *alpha.GetCounterRequest) (*alpha.Counter, error) {
	if !runtime.FeatureEnabled(runtime.FeatureCountsAndLists) {
		return nil, errors.Errorf("%s not enabled", runtime.FeatureCountsAndLists)
	}

	s.logger.WithField("name", in.Name).Debug("Getting Counter")

	gs, err := s.gameServer()
	if err != nil {
		return nil, err
	}

	s.gsUpdateMutex.RLock()
	defer s.gsUpdateMutex.RUnlock()

	counter, ok := gs.Status.Counters[in.Name]
	if !ok {
		return nil, errors.Errorf("NOT_FOUND. %s Counter not found", in.Name)
	}
	s.logger.WithField("Get Counter", counter).Debugf("Got Counter %s", in.Name)
	protoCounter := alpha.Counter{Name: in.Name, Count: counter.Count, Capacity: counter.Capacity}
	// If there are batched changes that have not yet been applied, apply them to the Counter.
	// This does NOT validate batched the changes.
	if counterUpdate, ok := s.gsCounterUpdates[in.Name]; ok {
		if counterUpdate.capacitySet != nil {
			protoCounter.Capacity = *counterUpdate.capacitySet
		}
		if counterUpdate.countSet != nil {
			protoCounter.Count = *counterUpdate.countSet
		}
		protoCounter.Count += counterUpdate.diff
		if protoCounter.Count < 0 {
			protoCounter.Count = 0
			s.logger.Debug("truncating Count in Get Counter request to 0")
		}
		if protoCounter.Count > protoCounter.Capacity {
			protoCounter.Count = protoCounter.Capacity
			s.logger.Debug("truncating Count in Get Counter request to Capacity")
		}
		s.logger.WithField("Get Counter", counter).Debugf("Applied Batched Counter Updates %v", counterUpdate)
	}

	return &protoCounter, nil
}

// UpdateCounter collapses all UpdateCounterRequests for a given Counter into a single request.
// Returns NOT_FOUND if the Counter does not exist (name cannot be updated).
// Returns OUT_OF_RANGE if the Count is out of range [0,Capacity].
// [Stage:Alpha]
// [FeatureFlag:CountsAndLists]
func (s *SDKServer) UpdateCounter(ctx context.Context, in *alpha.UpdateCounterRequest) (*alpha.Counter, error) {
	if !runtime.FeatureEnabled(runtime.FeatureCountsAndLists) {
		return nil, errors.Errorf("%s not enabled", runtime.FeatureCountsAndLists)
	}

	if in.CounterUpdateRequest == nil {
		return nil, errors.Errorf("INVALID_ARGUMENT. CounterUpdateRequest: %v cannot be nil", in.CounterUpdateRequest)
	}

	s.logger.WithField("name", in.CounterUpdateRequest.Name).Debug("Update Counter Request")

	gs, err := s.gameServer()
	if err != nil {
		return nil, err
	}

	s.gsUpdateMutex.Lock()
	defer s.gsUpdateMutex.Unlock()

	// Check if we already have a batch request started for this Counter. If not, add new request to
	// the gsCounterUpdates map.
	name := in.CounterUpdateRequest.Name
	batchCounter := s.gsCounterUpdates[name]

	counter, ok := gs.Status.Counters[name]
	// We didn't find the Counter named key in the gameserver.
	if !ok {
		return nil, errors.Errorf("NOT_FOUND. %s Counter not found", name)
	}

	batchCounter.counter = *counter.DeepCopy()

	switch {
	case in.CounterUpdateRequest.CountDiff != 0: // Update based on if Client call is CountIncrement or CountDecrement
		count := batchCounter.counter.Count
		if batchCounter.countSet != nil {
			count = *batchCounter.countSet
		}
		count += batchCounter.diff + in.CounterUpdateRequest.CountDiff
		// Verify that 0 <= Count >= Capacity
		capacity := batchCounter.counter.Capacity
		if batchCounter.capacitySet != nil {
			capacity = *batchCounter.capacitySet
		}
		if count < 0 || count > capacity {
			return nil, errors.Errorf("OUT_OF_RANGE. Count must be within range [0,Capacity]. Found Count: %d, Capacity: %d", count, capacity)
		}
		batchCounter.diff += in.CounterUpdateRequest.CountDiff
	case in.CounterUpdateRequest.Count != nil: // Update based on if Client call is CountSet
		// Verify that 0 <= Count >= Capacity
		countSet := in.CounterUpdateRequest.Count.GetValue()
		capacity := batchCounter.counter.Capacity
		if batchCounter.capacitySet != nil {
			capacity = *batchCounter.capacitySet
		}
		if countSet < 0 || countSet > capacity {
			return nil, errors.Errorf("OUT_OF_RANGE. Count must be within range [0,Capacity]. Found Count: %d, Capacity: %d", countSet, capacity)
		}
		batchCounter.countSet = &countSet
		// Clear any previous CountIncrement or CountDecrement requests, and add the CountSet as the first item.
		batchCounter.diff = 0
	case in.CounterUpdateRequest.Capacity != nil: // Updated based on if client call is CapacitySet
		if in.CounterUpdateRequest.Capacity.GetValue() < 0 {
			return nil, errors.Errorf("OUT_OF_RANGE. Capacity must be greater than or equal to 0. Found Capacity: %d", in.CounterUpdateRequest.Capacity.GetValue())
		}
		capacitySet := in.CounterUpdateRequest.Capacity.GetValue()
		batchCounter.capacitySet = &capacitySet
	default:
		return nil, errors.Errorf("INVALID_ARGUMENT. Malformed CounterUpdateRequest: %v", in.CounterUpdateRequest)
	}

	s.gsCounterUpdates[name] = batchCounter

	// Queue up the Update for later batch processing by updateCounters.
	s.workerqueue.Enqueue(cache.ExplicitKey(updateCounters))
	return &alpha.Counter{}, nil
}

// updateCounter updates the Counters in the GameServer's Status with the batched update requests.
func (s *SDKServer) updateCounter(ctx context.Context) error {
	gs, err := s.gameServer()
	if err != nil {
		return err
	}
	gsCopy := gs.DeepCopy()

	s.logger.WithField("batchCounterUpdates", s.gsCounterUpdates).Debug("Batch updating Counter(s)")
	s.gsUpdateMutex.Lock()
	defer s.gsUpdateMutex.Unlock()

	names := []string{}

	for name, ctrReq := range s.gsCounterUpdates {
		counter, ok := gsCopy.Status.Counters[name]
		if !ok {
			continue
		}
		// Changes may have been made to the Counter since we validated the incoming changes in
		// UpdateCounter, and we need to verify if the batched changes can be fully applied, partially
		// applied, or cannot be applied.
		if ctrReq.capacitySet != nil {
			counter.Capacity = *ctrReq.capacitySet
		}
		if ctrReq.countSet != nil {
			counter.Count = *ctrReq.countSet
		}
		newCnt := counter.Count + ctrReq.diff
		if newCnt < 0 {
			newCnt = 0
			s.logger.Debug("truncating Count in Update Counter request to 0")
		}
		if newCnt > counter.Capacity {
			newCnt = counter.Capacity
			s.logger.Debug("truncating Count in Update Counter request to Capacity")
		}
		counter.Count = newCnt
		gsCopy.Status.Counters[name] = counter
		names = append(names, name)
	}

	gs, err = s.gameServerGetter.GameServers(s.namespace).Update(ctx, gsCopy, metav1.UpdateOptions{})
	if err != nil {
		return err
	}

	// Record an event per update Counter
	for _, name := range names {
		s.recorder.Event(gs, corev1.EventTypeNormal, "UpdateCounter",
			fmt.Sprintf("Counter %s updated to Count:%d Capacity:%d",
				name, gs.Status.Counters[name].Count, gs.Status.Counters[name].Capacity))
	}

	// Cache a copy of the successfully updated gameserver
	s.gsCopy = gs
	// Clear the gsCounterUpdates
	s.gsCounterUpdates = map[string]counterUpdateRequest{}

	return nil
}

// GetList returns a List. Returns NOT_FOUND if the List does not exist.
// [Stage:Alpha]
// [FeatureFlag:CountsAndLists]
func (s *SDKServer) GetList(ctx context.Context, in *alpha.GetListRequest) (*alpha.List, error) {
	if !runtime.FeatureEnabled(runtime.FeatureCountsAndLists) {
		return nil, errors.Errorf("%s not enabled", runtime.FeatureCountsAndLists)
	}
	// TODO(#2716): Implement me
	return nil, errors.Errorf("Unimplemented -- GetList coming soon")
}

// UpdateList returns the updated List.
// [Stage:Alpha]
// [FeatureFlag:CountsAndLists]
func (s *SDKServer) UpdateList(ctx context.Context, in *alpha.UpdateListRequest) (*alpha.List, error) {
	if !runtime.FeatureEnabled(runtime.FeatureCountsAndLists) {
		return nil, errors.Errorf("%s not enabled", runtime.FeatureCountsAndLists)
	}
	// TODO(#2716): Implement Me
	return nil, errors.Errorf("Unimplemented -- UpdateList coming soon")
}

// AddListValue returns the updated List.
// [Stage:Alpha]
// [FeatureFlag:CountsAndLists]
func (s *SDKServer) AddListValue(ctx context.Context, in *alpha.AddListValueRequest) (*alpha.List, error) {
	if !runtime.FeatureEnabled(runtime.FeatureCountsAndLists) {
		return nil, errors.Errorf("%s not enabled", runtime.FeatureCountsAndLists)
	}
	// TODO(#2716): Implement Me
	return nil, errors.Errorf("Unimplemented -- AddListValue coming soon")
}

// RemoveListValue returns the updated List.
// [Stage:Alpha]
// [FeatureFlag:CountsAndLists]
func (s *SDKServer) RemoveListValue(ctx context.Context, in *alpha.RemoveListValueRequest) (*alpha.List, error) {
	if !runtime.FeatureEnabled(runtime.FeatureCountsAndLists) {
		return nil, errors.Errorf("%s not enabled", runtime.FeatureCountsAndLists)
	}
	// TODO(#2716): Implement Me
	return nil, errors.Errorf("Unimplemented -- RemoveListValue coming soon")
}

// sendGameServerUpdate sends a watch game server event
func (s *SDKServer) sendGameServerUpdate(gs *agonesv1.GameServer) {
	s.logger.Debug("Sending GameServer Event to connectedStreams")

	s.streamMutex.Lock()
	defer s.streamMutex.Unlock()

	for i, stream := range s.connectedStreams {
		select {
		case <-stream.Context().Done():
			s.connectedStreams = append(s.connectedStreams[:i], s.connectedStreams[i+1:]...)

			err := stream.Context().Err()
			switch {
			case err != nil:
				s.logger.WithError(errors.WithStack(err)).Error("stream closed with error")
			default:
				s.logger.Debug("stream closed")
			}
			continue
		default:
		}

		if err := stream.Send(convert(gs)); err != nil {
			s.logger.WithError(errors.WithStack(err)).
				Error("error sending game server update event")
		}
	}

	if gs.Status.State == agonesv1.GameServerStateShutdown {
		// Wrap this in a go func(), just in case pushing to this channel deadlocks since there is only one instance of
		// a receiver. In theory, This could leak goroutines a bit, but since we're shuttling down everything anyway,
		// it shouldn't matter.
		go func() {
			s.gsStateChannel <- agonesv1.GameServerStateShutdown
		}()
	}
}

// checkHealthUpdateState checks the health as part of the /gshealthz hook, and if not
// healthy will push the Unhealthy state into the queue so it can be updated.
func (s *SDKServer) checkHealthUpdateState() {
	s.checkHealth()
	if !s.healthy() {
		s.logger.WithField("gameServerName", s.gameServerName).Warn("GameServer has failed health check")
		s.enqueueState(agonesv1.GameServerStateUnhealthy)
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

func (s *SDKServer) ensureHealthChecksRunning() {
	if s.health.Disabled {
		return
	}
	s.healthChecksRunning.Do(func() {
		// start health checking running
		s.logger.Debug("Starting GameServer health checking")
		go wait.Until(s.checkHealthUpdateState, s.healthTimeout, s.ctx.Done())
	})
}

// checkHealth checks the healthLastUpdated value
// and if it is outside the timeout value, logger and
// count a failure
func (s *SDKServer) checkHealth() {
	s.healthMutex.Lock()
	defer s.healthMutex.Unlock()

	timeout := s.healthLastUpdated.Add(s.healthTimeout)
	if timeout.Before(s.clock.Now().UTC()) {
		s.healthFailureCount++
		s.logger.WithField("failureCount", s.healthFailureCount).Warn("GameServer Health Check failed")
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

// updatePlayerCapacity updates the Player Capacity field in the GameServer's Status.
func (s *SDKServer) updatePlayerCapacity(ctx context.Context) error {
	if !runtime.FeatureEnabled(runtime.FeaturePlayerTracking) {
		return errors.Errorf("%s not enabled", runtime.FeaturePlayerTracking)
	}
	s.logger.WithField("capacity", s.gsPlayerCapacity).Debug("updating player capacity")
	gs, err := s.gameServer()
	if err != nil {
		return err
	}

	gsCopy := gs.DeepCopy()

	s.gsUpdateMutex.RLock()
	gsCopy.Status.Players.Capacity = s.gsPlayerCapacity
	s.gsUpdateMutex.RUnlock()

	gs, err = s.gameServerGetter.GameServers(s.namespace).Update(ctx, gsCopy, metav1.UpdateOptions{})
	if err != nil {
		return err
	}
	s.recorder.Event(gs, corev1.EventTypeNormal, "PlayerCapacity", fmt.Sprintf("Set to %d", gs.Status.Players.Capacity))
	return nil
}

// updateConnectedPlayers updates the Player IDs and Count fields in the GameServer's Status.
func (s *SDKServer) updateConnectedPlayers(ctx context.Context) error {
	if !runtime.FeatureEnabled(runtime.FeaturePlayerTracking) {
		return errors.Errorf("%s not enabled", runtime.FeaturePlayerTracking)
	}
	gs, err := s.gameServer()
	if err != nil {
		return err
	}

	gsCopy := gs.DeepCopy()
	same := false
	s.gsUpdateMutex.RLock()
	s.logger.WithField("playerIDs", s.gsConnectedPlayers).Debug("updating connected players")
	same = apiequality.Semantic.DeepEqual(gsCopy.Status.Players.IDs, s.gsConnectedPlayers)
	gsCopy.Status.Players.IDs = s.gsConnectedPlayers
	gsCopy.Status.Players.Count = int64(len(s.gsConnectedPlayers))
	s.gsUpdateMutex.RUnlock()
	// if there is no change, then don't update
	// since it's possible this could fire quite a lot, let's reduce the
	// amount of requests as much as possible.
	if same {
		return nil
	}

	gs, err = s.gameServerGetter.GameServers(s.namespace).Update(ctx, gsCopy, metav1.UpdateOptions{})
	if err != nil {
		return err
	}
	s.recorder.Event(gs, corev1.EventTypeNormal, "PlayerCount", fmt.Sprintf("Set to %d", gs.Status.Players.Count))
	return nil
}

// NewSDKServerContext returns a Context that cancels when SIGTERM or os.Interrupt
// is received and the GameServer's Status is shutdown
func (s *SDKServer) NewSDKServerContext(ctx context.Context) context.Context {
	sdkCtx, cancel := context.WithCancel(context.Background())
	go func() {
		<-ctx.Done()

		keepWaiting := true
		s.gsUpdateMutex.RLock()
		if len(s.gsState) > 0 {
			s.logger.WithField("state", s.gsState).Info("SDK server shutdown requested, waiting for game server shutdown")
		} else {
			s.logger.Info("SDK server state never updated by game server, shutting down sdk server without waiting")
			keepWaiting = false
		}
		s.gsUpdateMutex.RUnlock()

		for keepWaiting {
			gsState := <-s.gsStateChannel
			if gsState == agonesv1.GameServerStateShutdown {
				keepWaiting = false
			}
		}

		cancel()
	}()
	return sdkCtx
}
