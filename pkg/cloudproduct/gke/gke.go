// Copyright 2022 Google LLC All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package gke

import (
	"context"
	"encoding/json"

	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	"agones.dev/agones/pkg/util/runtime"
	"cloud.google.com/go/compute/metadata"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	workloadDefaulterWebhook     = "workload-defaulter.config.common-webhooks.networking.gke.io"
	noWorkloadDefaulter          = "failed to get MutatingWebhookConfigurations/workload-defaulter.config.common-webhooks.networking.gke.io (error expected if not on GKE Autopilot)"
	hostPortAssignmentAnnotation = "autopilot.gke.io/host-port-assignment"
)

var logger = runtime.NewLoggerWithSource("gke")

type gkeAutopilot struct{}

// hostPortAssignment is the JSON structure of the `host-port-assignment` annotation
type hostPortAssignment struct {
	Min           int32           `json:"min,omitempty"`
	Max           int32           `json:"max,omitempty"`
	PortsAssigned map[int32]int32 `json:"portsAssigned,omitempty"` // old -> new
}

func Detect(ctx context.Context, kc *kubernetes.Clientset) string {
	if !metadata.OnGCE() {
		return ""
	}
	// Look for the workload defaulter - this is the current best method to detect Autopilot
	if _, err := kc.AdmissionregistrationV1().MutatingWebhookConfigurations().Get(
		ctx, workloadDefaulterWebhook, metav1.GetOptions{}); err != nil {
		logger.WithError(err).WithField("reason", noWorkloadDefaulter).Info(
			"Assuming GKE Standard and defaulting to generic provider")
		return "" // GKE standard, but we don't need an interface for it just yet.
	}
	logger.Info("Running on GKE Autopilot (skip detection with --cloud-product=gke-autopilot)")
	return "gke-autopilot"
}

func Autopilot() (*gkeAutopilot, error) { return &gkeAutopilot{}, nil }

func (*gkeAutopilot) SyncPodPortsToGameServer(gs *agonesv1.GameServer, pod *corev1.Pod) error {
	// If applyGameServerAddressAndPort has already filled in Status, SyncPodPortsToGameServer
	// has already run. Skip syncing from the Pod again - this avoids having to reason
	// about whether we're re-applying the old->new mapping.
	if len(gs.Status.Ports) == len(gs.Spec.Ports) {
		return nil
	}
	annotation, ok := pod.ObjectMeta.Annotations[hostPortAssignmentAnnotation]
	if !ok {
		return nil
	}
	var hpa hostPortAssignment
	if err := json.Unmarshal([]byte(annotation), &hpa); err != nil {
		return errors.Wrapf(err, "could not unmarshal annotation %s (value %q)", hostPortAssignmentAnnotation, annotation)
	}
	for i, p := range gs.Spec.Ports {
		if newPort, ok := hpa.PortsAssigned[p.HostPort]; ok {
			gs.Spec.Ports[i].HostPort = newPort
		}
	}
	return nil
}
