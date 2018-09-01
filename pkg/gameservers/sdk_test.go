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

package gameservers

import (
	"testing"

	"agones.dev/agones/pkg/apis/stable/v1alpha1"
	"agones.dev/agones/pkg/sdk"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestConvert(t *testing.T) {
	t.Parallel()

	fixture := &v1alpha1.GameServer{
		ObjectMeta: v1.ObjectMeta{
			CreationTimestamp: v1.Now(),
			Namespace:         "default",
			Name:              "test",
			Labels:            map[string]string{"foo": "bar"},
			Annotations:       map[string]string{"stuff": "things"},
			UID:               "1234",
		},
		Spec: v1alpha1.GameServerSpec{
			Health: v1alpha1.Health{
				Disabled:            false,
				InitialDelaySeconds: 10,
				FailureThreshold:    15,
				PeriodSeconds:       20,
			},
		},
		Status: v1alpha1.GameServerStatus{
			NodeName: "george",
			Address:  "127.0.0.1",
			State:    "Ready",
			Ports: []v1alpha1.GameServerStatusPort{
				{Name: "default", Port: 12345},
				{Name: "beacon", Port: 123123},
			},
		},
	}

	eq := func(t *testing.T, fixture *v1alpha1.GameServer, sdkGs *sdk.GameServer) {
		assert.Equal(t, fixture.ObjectMeta.Name, sdkGs.ObjectMeta.Name)
		assert.Equal(t, fixture.ObjectMeta.Namespace, sdkGs.ObjectMeta.Namespace)
		assert.Equal(t, fixture.ObjectMeta.CreationTimestamp.Unix(), sdkGs.ObjectMeta.CreationTimestamp)
		assert.Equal(t, string(fixture.ObjectMeta.UID), sdkGs.ObjectMeta.Uid)
		assert.Equal(t, fixture.ObjectMeta.Labels, sdkGs.ObjectMeta.Labels)
		assert.Equal(t, fixture.ObjectMeta.Annotations, sdkGs.ObjectMeta.Annotations)
		assert.Equal(t, fixture.Spec.Health.Disabled, sdkGs.Spec.Health.Disabled)
		assert.Equal(t, fixture.Spec.Health.InitialDelaySeconds, sdkGs.Spec.Health.InitialDelaySeconds)
		assert.Equal(t, fixture.Spec.Health.FailureThreshold, sdkGs.Spec.Health.FailureThreshold)
		assert.Equal(t, fixture.Spec.Health.PeriodSeconds, sdkGs.Spec.Health.PeriodSeconds)
		assert.Equal(t, fixture.Status.Address, sdkGs.Status.Address)
		assert.Equal(t, string(fixture.Status.State), sdkGs.Status.State)
		assert.Len(t, sdkGs.Status.Ports, len(fixture.Status.Ports))
		for i, fp := range fixture.Status.Ports {
			p := sdkGs.Status.Ports[i]
			assert.Equal(t, fp.Name, p.Name)
			assert.Equal(t, fp.Port, p.Port)
		}
	}

	sdkGs := convert(fixture)
	eq(t, fixture, sdkGs)
	assert.Zero(t, sdkGs.ObjectMeta.DeletionTimestamp)

	now := v1.Now()
	fixture.DeletionTimestamp = &now
	sdkGs = convert(fixture)
	eq(t, fixture, sdkGs)
	assert.Equal(t, fixture.ObjectMeta.DeletionTimestamp.Unix(), sdkGs.ObjectMeta.DeletionTimestamp)
}
