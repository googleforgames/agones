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
	"fmt"

	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	"agones.dev/agones/pkg/client/informers/externalversions"
	"agones.dev/agones/pkg/portallocator"
	"agones.dev/agones/pkg/util/runtime"
	"cloud.google.com/go/compute/metadata"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
)

const (
	workloadDefaulterWebhook     = "workload-defaulter.config.common-webhooks.networking.gke.io"
	noWorkloadDefaulter          = "failed to get MutatingWebhookConfigurations/workload-defaulter.config.common-webhooks.networking.gke.io (error expected if not on GKE Autopilot)"
	hostPortAssignmentAnnotation = "autopilot.gke.io/host-port-assignment"

	errPortPolicyMustBeDynamic = "PortPolicy must be Dynamic on GKE Autopilot"
)

var logger = runtime.NewLoggerWithSource("gke")

type gkeAutopilot struct{}

// hostPortAssignment is the JSON structure of the `host-port-assignment` annotation
//
//nolint:govet // API-like, keep consistent
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

func (*gkeAutopilot) NewPortAllocator(minPort, maxPort int32,
	_ informers.SharedInformerFactory,
	_ externalversions.SharedInformerFactory) portallocator.Interface {
	return &autopilotPortAllocator{minPort: minPort, maxPort: maxPort}
}

func (*gkeAutopilot) ValidateGameServer(gs *agonesv1.GameServer) []metav1.StatusCause {
	var causes []metav1.StatusCause
	for _, p := range gs.Spec.Ports {
		if p.PortPolicy != agonesv1.Dynamic {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Field:   fmt.Sprintf("%s.portPolicy", p.Name),
				Message: errPortPolicyMustBeDynamic,
			})
		}
	}
	return causes
}

type autopilotPortAllocator struct {
	minPort int32
	maxPort int32
}

func (*autopilotPortAllocator) Run(_ context.Context) error        { return nil }
func (*autopilotPortAllocator) DeAllocate(gs *agonesv1.GameServer) {}

func (apa *autopilotPortAllocator) Allocate(gs *agonesv1.GameServer) *agonesv1.GameServer {
	if len(gs.Spec.Ports) == 0 {
		return gs // Nothing to do.
	}

	var ports []agonesv1.GameServerPort
	for i, p := range gs.Spec.Ports {
		if p.PortPolicy != agonesv1.Dynamic {
			logger.WithField("gs", gs.Name).WithField("portPolicy", p.PortPolicy).Error(
				"GameServer has invalid PortPolicy for Autopilot - this should have been rejected by webhooks. Refusing to assign ports.")
			return gs
		}
		p.HostPort = int32(i + 1) // Autopilot expects _some_ host port - use a value unique to this GameServer Port.

		if p.Protocol == agonesv1.ProtocolTCPUDP {
			tcp := p
			tcp.Name = p.Name + "-tcp"
			tcp.Protocol = corev1.ProtocolTCP
			ports = append(ports, tcp)

			p.Name += "-udp"
			p.Protocol = corev1.ProtocolUDP
		}
		ports = append(ports, p)
	}

	hpa := hostPortAssignment{Min: apa.minPort, Max: apa.maxPort}
	hpaJSON, err := json.Marshal(hpa)
	if err != nil {
		logger.WithError(err).WithField("hostPort", hpa).WithField("gs", gs.Name).Error("Internal error marshalling hostPortAssignment for GameServer")
		// In error cases, return the original gs - on Autopilot this will result in a policy failure.
		return gs
	}

	// No errors past here.
	gs.Spec.Ports = ports
	if gs.Spec.Template.ObjectMeta.Annotations == nil {
		gs.Spec.Template.ObjectMeta.Annotations = make(map[string]string)
	}
	gs.Spec.Template.ObjectMeta.Annotations[hostPortAssignmentAnnotation] = string(hpaJSON)
	return gs
}
