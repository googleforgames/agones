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
	"strings"
	"time"

	"agones.dev/agones/pkg"
	"agones.dev/agones/pkg/util/httpserver"
	"agones.dev/agones/pkg/util/runtime"
	"agones.dev/agones/pkg/util/signals"

	"github.com/google/uuid"
	"github.com/heptiolabs/healthcheck"
	"github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	leaderelection "k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
)

const (
	logLevelFlag       = "log-level"
	leaderElectionFlag = "leader-election"
	podNamespace       = "pod-namespace"
)

type processorConfig struct {
	LogLevel       string
	LeaderElection bool
	PodNamespace   string
}

func parseEnvFlags() processorConfig {
	viper.SetDefault(logLevelFlag, "Info")
	viper.SetDefault(leaderElectionFlag, false)
	viper.SetDefault(podNamespace, "")

	pflag.String(logLevelFlag, viper.GetString(logLevelFlag), "Log level")
	pflag.Bool(leaderElectionFlag, viper.GetBool(leaderElectionFlag), "Enable leader election")
	pflag.String(podNamespace, viper.GetString(podNamespace), "Pod namespace")
	pflag.Parse()

	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.AutomaticEnv()
	_ = viper.BindPFlags(pflag.CommandLine)

	return processorConfig{
		LogLevel:       viper.GetString(logLevelFlag),
		LeaderElection: viper.GetBool(leaderElectionFlag),
		PodNamespace:   viper.GetString(podNamespace),
	}
}

func main() {
	ctx, cancelCtx := context.WithCancel(context.Background())
	defer cancelCtx()

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

	config, err := rest.InClusterConfig()
	if err != nil {
		logger.WithError(err).Fatal("Failed to create in-cluster config")
		panic("Failed to create in-cluster config: " + err.Error())
	}
	kubeClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		logger.WithError(err).Fatal("Failed to create Kubernetes client")
		panic("Failed to create Kubernetes client: " + err.Error())
	}

	go func() {
		healthserver.Handle("/", health)
		_ = healthserver.Run(context.Background(), 0)
	}()

	signals.NewSigTermHandler(func() {
		logger.Info("Pod shutdown has been requested, failing readiness check")
		cancelCtx()
		time.Sleep(1 * time.Second)
		os.Exit(0)
	})

	whenLeader(ctx, cancelCtx, logger, conf.LeaderElection, kubeClient, conf.PodNamespace, func(ctx context.Context) {
		logger.Info("Starting processor work as leader")

		// Simulate processor work (to ensure the leader is working)
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				logger.Info("Processor work completed")
				return
			case <-ticker.C:
				logger.Info("Processor is active as leader")
			}
		}
	})

	logger.Info("Processor exited gracefully.")
}

func whenLeader(ctx context.Context, cancel context.CancelFunc, logger *logrus.Entry, doLeaderElection bool, kubeClient *kubernetes.Clientset, namespace string, start func(_ context.Context)) {
	if !doLeaderElection {
		start(ctx)
		return
	}

	id := uuid.New().String()

	lock := &resourcelock.LeaseLock{
		LeaseMeta: metav1.ObjectMeta{
			Name:      "agones-processor-lock",
			Namespace: namespace,
		},
		Client: kubeClient.CoordinationV1(),
		LockConfig: resourcelock.ResourceLockConfig{
			Identity: id,
		},
	}

	logger.WithField("id", id).Info("Leader Election ID")

	leaderelection.RunOrDie(ctx, leaderelection.LeaderElectionConfig{
		Lock: lock,
		// IMPORTANT: you MUST ensure that any code you have that
		// is protected by the lease must terminate **before**
		// you call cancel. Otherwise, you could have a background
		// loop still running and another process could
		// get elected before your background loop finished, violating
		// the stated goal of the lease.
		ReleaseOnCancel: true,
		LeaseDuration:   15 * time.Second,
		RenewDeadline:   10 * time.Second,
		RetryPeriod:     2 * time.Second,
		Callbacks: leaderelection.LeaderCallbacks{
			OnStartedLeading: start,
			OnStoppedLeading: func() {
				logger.WithField("id", id).Info("Leader Lost")
				cancel()
				os.Exit(0)
			},
			OnNewLeader: func(identity string) {
				if identity == id {
					return
				}
				logger.WithField("id", id).Info("New Leader Elected")
			},
		},
	})
}
