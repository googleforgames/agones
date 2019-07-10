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

package e2e

import (
	"testing"
	"time"

	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	e2eframework "agones.dev/agones/test/e2e/framework"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

const (
	fakeIPAddress = "192.1.1.2"
)

func TestCreateConnect(t *testing.T) {
	t.Parallel()
	gs := defaultGameServer()
	readyGs, err := framework.CreateGameServerAndWaitUntilReady(defaultNs, gs)

	if err != nil {
		t.Fatalf("Could not get a GameServer ready: %v", err)
	}
	assert.Equal(t, len(readyGs.Status.Ports), 1)
	assert.NotEmpty(t, readyGs.Status.Ports[0].Port)
	assert.NotEmpty(t, readyGs.Status.Address)
	assert.NotEmpty(t, readyGs.Status.NodeName)
	assert.Equal(t, readyGs.Status.State, agonesv1.GameServerStateReady)

	reply, err := e2eframework.SendGameServerUDP(readyGs, "Hello World !")

	if err != nil {
		t.Fatalf("Could ping GameServer: %v", err)
	}

	assert.Equal(t, "ACK: Hello World !\n", reply)
}

// nolint:dupl
func TestSDKSetLabel(t *testing.T) {
	t.Parallel()
	gs := defaultGameServer()
	readyGs, err := framework.CreateGameServerAndWaitUntilReady(defaultNs, gs)
	if err != nil {
		t.Fatalf("Could not get a GameServer ready: %v", err)
	}

	assert.Equal(t, readyGs.Status.State, agonesv1.GameServerStateReady)
	reply, err := e2eframework.SendGameServerUDP(readyGs, "LABEL")

	if err != nil {
		t.Fatalf("Could ping GameServer: %v", err)
	}

	assert.Equal(t, "ACK: LABEL\n", reply)

	// the label is set in a queue, so it may take a moment
	err = wait.PollImmediate(time.Second, 10*time.Second, func() (bool, error) {
		gs, err = framework.AgonesClient.StableV1alpha1().GameServers(defaultNs).Get(readyGs.ObjectMeta.Name, metav1.GetOptions{})
		if err != nil {
			return true, err
		}
		return gs.ObjectMeta.Labels != nil, nil
	})

	assert.Nil(t, err)
	assert.NotEmpty(t, gs.ObjectMeta.Labels["agones.dev/sdk-timestamp"])
}

func TestHealthCheckDisable(t *testing.T) {
	t.Parallel()
	gs := defaultGameServer()
	gs.Spec.Health = agonesv1.Health{
		Disabled:            true,
		FailureThreshold:    1,
		InitialDelaySeconds: 1,
		PeriodSeconds:       1,
	}
	readyGs, err := framework.CreateGameServerAndWaitUntilReady(defaultNs, gs)
	if err != nil {
		t.Fatalf("Could not get a GameServer ready: %v", err)
	}
	defer framework.AgonesClient.AgonesV1().GameServers(defaultNs).Delete(readyGs.ObjectMeta.Name, nil) // nolint: errcheck

	_, err = e2eframework.SendGameServerUDP(readyGs, "UNHEALTHY")

	if err != nil {
		t.Fatalf("Could not ping GameServer: %v", err)
	}

	time.Sleep(10 * time.Second)

	gs, err = framework.AgonesClient.AgonesV1().GameServers(defaultNs).Get(readyGs.ObjectMeta.Name, metav1.GetOptions{})
	if !assert.NoError(t, err) {
		assert.FailNow(t, "gameserver get failed")
	}

	assert.Equal(t, agonesv1.GameServerStateReady, gs.Status.State)
}

// nolint:dupl
func TestSDKSetAnnotation(t *testing.T) {
	t.Parallel()
	gs := defaultGameServer()
	annotation := "agones.dev/sdk-timestamp"
	readyGs, err := framework.CreateGameServerAndWaitUntilReady(defaultNs, gs)
	if err != nil {
		t.Fatalf("Could not get a GameServer ready: %v", err)
	}
	defer framework.AgonesClient.AgonesV1().GameServers(defaultNs).Delete(readyGs.ObjectMeta.Name, nil) // nolint: errcheck

	assert.Equal(t, readyGs.Status.State, agonesv1.GameServerStateReady)
	reply, err := e2eframework.SendGameServerUDP(readyGs, "ANNOTATION")

	if err != nil {
		t.Fatalf("Could ping GameServer: %v", err)
	}

	assert.Equal(t, "ACK: ANNOTATION\n", reply)

	// the label is set in a queue, so it may take a moment
	err = wait.PollImmediate(time.Second, time.Minute, func() (bool, error) {
		gs, err = framework.AgonesClient.AgonesV1().GameServers(defaultNs).Get(readyGs.ObjectMeta.Name, metav1.GetOptions{})
		if err != nil {
			return true, err
		}

		_, ok := gs.ObjectMeta.Annotations[annotation]
		return ok, nil
	})

	logrus.WithField("annotations", gs.ObjectMeta.Annotations).Info("annotation information")

	if !assert.Nil(t, err) {
		assert.FailNow(t, "error waiting on annotation to be set")
	}
	assert.NotEmpty(t, gs.ObjectMeta.Annotations[annotation])
	assert.NotEmpty(t, gs.ObjectMeta.Annotations[agonesv1.VersionAnnotation])
}

func TestUnhealthyGameServersWithoutFreePorts(t *testing.T) {
	t.Parallel()
	nodes, err := framework.KubeClient.CoreV1().Nodes().List(metav1.ListOptions{})
	assert.Nil(t, err)

	// gate
	assert.True(t, len(nodes.Items) > 0)

	gs := defaultGameServer()
	gs.Spec.Ports[0].HostPort = 7515
	gs.Spec.Ports[0].PortPolicy = agonesv1.Static

	gameServers := framework.AgonesClient.AgonesV1().GameServers(defaultNs)

	for range nodes.Items {
		_, err := gameServers.Create(gs.DeepCopy())
		assert.Nil(t, err)
	}

	newGs, err := gameServers.Create(gs.DeepCopy())
	assert.NoError(t, err)

	_, err = framework.WaitForGameServerState(newGs, agonesv1.GameServerStateUnhealthy, time.Minute)
	assert.NoError(t, err)
}

func TestGameServerUnhealthyAfterDeletingPod(t *testing.T) {
	t.Parallel()
	gs := defaultGameServer()
	readyGs, err := framework.CreateGameServerAndWaitUntilReady(defaultNs, gs)
	if err != nil {
		t.Fatalf("Could not get a GameServer ready: %v", err)
	}

	logrus.WithField("gsKey", readyGs.ObjectMeta.Name).Info("GameServer Ready")

	gsClient := framework.AgonesClient.AgonesV1().GameServers(defaultNs)
	podClient := framework.KubeClient.CoreV1().Pods(defaultNs)

	defer gsClient.Delete(readyGs.ObjectMeta.Name, nil) // nolint: errcheck

	pod, err := podClient.Get(readyGs.ObjectMeta.Name, metav1.GetOptions{})
	assert.NoError(t, err)

	assert.True(t, metav1.IsControlledBy(pod, readyGs))

	err = podClient.Delete(pod.ObjectMeta.Name, nil)
	assert.NoError(t, err)

	_, err = framework.WaitForGameServerState(readyGs, agonesv1.GameServerStateUnhealthy, time.Minute)
	assert.NoError(t, err)
}

func TestDevelopmentGameServerLifecycle(t *testing.T) {
	t.Parallel()
	gs := &agonesv1.GameServer{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "udp-server",
			Namespace:    defaultNs,
			Annotations:  map[string]string{agonesv1.DevAddressAnnotation: fakeIPAddress},
		},
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
					Containers: []corev1.Container{{
						Name:  "placebo",
						Image: "this is ignored",
					}},
				},
			},
		},
	}
	readyGs, err := framework.CreateGameServerAndWaitUntilReady(defaultNs, gs)
	if err != nil {
		t.Fatalf("Could not get a GameServer ready: %v", err)
	}

	assert.Equal(t, readyGs.Status.State, agonesv1.GameServerStateReady)

	//confirm delete works, because if the finalisers don't get removed, this won't work.
	err = framework.AgonesClient.AgonesV1().GameServers(defaultNs).Delete(readyGs.ObjectMeta.Name, nil)
	assert.NoError(t, err)

	err = wait.PollImmediate(time.Second, 10*time.Second, func() (bool, error) {
		_, err = framework.AgonesClient.AgonesV1().GameServers(defaultNs).Get(readyGs.ObjectMeta.Name, metav1.GetOptions{})

		if k8serrors.IsNotFound(err) {
			return true, nil
		}

		return false, err
	})

	assert.NoError(t, err)
}

func TestGameServerSelfAllocate(t *testing.T) {
	t.Parallel()
	gs := defaultGameServer()
	readyGs, err := framework.CreateGameServerAndWaitUntilReady(defaultNs, gs)
	if err != nil {
		t.Fatalf("Could not get a GameServer ready: %v", err)
	}
	defer framework.AgonesClient.AgonesV1().GameServers(defaultNs).Delete(readyGs.ObjectMeta.Name, nil) // nolint: errcheck

	assert.Equal(t, readyGs.Status.State, agonesv1.GameServerStateReady)
	reply, err := e2eframework.SendGameServerUDP(readyGs, "ALLOCATE")

	if err != nil {
		t.Fatalf("Could not message GameServer: %v", err)
	}

	assert.Equal(t, "ACK: ALLOCATE\n", reply)
	gs, err = framework.WaitForGameServerState(readyGs, agonesv1.GameServerStateAllocated, time.Minute)
	assert.NoError(t, err)
	assert.Equal(t, agonesv1.GameServerStateAllocated, gs.Status.State)
}

func TestGameServerReadyAllocateReady(t *testing.T) {
	t.Parallel()
	gs := defaultGameServer()
	readyGs, err := framework.CreateGameServerAndWaitUntilReady(defaultNs, gs)
	if err != nil {
		t.Fatalf("Could not get a GameServer ready: %v", err)
	}
	defer framework.AgonesClient.AgonesV1().GameServers(defaultNs).Delete(readyGs.ObjectMeta.Name, nil) // nolint: errcheck
	assert.Equal(t, readyGs.Status.State, agonesv1.GameServerStateReady)

	reply, err := e2eframework.SendGameServerUDP(readyGs, "ALLOCATE")
	if err != nil {
		t.Fatalf("Could not message GameServer: %v", err)
	}
	assert.Equal(t, "ACK: ALLOCATE\n", reply)
	gs, err = framework.WaitForGameServerState(readyGs, agonesv1.GameServerStateAllocated, time.Minute)
	assert.NoError(t, err)
	assert.Equal(t, agonesv1.GameServerStateAllocated, gs.Status.State)

	reply, err = e2eframework.SendGameServerUDP(readyGs, "READY")
	if !assert.NoError(t, err) {
		assert.FailNow(t, "Could not message GameServer")
	}
	assert.Equal(t, "ACK: READY\n", reply)
	gs, err = framework.WaitForGameServerState(gs, agonesv1.GameServerStateReady, time.Minute)
	assert.NoError(t, err)
	assert.Equal(t, agonesv1.GameServerStateReady, gs.Status.State)
}

func TestGameServerShutdown(t *testing.T) {
	t.Parallel()
	gs := defaultGameServer()
	readyGs, err := framework.CreateGameServerAndWaitUntilReady(defaultNs, gs)
	if err != nil {
		t.Fatalf("Could not get a GameServer ready: %v", err)
	}
	assert.Equal(t, readyGs.Status.State, agonesv1.GameServerStateReady)

	reply, err := e2eframework.SendGameServerUDP(readyGs, "EXIT")
	if err != nil {
		t.Fatalf("Could not message GameServer: %v", err)
	}

	assert.Equal(t, "ACK: EXIT\n", reply)

	err = wait.PollImmediate(time.Second, time.Minute, func() (bool, error) {
		gs, err = framework.AgonesClient.AgonesV1().GameServers(defaultNs).Get(readyGs.ObjectMeta.Name, metav1.GetOptions{})

		if k8serrors.IsNotFound(err) {
			return true, nil
		}

		return false, err
	})

	assert.NoError(t, err)
}

func TestGameServerPassthroughPort(t *testing.T) {
	t.Parallel()
	gs := defaultGameServer()
	gs.Spec.Ports[0] = agonesv1.GameServerPort{PortPolicy: agonesv1.Passthrough}
	gs.Spec.Template.Spec.Containers[0].Env = []corev1.EnvVar{{Name: "PASSTHROUGH", Value: "TRUE"}}
	// gate
	_, valid := gs.Validate()
	assert.True(t, valid)

	readyGs, err := framework.CreateGameServerAndWaitUntilReady(defaultNs, gs)
	if !assert.NoError(t, err) {
		assert.FailNow(t, "Could not get a GameServer ready")
	}

	port := readyGs.Spec.Ports[0]
	assert.Equal(t, agonesv1.Passthrough, port.PortPolicy)
	assert.NotEmpty(t, port.HostPort)
	assert.Equal(t, port.HostPort, port.ContainerPort)

	reply, err := e2eframework.SendGameServerUDP(readyGs, "Hello World !")
	if err != nil {
		t.Fatalf("Could ping GameServer: %v", err)
	}

	assert.Equal(t, "ACK: Hello World !\n", reply)
}

func defaultGameServer() *agonesv1.GameServer {
	gs := &agonesv1.GameServer{ObjectMeta: metav1.ObjectMeta{GenerateName: "udp-server", Namespace: defaultNs},
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
						Image:           framework.GameServerImage,
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

	if framework.PullSecret != "" {
		gs.Spec.Template.Spec.ImagePullSecrets = []corev1.LocalObjectReference{{
			Name: framework.PullSecret}}
	}

	return gs
}
