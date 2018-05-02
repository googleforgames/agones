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

package fleets

import (
	stablev1alpha1 "agones.dev/agones/pkg/apis/stable/v1alpha1"
	listerv1alpha1 "agones.dev/agones/pkg/client/listers/stable/v1alpha1"
	"agones.dev/agones/pkg/gameserversets"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

// ListGameServerSetsByFleetOwner lists all the GameServerSets for a given
// Fleet
func ListGameServerSetsByFleetOwner(gameServerSetLister listerv1alpha1.GameServerSetLister, f *stablev1alpha1.Fleet) ([]*stablev1alpha1.GameServerSet, error) {
	list, err := gameServerSetLister.List(labels.SelectorFromSet(labels.Set{stablev1alpha1.FleetGameServerSetLabel: f.ObjectMeta.Name}))
	if err != nil {
		return list, errors.Wrapf(err, "error listing gameserversets for fleet %s", f.ObjectMeta.Name)
	}

	var result []*stablev1alpha1.GameServerSet
	for _, gsSet := range list {
		if metav1.IsControlledBy(gsSet, f) {
			result = append(result, gsSet)
		}
	}

	return result, nil
}

// ListGameServersByFleetOwner lists all GameServers that belong to a fleet through the
// GameServer -> GameServerSet -> Fleet owner chain
func ListGameServersByFleetOwner(gameServerLister listerv1alpha1.GameServerLister,
	gameServerSetLister listerv1alpha1.GameServerSetLister, fleet *stablev1alpha1.Fleet) ([]*stablev1alpha1.GameServer, error) {
	var result []*stablev1alpha1.GameServer

	gsSetList, err := ListGameServerSetsByFleetOwner(gameServerSetLister, fleet)
	if err != nil {
		return result, err
	}

	for _, gsSet := range gsSetList {
		gsList, err := gameserversets.ListGameServersByGameServerSetOwner(gameServerLister, gsSet)
		if err != nil {
			return result, err
		}
		result = append(result, gsList...)
	}

	return result, nil
}
