// Copyright 2019 Google LLC All Rights Reserved.
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

package metrics

import (
	"context"

	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	"go.opencensus.io/stats"
	"go.opencensus.io/tag"
	"k8s.io/apimachinery/pkg/util/errors"
)

// GameServerCount  is the count of gameserver per current state and per fleet name
type GameServerCount map[agonesv1.GameServerState]map[string]int64

// increment adds the count of gameservers for a given fleetName and state
func (c GameServerCount) increment(fleetName string, state agonesv1.GameServerState) {
	fleets, ok := c[state]
	if !ok {
		fleets = map[string]int64{}
		c[state] = fleets
	}
	fleets[fleetName]++
}

// reset sets zero to the whole metrics set
func (c GameServerCount) reset() {
	for _, fleets := range c {
		for fleet := range fleets {
			fleets[fleet] = 0
		}
	}
}

// record counts the list of gameserver per status and fleet name and record it to OpenCensus
func (c GameServerCount) record(gameservers []*agonesv1.GameServer) error {
	// Currently there is no way to remove a metric so we have to reset our values to zero
	// so that statuses that have no count anymore are zeroed.
	// Otherwise OpenCensus will write the last value recorded to the prom endpoint.
	// TL;DR we can't remove a gauge
	c.reset()

	var namespace string
	if len(gameservers) != 0 {
		// counts gameserver per state and fleet
		for _, g := range gameservers {
			c.increment(g.Labels[agonesv1.FleetNameLabel], g.Status.State)
		}
		namespace = gameservers[0].GetNamespace()
	}

	errs := []error{}
	for state, fleets := range c {
		for fleet, count := range fleets {
			if fleet == "" {
				fleet = "none"
			}
			if err := stats.RecordWithTags(context.Background(), []tag.Mutator{tag.Upsert(keyType, string(state)),
				tag.Upsert(keyFleetName, fleet), tag.Upsert(keyNamespace, namespace)}, gameServerCountStats.M(count)); err != nil {
				errs = append(errs, err)
			}
		}
	}
	return errors.NewAggregate(errs)
}
