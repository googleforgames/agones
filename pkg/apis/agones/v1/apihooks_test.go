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

package v1

import (
	"testing"

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
		safeToEvict  EvictionSafe
		pod          *corev1.Pod
		wantPod      *corev1.Pod
	}{
		"SafeToEvict feature gate disabled => no change": {
			featureFlags: "SafeToEvict=false",
			// intentionally leave pod nil, it'll crash if anything's touched.
		},
		"SafeToEvict: Always, no incoming labels/annotations": {
			featureFlags: "SafeToEvict=true",
			safeToEvict:  EvictionSafeAlways,
			pod:          emptyPodAnd(func(*corev1.Pod) {}),
			wantPod: emptyPodAnd(func(pod *corev1.Pod) {
				pod.ObjectMeta.Annotations[PodSafeToEvictAnnotation] = True
				pod.ObjectMeta.Labels[SafeToEvictLabel] = True
			}),
		},
		"SafeToEvict: OnUpgrade, no incoming labels/annotations": {
			featureFlags: "SafeToEvict=true",
			safeToEvict:  EvictionSafeOnUpgrade,
			pod:          emptyPodAnd(func(*corev1.Pod) {}),
			wantPod: emptyPodAnd(func(pod *corev1.Pod) {
				pod.ObjectMeta.Annotations[PodSafeToEvictAnnotation] = False
				pod.ObjectMeta.Labels[SafeToEvictLabel] = True
			}),
		},
		"SafeToEvict: Never, no incoming labels/annotations": {
			featureFlags: "SafeToEvict=true",
			safeToEvict:  EvictionSafeNever,
			pod:          emptyPodAnd(func(*corev1.Pod) {}),
			wantPod: emptyPodAnd(func(pod *corev1.Pod) {
				pod.ObjectMeta.Annotations[PodSafeToEvictAnnotation] = False
				pod.ObjectMeta.Labels[SafeToEvictLabel] = False
			}),
		},
		"SafeToEvict: Always, incoming labels/annotations": {
			featureFlags: "SafeToEvict=true",
			safeToEvict:  EvictionSafeAlways,
			pod: emptyPodAnd(func(pod *corev1.Pod) {
				pod.ObjectMeta.Annotations[PodSafeToEvictAnnotation] = "just don't touch, ok?"
				pod.ObjectMeta.Labels[SafeToEvictLabel] = "seriously, leave it"
			}),
			wantPod: emptyPodAnd(func(pod *corev1.Pod) {
				pod.ObjectMeta.Annotations[PodSafeToEvictAnnotation] = "just don't touch, ok?"
				pod.ObjectMeta.Labels[SafeToEvictLabel] = "seriously, leave it"
			}),
		},
		"SafeToEvict: OnUpgrade, incoming labels/annotations": {
			featureFlags: "SafeToEvict=true",
			safeToEvict:  EvictionSafeOnUpgrade,
			pod: emptyPodAnd(func(pod *corev1.Pod) {
				pod.ObjectMeta.Annotations[PodSafeToEvictAnnotation] = "better not touch"
				pod.ObjectMeta.Labels[SafeToEvictLabel] = "not another one"
			}),
			wantPod: emptyPodAnd(func(pod *corev1.Pod) {
				pod.ObjectMeta.Annotations[PodSafeToEvictAnnotation] = "better not touch"
				pod.ObjectMeta.Labels[SafeToEvictLabel] = "not another one"
			}),
		},
		"SafeToEvict: Never, incoming labels/annotations": {
			featureFlags: "SafeToEvict=true",
			safeToEvict:  EvictionSafeNever,
			pod: emptyPodAnd(func(pod *corev1.Pod) {
				pod.ObjectMeta.Annotations[PodSafeToEvictAnnotation] = "a passthrough"
				pod.ObjectMeta.Labels[SafeToEvictLabel] = "or is it passthru?"
			}),
			wantPod: emptyPodAnd(func(pod *corev1.Pod) {
				pod.ObjectMeta.Annotations[PodSafeToEvictAnnotation] = "a passthrough"
				pod.ObjectMeta.Labels[SafeToEvictLabel] = "or is it passthru?"
			}),
		},
	} {
		t.Run(desc, func(t *testing.T) {
			err := runtime.ParseFeatures(tc.featureFlags)
			assert.NoError(t, err)

			err = (generic{}).SetEviction(tc.safeToEvict, tc.pod)
			assert.NoError(t, err)
			assert.Equal(t, tc.wantPod, tc.pod)
		})
	}
}
