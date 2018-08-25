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
	"os"
	"path/filepath"
	"strings"

	"agones.dev/agones/pkg"
	"agones.dev/agones/pkg/apis/stable/v1alpha1"
	"agones.dev/agones/pkg/client/clientset/versioned"
	"agones.dev/agones/pkg/gameservers"
	"agones.dev/agones/pkg/sdk"
	"agones.dev/agones/pkg/util/runtime"
	"agones.dev/agones/pkg/util/signals"
	gwruntime "github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/pkg/errors"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"k8s.io/apimachinery/pkg/util/yaml"
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
	localFlag   = "local"
	fileFlag    = "file"
	addressFlag = "address"
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
		err = registerLocal(grpcServer, ctlConf)
		if err != nil {
			logger.WithError(err).Fatal("Could not start local sdk server")
		}
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
		s, err = gameservers.NewSDKServer(viper.GetString(gameServerNameEnv),
			viper.GetString(podNamespaceEnv), kubeClient, agonesClient)
		if err != nil {
			logger.WithError(err).Fatalf("Could not start sidecar")
		}

		go func() {
			err := s.Run(ctx.Done())
			if err != nil {
				logger.WithError(err).Fatalf("Could not run sidecar")
			}
		}()
		sdk.RegisterSDKServer(grpcServer, s)
	}

	go runGrpc(grpcServer, lis)
	go runGateway(ctx, grpcEndpoint, mux, httpServer)

	<-stop
	logger.Info("shutting down sdk server")
}

func registerLocal(grpcServer *grpc.Server, ctlConf config) error {
	var local *gameservers.LocalSDKServer
	if ctlConf.LocalFile != "" {
		path, err := filepath.Abs(ctlConf.LocalFile)
		if err != nil {
			return err
		}

		if _, err = os.Stat(path); os.IsNotExist(err) {
			return errors.Errorf("Could not find file: %s", path)
		}

		logger.WithField("path", path).Info("Reading GameServer configuration")
		reader, err := os.Open(path) // nolint: gosec
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
		local = gameservers.NewLocalSDKServer(&gs)
	} else {
		local = gameservers.NewLocalSDKServer(nil)
	}

	sdk.RegisterSDKServer(grpcServer, local)

	return nil
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
	viper.SetDefault(fileFlag, "")
	viper.SetDefault(addressFlag, "localhost")
	pflag.Bool(localFlag, viper.GetBool(localFlag),
		"Set this, or LOCAL env, to 'true' to run this binary in local development mode. Defaults to 'false'")
	pflag.StringP(fileFlag, "f", viper.GetString(fileFlag), "Set this, or FILE env var to the path of a local yaml or json file that contains your GameServer resoure configuration")
	pflag.String(addressFlag, viper.GetString(addressFlag), "The Address to bind the server grpcPort to. Defaults to 'localhost")
	pflag.Parse()

	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	runtime.Must(viper.BindEnv(localFlag))
	runtime.Must(viper.BindEnv(gameServerNameEnv))
	runtime.Must(viper.BindEnv(podNamespaceEnv))
	runtime.Must(viper.BindPFlags(pflag.CommandLine))

	return config{
		IsLocal:   viper.GetBool(localFlag),
		Address:   viper.GetString(addressFlag),
		LocalFile: viper.GetString(fileFlag),
	}
}

// config is all the configuration for this program
type config struct {
	Address   string
	IsLocal   bool
	LocalFile string
}
