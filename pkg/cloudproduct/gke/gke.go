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

// Package gke implements the GKE cloud product (specifically Autopilot for now)
package gke

import (
	"context"
	"encoding/json"
	"fmt"

	"agones.dev/agones/pkg/apis"
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
	hostPortAssignmentAnnotation = "autopilot.gke.io/host-port-assignment"

	errPortPolicyMustBeDynamic      = "portPolicy must be Dynamic on GKE Autopilot"
	errSchedulingMustBePacked       = "scheduling strategy must be Packed on GKE Autopilot"
	errEvictionSafeOnUpgradeInvalid = "eviction.safe OnUpgrade not supported on GKE Autopilot"
)

var (
	autopilotMutatingWebhooks = []string{
		"workload-defaulter.config.common-webhooks.networking.gke.io", // pre-1.26
		"warden-mutating.config.common-webhooks.networking.gke.io",    // 1.26+
	}
	noWorkloadDefaulter = fmt.Sprintf("found no MutatingWebhookConfigurations matching %v", autopilotMutatingWebhooks)

	logger = runtime.NewLoggerWithSource("gke")
)

type gkeAutopilot struct{}

// hostPortAssignment is the JSON structure of the `host-port-assignment` annotation
//
//nolint:govet // API-like, keep consistent
type hostPortAssignment struct {
	Min           int32           `json:"min,omitempty"`
	Max           int32           `json:"max,omitempty"`
	PortsAssigned map[int32]int32 `json:"portsAssigned,omitempty"` // old -> new
}

// Detect whether we're running on GKE and/or Autopilot and return the appropriate
// cloud product string.
func Detect(ctx context.Context, kc *kubernetes.Clientset) string {
	if !metadata.OnGCE() {
		return ""
	}
	// Look for the workload defaulter - this is the current best method to detect Autopilot
	found := false
	for _, webhook := range autopilotMutatingWebhooks {
		if _, err := kc.AdmissionregistrationV1().MutatingWebhookConfigurations().Get(
			ctx, webhook, metav1.GetOptions{}); err != nil {
			logger.WithError(err).WithField("webhook", webhook).Info("Detecting Autopilot MutatingWebhookConfiguration")
		} else {
			found = true
			break
		}
	}
	if !found {
		logger.WithField("reason", noWorkloadDefaulter).Info(
			"Assuming GKE Standard and defaulting to generic provider (expected if not on GKE Autopilot)")
		return "" // GKE standard, but we don't need an interface for it just yet.
	}
	logger.Info("Running on GKE Autopilot (skip detection with --cloud-product=gke-autopilot)")
	return "gke-autopilot"
}

// Autopilot returns a GKE Autopilot cloud product
//
//nolint:revive // ignore the unexported return; implements ControllerHooksInterface
func Autopilot() *gkeAutopilot { return &gkeAutopilot{} }

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

func (*gkeAutopilot) WaitOnFreePorts() bool { return true }

func (g *gkeAutopilot) ValidateGameServerSpec(gss *agonesv1.GameServerSpec) []metav1.StatusCause {
	causes := g.ValidateScheduling(gss.Scheduling)
	for _, p := range gss.Ports {
		if p.PortPolicy != agonesv1.Dynamic {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Field:   fmt.Sprintf("%s.portPolicy", p.Name),
				Message: errPortPolicyMustBeDynamic,
			})
		}
	}
	// See SetEviction comment below for why we block EvictionSafeOnUpgrade.
	if gss.Eviction.Safe == agonesv1.EvictionSafeOnUpgrade {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Field:   "eviction.safe",
			Message: errEvictionSafeOnUpgradeInvalid,
		})
	}
	return causes
}

func (*gkeAutopilot) ValidateScheduling(ss apis.SchedulingStrategy) []metav1.StatusCause {
	if ss != apis.Packed {
		return []metav1.StatusCause{{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Field:   "scheduling",
			Message: errSchedulingMustBePacked,
		}}
	}
	return nil
}

func (*gkeAutopilot) MutateGameServerPodSpec(gss *agonesv1.GameServerSpec, podSpec *corev1.PodSpec) error {
	podSpecSeccompUnconfined(podSpec)
	return nil
}

// podSpecSeccompUnconfined sets to seccomp profile to `Unconfined` to avoid serious performance
// degradation possible with seccomp. We only set the pod level seccompProfile, and only set
// it if it hasn't been set - users can then override at either the pod or container level
// in the GameServer spec.
func podSpecSeccompUnconfined(podSpec *corev1.PodSpec) {
	if podSpec.SecurityContext != nil && podSpec.SecurityContext.SeccompProfile != nil {
		return
	}
	if podSpec.SecurityContext == nil {
		podSpec.SecurityContext = &corev1.PodSecurityContext{}
	}
	podSpec.SecurityContext.SeccompProfile = &corev1.SeccompProfile{Type: corev1.SeccompProfileTypeUnconfined}
}

// SetEviction sets disruption controls based on GameServer.Status.Eviction. For Autopilot:
//   - Since the safe-to-evict pod annotation is not supported if "false", we delete it (if it's set
//     to anything else, we allow it - Autopilot only rejects "false").
//   - OnUpgrade is not supported and rejected by validation above. Since we can't support
//     safe-to-evict=false but can support a restrictive PDB, we can support Never and Always, but
//     OnUpgrade doesn't make sense on Autopilot today. - an overly restrictive PDB prevents
//     any sort of graceful eviction.
func (*gkeAutopilot) SetEviction(eviction *agonesv1.Eviction, pod *corev1.Pod) error {
	if !runtime.FeatureEnabled(runtime.FeatureSafeToEvict) {
		return nil
	}
	if safeAnnotation := pod.ObjectMeta.Annotations[agonesv1.PodSafeToEvictAnnotation]; safeAnnotation == agonesv1.False {
		delete(pod.ObjectMeta.Annotations, agonesv1.PodSafeToEvictAnnotation)
	}
	if eviction == nil {
		return errors.New("No eviction value set. Should be the default value")
	}
	if _, exists := pod.ObjectMeta.Labels[agonesv1.SafeToEvictLabel]; !exists {
		switch eviction.Safe {
		case agonesv1.EvictionSafeAlways:
			// For EvictionSafeAlways, we use a label value that does not match the
			// agones-gameserver-safe-to-evict-false PDB. But we go ahead and label
			// it, in case someone wants to adopt custom logic for this group of
			// game servers.
			pod.ObjectMeta.Labels[agonesv1.SafeToEvictLabel] = agonesv1.True
		case agonesv1.EvictionSafeNever:
			pod.ObjectMeta.Labels[agonesv1.SafeToEvictLabel] = agonesv1.False
		default:
			return errors.Errorf("eviction.safe == %s, which webhook should have rejected on Autopilot", eviction.Safe)
		}
	}
	return nil
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
