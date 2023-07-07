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

	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	autoscalingv1 "agones.dev/agones/pkg/apis/autoscaling/v1"
	listeragonesv1 "agones.dev/agones/pkg/client/listers/agones/v1"
	"agones.dev/agones/pkg/fleets"
	"agones.dev/agones/pkg/gameservers"
	gssets "agones.dev/agones/pkg/gameserversets"
	"agones.dev/agones/pkg/util/runtime"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/uuid"
)

var tlsConfig = &tls.Config{}
var client = http.Client{
	Timeout: 15 * time.Second,
	Transport: &http.Transport{
		TLSClientConfig: tlsConfig,
	},
}

// computeDesiredFleetSize computes the new desired size of the given fleet
func computeDesiredFleetSize(fas *autoscalingv1.FleetAutoscaler, f *agonesv1.Fleet,
	gameServerLister listeragonesv1.GameServerLister, nodeCounts map[string]gameservers.NodeCount) (int32, bool, error) {
	switch fas.Spec.Policy.Type {
	case autoscalingv1.BufferPolicyType:
		return applyBufferPolicy(fas.Spec.Policy.Buffer, f)
	case autoscalingv1.WebhookPolicyType:
		return applyWebhookPolicy(fas.Spec.Policy.Webhook, f)
	case autoscalingv1.CounterPolicyType:
		return applyCounterOrListPolicy(fas.Spec.Policy.Counter, nil, f, gameServerLister, nodeCounts)
	case autoscalingv1.ListPolicyType:
		return applyCounterOrListPolicy(nil, fas.Spec.Policy.List, f, gameServerLister, nodeCounts)
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

	limited := false

	if replicas < b.MinReplicas {
		replicas = b.MinReplicas
		limited = true
	}
	if replicas > b.MaxReplicas {
		replicas = b.MaxReplicas
		limited = true
	}

	return replicas, limited, nil
}

func applyCounterOrListPolicy(c *autoscalingv1.CounterPolicy, l *autoscalingv1.ListPolicy,
	f *agonesv1.Fleet, gameServerLister listeragonesv1.GameServerLister,
	nodeCounts map[string]gameservers.NodeCount) (int32, bool, error) {

	if !runtime.FeatureEnabled(runtime.FeatureCountsAndLists) {
		return 0, false, errors.Errorf("cannot apply CounterPolicy unless feature flag %s is enabled", runtime.FeatureCountsAndLists)
	}

	var isCounter bool    // True if a CounterPolicy False if a ListPolicy
	var key string        // The specified Counter or List
	var count int64       // The Count or number of Values in the template Game Server
	var capacity int64    // The Capacity in the template Game Server
	var aggCount int64    // The Aggregate Count of the specified Counter or List of all GameServers across the GameServerSet in the Fleet
	var aggCapacity int64 // The Aggregate Capacity of the specified Counter or List of all GameServers across the GameServerSet in the Fleet
	var minCapacity int64 // The Minimum Aggregate Capacity
	var maxCapacity int64 // The Maximum Aggregate Capacity

	var bufferSize intstr.IntOrString
	// TODO: Our aggregate counts in include Allocated, Ready, and Reserved replicas, however
	// I'm not 100% sure if f.Status.Replicas includes all three states or just Allocated and Ready?
	// If it's the latter, then we would need to add in f.Status.ReservedReplicas.
	// There may be some additional nuance here since "Reserved instances won't be deleted on scale down, but won't cause an autoscaler to scale up"
	replicas := f.Status.Replicas

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
		minCapacity = l.MinCapacity
		maxCapacity = l.MaxCapacity
		bufferSize = l.BufferSize
	}

	// CASES:
	// No Scaling Integer (at desired Buffer) -- Check if below/above min/max capacity (Limited) and return current number of replicas if not Limited
	// Scale Up Integer -- May be Limited before or after Scaling
	// Scale Down Integer -- May be Limited before or after Scaling
	// No Scaling Percent -- Check if Limited and return current number of replicas if not Limited
	// Scale Up Percent -- May be Limited before or after Scaling
	// Scale Down Percent -- May be Limited before or after Scaling
	// If none of the above return error Unable to Apply Counter or List Policy

	limited, scale := isLimited(aggCapacity, minCapacity, maxCapacity)

	// Desired replicas based on BufferSize specified as an absolute value (i.e. 5)
	if bufferSize.Type == intstr.Int {
		buffer := int64(bufferSize.IntValue())

		// Current available capacity across the fleet
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
			return scaleUpByInteger(replicas, capacity, count,
				aggCapacity, availableCapacity, maxCapacity, buffer)
		case availableCapacity > buffer: // Scale Down
			if limited && scale == 1 { // Case where we want to scale down but we're already limited by MinCapacity
				return scaleLimited(scale, f, gameServerLister, nodeCounts, key, isCounter, replicas,
					capacity, aggCapacity, minCapacity, maxCapacity)
			}
			return scaleDownByInteger(f, gameServerLister, nodeCounts, key, isCounter, replicas,
				availableCapacity, aggCount, aggCapacity, minCapacity, buffer)
		}
	}

	// Desired replicas based on BufferSize specified as a percent (i.e. 5%)
	bufferPercent, err := intstr.GetValueFromIntOrPercent(&bufferSize, 100, isCounter)
	if err != nil {
		return 0, false, err
	}
	// The desired total capacity across the fleet (see applyBufferPolicy for explanation)
	desiredCapacity := float64(aggCount*100) / float64(100-bufferPercent)

	switch roundedDesiredCapacity := int64(math.Ceil(desiredCapacity)); {
	case roundedDesiredCapacity == aggCapacity:
		if limited {
			return scaleLimited(scale, f, gameServerLister, nodeCounts, key, isCounter, replicas,
				capacity, aggCapacity, minCapacity, maxCapacity)
		}
		return replicas, false, nil

	case roundedDesiredCapacity > aggCapacity: // Scale up
		if limited && scale == -1 { // Case where we want to scale up but we're already limited by MaxCapacity
			return scaleLimited(scale, f, gameServerLister, nodeCounts, key, isCounter, replicas,
				capacity, aggCapacity, minCapacity, maxCapacity)
		}
		return scaleUpByPercent(replicas, count, aggCount, capacity, aggCapacity, maxCapacity,
			desiredCapacity, float64(bufferPercent))

	case roundedDesiredCapacity < aggCapacity: // Scale down
		if limited && scale == 1 { // Case where we want to scale down but we're already limited by MinCapacity
			return scaleLimited(scale, f, gameServerLister, nodeCounts, key, isCounter, replicas,
				capacity, aggCapacity, minCapacity, maxCapacity)
		}
		return scaleDownByPercent(f, gameServerLister, nodeCounts, key, isCounter, replicas,
			aggCount, aggCapacity, minCapacity, float64(bufferPercent))
	}

	if isCounter {
		return 0, false, errors.Errorf("unable to apply CounterPolicy %v", c)
	}
	return 0, false, errors.Errorf("unable to apply ListPolicy %v", l)
}

// getSortedGameServers returns the list of Game Servers for the Fleet in the order in which the
// Game Servers would be deleted.
func getSortedGameServers(f *agonesv1.Fleet, gameServerLister listeragonesv1.GameServerLister,
	nodeCounts map[string]gameservers.NodeCount) ([]*agonesv1.GameServer, error) {
	// TODO: Should we handle this differently for strategy Distributed?
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

	if replicas < 0 { // This shouldn't ever happen, but putting it here just in case.
		replicas = 0
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

// scaleUpByInteger
func scaleUpByInteger(replicas int32, capacity, count, aggCapacity, availableCapacity, maxCapacity,
	buffer int64) (int32, bool, error) {

	// How much capacity is gained by adding one more replica to the fleet.
	replicaCapacity := capacity - count
	if replicaCapacity == 0 {
		return 0, false, errors.Errorf("cannot scale up as adding additional replicas does not increase Capacity")
	}

	additionalReplicas := int64(math.Ceil((float64(buffer) - float64(availableCapacity)) / float64(replicaCapacity)))

	// Check if we've hit MaxCapacity
	if (additionalReplicas*capacity)+aggCapacity > maxCapacity {
		additionalReplicas = (maxCapacity - aggCapacity) / capacity
		return replicas + int32(additionalReplicas), true, nil
	}

	return replicas + int32(additionalReplicas), false, nil
}

func scaleDownByInteger(f *agonesv1.Fleet, gameServerLister listeragonesv1.GameServerLister,
	nodeCounts map[string]gameservers.NodeCount, key string, isCounter bool, replicas int32,
	availableCapacity, aggCount, aggCapacity, minCapacity, buffer int64) (int32, bool, error) {

	if aggCapacity == minCapacity {
		return replicas, true, nil
	}

	if availableCapacity == buffer {
		return replicas, false, nil
	}

	gameServers, err := getSortedGameServers(f, gameServerLister, nodeCounts)
	if err != nil {
		return 0, false, err
	}

	for _, gs := range gameServers {
		replicas--
		switch isCounter {
		case true:
			if counter, ok := gs.Status.Counters[key]; ok {
				aggCount -= counter.Count
				aggCapacity -= counter.Capacity
			}
		case false:
			if list, ok := gs.Status.Lists[key]; ok {
				aggCount -= int64(len(list.Values))
				aggCapacity -= list.Capacity
			}
		}
		availableCapacity = aggCapacity - aggCount
		// Check if we've hit MinCapacity
		if aggCapacity == minCapacity {
			return replicas, true, nil
		}
		if aggCapacity < minCapacity {
			return replicas + 1, true, nil
		}
		// Check if we're at our desired Buffer
		if availableCapacity == buffer {
			return replicas, false, nil
		}
		if availableCapacity < buffer {
			return replicas + 1, false, nil
		}
	}

	if replicas < 0 { // This shouldn't ever happen, but putting it here just in case.
		replicas = 0
	}

	return replicas, false, nil
}

// scaleUpByPercent
func scaleUpByPercent(currReplicas int32, count, aggCount, capacity, aggCapacity,
	maxCapacity int64, desiredCapacity, bufferPercent float64) (int32, bool, error) {

	if capacity == 0 {
		return 0, false, errors.Errorf("cannot scale up as Capacity is equal to 0")
	}

	additionalReplicas := int64(math.Ceil((desiredCapacity - float64(aggCapacity)) / (float64(capacity) - float64(count))))
	replicas := currReplicas + int32(additionalReplicas)
	// Case where adding a List or Counter does not change the Count (and thus does not change our desiredCapacity)
	if count == 0 {
		return replicas, false, nil
	}

	// In case where len(list.Values) or counter.Count != 0 then we need to update desiredCapacity
	// each time we add a List or Counter. Start at point where desired replicas was determined based
	// on case where len(list.Values) or counter.Count == 0.
	aggCount += count * additionalReplicas
	aggCapacity += capacity * additionalReplicas
	limited := false
	for {
		// Check if we've reached MaxCapacity
		if aggCapacity == maxCapacity {
			limited = true
			break
		}
		if aggCapacity > maxCapacity {
			limited = true
			replicas--
			break
		}
		// Check if we've reached desiredCapacity
		desiredCapacity = (float64(aggCount) * 100) / (100 - bufferPercent)
		desiredReplicas := math.Ceil(desiredCapacity / float64(capacity))
		if replicas >= int32(desiredReplicas) {
			break
		}
		// Keep checking if adding one List or Counter will reach our desiredCapacity
		aggCount += count
		aggCapacity += capacity
		replicas++
	}

	return replicas, limited, nil
}

// scaleDownByPercent
// TODO: This death-spirals down to min capacity, so the customer would need to have something
// in place to prevent gameservers from being evicted elsewhere if they have Count / Values on
// them. I'm assuming this is the behavior we intend, and it just needs good documentation?
func scaleDownByPercent(f *agonesv1.Fleet, gameServerLister listeragonesv1.GameServerLister,
	nodeCounts map[string]gameservers.NodeCount, key string, isCounter bool, replicas int32,
	aggCount, aggCapacity, minCapacity int64, bufferPercent float64) (int32, bool, error) {
	// Exit early if we're already at MinCapacity to avoid calling getSortedGameServers if unnecessary
	if aggCapacity == minCapacity {
		return replicas, true, nil
	}

	// We first need to get the individual game servers in order of deletion on scale down, as both
	// the Count and Capacity can change, so we do not know from the aggregate counts how much
	// removing x game servers will affect the aggregate count and capacity.
	gameServers, err := getSortedGameServers(f, gameServerLister, nodeCounts)
	if err != nil {
		return 0, false, err
	}

	// Determine the desiredCapacity based on removing one gameserver at a time
	limited := false
	var desiredCapacity int64
	for _, gs := range gameServers {
		// Check if we've reached desiredCapacity
		desiredCapacity = int64(math.Ceil((float64(aggCount) * 100) / (100 - bufferPercent)))
		if desiredCapacity >= aggCapacity {
			break
		}
		// Keep checking if adding removing one Counter or List will reach our desiredCapacity
		switch isCounter {
		case true:
			if counter, ok := gs.Status.Counters[key]; ok {
				aggCount -= counter.Count
				aggCapacity -= counter.Capacity
			}
		case false:
			if list, ok := gs.Status.Lists[key]; ok {
				aggCount -= int64(len(list.Values))
				aggCapacity -= list.Capacity
			}
		}
		replicas--
		// Check if we've reached MinCapacity
		if aggCapacity == minCapacity {
			limited = true // TODO: Should we have this return true or false when scaling down to 0 (MinCapacity == 0)?
			break
		}
		if aggCapacity < minCapacity {
			limited = true
			replicas++
			break
		}
	}

	if replicas < 0 {
		replicas = 0
	}

	return replicas, limited, nil
}
