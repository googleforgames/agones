// Copyright 2019 Google Inc. All Rights Reserved.
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

package gameserverallocations

import (
	"testing"

	"agones.dev/agones/pkg/apis/stable/v1alpha1"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGameServerCacheEntry(t *testing.T) {
	gs1 := &v1alpha1.GameServer{ObjectMeta: metav1.ObjectMeta{Name: "gs1"}}
	gs2 := &v1alpha1.GameServer{ObjectMeta: metav1.ObjectMeta{Name: "gs2"}}
	gs3 := &v1alpha1.GameServer{ObjectMeta: metav1.ObjectMeta{Name: "gs3"}}

	cache := gameServerCacheEntry{}

	gs, ok := cache.Load("gs1")
	assert.Nil(t, gs)
	assert.False(t, ok)

	cache.Store("gs1", gs1)
	gs, ok = cache.Load("gs1")

	assert.Equal(t, gs, gs1)
	assert.True(t, ok)

	cache.Store("gs2", gs2)
	cache.Store("gs3", gs3)

	count := 0
	cache.Range(func(key string, gs *v1alpha1.GameServer) bool {
		if count++; count == 2 {
			return false
		}

		return true
	})

	assert.Equal(t, 2, count, "Should only process one item")

	cache.Delete("gs1")
	gs, ok = cache.Load("gs1")
	assert.Nil(t, gs)
	assert.False(t, ok)
}
