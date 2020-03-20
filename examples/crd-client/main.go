// Copyright 2020 Google LLC All Rights Reserved.
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

package main

import (
	"fmt"
	"strings"

	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	"agones.dev/agones/pkg/client/clientset/versioned"
	"agones.dev/agones/pkg/util/runtime"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const (
	gameServerImage = "GAMESERVER_IMAGE"

	defaultImage = "gcr.io/agones-images/udp-server:0.19"
)

func main() {
	viper.AllowEmptyEnv(true)
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))

	pflag.String(gameServerImage, viper.GetString(gameServerImage), "The Address to bind the server grpcPort to. Defaults to 'localhost'")
	viper.SetDefault(gameServerImage, defaultImage)
	runtime.Must(viper.BindEnv(gameServerImage))
	viper.GetString(gameServerImage)
	config, err := rest.InClusterConfig()
	logger := runtime.NewLoggerWithSource("main")
	if err != nil {
		logger.WithError(err).Fatal("Could not create in cluster config")
	}

	// Access to standard Kubernetes resources through the Kubernetes Clientset
	// We don't actually need this for this example, but it's just here for
	// illustrative purposes
	kubeClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		logger.WithError(err).Fatal("Could not create the kubernetes clientset")
	}
	p := kubeClient.CoreV1().Pods("default")
	logger.Info(p)

	// Access to the Agones resources through the Agones Clientset
	// Note that we reuse the same config as we used for the Kubernetes Clientset
	agonesClient, err := versioned.NewForConfig(config)
	if err != nil {
		logger.WithError(err).Fatal("Could not create the agones api clientset")
	}

	// Create a GameServer
	gs := &agonesv1.GameServer{ObjectMeta: metav1.ObjectMeta{GenerateName: "udp-server", Namespace: "default"},
		Spec: agonesv1.GameServerSpec{
			Container: "udp-server",
			Ports: []agonesv1.GameServerPort{{
				ContainerPort: 7654,
				HostPort:      7654,
				Name:          "gameport",
				PortPolicy:    agonesv1.Static,
				Protocol:      corev1.ProtocolUDP,
			}},
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{Name: "udp-server", Image: viper.GetString(gameServerImage)}},
				},
			},
		},
	}
	newGS, err := agonesClient.AgonesV1().GameServers("default").Create(gs)
	if err != nil {
		panic(err)
	}

	fmt.Printf("New game servers' name is: %s", newGS.ObjectMeta.Name)
}
