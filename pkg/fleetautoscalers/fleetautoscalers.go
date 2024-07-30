/*
 * Copyright 2018 Google LLC All Rights Reserved.
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
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/robfig/cron/v3"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/uuid"

	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	autoscalingv1 "agones.dev/agones/pkg/apis/autoscaling/v1"
	listeragonesv1 "agones.dev/agones/pkg/client/listers/agones/v1"
	"agones.dev/agones/pkg/fleets"
	"agones.dev/agones/pkg/gameservers"
	gssets "agones.dev/agones/pkg/gameserversets"
	"agones.dev/agones/pkg/util/runtime"
)

var tlsConfig = &tls.Config{}
var client = http.Client{
	Timeout: 15 * time.Second,
	Transport: &http.Transport{
		TLSClientConfig: tlsConfig,
	},
}

// computeDesiredFleetSize computes the new desired size of the given fleet
func computeDesiredFleetSize(pol autoscalingv1.FleetAutoscalerPolicy, f *agonesv1.Fleet,
	gameServerLister listeragonesv1.GameServerLister, nodeCounts map[string]gameservers.NodeCount) (int32, bool, error) {
	switch pol.Type {
	case autoscalingv1.BufferPolicyType:
		return applyBufferPolicy(pol.Buffer, f)
	case autoscalingv1.WebhookPolicyType:
		return applyWebhookPolicy(pol.Webhook, f)
	case autoscalingv1.CounterPolicyType:
		return applyCounterOrListPolicy(pol.Counter, nil, f, gameServerLister, nodeCounts)
	case autoscalingv1.ListPolicyType:
		return applyCounterOrListPolicy(nil, pol.List, f, gameServerLister, nodeCounts)
	case autoscalingv1.SchedulePolicyType:
		return applySchedulePolicy(pol.Schedule, f, gameServerLister, nodeCounts)
	case autoscalingv1.ChainPolicyType:
		return applyChainPolicy(pol.Chain, f, gameServerLister, nodeCounts)
	}

	return 0, false, errors.New("wrong policy type, should be one of: Buffer, Webhook, Counter, List")
}

// buildURLFromWebhookPolicy - build URL for Webhook and set CARoot for client Transport
func buildURLFromWebhookPolicy(w *autoscalingv1.WebhookPolicy) (u *url.URL, err error) {
	if w.URL != nil && w.Service != nil {
		return nil, errors.New("service and URL cannot be used simultaneously")
	}

	scheme := "http"
	if w.CABundle != nil {
		scheme = "https"

		if err := setCABundle(w.CABundle); err != nil {
			return nil, err
		}
	}

	if w.URL != nil {
		if *w.URL == "" {
			return nil, errors.New("URL was not provided")
		}

		return url.ParseRequestURI(*w.URL)
	}

	if w.Service == nil {
		return nil, errors.New("service was not provided, either URL or Service must be provided")
	}

	if w.Service.Name == "" {
		return nil, errors.New("service name was not provided")
	}

	if w.Service.Path == nil {
		empty := ""
		w.Service.Path = &empty
	}

	if w.Service.Namespace == "" {
		w.Service.Namespace = "default"
	}

	return createURL(scheme, w.Service.Name, w.Service.Namespace, *w.Service.Path, w.Service.Port), nil
}

// moved to a separate method to cover it with unit tests and check that URL corresponds to a proper pattern
func createURL(scheme, name, namespace, path string, port *int32) *url.URL {
	var hostPort int32 = 8000
	if port != nil {
		hostPort = *port
	}

	return &url.URL{
		Scheme: scheme,
		Host:   fmt.Sprintf("%s.%s.svc:%d", name, namespace, hostPort),
		Path:   path,
	}
}

func setCABundle(caBundle []byte) error {
	// We can have multiple fleetautoscalers with different CABundles defined,
	// so we switch client.Transport before each POST request
	rootCAs := x509.NewCertPool()
	if ok := rootCAs.AppendCertsFromPEM(caBundle); !ok {
		return errors.New("no certs were appended from caBundle")
	}
	tlsConfig.RootCAs = rootCAs
	return nil
}

func applyWebhookPolicy(w *autoscalingv1.WebhookPolicy, f *agonesv1.Fleet) (replicas int32, limited bool, err error) {
	if w == nil {
		return 0, false, errors.New("webhookPolicy parameter must not be nil")
	}

	if f == nil {
		return 0, false, errors.New("fleet parameter must not be nil")
	}

	u, err := buildURLFromWebhookPolicy(w)
	if err != nil {
		return 0, false, err
	}

	faReq := autoscalingv1.FleetAutoscaleReview{
		Request: &autoscalingv1.FleetAutoscaleRequest{
			UID:       uuid.NewUUID(),
			Name:      f.Name,
			Namespace: f.Namespace,
			Status:    f.Status,
		},
		Response: nil,
	}

	b, err := json.Marshal(faReq)
	if err != nil {
		return 0, false, err
	}

	res, err := client.Post(
		u.String(),
		"application/json",
		strings.NewReader(string(b)),
	)
	if err != nil {
		return 0, false, err
	}
	defer func() {
		if cerr := res.Body.Close(); cerr != nil {
			if err != nil {
				err = errors.Wrap(err, cerr.Error())
			} else {
				err = cerr
			}
		}
	}()

	if res.StatusCode != http.StatusOK {
		return 0, false, fmt.Errorf("bad status code %d from the server: %s", res.StatusCode, u.String())
	}
	result, err := io.ReadAll(res.Body)
	if err != nil {
		return 0, false, err
	}

	var faResp autoscalingv1.FleetAutoscaleReview
	err = json.Unmarshal(result, &faResp)
	if err != nil {
		return 0, false, err
	}

	if faResp.Response.Scale {
		return faResp.Response.Replicas, false, nil
	}
	return f.Status.Replicas, false, nil
}

func applyBufferPolicy(b *autoscalingv1.BufferPolicy, f *agonesv1.Fleet) (int32, bool, error) {
	var replicas int32

	if b.BufferSize.Type == intstr.Int {
		replicas = f.Status.AllocatedReplicas + int32(b.BufferSize.IntValue())
	} else {
		// the percentage value is a little more complex, as we can't apply
		// the desired percentage to any current value, but to the future one
		// Example: we have 8 allocated replicas, 10 total replicas and bufferSize set to 30%
		// 30% means that we must have 30% ready instances in the fleet
		// Right now there are 20%, so we must increase the fleet until we reach 30%
		// To compute the new size, we start from the other end: if ready must be 30%
		// it means that allocated must be 70% and adjust the fleet size to make that true.
		bufferPercent, err := intstr.GetValueFromIntOrPercent(&b.BufferSize, 100, true)
		if err != nil {
			return 0, false, err
		}
		// use Math.Ceil to round the result up
		replicas = int32(math.Ceil(float64(f.Status.AllocatedReplicas*100) / float64(100-bufferPercent)))
	}

	scalingInLimited := false
	scalingOutLimited := false

	if replicas < b.MinReplicas {
		replicas = b.MinReplicas
		scalingInLimited = true
	}
	if replicas > b.MaxReplicas {
		replicas = b.MaxReplicas
		scalingOutLimited = true
	}

	return replicas, scalingInLimited || scalingOutLimited, nil
}

func applyCounterOrListPolicy(c *autoscalingv1.CounterPolicy, l *autoscalingv1.ListPolicy,
	f *agonesv1.Fleet, gameServerLister listeragonesv1.GameServerLister,
	nodeCounts map[string]gameservers.NodeCount) (int32, bool, error) {

	if !runtime.FeatureEnabled(runtime.FeatureCountsAndLists) {
		return 0, false, errors.Errorf("cannot apply CounterPolicy unless feature flag %s is enabled", runtime.FeatureCountsAndLists)
	}

	var isCounter bool          // True if a CounterPolicy False if a ListPolicy
	var key string              // The specified Counter or List
	var count int64             // The Count or number of Values in the template Game Server
	var capacity int64          // The Capacity in the template Game Server
	var aggCount int64          // The Aggregate Count of the specified Counter or List of all GameServers across the GameServerSet in the Fleet
	var aggCapacity int64       // The Aggregate Capacity of the specified Counter or List of all GameServers across the GameServerSet in the Fleet
	var aggAllocatedCount int64 // The Aggregate Count of the specified Counter or List of GameServers in an Allocated state across the GameServerSet in the Fleet
	var minCapacity int64       // The Minimum Aggregate Capacity
	var maxCapacity int64       // The Maximum Aggregate Capacity
	var bufferSize intstr.IntOrString

	if c != nil {
		isCounter = true
		counter, ok := f.Spec.Template.Spec.Counters[c.Key]
		if !ok {
			return 0, false, errors.Errorf("cannot apply CounterPolicy as Counter key %s does not exist in the Fleet Spec", c.Key)
		}

		aggCounter, ok := f.Status.Counters[c.Key]
		if !ok {
			return 0, false, errors.Errorf("cannot apply CounterPolicy as Counter key %s does not exist in the Fleet Status", c.Key)
		}

		key = c.Key
		count = counter.Count
		capacity = counter.Capacity
		aggCount = aggCounter.Count
		aggCapacity = aggCounter.Capacity
		aggAllocatedCount = aggCounter.AllocatedCount
		minCapacity = c.MinCapacity
		maxCapacity = c.MaxCapacity
		bufferSize = c.BufferSize

	} else {
		isCounter = false
		list, ok := f.Spec.Template.Spec.Lists[l.Key]
		if !ok {
			return 0, false, errors.Errorf("cannot apply ListPolicy as List key %s does not exist in the Fleet Spec", l.Key)
		}

		aggList, ok := f.Status.Lists[l.Key]
		if !ok {
			return 0, false, errors.Errorf("cannot apply ListPolicy as List key %s does not exist in the Fleet Status", l.Key)
		}

		key = l.Key
		count = int64(len(list.Values))
		capacity = list.Capacity
		aggCount = aggList.Count
		aggCapacity = aggList.Capacity
		aggAllocatedCount = aggList.AllocatedCount
		minCapacity = l.MinCapacity
		maxCapacity = l.MaxCapacity
		bufferSize = l.BufferSize
	}

	// Checks if we've limited by TOTAL capacity
	limited, scale := isLimited(aggCapacity, minCapacity, maxCapacity)

	// Total current number of Replicas
	replicas := f.Status.Replicas

	// The buffer is the desired available capacity
	var buffer int64

	switch {
	// Desired replicas based on BufferSize specified as an absolute value (i.e. 5)
	case bufferSize.Type == intstr.Int:
		buffer = int64(bufferSize.IntValue())
	// Desired replicas based on BufferSize specified as a percent (i.e. 5%)
	case bufferSize.Type == intstr.String:
		bufferPercent, err := intstr.GetValueFromIntOrPercent(&bufferSize, 100, isCounter)
		if err != nil {
			return 0, false, err
		}
		// If the Aggregated Allocated Counts is 0 then desired capacity gets calculated as 0. If the
		// capacity of 1 replica is equal to or greater than minimum capacity we can exit early.
		if aggAllocatedCount <= 0 && capacity >= minCapacity {
			return 1, true, nil
		}

		// The desired TOTAL capacity based on the Aggregated Allocated Counts (see applyBufferPolicy for explanation)
		desiredCapacity := int64(math.Ceil(float64(aggAllocatedCount*100) / float64(100-bufferPercent)))
		// Convert into a desired AVAILABLE capacity aka the buffer
		buffer = desiredCapacity - aggAllocatedCount
	}

	// Current available capacity across the TOTAL fleet
	switch availableCapacity := aggCapacity - aggCount; {
	case availableCapacity == buffer:
		if limited {
			return scaleLimited(scale, f, gameServerLister, nodeCounts, key, isCounter, replicas,
				capacity, aggCapacity, minCapacity, maxCapacity)
		}
		return replicas, false, nil
	case availableCapacity < buffer: // Scale Up
		if limited && scale == -1 { // Case where we want to scale up but we're already limited by MaxCapacity
			return scaleLimited(scale, f, gameServerLister, nodeCounts, key, isCounter, replicas,
				capacity, aggCapacity, minCapacity, maxCapacity)
		}
		return scaleUp(replicas, capacity, count, aggCapacity, availableCapacity, maxCapacity,
			minCapacity, buffer)
	case availableCapacity > buffer: // Scale Down
		if limited && scale == 1 { // Case where we want to scale down but we're already limited by MinCapacity
			return scaleLimited(scale, f, gameServerLister, nodeCounts, key, isCounter, replicas,
				capacity, aggCapacity, minCapacity, maxCapacity)
		}
		return scaleDown(f, gameServerLister, nodeCounts, key, isCounter, replicas, aggCount,
			aggCapacity, minCapacity, buffer)
	}

	if isCounter {
		return 0, false, errors.Errorf("unable to apply CounterPolicy %v", c)
	}
	return 0, false, errors.Errorf("unable to apply ListPolicy %v", l)
}

func applySchedulePolicy(s *autoscalingv1.SchedulePolicy, f *agonesv1.Fleet, gameServerLister listeragonesv1.GameServerLister, nodeCounts map[string]gameservers.NodeCount) (int32, bool, error) {
	// Ensure the scheduled autoscaler feature gate is enabled
	if !runtime.FeatureEnabled(runtime.FeatureScheduledAutoscaler) {
		return 0, false, errors.Errorf("cannot apply SchedulePolicy unless feature flag %s is enabled", runtime.FeatureScheduledAutoscaler)
	}

	if isScheduleActive(s) {
		return computeDesiredFleetSize(s.Policy, f, gameServerLister, nodeCounts)
	}

	return f.Status.Replicas, false, nil
}

func applyChainPolicy(c autoscalingv1.ChainPolicy, f *agonesv1.Fleet, gameServerLister listeragonesv1.GameServerLister, nodeCounts map[string]gameservers.NodeCount) (int32, bool, error) {
	// Ensure the scheduled autoscaler feature gate is enabled
	if !runtime.FeatureEnabled(runtime.FeatureScheduledAutoscaler) {
		return 0, false, errors.Errorf("cannot apply ChainPolicy unless feature flag %s is enabled", runtime.FeatureScheduledAutoscaler)
	}

	// Loop over all entries in the chain
	for _, entry := range c {
		switch entry.Type {
		case autoscalingv1.SchedulePolicyType:
			schedRep, schedLim, schedErr := applySchedulePolicy(entry.Schedule, f, gameServerLister, nodeCounts)
			// If the schedule is active and no error was returned from the policy, then return the replicas, limited and error
			if isScheduleActive(entry.Schedule) && schedErr == nil {
				return schedRep, schedLim, nil
			}
		case autoscalingv1.WebhookPolicyType:
			webhookRep, webhookLim, webhookErr := applyWebhookPolicy(entry.Webhook, f)
			if webhookErr == nil {
				return webhookRep, webhookLim, nil
			}
		default:
			return computeDesiredFleetSize(entry.FleetAutoscalerPolicy, f, gameServerLister, nodeCounts)
		}
	}

	return f.Status.Replicas, false, nil
}

// isScheduleActive checks if a chain entry's is active and returns a boolean, true if active, false otherwise
func isScheduleActive(s *autoscalingv1.SchedulePolicy) bool {
	now := time.Now()
	scheduleDelta := time.Minute * -1

	// If a start time is present and the current time is before the start time, the schedule is inactive so return false
	startTime := s.Between.Start.Time
	if !startTime.IsZero() && now.Before(startTime) {
		return false
	}

	// If an end time is present and the current time is after the end time, the schedule is inactive so return false
	endTime := s.Between.End.Time
	if !endTime.IsZero() && now.After(endTime) {
		return false
	}

	// If no startCron field is specified, then it's automatically true (duration is no longer relevant since we're always running)
	if s.ActivePeriod.StartCron == "" {
		return true
	}

	location, _ := time.LoadLocation(s.ActivePeriod.Timezone)
	startCron, _ := cron.ParseStandard(s.ActivePeriod.StartCron)
	nextStart := startCron.Next(now.In(location)).Add(scheduleDelta)
	duration, err := time.ParseDuration(s.ActivePeriod.Duration)

	// If there's an err, then the duration field is empty, meaning duration is indefinite
	if err != nil {
		duration = 0 // Indefinite duration if not set
	}

	// If the current time is after the next start time, and the duration is indefinite or the current time is before the next start time + duration,
	// then return true
	if now.After(nextStart) && (duration == 0 || now.Before(nextStart.Add(duration))) {
		return true
	}

	return false
}

// getSortedGameServers returns the list of Game Servers for the Fleet in the order in which the
// Game Servers would be deleted.
func getSortedGameServers(f *agonesv1.Fleet, gameServerLister listeragonesv1.GameServerLister,
	nodeCounts map[string]gameservers.NodeCount) ([]*agonesv1.GameServer, error) {
	gsList, err := fleets.ListGameServersByFleetOwner(gameServerLister, f)
	if err != nil {
		return nil, err
	}
	gameServers := gssets.SortGameServersByStrategy(f.Spec.Scheduling, gsList, nodeCounts, f.Spec.Priorities)
	return gameServers, nil
}

// isLimited indicates that the calculated scale would be above or below the range defined by
// MinCapacity and MaxCapacity in the ListPolicy or CounterPolicy.
// Return 1 if the fleet needs to scale up, -1 if the fleets need to scale down, 0 if the fleet does
// not need to scale, or if the fleet is not limited.
func isLimited(aggCapacity, minCapacity, maxCapacity int64) (bool, int) {
	if aggCapacity < minCapacity { // Scale up
		return true, 1
	}
	if aggCapacity > maxCapacity { // Scale down
		return true, -1
	}
	return false, 0
}

// scaleUpLimited scales up the fleet to meet the MinCapacity
func scaleUpLimited(replicas int32, capacity, aggCapacity, minCapacity int64) (int32, bool, error) {
	if capacity == 0 {
		return 0, false, errors.Errorf("cannot scale up as Capacity is equal to 0")
	}
	for aggCapacity < minCapacity {
		aggCapacity += capacity
		replicas++
	}
	return replicas, true, nil
}

// scaleDownLimited scales down the fleet to meet the MaxCapacity
func scaleDownLimited(f *agonesv1.Fleet, gameServerLister listeragonesv1.GameServerLister,
	nodeCounts map[string]gameservers.NodeCount, key string, isCounter bool, replicas int32,
	aggCapacity, maxCapacity int64) (int32, bool, error) {
	// Game Servers in order of deletion on scale down
	gameServers, err := getSortedGameServers(f, gameServerLister, nodeCounts)
	if err != nil {
		return 0, false, err
	}
	for _, gs := range gameServers {
		if aggCapacity <= maxCapacity {
			break
		}
		switch isCounter {
		case true:
			if counter, ok := gs.Status.Counters[key]; ok {
				aggCapacity -= counter.Capacity
			}
		case false:
			if list, ok := gs.Status.Lists[key]; ok {
				aggCapacity -= list.Capacity
			}
		}
		replicas--
	}

	// We are not currently able to scale down to zero replicas, so one replica is the minimum allowed
	if replicas < 1 {
		replicas = 1
	}

	return replicas, true, nil
}

func scaleLimited(scale int, f *agonesv1.Fleet, gameServerLister listeragonesv1.GameServerLister,
	nodeCounts map[string]gameservers.NodeCount, key string, isCounter bool, replicas int32,
	capacity, aggCapacity, minCapacity, maxCapacity int64) (int32, bool, error) {

	switch scale {
	case 1: // scale up
		return scaleUpLimited(replicas, capacity, aggCapacity, minCapacity)
	case -1: // scale down
		return scaleDownLimited(f, gameServerLister, nodeCounts, key, isCounter, replicas,
			aggCapacity, maxCapacity)
	case 0:
		return replicas, false, nil
	}

	return 0, false, errors.Errorf("cannot scale due to error in scaleLimited function")
}

// scaleUp scales up for either Integer or Percentage Buffer.
func scaleUp(replicas int32, capacity, count, aggCapacity, availableCapacity, maxCapacity,
	minCapacity, buffer int64) (int32, bool, error) {

	// How much capacity is gained by adding one more replica to the fleet.
	replicaCapacity := capacity - count
	if replicaCapacity <= 0 {
		return 0, false, errors.Errorf("cannot scale up as adding additional replicas does not increase available Capacity")
	}

	additionalReplicas := int32(math.Ceil((float64(buffer) - float64(availableCapacity)) / float64(replicaCapacity)))

	// Check to make sure we're not limited (over Max Capacity)
	limited, _ := isLimited(aggCapacity+(int64(additionalReplicas)*capacity), minCapacity, maxCapacity)
	if limited {
		additionalReplicas = int32((maxCapacity - aggCapacity) / capacity)
	}

	return replicas + additionalReplicas, limited, nil
}

// scaleDown scales down for either Integer or Percentage Buffer.
func scaleDown(f *agonesv1.Fleet, gameServerLister listeragonesv1.GameServerLister,
	nodeCounts map[string]gameservers.NodeCount, key string, isCounter bool, replicas int32,
	aggCount, aggCapacity, minCapacity, buffer int64) (int32, bool, error) {
	// Exit early if we're already at MinCapacity to avoid calling getSortedGameServers if unnecessary
	if aggCapacity == minCapacity {
		return replicas, true, nil
	}

	// We first need to get the individual game servers in order of deletion on scale down, as any
	// game server may have a unique value for counts and / or capacity.
	gameServers, err := getSortedGameServers(f, gameServerLister, nodeCounts)
	if err != nil {
		return 0, false, err
	}

	var availableCapacity int64

	// "Remove" one game server at a time in order of potential deletion. (Not actually removed here,
	// that's done later, if possible, by the fleetautoscaler controller.)
	for _, gs := range gameServers {
		replicas--
		switch isCounter {
		case true:
			if counter, ok := gs.Status.Counters[key]; ok {
				aggCount -= counter.Count
				aggCapacity -= counter.Capacity
			} else {
				continue
			}
		case false:
			if list, ok := gs.Status.Lists[key]; ok {
				aggCount -= int64(len(list.Values))
				aggCapacity -= list.Capacity
			} else {
				continue
			}
		}
		availableCapacity = aggCapacity - aggCount
		// Check if we've overshot our buffer
		if availableCapacity < buffer {
			return replicas + 1, false, nil
		}
		// Check if we're Limited (Below MinCapacity)
		if aggCapacity < minCapacity {
			return replicas + 1, true, nil
		}
		// Check if we're at our desired Buffer
		if availableCapacity == buffer {
			return replicas, false, nil
		}
		// Check if we're at Limited
		if aggCapacity == minCapacity {
			return replicas, true, nil
		}
	}

	// We are not currently able to scale down to zero replicas, so one replica is the minimum allowed.
	if replicas < 1 {
		replicas = 1
	}

	return replicas, false, nil
}
