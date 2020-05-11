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
	"io/ioutil"
	"math"
	"net/http"
	"net/url"
	"strings"
	"time"

	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	autoscalingv1 "agones.dev/agones/pkg/apis/autoscaling/v1"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/uuid"
)

var client = http.Client{
	Timeout: 15 * time.Second,
}

// computeDesiredFleetSize computes the new desired size of the given fleet
func computeDesiredFleetSize(fas *autoscalingv1.FleetAutoscaler, f *agonesv1.Fleet) (int32, bool, error) {

	switch fas.Spec.Policy.Type {
	case autoscalingv1.BufferPolicyType:
		return applyBufferPolicy(fas.Spec.Policy.Buffer, f)
	case autoscalingv1.WebhookPolicyType:
		return applyWebhookPolicy(fas.Spec.Policy.Webhook, f)
	}

	return f.Status.Replicas, false, errors.New("wrong policy type, should be one of: Buffer, Webhook")
}

func applyWebhookPolicy(w *autoscalingv1.WebhookPolicy, f *agonesv1.Fleet) (int32, bool, error) {
	if w == nil {
		return 0, false, errors.New("nil WebhookPolicy passed")
	}

	if f == nil {
		return 0, false, errors.New("nil Fleet passed")
	}

	if w.URL != nil && w.Service != nil {
		return 0, false, errors.New("service and url cannot be used simultaneously")
	}

	var u *url.URL
	var err error

	if w.URL != nil {
		if *w.URL == "" {
			return 0, false, errors.New("URL was not provided")
		}
		u, err = url.ParseRequestURI(*w.URL)
		if err != nil {
			return 0, false, err
		}
	} else {
		if w.Service.Name == "" {
			return 0, false, errors.New("service name was not provided")
		}

		var servicePath string
		if w.Service.Path != nil {
			servicePath = *w.Service.Path
		}

		if w.Service.Namespace == "" {
			w.Service.Namespace = "default"
		}

		scheme := "http"
		if w.CABundle != nil {
			scheme = "https"

			// We can have multiple fleetautoscalers with different CABundles defined,
			// so we switch client.Transport before each POST request
			rootCAs := x509.NewCertPool()
			if ok := rootCAs.AppendCertsFromPEM(w.CABundle); !ok {
				return 0, false, errors.New("no certs were appended from caBundle")
			}
			client.Transport = &http.Transport{
				TLSClientConfig: &tls.Config{
					RootCAs: rootCAs,
				},
			}
		}

		u = buildURL(scheme, w.Service.Name, w.Service.Namespace, servicePath)
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
			log.Error(cerr)
		}
	}()

	if res.StatusCode != http.StatusOK {
		return 0, false, fmt.Errorf("bad status code %d from the server: %s", res.StatusCode, u.String())
	}
	result, err := ioutil.ReadAll(res.Body)
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

func buildURL(scheme, name, namespace, path string) *url.URL {
	return &url.URL{
		Scheme: scheme,
		Host:   fmt.Sprintf("%s.%s.svc:8000", name, namespace),
		Path:   path,
	}
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
			return f.Status.Replicas, false, err
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
