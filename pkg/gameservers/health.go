// Copyright 2018 Google Inc. All Rights Reserved.
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

package gameservers

import (
	"agones.dev/agones/pkg/apis/stable"
	"agones.dev/agones/pkg/apis/stable/v1alpha1"
	"agones.dev/agones/pkg/client/clientset/versioned"
	getterv1alpha1 "agones.dev/agones/pkg/client/clientset/versioned/typed/stable/v1alpha1"
	"agones.dev/agones/pkg/client/informers/externalversions"
	listerv1alpha1 "agones.dev/agones/pkg/client/listers/stable/v1alpha1"
	"agones.dev/agones/pkg/util/runtime"
	"agones.dev/agones/pkg/util/workerqueue"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	corelisterv1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
)

// HealthController watches Pods, and applies
// an Unhealthy state if the GameServer main container exits when in
// a Ready state
type HealthController struct {
	logger           *logrus.Entry
	podSynced        cache.InformerSynced
	podLister        corelisterv1.PodLister
	gameServerGetter getterv1alpha1.GameServersGetter
	gameServerLister listerv1alpha1.GameServerLister
	workerqueue      *workerqueue.WorkerQueue
	recorder         record.EventRecorder
}

// NewHealthController returns a HealthController
func NewHealthController(kubeClient kubernetes.Interface, agonesClient versioned.Interface, kubeInformerFactory informers.SharedInformerFactory,
	agonesInformerFactory externalversions.SharedInformerFactory) *HealthController {

	podInformer := kubeInformerFactory.Core().V1().Pods().Informer()
	hc := &HealthController{
		podSynced:        podInformer.HasSynced,
		podLister:        kubeInformerFactory.Core().V1().Pods().Lister(),
		gameServerGetter: agonesClient.StableV1alpha1(),
		gameServerLister: agonesInformerFactory.Stable().V1alpha1().GameServers().Lister(),
	}

	hc.logger = runtime.NewLoggerWithType(hc)
	hc.workerqueue = workerqueue.NewWorkerQueue(hc.syncGameServer, hc.logger, stable.GroupName+".HealthController")

	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(hc.logger.Infof)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: kubeClient.CoreV1().Events("")})
	hc.recorder = eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: "health-controller"})

	podInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		UpdateFunc: func(oldObj, newObj interface{}) {
			pod := newObj.(*corev1.Pod)
			if owner := metav1.GetControllerOf(pod); owner != nil && owner.Kind == "GameServer" {
				if v1alpha1.GameServerRolePodSelector.Matches(labels.Set(pod.Labels)) && hc.failedContainer(pod) {
					key := pod.ObjectMeta.Namespace + "/" + owner.Name
					hc.logger.WithField("key", key).Info("GameServer container has terminated")
					hc.workerqueue.Enqueue(cache.ExplicitKey(key))
				}
			}
		},
	})
	return hc
}

// failedContainer checks each container, and determines if there was a failed
// container
func (hc *HealthController) failedContainer(pod *corev1.Pod) bool {
	container := pod.Annotations[v1alpha1.GameServerContainerAnnotation]
	for _, cs := range pod.Status.ContainerStatuses {
		if cs.Name == container && cs.State.Terminated != nil {
			return true
		}
	}
	return false

}

// Run processes the rate limited queue.
// Will block until stop is closed
func (hc *HealthController) Run(stop <-chan struct{}) {
	hc.workerqueue.Run(1, stop)
}

// syncGameServer sets the GameSerer to Unhealthy, if its state is Ready
func (hc *HealthController) syncGameServer(key string) error {
	hc.logger.WithField("key", key).Info("Synchronising")

	// Convert the namespace/name string into a distinct namespace and name
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		// don't return an error, as we don't want this retried
		runtime.HandleError(hc.logger.WithField("key", key), errors.Wrapf(err, "invalid resource key"))
		return nil
	}

	gs, err := hc.gameServerLister.GameServers(namespace).Get(name)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			hc.logger.WithField("key", key).Info("GameServer is no longer available for syncing")
			return nil
		}
		return errors.Wrapf(err, "error retrieving GameServer %s from namespace %s", name, namespace)
	}

	if gs.Status.State == v1alpha1.Ready {
		hc.logger.WithField("gs", gs).Infof("Marking GameServer as Unhealthy")
		gsCopy := gs.DeepCopy()
		gsCopy.Status.State = v1alpha1.Unhealthy

		if _, err := hc.gameServerGetter.GameServers(gs.ObjectMeta.Namespace).Update(gsCopy); err != nil {
			return errors.Wrapf(err, "error updating GameServer %s to unhealthy", gs.ObjectMeta.Name)
		}

		hc.recorder.Event(gs, corev1.EventTypeWarning, string(gsCopy.Status.State), "GameServer container terminated")
	}

	return nil
}
