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
	stablev1alpha1 "agones.dev/agones/pkg/apis/stable/v1alpha1"
	listerv1alpha1 "agones.dev/agones/pkg/client/listers/stable/v1alpha1"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

// ListGameServersByGameServerSetOwner lists the GameServers for a given GameServerSet
func ListGameServersByGameServerSetOwner(gameServerLister listerv1alpha1.GameServerLister,
	gsSet *stablev1alpha1.GameServerSet) ([]*stablev1alpha1.GameServer, error) {
	list, err := gameServerLister.List(labels.SelectorFromSet(labels.Set{stablev1alpha1.GameServerSetGameServerLabel: gsSet.ObjectMeta.Name}))
	if err != nil {
		return list, errors.Wrapf(err, "error listing gameservers for gameserverset %s", gsSet.ObjectMeta.Name)
	}

	var result []*stablev1alpha1.GameServer
	for _, gs := range list {
		if metav1.IsControlledBy(gs, gsSet) {
			result = append(result, gs)
		}
	}

	return result, nil
}
