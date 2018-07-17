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
	"net/http"
	"strings"
	"time"

	"agones.dev/agones/pkg"
	"agones.dev/agones/pkg/client/clientset/versioned"
	"agones.dev/agones/pkg/gameservers"
	"agones.dev/agones/pkg/sdk"
	"agones.dev/agones/pkg/util/runtime"
	"agones.dev/agones/pkg/util/signals"
	gwruntime "github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const (
	grpcPort = 59357
	httpPort = 59358

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
	ctlConf := parseEnvFlags()
	logger.WithField("version", pkg.Version).
		WithField("grpcPort", grpcPort).WithField("httpPort", httpPort).
		WithField("ctlConf", ctlConf).Info("Starting sdk sidecar")

	grpcEndpoint := fmt.Sprintf("%s:%d", ctlConf.Address, grpcPort)
	lis, err := net.Listen("tcp", grpcEndpoint)
	if err != nil {
		logger.WithField("grpcPort", grpcPort).WithField("Address", ctlConf.Address).Fatalf("Could not listen on grpcPort")
	}
	stop := signals.NewStopChannel()
	grpcServer := grpc.NewServer()
	// don't graceful stop, because if we get a kill signal
	// then the gameserver is being shut down, and we no longer
	// care about running RPC calls.
	defer grpcServer.Stop()

	mux := gwruntime.NewServeMux()
	httpServer := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", ctlConf.Address, httpPort),
		Handler: mux,
	}
	defer httpServer.Close() // nolint: errcheck
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if ctlConf.IsLocal {
		sdk.RegisterSDKServer(grpcServer, gameservers.NewLocalSDKServer())
	} else {
		var config *rest.Config
		config, err = rest.InClusterConfig()
		if err != nil {
			logger.WithError(err).Fatal("Could not create in cluster config")
		}

		var kubeClient *kubernetes.Clientset
		kubeClient, err = kubernetes.NewForConfig(config)
		if err != nil {
			logger.WithError(err).Fatal("Could not create the kubernetes clientset")
		}

		var agonesClient *versioned.Clientset
		agonesClient, err = versioned.NewForConfig(config)
		if err != nil {
			logger.WithError(err).Fatalf("Could not create the agones api clientset")
		}

		var s *gameservers.SDKServer
		s, err = gameservers.NewSDKServer(viper.GetString(gameServerNameEnv), viper.GetString(podNamespaceEnv),
			ctlConf.HealthDisabled, ctlConf.HealthTimeout, ctlConf.HealthFailureThreshold,
			ctlConf.HealthInitialDelay, kubeClient, agonesClient)
		if err != nil {
			logger.WithError(err).Fatalf("Could not start sidecar")
		}

		go s.Run(ctx.Done())
		sdk.RegisterSDKServer(grpcServer, s)
	}

	go runGrpc(grpcServer, lis)
	go runGateway(ctx, grpcEndpoint, mux, httpServer)

	<-stop
	logger.Info("shutting down sdk server")
}

// runGrpc runs the grpc service
func runGrpc(grpcServer *grpc.Server, lis net.Listener) {
	logger.Info("Starting SDKServer grpc service...")
	if err := grpcServer.Serve(lis); err != nil {
		logger.WithError(err).Fatal("Could not serve grpc server")
	}
}

// runGateway runs the grpc-gateway
func runGateway(ctx context.Context, grpcEndpoint string, mux *gwruntime.ServeMux, httpServer *http.Server) {
	conn, err := grpc.DialContext(ctx, grpcEndpoint, grpc.WithBlock(), grpc.WithInsecure())
	if err != nil {
		logger.WithError(err).Fatal("Could not dial grpc server...")
	}

	if err = sdk.RegisterSDKHandler(ctx, mux, conn); err != nil {
		logger.WithError(err).Fatal("Could not register grpc-gateway")
	}

	logger.Info("Starting SDKServer grpc-gateway...")
	if err := httpServer.ListenAndServe(); err != nil {
		if err == http.ErrServerClosed {
			logger.WithError(err).Info("http server closed")
		} else {
			logger.WithError(err).Fatal("Could not serve http server")
		}
	}
}

// parseEnvFlags parses all the flags and environment variables and returns
// a configuration structure
func parseEnvFlags() config {
	viper.SetDefault(localFlag, false)
	viper.SetDefault(addressFlag, "localhost")
	viper.SetDefault(healthDisabledFlag, false)
	viper.SetDefault(healthTimeoutFlag, 5)
	viper.SetDefault(healthInitialDelayFlag, 5)
	viper.SetDefault(healthFailureThresholdFlag, 3)
	pflag.Bool(localFlag, viper.GetBool(localFlag),
		"Set this, or LOCAL env, to 'true' to run this binary in local development mode. Defaults to 'false'")
	pflag.String(addressFlag, viper.GetString(addressFlag), "The Address to bind the server grpcPort to. Defaults to 'localhost")
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

	return config{
		IsLocal:                viper.GetBool(localFlag),
		Address:                viper.GetString(addressFlag),
		HealthDisabled:         viper.GetBool(healthDisabledFlag),
		HealthTimeout:          time.Duration(viper.GetInt64(healthTimeoutFlag)) * time.Second,
		HealthInitialDelay:     time.Duration(viper.GetInt64(healthInitialDelayFlag)) * time.Second,
		HealthFailureThreshold: viper.GetInt64(healthFailureThresholdFlag),
	}
}

// config is all the configuration for this program
type config struct {
	Address                string
	IsLocal                bool
	HealthDisabled         bool
	HealthTimeout          time.Duration
	HealthInitialDelay     time.Duration
	HealthFailureThreshold int64
}
