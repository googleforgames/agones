// Copyright 2017 by the contributors.
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

package healthcheck

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
)

type metricsHandler struct {
	handler   Handler
	registry  prometheus.Registerer
	namespace string
}

// NewMetricsHandler returns a healthcheck Handler that also exposes metrics
// into the provided Prometheus registry.
func NewMetricsHandler(registry prometheus.Registerer, namespace string) Handler {
	return &metricsHandler{
		handler:   NewHandler(),
		registry:  registry,
		namespace: namespace,
	}
}

func (h *metricsHandler) AddLivenessCheck(name string, check Check) {
	h.handler.AddLivenessCheck(name, h.wrap(name, check))
}

func (h *metricsHandler) AddReadinessCheck(name string, check Check) {
	h.handler.AddReadinessCheck(name, h.wrap(name, check))
}

func (h *metricsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.handler.ServeHTTP(w, r)
}

func (h *metricsHandler) LiveEndpoint(w http.ResponseWriter, r *http.Request) {
	h.handler.LiveEndpoint(w, r)
}

func (h *metricsHandler) ReadyEndpoint(w http.ResponseWriter, r *http.Request) {
	h.handler.ReadyEndpoint(w, r)
}

func (h *metricsHandler) wrap(name string, check Check) Check {
	h.registry.MustRegister(prometheus.NewGaugeFunc(
		prometheus.GaugeOpts{
			Namespace:   h.namespace,
			Subsystem:   "healthcheck",
			Name:        "status",
			Help:        "Current check status (0 indicates success, 1 indicates failure)",
			ConstLabels: prometheus.Labels{"check": name},
		},
		func() float64 {
			if check() == nil {
				return 0
			}
			return 1
		},
	))
	return check
}
