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

	"agones.dev/agones/pkg/apis/stable"
	"agones.dev/agones/pkg/apis/stable/v1alpha1"
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
	assert.Equal(t, readyGs.Status.State, v1alpha1.GameServerStateReady)

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

	assert.Equal(t, readyGs.Status.State, v1alpha1.GameServerStateReady)
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
	assert.NotEmpty(t, gs.ObjectMeta.Labels["stable.agones.dev/sdk-timestamp"])
}

func TestHealthCheckDisable(t *testing.T) {
	t.Parallel()
	gs := defaultGameServer()
	gs.Spec.Health = v1alpha1.Health{
		Disabled:            true,
		FailureThreshold:    1,
		InitialDelaySeconds: 1,
		PeriodSeconds:       1,
	}
	readyGs, err := framework.CreateGameServerAndWaitUntilReady(defaultNs, gs)
	if err != nil {
		t.Fatalf("Could not get a GameServer ready: %v", err)
	}
	defer framework.AgonesClient.StableV1alpha1().GameServers(defaultNs).Delete(readyGs.ObjectMeta.Name, nil) // nolint: errcheck

	_, err = e2eframework.SendGameServerUDP(readyGs, "UNHEALTHY")

	if err != nil {
		t.Fatalf("Could not ping GameServer: %v", err)
	}

	time.Sleep(10 * time.Second)

	gs, err = framework.AgonesClient.StableV1alpha1().GameServers(defaultNs).Get(readyGs.ObjectMeta.Name, metav1.GetOptions{})
	if !assert.NoError(t, err) {
		assert.FailNow(t, "gameserver get failed")
	}

	assert.Equal(t, v1alpha1.GameServerStateReady, gs.Status.State)
}

// nolint:dupl
func TestSDKSetAnnotation(t *testing.T) {
	t.Parallel()
	gs := defaultGameServer()
	annotation := "stable.agones.dev/sdk-timestamp"
	readyGs, err := framework.CreateGameServerAndWaitUntilReady(defaultNs, gs)
	if err != nil {
		t.Fatalf("Could not get a GameServer ready: %v", err)
	}
	defer framework.AgonesClient.StableV1alpha1().GameServers(defaultNs).Delete(readyGs.ObjectMeta.Name, nil) // nolint: errcheck

	assert.Equal(t, readyGs.Status.State, v1alpha1.GameServerStateReady)
	reply, err := e2eframework.SendGameServerUDP(readyGs, "ANNOTATION")

	if err != nil {
		t.Fatalf("Could ping GameServer: %v", err)
	}

	assert.Equal(t, "ACK: ANNOTATION\n", reply)

	// the label is set in a queue, so it may take a moment
	err = wait.PollImmediate(time.Second, 10*time.Second, func() (bool, error) {
		gs, err = framework.AgonesClient.StableV1alpha1().GameServers(defaultNs).Get(readyGs.ObjectMeta.Name, metav1.GetOptions{})
		if err != nil {
			return true, err
		}

		_, ok := gs.ObjectMeta.Annotations[annotation]
		return ok, nil
	})

	assert.Nil(t, err)
	assert.NotEmpty(t, gs.ObjectMeta.Annotations[annotation])
	assert.NotEmpty(t, gs.ObjectMeta.Annotations[stable.VersionAnnotation])
}

func TestUnhealthyGameServersWithoutFreePorts(t *testing.T) {
	t.Parallel()
	nodes, err := framework.KubeClient.CoreV1().Nodes().List(metav1.ListOptions{})
	assert.Nil(t, err)

	// gate
	assert.True(t, len(nodes.Items) > 0)

	gs := defaultGameServer()
	gs.Spec.Ports[0].HostPort = 7515
	gs.Spec.Ports[0].PortPolicy = v1alpha1.Static

	gameServers := framework.AgonesClient.StableV1alpha1().GameServers(defaultNs)

	for range nodes.Items {
		_, err := gameServers.Create(gs.DeepCopy())
		assert.Nil(t, err)
	}

	newGs, err := gameServers.Create(gs.DeepCopy())
	assert.Nil(t, err)

	_, err = framework.WaitForGameServerState(newGs, v1alpha1.GameServerStateUnhealthy, 10*time.Second)
	assert.NotNil(t, err)

	_, err = gameServers.Get(newGs.Name, metav1.GetOptions{})
	assert.NotNil(t, err)
	assert.True(t, k8serrors.IsNotFound(err))
}

func TestGameServerUnhealthyAfterDeletingPod(t *testing.T) {
	t.Parallel()
	gs := defaultGameServer()
	readyGs, err := framework.CreateGameServerAndWaitUntilReady(defaultNs, gs)
	if err != nil {
		t.Fatalf("Could not get a GameServer ready: %v", err)
	}

	logrus.WithField("gsKey", readyGs.ObjectMeta.Name).Info("GameServer Ready")

	gsClient := framework.AgonesClient.StableV1alpha1().GameServers(defaultNs)
	podClient := framework.KubeClient.CoreV1().Pods(defaultNs)

	defer gsClient.Delete(readyGs.ObjectMeta.Name, nil) // nolint: errcheck

	pod, err := podClient.Get(readyGs.ObjectMeta.Name, metav1.GetOptions{})
	assert.NoError(t, err)

	assert.True(t, metav1.IsControlledBy(pod, readyGs))

	err = podClient.Delete(pod.ObjectMeta.Name, nil)
	assert.NoError(t, err)

	// TODO [markmandel@google.com]: Should GameServers that are Unhealthy and not in a fleet be deleted?
	err = wait.PollImmediate(2*time.Second, time.Minute, func() (bool, error) {
		gs, err := framework.AgonesClient.StableV1alpha1().GameServers(readyGs.Namespace).Get(readyGs.ObjectMeta.Name, metav1.GetOptions{})

		// just in case
		if k8serrors.IsNotFound(err) {
			return true, nil
		}

		if err != nil {
			logrus.WithError(err).Warn("error retrieving gameserver")
			return false, nil
		}

		if gs.Status.State == v1alpha1.GameServerStateUnhealthy {
			return true, nil
		}

		return false, nil
	})

	assert.NoError(t, err)
}

func TestDevelopmentGameServerLifecycle(t *testing.T) {
	t.Parallel()
	gs := &v1alpha1.GameServer{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "udp-server",
			Namespace:    defaultNs,
			Annotations:  map[string]string{v1alpha1.DevAddressAnnotation: fakeIPAddress},
		},
		Spec: v1alpha1.GameServerSpec{
			Container: "udp-server",
			Ports: []v1alpha1.GameServerPort{{
				ContainerPort: 7654,
				HostPort:      7654,
				Name:          "gameport",
				PortPolicy:    v1alpha1.Static,
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

	assert.Equal(t, readyGs.Status.State, v1alpha1.GameServerStateReady)

	//confirm delete works, because if the finalisers don't get removed, this won't work.
	err = framework.AgonesClient.StableV1alpha1().GameServers(defaultNs).Delete(readyGs.ObjectMeta.Name, nil)
	assert.NoError(t, err)

	err = wait.PollImmediate(time.Second, 10*time.Second, func() (bool, error) {
		_, err = framework.AgonesClient.StableV1alpha1().GameServers(defaultNs).Get(readyGs.ObjectMeta.Name, metav1.GetOptions{})

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
	defer framework.AgonesClient.StableV1alpha1().GameServers(defaultNs).Delete(readyGs.ObjectMeta.Name, nil) // nolint: errcheck

	assert.Equal(t, readyGs.Status.State, v1alpha1.GameServerStateReady)
	reply, err := e2eframework.SendGameServerUDP(readyGs, "ALLOCATE")

	if err != nil {
		t.Fatalf("Could not message GameServer: %v", err)
	}

	assert.Equal(t, "ACK: ALLOCATE\n", reply)

	// the label is set in a queue, so it may take a moment
	err = wait.PollImmediate(time.Second, 10*time.Second, func() (bool, error) {
		gs, err = framework.AgonesClient.StableV1alpha1().GameServers(defaultNs).Get(readyGs.ObjectMeta.Name, metav1.GetOptions{})
		if err != nil {
			return true, err
		}

		return gs.Status.State == v1alpha1.GameServerStateAllocated, nil
	})

	assert.NoError(t, err)
}

func defaultGameServer() *v1alpha1.GameServer {
	gs := &v1alpha1.GameServer{ObjectMeta: metav1.ObjectMeta{GenerateName: "udp-server", Namespace: defaultNs},
		Spec: v1alpha1.GameServerSpec{
			Container: "udp-server",
			Ports: []v1alpha1.GameServerPort{{
				ContainerPort: 7654,
				Name:          "gameport",
				PortPolicy:    v1alpha1.Dynamic,
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
