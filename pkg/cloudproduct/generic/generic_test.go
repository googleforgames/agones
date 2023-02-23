// Copyright 2023 Google LLC All Rights Reserved.
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
package generic

import (
	"testing"

	"agones.dev/agones/pkg/apis"
	"agones.dev/agones/pkg/apis/agones"
	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	"agones.dev/agones/pkg/util/runtime"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestSetEviction(t *testing.T) {
	runtime.FeatureTestMutex.Lock()
	defer runtime.FeatureTestMutex.Unlock()

	emptyPodAnd := func(f func(*corev1.Pod)) *corev1.Pod {
		pod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{},
				Labels:      map[string]string{},
			},
		}
		f(pod)
		return pod
	}
	for desc, tc := range map[string]struct {
		featureFlags string
		eviction     *agonesv1.Eviction
		pod          *corev1.Pod
		wantPod      *corev1.Pod
	}{
		"SafeToEvict feature gate disabled => no change": {
			featureFlags: "SafeToEvict=false",
			// intentionally leave pod nil, it'll crash if anything's touched.
		},
		"SafeToEvict: Always, no incoming labels/annotations": {
			featureFlags: "SafeToEvict=true",
			eviction:     &agonesv1.Eviction{Safe: agonesv1.EvictionSafeAlways},
			pod:          emptyPodAnd(func(*corev1.Pod) {}),
			wantPod: emptyPodAnd(func(pod *corev1.Pod) {
				pod.ObjectMeta.Annotations[agonesv1.PodSafeToEvictAnnotation] = agonesv1.True
				pod.ObjectMeta.Labels[agonesv1.SafeToEvictLabel] = agonesv1.True
			}),
		},
		"SafeToEvict: OnUpgrade, no incoming labels/annotations": {
			featureFlags: "SafeToEvict=true",
			eviction:     &agonesv1.Eviction{Safe: agonesv1.EvictionSafeOnUpgrade},
			pod:          emptyPodAnd(func(*corev1.Pod) {}),
			wantPod: emptyPodAnd(func(pod *corev1.Pod) {
				pod.ObjectMeta.Annotations[agonesv1.PodSafeToEvictAnnotation] = agonesv1.False
				pod.ObjectMeta.Labels[agonesv1.SafeToEvictLabel] = agonesv1.True
			}),
		},
		"SafeToEvict: Never, no incoming labels/annotations": {
			featureFlags: "SafeToEvict=true",
			eviction:     &agonesv1.Eviction{Safe: agonesv1.EvictionSafeNever},
			pod:          emptyPodAnd(func(*corev1.Pod) {}),
			wantPod: emptyPodAnd(func(pod *corev1.Pod) {
				pod.ObjectMeta.Annotations[agonesv1.PodSafeToEvictAnnotation] = agonesv1.False
				pod.ObjectMeta.Labels[agonesv1.SafeToEvictLabel] = agonesv1.False
			}),
		},
		"SafeToEvict: Always, incoming labels/annotations": {
			featureFlags: "SafeToEvict=true",
			eviction:     &agonesv1.Eviction{Safe: agonesv1.EvictionSafeAlways},
			pod: emptyPodAnd(func(pod *corev1.Pod) {
				pod.ObjectMeta.Annotations[agonesv1.PodSafeToEvictAnnotation] = "just don't touch, ok?"
				pod.ObjectMeta.Labels[agonesv1.SafeToEvictLabel] = "seriously, leave it"
			}),
			wantPod: emptyPodAnd(func(pod *corev1.Pod) {
				pod.ObjectMeta.Annotations[agonesv1.PodSafeToEvictAnnotation] = "just don't touch, ok?"
				pod.ObjectMeta.Labels[agonesv1.SafeToEvictLabel] = "seriously, leave it"
			}),
		},
		"SafeToEvict: OnUpgrade, incoming labels/annotations": {
			featureFlags: "SafeToEvict=true",
			eviction:     &agonesv1.Eviction{Safe: agonesv1.EvictionSafeOnUpgrade},
			pod: emptyPodAnd(func(pod *corev1.Pod) {
				pod.ObjectMeta.Annotations[agonesv1.PodSafeToEvictAnnotation] = "better not touch"
				pod.ObjectMeta.Labels[agonesv1.SafeToEvictLabel] = "not another one"
			}),
			wantPod: emptyPodAnd(func(pod *corev1.Pod) {
				pod.ObjectMeta.Annotations[agonesv1.PodSafeToEvictAnnotation] = "better not touch"
				pod.ObjectMeta.Labels[agonesv1.SafeToEvictLabel] = "not another one"
			}),
		},
		"SafeToEvict: Never, incoming labels/annotations": {
			featureFlags: "SafeToEvict=true",
			eviction:     &agonesv1.Eviction{Safe: agonesv1.EvictionSafeNever},
			pod: emptyPodAnd(func(pod *corev1.Pod) {
				pod.ObjectMeta.Annotations[agonesv1.PodSafeToEvictAnnotation] = "a passthrough"
				pod.ObjectMeta.Labels[agonesv1.SafeToEvictLabel] = "or is it passthru?"
			}),
			wantPod: emptyPodAnd(func(pod *corev1.Pod) {
				pod.ObjectMeta.Annotations[agonesv1.PodSafeToEvictAnnotation] = "a passthrough"
				pod.ObjectMeta.Labels[agonesv1.SafeToEvictLabel] = "or is it passthru?"
			}),
		},
	} {
		t.Run(desc, func(t *testing.T) {
			err := runtime.ParseFeatures(tc.featureFlags)
			assert.NoError(t, err)

			err = (&generic{}).SetEviction(tc.eviction, tc.pod)
			assert.NoError(t, err)
			assert.Equal(t, tc.wantPod, tc.pod)
		})
	}
}

func TestGameServerPodAutoscalerAnnotations(t *testing.T) {
	runtime.FeatureTestMutex.Lock()
	defer runtime.FeatureTestMutex.Unlock()

	testCases := []struct {
		featureFlags       string
		description        string
		scheduling         apis.SchedulingStrategy
		setAnnotation      bool
		expectedAnnotation string
	}{
		{
			featureFlags:       "SafeToEvict=false",
			description:        "Packed, SafeToEvict=false",
			scheduling:         apis.Packed,
			expectedAnnotation: "false",
		},
		{
			featureFlags:       "SafeToEvict=false",
			description:        "Distributed, SafeToEvict=false",
			scheduling:         apis.Distributed,
			expectedAnnotation: "",
		},
		{
			featureFlags:       "SafeToEvict=false",
			description:        "Packed with autoscaler annotation, SafeToEvict=false",
			scheduling:         apis.Packed,
			setAnnotation:      true,
			expectedAnnotation: "true",
		},
		{
			featureFlags:       "SafeToEvict=false",
			description:        "Distributed with autoscaler annotation, SafeToEvict=false",
			scheduling:         apis.Distributed,
			setAnnotation:      true,
			expectedAnnotation: "true",
		},
		{
			featureFlags:       "SafeToEvict=true",
			description:        "Packed, SafeToEvict=true",
			scheduling:         apis.Packed,
			expectedAnnotation: "false",
		},
		{
			featureFlags:       "SafeToEvict=true",
			description:        "Distributed, SafeToEvict=true",
			scheduling:         apis.Distributed,
			expectedAnnotation: "false",
		},
		{
			featureFlags:       "SafeToEvict=true",
			description:        "Packed with autoscaler annotation, SafeToEvict=true",
			scheduling:         apis.Packed,
			setAnnotation:      true,
			expectedAnnotation: "true",
		},
		{
			featureFlags:       "SafeToEvict=true",
			description:        "Distributed with autoscaler annotation, SafeToEvict=true",
			scheduling:         apis.Distributed,
			setAnnotation:      true,
			expectedAnnotation: "true",
		},
	}

	fixture := &agonesv1.GameServer{
		ObjectMeta: metav1.ObjectMeta{Name: "logan"},
		Spec:       agonesv1.GameServerSpec{Container: "sheep"},
		Status:     agonesv1.GameServerStatus{Eviction: &agonesv1.Eviction{Safe: agonesv1.EvictionSafeNever}},
	}
	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			err := runtime.ParseFeatures(tc.featureFlags)
			assert.NoError(t, err)

			gs := fixture.DeepCopy()
			gs.Spec.Scheduling = tc.scheduling
			if tc.setAnnotation {
				gs.Spec.Template = corev1.PodTemplateSpec{ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{agonesv1.PodSafeToEvictAnnotation: "true"},
				}}
			}
			pod, err := gs.Pod(&generic{})
			assert.Nil(t, err, "Pod should not return an error")
			assert.Equal(t, gs.ObjectMeta.Name, pod.ObjectMeta.Name)
			assert.Equal(t, gs.ObjectMeta.Namespace, pod.ObjectMeta.Namespace)
			assert.Equal(t, agonesv1.GameServerLabelRole, pod.ObjectMeta.Labels[agonesv1.RoleLabel])
			assert.Equal(t, "gameserver", pod.ObjectMeta.Labels[agones.GroupName+"/role"])
			assert.Equal(t, gs.ObjectMeta.Name, pod.ObjectMeta.Labels[agonesv1.GameServerPodLabel])
			assert.Equal(t, "sheep", pod.ObjectMeta.Annotations[agonesv1.GameServerContainerAnnotation])
			assert.True(t, metav1.IsControlledBy(pod, gs))
			assert.Equal(t, tc.expectedAnnotation, pod.ObjectMeta.Annotations[agonesv1.PodSafeToEvictAnnotation])
		})
	}
}
