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

// Controller for gameservers
package main

import (
	"strings"
	"time"

	"github.com/agonio/agon/pkg"
	"github.com/agonio/agon/pkg/client/clientset/versioned"
	"github.com/agonio/agon/pkg/client/informers/externalversions"
	"github.com/agonio/agon/pkg/signals"
	"github.com/agonio/agon/pkg/util/runtime"
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
)

func init() {
	logrus.SetFormatter(&logrus.JSONFormatter{})
}

// main starts the operator for the gameserver CRD
func main() {
	viper.SetDefault(sidecarFlag, "gcr.io/agon-images/gameservers-sidecar:"+pkg.Version)
	viper.SetDefault(pullSidecarFlag, false)

	pflag.String(sidecarFlag, viper.GetString(sidecarFlag), "Flag to overwrite the GameServer sidecar image that is used. Can also use SIDECAR env variable")
	pflag.Bool(pullSidecarFlag, viper.GetBool(pullSidecarFlag), "For development purposes, set the sidecar image to have a ImagePullPolicy of Always. Can also use ALWAYS_PULL_SIDECAR env variable")
	pflag.Parse()

	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	runtime.Must(viper.BindEnv(sidecarFlag))
	runtime.Must(viper.BindEnv(pullSidecarFlag))
	runtime.Must(viper.BindPFlags(pflag.CommandLine))

	sidecarImage := viper.GetString(sidecarFlag)
	alwaysPullSidecar := viper.GetBool(pullSidecarFlag)

	logrus.WithField(sidecarFlag, sidecarImage).
		WithField("alwaysPullSidecarImage", alwaysPullSidecar).
		WithField("Version", pkg.Version).Info("starting gameServer operator...")

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

	agonClient, err := versioned.NewForConfig(config)
	if err != nil {
		logrus.WithError(err).Fatal("Could not create the agon api clientset")
	}

	agonInformerFactory := externalversions.NewSharedInformerFactory(agonClient, 30*time.Second)
	kubeInformationFactory := informers.NewSharedInformerFactory(kubeClient, 30*time.Second)
	c := NewController(sidecarImage, alwaysPullSidecar, kubeClient, kubeInformationFactory, extClient, agonClient, agonInformerFactory)

	stop := signals.NewStopChannel()

	kubeInformationFactory.Start(stop)
	agonInformerFactory.Start(stop)

	err = c.Run(2, stop)
	if err != nil {
		logrus.WithError(err).Fatal("Could not run gameserver controller")
	}

	logrus.Info("Shut down gameserver controller")
}
