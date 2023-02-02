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

// Package generic implements generic cloud product hooks
package generic

import (
	"agones.dev/agones/pkg/apis"
	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	"agones.dev/agones/pkg/client/informers/externalversions"
	"agones.dev/agones/pkg/portallocator"
	"agones.dev/agones/pkg/util/runtime"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
)

// New returns a new generic cloud product
//
//nolint:revive // ignore the unexported return; implements ControllerHooksInterface
func New() (*generic, error) { return &generic{}, nil }

type generic struct{}

func (*generic) ValidateGameServerSpec(*agonesv1.GameServerSpec) []metav1.StatusCause    { return nil }
func (*generic) ValidateScheduling(apis.SchedulingStrategy) []metav1.StatusCause         { return nil }
func (*generic) MutateGameServerPodSpec(*agonesv1.GameServerSpec, *corev1.PodSpec) error { return nil }

// SetEviction sets disruptions controls based on GameServer.Status.Eviction.
func (*generic) SetEviction(safe agonesv1.EvictionSafe, pod *corev1.Pod) error {
	if !runtime.FeatureEnabled(runtime.FeatureSafeToEvict) {
		return nil
	}
	if _, exists := pod.ObjectMeta.Annotations[agonesv1.PodSafeToEvictAnnotation]; !exists {
		switch safe {
		case agonesv1.EvictionSafeAlways:
			pod.ObjectMeta.Annotations[agonesv1.PodSafeToEvictAnnotation] = agonesv1.True
		case agonesv1.EvictionSafeOnUpgrade, agonesv1.EvictionSafeNever:
			// For EvictionSafeOnUpgrade and EvictionSafeNever, we block Cluster Autoscaler.
			pod.ObjectMeta.Annotations[agonesv1.PodSafeToEvictAnnotation] = agonesv1.False
		default:
			return errors.Errorf("unknown eviction.safe value %q", string(safe))
		}
	}
	if _, exists := pod.ObjectMeta.Labels[agonesv1.SafeToEvictLabel]; !exists {
		switch safe {
		case agonesv1.EvictionSafeAlways, agonesv1.EvictionSafeOnUpgrade:
			// For EvictionSafeAlways and EvictionSafeOnUpgrade, we use a label value
			// that does not match the agones-gameserver-safe-to-evict-false PDB. But
			// we go ahead and label it, in case someone wants to adopt custom logic
			// for this group of game servers.
			pod.ObjectMeta.Labels[agonesv1.SafeToEvictLabel] = agonesv1.True
		case agonesv1.EvictionSafeNever:
			// For EvictionSafeNever, match gones-gameserver-safe-to-evict-false PDB.
			pod.ObjectMeta.Labels[agonesv1.SafeToEvictLabel] = agonesv1.False
		default:
			return errors.Errorf("unknown eviction.safe value %q", string(safe))
		}
	}
	return nil
}

func (*generic) SyncPodPortsToGameServer(*agonesv1.GameServer, *corev1.Pod) error { return nil }

func (*generic) NewPortAllocator(minPort, maxPort int32,
	kubeInformerFactory informers.SharedInformerFactory,
	agonesInformerFactory externalversions.SharedInformerFactory) portallocator.Interface {
	return portallocator.New(minPort, maxPort, kubeInformerFactory, agonesInformerFactory)
}

func (*generic) WaitOnFreePorts() bool { return false }
