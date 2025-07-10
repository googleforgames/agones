// Copyright 2018 Google LLC All Rights Reserved.
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

	"github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"agones.dev/agones/pkg"
	"agones.dev/agones/pkg/client/clientset/versioned"
	"agones.dev/agones/pkg/client/informers/externalversions"
	"agones.dev/agones/pkg/gameserverallocations"
	"agones.dev/agones/pkg/gameserverallocations/distributedallocator"
	"agones.dev/agones/pkg/gameservers"
	"agones.dev/agones/pkg/util/httpserver"
	"agones.dev/agones/pkg/util/leader"
	"agones.dev/agones/pkg/util/runtime"
	"github.com/heptiolabs/healthcheck"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const (
	grpcPortFlag                 = "grpc-port"
	logLevelFlag                 = "log-level"
	remoteAllocationTimeout      = "remote-allocation-timeout"
	totalRemoteAllocationTimeout = "total-remote-allocation-timeout"
	batchWaitTimeFlag            = "batch-wait-time"
)

func parseEnvFlags() distributedallocator.ProcessorConfig {
	viper.SetDefault(grpcPortFlag, 8443)
	viper.SetDefault(logLevelFlag, "Info")
	viper.SetDefault(remoteAllocationTimeout, 2*time.Second)
	viper.SetDefault(totalRemoteAllocationTimeout, 10*time.Second)
	viper.SetDefault(batchWaitTimeFlag, 100*time.Millisecond)

	pflag.Int(grpcPortFlag, viper.GetInt(grpcPortFlag), "Port to listen on for gRPC requests")
	pflag.String(logLevelFlag, viper.GetString(logLevelFlag), "Log level")
	pflag.Duration(remoteAllocationTimeout, viper.GetDuration(remoteAllocationTimeout), "Remote allocation call timeout")
	pflag.Duration(totalRemoteAllocationTimeout, viper.GetDuration(totalRemoteAllocationTimeout), "Total remote allocation timeout including retries")
	pflag.Duration(batchWaitTimeFlag, viper.GetDuration(batchWaitTimeFlag), "Batch wait time for batch aggregation")
	pflag.Parse()

	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.AutomaticEnv()
	_ = viper.BindPFlags(pflag.CommandLine)

	return distributedallocator.ProcessorConfig{
		GRPCPort:                     viper.GetInt(grpcPortFlag),
		LogLevel:                     viper.GetString(logLevelFlag),
		RemoteAllocationTimeout:      viper.GetDuration(remoteAllocationTimeout),
		TotalRemoteAllocationTimeout: viper.GetDuration(totalRemoteAllocationTimeout),
		BatchWaitTime:                viper.GetDuration(batchWaitTimeFlag),
	}
}

func main() {
	conf := parseEnvFlags()

	logger := runtime.NewLoggerWithSource("main")
	logger.WithField("version", pkg.Version).WithField("ctlConf", conf).
		Info("Starting agones-processor")

	logger.WithField("logLevel", conf.LogLevel).Info("Setting LogLevel configuration")
	level, err := logrus.ParseLevel(strings.ToLower(conf.LogLevel))
	if err == nil {
		runtime.SetLevel(level)
	} else {
		logger.WithError(err).Info("Specified wrong Logging. Setting default loglevel - Info")
		runtime.SetLevel(logrus.InfoLevel)
	}

	if conf.GRPCPort <= 0 || conf.GRPCPort > 65535 {
		logger.WithField("grpc-port", conf.GRPCPort).Fatal("Must specify a valid gRPC port for the processor service")
	}

	kubeClient, agonesClient, err := getClients()
	if err != nil {
		logger.WithError(err).Fatal("could not create clients")
	}

	defaultResync := 30 * time.Second
	informerFactory := informers.NewSharedInformerFactory(kubeClient, defaultResync)
	agonesInformerFactory := externalversions.NewSharedInformerFactory(agonesClient, defaultResync)
	policyInformer := agonesInformerFactory.Multicluster().V1().GameServerAllocationPolicies()
	secretInformer := informerFactory.Core().V1().Secrets()
	gameServerGetter := agonesClient.AgonesV1()
	gsCounter := gameservers.NewPerNodeCounter(informerFactory, agonesInformerFactory)

	healthserver := &httpserver.Server{Logger: logger}
	health := healthcheck.NewHandler()

	go func() {
		healthserver.Handle("/", health)
		_ = healthserver.Run(context.Background(), 0)
	}()

	allocationCache := gameserverallocations.NewAllocationCache(
		agonesInformerFactory.Agones().V1().GameServers(),
		gsCounter,
		health,
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	allocator := gameserverallocations.NewAllocator(
		policyInformer,
		secretInformer,
		gameServerGetter,
		kubeClient,
		allocationCache,
		conf.RemoteAllocationTimeout,
		conf.TotalRemoteAllocationTimeout,
		conf.BatchWaitTime,
	)

	informerFactory.Start(ctx.Done())
	agonesInformerFactory.Start(ctx.Done())

	processorRuntime := distributedallocator.NewProcessorRuntime(conf.GRPCPort, logger, allocator)

	go func() {
		ch := make(chan os.Signal, 1)
		signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
		<-ch
		cancel()
	}()

	namespace := os.Getenv("POD_NAMESPACE")
	lockName := "agones-processor-leader-election"

	hostname, err := os.Hostname()
	if err != nil {
		logger.WithError(err).Fatal("Failed to get hostname")
	}

	err = leader.RunLeaderElection(
		ctx, kubeClient, hostname, lockName, namespace,
		processorRuntime.OnStartedLeading,
		processorRuntime.OnStoppedLeading,
	)
	if err != nil {
		logger.WithError(err).Fatal("Processor exited with error")
	}

	logger.Info("Processor exited gracefully.")
}

func getClients() (*kubernetes.Clientset, *versioned.Clientset, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, nil, err
	}
	kubeClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, nil, err
	}
	agonesClient, err := versioned.NewForConfig(config)
	if err != nil {
		return nil, nil, err
	}
	return kubeClient, agonesClient, nil
}
