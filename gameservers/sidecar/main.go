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

// sidecar for the game server that the sdk connects to
package main

import (
	"fmt"
	"net"

	"github.com/agonio/agon/gameservers/sidecar/sdk"
	"github.com/agonio/agon/pkg/util/runtime"
	"github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

const (
	// Version the release version of the sidecar
	Version = "0.1"
	port    = 59357

	// gameServerNameEnv is the environment variable for the Game Server name
	gameServerNameEnv = "GAMESERVER_NAME"
	// podNamespaceEnv is the environment variable for the current Game Server namespace
	podNamespaceEnv = "POD_NAMESPACE"

	// localFlag determines if this is running locally or not
	localFlag = "local"
)

func init() {
	logrus.SetFormatter(&logrus.JSONFormatter{})
}

func main() {
	viper.SetDefault(localFlag, false)
	pflag.Bool(localFlag, viper.GetBool(localFlag), "Set this, or LOCAL env, to 'true' to run this binary in local development mode. Defaults to 'false'")
	pflag.Parse()

	runtime.Must(viper.BindEnv(localFlag))
	runtime.Must(viper.BindEnv(gameServerNameEnv))
	runtime.Must(viper.BindEnv(podNamespaceEnv))
	runtime.Must(viper.BindPFlags(pflag.CommandLine))

	isLocal := viper.GetBool(localFlag)

	logrus.WithField(localFlag, isLocal).WithField("version", Version).WithField("port", port).Info("Starting sdk sidecar")

	lis, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", port))
	if err != nil {
		logrus.WithField("port", port).Fatalf("Could not listen on port")
	}
	grpcServer := grpc.NewServer()

	if isLocal {
		sdk.RegisterSDKServer(grpcServer, &Local{})
	} else {
		var s *Sidecar
		s, err = NewSidecar(viper.GetString(gameServerNameEnv), viper.GetString(podNamespaceEnv))
		if err != nil {
			logrus.WithError(err).Fatalf("Could not start sidecar")
		}
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		go s.Run(ctx.Done())
		sdk.RegisterSDKServer(grpcServer, s)
	}

	err = grpcServer.Serve(lis)
	if err != nil {
		logrus.WithError(err).Error("Could not serve grpc server")
	}
}
