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
	"os"
	"sync"
	"time"

	"agones.dev/agones/pkg/apis/stable/v1alpha1"
	"agones.dev/agones/pkg/sdk"
	"github.com/fsnotify/fsnotify"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/context"
	"k8s.io/apimachinery/pkg/util/yaml"
)

var (
	_ sdk.SDKServer = &LocalSDKServer{}

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
	gsMutex         sync.RWMutex
	gs              *sdk.GameServer
	update          chan struct{}
	updateObservers sync.Map
}

// NewLocalSDKServer returns the default LocalSDKServer
func NewLocalSDKServer(filePath string) (*LocalSDKServer, error) {
	l := &LocalSDKServer{
		gsMutex:         sync.RWMutex{},
		gs:              defaultGs,
		update:          make(chan struct{}),
		updateObservers: sync.Map{},
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
				if event.Op == fsnotify.Write {
					logrus.WithField("event", event).Info("File has been changed!")
					err := l.setGameServerFromFilePath(filePath)
					if err != nil {
						logrus.WithError(err).Error("error setting GameServer from file")
						continue
					}
					logrus.Info("Sending watched GameServer!")
					l.update <- struct{}{}
				}
			}
		}()

		err = watcher.Add(filePath)
		if err != nil {
			logrus.WithError(err).WithField("filePath", filePath).Error("error adding watcher")
		}
	}

	go func() {
		for value := range l.update {
			logrus.Info("gameserver update received")
			l.updateObservers.Range(func(observer, _ interface{}) bool {
				observer.(chan struct{}) <- value
				return true
			})
		}
	}()

	return l, nil
}

// Ready logs that the Ready request has been received
func (l *LocalSDKServer) Ready(context.Context, *sdk.Empty) (*sdk.Empty, error) {
	logrus.Info("Ready request has been received!")
	return &sdk.Empty{}, nil
}

// Shutdown logs that the shutdown request has been received
func (l *LocalSDKServer) Shutdown(context.Context, *sdk.Empty) (*sdk.Empty, error) {
	logrus.Info("Shutdown request has been received!")
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
	return &sdk.Empty{}, nil
}

// GetGameServer returns a dummy game server.
func (l *LocalSDKServer) GetGameServer(context.Context, *sdk.Empty) (*sdk.GameServer, error) {
	logrus.Info("getting GameServer details")
	l.gsMutex.RLock()
	defer l.gsMutex.RUnlock()
	return l.gs, nil
}

// WatchGameServer will return a dummy GameServer (with no changes), 3 times, every 5 seconds
func (l *LocalSDKServer) WatchGameServer(_ *sdk.Empty, stream sdk.SDK_WatchGameServerServer) error {
	logrus.Info("connected to watch GameServer...")
	observer := make(chan struct{})

	defer func() {
		l.updateObservers.Delete(observer)
	}()

	l.updateObservers.Store(observer, true)

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

// Close tears down all the things
func (l *LocalSDKServer) Close() {
	l.updateObservers.Range(func(observer, _ interface{}) bool {
		close(observer.(chan struct{}))
		return true
	})
}

func (l *LocalSDKServer) setGameServerFromFilePath(filePath string) error {
	logrus.WithField("filePath", filePath).Info("Reading GameServer configuration")

	reader, err := os.Open(filePath) // nolint: gosec
	defer reader.Close()             // nolint: megacheck,errcheck

	if err != nil {
		return err
	}

	var gs v1alpha1.GameServer
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
