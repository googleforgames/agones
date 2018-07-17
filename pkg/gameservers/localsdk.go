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
	"time"

	"agones.dev/agones/pkg/sdk"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/context"
)

var (
	_ sdk.SDKServer = &LocalSDKServer{}

	fixture = &sdk.GameServer{
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
	watchPeriod time.Duration
}

// NewLocalSDKServer returns the default LocalSDKServer
func NewLocalSDKServer() *LocalSDKServer {
	return &LocalSDKServer{
		watchPeriod: 5 * time.Second,
	}
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

// GetGameServer returns a dummy game server.
func (l *LocalSDKServer) GetGameServer(context.Context, *sdk.Empty) (*sdk.GameServer, error) {
	logrus.Info("getting GameServer details")
	return fixture, nil
}

// WatchGameServer will return a dummy GameServer (with no changes), 3 times, every 5 seconds
func (l *LocalSDKServer) WatchGameServer(_ *sdk.Empty, stream sdk.SDK_WatchGameServerServer) error {
	logrus.Info("connected to watch GameServer...")
	times := 3

	for i := 0; i < times; i++ {
		logrus.Info("Sending watched GameServer!")
		err := stream.Send(fixture)
		if err != nil {
			logrus.WithError(err).Error("error sending gameserver")
			return err
		}

		time.Sleep(l.watchPeriod)
	}

	return nil
}
