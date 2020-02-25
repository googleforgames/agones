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
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"os/user"
	"path/filepath"
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/api/resource"

	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	allocationv1 "agones.dev/agones/pkg/apis/allocation/v1"
	autoscaling "agones.dev/agones/pkg/apis/autoscaling/v1"
	"agones.dev/agones/pkg/client/clientset/versioned"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	types "k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"

	// required to use gcloud login see: https://github.com/kubernetes/client-go/issues/242
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/tools/clientcmd"
)

// special labels that can be put on pods to trigger automatic cleanup.
const (
	AutoCleanupLabelKey   = "agones.dev/e2e-test-auto-cleanup"
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

// NewFromFlags sets up the testing framework with the standard command line flags.
func NewFromFlags() (*Framework, error) {
	usr, _ := user.Current()
	kubeconfig := flag.String("kubeconfig", filepath.Join(usr.HomeDir, "/.kube/config"),
		"kube config path, e.g. $HOME/.kube/config")
	gsimage := flag.String("gameserver-image", "gcr.io/agones-images/udp-server:0.18",
		"gameserver image to use for those tests, gcr.io/agones-images/udp-server:0.18")
	pullSecret := flag.String("pullsecret", "",
		"optional secret to be used for pulling the gameserver and/or Agones SDK sidecar images")
	stressTestLevel := flag.Int("stress", 0, "enable stress test at given level 0-100")
	perfOutputDir := flag.String("perf-output", "", "write performance statistics to the specified directory")

	flag.Parse()

	framework, err := New(*kubeconfig)
	if err != nil {
		return framework, err
	}

	framework.GameServerImage = *gsimage
	framework.PullSecret = *pullSecret
	framework.StressTestLevel = *stressTestLevel
	framework.PerfOutputDir = *perfOutputDir

	return framework, nil
}

// CreateGameServerAndWaitUntilReady Creates a GameServer and wait for its state to become ready.
func (f *Framework) CreateGameServerAndWaitUntilReady(ns string, gs *agonesv1.GameServer) (*agonesv1.GameServer, error) {
	newGs, err := f.AgonesClient.AgonesV1().GameServers(ns).Create(gs)
	if err != nil {
		return nil, fmt.Errorf("creating %v GameServer instances failed (%v): %v", gs.Spec, gs.Name, err)
	}

	logrus.WithField("name", newGs.ObjectMeta.Name).Info("GameServer created, waiting for Ready")

	readyGs, err := f.WaitForGameServerState(newGs, agonesv1.GameServerStateReady, 5*time.Minute)

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
func (f *Framework) WaitForGameServerState(gs *agonesv1.GameServer, state agonesv1.GameServerState,
	timeout time.Duration) (*agonesv1.GameServer, error) {
	var readyGs *agonesv1.GameServer

	err := wait.PollImmediate(2*time.Second, timeout, func() (bool, error) {
		var err error
		readyGs, err = f.AgonesClient.AgonesV1().GameServers(gs.Namespace).Get(gs.Name, metav1.GetOptions{})

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

// AssertFleetCondition waits for the Fleet to be in a specific condition or fails the test if the condition can't be met in 5 minutes.
func (f *Framework) AssertFleetCondition(t *testing.T, flt *agonesv1.Fleet, condition func(fleet *agonesv1.Fleet) bool) {
	t.Helper()
	err := f.WaitForFleetCondition(t, flt, condition)
	if err != nil {
		// Do not call Fatalf() from go routine other than main test go routine, because it could cause a race
		t.Fatalf("error waiting for fleet condition on fleet %v", flt.Name)
	}
}

// WaitForFleetCondition waits for the Fleet to be in a specific condition or returns an error if the condition can't be met in 5 minutes.
func (f *Framework) WaitForFleetCondition(t *testing.T, flt *agonesv1.Fleet, condition func(fleet *agonesv1.Fleet) bool) error {
	t.Helper()
	logrus.WithField("fleet", flt.Name).Info("waiting for fleet condition")
	err := wait.PollImmediate(2*time.Second, 5*time.Minute, func() (bool, error) {
		fleet, err := f.AgonesClient.AgonesV1().Fleets(flt.ObjectMeta.Namespace).Get(flt.ObjectMeta.Name, metav1.GetOptions{})
		if err != nil {
			return true, err
		}

		return condition(fleet), nil
	})
	if err != nil {
		logrus.WithField("fleet", flt.Name).WithError(err).Info("error waiting for fleet condition")
		return err
	}
	return nil
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
func (f *Framework) ListGameServersFromFleet(flt *agonesv1.Fleet) ([]agonesv1.GameServer, error) {
	var results []agonesv1.GameServer

	opts := metav1.ListOptions{LabelSelector: labels.Set{agonesv1.FleetNameLabel: flt.ObjectMeta.Name}.String()}
	gsSetList, err := f.AgonesClient.AgonesV1().GameServerSets(flt.ObjectMeta.Namespace).List(opts)
	if err != nil {
		return results, err
	}

	for i := range gsSetList.Items {
		gsSet := &gsSetList.Items[i]
		opts := metav1.ListOptions{LabelSelector: labels.Set{agonesv1.GameServerSetGameServerLabel: gsSet.ObjectMeta.Name}.String()}
		gsList, err := f.AgonesClient.AgonesV1().GameServers(flt.ObjectMeta.Namespace).List(opts)
		if err != nil {
			return results, err
		}

		results = append(results, gsList.Items...)
	}

	return results, nil
}

// FleetReadyCount returns the ready count in a fleet
func FleetReadyCount(amount int32) func(fleet *agonesv1.Fleet) bool {
	return func(fleet *agonesv1.Fleet) bool {
		logrus.Infof("fleet %v has %v/%v ready replicas", fleet.Name, fleet.Status.ReadyReplicas, amount)
		return fleet.Status.ReadyReplicas == amount
	}
}

// WaitForFleetGameServersCondition waits for all GameServers for a given fleet to match
// a condition specified by a callback.
func (f *Framework) WaitForFleetGameServersCondition(flt *agonesv1.Fleet,
	cond func(server *agonesv1.GameServer) bool) error {
	return f.WaitForFleetGameServerListCondition(flt,
		func(servers []agonesv1.GameServer) bool {
			for i := range servers {
				gs := &servers[i]
				if !cond(gs) {
					return false
				}
			}
			return true
		})
}

// WaitForFleetGameServerListCondition waits for the list of GameServers to match a condition
// specified by a callback and the size of GameServers to match fleet's Spec.Replicas.
func (f *Framework) WaitForFleetGameServerListCondition(flt *agonesv1.Fleet,
	cond func(servers []agonesv1.GameServer) bool) error {
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
	agonesV1 := f.AgonesClient.AgonesV1()
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

	for i := range podList.Items {
		p := &podList.Items[i]
		if err := pods.Delete(p.ObjectMeta.Name, deleteOptions); err != nil {
			return err
		}
	}

	err = agonesV1.Fleets(ns).DeleteCollection(deleteOptions, listOptions)
	if err != nil {
		return err
	}

	err = f.AgonesClient.AutoscalingV1().FleetAutoscalers(ns).DeleteCollection(deleteOptions, listOptions)
	if err != nil {
		return err
	}

	return agonesV1.GameServers(ns).
		DeleteCollection(deleteOptions, listOptions)
}

// CreateAndApplyAllocation creates and applies an Allocation to a Fleet
func (f *Framework) CreateAndApplyAllocation(t *testing.T, flt *agonesv1.Fleet) *allocationv1.GameServerAllocation {
	gsa := GetAllocation(flt)
	gsa, err := f.AgonesClient.AllocationV1().GameServerAllocations(flt.ObjectMeta.Namespace).Create(gsa)
	if !assert.NoError(t, err) {
		assert.FailNow(t, "gameserverallocation could not be created")
	}
	assert.Equal(t, string(allocationv1.GameServerAllocationAllocated), string(gsa.Status.State))
	return gsa
}

// SendGameServerUDP sends a message to a gameserver and returns its reply
// assumes the first port is the port to send the message to,
// returns error if no Ports were allocated
func SendGameServerUDP(gs *agonesv1.GameServer, msg string) (string, error) {
	if len(gs.Status.Ports) == 0 {
		return "", errors.New("Empty Ports array")
	}
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
func GetAllocation(f *agonesv1.Fleet) *allocationv1.GameServerAllocation {
	// get an allocation
	return &allocationv1.GameServerAllocation{
		Spec: allocationv1.GameServerAllocationSpec{
			Required: metav1.LabelSelector{MatchLabels: map[string]string{agonesv1.FleetNameLabel: f.ObjectMeta.Name}},
		}}
}

// CreateNamespace creates a namespace in the test cluster
func (f *Framework) CreateNamespace(t *testing.T, namespace string) {
	t.Helper()

	kubeCore := f.KubeClient.CoreV1()
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:   namespace,
			Labels: map[string]string{"owner": "e2e-test"},
		},
	}
	if _, err := kubeCore.Namespaces().Create(ns); err != nil {
		t.Fatalf("creating namespace %s failed: %s", namespace, err)
	}
	t.Logf("Namespace %s is created", namespace)

	saName := "agones-sdk"
	if _, err := kubeCore.ServiceAccounts(namespace).Create(&corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      saName,
			Namespace: namespace,
			Labels:    map[string]string{"app": "agones"},
		},
	}); err != nil {
		t.Fatalf("creating ServiceAccount %s in namespace %s failed: %s", saName, namespace, err)
	}
	t.Logf("ServiceAccount %s/%s is created", namespace, saName)

	rb := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "agones-sdk-access",
			Namespace: namespace,
			Labels:    map[string]string{"app": "agones"},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     "agones-sdk",
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      saName,
				Namespace: namespace,
			},
		},
	}
	if _, err := f.KubeClient.RbacV1().RoleBindings(namespace).Create(rb); err != nil {
		t.Fatalf("creating RoleBinding for service account %q in namespace %q failed: %s", saName, namespace, err)
	}
	t.Logf("RoleBinding %s/%s is created", namespace, rb.Name)
}

// DeleteNamespace deletes a namespace from the test cluster
func (f *Framework) DeleteNamespace(t *testing.T, namespace string) {
	t.Helper()

	kubeCore := f.KubeClient.CoreV1()

	// Remove finalizers
	pods, err := kubeCore.Pods(namespace).List(metav1.ListOptions{})
	if err != nil {
		t.Fatalf("listing pods in namespace %s failed: %s", namespace, err)
	}
	for i := range pods.Items {
		pod := &pods.Items[i]
		if len(pod.Finalizers) > 0 {
			pod.Finalizers = nil
			payload := []patchRemoveNoValue{{
				Op:   "remove",
				Path: "/metadata/finalizers",
			}}
			payloadBytes, _ := json.Marshal(payload)
			if _, err := kubeCore.Pods(namespace).Patch(pod.Name, types.JSONPatchType, payloadBytes); err != nil {
				t.Errorf("updating pod %s failed: %s", pod.GetName(), err)
			}
		}
	}

	if err := kubeCore.Namespaces().Delete(namespace, &metav1.DeleteOptions{}); err != nil {
		t.Fatalf("deleting namespace %s failed: %s", namespace, err)
	}
	t.Logf("Namespace %s is deleted", namespace)
}

type patchRemoveNoValue struct {
	Op   string `json:"op"`
	Path string `json:"path"`
}

// DefaultGameServer provides a default GameServer fixture, based on parameters
// passed to the Test Framework.
func (f *Framework) DefaultGameServer(namespace string) *agonesv1.GameServer {
	gs := &agonesv1.GameServer{ObjectMeta: metav1.ObjectMeta{GenerateName: "udp-server", Namespace: namespace},
		Spec: agonesv1.GameServerSpec{
			Container: "udp-server",
			Ports: []agonesv1.GameServerPort{{
				ContainerPort: 7654,
				Name:          "gameport",
				PortPolicy:    agonesv1.Dynamic,
				Protocol:      corev1.ProtocolUDP,
			}},
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Name:            "udp-server",
						Image:           f.GameServerImage,
						ImagePullPolicy: corev1.PullIfNotPresent,
						Resources: corev1.ResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("30m"),
								corev1.ResourceMemory: resource.MustParse("32Mi"),
							},
							Limits: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("30m"),
								corev1.ResourceMemory: resource.MustParse("32Mi"),
							},
						},
					}},
				},
			},
		},
	}

	if f.PullSecret != "" {
		gs.Spec.Template.Spec.ImagePullSecrets = []corev1.LocalObjectReference{{
			Name: f.PullSecret}}
	}

	return gs
}
