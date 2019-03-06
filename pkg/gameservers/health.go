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
	"strings"

	"agones.dev/agones/pkg/apis/stable"
	"agones.dev/agones/pkg/apis/stable/v1alpha1"
	"agones.dev/agones/pkg/client/clientset/versioned"
	getterv1alpha1 "agones.dev/agones/pkg/client/clientset/versioned/typed/stable/v1alpha1"
	"agones.dev/agones/pkg/client/informers/externalversions"
	listerv1alpha1 "agones.dev/agones/pkg/client/listers/stable/v1alpha1"
	"agones.dev/agones/pkg/util/logfields"
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
// an Unhealthy state if certain pods crash, or can't be assigned a port, and other
// similar type conditions.
type HealthController struct {
	baseLogger       *logrus.Entry
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

	hc.baseLogger = runtime.NewLoggerWithType(hc)
	hc.workerqueue = workerqueue.NewWorkerQueue(hc.syncGameServer, hc.baseLogger, logfields.GameServerKey, stable.GroupName+".HealthController")

	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(hc.baseLogger.Infof)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: kubeClient.CoreV1().Events("")})
	hc.recorder = eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: "health-controller"})

	podInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		UpdateFunc: func(oldObj, newObj interface{}) {
			pod := newObj.(*corev1.Pod)
			if owner := metav1.GetControllerOf(pod); owner != nil && owner.Kind == "GameServer" {
				if v1alpha1.GameServerRolePodSelector.Matches(labels.Set(pod.Labels)) && hc.isUnhealthy(pod) {
					key := pod.ObjectMeta.Namespace + "/" + owner.Name
					hc.workerqueue.Enqueue(cache.ExplicitKey(key))
				}
			}
		},
	})
	return hc
}

// isUnhealthy returns if the Pod event is going
// to cause the GameServer to become Unhealthy
func (hc *HealthController) isUnhealthy(pod *corev1.Pod) bool {
	return hc.unschedulableWithNoFreePorts(pod) || hc.failedContainer(pod)
}

// unschedulableWithNoFreePorts checks if the reason the Pod couldn't be scheduled
// was because there weren't any free ports in the range specified
func (hc *HealthController) unschedulableWithNoFreePorts(pod *corev1.Pod) bool {
	for _, cond := range pod.Status.Conditions {
		if cond.Type == corev1.PodScheduled && cond.Reason == corev1.PodReasonUnschedulable {
			if strings.Contains(cond.Message, "free ports") {
				return true
			}
		}
	}
	return false
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

func (hc *HealthController) loggerForGameServerKey(key string) *logrus.Entry {
	return logfields.AugmentLogEntry(hc.baseLogger, logfields.GameServerKey, key)
}

func (hc *HealthController) loggerForGameServer(gs *v1alpha1.GameServer) *logrus.Entry {
	gsName := "NilGameServer"
	if gs != nil {
		gsName = gs.Namespace + "/" + gs.Name
	}
	return hc.loggerForGameServerKey(gsName).WithField("gs", gs)
}

// syncGameServer sets the GameSerer to Unhealthy, if its state is Ready
func (hc *HealthController) syncGameServer(key string) error {
	hc.loggerForGameServerKey(key).Info("Synchronising")

	// Convert the namespace/name string into a distinct namespace and name
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		// don't return an error, as we don't want this retried
		runtime.HandleError(hc.loggerForGameServerKey(key), errors.Wrapf(err, "invalid resource key"))
		return nil
	}

	gs, err := hc.gameServerLister.GameServers(namespace).Get(name)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			hc.loggerForGameServerKey(key).Info("GameServer is no longer available for syncing")
			return nil
		}
		return errors.Wrapf(err, "error retrieving GameServer %s from namespace %s", name, namespace)
	}

	var reason string
	unhealthy := false

	switch gs.Status.State {

	case v1alpha1.GameServerStateStarting:
		hc.loggerForGameServer(gs).Info("GameServer cannot start on this port")
		unhealthy = true
		reason = "No nodes have free ports for the allocated ports"

	case v1alpha1.GameServerStateReady:
		hc.loggerForGameServer(gs).Info("GameServer container has terminated")
		unhealthy = true
		reason = "GameServer container terminated"
	}

	if unhealthy {
		hc.loggerForGameServer(gs).Infof("Marking GameServer as GameServerStateUnhealthy")
		gsCopy := gs.DeepCopy()
		gsCopy.Status.State = v1alpha1.GameServerStateUnhealthy

		if _, err := hc.gameServerGetter.GameServers(gs.ObjectMeta.Namespace).Update(gsCopy); err != nil {
			return errors.Wrapf(err, "error updating GameServer %s to unhealthy", gs.ObjectMeta.Name)
		}

		hc.recorder.Event(gs, corev1.EventTypeWarning, string(gsCopy.Status.State), reason)
	}

	return nil
}
