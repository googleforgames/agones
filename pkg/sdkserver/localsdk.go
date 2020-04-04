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
	"math/rand"
	"os"
	"strconv"
	"sync"
	"time"

	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	"agones.dev/agones/pkg/sdk"
	"agones.dev/agones/pkg/sdk/alpha"
	"agones.dev/agones/pkg/sdk/beta"
	"agones.dev/agones/pkg/util/runtime"
	"github.com/fsnotify/fsnotify"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/context"
	"k8s.io/apimachinery/pkg/util/yaml"
)

var (
	_ sdk.SDKServer   = &LocalSDKServer{}
	_ alpha.SDKServer = &LocalSDKServer{}
	_ beta.SDKServer  = &LocalSDKServer{}

	defaultGs = &sdk.GameServer{
		ObjectMeta: &sdk.GameServer_ObjectMeta{
			Name:              "local",
			Namespace:         "default",
			Uid:               "1234",
			Generation:        1,
			ResourceVersion:   "v1",
			CreationTimestamp: time.Now().Unix(),
			Labels:            map[string]string{"islocal": "true"},
			Annotations:       map[string]string{"annotation": "true"},
		},
		Spec: &sdk.GameServer_Spec{
			Health: &sdk.GameServer_Spec_Health{
				Disabled:            false,
				PeriodSeconds:       3,
				FailureThreshold:    5,
				InitialDelaySeconds: 10,
			},
		},
		Status: &sdk.GameServer_Status{
			State:   "Ready",
			Address: "127.0.0.1",
			Ports:   []*sdk.GameServer_Status_Port{{Name: "default", Port: 7777}},
		},
	}
)

// LocalSDKServer type is the SDKServer implementation for when the sidecar
// is being run for local development, and doesn't connect to the
// Kubernetes cluster
type LocalSDKServer struct {
	gsMutex           sync.RWMutex
	gs                *sdk.GameServer
	update            chan struct{}
	updateObservers   sync.Map
	requestSequence   []string
	expectedSequence  []string
	gsState           agonesv1.GameServerState
	gsReserveDuration *time.Duration
	reserveTimer      *time.Timer
	testMode          bool
}

// NewLocalSDKServer returns the default LocalSDKServer
func NewLocalSDKServer(filePath string) (*LocalSDKServer, error) {
	l := &LocalSDKServer{
		gsMutex:         sync.RWMutex{},
		gs:              defaultGs,
		update:          make(chan struct{}),
		updateObservers: sync.Map{},
		requestSequence: make([]string, 0),
		testMode:        false,
		gsState:         agonesv1.GameServerStateScheduled,
	}

	if filePath != "" {
		err := l.setGameServerFromFilePath(filePath)
		if err != nil {
			return l, err
		}

		watcher, err := fsnotify.NewWatcher()
		if err != nil {
			return l, err
		}

		go func() {
			for event := range watcher.Events {
				if event.Op != fsnotify.Write {
					continue
				}
				logrus.WithField("event", event).Info("File has been changed!")
				err := l.setGameServerFromFilePath(filePath)
				if err != nil {
					logrus.WithError(err).Error("error setting GameServer from file")
					continue
				}
				logrus.Info("Sending watched GameServer!")
				l.update <- struct{}{}
			}
		}()

		err = watcher.Add(filePath)
		if err != nil {
			logrus.WithError(err).WithField("filePath", filePath).Error("error adding watcher")
		}
	}

	go func() {
		for value := range l.update {
			logrus.Info("Gameserver update received")
			l.updateObservers.Range(func(observer, _ interface{}) bool {
				observer.(chan struct{}) <- value
				return true
			})
		}
	}()

	return l, nil
}

// GenerateUID - generate gameserver UID at random for testing
func (l *LocalSDKServer) GenerateUID() {
	// Generating Random UID
	seededRand := rand.New(
		rand.NewSource(time.Now().UnixNano()))
	UID := fmt.Sprintf("%d", seededRand.Int())
	l.gs.ObjectMeta.Uid = UID
}

// SetTestMode set test mode to collect the sequence of performed requests
func (l *LocalSDKServer) SetTestMode(testMode bool) {
	l.testMode = testMode
}

// SetExpectedSequence set expected request sequence which would be
// verified against after run was completed
func (l *LocalSDKServer) SetExpectedSequence(sequence []string) {
	l.expectedSequence = sequence
}

// recordRequest append request name to slice
func (l *LocalSDKServer) recordRequest(request string) {
	if l.testMode {
		l.requestSequence = append(l.requestSequence, request)
	}
}

// recordRequestWithValue append request name to slice only if
// value equals to objMetaField: creationTimestamp or UID
func (l *LocalSDKServer) recordRequestWithValue(request string, value string, objMetaField string) {
	if l.testMode {
		fieldVal := ""
		switch objMetaField {
		case "CreationTimestamp":
			fieldVal = strconv.FormatInt(l.gs.ObjectMeta.CreationTimestamp, 10)
		case "UID":
			fieldVal = l.gs.ObjectMeta.Uid
		case "PlayerCapacity":
			fieldVal = strconv.FormatInt(l.gs.Status.Players.Capacity, 10)
		default:
			fmt.Printf("Error: Unexpected Field to compare")
		}

		if value == fieldVal {
			l.requestSequence = append(l.requestSequence, request)
		} else {
			fmt.Printf("Error: we expected to receive '%s' as value for '%s' request but received '%s'. \n", fieldVal, request, value)
		}
	}
}

func (l *LocalSDKServer) updateState(newState agonesv1.GameServerState) {
	l.gsState = newState
	l.gs.Status.State = string(l.gsState)
}

// Ready logs that the Ready request has been received
func (l *LocalSDKServer) Ready(context.Context, *sdk.Empty) (*sdk.Empty, error) {
	logrus.Info("Ready request has been received!")
	l.recordRequest("ready")
	l.gsMutex.Lock()
	defer l.gsMutex.Unlock()

	// Follow the GameServer state diagram
	l.updateState(agonesv1.GameServerStateReady)
	l.stopReserveTimer()
	l.update <- struct{}{}
	return &sdk.Empty{}, nil
}

// Allocate logs that an allocate request has been received
func (l *LocalSDKServer) Allocate(context.Context, *sdk.Empty) (*sdk.Empty, error) {
	logrus.Info("Allocate request has been received!")
	l.recordRequest("allocate")
	l.gsMutex.Lock()
	defer l.gsMutex.Unlock()
	l.updateState(agonesv1.GameServerStateAllocated)
	l.stopReserveTimer()
	l.update <- struct{}{}

	return &sdk.Empty{}, nil
}

// Shutdown logs that the shutdown request has been received
func (l *LocalSDKServer) Shutdown(context.Context, *sdk.Empty) (*sdk.Empty, error) {
	logrus.Info("Shutdown request has been received!")
	l.recordRequest("shutdown")
	l.gsMutex.Lock()
	defer l.gsMutex.Unlock()
	l.updateState(agonesv1.GameServerStateShutdown)
	l.stopReserveTimer()
	l.update <- struct{}{}
	return &sdk.Empty{}, nil
}

// Health logs each health ping that comes down the stream
func (l *LocalSDKServer) Health(stream sdk.SDK_HealthServer) error {
	for {
		_, err := stream.Recv()
		if err == io.EOF {
			logrus.Info("Health stream closed.")
			return stream.SendAndClose(&sdk.Empty{})
		}
		if err != nil {
			return errors.Wrap(err, "Error with Health check")
		}
		l.recordRequest("health")
		logrus.Info("Health Ping Received!")
	}
}

// SetLabel applies a Label to the backing GameServer metadata
func (l *LocalSDKServer) SetLabel(_ context.Context, kv *sdk.KeyValue) (*sdk.Empty, error) {
	logrus.WithField("values", kv).Info("Setting label")
	l.gsMutex.Lock()
	defer l.gsMutex.Unlock()

	if l.gs.ObjectMeta == nil {
		l.gs.ObjectMeta = &sdk.GameServer_ObjectMeta{}
	}
	if l.gs.ObjectMeta.Labels == nil {
		l.gs.ObjectMeta.Labels = map[string]string{}
	}

	l.gs.ObjectMeta.Labels[metadataPrefix+kv.Key] = kv.Value
	l.update <- struct{}{}
	l.recordRequestWithValue("setlabel", kv.Value, "CreationTimestamp")
	return &sdk.Empty{}, nil
}

// SetAnnotation applies a Annotation to the backing GameServer metadata
func (l *LocalSDKServer) SetAnnotation(_ context.Context, kv *sdk.KeyValue) (*sdk.Empty, error) {
	logrus.WithField("values", kv).Info("Setting annotation")
	l.gsMutex.Lock()
	defer l.gsMutex.Unlock()

	if l.gs.ObjectMeta == nil {
		l.gs.ObjectMeta = &sdk.GameServer_ObjectMeta{}
	}
	if l.gs.ObjectMeta.Annotations == nil {
		l.gs.ObjectMeta.Annotations = map[string]string{}
	}

	l.gs.ObjectMeta.Annotations[metadataPrefix+kv.Key] = kv.Value
	l.update <- struct{}{}
	l.recordRequestWithValue("setannotation", kv.Value, "UID")
	return &sdk.Empty{}, nil
}

// GetGameServer returns current GameServer configuration.
func (l *LocalSDKServer) GetGameServer(context.Context, *sdk.Empty) (*sdk.GameServer, error) {
	logrus.Info("Getting GameServer details")
	l.recordRequest("gameserver")
	l.gsMutex.RLock()
	defer l.gsMutex.RUnlock()
	return l.gs, nil
}

// WatchGameServer will return current GameServer configuration, 3 times, every 5 seconds
func (l *LocalSDKServer) WatchGameServer(_ *sdk.Empty, stream sdk.SDK_WatchGameServerServer) error {
	logrus.Info("Connected to watch GameServer...")
	observer := make(chan struct{})

	defer func() {
		l.updateObservers.Delete(observer)
	}()

	l.updateObservers.Store(observer, true)

	l.recordRequest("watch")
	for range observer {
		l.gsMutex.RLock()
		err := stream.Send(l.gs)
		l.gsMutex.RUnlock()
		if err != nil {
			logrus.WithError(err).Error("error sending gameserver")
			return err
		}
	}

	return nil
}

// Reserve moves this GameServer to the Reserved state for the Duration specified
func (l *LocalSDKServer) Reserve(ctx context.Context, d *sdk.Duration) (*sdk.Empty, error) {
	logrus.WithField("duration", d).Info("Reserve request has been received!")
	l.recordRequest("reserve")
	l.gsMutex.Lock()
	defer l.gsMutex.Unlock()
	if d.Seconds > 0 {
		duration := time.Duration(d.Seconds) * time.Second
		l.gsReserveDuration = &duration
		l.resetReserveAfter(ctx, *l.gsReserveDuration)
	}

	l.updateState(agonesv1.GameServerStateReserved)
	l.update <- struct{}{}

	return &sdk.Empty{}, nil
}

func (l *LocalSDKServer) resetReserveAfter(ctx context.Context, duration time.Duration) {
	if l.reserveTimer != nil {
		l.reserveTimer.Stop()
	}

	l.reserveTimer = time.AfterFunc(duration, func() {
		if _, err := l.Ready(ctx, &sdk.Empty{}); err != nil {
			logrus.WithError(err).Error("error returning to Ready after reserved ")
		}
	})
}

func (l *LocalSDKServer) stopReserveTimer() {
	if l.reserveTimer != nil {
		l.reserveTimer.Stop()
	}
	l.gsReserveDuration = nil
}

// PlayerConnect should be called when a player connects.
// [Stage:Alpha]
// [FeatureFlag:PlayerTesting]
func (l *LocalSDKServer) PlayerConnect(ctx context.Context, id *alpha.PlayerId) (*alpha.Empty, error) {
	panic("implement me")
}

// PlayerDisconnect should be called when a player disconnects.
// [Stage:Alpha]
// [FeatureFlag:PlayerTesting]
func (l *LocalSDKServer) PlayerDisconnect(ctx context.Context, id *alpha.PlayerId) (*alpha.Empty, error) {
	panic("implement me")
}

// SetPlayerCapacity to change the game server's player capacity.
// [Stage:Alpha]
// [FeatureFlag:PlayerTesting]
func (l *LocalSDKServer) SetPlayerCapacity(_ context.Context, count *alpha.Count) (*alpha.Empty, error) {
	if !runtime.FeatureEnabled(runtime.FeaturePlayerTracking) {
		return nil, errors.New(string(runtime.FeaturePlayerTracking) + " not enabled")
	}

	logrus.WithField("capacity", count.Count).Info("Setting Player Capacity")
	l.gsMutex.Lock()
	defer l.gsMutex.Unlock()

	if l.gs.Status.Players == nil {
		l.gs.Status.Players = &sdk.GameServer_Status_PlayerStatus{}
	}

	l.gs.Status.Players.Capacity = count.Count

	l.update <- struct{}{}
	l.recordRequestWithValue("setplayercapacity", strconv.FormatInt(count.Count, 10), "PlayerCapacity")
	return &alpha.Empty{}, nil
}

// GetPlayerCapacity returns the current player capacity.
// [Stage:Alpha]
// [FeatureFlag:PlayerTesting]
func (l *LocalSDKServer) GetPlayerCapacity(_ context.Context, _ *alpha.Empty) (*alpha.Count, error) {
	if !runtime.FeatureEnabled(runtime.FeaturePlayerTracking) {
		return nil, errors.New(string(runtime.FeaturePlayerTracking) + " not enabled")
	}
	logrus.Info("Getting Player Capacity")
	l.recordRequest("getplayercapacity")
	l.gsMutex.RLock()
	defer l.gsMutex.RUnlock()

	// SDK.GetPlayerCapacity() has a contract of always return a number,
	// so if we're nil, then let's always return a value, and
	// remove lots of special cases upstream.
	result := &alpha.Count{}
	if l.gs.Status.Players != nil {
		result.Count = l.gs.Status.Players.Capacity
	}

	return result, nil
}

// GetPlayerCount returns the current player count.
// [Stage:Alpha]
// [FeatureFlag:PlayerTesting]
func (l *LocalSDKServer) GetPlayerCount(ctx context.Context, _ *alpha.Empty) (*alpha.Count, error) {
	panic("implement me")
}

// Close tears down all the things
func (l *LocalSDKServer) Close() {
	l.updateObservers.Range(func(observer, _ interface{}) bool {
		close(observer.(chan struct{}))
		return true
	})
	l.compare()
}

// EqualSets tells whether a and b contain the same elements.
// A nil argument is equivalent to an empty slice.
func EqualSets(a, b []string) bool {
	aSet := make(map[string]bool)
	bSet := make(map[string]bool)
	for _, v := range a {
		aSet[v] = true
	}
	for _, v := range b {
		if _, ok := aSet[v]; !ok {
			return false
		}
		bSet[v] = true
	}
	for _, v := range a {
		if _, ok := bSet[v]; !ok {
			return false
		}
	}
	return true
}

// Close tears down all the things
func (l *LocalSDKServer) compare() {
	logrus.Info("Compare")

	if l.testMode {
		if !EqualSets(l.expectedSequence, l.requestSequence) {
			logrus.Info(fmt.Sprintf("Testing Failed %v %v", l.expectedSequence, l.requestSequence))
			os.Exit(1)
		}
	}
}

func (l *LocalSDKServer) setGameServerFromFilePath(filePath string) error {
	logrus.WithField("filePath", filePath).Info("Reading GameServer configuration")

	reader, err := os.Open(filePath) // nolint: gosec
	defer reader.Close()             // nolint: megacheck,errcheck

	if err != nil {
		return err
	}

	var gs agonesv1.GameServer
	// 4096 is the number of bytes the YAMLOrJSONDecoder goes looking
	// into the file to determine if it's JSON or YAML
	// (JSON == has whitespace followed by an open brace).
	// The Kubernetes uses 4096 bytes as its default, so that's what we'll
	// use as well.
	// https://github.com/kubernetes/kubernetes/blob/master/plugin/pkg/admission/podnodeselector/admission.go#L86
	decoder := yaml.NewYAMLOrJSONDecoder(reader, 4096)
	err = decoder.Decode(&gs)
	if err != nil {
		return err
	}

	l.gsMutex.Lock()
	defer l.gsMutex.Unlock()
	l.gs = convert(&gs)
	return nil
}
