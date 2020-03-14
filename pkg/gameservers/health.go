// Copyright 2018 Google LLC All Rights Reserved.
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

	"agones.dev/agones/pkg/apis/agones"
	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	"agones.dev/agones/pkg/client/clientset/versioned"
	getterv1 "agones.dev/agones/pkg/client/clientset/versioned/typed/agones/v1"
	"agones.dev/agones/pkg/client/informers/externalversions"
	listerv1 "agones.dev/agones/pkg/client/listers/agones/v1"
	"agones.dev/agones/pkg/util/logfields"
	"agones.dev/agones/pkg/util/runtime"
	"agones.dev/agones/pkg/util/workerqueue"
	"github.com/heptiolabs/healthcheck"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	gameServerSynced cache.InformerSynced
	gameServerGetter getterv1.GameServersGetter
	gameServerLister listerv1.GameServerLister
	workerqueue      *workerqueue.WorkerQueue
	recorder         record.EventRecorder
}

// NewHealthController returns a HealthController
func NewHealthController(health healthcheck.Handler,
	kubeClient kubernetes.Interface,
	agonesClient versioned.Interface,
	kubeInformerFactory informers.SharedInformerFactory,
	agonesInformerFactory externalversions.SharedInformerFactory) *HealthController {

	podInformer := kubeInformerFactory.Core().V1().Pods().Informer()
	gameserverInformer := agonesInformerFactory.Agones().V1().GameServers()
	hc := &HealthController{
		podSynced:        podInformer.HasSynced,
		podLister:        kubeInformerFactory.Core().V1().Pods().Lister(),
		gameServerSynced: gameserverInformer.Informer().HasSynced,
		gameServerGetter: agonesClient.AgonesV1(),
		gameServerLister: gameserverInformer.Lister(),
	}

	hc.baseLogger = runtime.NewLoggerWithType(hc)
	hc.workerqueue = workerqueue.NewWorkerQueue(hc.syncGameServer, hc.baseLogger, logfields.GameServerKey, agones.GroupName+".HealthController")
	health.AddLivenessCheck("gameserver-health-workerqueue", healthcheck.Check(hc.workerqueue.Healthy))

	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(hc.baseLogger.Debugf)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: kubeClient.CoreV1().Events("")})
	hc.recorder = eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: "health-controller"})

	podInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		UpdateFunc: func(oldObj, newObj interface{}) {
			pod := newObj.(*corev1.Pod)
			if isGameServerPod(pod) && hc.isUnhealthy(pod) {
				hc.workerqueue.Enqueue(pod)
			}
		},
		DeleteFunc: func(obj interface{}) {
			// Could be a DeletedFinalStateUnknown, in which case, just ignore it
			pod, ok := obj.(*corev1.Pod)
			if ok && isGameServerPod(pod) {
				hc.workerqueue.Enqueue(pod)
			}
		},
	})
	return hc
}

// isUnhealthy returns if the Pod event is going
// to cause the GameServer to become Unhealthy
func (hc *HealthController) isUnhealthy(pod *corev1.Pod) bool {
	return hc.evictedPod(pod) || hc.unschedulableWithNoFreePorts(pod) || hc.failedContainer(pod)
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

// evictedPod checks if the Pod was Evicted
// could be caused by reaching limit on Ephemeral storage
func (hc *HealthController) evictedPod(pod *corev1.Pod) bool {
	return pod.Status.Reason == "Evicted"
}

// failedContainer checks each container, and determines if there was a failed
// container
func (hc *HealthController) failedContainer(pod *corev1.Pod) bool {
	container := pod.Annotations[agonesv1.GameServerContainerAnnotation]
	for _, cs := range pod.Status.ContainerStatuses {
		if cs.Name == container {
			// sometimes on a restart, the cs.State can be running and the last state will be merged
			return cs.State.Terminated != nil || cs.LastTerminationState.Terminated != nil
		}
	}
	return false
}

// Run processes the rate limited queue.
// Will block until stop is closed
func (hc *HealthController) Run(stop <-chan struct{}) error {
	hc.baseLogger.Debug("Wait for cache sync")
	if !cache.WaitForCacheSync(stop, hc.gameServerSynced, hc.podSynced) {
		return errors.New("failed to wait for caches to sync")
	}

	hc.workerqueue.Run(1, stop)

	return nil
}

func (hc *HealthController) loggerForGameServerKey(key string) *logrus.Entry {
	return logfields.AugmentLogEntry(hc.baseLogger, logfields.GameServerKey, key)
}

func (hc *HealthController) loggerForGameServer(gs *agonesv1.GameServer) *logrus.Entry {
	gsName := logfields.NilGameServer
	if gs != nil {
		gsName = gs.Namespace + "/" + gs.Name
	}
	return hc.loggerForGameServerKey(gsName).WithField("gs", gs)
}

// syncGameServer sets the GameSerer to Unhealthy, if its state is Ready
func (hc *HealthController) syncGameServer(key string) error {
	hc.loggerForGameServerKey(key).Debug("Synchronising")

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
			hc.loggerForGameServerKey(key).Debug("GameServer is no longer available for syncing")
			return nil
		}
		return errors.Wrapf(err, "error retrieving GameServer %s from namespace %s", name, namespace)
	}

	// at this point we don't care, we're already Unhealthy / deleting
	if gs.IsBeingDeleted() || gs.Status.State == agonesv1.GameServerStateUnhealthy {
		return nil
	}

	if skip, err := hc.skipUnhealthy(gs); err != nil || skip {
		return err
	}

	hc.loggerForGameServer(gs).Debug("Issue with GameServer pod, marking as GameServerStateUnhealthy")
	gsCopy := gs.DeepCopy()
	gsCopy.Status.State = agonesv1.GameServerStateUnhealthy

	if _, err := hc.gameServerGetter.GameServers(gs.ObjectMeta.Namespace).Update(gsCopy); err != nil {
		return errors.Wrapf(err, "error updating GameServer %s to unhealthy", gs.ObjectMeta.Name)
	}

	hc.recorder.Event(gs, corev1.EventTypeWarning, string(gsCopy.Status.State), "Issue with Gameserver pod")

	return nil
}

// skipUnhealthy determines if it's appropriate to not move to Unhealthy when a Pod's
// gameserver container has crashed, or let it restart as per usual K8s operations.
// It does this by checking a combination of the current GameServer state and annotation data that stores
// which container instance was live if the GameServer has been marked as Ready.
// The logic is as follows:
//   - If the GameServer is not yet Ready, allow to restart (return true)
//   - If the GameServer is in a state past Ready, move to Unhealthy
func (hc *HealthController) skipUnhealthy(gs *agonesv1.GameServer) (bool, error) {
	pod, err := hc.podLister.Pods(gs.ObjectMeta.Namespace).Get(gs.ObjectMeta.Name)
	if err != nil {
		// Pod doesn't exist, so the GameServer is definitely not healthy
		if k8serrors.IsNotFound(err) {
			return false, nil
		}
		// if it's something else, go back into the queue
		return false, errors.Wrapf(err, "error retrieving Pod %s for GameServer to check status", gs.ObjectMeta.Name)
	}
	if !metav1.IsControlledBy(pod, gs) {
		// This is not the Pod we are looking for 🤖
		return false, nil
	}
	if gs.IsBeforeReady() {
		return hc.failedContainer(pod), nil
	}

	// finally, we need to check if the failed container happened after the gameserver was ready or before.
	for _, cs := range pod.Status.ContainerStatuses {
		if cs.Name == gs.Spec.Container {
			if cs.State.Terminated != nil {
				return false, nil
			}
			if cs.LastTerminationState.Terminated != nil {
				// if the current container is running, and is the ready container, then we know this is some
				// other pod update, and we previously had a restart before we got to being Ready, and therefore
				// shouldn't move to Unhealthy.
				return cs.ContainerID == gs.Annotations[agonesv1.GameServerReadyContainerIDAnnotation], nil
			}
			break
		}
	}

	return false, nil
}
