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
func computeDesiredFleetSize(fas *autoscalingv1.FleetAutoscaler, f *agonesv1.Fleet) (int32, bool, error) {
	switch fas.Spec.Policy.Type {
	case autoscalingv1.BufferPolicyType:
		return applyBufferPolicy(fas.Spec.Policy.Buffer, f)
	case autoscalingv1.WebhookPolicyType:
		return applyWebhookPolicy(fas.Spec.Policy.Webhook, f)
	case autoscalingv1.CounterPolicyType:
		return applyCounterPolicy(fas.Spec.Policy.Counter, f)
	}

	return 0, false, errors.New("wrong policy type, should be one of: Buffer, Webhook")
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

func applyCounterPolicy(c *autoscalingv1.CounterPolicy, f *agonesv1.Fleet) (int32, bool, error) {
	if !runtime.FeatureEnabled(runtime.FeatureCountsAndLists) {
		return 0, false, errors.Errorf("cannot apply CounterPolicy unless feature flag %s is enabled", runtime.FeatureCountsAndLists)
	}

	counter, ok := f.Spec.Template.Spec.Counters[c.Key]
	if !ok {
		return 0, false, errors.Errorf("cannot apply CounterPolicy as counter key %s does not exist in the Fleet Spec", c.Key)
	}

	aggCounter, ok := f.Status.Counters[c.Key]
	if !ok {
		return 0, false, errors.Errorf("cannot apply CounterPolicy as counter key %s does not exist in the Fleet Status", c.Key)
	}

	var replicas float64

	// Current available capacity across the fleet
	availableCapacity := float64(aggCounter.Capacity - aggCounter.Count)

	// How much available capacity is gained by adding one more replica to the fleet.
	replicaCapacity := float64(counter.Capacity - counter.Count)

	// Desired replicas based on BufferSize specified as an absolute value (i.e. 5)
	if c.BufferSize.Type == intstr.Int {
		// If our current available is the same as our buffer, then we already have the desired replicas
		buffer := float64(c.BufferSize.IntValue())
		if availableCapacity == buffer {
			replicas = float64(f.Status.Replicas)
		} else {
			diffReplicas := math.Floor((availableCapacity - buffer) / replicaCapacity)
			// TODO: Our aggregate counts in include Allocated, Ready, and Reserved replicas, however
			// I'm not 100% sure if f.Status.Replicas includes all three states or just Allocated and Ready?
			// If it's the latter, then we would need to add in f.Status.ReservedReplicas.
			// There may be some additional nuance here since "Reserved instances won't be deleted on scale down, but won't cause an autoscaler to scale up"
			replicas = float64(f.Status.Replicas) - diffReplicas
		}
	} else {
		// Desired replicas based on BufferSize specified as a percent (i.e. 5%)
		bufferPercent, err := intstr.GetValueFromIntOrPercent(&c.BufferSize, 100, true)
		if err != nil {
			return 0, false, err
		}
		// The desired total capacity across the fleet (see applyBufferPolicy for explanation)
		desiredCapacity := float64(aggCounter.Count*100) / float64(100-bufferPercent)
		if math.Ceil(desiredCapacity) == float64(aggCounter.Capacity) {
			replicas = float64(f.Status.Replicas)
		} else {
			// TODO: How to better handle case where removing ready game servers also reduces the count?
			// Using replicaCapacity or using counter.Capacity do not work in all cases.
			// replicas = math.Ceil(desiredCapacity / replicaCapacity)
			replicas = math.Ceil(desiredCapacity / float64(counter.Capacity))
		}
	}

	// limited indicates that the calculated scale would be above or below the range defined by
	// MinCapacity and MaxCapacity
	limited := false

	if replicas < math.Ceil(float64(c.MinCapacity)/float64(counter.Capacity)) {
		replicas = math.Ceil(float64(c.MinCapacity) / float64(counter.Capacity))
		limited = true
	}
	// Note that we may have a greater MaxCapacity than stated in the CounterPolicy. For example, if
	// the MaxCapacity is 10 and each replica adds 3 to the total capacity we could end up with 4
	// replicas for a total capacity of 12.
	if replicas > math.Ceil(float64(c.MaxCapacity)/float64(counter.Capacity)) {
		replicas = math.Ceil(float64(c.MaxCapacity) / float64(counter.Capacity))
		limited = true
	}

	return int32(replicas), limited, nil
}

func applyListPolicy(l *autoscalingv1.ListPolicy, f *agonesv1.Fleet) (int32, bool, error) {
	if !runtime.FeatureEnabled(runtime.FeatureCountsAndLists) {
		return 0, false, errors.Errorf("cannot apply ListPolicy unless feature flag %s is enabled", runtime.FeatureCountsAndLists)
	}

	list, ok := f.Spec.Template.Spec.Lists[l.Key]
	if !ok {
		return 0, false, errors.Errorf("cannot apply ListPolicy as list key %s does not exist in the Fleet Spec", l.Key)
	}

	aggList, ok := f.Status.Lists[l.Key]
	if !ok {
		return 0, false, errors.Errorf("cannot apply ListPolicy as list key %s does not exist in the Fleet Status", l.Key)
	}

	var replicas float64

	// Current available capacity across the fleet
	availableCapacity := float64(aggList.Capacity - aggList.Count)

	// How much capacity is gained by adding one more replica to the fleet.
	replicaCapacity := float64(list.Capacity - int64(len(list.Values)))

	// Desired replicas based on BufferSize specified as an absolute value (i.e. 5)
	if l.BufferSize.Type == intstr.Int {
		// If our current available is the same as our buffer, then we already have the desired replicas
		buffer := float64(l.BufferSize.IntValue())
		if availableCapacity == buffer {
			replicas = float64(f.Status.Replicas)
		} else {
			diffReplicas := math.Floor((availableCapacity - buffer) / replicaCapacity)
			replicas = float64(f.Status.Replicas) - diffReplicas
		}
	} else {
		// Desired replicas based on BufferSize specified as a percent (i.e. 5%)
		bufferPercent, err := intstr.GetValueFromIntOrPercent(&l.BufferSize, 100, true)
		if err != nil {
			return 0, false, err
		}
		// The desired total capacity across the fleet (see applyBufferPolicy for explanation)
		desiredCapacity := float64(aggList.Count*100) / float64(100-bufferPercent)
		if math.Ceil(desiredCapacity) == float64(aggList.Capacity) {
			replicas = float64(f.Status.Replicas)
		} else {
			// TODO: Scale down & up roesn't work in all cases -- see applyCounterPolicy and TestApplyListPolicy
			replicas = math.Ceil(desiredCapacity / float64(list.Capacity))
		}
	}

	// limited indicates that the calculated scale would be above or below the range defined by
	// MinCapacity and MaxCapacity
	limited := false

	if replicas < math.Ceil(float64(l.MinCapacity)/float64(list.Capacity)) {
		replicas = math.Ceil(float64(l.MinCapacity) / float64(list.Capacity))
		limited = true
	}
	// Note that we may have a greater MaxCapacity than stated in the ListPolicy. For example, if
	// the MaxCapacity is 10 and each replica adds 3 to the total capacity we could end up with 4
	// replicas for a total capacity of 12.
	if replicas > math.Ceil(float64(l.MaxCapacity)/float64(list.Capacity)) {
		replicas = math.Ceil(float64(l.MaxCapacity) / float64(list.Capacity))
		limited = true
	}

	return int32(replicas), limited, nil
}
