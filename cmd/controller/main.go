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

// Controller for gameservers
package main

import (
	"strings"
	"time"

	"agones.dev/agones/pkg"
	"agones.dev/agones/pkg/client/clientset/versioned"
	"agones.dev/agones/pkg/client/informers/externalversions"
	"agones.dev/agones/pkg/gameservers"
	"agones.dev/agones/pkg/health"
	"agones.dev/agones/pkg/util/runtime"
	"agones.dev/agones/pkg/util/signals"
	"github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	extclientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const (
	sidecarFlag     = "sidecar"
	pullSidecarFlag = "always-pull-sidecar"
	minPortFlag     = "min-port"
	maxPortFlag     = "max-port"
)

func init() {
	logrus.SetFormatter(&logrus.JSONFormatter{})
}

// main starts the operator for the gameserver CRD
func main() {
	viper.SetDefault(sidecarFlag, "gcr.io/agones-images/agones-sdk:"+pkg.Version)
	viper.SetDefault(pullSidecarFlag, false)

	pflag.String(sidecarFlag, viper.GetString(sidecarFlag), "Flag to overwrite the GameServer sidecar image that is used. Can also use SIDECAR env variable")
	pflag.Bool(pullSidecarFlag, viper.GetBool(pullSidecarFlag), "For development purposes, set the sidecar image to have a ImagePullPolicy of Always. Can also use ALWAYS_PULL_SIDECAR env variable")
	pflag.Int32(minPortFlag, 0, "Required. The minimum port that that a GameServer can be allocated to. Can also use MIN_PORT env variable.")
	pflag.Int32(maxPortFlag, 0, "Required. The maximum port that that a GameServer can be allocated to. Can also use MAX_PORT env variable")
	pflag.Parse()

	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	runtime.Must(viper.BindEnv(sidecarFlag))
	runtime.Must(viper.BindEnv(pullSidecarFlag))
	runtime.Must(viper.BindEnv(minPortFlag))
	runtime.Must(viper.BindEnv(maxPortFlag))
	runtime.Must(viper.BindPFlags(pflag.CommandLine))

	minPort := int32(viper.GetInt64(minPortFlag))
	maxPort := int32(viper.GetInt64(maxPortFlag))
	sidecarImage := viper.GetString(sidecarFlag)
	alwaysPullSidecar := viper.GetBool(pullSidecarFlag)

	logrus.WithField(sidecarFlag, sidecarImage).
		WithField("minPort", minPort).
		WithField("maxPort", maxPort).
		WithField("alwaysPullSidecarImage", alwaysPullSidecar).
		WithField("Version", pkg.Version).Info("starting gameServer operator...")

	if minPort <= 0 || maxPort <= 0 {
		logrus.Fatal("Min Port and Max Port values are required.")
	} else if maxPort < minPort {
		logrus.Fatal("Max Port cannot be set less that the Min Port")
	}

	config, err := rest.InClusterConfig()
	if err != nil {
		logrus.WithError(err).Fatal("Could not create in cluster config")
	}

	kubeClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		logrus.WithError(err).Fatal("Could not create the kubernetes clientset")
	}

	extClient, err := extclientset.NewForConfig(config)
	if err != nil {
		logrus.WithError(err).Fatal("Could not create the api extension clientset")
	}

	agonesClient, err := versioned.NewForConfig(config)
	if err != nil {
		logrus.WithError(err).Fatal("Could not create the agones api clientset")
	}

	agonesInformerFactory := externalversions.NewSharedInformerFactory(agonesClient, 30*time.Second)
	kubeInformationFactory := informers.NewSharedInformerFactory(kubeClient, 30*time.Second)
	c := gameservers.NewController(
		minPort,
		maxPort,
		sidecarImage,
		alwaysPullSidecar,
		kubeClient,
		kubeInformationFactory,
		extClient,
		agonesClient,
		agonesInformerFactory,
		health.NewMonitor(kubeClient, agonesClient, kubeInformationFactory, agonesInformerFactory),
	)

	stop := signals.NewStopChannel()

	kubeInformationFactory.Start(stop)
	agonesInformerFactory.Start(stop)

	err = c.Run(2, stop)
	if err != nil {
		logrus.WithError(err).Fatal("Could not run gameserver controller")
	}

	logrus.Info("Shut down gameserver controller")
}
