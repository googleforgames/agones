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

package v1alpha1

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func TestFleetAutoscalerValidateUpdate(t *testing.T) {
	t.Parallel()

	t.Run("bad buffer size", func(t *testing.T) {
		fas := defaultFixture()
		fas.Spec.Policy.Buffer.BufferSize = intstr.FromInt(0)
		causes := fas.Validate(nil)

		assert.Len(t, causes, 1)
		assert.Equal(t, "bufferSize", causes[0].Field)
	})

	t.Run("bad min replicas", func(t *testing.T) {

		fas := defaultFixture()
		fas.Spec.Policy.Buffer.MinReplicas = 2

		causes := fas.Validate(nil)

		assert.Len(t, causes, 1)
		assert.Equal(t, "minReplicas", causes[0].Field)
	})

	t.Run("bad max replicas", func(t *testing.T) {
		fas := defaultFixture()
		fas.Spec.Policy.Buffer.MaxReplicas = 2
		causes := fas.Validate(nil)

		assert.Len(t, causes, 1)
		assert.Equal(t, "maxReplicas", causes[0].Field)
	})

	t.Run("minReplicas > maxReplicas", func(t *testing.T) {
		fas := defaultFixture()
		fas.Spec.Policy.Buffer.MinReplicas = 20
		causes := fas.Validate(nil)

		assert.Len(t, causes, 1)
		assert.Equal(t, "minReplicas", causes[0].Field)
	})

	t.Run("bufferSize good percent", func(t *testing.T) {
		fas := defaultFixture()
		fas.Spec.Policy.Buffer.BufferSize = intstr.FromString("20%")
		causes := fas.Validate(nil)

		assert.Len(t, causes, 0)
	})

	t.Run("bufferSize bad percent", func(t *testing.T) {
		fas := defaultFixture()

		fasCopy := fas.DeepCopy()
		fasCopy.Spec.Policy.Buffer.BufferSize = intstr.FromString("120%")
		causes := fasCopy.Validate(nil)
		assert.Len(t, causes, 1)
		assert.Equal(t, "bufferSize", causes[0].Field)

		fasCopy = fas.DeepCopy()
		fasCopy.Spec.Policy.Buffer.BufferSize = intstr.FromString("0%")
		causes = fasCopy.Validate(nil)
		assert.Len(t, causes, 1)
		assert.Equal(t, "bufferSize", causes[0].Field)

		fasCopy = fas.DeepCopy()
		fasCopy.Spec.Policy.Buffer.BufferSize = intstr.FromString("-10%")
		causes = fasCopy.Validate(nil)
		assert.Len(t, causes, 1)
		assert.Equal(t, "bufferSize", causes[0].Field)
		fasCopy = fas.DeepCopy()

		fasCopy.Spec.Policy.Buffer.BufferSize = intstr.FromString("notgood")
		causes = fasCopy.Validate(nil)
		assert.Len(t, causes, 1)
		assert.Equal(t, "bufferSize", causes[0].Field)
	})
}

func defaultFixture() *FleetAutoscaler {
	return &FleetAutoscaler{
		ObjectMeta: metav1.ObjectMeta{Name: "test"},
		Spec: FleetAutoscalerSpec{
			FleetName: "testing",
			Policy: FleetAutoscalerPolicy{
				Type: BufferPolicyType,
				Buffer: &BufferPolicy{
					BufferSize:  intstr.FromInt(5),
					MaxReplicas: 10,
				},
			},
		},
	}
}
