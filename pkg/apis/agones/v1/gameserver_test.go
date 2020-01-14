// Copyright 2017 Google LLC All Rights Reserved.
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

package v1

import (
	"fmt"
	"strings"
	"testing"

	"agones.dev/agones/pkg"
	"agones.dev/agones/pkg/apis"
	"agones.dev/agones/pkg/apis/agones"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation"
)

const (
	ipFixture = "127.1.1.1"
)

func TestGameServerFindGameServerContainer(t *testing.T) {
	t.Parallel()

	fixture := corev1.Container{Name: "mycontainer", Image: "foo/mycontainer"}
	gs := &GameServer{
		Spec: GameServerSpec{
			Container: "mycontainer",
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						fixture,
						{Name: "notmycontainer", Image: "foo/notmycontainer"},
					},
				},
			},
		},
	}

	i, container, err := gs.FindGameServerContainer()
	assert.Nil(t, err)
	assert.Equal(t, fixture, container)
	container.Ports = append(container.Ports, corev1.ContainerPort{HostPort: 1234})
	gs.Spec.Template.Spec.Containers[i] = container
	assert.Equal(t, gs.Spec.Template.Spec.Containers[0], container)
}

func TestGameServerApplyDefaults(t *testing.T) {
	t.Parallel()

	type expected struct {
		protocol   corev1.Protocol
		state      GameServerState
		policy     PortPolicy
		health     Health
		scheduling apis.SchedulingStrategy
		sdkServer  SdkServer
	}
	data := map[string]struct {
		gameServer GameServer
		container  string
		expected   expected
	}{
		"set basic defaults on a very simple gameserver": {
			gameServer: GameServer{
				Spec: GameServerSpec{
					Ports: []GameServerPort{{ContainerPort: 999}},
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{Containers: []corev1.Container{
							{Name: "testing", Image: "testing/image"},
						}}}},
			},
			container: "testing",
			expected: expected{
				protocol:   "UDP",
				state:      GameServerStatePortAllocation,
				policy:     Dynamic,
				scheduling: apis.Packed,
				health: Health{
					Disabled:            false,
					FailureThreshold:    3,
					InitialDelaySeconds: 5,
					PeriodSeconds:       5,
				},
				sdkServer: SdkServer{
					LogLevel: SdkServerLogLevelInfo,
					GRPCPort: 9357,
					HTTPPort: 9358,
				},
			},
		},
		"defaults on passthrough": {
			gameServer: GameServer{
				Spec: GameServerSpec{
					Ports: []GameServerPort{{PortPolicy: Passthrough}},
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{Containers: []corev1.Container{
							{Name: "testing", Image: "testing/image"},
						}}}},
			},
			container: "testing",
			expected: expected{
				protocol:   "UDP",
				state:      GameServerStatePortAllocation,
				policy:     Passthrough,
				scheduling: apis.Packed,
				health: Health{
					Disabled:            false,
					FailureThreshold:    3,
					InitialDelaySeconds: 5,
					PeriodSeconds:       5,
				},
				sdkServer: SdkServer{
					LogLevel: SdkServerLogLevelInfo,
					GRPCPort: 9357,
					HTTPPort: 9358,
				},
			},
		},
		"defaults are already set": {
			gameServer: GameServer{
				Spec: GameServerSpec{
					Container: "testing2",
					Ports: []GameServerPort{{
						Protocol:   "TCP",
						PortPolicy: Static,
					}},
					Health: Health{
						Disabled:            false,
						PeriodSeconds:       12,
						InitialDelaySeconds: 11,
						FailureThreshold:    10,
					},
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{Name: "testing", Image: "testing/image"},
								{Name: "testing2", Image: "testing/image2"}}},
					},
					SdkServer: SdkServer{
						LogLevel: SdkServerLogLevelInfo,
						GRPCPort: 9357,
						HTTPPort: 9358,
					},
				},
				Status: GameServerStatus{State: "TestState"}},
			container: "testing2",
			expected: expected{
				protocol:   "TCP",
				state:      "TestState",
				policy:     Static,
				scheduling: apis.Packed,
				health: Health{
					Disabled:            false,
					FailureThreshold:    10,
					InitialDelaySeconds: 11,
					PeriodSeconds:       12,
				},
				sdkServer: SdkServer{
					LogLevel: SdkServerLogLevelInfo,
					GRPCPort: 9357,
					HTTPPort: 9358,
				},
			},
		},
		"set basic defaults on static gameserver": {
			gameServer: GameServer{
				Spec: GameServerSpec{
					Ports: []GameServerPort{{PortPolicy: Static}},
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "testing", Image: "testing/image"}}}}},
			},
			container: "testing",
			expected: expected{
				protocol:   "UDP",
				state:      GameServerStateCreating,
				policy:     Static,
				scheduling: apis.Packed,
				health: Health{
					Disabled:            false,
					FailureThreshold:    3,
					InitialDelaySeconds: 5,
					PeriodSeconds:       5,
				},
				sdkServer: SdkServer{
					LogLevel: SdkServerLogLevelInfo,
					GRPCPort: 9357,
					HTTPPort: 9358,
				},
			},
		},
		"health is disabled": {
			gameServer: GameServer{
				Spec: GameServerSpec{
					Ports:  []GameServerPort{{ContainerPort: 999}},
					Health: Health{Disabled: true},
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "testing", Image: "testing/image"}}}}},
			},
			container: "testing",
			expected: expected{
				protocol:   "UDP",
				state:      GameServerStatePortAllocation,
				policy:     Dynamic,
				scheduling: apis.Packed,
				health: Health{
					Disabled: true,
				},
				sdkServer: SdkServer{
					LogLevel: SdkServerLogLevelInfo,
					GRPCPort: 9357,
					HTTPPort: 9358,
				},
			},
		},
		"convert from legacy single port to multiple": {
			gameServer: GameServer{
				Spec: GameServerSpec{
					Ports: []GameServerPort{
						{
							ContainerPort: 777,
							HostPort:      777,
							PortPolicy:    Static,
							Protocol:      corev1.ProtocolTCP,
						},
					},
					Health: Health{Disabled: true},
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "testing", Image: "testing/image"}}}}},
			},
			container: "testing",
			expected: expected{
				protocol:   corev1.ProtocolTCP,
				state:      GameServerStateCreating,
				policy:     Static,
				scheduling: apis.Packed,
				health:     Health{Disabled: true},
				sdkServer: SdkServer{
					LogLevel: SdkServerLogLevelInfo,
					GRPCPort: 9357,
					HTTPPort: 9358,
				},
			},
		},
		"set Debug logging level": {
			gameServer: GameServer{
				Spec: GameServerSpec{
					Ports:     []GameServerPort{{ContainerPort: 999}},
					SdkServer: SdkServer{LogLevel: SdkServerLogLevelDebug},
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{Containers: []corev1.Container{
							{Name: "testing", Image: "testing/image"},
						}}}},
			},
			container: "testing",
			expected: expected{
				protocol:   "UDP",
				state:      GameServerStatePortAllocation,
				policy:     Dynamic,
				scheduling: apis.Packed,
				health: Health{
					Disabled:            false,
					FailureThreshold:    3,
					InitialDelaySeconds: 5,
					PeriodSeconds:       5,
				},
				sdkServer: SdkServer{
					LogLevel: SdkServerLogLevelDebug,
					GRPCPort: 9357,
					HTTPPort: 9358,
				},
			},
		},
		"set gRPC and HTTP ports on SDK Server": {
			gameServer: GameServer{
				Spec: GameServerSpec{
					Ports: []GameServerPort{{ContainerPort: 999}},
					SdkServer: SdkServer{
						LogLevel: SdkServerLogLevelError,
						GRPCPort: 19357,
						HTTPPort: 19358,
					},
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{Containers: []corev1.Container{
							{Name: "testing", Image: "testing/image"},
						}}}},
			},
			container: "testing",
			expected: expected{
				protocol:   "UDP",
				state:      GameServerStatePortAllocation,
				policy:     Dynamic,
				scheduling: apis.Packed,
				health: Health{
					Disabled:            false,
					FailureThreshold:    3,
					InitialDelaySeconds: 5,
					PeriodSeconds:       5,
				},
				sdkServer: SdkServer{
					LogLevel: SdkServerLogLevelError,
					GRPCPort: 19357,
					HTTPPort: 19358,
				},
			},
		},
	}

	for name, test := range data {
		t.Run(name, func(t *testing.T) {
			test.gameServer.ApplyDefaults()

			assert.Equal(t, pkg.Version, test.gameServer.Annotations[VersionAnnotation])

			spec := test.gameServer.Spec
			assert.Contains(t, test.gameServer.ObjectMeta.Finalizers, agones.GroupName)
			assert.Equal(t, test.container, spec.Container)
			assert.Equal(t, test.expected.protocol, spec.Ports[0].Protocol)
			assert.Equal(t, test.expected.state, test.gameServer.Status.State)
			assert.Equal(t, test.expected.scheduling, test.gameServer.Spec.Scheduling)
			assert.Equal(t, test.expected.health, test.gameServer.Spec.Health)
			assert.Equal(t, test.expected.sdkServer, test.gameServer.Spec.SdkServer)
		})
	}
}

func TestGameServerValidate(t *testing.T) {
	gs := GameServer{
		Spec: GameServerSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "testing", Image: "testing/image"}}}}},
	}
	gs.ApplyDefaults()
	causes, ok := gs.Validate()
	assert.True(t, ok)
	assert.Empty(t, causes)

	gs = GameServer{
		Spec: GameServerSpec{
			Container: "",
			Ports: []GameServerPort{{
				Name:       "main",
				HostPort:   5001,
				PortPolicy: Dynamic,
			}},
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{Containers: []corev1.Container{
					{Name: "testing", Image: "testing/image"},
					{Name: "anothertest", Image: "testing/image"},
				}}}},
	}
	causes, ok = gs.Validate()
	var fields []string
	for _, f := range causes {
		fields = append(fields, f.Field)
	}
	assert.False(t, ok)
	assert.Len(t, causes, 4)
	assert.Contains(t, fields, "container")
	assert.Contains(t, fields, "main.hostPort")
	assert.Contains(t, fields, "main.containerPort")
	assert.Equal(t, causes[0].Type, metav1.CauseTypeFieldValueInvalid)

	gs = GameServer{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "dev-game",
			Namespace:   "default",
			Annotations: map[string]string{DevAddressAnnotation: "invalid-ip"},
		},
		Spec: GameServerSpec{
			Ports: []GameServerPort{{Name: "main", ContainerPort: 7777, PortPolicy: Static}},
		},
	}
	causes, ok = gs.Validate()
	for _, f := range causes {
		fields = append(fields, f.Field)
	}
	assert.False(t, ok)
	assert.Len(t, causes, 2)
	assert.Contains(t, fields, fmt.Sprintf("annotations.%s", DevAddressAnnotation))
	assert.Contains(t, fields, "main.hostPort")
	assert.Equal(t, causes[1].Type, metav1.CauseTypeFieldValueRequired)

	gs = GameServer{
		Spec: GameServerSpec{
			Container: "my_image",
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{Name: "my_image", Image: "foo/my_image"},
					},
				},
			},
		},
	}

	longName := strings.Repeat("f", validation.LabelValueMaxLength+1)
	gs.Name = longName
	causes, ok = gs.Validate()
	assert.False(t, ok)
	assert.Len(t, causes, 1)
	assert.Equal(t, "Name", causes[0].Field)

	gs.Name = ""
	gs.GenerateName = longName
	causes, ok = gs.Validate()
	assert.True(t, ok)
	assert.Len(t, causes, 0)

	gs.Spec.Template.ObjectMeta.Labels = make(map[string]string)
	gs.Spec.Template.ObjectMeta.Labels[longName] = ""
	causes, ok = gs.Validate()
	assert.False(t, ok)
	assert.Len(t, causes, 1)
	assert.Equal(t, "labels", causes[0].Field)

	gs.Spec.Template.ObjectMeta.Labels = make(map[string]string)
	gs.Spec.Template.ObjectMeta.Labels["agones.dev/longValueKey"] = longName
	causes, ok = gs.Validate()
	assert.False(t, ok)
	assert.Len(t, causes, 1)
	assert.Equal(t, "labels", causes[0].Field)

	// Validate Labels and Annotations
	gs.Spec.Template.ObjectMeta.Annotations = make(map[string]string)
	gs.Spec.Template.ObjectMeta.Annotations[longName] = longName
	causes, ok = gs.Validate()
	assert.False(t, ok)
	assert.Len(t, causes, 2)

	// No errors if valid Annotation was used
	gs.Spec.Template.ObjectMeta.Labels = make(map[string]string)
	gs.Spec.Template.ObjectMeta.Annotations = make(map[string]string)
	shortName := "agones.dev/shortName"
	gs.Spec.Template.ObjectMeta.Annotations[shortName] = "shortValue"
	causes, ok = gs.Validate()
	assert.True(t, ok)
	assert.Len(t, causes, 0)

	gs.Spec.Template.ObjectMeta.Annotations[shortName] = longName
	causes, ok = gs.Validate()
	assert.True(t, ok)
	assert.Len(t, causes, 0)

	gs.Spec.Template.ObjectMeta.Annotations["agones.dev/short±Name"] = "shortValue"
	causes, ok = gs.Validate()
	assert.False(t, ok)
	assert.Len(t, causes, 1)
	assert.Equal(t, "annotations", causes[0].Field)

	gs = GameServer{
		Spec: GameServerSpec{
			Ports: []GameServerPort{{Name: "one", PortPolicy: Passthrough, ContainerPort: 1294}, {PortPolicy: Passthrough, Name: "two", HostPort: 7890}},
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "testing", Image: "testing/image"}}}},
		},
	}
	gs.ApplyDefaults()
	causes, ok = gs.Validate()
	for _, f := range causes {
		fields = append(fields, f.Field)
	}
	assert.False(t, ok)
	assert.Len(t, causes, 2)
	assert.Contains(t, fields, "one.containerPort")
	assert.Contains(t, fields, "two.hostPort")
}

func TestGameServerPod(t *testing.T) {
	fixture := defaultGameServer()
	fixture.ApplyDefaults()

	pod, err := fixture.Pod()
	assert.Nil(t, err, "Pod should not return an error")
	assert.Equal(t, fixture.ObjectMeta.Name, pod.ObjectMeta.Name)
	assert.Equal(t, fixture.ObjectMeta.Namespace, pod.ObjectMeta.Namespace)
	assert.Equal(t, "gameserver", pod.ObjectMeta.Labels[agones.GroupName+"/role"])
	assert.Equal(t, fixture.ObjectMeta.Name, pod.ObjectMeta.Labels[GameServerPodLabel])
	assert.Equal(t, fixture.Spec.Container, pod.ObjectMeta.Annotations[GameServerContainerAnnotation])
	assert.True(t, metav1.IsControlledBy(pod, fixture))
	assert.Equal(t, fixture.Spec.Ports[0].HostPort, pod.Spec.Containers[0].Ports[0].HostPort)
	assert.Equal(t, fixture.Spec.Ports[0].ContainerPort, pod.Spec.Containers[0].Ports[0].ContainerPort)
	assert.Equal(t, corev1.Protocol("UDP"), pod.Spec.Containers[0].Ports[0].Protocol)
	assert.True(t, metav1.IsControlledBy(pod, fixture))

	sidecar := corev1.Container{Name: "sidecar", Image: "container/sidecar"}
	fixture.Spec.Template.Spec.ServiceAccountName = "other-agones-sdk"
	pod, err = fixture.Pod(sidecar)
	assert.Nil(t, err, "Pod should not return an error")
	assert.Equal(t, fixture.ObjectMeta.Name, pod.ObjectMeta.Name)
	assert.Len(t, pod.Spec.Containers, 2, "Should have two containers")
	assert.Equal(t, "other-agones-sdk", pod.Spec.ServiceAccountName)
	assert.Equal(t, "container", pod.Spec.Containers[0].Name)
	assert.Equal(t, "sidecar", pod.Spec.Containers[1].Name)
	assert.True(t, metav1.IsControlledBy(pod, fixture))
}

func TestGameServerPodObjectMeta(t *testing.T) {
	fixture := &GameServer{ObjectMeta: metav1.ObjectMeta{Name: "lucy"},
		Spec: GameServerSpec{Container: "goat"}}

	f := func(t *testing.T, gs *GameServer, pod *corev1.Pod) {
		assert.Equal(t, gs.ObjectMeta.Name, pod.ObjectMeta.Name)
		assert.Equal(t, gs.ObjectMeta.Namespace, pod.ObjectMeta.Namespace)
		assert.Equal(t, GameServerLabelRole, pod.ObjectMeta.Labels[RoleLabel])
		assert.Equal(t, "gameserver", pod.ObjectMeta.Labels[agones.GroupName+"/role"])
		assert.Equal(t, gs.ObjectMeta.Name, pod.ObjectMeta.Labels[GameServerPodLabel])
		assert.Equal(t, "goat", pod.ObjectMeta.Annotations[GameServerContainerAnnotation])
		assert.True(t, metav1.IsControlledBy(pod, gs))
	}

	t.Run("packed", func(t *testing.T) {
		gs := fixture.DeepCopy()
		gs.Spec.Scheduling = apis.Packed
		pod := &corev1.Pod{}

		gs.podObjectMeta(pod)
		f(t, gs, pod)

		assert.Equal(t, "false", pod.ObjectMeta.Annotations["cluster-autoscaler.kubernetes.io/safe-to-evict"])
	})

	t.Run("distributed", func(t *testing.T) {
		gs := fixture.DeepCopy()
		gs.Spec.Scheduling = apis.Distributed
		pod := &corev1.Pod{}

		gs.podObjectMeta(pod)
		f(t, gs, pod)

		assert.Equal(t, "", pod.ObjectMeta.Annotations["cluster-autoscaler.kubernetes.io/safe-to-evict"])
	})
}

func TestGameServerPodScheduling(t *testing.T) {
	fixture := &corev1.Pod{Spec: corev1.PodSpec{}}

	t.Run("packed", func(t *testing.T) {
		gs := &GameServer{Spec: GameServerSpec{Scheduling: apis.Packed}}
		pod := fixture.DeepCopy()
		gs.podScheduling(pod)

		assert.Len(t, pod.Spec.Affinity.PodAffinity.PreferredDuringSchedulingIgnoredDuringExecution, 1)
		wpat := pod.Spec.Affinity.PodAffinity.PreferredDuringSchedulingIgnoredDuringExecution[0]
		assert.Equal(t, int32(100), wpat.Weight)
		assert.Contains(t, wpat.PodAffinityTerm.LabelSelector.String(), GameServerLabelRole)
		assert.Contains(t, wpat.PodAffinityTerm.LabelSelector.String(), RoleLabel)
	})

	t.Run("distributed", func(t *testing.T) {
		gs := &GameServer{Spec: GameServerSpec{Scheduling: apis.Distributed}}
		pod := fixture.DeepCopy()
		gs.podScheduling(pod)
		assert.Empty(t, pod.Spec.Affinity)
	})
}

func TestGameServerDisableServiceAccount(t *testing.T) {
	t.Parallel()

	gs := &GameServer{ObjectMeta: metav1.ObjectMeta{Name: "gameserver", UID: "1234"}, Spec: GameServerSpec{
		Template: corev1.PodTemplateSpec{
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{{Name: "container", Image: "container/image"}},
			},
		}}}

	gs.ApplyDefaults()
	pod, err := gs.Pod()
	assert.NoError(t, err)
	assert.Len(t, pod.Spec.Containers, 1)
	assert.Empty(t, pod.Spec.Containers[0].VolumeMounts)

	gs.DisableServiceAccount(pod)
	assert.Len(t, pod.Spec.Containers, 1)
	assert.Len(t, pod.Spec.Containers[0].VolumeMounts, 1)
	assert.Equal(t, "/var/run/secrets/kubernetes.io/serviceaccount", pod.Spec.Containers[0].VolumeMounts[0].MountPath)
}

func TestGameServerCountPorts(t *testing.T) {
	fixture := &GameServer{Spec: GameServerSpec{Ports: []GameServerPort{
		{PortPolicy: Dynamic},
		{PortPolicy: Dynamic},
		{PortPolicy: Dynamic},
		{PortPolicy: Static},
	}}}

	assert.Equal(t, 3, fixture.CountPorts(func(policy PortPolicy) bool {
		return policy == Dynamic
	}))
	assert.Equal(t, 1, fixture.CountPorts(func(policy PortPolicy) bool {
		return policy == Static
	}))
}

func TestGameServerPatch(t *testing.T) {
	fixture := &GameServer{ObjectMeta: metav1.ObjectMeta{Name: "lucy"},
		Spec: GameServerSpec{Container: "goat"}}

	delta := fixture.DeepCopy()
	delta.Spec.Container = "bear"

	patch, err := fixture.Patch(delta)
	assert.Nil(t, err)

	assert.Contains(t, string(patch), `{"op":"replace","path":"/spec/container","value":"bear"}`)
}

func TestGameServerGetDevAddress(t *testing.T) {
	devGs := &GameServer{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "dev-game",
			Namespace:   "default",
			Annotations: map[string]string{DevAddressAnnotation: ipFixture},
		},
		Spec: GameServerSpec{
			Ports: []GameServerPort{{HostPort: 7777, PortPolicy: Static}},
			Template: corev1.PodTemplateSpec{Spec: corev1.PodSpec{
				Containers: []corev1.Container{{Name: "container", Image: "container/image"}},
			},
			},
		},
	}

	devAddress, isDev := devGs.GetDevAddress()
	assert.True(t, isDev, "dev-game should had a dev-address")
	assert.Equal(t, ipFixture, devAddress, "dev-address IP address should be 127.1.1.1")

	regularGs := devGs.DeepCopy()
	regularGs.ObjectMeta.Annotations = map[string]string{}
	devAddress, isDev = regularGs.GetDevAddress()
	assert.False(t, isDev, "dev-game should NOT have a dev-address")
	assert.Equal(t, "", devAddress, "dev-address IP address should be 127.1.1.1")
}

func TestGameServerIsDeletable(t *testing.T) {
	gs := &GameServer{Status: GameServerStatus{State: GameServerStateStarting}}
	assert.True(t, gs.IsDeletable())

	gs.Status.State = GameServerStateAllocated
	assert.False(t, gs.IsDeletable())

	gs.Status.State = GameServerStateReserved
	assert.False(t, gs.IsDeletable())

	now := metav1.Now()
	gs.ObjectMeta.DeletionTimestamp = &now
	assert.True(t, gs.IsDeletable())

	gs.Status.State = GameServerStateAllocated
	assert.True(t, gs.IsDeletable())

	gs.Status.State = GameServerStateReady
	assert.True(t, gs.IsDeletable())
}

func TestGameServerIsBeforeReady(t *testing.T) {
	fixtures := []struct {
		state    GameServerState
		expected bool
	}{
		{GameServerStatePortAllocation, true},
		{GameServerStateCreating, true},
		{GameServerStateStarting, true},
		{GameServerStateScheduled, true},
		{GameServerStateRequestReady, true},
		{GameServerStateReady, false},
		{GameServerStateShutdown, false},
		{GameServerStateError, false},
		{GameServerStateUnhealthy, false},
		{GameServerStateReserved, false},
		{GameServerStateAllocated, false},
	}

	for _, test := range fixtures {
		t.Run(string(test.state), func(t *testing.T) {
			gs := &GameServer{Status: GameServerStatus{State: test.state}}
			assert.Equal(t, test.expected, gs.IsBeforeReady(), test.state)
		})
	}

}

func TestGameServerIsUnhealthy(t *testing.T) {
	fixtures := []struct {
		state    GameServerState
		expected bool
	}{
		{GameServerStatePortAllocation, false},
		{GameServerStateCreating, false},
		{GameServerStateStarting, false},
		{GameServerStateScheduled, false},
		{GameServerStateRequestReady, false},
		{GameServerStateReady, false},
		{GameServerStateShutdown, false},
		{GameServerStateError, true},
		{GameServerStateUnhealthy, true},
		{GameServerStateReserved, false},
		{GameServerStateAllocated, false},
	}

	for _, test := range fixtures {
		t.Run(string(test.state), func(t *testing.T) {
			gs := &GameServer{Status: GameServerStatus{State: test.state}}
			assert.Equal(t, test.expected, gs.IsUnhealthy(), test.state)
		})
	}

}

func TestGameServerApplyToPodGameServerContainer(t *testing.T) {
	t.Parallel()

	name := "mycontainer"
	gs := &GameServer{
		Spec: GameServerSpec{
			Container: name,
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{Name: name, Image: "foo/mycontainer"},
						{Name: "notmycontainer", Image: "foo/notmycontainer"},
					},
				},
			},
		},
	}

	p1 := &corev1.Pod{Spec: *gs.Spec.Template.Spec.DeepCopy()}

	p2 := gs.ApplyToPodGameServerContainer(p1, func(c corev1.Container) corev1.Container {
		//  easy thing to change and test for
		c.TTY = true

		return c
	})

	assert.Len(t, p2.Spec.Containers, 2)
	assert.True(t, p2.Spec.Containers[0].TTY)
	assert.False(t, p2.Spec.Containers[1].TTY)
}

func defaultGameServer() *GameServer {
	return &GameServer{ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "default", UID: "1234"},
		Spec: GameServerSpec{
			Ports: []GameServerPort{
				{
					ContainerPort: 7777,
					HostPort:      9999,
					PortPolicy:    Static,
				},
			},
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{Name: "container", Image: "container/image"}},
				},
			},
		}, Status: GameServerStatus{State: GameServerStateCreating}}
}
