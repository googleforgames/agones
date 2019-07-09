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

// Package framework is a package helping setting up end-to-end testing across a
// Kubernetes cluster.
package framework

import (
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	allocationv1alpha1 "agones.dev/agones/pkg/apis/allocation/v1alpha1"
	autoscaling "agones.dev/agones/pkg/apis/autoscaling/v1"
	stable "agones.dev/agones/pkg/apis/stable/v1alpha1"
	"agones.dev/agones/pkg/client/clientset/versioned"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"

	// required to use gcloud login see: https://github.com/kubernetes/client-go/issues/242
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/tools/clientcmd"
)

// special labels that can be put on pods to trigger automatic cleanup.
const (
	AutoCleanupLabelKey   = "stable.agones.dev/e2e-test-auto-cleanup"
	AutoCleanupLabelValue = "true"
)

// Framework is a testing framework
type Framework struct {
	KubeClient      kubernetes.Interface
	AgonesClient    versioned.Interface
	GameServerImage string
	PullSecret      string
	StressTestLevel int
	PerfOutputDir   string
}

// New setups a testing framework using a kubeconfig path and the game server image to use for testing.
func New(kubeconfig string) (*Framework, error) {
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
		KubeClient:   kubeClient,
		AgonesClient: agonesClient,
	}, nil
}

// CreateGameServerAndWaitUntilReady Creates a GameServer and wait for its state to become ready.
func (f *Framework) CreateGameServerAndWaitUntilReady(ns string, gs *stable.GameServer) (*stable.GameServer, error) {
	newGs, err := f.AgonesClient.StableV1alpha1().GameServers(ns).Create(gs)
	if err != nil {
		return nil, fmt.Errorf("creating %v GameServer instances failed (%v): %v", gs.Spec, gs.Name, err)
	}

	logrus.WithField("name", newGs.ObjectMeta.Name).Info("GameServer created, waiting for Ready")

	readyGs, err := f.WaitForGameServerState(newGs, stable.GameServerStateReady, 5*time.Minute)

	if err != nil {
		return nil, fmt.Errorf("waiting for %v GameServer instance readiness timed out (%v): %v",
			gs.Spec, gs.Name, err)
	}
	if len(readyGs.Status.Ports) == 0 {
		return nil, fmt.Errorf("Ready GameServer instance has no port: %v", readyGs.Status)
	}

	logrus.WithField("name", newGs.ObjectMeta.Name).Info("GameServer Ready")

	return readyGs, nil
}

// WaitForGameServerState Waits untils the gameserver reach a given state before the timeout expires
func (f *Framework) WaitForGameServerState(gs *stable.GameServer, state stable.GameServerState,
	timeout time.Duration) (*stable.GameServer, error) {
	var readyGs *stable.GameServer

	err := wait.PollImmediate(2*time.Second, timeout, func() (bool, error) {
		var err error
		readyGs, err = f.AgonesClient.StableV1alpha1().GameServers(gs.Namespace).Get(gs.Name, metav1.GetOptions{})

		if err != nil {
			logrus.WithError(err).Warn("error retrieving gameserver")
			return false, nil
		}

		if readyGs.Status.State == state {
			return true, nil
		}

		return false, nil
	})

	return readyGs, errors.Wrapf(err, "waiting for GameServer to be %v %v/%v",
		state, gs.Namespace, gs.Name)
}

// WaitForFleetCondition waits for the Fleet to be in a specific condition or fails the test if the condition can't be met in 5 minutes.
// nolint: dupl
func (f *Framework) WaitForFleetCondition(t *testing.T, flt *stable.Fleet, condition func(fleet *stable.Fleet) bool) {
	t.Helper()
	logrus.WithField("fleet", flt.Name).Info("waiting for fleet condition")
	err := wait.PollImmediate(2*time.Second, 5*time.Minute, func() (bool, error) {
		fleet, err := f.AgonesClient.StableV1alpha1().Fleets(flt.ObjectMeta.Namespace).Get(flt.ObjectMeta.Name, metav1.GetOptions{})
		if err != nil {
			return true, err
		}

		return condition(fleet), nil
	})
	if err != nil {
		logrus.WithField("fleet", flt.Name).WithError(err).Info("error waiting for fleet condition")
		t.Fatalf("error waiting for fleet condition on fleet %v", flt.Name)
	}
}

// WaitForFleetAutoScalerCondition waits for the FleetAutoscaler to be in a specific condition or fails the test if the condition can't be met in 2 minutes.
// nolint: dupl
func (f *Framework) WaitForFleetAutoScalerCondition(t *testing.T, fas *autoscaling.FleetAutoscaler, condition func(fas *autoscaling.FleetAutoscaler) bool) {
	t.Helper()
	logrus.WithField("fleetautoscaler", fas.Name).Info("waiting for fleetautoscaler condition")
	err := wait.PollImmediate(2*time.Second, 2*time.Minute, func() (bool, error) {
		fleetautoscaler, err := f.AgonesClient.AutoscalingV1().FleetAutoscalers(fas.ObjectMeta.Namespace).Get(fas.ObjectMeta.Name, metav1.GetOptions{})
		if err != nil {
			return true, err
		}

		return condition(fleetautoscaler), nil
	})
	if err != nil {
		logrus.WithField("fleetautoscaler", fas.Name).WithError(err).Info("error waiting for fleetautoscaler condition")
		t.Fatalf("error waiting for fleetautoscaler condition on fleetautoscaler %v", fas.Name)
	}
}

// ListGameServersFromFleet lists GameServers from a particular fleet
func (f *Framework) ListGameServersFromFleet(flt *stable.Fleet) ([]stable.GameServer, error) {
	var results []stable.GameServer

	opts := metav1.ListOptions{LabelSelector: labels.Set{stable.FleetNameLabel: flt.ObjectMeta.Name}.String()}
	gsSetList, err := f.AgonesClient.StableV1alpha1().GameServerSets(flt.ObjectMeta.Namespace).List(opts)
	if err != nil {
		return results, err
	}

	for _, gsSet := range gsSetList.Items {
		opts := metav1.ListOptions{LabelSelector: labels.Set{stable.GameServerSetGameServerLabel: gsSet.ObjectMeta.Name}.String()}
		gsList, err := f.AgonesClient.StableV1alpha1().GameServers(flt.ObjectMeta.Namespace).List(opts)
		if err != nil {
			return results, err
		}

		results = append(results, gsList.Items...)
	}

	return results, nil
}

// FleetReadyCount returns the ready count in a fleet
func FleetReadyCount(amount int32) func(fleet *stable.Fleet) bool {
	return func(fleet *stable.Fleet) bool {
		logrus.Infof("fleet %v has %v/%v ready replicas", fleet.Name, fleet.Status.ReadyReplicas, amount)
		return fleet.Status.ReadyReplicas == amount
	}
}

// WaitForFleetGameServersCondition waits for all GameServers for a given fleet to match
// a condition specified by a callback.
func (f *Framework) WaitForFleetGameServersCondition(flt *stable.Fleet,
	cond func(server stable.GameServer) bool) error {
	return f.WaitForFleetGameServerListCondition(flt,
		func(servers []stable.GameServer) bool {
			for _, gs := range servers {
				if !cond(gs) {
					return false
				}
			}
			return true
		})
}

// WaitForFleetGameServerListCondition waits for the list of GameServers to match a condition
// specified by a callback and the size of GameServers to match fleet's Spec.Replicas.
func (f *Framework) WaitForFleetGameServerListCondition(flt *stable.Fleet,
	cond func(servers []stable.GameServer) bool) error {
	return wait.Poll(2*time.Second, 5*time.Minute, func() (done bool, err error) {
		gsList, err := f.ListGameServersFromFleet(flt)
		if err != nil {
			return false, err
		}
		if int32(len(gsList)) != flt.Spec.Replicas {
			return false, nil
		}
		return cond(gsList), nil
	})
}

// NewStatsCollector returns new instance of statistics collector,
// which can be used to emit performance statistics for load tests and stress tests.
func (f *Framework) NewStatsCollector(name string) *StatsCollector {
	if f.StressTestLevel > 0 {
		name = fmt.Sprintf("stress_%v_%v", f.StressTestLevel, name)
	}
	return &StatsCollector{name: name, outputDir: f.PerfOutputDir}
}

// CleanUp Delete all Agones resources in a given namespace.
func (f *Framework) CleanUp(ns string) error {
	logrus.Info("Cleaning up now.")
	defer logrus.Info("Finished cleanup.")
	stable := f.AgonesClient.StableV1alpha1()
	deleteOptions := &metav1.DeleteOptions{}
	listOptions := metav1.ListOptions{}

	// find and delete pods created by tests and labeled with our special label
	pods := f.KubeClient.CoreV1().Pods(ns)
	podList, err := pods.List(metav1.ListOptions{
		LabelSelector: AutoCleanupLabelKey + "=" + AutoCleanupLabelValue,
	})
	if err != nil {
		return err
	}

	for _, p := range podList.Items {
		if err = pods.Delete(p.ObjectMeta.Name, deleteOptions); err != nil {
			return err
		}
	}

	err = stable.Fleets(ns).DeleteCollection(deleteOptions, listOptions)
	if err != nil {
		return err
	}

	err = f.AgonesClient.AutoscalingV1().FleetAutoscalers(ns).DeleteCollection(deleteOptions, listOptions)
	if err != nil {
		return err
	}

	return stable.GameServers(ns).
		DeleteCollection(deleteOptions, listOptions)
}

// CreateAndApplyAllocation creates and applies an Allocation to a Fleet
func (f *Framework) CreateAndApplyAllocation(t *testing.T, flt *stable.Fleet) *allocationv1alpha1.GameServerAllocation {
	gsa := GetAllocation(flt)
	gsa, err := f.AgonesClient.AllocationV1alpha1().GameServerAllocations(flt.ObjectMeta.Namespace).Create(gsa)
	if !assert.NoError(t, err) {
		assert.FailNow(t, "gameserverallocation could not be created")
	}
	assert.Equal(t, string(allocationv1alpha1.GameServerAllocationAllocated), string(gsa.Status.State))
	return gsa
}

// SendGameServerUDP sends a message to a gameserver and returns its reply
// assumes the first port is the port to send the message to
func SendGameServerUDP(gs *stable.GameServer, msg string) (string, error) {
	address := fmt.Sprintf("%s:%d", gs.Status.Address, gs.Status.Ports[0].Port)
	return SendUDP(address, msg)
}

// SendUDP sends a message to an address, and returns its reply if
// it returns one in 30 seconds
func SendUDP(address, msg string) (string, error) {
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

	err = conn.SetReadDeadline(time.Now().Add(30 * time.Second))
	if err != nil {
		return "", err
	}
	n, err := conn.Read(b)
	if err != nil {
		return "", err
	}
	return string(b[:n]), nil
}

// GetAllocation returns a GameServerAllocation that is looking for a Ready
// GameServer from this fleet.
func GetAllocation(f *stable.Fleet) *allocationv1alpha1.GameServerAllocation {
	// get an allocation
	return &allocationv1alpha1.GameServerAllocation{
		Spec: allocationv1alpha1.GameServerAllocationSpec{
			Required: metav1.LabelSelector{MatchLabels: map[string]string{stable.FleetNameLabel: f.ObjectMeta.Name}},
		}}
}
