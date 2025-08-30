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
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"agones.dev/agones/pkg"
	allocationpb "agones.dev/agones/pkg/allocation/go"
	"agones.dev/agones/pkg/client/clientset/versioned"
	"agones.dev/agones/pkg/util/httpserver"
	"agones.dev/agones/pkg/util/runtime"
	"agones.dev/agones/pkg/util/signals"

	"github.com/google/uuid"
	"github.com/heptiolabs/healthcheck"
	"github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	grpchealth "google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	leaderelection "k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
)

// TODO: tlacroix

const (
	logLevelFlag       = "log-level"
	leaderElectionFlag = "leader-election"
	podNamespace       = "pod-namespace"
	leaseDurationFlag  = "lease-duration"
	renewDeadlineFlag  = "renew-deadline"
	retryPeriodFlag    = "retry-period"

	grpcPortFlag                     = "grpc-port"
	apiServerBurstQPSFlag            = "api-server-qps-burst"
	apiServerSustainedQPSFlag        = "api-server-qps"
	totalRemoteAllocationTimeoutFlag = "total-remote-allocation-timeout"
	remoteAllocationTimeoutFlag      = "remote-allocation-timeout"
	allocationBatchWaitTime          = "allocation-batch-wait-time"
	httpUnallocatedStatusCode        = "http-unallocated-status-code"
	pullIntervalFlag                 = "pull-interval"
)

var (
	logger = runtime.NewLoggerWithSource("main")
)

// TODO: tlacroix Add new config to deployments etc.
type processorConfig struct {
	LogLevel                     string
	PodNamespace                 string
	LeaderElection               bool
	GRPCPort                     int
	APIServerBurstQPS            int
	APIServerSustainedQPS        int
	LeaseDuration                time.Duration
	RenewDeadline                time.Duration
	RetryPeriod                  time.Duration
	totalRemoteAllocationTimeout time.Duration
	remoteAllocationTimeout      time.Duration
	allocationBatchWaitTime      time.Duration
	pullInterval                 time.Duration
	httpUnallocatedStatusCode    int
}

func parseEnvFlags() processorConfig {
	viper.SetDefault(logLevelFlag, "Info")
	viper.SetDefault(grpcPortFlag, 9090)
	viper.SetDefault(apiServerSustainedQPSFlag, 400)
	viper.SetDefault(apiServerBurstQPSFlag, 500)
	viper.SetDefault(remoteAllocationTimeoutFlag, 10*time.Second)
	viper.SetDefault(totalRemoteAllocationTimeoutFlag, 30*time.Second)
	viper.SetDefault(allocationBatchWaitTime, 50*time.Millisecond)
	viper.SetDefault(leaderElectionFlag, false)
	viper.SetDefault(podNamespace, "")
	viper.SetDefault(leaseDurationFlag, 15*time.Second)
	viper.SetDefault(renewDeadlineFlag, 10*time.Second)
	viper.SetDefault(retryPeriodFlag, 2*time.Second)
	viper.SetDefault(httpUnallocatedStatusCode, http.StatusTooManyRequests)
	viper.SetDefault(pullIntervalFlag, 200*time.Millisecond)

	pflag.String(logLevelFlag, viper.GetString(logLevelFlag), "Log level")
	pflag.Int32(grpcPortFlag, viper.GetInt32(grpcPortFlag), "Port to listen on for gRPC requests")
	pflag.Int32(apiServerSustainedQPSFlag, viper.GetInt32(apiServerSustainedQPSFlag), "Maximum sustained queries per second to send to the API server")
	pflag.Int32(apiServerBurstQPSFlag, viper.GetInt32(apiServerBurstQPSFlag), "Maximum burst queries per second to send to the API server")
	pflag.Bool(leaderElectionFlag, viper.GetBool(leaderElectionFlag), "Enable leader election")
	pflag.String(podNamespace, viper.GetString(podNamespace), "Pod namespace")
	pflag.Duration(remoteAllocationTimeoutFlag, viper.GetDuration(remoteAllocationTimeoutFlag), "Flag to set remote allocation call timeout.")
	pflag.Duration(totalRemoteAllocationTimeoutFlag, viper.GetDuration(totalRemoteAllocationTimeoutFlag), "Flag to set total remote allocation timeout including retries.")
	pflag.Duration(leaseDurationFlag, viper.GetDuration(leaseDurationFlag), "Leader election lease duration")
	pflag.Duration(renewDeadlineFlag, viper.GetDuration(renewDeadlineFlag), "Leader election renew deadline")
	pflag.Duration(allocationBatchWaitTime, viper.GetDuration(allocationBatchWaitTime), "Flag to configure the waiting period between allocations batches")
	pflag.Duration(retryPeriodFlag, viper.GetDuration(retryPeriodFlag), "Leader election retry period")
	pflag.Int32(httpUnallocatedStatusCode, viper.GetInt32(httpUnallocatedStatusCode), "HTTP status code to return when no GameServer is available")
	pflag.Duration(pullIntervalFlag, viper.GetDuration(pullIntervalFlag), "Interval between pull requests sent to processor clients")
	pflag.Parse()

	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	runtime.Must(viper.BindEnv(grpcPortFlag))
	runtime.Must(viper.BindEnv(leaderElectionFlag))
	runtime.Must(viper.BindEnv(podNamespace))
	runtime.Must(viper.BindEnv(leaseDurationFlag))
	runtime.Must(viper.BindEnv(renewDeadlineFlag))
	runtime.Must(viper.BindEnv(retryPeriodFlag))
	runtime.Must(viper.BindEnv(apiServerSustainedQPSFlag))
	runtime.Must(viper.BindEnv(apiServerBurstQPSFlag))
	runtime.Must(viper.BindEnv(remoteAllocationTimeoutFlag))
	runtime.Must(viper.BindEnv(totalRemoteAllocationTimeoutFlag))
	runtime.Must(viper.BindEnv(logLevelFlag))
	runtime.Must(viper.BindEnv(allocationBatchWaitTime))
	runtime.Must(viper.BindEnv(httpUnallocatedStatusCode))
	runtime.Must(runtime.FeaturesBindEnv())

	runtime.Must(runtime.ParseFeaturesFromEnv())

	return processorConfig{
		LogLevel:                     viper.GetString(logLevelFlag),
		LeaderElection:               viper.GetBool(leaderElectionFlag),
		PodNamespace:                 viper.GetString(podNamespace),
		LeaseDuration:                viper.GetDuration(leaseDurationFlag),
		RenewDeadline:                viper.GetDuration(renewDeadlineFlag),
		RetryPeriod:                  viper.GetDuration(retryPeriodFlag),
		GRPCPort:                     int(viper.GetInt32(grpcPortFlag)),
		APIServerSustainedQPS:        int(viper.GetInt32(apiServerSustainedQPSFlag)),
		APIServerBurstQPS:            int(viper.GetInt32(apiServerBurstQPSFlag)),
		remoteAllocationTimeout:      viper.GetDuration(remoteAllocationTimeoutFlag),
		totalRemoteAllocationTimeout: viper.GetDuration(totalRemoteAllocationTimeoutFlag),
		allocationBatchWaitTime:      viper.GetDuration(allocationBatchWaitTime),
		httpUnallocatedStatusCode:    int(viper.GetInt32(httpUnallocatedStatusCode)),
		pullInterval:                 viper.GetDuration(pullIntervalFlag),
	}
}

func main() {
	ctx, cancelCtx := context.WithCancel(context.Background())
	defer cancelCtx()

	conf := parseEnvFlags()

	logger.WithField("version", pkg.Version).WithField("processorConf", conf).
		WithField("featureGates", runtime.EncodeFeatures()).
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

	kubeClient, agonesClient, err := getClients(conf)
	if err != nil {
		logger.WithError(err).Fatal("could not create clients")
	}

	grpcUnallocatedStatusCode := grpcCodeFromHTTPStatus(conf.httpUnallocatedStatusCode)

	processorService := newServiceHandler(ctx, kubeClient, agonesClient, health, conf, grpcUnallocatedStatusCode)

	grpcHealth := grpchealth.NewServer()
	grpcHealth.SetServingStatus("processor", grpc_health_v1.HealthCheckResponse_NOT_SERVING)

	runGRPC(ctx, processorService, grpcHealth, conf.GRPCPort)

	go func() {
		healthserver.Handle("/", health)
		_ = healthserver.Run(context.Background(), 0)
	}()

	signals.NewSigTermHandler(func() {
		logger.Info("Pod shutdown has been requested, failing readiness check")
		grpcHealth.Shutdown()
		cancelCtx()
		os.Exit(0)
	})

	whenLeader(ctx, cancelCtx, logger, conf, kubeClient, func(_ context.Context) {
		logger.Info("Starting processor work as leader")
		grpcHealth.SetServingStatus("processor", grpc_health_v1.HealthCheckResponse_SERVING)
		processorService.startPullRequestTicker()
	})

	logger.Info("Processor exited gracefully.")
}

func whenLeader(ctx context.Context, cancel context.CancelFunc, logger *logrus.Entry,
	conf processorConfig, kubeClient *kubernetes.Clientset, start func(_ context.Context)) {

	logger.WithField("leaderElectionEnabled", conf.LeaderElection).Info("Leader election configuration")

	if !conf.LeaderElection {
		start(ctx)
		return
	}

	id := uuid.New().String()

	lock := &resourcelock.LeaseLock{
		LeaseMeta: metav1.ObjectMeta{
			Name:      "agones-processor-lock",
			Namespace: conf.PodNamespace,
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
		LeaseDuration:   conf.LeaseDuration,
		RenewDeadline:   conf.RenewDeadline,
		RetryPeriod:     conf.RetryPeriod,
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

func runGRPC(ctx context.Context, h *processorHandler, grpcHealth *grpchealth.Server, grpcPort int) {
	logger.WithField("port", grpcPort).Info("Running the grpc handler on port")
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", grpcPort))
	if err != nil {
		logger.WithError(err).Fatalf("failed to listen on TCP port %d", grpcPort)
		os.Exit(1)
	}

	grpcServer := grpc.NewServer(h.getGRPCServerOptions()...)
	allocationpb.RegisterProcessorServer(grpcServer, h)
	grpc_health_v1.RegisterHealthServer(grpcServer, grpcHealth)

	go func() {
		go func() {
			<-ctx.Done()
			grpcServer.GracefulStop()
		}()

		err := grpcServer.Serve(listener)
		if err != nil {
			logger.WithError(err).Fatal("allocation service crashed")
			os.Exit(1)
		}
		logger.Info("allocation server closed")
		os.Exit(0)

	}()
}

// Set up our client which we will use to call the API
func getClients(ctlConfig processorConfig) (*kubernetes.Clientset, *versioned.Clientset, error) {
	// Create the in-cluster config
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, nil, errors.New("Could not create in cluster config")
	}

	config.QPS = float32(ctlConfig.APIServerSustainedQPS)
	config.Burst = ctlConfig.APIServerBurstQPS

	// Access to the Agones resources through the Agones Clientset
	kubeClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, nil, errors.New("Could not create the kubernetes api clientset")
	}

	// Access to the Agones resources through the Agones Clientset
	agonesClient, err := versioned.NewForConfig(config)
	if err != nil {
		return nil, nil, errors.New("Could not create the agones api clientset")
	}
	return kubeClient, agonesClient, nil
}

// grpcCodeFromHTTPStatus converts an HTTP status code to the corresponding gRPC status code.
func grpcCodeFromHTTPStatus(httpUnallocatedStatusCode int) codes.Code {
	switch httpUnallocatedStatusCode {
	case http.StatusOK:
		return codes.OK
	case 499:
		return codes.Canceled
	case http.StatusInternalServerError:
		return codes.Internal
	case http.StatusBadRequest:
		return codes.InvalidArgument
	case http.StatusGatewayTimeout:
		return codes.DeadlineExceeded
	case http.StatusNotFound:
		return codes.NotFound
	case http.StatusConflict:
		return codes.AlreadyExists
	case http.StatusForbidden:
		return codes.PermissionDenied
	case http.StatusUnauthorized:
		return codes.Unauthenticated
	case http.StatusTooManyRequests:
		return codes.ResourceExhausted
	case http.StatusNotImplemented:
		return codes.Unimplemented
	case http.StatusServiceUnavailable:
		return codes.Unavailable
	default:
		logger.WithField("httpStatusCode", httpUnallocatedStatusCode).Warnf("received unknown http status code, defaulting to codes.ResourceExhausted / 429")
		return codes.ResourceExhausted
	}
}
