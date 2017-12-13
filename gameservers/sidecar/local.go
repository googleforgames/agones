// Copyright 2017 Google Inc. All Rights Reserved.
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

package main

import (
	"github.com/agonio/agon/gameservers/sidecar/sdk"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/context"
)

var _ sdk.SDKServer = &Local{}

// Local type is the SDKServer implementation for when the sidecar
// is being run for local development, and doesn't connect to the
// Kubernetes cluster
type Local struct {
}

// Ready logs that the Ready request has been received
func (l *Local) Ready(context.Context, *sdk.Empty) (*sdk.Empty, error) {
	logrus.Info("Ready request has been received!")
	return &sdk.Empty{}, nil
}

// Shutdown logs that the shutdown request has been received
func (l *Local) Shutdown(context.Context, *sdk.Empty) (*sdk.Empty, error) {
	logrus.Info("Shutdown request has been received!")
	return &sdk.Empty{}, nil
}
