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

// sidecar for the game server that the sdk connects to
package main

import (
	"fmt"
	"net"
	"strings"
	"time"

	"agones.dev/agones/pkg"
	"agones.dev/agones/pkg/client/clientset/versioned"
	"agones.dev/agones/pkg/gameservers"
	"agones.dev/agones/pkg/sdk"
	"agones.dev/agones/pkg/util/runtime"
	"agones.dev/agones/pkg/util/signals"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const (
	port = 59357

	// specifically env vars
	gameServerNameEnv = "GAMESERVER_NAME"
	podNamespaceEnv   = "POD_NAMESPACE"

	// Flags (that can also be env vars)
	localFlag                  = "local"
	addressFlag                = "address"
	healthDisabledFlag         = "health-disabled"
	healthTimeoutFlag          = "health-timeout"
	healthInitialDelayFlag     = "health-initial-delay"
	healthFailureThresholdFlag = "health-failure-threshold"
)

var (
	logger = runtime.NewLoggerWithSource("main")
)

func main() {
	viper.SetDefault(localFlag, false)
	viper.SetDefault(addressFlag, "localhost")
	viper.SetDefault(healthDisabledFlag, false)
	viper.SetDefault(healthTimeoutFlag, 5)
	viper.SetDefault(healthInitialDelayFlag, 5)
	viper.SetDefault(healthFailureThresholdFlag, 3)
	pflag.Bool(localFlag, viper.GetBool(localFlag),
		"Set this, or LOCAL env, to 'true' to run this binary in local development mode. Defaults to 'false'")
	pflag.String(addressFlag, viper.GetString(addressFlag), "The address to bind the server port to. Defaults to 'localhost")
	pflag.Bool(healthDisabledFlag, viper.GetBool(healthDisabledFlag),
		"Set this, or HEALTH_ENABLED env, to 'true' to enable health checking on the GameServer. Defaults to 'true'")
	pflag.Int64(healthTimeoutFlag, viper.GetInt64(healthTimeoutFlag),
		"Set this or HEALTH_TIMEOUT env to the number of seconds that the health check times out at. Defaults to 5")
	pflag.Int64(healthInitialDelayFlag, viper.GetInt64(healthInitialDelayFlag),
		"Set this or HEALTH_INITIAL_DELAY env to the number of seconds that the health will wait before starting. Defaults to 5")
	pflag.Int64(healthFailureThresholdFlag, viper.GetInt64(healthFailureThresholdFlag),
		"Set this or HEALTH_FAILURE_THRESHOLD env to the number of times the health check needs to fail to be deemed unhealthy. Defaults to 3")
	pflag.Parse()

	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	runtime.Must(viper.BindEnv(localFlag))
	runtime.Must(viper.BindEnv(gameServerNameEnv))
	runtime.Must(viper.BindEnv(podNamespaceEnv))
	runtime.Must(viper.BindEnv(healthDisabledFlag))
	runtime.Must(viper.BindEnv(healthTimeoutFlag))
	runtime.Must(viper.BindEnv(healthInitialDelayFlag))
	runtime.Must(viper.BindEnv(healthFailureThresholdFlag))
	runtime.Must(viper.BindPFlags(pflag.CommandLine))

	isLocal := viper.GetBool(localFlag)
	address := viper.GetString(addressFlag)
	healthDisabled := viper.GetBool(healthDisabledFlag)
	healthTimeout := time.Duration(viper.GetInt64(healthTimeoutFlag)) * time.Second
	healthInitialDelay := time.Duration(viper.GetInt64(healthInitialDelayFlag)) * time.Second
	healthFailureThreshold := viper.GetInt64(healthFailureThresholdFlag)

	logger.WithField(localFlag, isLocal).WithField("version", pkg.Version).
		WithField("port", port).WithField(addressFlag, address).
		WithField(healthDisabledFlag, healthDisabled).WithField(healthTimeoutFlag, healthTimeout).
		WithField(healthFailureThresholdFlag, healthFailureThreshold).
		WithField(healthInitialDelayFlag, healthInitialDelay).Info("Starting sdk sidecar")

	lis, err := net.Listen("tcp", fmt.Sprintf("%s:%d", address, port))
	if err != nil {
		logger.WithField("port", port).WithField("address", address).Fatalf("Could not listen on port")
	}
	stop := signals.NewStopChannel()
	grpcServer := grpc.NewServer()

	if isLocal {
		sdk.RegisterSDKServer(grpcServer, &gameservers.LocalSDKServer{})
	} else {
		config, err := rest.InClusterConfig()
		if err != nil {
			logger.WithError(err).Fatal("Could not create in cluster config")
		}

		kubeClient, err := kubernetes.NewForConfig(config)
		if err != nil {
			logger.WithError(err).Fatal("Could not create the kubernetes clientset")
		}

		agonesClient, err := versioned.NewForConfig(config)
		if err != nil {
			logger.WithError(err).Fatalf("Could not create the agones api clientset")
		}

		var s *gameservers.SDKServer
		s, err = gameservers.NewSDKServer(viper.GetString(gameServerNameEnv), viper.GetString(podNamespaceEnv),
			healthDisabled, healthTimeout, healthFailureThreshold, healthInitialDelay, kubeClient, agonesClient)
		if err != nil {
			logger.WithError(err).Fatalf("Could not start sidecar")
		}
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		go s.Run(ctx.Done())
		sdk.RegisterSDKServer(grpcServer, s)
	}

	go func() {
		err = grpcServer.Serve(lis)
		if err != nil {
			logger.WithError(err).Fatal("Could not serve grpc server")
		}
	}()

	<-stop
	logger.Info("shutting down grpc server")
	// don't graceful stop, because if we get a kill signal
	// then the gameserver is being shut down, and we no longer
	// care about running RPC calls.
	grpcServer.Stop()
}
