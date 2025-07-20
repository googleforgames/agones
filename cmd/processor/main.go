// Copyright 2025 Google LLC All Rights Reserved.
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

// Processor
package main

import (
	"context"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"agones.dev/agones/pkg"
	"agones.dev/agones/pkg/util/httpserver"
	"agones.dev/agones/pkg/util/runtime"

	"github.com/heptiolabs/healthcheck"
	"github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	leaderelection "k8s.io/client-go/tools/leaderelection"
	leaderelectionresourcelock "k8s.io/client-go/tools/leaderelection/resourcelock"
)

const (
	logLevelFlag = "log-level"
)

type processorConfig struct {
	LogLevel string
}

func parseEnvFlags() processorConfig {
	viper.SetDefault(logLevelFlag, "Info")

	pflag.String(logLevelFlag, viper.GetString(logLevelFlag), "Log level")
	pflag.Parse()

	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.AutomaticEnv()
	_ = viper.BindPFlags(pflag.CommandLine)

	return processorConfig{
		LogLevel: viper.GetString(logLevelFlag),
	}
}

func main() {
	conf := parseEnvFlags()

	logger := runtime.NewLoggerWithSource("main")
	logger.WithField("version", pkg.Version).WithField("processorConf", conf).
		Info("Starting agones-processor")

	logger.WithField("logLevel", conf.LogLevel).Info("Setting LogLevel configuration")
	level, err := logrus.ParseLevel(strings.ToLower(conf.LogLevel))
	if err == nil {
		runtime.SetLevel(level)
	} else {
		logger.WithError(err).Info("Specified wrong Logging. Setting default loglevel - Info")
		runtime.SetLevel(logrus.InfoLevel)
	}

	healthserver := &httpserver.Server{Logger: logger}
	health := healthcheck.NewHandler()

	ctx := context.Background()
	config, err := rest.InClusterConfig()
	if err != nil {
		panic("Failed to create in-cluster config: " + err.Error())
	}
	kubeClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic("Failed to create Kubernetes client: " + err.Error())
	}
	namespace := os.Getenv("POD_NAMESPACE")
	lockName := "agones-processor-leader-election"
	hostname, err := os.Hostname()
	if err != nil {
		logger.WithError(err).Fatal("Failed to get hostname")
		panic("Failed to get hostname")
	}

	lock := &leaderelectionresourcelock.LeaseLock{
		LeaseMeta: metav1.ObjectMeta{
			Name:      lockName,
			Namespace: namespace,
		},
		Client: kubeClient.CoordinationV1(),
		LockConfig: leaderelectionresourcelock.ResourceLockConfig{
			Identity: hostname,
		},
	}

	leaderElection, err := leaderelection.NewLeaderElector(leaderelection.LeaderElectionConfig{
		Lock:          lock,
		LeaseDuration: 15 * time.Second,
		RenewDeadline: 10 * time.Second,
		RetryPeriod:   2 * time.Second,
		Callbacks: leaderelection.LeaderCallbacks{
			OnStartedLeading: func(ctx context.Context) {
				logger.Debug("Processor is now the leader")
			},
			OnStoppedLeading: func() {
				logger.Debug("Processor has stopped leading")
			},
			OnNewLeader: func(newIdentity string) {
				if newIdentity == hostname {
					logger.Debug("Processor has become the leader")
				} else {
					logger.Debug("Processor has a new leader: ", newIdentity)
				}
			},
		},
		ReleaseOnCancel: true,
	})

	go leaderElection.Run(ctx)

	go func() {
		healthserver.Handle("/", health)
		_ = healthserver.Run(context.Background(), 0)
	}()

	go func() {
		ch := make(chan os.Signal, 1)
		signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
		<-ch
	}()

	logger.Info("Processor exited gracefully.")
}
