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
	"fmt"
	"net"
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
	gs := defaultGameServer(defaultNs)
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
	gs := defaultGameServer(defaultNs)
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
		gs, err = framework.AgonesClient.AgonesV1().GameServers(defaultNs).Get(readyGs.ObjectMeta.Name, metav1.GetOptions{})
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
	gs := defaultGameServer(defaultNs)
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
	gs := defaultGameServer(defaultNs)
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

	gs := defaultGameServer(defaultNs)
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
	gs := defaultGameServer(defaultNs)
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

	_, err = framework.WaitForGameServerState(readyGs, agonesv1.GameServerStateUnhealthy, 3*time.Minute)
	assert.NoError(t, err)
}

func TestGameServerRestartBeforeReadyCrash(t *testing.T) {
	t.Parallel()
	logger := logrus.WithField("test", t.Name())

	gs := defaultGameServer(defaultNs)
	// give some buffer with gameservers crashing and coming back
	gs.Spec.Health.PeriodSeconds = 60 * 60
	gs.Spec.Template.Spec.Containers[0].Env = append(gs.Spec.Template.Spec.Containers[0].Env, corev1.EnvVar{Name: "READY", Value: "FALSE"})
	gsClient := framework.AgonesClient.AgonesV1().GameServers(defaultNs)
	newGs, err := gsClient.Create(gs)
	if !assert.NoError(t, err) {
		assert.Fail(t, "could not create the gameserver")
	}
	defer gsClient.Delete(newGs.ObjectMeta.Name, nil) // nolint: errcheck

	logger.Info("Waiting for us to have an address to send things to")
	newGs, err = framework.WaitForGameServerState(newGs, agonesv1.GameServerStateScheduled, time.Minute)
	assert.NoError(t, err)

	logger.WithField("gs", newGs.ObjectMeta.Name).Info("GameServer created")

	address := fmt.Sprintf("%s:%d", newGs.Status.Address, newGs.Status.Ports[0].Port)
	logger.WithField("address", address).Info("Dialing UDP message to address")

	messageAndWait := func(gs *agonesv1.GameServer, msg string, check func(gs *agonesv1.GameServer, pod *corev1.Pod) bool) error {
		return wait.PollImmediate(3*time.Second, 3*time.Minute, func() (bool, error) {
			gs, err := gsClient.Get(gs.ObjectMeta.Name, metav1.GetOptions{})
			if err != nil {
				logger.WithError(err).Warn("could not get gameserver")
				return true, err
			}
			pod, err := framework.KubeClient.CoreV1().Pods(defaultNs).Get(newGs.ObjectMeta.Name, metav1.GetOptions{})
			if err != nil {
				logger.WithError(err).Warn("could not get pod for gameserver")
				return true, err
			}

			if check(gs, pod) {
				return true, nil
			}

			// create a connection each time, as weird stuff happens if the receiver isn't up and running.
			conn, err := net.Dial("udp", address)
			assert.NoError(t, err)
			defer conn.Close() // nolint: errcheck
			// doing this last, so that there is a short delay between the msg being sent, and the check.
			logger.WithField("gs", gs.ObjectMeta.Name).WithField("msg", msg).Info("sending message")
			if _, err = conn.Write([]byte(msg)); err != nil {
				logger.WithError(err).WithField("gs", gs.ObjectMeta.Name).
					WithField("state", gs.Status.State).Info("error sending packet")
			}
			return false, nil
		})
	}

	logger.Info("crashing, and waiting to see restart")
	err = messageAndWait(newGs, "CRASH", func(gs *agonesv1.GameServer, pod *corev1.Pod) bool {
		for _, c := range pod.Status.ContainerStatuses {
			if c.Name == newGs.Spec.Container && c.RestartCount > 0 {
				logger.Info("successfully crashed. Moving on!")
				return true
			}
		}
		return false
	})
	assert.NoError(t, err)

	// check that the GameServer is not in an unhealthy state. If it does happen, it should happen pretty quick
	newGs, err = framework.WaitForGameServerState(newGs, agonesv1.GameServerStateUnhealthy, 5*time.Second)
	// should be an error, as the state should not occur
	if !assert.Error(t, err) {
		assert.FailNow(t, "GameServer should not be Unhealthy")
	}
	assert.Contains(t, err.Error(), "waiting for GameServer")

	// ping READY until it doesn't fail anymore - since it may take a while
	// for this to come back up -- or we could get a delayed CRASH, so we have to
	// wait for the process to restart again to fire the SDK.Ready()
	logger.Info("marking GameServer as ready")
	err = messageAndWait(newGs, "READY", func(gs *agonesv1.GameServer, pod *corev1.Pod) bool {
		if gs.Status.State == agonesv1.GameServerStateReady {
			logger.Info("ready! Moving On!")
			return true
		}
		return false
	})
	if err != nil {
		assert.Failf(t, "Could not make GameServer Ready: %v", err.Error())
	}
	// now crash, should be unhealthy, since it's after being Ready
	logger.Info("crashing again, should be unhealthy")
	// retry on crash, as with the restarts, sometimes Go takes a moment to send this through.
	err = messageAndWait(newGs, "CRASH", func(gs *agonesv1.GameServer, pod *corev1.Pod) bool {
		logger.WithField("gs", gs.ObjectMeta.Name).WithField("state", gs.Status.State).
			Info("checking final crash state")
		if gs.Status.State == agonesv1.GameServerStateUnhealthy {
			logger.Info("Unhealthy! We are done!")
			return true
		}
		return false
	})
	assert.NoError(t, err)
}

func TestGameServerUnhealthyAfterReadyCrash(t *testing.T) {
	t.Parallel()

	l := logrus.WithField("test", "TestGameServerUnhealthyAfterReadyCrash")

	gs := defaultGameServer(defaultNs)
	readyGs, err := framework.CreateGameServerAndWaitUntilReady(defaultNs, gs)
	if err != nil {
		t.Fatalf("Could not get a GameServer ready: %v", err)
	}

	l.WithField("gs", readyGs.ObjectMeta.Name).Info("GameServer created")

	gsClient := framework.AgonesClient.AgonesV1().GameServers(defaultNs)
	defer gsClient.Delete(readyGs.ObjectMeta.Name, nil) // nolint: errcheck

	address := fmt.Sprintf("%s:%d", readyGs.Status.Address, readyGs.Status.Ports[0].Port)
	conn, err := net.Dial("udp", address)
	assert.NoError(t, err)
	defer conn.Close() // nolint: errcheck
	_, err = conn.Write([]byte("CRASH"))
	assert.NoError(t, err)
	l.WithField("address", address).Info("sent UDP packet")

	_, err = framework.WaitForGameServerState(readyGs, agonesv1.GameServerStateUnhealthy, 3*time.Minute)
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
	gs := defaultGameServer(defaultNs)
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
	gs := defaultGameServer(defaultNs)
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

func TestGameServerReserve(t *testing.T) {
	t.Parallel()
	gs := defaultGameServer(defaultNs)
	readyGs, err := framework.CreateGameServerAndWaitUntilReady(defaultNs, gs)
	if err != nil {
		t.Fatalf("Could not get a GameServer ready: %v", err)
	}
	defer framework.AgonesClient.AgonesV1().GameServers(defaultNs).Delete(readyGs.ObjectMeta.Name, nil) // nolint: errcheck
	assert.Equal(t, readyGs.Status.State, agonesv1.GameServerStateReady)

	reply, err := e2eframework.SendGameServerUDP(readyGs, "RESERVE")
	if !assert.NoError(t, err) {
		assert.FailNow(t, "Could not message GameServer")
	}
	assert.Equal(t, "ACK: RESERVE\n", reply)

	gs, err = framework.WaitForGameServerState(readyGs, agonesv1.GameServerStateReserved, time.Minute)
	assert.NoError(t, err)
	assert.Equal(t, agonesv1.GameServerStateReserved, gs.Status.State)

	// it should go back after 10 seconds
	gs, err = framework.WaitForGameServerState(readyGs, agonesv1.GameServerStateReady, 15*time.Second)
	assert.NoError(t, err)
	assert.Equal(t, agonesv1.GameServerStateReady, gs.Status.State)
}

func TestGameServerShutdown(t *testing.T) {
	t.Parallel()
	gs := defaultGameServer(defaultNs)
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

	err = wait.PollImmediate(time.Second, 3*time.Minute, func() (bool, error) {
		gs, err = framework.AgonesClient.AgonesV1().GameServers(defaultNs).Get(readyGs.ObjectMeta.Name, metav1.GetOptions{})

		if k8serrors.IsNotFound(err) {
			return true, nil
		}

		return false, err
	})

	assert.NoError(t, err)
}

// TestGameServerEvicted test that if Gameserver would be evicted than it becomes Unhealthy
// Ephemeral Storage limit set to 0Mi
func TestGameServerEvicted(t *testing.T) {
	t.Parallel()
	gs := defaultGameServer(defaultNs)
	gs.Spec.Template.Spec.Containers[0].Resources.Limits[corev1.ResourceEphemeralStorage] = resource.MustParse("0Mi")
	newGs, err := framework.AgonesClient.AgonesV1().GameServers(defaultNs).Create(gs)

	assert.Nil(t, err, fmt.Sprintf("creating %v GameServer instances failed (%v): %v", gs.Spec, gs.Name, err))

	logrus.WithField("name", newGs.ObjectMeta.Name).Info("GameServer created, waiting for being Evicted and Unhealthy")

	_, err = framework.WaitForGameServerState(newGs, agonesv1.GameServerStateUnhealthy, 5*time.Minute)

	assert.Nil(t, err, fmt.Sprintf("waiting for %v GameServer Unhealthy state timed out (%v): %v",
		gs.Spec, gs.Name, err))
}

func TestGameServerPassthroughPort(t *testing.T) {
	t.Parallel()
	gs := defaultGameServer(defaultNs)
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

func defaultGameServer(namespace string) *agonesv1.GameServer {
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
