/*
 * Copyright 2018 Google Inc. All Rights Reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package fleetautoscalers

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func TestComputeDesiredFleetSize(t *testing.T) {
	t.Parallel()

	fas, f := defaultFixtures()

	fas.Spec.Policy.Buffer.BufferSize = intstr.FromInt(20)
	fas.Spec.Policy.Buffer.MaxReplicas = 100
	f.Spec.Replicas = 50
	f.Status.Replicas = f.Spec.Replicas
	f.Status.AllocatedReplicas = 40
	f.Status.ReadyReplicas = 10

	replicas, limited, err := computeDesiredFleetSize(fas, f)
	assert.Nil(t, err)
	assert.Equal(t, replicas, int32(60))
	assert.Equal(t, limited, false)

	// test empty Policy Type
	f.Status.Replicas = 61
	fas.Spec.Policy.Type = ""
	replicas, limited, err = computeDesiredFleetSize(fas, f)
	assert.Nil(t, err)
	assert.Equal(t, replicas, int32(61))
	assert.Equal(t, limited, false)
}

func TestApplyBufferPolicy(t *testing.T) {
	t.Parallel()

	fas, f := defaultFixtures()
	b := fas.Spec.Policy.Buffer

	b.BufferSize = intstr.FromInt(20)
	b.MaxReplicas = 100
	f.Spec.Replicas = 50
	f.Status.Replicas = f.Spec.Replicas
	f.Status.AllocatedReplicas = 40
	f.Status.ReadyReplicas = 10

	replicas, limited, err := applyBufferPolicy(b, f)
	assert.Nil(t, err)
	assert.Equal(t, replicas, int32(60))
	assert.Equal(t, limited, false)

	b.MinReplicas = 65
	f.Spec.Replicas = 50
	f.Status.Replicas = f.Spec.Replicas
	f.Status.AllocatedReplicas = 40
	f.Status.ReadyReplicas = 10
	replicas, limited, err = applyBufferPolicy(b, f)
	assert.Nil(t, err)
	assert.Equal(t, replicas, int32(65))
	assert.Equal(t, limited, true)

	b.MinReplicas = 0
	b.MaxReplicas = 55
	f.Spec.Replicas = 50
	f.Status.Replicas = f.Spec.Replicas
	f.Status.AllocatedReplicas = 40
	f.Status.ReadyReplicas = 10
	replicas, limited, err = applyBufferPolicy(b, f)
	assert.Nil(t, err)
	assert.Equal(t, replicas, int32(55))
	assert.Equal(t, limited, true)

	b.BufferSize = intstr.FromString("20%")
	b.MinReplicas = 0
	b.MaxReplicas = 100
	f.Spec.Replicas = 50
	f.Status.Replicas = f.Spec.Replicas
	f.Status.AllocatedReplicas = 50
	f.Status.ReadyReplicas = 0
	replicas, limited, err = applyBufferPolicy(b, f)
	assert.Nil(t, err)
	assert.Equal(t, replicas, int32(63))
	assert.Equal(t, limited, false)

	b.BufferSize = intstr.FromString("10%")
	b.MinReplicas = 0
	b.MaxReplicas = 10
	f.Spec.Replicas = 1
	f.Status.Replicas = f.Spec.Replicas
	f.Status.AllocatedReplicas = 1
	f.Status.ReadyReplicas = 0
	replicas, limited, err = applyBufferPolicy(b, f)
	assert.Nil(t, err)
	assert.Equal(t, replicas, int32(2))
	assert.Equal(t, limited, false)
}
