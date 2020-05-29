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

package sdkserver

import (
	"testing"

	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	"agones.dev/agones/pkg/sdk"
	"agones.dev/agones/pkg/util/runtime"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestConvert(t *testing.T) {
	t.Parallel()

	runtime.FeatureTestMutex.Lock()
	defer runtime.FeatureTestMutex.Unlock()

	fixture := &agonesv1.GameServer{
		ObjectMeta: metav1.ObjectMeta{
			CreationTimestamp: metav1.Now(),
			Namespace:         "default",
			Name:              "test",
			Labels:            map[string]string{"foo": "bar"},
			Annotations:       map[string]string{"stuff": "things"},
			UID:               "1234",
		},
		Spec: agonesv1.GameServerSpec{
			Health: agonesv1.Health{
				Disabled:            false,
				InitialDelaySeconds: 10,
				FailureThreshold:    15,
				PeriodSeconds:       20,
			},
		},
		Status: agonesv1.GameServerStatus{
			NodeName: "george",
			Address:  "127.0.0.1",
			State:    "Ready",
			Ports: []agonesv1.GameServerStatusPort{
				{Name: "default", Port: 12345},
				{Name: "beacon", Port: 123123},
			},
		},
	}

	eq := func(t *testing.T, fixture *agonesv1.GameServer, sdkGs *sdk.GameServer) {
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

	t.Run(string(runtime.FeaturePlayerTracking)+" disabled", func(t *testing.T) {
		assert.NoError(t, runtime.ParseFeatures(""))

		gs := fixture.DeepCopy()

		sdkGs := convert(gs)
		eq(t, fixture, sdkGs)
		assert.Zero(t, sdkGs.ObjectMeta.DeletionTimestamp)
		assert.Nil(t, sdkGs.Status.Players)
	})

	t.Run(string(runtime.FeaturePlayerTracking)+" enabled", func(t *testing.T) {
		assert.NoError(t, runtime.ParseFeatures(string(runtime.FeaturePlayerTracking)+"=true"))

		gs := fixture.DeepCopy()
		gs.Status.Players = &agonesv1.PlayerStatus{Capacity: 10, Count: 5, IDs: []string{"one", "two"}}

		sdkGs := convert(gs)
		eq(t, fixture, sdkGs)
		assert.Zero(t, sdkGs.ObjectMeta.DeletionTimestamp)
		assert.Equal(t, gs.Status.Players.Capacity, sdkGs.Status.Players.Capacity)
		assert.Equal(t, gs.Status.Players.Count, sdkGs.Status.Players.Count)
		assert.Equal(t, gs.Status.Players.IDs, sdkGs.Status.Players.Ids)
	})

	t.Run("DeletionTimestamp", func(t *testing.T) {
		gs := fixture.DeepCopy()

		now := metav1.Now()
		gs.DeletionTimestamp = &now
		sdkGs := convert(gs)
		eq(t, gs, sdkGs)
		assert.Equal(t, gs.ObjectMeta.DeletionTimestamp.Unix(), sdkGs.ObjectMeta.DeletionTimestamp)
	})
}
