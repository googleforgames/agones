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

	"agones.dev/agones/pkg/sdk"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/context"
)

var _ sdk.SDKServer = &LocalSDKServer{}

// LocalSDKServer type is the SDKServer implementation for when the sidecar
// is being run for local development, and doesn't connect to the
// Kubernetes cluster
type LocalSDKServer struct {
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
