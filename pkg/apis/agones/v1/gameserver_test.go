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
	"time"

	"agones.dev/agones/pkg"
	"agones.dev/agones/pkg/apis"
	"agones.dev/agones/pkg/apis/agones"
	"agones.dev/agones/pkg/util/runtime"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation"
)

const (
	ipFixture = "127.1.1.1"
)

func TestStatus(t *testing.T) {
	name := "test-name"
	port := int32(7788)
	p := GameServerPort{Name: name, HostPort: port}

	res := p.Status()
	assert.Equal(t, name, res.Name)
	assert.Equal(t, port, res.Port)
}

func TestIsBeingDeleted(t *testing.T) {
	deletionTimestamp := metav1.Date(2009, 11, 17, 20, 34, 58, 651387237, time.UTC)
	var testCases = []struct {
		description string
		gs          *GameServer
		expected    bool
	}{
		{
			description: "ready gs, is not being deleted",
			gs: &GameServer{
				ObjectMeta: metav1.ObjectMeta{
					DeletionTimestamp: nil,
				},
				Status: GameServerStatus{State: GameServerStateReady},
			},
			expected: false,
		},
		{
			description: "DeletionTimestamp is set, gs is being deleted",
			gs: &GameServer{
				ObjectMeta: metav1.ObjectMeta{
					DeletionTimestamp: &deletionTimestamp,
				},
				Status: GameServerStatus{State: GameServerStateReady},
			},
			expected: true,
		},
		{
			description: "gs status is GameServerStateShutdown, gs is being deleted",
			gs: &GameServer{
				ObjectMeta: metav1.ObjectMeta{
					DeletionTimestamp: nil,
				},
				Status: GameServerStatus{State: GameServerStateShutdown},
			},
			expected: true,
		},
		{
			description: "gs status is GameServerStateShutdown and DeletionTimestamp is set, gs is being deleted",
			gs: &GameServer{
				ObjectMeta: metav1.ObjectMeta{
					DeletionTimestamp: &deletionTimestamp,
				},
				Status: GameServerStatus{State: GameServerStateShutdown},
			},
			expected: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			result := tc.gs.IsBeingDeleted()
			assert.Equal(t, tc.expected, result)
		})
	}
}

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

	ten := int64(10)

	type expected struct {
		protocol            corev1.Protocol
		state               GameServerState
		policy              PortPolicy
		health              Health
		scheduling          apis.SchedulingStrategy
		sdkServer           SdkServer
		alphaPlayerCapacity *int64
	}
	data := map[string]struct {
		gameServer   GameServer
		container    string
		featureFlags string
		expected     expected
	}{
		"set basic defaults on a very simple gameserver": {
			featureFlags: string(runtime.FeaturePlayerTracking) + "=true",
			gameServer: GameServer{
				Spec: GameServerSpec{
					Players: &PlayersSpec{InitialCapacity: 10},
					Ports:   []GameServerPort{{ContainerPort: 999}},
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
				alphaPlayerCapacity: &ten,
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

	runtime.FeatureTestMutex.Lock()
	defer runtime.FeatureTestMutex.Unlock()

	for name, test := range data {
		t.Run(name, func(t *testing.T) {
			err := runtime.ParseFeatures(test.featureFlags)
			assert.NoError(t, err)

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
			if test.expected.alphaPlayerCapacity != nil {
				assert.Equal(t, *test.expected.alphaPlayerCapacity, test.gameServer.Status.Players.Capacity)
			} else {
				assert.Nil(t, test.gameServer.Spec.Players)
				assert.Nil(t, test.gameServer.Status.Players)
			}
		})
	}
}

// nolint:dupl
func TestGameServerValidate(t *testing.T) {
	t.Parallel()

	longNameLen64 := strings.Repeat("f", validation.LabelValueMaxLength+1)

	var testCases = []struct {
		description    string
		gs             GameServer
		applyDefaults  bool
		isValid        bool
		causesExpected []metav1.StatusCause
	}{
		{
			description: "Valid game server",
			gs: GameServer{
				Spec: GameServerSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "testing", Image: "testing/image"}}}}},
			},
			applyDefaults: true,
			isValid:       true,
		},
		{
			description: "Invalid gs: container, containerPort, hostPort",
			gs: GameServer{
				Spec: GameServerSpec{
					Container: "",
					Ports: []GameServerPort{{
						Name:       "main",
						HostPort:   5001,
						PortPolicy: Dynamic,
					}, {
						Name:          "sidecar",
						HostPort:      5002,
						PortPolicy:    Static,
						ContainerPort: 5002,
					}},
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{Containers: []corev1.Container{
							{Name: "testing", Image: "testing/image"},
							{Name: "anothertest", Image: "testing/image"},
						}}}},
			},
			applyDefaults: false,
			isValid:       false,
			causesExpected: []metav1.StatusCause{
				{Type: metav1.CauseTypeFieldValueInvalid, Message: "Container is required when using multiple containers in the pod template", Field: "container"},
				{Type: metav1.CauseTypeFieldValueInvalid, Message: "Could not find a container named ", Field: "container"},
				{Type: metav1.CauseTypeFieldValueInvalid, Message: "ContainerPort must be defined for Dynamic and Static PortPolicies", Field: "main.containerPort"},
				{Type: metav1.CauseTypeFieldValueInvalid, Message: "HostPort cannot be specified with a Dynamic or Passthrough PortPolicy", Field: "main.hostPort"},
			},
		},
		{
			description: "DevAddressAnnotation: Invalid IP, no host port",
			gs: GameServer{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "dev-game",
					Namespace:   "default",
					Annotations: map[string]string{DevAddressAnnotation: "invalid-ip"},
				},
				Spec: GameServerSpec{
					Ports: []GameServerPort{{Name: "main", ContainerPort: 7777, PortPolicy: Static}},
				},
			},
			applyDefaults: false,
			isValid:       false,
			causesExpected: []metav1.StatusCause{
				{Type: metav1.CauseTypeFieldValueInvalid, Message: "Value 'invalid-ip' of annotation 'agones.dev/dev-address' must be a valid IP address", Field: "annotations.agones.dev/dev-address"},
				{Type: metav1.CauseTypeFieldValueRequired, Message: "HostPort is required if GameServer is annotated with 'agones.dev/dev-address'", Field: "main.hostPort"},
			},
		},
		{
			description: "Long gs name",
			gs: GameServer{
				ObjectMeta: metav1.ObjectMeta{
					Name: longNameLen64,
				},
				TypeMeta: metav1.TypeMeta{
					Kind: "test-kind",
				},
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
			},
			applyDefaults: false,
			isValid:       false,
			causesExpected: []metav1.StatusCause{
				{
					Type: metav1.CauseTypeFieldValueInvalid, Message: fmt.Sprintf("Length of test-kind '%s' name should be no more than 63 characters.", longNameLen64), Field: "Name",
				},
			},
		},
		{
			description: "Long gs GenerateName is not validated on agones side",
			gs: GameServer{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: longNameLen64,
				},
				TypeMeta: metav1.TypeMeta{
					Kind: "test-kind",
				},
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
			},
			applyDefaults:  false,
			isValid:        true,
			causesExpected: []metav1.StatusCause{},
		},
		{
			description: "Long label key is invalid",
			gs: GameServer{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: longNameLen64,
				},
				TypeMeta: metav1.TypeMeta{
					Kind: "test-kind",
				},
				Spec: GameServerSpec{
					Container: "my_image",
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{Name: "my_image", Image: "foo/my_image"},
							},
						},
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{longNameLen64: ""},
						},
					},
				},
			},
			applyDefaults: false,
			isValid:       false,
			causesExpected: []metav1.StatusCause{
				{
					// error message is coming from k8s.io/apimachinery/pkg/apis/meta/v1/validation
					Type: metav1.CauseTypeFieldValueInvalid, Message: fmt.Sprintf("labels: Invalid value: %q: name part must be no more than 63 characters", longNameLen64), Field: "labels",
				},
			},
		},
		{
			description: "Long label value is invalid",
			gs: GameServer{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "ok-name",
				},
				TypeMeta: metav1.TypeMeta{
					Kind: "test-kind",
				},
				Spec: GameServerSpec{
					Container: "my_image",
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{Name: "my_image", Image: "foo/my_image"},
							},
						},
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{"agones.dev/longValueKey": longNameLen64},
						},
					},
				},
			},
			applyDefaults: false,
			isValid:       false,
			causesExpected: []metav1.StatusCause{
				{
					// error message is coming from k8s.io/apimachinery/pkg/apis/meta/v1/validation
					Type: metav1.CauseTypeFieldValueInvalid, Message: fmt.Sprintf("labels: Invalid value: %q: must be no more than 63 characters", longNameLen64), Field: "labels",
				},
			},
		},
		{
			description: "Long annotation key is invalid",
			gs: GameServer{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "ok-name",
				},
				TypeMeta: metav1.TypeMeta{
					Kind: "test-kind",
				},
				Spec: GameServerSpec{
					Container: "my_image",
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{Name: "my_image", Image: "foo/my_image"},
							},
						},
						ObjectMeta: metav1.ObjectMeta{
							Annotations: map[string]string{longNameLen64: longNameLen64},
						},
					},
				},
			},
			applyDefaults: false,
			isValid:       false,
			causesExpected: []metav1.StatusCause{
				{
					// error message is coming from k8s.io/apimachinery/pkg/apis/meta/v1/validation
					Type: metav1.CauseTypeFieldValueInvalid, Message: fmt.Sprintf("annotations: Invalid value: %q: name part must be no more than 63 characters", longNameLen64), Field: "annotations",
				},
			},
		},
		{
			description: "Invalid character in annotation key",
			gs: GameServer{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "ok-name",
				},
				TypeMeta: metav1.TypeMeta{
					Kind: "test-kind",
				},
				Spec: GameServerSpec{
					Container: "my_image",
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{Name: "my_image", Image: "foo/my_image"},
							},
						},
						ObjectMeta: metav1.ObjectMeta{
							Annotations: map[string]string{"agones.dev/short±Name": longNameLen64},
						},
					},
				},
			},
			applyDefaults: false,
			isValid:       false,
			causesExpected: []metav1.StatusCause{
				{
					// error message is coming from k8s.io/apimachinery/pkg/apis/meta/v1/validation
					Type: metav1.CauseTypeFieldValueInvalid, Message: fmt.Sprintf("annotations: Invalid value: %q: name part must consist of alphanumeric characters, '-', '_' or '.', and must start and end with an alphanumeric character (e.g. 'MyName',  or 'my.name',  or '123-abc', regex used for validation is '([A-Za-z0-9][-A-Za-z0-9_.]*)?[A-Za-z0-9]')", "agones.dev/short±Name"), Field: "annotations",
				},
			},
		},
		{
			description: "Valid annotation key",
			gs: GameServer{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "ok-name",
				},
				TypeMeta: metav1.TypeMeta{
					Kind: "test-kind",
				},
				Spec: GameServerSpec{
					Container: "my_image",
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{Name: "my_image", Image: "foo/my_image"},
							},
						},
						ObjectMeta: metav1.ObjectMeta{
							Annotations: map[string]string{"agones.dev/shortName": longNameLen64},
						},
					},
				},
			},
			applyDefaults:  false,
			isValid:        true,
			causesExpected: []metav1.StatusCause{},
		},
		{
			description: "Check ContainerPort and HostPort with different policies",
			gs: GameServer{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "ok-name",
				},
				TypeMeta: metav1.TypeMeta{
					Kind: "test-kind",
				},
				Spec: GameServerSpec{
					Ports: []GameServerPort{
						{Name: "one", PortPolicy: Passthrough, ContainerPort: 1294},
						{PortPolicy: Passthrough, Name: "two", HostPort: 7890},
						{PortPolicy: Dynamic, Name: "three", HostPort: 7890, ContainerPort: 1294},
					},
					Container: "my_image",
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{Name: "my_image", Image: "foo/my_image"},
							},
						},
					},
				},
			},
			applyDefaults: true,
			isValid:       false,
			causesExpected: []metav1.StatusCause{
				{Type: metav1.CauseTypeFieldValueInvalid, Message: "ContainerPort cannot be specified with Passthrough PortPolicy", Field: "one.containerPort"},
				{Type: metav1.CauseTypeFieldValueInvalid, Message: "HostPort cannot be specified with a Dynamic or Passthrough PortPolicy", Field: "two.hostPort"},
				{Type: metav1.CauseTypeFieldValueInvalid, Message: "HostPort cannot be specified with a Dynamic or Passthrough PortPolicy", Field: "three.hostPort"},
			},
		},
		{
			description: "PortPolicy must be Static with HostPort specified",
			gs: GameServer{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "dev-game",
					Namespace:   "default",
					Annotations: map[string]string{DevAddressAnnotation: ipFixture},
				},
				Spec: GameServerSpec{
					Ports: []GameServerPort{
						{PortPolicy: Passthrough, Name: "main", HostPort: 7890, ContainerPort: 7777},
					},
					Container: "my_image",
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{Name: "my_image", Image: "foo/my_image"},
							},
						},
					},
				},
			},
			applyDefaults: true,
			isValid:       false,
			causesExpected: []metav1.StatusCause{
				{Type: metav1.CauseTypeFieldValueRequired, Message: "PortPolicy must be Static", Field: "main.portPolicy"},
			},
		},
		{
			description: "ContainerPort is less than zero",
			gs: GameServer{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "dev-game",
					Namespace: "default",
				},
				Spec: GameServerSpec{
					Ports: []GameServerPort{{Name: "main",
						ContainerPort: -4,
						PortPolicy:    Dynamic}},
					Container: "testing",
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{Containers: []corev1.Container{
							{Name: "testing", Image: "testing/image"},
						}}},
				},
			},
			applyDefaults: false,
			isValid:       false,
			causesExpected: []metav1.StatusCause{
				{Type: metav1.CauseTypeFieldValueInvalid, Message: "ContainerPort must be defined for Dynamic and Static PortPolicies", Field: "main.containerPort"},
			},
		},
		{
			description: "CPU Request > Limit",
			gs: GameServer{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "dev-game",
					Namespace: "default",
				},
				Spec: GameServerSpec{
					Ports: []GameServerPort{{Name: "main",
						ContainerPort: 7777,
						PortPolicy:    Dynamic,
					}},
					Container: "testing",
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{Containers: []corev1.Container{
							{
								Name:  "testing",
								Image: "testing/image",
								Resources: corev1.ResourceRequirements{
									Requests: corev1.ResourceList{
										corev1.ResourceCPU:    resource.MustParse("50m"),
										corev1.ResourceMemory: resource.MustParse("32Mi"),
									},
									Limits: corev1.ResourceList{
										corev1.ResourceCPU:    resource.MustParse("30m"),
										corev1.ResourceMemory: resource.MustParse("32Mi"),
									},
								},
							},
						}}},
				},
			},
			applyDefaults: false,
			isValid:       false,
			causesExpected: []metav1.StatusCause{
				{Type: metav1.CauseTypeFieldValueInvalid, Message: "Request must be less than or equal to cpu limit", Field: "container"},
			},
		},
		{
			description: "CPU negative request",
			gs: GameServer{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "dev-game",
					Namespace: "default",
				},
				Spec: GameServerSpec{
					Ports: []GameServerPort{{Name: "main",
						ContainerPort: 7777,
						PortPolicy:    Dynamic,
					}},
					Container: "testing",
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{Containers: []corev1.Container{
							{
								Name:  "testing",
								Image: "testing/image",
								Resources: corev1.ResourceRequirements{
									Requests: corev1.ResourceList{
										corev1.ResourceCPU:    resource.MustParse("-30m"),
										corev1.ResourceMemory: resource.MustParse("32Mi"),
									},
									Limits: corev1.ResourceList{
										corev1.ResourceCPU:    resource.MustParse("30m"),
										corev1.ResourceMemory: resource.MustParse("32Mi"),
									},
								},
							},
						}}},
				},
			},
			applyDefaults: false,
			isValid:       false,
			causesExpected: []metav1.StatusCause{
				{Type: metav1.CauseTypeFieldValueInvalid, Message: "Resource cpu request value must be non negative", Field: "container"},
			},
		},
		{
			description: "CPU negative limit",
			gs: GameServer{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "dev-game",
					Namespace: "default",
				},
				Spec: GameServerSpec{
					Ports: []GameServerPort{{Name: "main",
						ContainerPort: 7777,
						PortPolicy:    Dynamic,
					}},
					Container: "testing",
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{Containers: []corev1.Container{
							{
								Name:  "testing",
								Image: "testing/image",
								Resources: corev1.ResourceRequirements{
									Requests: corev1.ResourceList{
										corev1.ResourceCPU:    resource.MustParse("30m"),
										corev1.ResourceMemory: resource.MustParse("32Mi"),
									},
									Limits: corev1.ResourceList{
										corev1.ResourceCPU:    resource.MustParse("-30m"),
										corev1.ResourceMemory: resource.MustParse("32Mi"),
									},
								},
							},
						}}},
				},
			},
			applyDefaults: false,
			isValid:       false,
			causesExpected: []metav1.StatusCause{
				{Type: metav1.CauseTypeFieldValueInvalid, Message: "Request must be less than or equal to cpu limit", Field: "container"},
				{Type: metav1.CauseTypeFieldValueInvalid, Message: "Resource cpu limit value must be non negative", Field: "container"},
			},
		},
		{
			description: "Memory Request > Limit",
			gs: GameServer{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "dev-game",
					Namespace: "default",
				},
				Spec: GameServerSpec{
					Ports: []GameServerPort{{Name: "main",
						ContainerPort: 7777,
						PortPolicy:    Dynamic,
					}},
					Container: "testing",
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{Containers: []corev1.Container{
							{
								Name:  "testing",
								Image: "testing/image",
								Resources: corev1.ResourceRequirements{
									Requests: corev1.ResourceList{
										corev1.ResourceCPU:    resource.MustParse("30m"),
										corev1.ResourceMemory: resource.MustParse("55Mi"),
									},
									Limits: corev1.ResourceList{
										corev1.ResourceCPU:    resource.MustParse("30m"),
										corev1.ResourceMemory: resource.MustParse("32Mi"),
									},
								},
							},
						}}},
				},
			},
			applyDefaults: false,
			isValid:       false,
			causesExpected: []metav1.StatusCause{
				{Type: metav1.CauseTypeFieldValueInvalid, Message: "Request must be less than or equal to memory limit", Field: "container"},
			},
		},
		{
			description: "Memory negative request",
			gs: GameServer{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "dev-game",
					Namespace: "default",
				},
				Spec: GameServerSpec{
					Ports: []GameServerPort{{Name: "main",
						ContainerPort: 7777,
						PortPolicy:    Dynamic,
					}},
					Container: "testing",
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{Containers: []corev1.Container{
							{
								Name:  "testing",
								Image: "testing/image",
								Resources: corev1.ResourceRequirements{
									Requests: corev1.ResourceList{
										corev1.ResourceCPU:    resource.MustParse("30m"),
										corev1.ResourceMemory: resource.MustParse("-32Mi"),
									},
									Limits: corev1.ResourceList{
										corev1.ResourceCPU:    resource.MustParse("30m"),
										corev1.ResourceMemory: resource.MustParse("32Mi"),
									},
								},
							},
						}}},
				},
			},
			applyDefaults: false,
			isValid:       false,
			causesExpected: []metav1.StatusCause{
				{Type: metav1.CauseTypeFieldValueInvalid, Message: "Resource memory request value must be non negative", Field: "container"},
			},
		},
		{
			description: "Memory negative limit",
			gs: GameServer{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "dev-game",
					Namespace: "default",
				},
				Spec: GameServerSpec{
					Ports: []GameServerPort{{Name: "main",
						ContainerPort: 7777,
						PortPolicy:    Dynamic,
					}},
					Container: "testing",
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{Containers: []corev1.Container{
							{
								Name:  "testing",
								Image: "testing/image",
								Resources: corev1.ResourceRequirements{
									Requests: corev1.ResourceList{
										corev1.ResourceCPU:    resource.MustParse("30m"),
										corev1.ResourceMemory: resource.MustParse("32Mi"),
									},
									Limits: corev1.ResourceList{
										corev1.ResourceCPU:    resource.MustParse("30m"),
										corev1.ResourceMemory: resource.MustParse("-32Mi"),
									},
								},
							},
						}}},
				},
			},
			applyDefaults: false,
			isValid:       false,
			causesExpected: []metav1.StatusCause{
				{Type: metav1.CauseTypeFieldValueInvalid, Message: "Request must be less than or equal to memory limit", Field: "container"},
				{Type: metav1.CauseTypeFieldValueInvalid, Message: "Resource memory limit value must be non negative", Field: "container"},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			if tc.applyDefaults {
				tc.gs.ApplyDefaults()
			}

			causes, ok := tc.gs.Validate()

			assert.Equal(t, tc.isValid, ok)
			assert.ElementsMatch(t, tc.causesExpected, causes, "causes check")
		})
	}
}

func TestGameServerValidateFeatures(t *testing.T) {
	t.Parallel()
	runtime.FeatureTestMutex.Lock()
	defer runtime.FeatureTestMutex.Unlock()

	portContainerName := "another-container"

	var testCases = []struct {
		description    string
		feature        string
		gs             GameServer
		isValid        bool
		causesExpected []metav1.StatusCause
	}{
		{
			description: "Invalid container name, container was not found",
			gs: GameServer{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "dev-game",
					Namespace: "default",
				},
				Spec: GameServerSpec{
					Ports: []GameServerPort{
						{
							Name:          "main",
							ContainerPort: 7777,
							PortPolicy:    Dynamic,
							Container:     &portContainerName,
						},
					},
					Container: "testing",
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{Containers: []corev1.Container{
							{Name: "testing", Image: "testing/image"},
						}}},
				},
			},
			isValid: false,
			causesExpected: []metav1.StatusCause{
				{Type: metav1.CauseTypeFieldValueInvalid, Message: "Container must be empty or the name of a container in the pod template", Field: "main.container"},
			},
		},
		{
			description: "Multiple container ports, OK scenario",
			gs: GameServer{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "dev-game",
					Namespace: "default",
				},
				Spec: GameServerSpec{
					Ports: []GameServerPort{
						{
							Name:          "main",
							ContainerPort: 7777,
							PortPolicy:    Dynamic,
						},
					},
					Container: "testing",
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{Containers: []corev1.Container{
							{Name: "testing", Image: "testing/image"},
						}}},
				},
			},
			isValid:        true,
			causesExpected: []metav1.StatusCause{},
		},
		{
			description: "PlayerTracking is disabled, Players field specified",
			feature:     fmt.Sprintf("%s=false", runtime.FeaturePlayerTracking),
			gs: GameServer{
				Spec: GameServerSpec{
					Container: "testing",
					Players:   &PlayersSpec{InitialCapacity: 10},
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "testing", Image: "testing/image"}}}}},
			},
			isValid: false,
			causesExpected: []metav1.StatusCause{
				{Type: metav1.CauseTypeFieldValueNotSupported, Message: "Value cannot be set unless feature flag PlayerTracking is enabled", Field: "players"},
			},
		},
		{
			description: "PlayerTracking is enabled, Players field specified",
			feature:     fmt.Sprintf("%s=true", runtime.FeaturePlayerTracking),
			gs: GameServer{
				Spec: GameServerSpec{
					Container: "testing",
					Players:   &PlayersSpec{InitialCapacity: 10},
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "testing", Image: "testing/image"}}}}},
			},
			isValid:        true,
			causesExpected: []metav1.StatusCause{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			err := runtime.ParseFeatures(tc.feature)
			assert.NoError(t, err)

			causes, ok := tc.gs.Validate()

			assert.Equal(t, tc.isValid, ok)
			assert.ElementsMatch(t, tc.causesExpected, causes, "causes check")
		})
	}
}

func TestGameServerPodNoErrors(t *testing.T) {
	t.Parallel()
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
}

func TestGameServerPodContainerNotFoundErrReturned(t *testing.T) {
	t.Parallel()

	containerName1 := "Container1"
	fixture := &GameServer{ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "default", UID: "1234"},
		Spec: GameServerSpec{
			Container: "can-not-find-this-name",
			Ports: []GameServerPort{
				{
					Container:     &containerName1,
					ContainerPort: 7777,
					HostPort:      9999,
					PortPolicy:    Static,
				},
			},
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{Name: "Container2", Image: "container/image"}},
				},
			},
		}, Status: GameServerStatus{State: GameServerStateCreating}}

	_, err := fixture.Pod()
	if assert.NotNil(t, err, "Pod should return an error") {
		assert.Equal(t, "failed to find container named Container1 in pod spec", err.Error())
	}
}

func TestGameServerPodWithSidecarNoErrors(t *testing.T) {
	t.Parallel()
	fixture := defaultGameServer()
	fixture.ApplyDefaults()

	sidecar := corev1.Container{Name: "sidecar", Image: "container/sidecar"}
	fixture.Spec.Template.Spec.ServiceAccountName = "other-agones-sdk"
	pod, err := fixture.Pod(sidecar)
	assert.Nil(t, err, "Pod should not return an error")
	assert.Equal(t, fixture.ObjectMeta.Name, pod.ObjectMeta.Name)
	assert.Len(t, pod.Spec.Containers, 2, "Should have two containers")
	assert.Equal(t, "other-agones-sdk", pod.Spec.ServiceAccountName)
	assert.Equal(t, "sidecar", pod.Spec.Containers[0].Name)
	assert.Equal(t, "container", pod.Spec.Containers[1].Name)
	assert.True(t, metav1.IsControlledBy(pod, fixture))
}

func TestGameServerPodWithMultiplePortAllocations(t *testing.T) {
	fixture := defaultGameServer()
	containerName := "authContainer"
	fixture.Spec.Template.Spec.Containers = append(fixture.Spec.Template.Spec.Containers, corev1.Container{
		Name: containerName,
	})
	fixture.Spec.Ports = append(fixture.Spec.Ports, GameServerPort{
		Name:          "containerPort",
		PortPolicy:    Dynamic,
		Container:     &containerName,
		ContainerPort: 5000,
	})
	fixture.Spec.Container = fixture.Spec.Template.Spec.Containers[0].Name
	fixture.ApplyDefaults()

	pod, err := fixture.Pod()
	assert.NoError(t, err, "Pod should not return an error")
	assert.Equal(t, fixture.ObjectMeta.Name, pod.ObjectMeta.Name)
	assert.Equal(t, fixture.ObjectMeta.Namespace, pod.ObjectMeta.Namespace)
	assert.Equal(t, "gameserver", pod.ObjectMeta.Labels[agones.GroupName+"/role"])
	assert.Equal(t, fixture.ObjectMeta.Name, pod.ObjectMeta.Labels[GameServerPodLabel])
	assert.Equal(t, fixture.Spec.Container, pod.ObjectMeta.Annotations[GameServerContainerAnnotation])
	assert.Equal(t, fixture.Spec.Ports[0].HostPort, pod.Spec.Containers[0].Ports[0].HostPort)
	assert.Equal(t, fixture.Spec.Ports[0].ContainerPort, pod.Spec.Containers[0].Ports[0].ContainerPort)
	assert.Equal(t, *fixture.Spec.Ports[0].Container, pod.Spec.Containers[0].Name)
	assert.Equal(t, corev1.Protocol("UDP"), pod.Spec.Containers[0].Ports[0].Protocol)
	assert.True(t, metav1.IsControlledBy(pod, fixture))
	assert.Equal(t, fixture.Spec.Ports[1].HostPort, pod.Spec.Containers[1].Ports[0].HostPort)
	assert.Equal(t, fixture.Spec.Ports[1].ContainerPort, pod.Spec.Containers[1].Ports[0].ContainerPort)
	assert.Equal(t, *fixture.Spec.Ports[1].Container, pod.Spec.Containers[1].Name)
	assert.Equal(t, corev1.Protocol("UDP"), pod.Spec.Containers[1].Ports[0].Protocol)
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

		assert.Equal(t, "false", pod.ObjectMeta.Annotations[PodSafeToEvictAnnotation])
	})

	t.Run("distributed", func(t *testing.T) {
		gs := fixture.DeepCopy()
		gs.Spec.Scheduling = apis.Distributed
		pod := &corev1.Pod{}

		gs.podObjectMeta(pod)
		f(t, gs, pod)

		assert.Equal(t, "", pod.ObjectMeta.Annotations[PodSafeToEvictAnnotation])
	})
}

func TestGameServerPodAutoscalerAnnotations(t *testing.T) {
	testCases := []struct {
		description        string
		scheduling         apis.SchedulingStrategy
		setAnnotation      bool
		expectedAnnotation string
	}{
		{
			description:        "Packed",
			scheduling:         apis.Packed,
			expectedAnnotation: "false",
		},
		{
			description:        "Distributed",
			scheduling:         apis.Distributed,
			expectedAnnotation: "",
		},
		{
			description:        "Packed with autoscaler annotation",
			scheduling:         apis.Packed,
			setAnnotation:      true,
			expectedAnnotation: "true",
		},
		{
			description:        "Distributed with autoscaler annotation",
			scheduling:         apis.Distributed,
			setAnnotation:      true,
			expectedAnnotation: "true",
		},
	}

	fixture := &GameServer{
		ObjectMeta: metav1.ObjectMeta{Name: "logan"},
		Spec:       GameServerSpec{Container: "sheep"},
	}
	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			gs := fixture.DeepCopy()
			gs.Spec.Scheduling = tc.scheduling
			if tc.setAnnotation {
				gs.Spec.Template = corev1.PodTemplateSpec{ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{PodSafeToEvictAnnotation: "true"},
				}}
			}
			pod, err := gs.Pod()
			assert.Nil(t, err, "Pod should not return an error")
			assert.Equal(t, gs.ObjectMeta.Name, pod.ObjectMeta.Name)
			assert.Equal(t, gs.ObjectMeta.Namespace, pod.ObjectMeta.Namespace)
			assert.Equal(t, GameServerLabelRole, pod.ObjectMeta.Labels[RoleLabel])
			assert.Equal(t, "gameserver", pod.ObjectMeta.Labels[agones.GroupName+"/role"])
			assert.Equal(t, gs.ObjectMeta.Name, pod.ObjectMeta.Labels[GameServerPodLabel])
			assert.Equal(t, "sheep", pod.ObjectMeta.Annotations[GameServerContainerAnnotation])
			assert.True(t, metav1.IsControlledBy(pod, gs))
			assert.Equal(t, tc.expectedAnnotation, pod.ObjectMeta.Annotations[PodSafeToEvictAnnotation])
		})
	}
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

	err = gs.DisableServiceAccount(pod)
	assert.NoError(t, err)
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

func TestGameServerApplyToPodContainer(t *testing.T) {
	t.Parallel()
	type expected struct {
		err string
		tty bool
	}

	var testCases = []struct {
		description string
		gs          *GameServer
		expected    expected
	}{
		{
			description: "OK, no error",
			gs: &GameServer{
				Spec: GameServerSpec{
					Container: "mycontainer",
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{Name: "mycontainer", Image: "foo/mycontainer"},
								{Name: "notmycontainer", Image: "foo/notmycontainer"},
							},
						},
					},
				},
			},
			expected: expected{
				err: "",
				tty: true,
			},
		},
		{
			description: "container not found, error is returned",
			gs: &GameServer{
				Spec: GameServerSpec{
					Container: "mycontainer-WRONG-NAME",
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{Name: "mycontainer", Image: "foo/mycontainer"},
								{Name: "notmycontainer", Image: "foo/notmycontainer"},
							},
						},
					},
				},
			},
			expected: expected{
				err: "failed to find container named mycontainer-WRONG-NAME in pod spec",
				tty: false,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			pod := &corev1.Pod{Spec: *tc.gs.Spec.Template.Spec.DeepCopy()}
			result := tc.gs.ApplyToPodContainer(pod, tc.gs.Spec.Container, func(c corev1.Container) corev1.Container {
				//  easy thing to change and test for
				c.TTY = true
				return c
			})

			if tc.expected.err != "" && assert.NotNil(t, result) {
				assert.Equal(t, tc.expected.err, result.Error())
			}
			assert.Equal(t, tc.expected.tty, pod.Spec.Containers[0].TTY)
			assert.False(t, pod.Spec.Containers[1].TTY)
		})
	}
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
