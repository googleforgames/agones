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
	updateState             Operation     = "updateState"
	updateLabel             Operation     = "updateLabel"
	updateAnnotation        Operation     = "updateAnnotation"
	updatePlayerCapacity    Operation     = "updatePlayerCapacity"
	updateConnectedPlayers  Operation     = "updateConnectedPlayers"
	playerCountUpdatePeriod time.Duration = time.Second
)

var (
	_ sdk.SDKServer   = &SDKServer{}
	_ alpha.SDKServer = &SDKServer{}
	_ beta.SDKServer  = &SDKServer{}
)

// SDKServer is a gRPC server, that is meant to be a sidecar
// for a GameServer that will update the game server status on SDK requests
//
//nolint:govet // ignore fieldalignment, singleton
type SDKServer struct {
	logger             *logrus.Entry
	gameServerName     string
	namespace          string
	informerFactory    externalversions.SharedInformerFactory
	gameServerGetter   typedv1.GameServersGetter
	gameServerLister   listersv1.GameServerLister
	gameServerSynced   cache.InformerSynced
	server             *http.Server
	clock              clock.Clock
	health             agonesv1.Health
	healthTimeout      time.Duration
	healthMutex        sync.RWMutex
	healthLastUpdated  time.Time
	healthFailureCount int32
	workerqueue        *workerqueue.WorkerQueue
	streamMutex        sync.RWMutex
	connectedStreams   []sdk.SDK_WatchGameServerServer
	ctx                context.Context
	recorder           record.EventRecorder
	gsLabels           map[string]string
	gsAnnotations      map[string]string
	gsState            agonesv1.GameServerState
	gsStateChannel     chan agonesv1.GameServerState
	gsUpdateMutex      sync.RWMutex
	gsWaitForSync      sync.WaitGroup
	reserveTimer       *time.Timer
	gsReserveDuration  *time.Duration
	gsPlayerCapacity   int64
	gsConnectedPlayers []string
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

	s.informerFactory = factory
	s.logger = runtime.NewLoggerWithType(s).WithField("gsKey", namespace+"/"+gameServerName)

	gameServers.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
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

// initHealthLastUpdated adds the initial delay to now, then it will always be after `now`
// until the delay passes
func (s *SDKServer) initHealthLastUpdated(healthInitialDelay time.Duration) {
	s.healthLastUpdated = s.clock.Now().UTC().Add(healthInitialDelay)
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
	s.initHealthLastUpdated(time.Duration(gs.Spec.Health.InitialDelaySeconds) * time.Second)

	if gs.Status.State == agonesv1.GameServerStateReserved && gs.Status.ReservedUntil != nil {
		s.gsUpdateMutex.Lock()
		s.resetReserveAfter(context.Background(), time.Until(gs.Status.ReservedUntil.Time))
		s.gsUpdateMutex.Unlock()
	}

	// start health checking running
	if !s.health.Disabled {
		s.logger.Debug("Starting GameServer health checking")
		go wait.Until(s.runHealth, s.healthTimeout, ctx.Done())
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

func (s *SDKServer) gameServer() (*agonesv1.GameServer, error) {
	// this ensure that if we get requests for the gameserver before the cache has been synced,
	// they will block here until it's ready
	s.gsWaitForSync.Wait()
	gs, err := s.gameServerLister.GameServers(s.namespace).Get(s.gameServerName)
	return gs, errors.Wrapf(err, "could not retrieve GameServer %s/%s", s.namespace, s.gameServerName)
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
	s.gsState = state
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
	s.workerqueue.EnqueueAfter(cache.ExplicitKey(string(updateConnectedPlayers)), playerCountUpdatePeriod)

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
	s.workerqueue.EnqueueAfter(cache.ExplicitKey(string(updateConnectedPlayers)), playerCountUpdatePeriod)

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

// sendGameServerUpdate sends a watch game server event
func (s *SDKServer) sendGameServerUpdate(gs *agonesv1.GameServer) {
	s.logger.Debug("Sending GameServer Event to connectedStreams")

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

	if runtime.FeatureEnabled(runtime.FeatureSDKGracefulTermination) && gs.Status.State == agonesv1.GameServerStateShutdown {
		// Wrap this in a go func(), just in case pushing to this channel deadlocks since there is only one instance of
		// a receiver. In theory, This could leak goroutines a bit, but since we're shuttling down everything anyway,
		// it shouldn't matter.
		go func() {
			s.gsStateChannel <- agonesv1.GameServerStateShutdown
		}()
	}
}

// runHealth actively checks the health, and if not
// healthy will push the Unhealthy state into the queue so
// it can be updated
func (s *SDKServer) runHealth() {
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

// checkHealth checks the healthLastUpdated value
// and if it is outside the timeout value, logger and
// count a failure
func (s *SDKServer) checkHealth() {
	timeout := s.healthLastUpdated.Add(s.healthTimeout)
	if timeout.Before(s.clock.Now().UTC()) {
		s.healthMutex.Lock()
		defer s.healthMutex.Unlock()
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
		for {
			gsState := <-s.gsStateChannel
			if gsState == agonesv1.GameServerStateShutdown {
				cancel()
			}
		}
	}()
	return sdkCtx
}
