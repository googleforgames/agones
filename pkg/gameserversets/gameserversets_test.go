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

package gameserversets

import (
	"sort"
	"testing"

	"agones.dev/agones/pkg/apis/stable/v1alpha1"
	agtesting "agones.dev/agones/pkg/testing"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8stesting "k8s.io/client-go/testing"
)

func TestListGameServersByGameServerSetOwner(t *testing.T) {
	t.Parallel()

	gsSet := &v1alpha1.GameServerSet{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "test", UID: "1234"},
		Spec: v1alpha1.GameServerSetSpec{
			Replicas: 10,
			Template: v1alpha1.GameServerTemplateSpec{},
		},
	}

	gs1 := gsSet.GameServer()
	gs1.ObjectMeta.Name = "test-1"
	gs2 := gsSet.GameServer()
	assert.True(t, metav1.IsControlledBy(gs2, gsSet))

	gs2.ObjectMeta.Name = "test-2"
	gs3 := v1alpha1.GameServer{ObjectMeta: metav1.ObjectMeta{Name: "not-included"}}
	gs4 := gsSet.GameServer()
	gs4.ObjectMeta.OwnerReferences = nil

	m := agtesting.NewMocks()
	m.AgonesClient.AddReactor("list", "gameservers", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, &v1alpha1.GameServerList{Items: []v1alpha1.GameServer{*gs1, *gs2, gs3, *gs4}}, nil
	})

	gameServers := m.AgonesInformerFactory.Stable().V1alpha1().GameServers()
	_, cancel := agtesting.StartInformers(m, gameServers.Informer().HasSynced)
	defer cancel()

	list, err := ListGameServersByGameServerSetOwner(gameServers.Lister(), gsSet)
	assert.Nil(t, err)

	// sort of stable ordering
	sort.SliceStable(list, func(i, j int) bool {
		return list[i].ObjectMeta.Name < list[j].ObjectMeta.Name
	})
	assert.Equal(t, []*v1alpha1.GameServer{gs1, gs2}, list)
}
