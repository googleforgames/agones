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

// Package framework is a package helping setting up end-to-end testing accross a
// Kubernetes cluster.
package framework

import (
	"fmt"
	"net"
	"time"

	"agones.dev/agones/pkg/apis/stable/v1alpha1"
	"agones.dev/agones/pkg/client/clientset/versioned"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	// required to use gcloud login see: https://github.com/kubernetes/client-go/issues/242
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/tools/clientcmd"
)

// Framework is a testing framework
type Framework struct {
	KubeClient      kubernetes.Interface
	AgonesClient    versioned.Interface
	GameServerImage string
}

// New setups a testing framework using a kubeconfig path and the game server image to use for testing.
func New(kubeconfig, gsimage string) (*Framework, error) {
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, errors.Wrap(err, "build config from flags failed")
	}

	kubeClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, errors.Wrap(err, "creating new kube-client failed")
	}

	agonesClient, err := versioned.NewForConfig(config)
	if err != nil {
		return nil, errors.Wrap(err, "creating new agones-client failed")
	}

	return &Framework{
		KubeClient:      kubeClient,
		AgonesClient:    agonesClient,
		GameServerImage: gsimage,
	}, nil
}

// CreateGameServerAndWaitUntilReady Creates a GameServer and wait for its state to become ready.
func (f *Framework) CreateGameServerAndWaitUntilReady(ns string, gs *v1alpha1.GameServer) (*v1alpha1.GameServer, error) {
	newGs, err := f.AgonesClient.StableV1alpha1().GameServers(ns).Create(gs)
	if err != nil {
		return nil, fmt.Errorf("creating %v GameServer instances failed (%v): %v", gs.Spec, gs.Name, err)
	}

	readyGs, err := f.WaitForGameServerState(newGs, v1alpha1.Ready, 5*time.Minute)

	if err != nil {
		return nil, fmt.Errorf("waiting for %v GameServer instance readiness timed out (%v): %v",
			gs.Spec, gs.Name, err)
	}

	return readyGs, nil
}

// WaitForGameServerState Waits untils the gameserver reach a given state before the timeout expires
func (f *Framework) WaitForGameServerState(gs *v1alpha1.GameServer, state v1alpha1.State,
	timeout time.Duration) (*v1alpha1.GameServer, error) {
	var pollErr error
	var readyGs *v1alpha1.GameServer

	err := wait.PollImmediate(2*time.Second, timeout, func() (bool, error) {
		readyGs, pollErr = f.AgonesClient.StableV1alpha1().GameServers(gs.Namespace).Get(gs.Name, v1.GetOptions{})

		if pollErr != nil {
			return false, nil
		}

		if readyGs.Status.State == state {
			return true, nil
		}

		return false, nil
	})
	if err != nil {
		return nil, errors.Wrapf(pollErr, "waiting for GameServer to be %v %v/%v: %v",
			state, gs.Namespace, gs.Name, err)
	}
	return readyGs, nil
}

// CleanUp Delete all agones resources in a given namespace
func (f *Framework) CleanUp(ns string) error {
	return f.AgonesClient.StableV1alpha1().GameServers(ns).
		DeleteCollection(&v1.DeleteOptions{}, v1.ListOptions{})
}

// PingGameServer pings a gameserver and returns its reply
func PingGameServer(msg, address string) (reply string, err error) {
	conn, err := net.Dial("udp", address)
	if err != nil {
		return "", err
	}
	defer func() {
		err = conn.Close()
	}()
	_, err = conn.Write([]byte(msg))
	if err != nil {
		return "", errors.Wrapf(err, "Could not write message %s", msg)
	}
	b := make([]byte, 1024)
	n, err := conn.Read(b)
	if err != nil {
		return "", err
	}
	return string(b[:n]), nil
}
