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
	"bytes"
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	allocationv1 "agones.dev/agones/pkg/apis/allocation/v1"
	"agones.dev/agones/pkg/util/runtime"
	e2eframework "agones.dev/agones/test/e2e/framework"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/apimachinery/pkg/util/wait"
)

const (
	fakeIPAddress = "192.1.1.2"
)

func TestCreateConnect(t *testing.T) {
	t.Parallel()
	gs := framework.DefaultGameServer(framework.Namespace)
	readyGs, err := framework.CreateGameServerAndWaitUntilReady(t, framework.Namespace, gs)

	if err != nil {
		t.Fatalf("Could not get a GameServer ready: %v", err)
	}
	assert.Equal(t, len(readyGs.Status.Ports), 1)
	assert.NotEmpty(t, readyGs.Status.Ports[0].Port)
	assert.NotEmpty(t, readyGs.Status.Address)
	assert.NotEmpty(t, readyGs.Status.NodeName)
	assert.Equal(t, readyGs.Status.State, agonesv1.GameServerStateReady)

	reply, err := framework.SendGameServerUDP(t, readyGs, "Hello World !")

	if err != nil {
		t.Fatalf("Could ping GameServer: %v", err)
	}

	assert.Equal(t, "ACK: Hello World !\n", reply)
}

// nolint:dupl
func TestSDKSetLabel(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	gs := framework.DefaultGameServer(framework.Namespace)
	readyGs, err := framework.CreateGameServerAndWaitUntilReady(t, framework.Namespace, gs)
	if err != nil {
		t.Fatalf("Could not get a GameServer ready: %v", err)
	}

	assert.Equal(t, readyGs.Status.State, agonesv1.GameServerStateReady)
	reply, err := framework.SendGameServerUDP(t, readyGs, "LABEL")

	if err != nil {
		t.Fatalf("Could ping GameServer: %v", err)
	}

	assert.Equal(t, "ACK: LABEL\n", reply)

	// the label is set in a queue, so it may take a moment
	err = wait.PollImmediate(time.Second, 10*time.Second, func() (bool, error) {
		gs, err = framework.AgonesClient.AgonesV1().GameServers(framework.Namespace).Get(ctx, readyGs.ObjectMeta.Name, metav1.GetOptions{})
		if err != nil {
			return true, err
		}
		return gs.ObjectMeta.Labels != nil, nil
	})

	if assert.NoError(t, err) {
		assert.NotEmpty(t, gs.ObjectMeta.Labels["agones.dev/sdk-timestamp"])
	}
}

func TestHealthCheckDisable(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	gs := framework.DefaultGameServer(framework.Namespace)
	gs.Spec.Health = agonesv1.Health{
		Disabled:            true,
		FailureThreshold:    1,
		InitialDelaySeconds: 1,
		PeriodSeconds:       1,
	}
	readyGs, err := framework.CreateGameServerAndWaitUntilReady(t, framework.Namespace, gs)
	if err != nil {
		t.Fatalf("Could not get a GameServer ready: %v", err)
	}
	defer framework.AgonesClient.AgonesV1().GameServers(framework.Namespace).Delete(ctx, readyGs.ObjectMeta.Name, metav1.DeleteOptions{}) // nolint: errcheck

	_, err = framework.SendGameServerUDP(t, readyGs, "UNHEALTHY")

	if err != nil {
		t.Fatalf("Could not ping GameServer: %v", err)
	}

	time.Sleep(10 * time.Second)

	gs, err = framework.AgonesClient.AgonesV1().GameServers(framework.Namespace).Get(ctx, readyGs.ObjectMeta.Name, metav1.GetOptions{})
	if err != nil {
		assert.FailNow(t, "gameserver get failed", err.Error())
	}

	assert.Equal(t, agonesv1.GameServerStateReady, gs.Status.State)
}

// nolint:dupl
func TestSDKSetAnnotation(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	gs := framework.DefaultGameServer(framework.Namespace)
	annotation := "agones.dev/sdk-timestamp"
	readyGs, err := framework.CreateGameServerAndWaitUntilReady(t, framework.Namespace, gs)
	if err != nil {
		t.Fatalf("Could not get a GameServer ready: %v", err)
	}
	defer framework.AgonesClient.AgonesV1().GameServers(framework.Namespace).Delete(ctx, readyGs.ObjectMeta.Name, metav1.DeleteOptions{}) // nolint: errcheck

	assert.Equal(t, readyGs.Status.State, agonesv1.GameServerStateReady)
	reply, err := framework.SendGameServerUDP(t, readyGs, "ANNOTATION")

	if err != nil {
		t.Fatalf("Could ping GameServer: %v", err)
	}

	assert.Equal(t, "ACK: ANNOTATION\n", reply)

	// the label is set in a queue, so it may take a moment
	err = wait.PollImmediate(time.Second, time.Minute, func() (bool, error) {
		gs, err = framework.AgonesClient.AgonesV1().GameServers(framework.Namespace).Get(ctx, readyGs.ObjectMeta.Name, metav1.GetOptions{})
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

func TestUnhealthyGameServerAfterHealthCheckFail(t *testing.T) {
	t.Parallel()
	gs := framework.DefaultGameServer(framework.Namespace)
	gs.Spec.Health.FailureThreshold = 1

	gs, err := framework.CreateGameServerAndWaitUntilReady(t, framework.Namespace, gs)
	if err != nil {
		assert.FailNow(t, "Failed to create a gameserver", err.Error())
	}

	reply, err := framework.SendGameServerUDP(t, gs, "UNHEALTHY")
	if err != nil {
		assert.FailNow(t, "Failed to send a message to a gameserver", err.Error())
	}
	assert.Equal(t, "ACK: UNHEALTHY\n", reply)

	_, err = framework.WaitForGameServerState(t, gs, agonesv1.GameServerStateUnhealthy, time.Minute)
	assert.NoError(t, err)
}

func TestUnhealthyGameServersWithoutFreePorts(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	nodes, err := framework.KubeClient.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		assert.FailNow(t, "Failed to list nodes", err.Error())
	}
	assert.True(t, len(nodes.Items) > 0)

	template := framework.DefaultGameServer(framework.Namespace)
	// choose port out of the minport/maxport range
	// also rand it, just in case there are still previous static GameServers floating around.
	template.Spec.Ports[0].HostPort = int32(rand.IntnRange(4000, 5000))
	template.Spec.Ports[0].PortPolicy = agonesv1.Static

	gameServers := framework.AgonesClient.AgonesV1().GameServers(framework.Namespace)
	// one successful static port GameServer
	gs, err := framework.CreateGameServerAndWaitUntilReady(t, framework.Namespace, template.DeepCopy())
	require.NoError(t, err)

	// now let's create the same one, but this time, require it be on the same node.
	newGs := template.DeepCopy()

	if newGs.Spec.Template.Spec.NodeSelector == nil {
		newGs.Spec.Template.Spec.NodeSelector = map[string]string{}
	}
	newGs.Spec.Template.Spec.NodeSelector["kubernetes.io/hostname"] = gs.Status.NodeName
	newGs, err = gameServers.Create(ctx, newGs, metav1.CreateOptions{})
	require.NoError(t, err)

	_, err = framework.WaitForGameServerState(t, newGs, agonesv1.GameServerStateUnhealthy, 5*time.Minute)
	assert.NoError(t, err)
}

func TestGameServerUnhealthyAfterDeletingPod(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	gs := framework.DefaultGameServer(framework.Namespace)
	readyGs, err := framework.CreateGameServerAndWaitUntilReady(t, framework.Namespace, gs)
	if err != nil {
		t.Fatalf("Could not get a GameServer ready: %v", err)
	}

	logrus.WithField("gsKey", readyGs.ObjectMeta.Name).Info("GameServer Ready")

	gsClient := framework.AgonesClient.AgonesV1().GameServers(framework.Namespace)
	podClient := framework.KubeClient.CoreV1().Pods(framework.Namespace)

	defer gsClient.Delete(ctx, readyGs.ObjectMeta.Name, metav1.DeleteOptions{}) // nolint: errcheck

	pod, err := podClient.Get(ctx, readyGs.ObjectMeta.Name, metav1.GetOptions{})
	if err != nil {
		assert.FailNow(t, "Failed to get a pod", err.Error())
	}

	assert.True(t, metav1.IsControlledBy(pod, readyGs))

	err = podClient.Delete(ctx, pod.ObjectMeta.Name, metav1.DeleteOptions{})
	assert.NoError(t, err)

	_, err = framework.WaitForGameServerState(t, readyGs, agonesv1.GameServerStateUnhealthy, 3*time.Minute)
	assert.NoError(t, err)
}

func TestGameServerRestartBeforeReadyCrash(t *testing.T) {
	// TODO(#2445): The feature is flaky when pod updates are slow, run serially to avoid.
	// t.Parallel()
	ctx := context.Background()
	logger := e2eframework.TestLogger(t)

	gs := framework.DefaultGameServer(framework.Namespace)
	// give some buffer with gameservers crashing and coming back
	gs.Spec.Health.PeriodSeconds = 60 * 60
	gs.Spec.Template.Spec.Containers[0].Env = append(gs.Spec.Template.Spec.Containers[0].Env, corev1.EnvVar{Name: "READY", Value: "FALSE"})
	gsClient := framework.AgonesClient.AgonesV1().GameServers(framework.Namespace)
	newGs, err := gsClient.Create(ctx, gs, metav1.CreateOptions{})
	if err != nil {
		assert.Fail(t, "could not create the gameserver", err.Error())
	}
	defer gsClient.Delete(ctx, newGs.ObjectMeta.Name, metav1.DeleteOptions{}) // nolint: errcheck

	logger.Info("Waiting for us to have an address to send things to")
	newGs, err = framework.WaitForGameServerState(t, newGs, agonesv1.GameServerStateScheduled, time.Minute)
	if err != nil {
		assert.FailNow(t, "Failed schedule a pod", err.Error())
	}

	logger.WithField("gs", newGs.ObjectMeta.Name).Info("GameServer created")

	address := fmt.Sprintf("%s:%d", newGs.Status.Address, newGs.Status.Ports[0].Port)
	logger.WithField("address", address).Info("Dialing UDP message to address")

	messageAndWait := func(gs *agonesv1.GameServer, msg string, check func(gs *agonesv1.GameServer, pod *corev1.Pod) bool) error {
		return wait.PollImmediate(200*time.Millisecond, 3*time.Minute, func() (bool, error) {
			gs, err := gsClient.Get(ctx, gs.ObjectMeta.Name, metav1.GetOptions{})
			if err != nil {
				logger.WithError(err).Warn("could not get gameserver")
				return true, err
			}
			pod, err := framework.KubeClient.CoreV1().Pods(framework.Namespace).Get(ctx, newGs.ObjectMeta.Name, metav1.GetOptions{})
			if err != nil {
				logger.WithError(err).Warn("could not get pod for gameserver")
				return true, err
			}

			if check(gs, pod) {
				return true, nil
			}

			// create a connection each time, as weird stuff happens if the receiver isn't up and running.
			conn, err := net.Dial("udp", address)
			if err != nil {
				logger.WithError(err).Warn("could not create connection")
				return true, err
			}
			defer conn.Close() // nolint: errcheck
			// doing this last, so that there is a short delay between the msg being sent, and the check.
			logger.WithField("gs", gs.ObjectMeta.Name).WithField("msg", msg).
				WithField("state", gs.Status.State).Info("sending message")
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
	newGs, err = framework.WaitForGameServerState(t, newGs, agonesv1.GameServerStateUnhealthy, 5*time.Second)
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
	assert.NoError(t, err)

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
	ctx := context.Background()

	l := logrus.WithField("test", "TestGameServerUnhealthyAfterReadyCrash")

	gs := framework.DefaultGameServer(framework.Namespace)
	readyGs, err := framework.CreateGameServerAndWaitUntilReady(t, framework.Namespace, gs)
	if err != nil {
		t.Fatalf("Could not get a GameServer ready: %v", err)
	}

	l.WithField("gs", readyGs.ObjectMeta.Name).Info("GameServer created")

	gsClient := framework.AgonesClient.AgonesV1().GameServers(framework.Namespace)
	defer gsClient.Delete(ctx, readyGs.ObjectMeta.Name, metav1.DeleteOptions{}) // nolint: errcheck

	address := fmt.Sprintf("%s:%d", readyGs.Status.Address, readyGs.Status.Ports[0].Port)

	// keep crashing, until we move to Unhealthy. Solves potential issues with controller Informer cache
	// race conditions in which it has yet to see a GameServer is Ready before the crash.
	var stop int32
	defer func() {
		atomic.StoreInt32(&stop, 1)
	}()
	go func() {
		for {
			if atomic.LoadInt32(&stop) > 0 {
				l.Info("UDP Crash stop signal received. Stopping.")
				return
			}
			var writeErr error
			func() {
				conn, err := net.Dial("udp", address)
				assert.NoError(t, err)
				defer conn.Close() // nolint: errcheck
				_, writeErr = conn.Write([]byte("CRASH"))
			}()
			if writeErr != nil {
				l.WithError(err).Warn("error sending udp packet. Stopping.")
				return
			}
			l.WithField("address", address).Info("sent UDP packet")
			time.Sleep(5 * time.Second)
		}
	}()
	_, err = framework.WaitForGameServerState(t, readyGs, agonesv1.GameServerStateUnhealthy, 3*time.Minute)
	assert.NoError(t, err)
}

func TestDevelopmentGameServerLifecycle(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	labels := map[string]string{"development": "true"}
	gs := &agonesv1.GameServer{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "udp-server",
			Namespace:    framework.Namespace,
			Annotations:  map[string]string{agonesv1.DevAddressAnnotation: fakeIPAddress},
			Labels:       labels,
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
	readyGs, err := framework.CreateGameServerAndWaitUntilReady(t, framework.Namespace, gs.DeepCopy())
	if err != nil {
		t.Fatalf("Could not get a GameServer ready: %v", err)
	}
	require.Equal(t, readyGs.Status.State, agonesv1.GameServerStateReady)

	// confirm delete works, because if the finalisers don't get removed, this won't work.
	err = framework.AgonesClient.AgonesV1().GameServers(framework.Namespace).Delete(ctx, readyGs.ObjectMeta.Name, metav1.DeleteOptions{})
	require.NoError(t, err)

	err = wait.PollImmediate(time.Second, time.Minute, func() (bool, error) {
		_, err = framework.AgonesClient.AgonesV1().GameServers(framework.Namespace).Get(ctx, readyGs.ObjectMeta.Name, metav1.GetOptions{})
		if k8serrors.IsNotFound(err) {
			return true, nil
		}

		return false, err
	})
	require.NoError(t, err)

	// let's make sure we can allocate a dev gameserver
	readyGs, err = framework.CreateGameServerAndWaitUntilReady(t, framework.Namespace, gs.DeepCopy())
	require.NoError(t, err)

	gsa := &allocationv1.GameServerAllocation{
		Spec: allocationv1.GameServerAllocationSpec{
			Selectors: []allocationv1.GameServerSelector{{
				LabelSelector: metav1.LabelSelector{MatchLabels: labels},
			}},
		},
	}
	gsa, err = framework.AgonesClient.AllocationV1().GameServerAllocations(framework.Namespace).Create(ctx, gsa, metav1.CreateOptions{})
	require.NoError(t, err)

	require.Equal(t, readyGs.ObjectMeta.Name, gsa.Status.GameServerName)

	_, err = framework.WaitForGameServerState(t, readyGs, agonesv1.GameServerStateAllocated, time.Minute)
	require.NoError(t, err)

	// Also confirm that delete works for Allocated state, it should be fine but let's be sure.
	err = framework.AgonesClient.AgonesV1().GameServers(framework.Namespace).Delete(ctx, readyGs.ObjectMeta.Name, metav1.DeleteOptions{})
	require.NoError(t, err)

	err = wait.PollImmediate(time.Second, time.Minute, func() (bool, error) {
		_, err = framework.AgonesClient.AgonesV1().GameServers(framework.Namespace).Get(ctx, readyGs.ObjectMeta.Name, metav1.GetOptions{})
		if k8serrors.IsNotFound(err) {
			return true, nil
		}

		return false, err
	})
	require.NoError(t, err)
}

func TestGameServerSelfAllocate(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	gs := framework.DefaultGameServer(framework.Namespace)
	readyGs, err := framework.CreateGameServerAndWaitUntilReady(t, framework.Namespace, gs)
	if err != nil {
		t.Fatalf("Could not get a GameServer ready: %v", err)
	}
	defer framework.AgonesClient.AgonesV1().GameServers(framework.Namespace).Delete(ctx, readyGs.ObjectMeta.Name, metav1.DeleteOptions{}) // nolint: errcheck

	assert.Equal(t, readyGs.Status.State, agonesv1.GameServerStateReady)
	reply, err := framework.SendGameServerUDP(t, readyGs, "ALLOCATE")
	if err != nil {
		t.Fatalf("Could not message GameServer: %v", err)
	}

	assert.Equal(t, "ACK: ALLOCATE\n", reply)
	gs, err = framework.WaitForGameServerState(t, readyGs, agonesv1.GameServerStateAllocated, time.Minute)
	if assert.NoError(t, err) {
		assert.Equal(t, agonesv1.GameServerStateAllocated, gs.Status.State)
	}
}

func TestGameServerReadyAllocateReady(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	logger := e2eframework.TestLogger(t)

	gs := framework.DefaultGameServer(framework.Namespace)

	logger.Info("Moving to Ready")
	readyGs, err := framework.CreateGameServerAndWaitUntilReady(t, framework.Namespace, gs)
	require.NoError(t, err, "Could not get a GameServer ready")
	logger = logger.WithField("gs", readyGs.ObjectMeta.Name)

	defer framework.AgonesClient.AgonesV1().GameServers(framework.Namespace).Delete(ctx, readyGs.ObjectMeta.Name, metav1.DeleteOptions{}) // nolint: errcheck

	require.Equal(t, readyGs.Status.State, agonesv1.GameServerStateReady)

	logger.Info("Moving to Allocated")
	reply, err := framework.SendGameServerUDP(t, readyGs, "ALLOCATE")
	require.NoError(t, err, "Could not message GameServer")

	require.Equal(t, "ACK: ALLOCATE\n", reply)
	gs, err = framework.WaitForGameServerState(t, readyGs, agonesv1.GameServerStateAllocated, time.Minute)
	require.NoError(t, err)
	require.Equal(t, agonesv1.GameServerStateAllocated, gs.Status.State)

	logger.Info("Moving to Ready again")
	reply, err = framework.SendGameServerUDP(t, readyGs, "READY")
	require.NoError(t, err, "Could not message GameServer")
	require.Equal(t, "ACK: READY\n", reply)
	gs, err = framework.WaitForGameServerState(t, gs, agonesv1.GameServerStateReady, time.Minute)
	require.NoError(t, err)
	require.Equal(t, agonesv1.GameServerStateReady, gs.Status.State)
}

func TestGameServerWithPortsMappedToMultipleContainers(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	firstContainerName := "udp-server"
	secondContainerName := "second-udp-server"
	firstPort := "gameport"
	secondPort := "second-gameport"
	gs := &agonesv1.GameServer{ObjectMeta: metav1.ObjectMeta{GenerateName: "udp-server", Namespace: framework.Namespace},
		Spec: agonesv1.GameServerSpec{
			Container: firstContainerName,
			Ports: []agonesv1.GameServerPort{{
				ContainerPort: 7654,
				Name:          firstPort,
				PortPolicy:    agonesv1.Dynamic,
				Protocol:      corev1.ProtocolUDP,
			}, {
				ContainerPort: 5000,
				Name:          secondPort,
				PortPolicy:    agonesv1.Dynamic,
				Protocol:      corev1.ProtocolUDP,
				Container:     &secondContainerName,
			}},
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:            firstContainerName,
							Image:           framework.GameServerImage,
							ImagePullPolicy: corev1.PullIfNotPresent,
						},
						{
							Name:            secondContainerName,
							Image:           framework.GameServerImage,
							ImagePullPolicy: corev1.PullIfNotPresent,
							Args:            []string{"-port", "5000"},
						},
					},
				},
			},
		},
	}

	readyGs, err := framework.CreateGameServerAndWaitUntilReady(t, framework.Namespace, gs)
	if err != nil {
		t.Fatalf("Could not get a GameServer ready: %v", err)
	}
	defer framework.AgonesClient.AgonesV1().GameServers(framework.Namespace).Delete(ctx, readyGs.ObjectMeta.Name, metav1.DeleteOptions{}) // nolint: errcheck
	assert.Equal(t, readyGs.Status.State, agonesv1.GameServerStateReady)

	interval := 2 * time.Second
	timeOut := 60 * time.Second

	expectedMsg1 := "Ping 1"
	errPoll := wait.PollImmediate(interval, timeOut, func() (done bool, err error) {
		res, err := framework.SendGameServerUDPToPort(t, readyGs, firstPort, expectedMsg1)
		if err != nil {
			t.Logf("Could not message GameServer on %s: %v. Will try again...", firstPort, err)
		}
		return err == nil && strings.Contains(res, expectedMsg1), nil
	})
	if errPoll != nil {
		assert.FailNow(t, errPoll.Error(), "expected no errors after polling a port: %s", firstPort)
	}

	expectedMsg2 := "Ping 2"
	errPoll = wait.PollImmediate(interval, timeOut, func() (done bool, err error) {
		res, err := framework.SendGameServerUDPToPort(t, readyGs, secondPort, expectedMsg2)
		if err != nil {
			t.Logf("Could not message GameServer on %s: %v. Will try again...", secondPort, err)
		}
		return err == nil && strings.Contains(res, expectedMsg2), nil
	})

	assert.NoError(t, errPoll, "expected no errors after polling a port: %s", secondPort)
}

func TestGameServerReserve(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	// We are deliberately not trying to test the transition between Reserved -> Ready.
	//
	// We have found that trying to catch the GameServer in the Reserved state can be flaky,
	// as we can't control the speed in which the Kubernetes API is going to reply to request,
	// and we could sometimes miss when the GameServer is in the Reserved State before it goes to Ready.
	//
	// Therefore we are going to test for concrete states that we don't need to catch while
	// in a transitive state.

	gs := framework.DefaultGameServer(framework.Namespace)
	gs, err := framework.CreateGameServerAndWaitUntilReady(t, framework.Namespace, gs)
	if err != nil {
		assert.FailNow(t, "Could not get a GameServer ready", err.Error())
	}
	defer framework.AgonesClient.AgonesV1().GameServers(framework.Namespace).Delete(ctx, gs.ObjectMeta.Name, metav1.DeleteOptions{}) // nolint: errcheck
	assert.Equal(t, gs.Status.State, agonesv1.GameServerStateReady)

	reply, err := framework.SendGameServerUDP(t, gs, "RESERVE 0")
	if err != nil {
		assert.FailNow(t, "Could not message GameServer", err.Error())
	}
	assert.Equal(t, "ACK: RESERVE 0\n", reply)

	gs, err = framework.WaitForGameServerState(t, gs, agonesv1.GameServerStateReserved, 3*time.Minute)
	if err != nil {
		assert.FailNow(t, "Time out on waiting for gs in a Reserved state", err.Error())
	}

	reply, err = framework.SendGameServerUDP(t, gs, "ALLOCATE")
	if err != nil {
		assert.FailNow(t, "Could not message GameServer", err.Error())
	}
	assert.Equal(t, "ACK: ALLOCATE\n", reply)

	// put it in a totally different state, just to reset things.
	gs, err = framework.WaitForGameServerState(t, gs, agonesv1.GameServerStateAllocated, 3*time.Minute)
	if err != nil {
		assert.FailNow(t, "Time out on waiting for gs in an Allocated state", err.Error())
	}

	reply, err = framework.SendGameServerUDP(t, gs, "RESERVE 5s")
	if err != nil {
		assert.FailNow(t, "Could not message GameServer", err.Error())
	}
	assert.Equal(t, "ACK: RESERVE 5s\n", reply)

	// sleep, since we're going to wait for the Ready response.
	time.Sleep(5 * time.Second)
	_, err = framework.WaitForGameServerState(t, gs, agonesv1.GameServerStateReady, 3*time.Minute)
	assert.NoError(t, err)
}

func TestGameServerShutdown(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	gs := framework.DefaultGameServer(framework.Namespace)
	readyGs, err := framework.CreateGameServerAndWaitUntilReady(t, framework.Namespace, gs)
	if err != nil {
		t.Fatalf("Could not get a GameServer ready: %v", err)
	}
	assert.Equal(t, readyGs.Status.State, agonesv1.GameServerStateReady)

	reply, err := framework.SendGameServerUDP(t, readyGs, "EXIT")
	if err != nil {
		t.Fatalf("Could not message GameServer: %v", err)
	}

	assert.Equal(t, "ACK: EXIT\n", reply)

	err = wait.PollImmediate(time.Second, 3*time.Minute, func() (bool, error) {
		gs, err = framework.AgonesClient.AgonesV1().GameServers(framework.Namespace).Get(ctx, readyGs.ObjectMeta.Name, metav1.GetOptions{})

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
	ctx := context.Background()
	gs := framework.DefaultGameServer(framework.Namespace)
	gs.Spec.Template.Spec.Containers[0].Resources.Limits[corev1.ResourceEphemeralStorage] = resource.MustParse("0Mi")

	newGs, err := framework.AgonesClient.AgonesV1().GameServers(framework.Namespace).Create(ctx, gs, metav1.CreateOptions{})
	if err != nil {
		assert.FailNow(t, fmt.Sprintf("creating %v GameServer instances failed (%v): %v", gs.Spec, gs.Name, err))
	}

	logrus.WithField("name", newGs.ObjectMeta.Name).Info("GameServer created, waiting for being Evicted and Unhealthy")

	_, err = framework.WaitForGameServerState(t, newGs, agonesv1.GameServerStateUnhealthy, 5*time.Minute)

	assert.Nil(t, err, fmt.Sprintf("waiting for %v GameServer Unhealthy state timed out (%v): %v", gs.Spec, gs.Name, err))
}

func TestGameServerPassthroughPort(t *testing.T) {
	t.Parallel()
	gs := framework.DefaultGameServer(framework.Namespace)
	gs.Spec.Ports[0] = agonesv1.GameServerPort{PortPolicy: agonesv1.Passthrough}
	gs.Spec.Template.Spec.Containers[0].Env = []corev1.EnvVar{{Name: "PASSTHROUGH", Value: "TRUE"}}
	// gate
	_, valid := gs.Validate()
	assert.True(t, valid)

	readyGs, err := framework.CreateGameServerAndWaitUntilReady(t, framework.Namespace, gs)
	if err != nil {
		assert.FailNow(t, "Could not get a GameServer ready", err.Error())
	}

	port := readyGs.Spec.Ports[0]
	assert.Equal(t, agonesv1.Passthrough, port.PortPolicy)
	assert.NotEmpty(t, port.HostPort)
	assert.Equal(t, port.HostPort, port.ContainerPort)

	reply, err := framework.SendGameServerUDP(t, readyGs, "Hello World !")
	if err != nil {
		t.Fatalf("Could ping GameServer: %v", err)
	}

	assert.Equal(t, "ACK: Hello World !\n", reply)
}

func TestGameServerTcpProtocol(t *testing.T) {
	t.Parallel()
	gs := framework.DefaultGameServer(framework.Namespace)

	gs.Spec.Ports[0].Protocol = corev1.ProtocolTCP
	gs.Spec.Template.Spec.Containers[0].Env = []corev1.EnvVar{{Name: "TCP", Value: "TRUE"}}

	_, valid := gs.Validate()
	require.True(t, valid)

	readyGs, err := framework.CreateGameServerAndWaitUntilReady(t, framework.Namespace, gs)
	require.NoError(t, err)

	replyTCP, err := e2eframework.SendGameServerTCP(readyGs, "Hello World !")
	require.NoError(t, err)
	assert.Equal(t, "ACK TCP: Hello World !\n", replyTCP)
}

func TestGameServerTcpUdpProtocol(t *testing.T) {
	t.Parallel()
	gs := framework.DefaultGameServer(framework.Namespace)

	gs.Spec.Ports[0].Protocol = agonesv1.ProtocolTCPUDP
	gs.Spec.Ports[0].Name = "gameserver"
	gs.Spec.Template.Spec.Containers[0].Env = []corev1.EnvVar{{Name: "TCP", Value: "TRUE"}}

	_, valid := gs.Validate()
	assert.True(t, valid)

	readyGs, err := framework.CreateGameServerAndWaitUntilReady(t, framework.Namespace, gs)
	if err != nil {
		assert.FailNow(t, "Could not get a GameServer ready", err.Error())
	}

	tcpPort := readyGs.Spec.Ports[0]
	assert.Equal(t, corev1.ProtocolTCP, tcpPort.Protocol)
	assert.NotEmpty(t, tcpPort.HostPort)
	assert.Equal(t, "gameserver-tcp", tcpPort.Name)

	udpPort := readyGs.Spec.Ports[1]
	assert.Equal(t, corev1.ProtocolUDP, udpPort.Protocol)
	assert.NotEmpty(t, udpPort.HostPort)
	assert.Equal(t, "gameserver-udp", udpPort.Name)

	assert.Equal(t, tcpPort.HostPort, udpPort.HostPort)

	logrus.WithField("name", readyGs.ObjectMeta.Name).Info("GameServer created, sending UDP ping")

	replyUDP, err := framework.SendGameServerUDPToPort(t, readyGs, udpPort.Name, "Hello World !")
	require.NoError(t, err)
	if err != nil {
		t.Fatalf("Could not ping UDP GameServer: %v", err)
	}

	assert.Equal(t, "ACK: Hello World !\n", replyUDP)

	logrus.WithField("name", readyGs.ObjectMeta.Name).Info("UDP ping passed, sending TCP ping")

	replyTCP, err := e2eframework.SendGameServerTCPToPort(readyGs, tcpPort.Name, "Hello World !")
	if err != nil {
		t.Fatalf("Could not ping TCP GameServer: %v", err)
	}

	assert.Equal(t, "ACK TCP: Hello World !\n", replyTCP)
}

func TestGameServerWithoutPort(t *testing.T) {
	t.Parallel()
	gs := framework.DefaultGameServer(framework.Namespace)
	gs.Spec.Ports = nil

	_, valid := gs.Validate()
	assert.True(t, valid)

	readyGs, err := framework.CreateGameServerAndWaitUntilReady(t, framework.Namespace, gs)

	require.NoError(t, err, "Could not get a GameServer ready")
	assert.Empty(t, readyGs.Spec.Ports)
}

// TestGameServerResourceValidation - check that we are not able to use
// invalid PodTemplate for GameServer Spec with wrong Resource Requests and Limits
func TestGameServerResourceValidation(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	gs := framework.DefaultGameServer(framework.Namespace)
	mi64 := resource.MustParse("64Mi")
	gs.Spec.Template.Spec.Containers[0].Resources.Limits[corev1.ResourceMemory] = mi64
	gs.Spec.Template.Spec.Containers[0].Resources.Requests[corev1.ResourceMemory] = resource.MustParse("128Mi")

	_, valid := gs.Validate()
	assert.False(t, valid)

	gsClient := framework.AgonesClient.AgonesV1().GameServers(framework.Namespace)

	_, err := gsClient.Create(ctx, gs.DeepCopy(), metav1.CreateOptions{})
	assert.NotNil(t, err)
	statusErr, ok := err.(*k8serrors.StatusError)
	assert.True(t, ok)
	assert.Len(t, statusErr.Status().Details.Causes, 1)
	assert.Equal(t, metav1.CauseTypeFieldValueInvalid, statusErr.Status().Details.Causes[0].Type)
	assert.Equal(t, "container", statusErr.Status().Details.Causes[0].Field)

	gs.Spec.Template.Spec.Containers[0].Resources.Requests[corev1.ResourceCPU] = resource.MustParse("-50m")
	_, err = gsClient.Create(ctx, gs.DeepCopy(), metav1.CreateOptions{})
	assert.NotNil(t, err)
	statusErr, ok = err.(*k8serrors.StatusError)
	assert.True(t, ok)
	assert.Len(t, statusErr.Status().Details.Causes, 2)
	assert.Equal(t, metav1.CauseTypeFieldValueInvalid, statusErr.Status().Details.Causes[0].Type)
	assert.Equal(t, "container", statusErr.Status().Details.Causes[0].Field)

	// test that values are still being set correctly
	m50 := resource.MustParse("50m")
	gs.Spec.Template.Spec.Containers[0].Resources.Limits[corev1.ResourceMemory] = mi64
	gs.Spec.Template.Spec.Containers[0].Resources.Requests[corev1.ResourceMemory] = mi64
	gs.Spec.Template.Spec.Containers[0].Resources.Requests[corev1.ResourceCPU] = m50
	gs.Spec.Template.Spec.Containers[0].Resources.Limits[corev1.ResourceCPU] = m50

	// confirm we have a valid GameServer before running the test
	cause, valid := gs.Validate()
	require.True(t, valid, cause)

	gsCopy, err := gsClient.Create(ctx, gs.DeepCopy(), metav1.CreateOptions{})
	require.NoError(t, err)
	assert.Equal(t, mi64, gsCopy.Spec.Template.Spec.Containers[0].Resources.Limits[corev1.ResourceMemory])
	assert.Equal(t, mi64, gsCopy.Spec.Template.Spec.Containers[0].Resources.Requests[corev1.ResourceMemory])
	assert.Equal(t, m50, gsCopy.Spec.Template.Spec.Containers[0].Resources.Requests[corev1.ResourceCPU])
	assert.Equal(t, m50, gsCopy.Spec.Template.Spec.Containers[0].Resources.Limits[corev1.ResourceCPU])
}

func TestGameServerPodTemplateSpecFailSchemaValidation(t *testing.T) {
	t.Parallel()

	// The Kubernetes dynamic client skips schema validation (for reasons I'm not sure of), so the
	// best way I could determine to test schema validation is via calling kubectl directly.
	// The schema's from Kubernetes don't include anything like `pattern:` or `enum:` which would
	// potentially make this easier to test.

	gsYaml := `
apiVersion: "agones.dev/v1"
kind: GameServer
metadata:
  name: "invalid-gameserver"
spec:
  ports:
    - name: default
      portPolicy: Dynamic
      containerPort: 7654
  template:
    spec:
      affinity:
        nodeAffinity:
          preferredDuringSchedulingIgnoredDuringExecution: ERROR
      containers:
        - name: simple-game-server
          image: gcr.io/agones-images/simple-game-server:0.14
`
	err := os.WriteFile("/tmp/invalid.yaml", []byte(gsYaml), 0o644)
	require.NoError(t, err)

	cmd := exec.Command("kubectl", "apply", "-f", "/tmp/invalid.yaml")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err = cmd.Run()
	logrus.WithField("stdout", stdout.String()).WithField("stderr", stderr.String()).WithError(err).Info("Ran command!")
	require.Error(t, err)
	assert.Contains(t, stderr.String(), "ValidationError(GameServer.spec.template.spec.affinity.nodeAffinity.preferredDuringSchedulingIgnoredDuringExecution)")
}

func TestGameServerSetPlayerCapacity(t *testing.T) {
	if !runtime.FeatureEnabled(runtime.FeaturePlayerTracking) {
		t.SkipNow()
	}
	t.Parallel()
	ctx := context.Background()

	t.Run("no initial capacity set", func(t *testing.T) {
		gs := framework.DefaultGameServer(framework.Namespace)
		gs, err := framework.CreateGameServerAndWaitUntilReady(t, framework.Namespace, gs)
		if err != nil {
			t.Fatalf("Could not get a GameServer ready: %v", err)
		}
		assert.Equal(t, gs.Status.State, agonesv1.GameServerStateReady)
		assert.Equal(t, int64(0), gs.Status.Players.Capacity)

		reply, err := framework.SendGameServerUDP(t, gs, "PLAYER_CAPACITY")
		if err != nil {
			t.Fatalf("Could not message GameServer: %v", err)
		}
		assert.Equal(t, "0\n", reply)

		reply, err = framework.SendGameServerUDP(t, gs, "PLAYER_CAPACITY 20")
		if err != nil {
			t.Fatalf("Could not message GameServer: %v", err)
		}
		assert.Equal(t, "ACK: PLAYER_CAPACITY 20\n", reply)

		reply, err = framework.SendGameServerUDP(t, gs, "PLAYER_CAPACITY")
		if err != nil {
			t.Fatalf("Could not message GameServer: %v", err)
		}
		assert.Equal(t, "20\n", reply)

		err = wait.PollImmediate(time.Second, time.Minute, func() (bool, error) {
			gs, err := framework.AgonesClient.AgonesV1().GameServers(framework.Namespace).Get(ctx, gs.ObjectMeta.Name, metav1.GetOptions{})
			if err != nil {
				return false, err
			}
			return gs.Status.Players.Capacity == 20, nil
		})
		assert.NoError(t, err)
	})

	t.Run("initial capacity set", func(t *testing.T) {
		gs := framework.DefaultGameServer(framework.Namespace)
		gs.Spec.Players = &agonesv1.PlayersSpec{InitialCapacity: 10}
		gs, err := framework.CreateGameServerAndWaitUntilReady(t, framework.Namespace, gs)
		if err != nil {
			t.Fatalf("Could not get a GameServer ready: %v", err)
		}
		assert.Equal(t, gs.Status.State, agonesv1.GameServerStateReady)
		assert.Equal(t, int64(10), gs.Status.Players.Capacity)

		reply, err := framework.SendGameServerUDP(t, gs, "PLAYER_CAPACITY")
		if err != nil {
			t.Fatalf("Could not message GameServer: %v", err)
		}
		assert.Equal(t, "10\n", reply)

		reply, err = framework.SendGameServerUDP(t, gs, "PLAYER_CAPACITY 20")
		if err != nil {
			t.Fatalf("Could not message GameServer: %v", err)
		}
		assert.Equal(t, "ACK: PLAYER_CAPACITY 20\n", reply)

		reply, err = framework.SendGameServerUDP(t, gs, "PLAYER_CAPACITY")
		if err != nil {
			t.Fatalf("Could not message GameServer: %v", err)
		}
		assert.Equal(t, "20\n", reply)

		err = wait.PollImmediate(time.Second, time.Minute, func() (bool, error) {
			gs, err := framework.AgonesClient.AgonesV1().GameServers(framework.Namespace).Get(ctx, gs.ObjectMeta.Name, metav1.GetOptions{})
			if err != nil {
				return false, err
			}
			return gs.Status.Players.Capacity == 20, nil
		})
		assert.NoError(t, err)

		time.Sleep(30 * time.Second)
	})
}

func TestPlayerConnectWithCapacityZero(t *testing.T) {
	if !runtime.FeatureEnabled(runtime.FeaturePlayerTracking) {
		t.SkipNow()
	}
	t.Parallel()
	ctx := context.Background()

	gs := framework.DefaultGameServer(framework.Namespace)
	playerCount := int64(0)
	gs.Spec.Players = &agonesv1.PlayersSpec{InitialCapacity: playerCount}
	gs, err := framework.CreateGameServerAndWaitUntilReady(t, framework.Namespace, gs)
	require.NoError(t, err)
	assert.Equal(t, gs.Status.State, agonesv1.GameServerStateReady)
	assert.Equal(t, playerCount, gs.Status.Players.Capacity)

	// add a player
	msg := "PLAYER_CONNECT 1"
	logrus.WithField("msg", msg).Info("Sending Player Connect")
	_, err = framework.SendGameServerUDP(t, gs, msg)
	// expected error from the log.Fatalf("could not connect player: %v", err)
	require.Error(t, err)
	assert.Eventually(t, func() bool {
		gs, err = framework.AgonesClient.AgonesV1().GameServers(framework.Namespace).Get(ctx, gs.ObjectMeta.Name, metav1.GetOptions{})
		require.NoError(t, err)

		return assert.Equal(t, gs.Status.State, agonesv1.GameServerStateUnhealthy)
	}, time.Minute, time.Second)
}

func TestPlayerConnectAndDisconnect(t *testing.T) {
	if !runtime.FeatureEnabled(runtime.FeaturePlayerTracking) {
		t.SkipNow()
	}
	t.Parallel()
	ctx := context.Background()

	gs := framework.DefaultGameServer(framework.Namespace)
	playerCount := int64(3)
	gs.Spec.Players = &agonesv1.PlayersSpec{InitialCapacity: playerCount}
	gs, err := framework.CreateGameServerAndWaitUntilReady(t, framework.Namespace, gs)
	if err != nil {
		t.Fatalf("Could not get a GameServer ready: %v", err)
	}
	assert.Equal(t, gs.Status.State, agonesv1.GameServerStateReady)
	assert.Equal(t, playerCount, gs.Status.Players.Capacity)

	// add three players in quick succession
	for i := int64(1); i <= playerCount; i++ {
		msg := "PLAYER_CONNECT " + fmt.Sprintf("%d", i)
		logrus.WithField("msg", msg).Info("Sending Player Connect")
		reply, err := framework.SendGameServerUDP(t, gs, msg)
		if err != nil {
			t.Fatalf("Could not message GameServer: %v", err)
		}
		assert.Equal(t, fmt.Sprintf("ACK: %s\n", msg), reply)
	}

	// deliberately do this before polling, to test the SDK returning the correct
	// results before it is committed to the GameServer resource.
	reply, err := framework.SendGameServerUDP(t, gs, "PLAYER_CONNECTED 1")
	if err != nil {
		t.Fatalf("Could not message GameServer: %v", err)
	}
	assert.Equal(t, "true\n", reply)

	reply, err = framework.SendGameServerUDP(t, gs, "GET_PLAYERS")
	if err != nil {
		t.Fatalf("Could not message GameServer: %v", err)
	}
	assert.ElementsMatch(t, []string{"1", "2", "3"}, strings.Split(strings.TrimSpace(reply), ","))

	reply, err = framework.SendGameServerUDP(t, gs, "PLAYER_COUNT")
	if err != nil {
		t.Fatalf("Could not message GameServer: %v", err)
	}
	assert.Equal(t, "3\n", reply)

	err = wait.Poll(time.Second, time.Minute, func() (bool, error) {
		gs, err = framework.AgonesClient.AgonesV1().GameServers(framework.Namespace).Get(ctx, gs.ObjectMeta.Name, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		return gs.Status.Players.Count == playerCount, nil
	})
	assert.NoError(t, err)
	assert.ElementsMatch(t, []string{"1", "2", "3"}, gs.Status.Players.IDs)

	// let's disconnect player 2
	logrus.Info("Disconnect Player 2")
	reply, err = framework.SendGameServerUDP(t, gs, "PLAYER_DISCONNECT 2")
	if err != nil {
		t.Fatalf("Could not message GameServer: %v", err)
	}
	assert.Equal(t, "ACK: PLAYER_DISCONNECT 2\n", reply)

	reply, err = framework.SendGameServerUDP(t, gs, "PLAYER_CONNECTED 2")
	if err != nil {
		t.Fatalf("Could not message GameServer: %v", err)
	}
	assert.Equal(t, "false\n", reply)

	reply, err = framework.SendGameServerUDP(t, gs, "GET_PLAYERS")
	if err != nil {
		t.Fatalf("Could not message GameServer: %v", err)
	}
	assert.ElementsMatch(t, []string{"1", "3"}, strings.Split(strings.TrimSpace(reply), ","))

	reply, err = framework.SendGameServerUDP(t, gs, "PLAYER_COUNT")
	if err != nil {
		t.Fatalf("Could not message GameServer: %v", err)
	}
	assert.Equal(t, "2\n", reply)

	err = wait.Poll(time.Second, time.Minute, func() (bool, error) {
		gs, err = framework.AgonesClient.AgonesV1().GameServers(framework.Namespace).Get(ctx, gs.ObjectMeta.Name, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		return gs.Status.Players.Count == 2, nil
	})
	assert.NoError(t, err)
	assert.ElementsMatch(t, []string{"1", "3"}, gs.Status.Players.IDs)
}

func TestGracefulShutdown(t *testing.T) {
	if !runtime.FeatureEnabled(runtime.FeatureSDKGracefulTermination) {
		t.SkipNow()
	}
	t.Parallel()

	log := e2eframework.TestLogger(t)
	ctx := context.Background()
	gs := framework.DefaultGameServer(framework.Namespace)
	var minute int64 = 60
	gs.Spec.Template.Spec.TerminationGracePeriodSeconds = &minute
	readyGs, err := framework.CreateGameServerAndWaitUntilReady(t, framework.Namespace, gs)
	if err != nil {
		t.Fatalf("Could not get a GameServer ready: %v", err)
	}
	assert.Equal(t, readyGs.Status.State, agonesv1.GameServerStateReady)
	gameservers := framework.AgonesClient.AgonesV1().GameServers(framework.Namespace)
	err = gameservers.Delete(ctx, readyGs.ObjectMeta.Name, metav1.DeleteOptions{})
	require.NoError(t, err)
	log.Info("Deleted GameServer, waiting 20 seconds...")
	time.Sleep(20 * time.Second)
	log.WithField("gs", gs).Info("Checking GameServer")
	gs, err = gameservers.Get(ctx, readyGs.ObjectMeta.Name, metav1.GetOptions{})
	require.NoError(t, err)
	assert.Equal(t, readyGs.ObjectMeta.Name, gs.ObjectMeta.Name)

	// move it to shutdown
	gsCopy := gs.DeepCopy()
	gsCopy.Status.State = agonesv1.GameServerStateShutdown
	_, err = gameservers.Update(ctx, gsCopy, metav1.UpdateOptions{})
	require.NoError(t, err)

	start := time.Now()
	require.Eventually(t, func() bool {
		_, err := gameservers.Get(ctx, readyGs.ObjectMeta.Name, metav1.GetOptions{})
		log.WithError(err).Info("checking GameServer")
		if err == nil {
			return false
		}
		return k8serrors.IsNotFound(err)
	}, 40*time.Second, time.Second)

	diff := int(time.Since(start).Seconds())
	log.WithField("diff", diff).Info("Time difference")
	require.Less(t, diff, 40)
}
