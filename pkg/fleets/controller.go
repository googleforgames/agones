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

package fleets

import (
	"agones.dev/agones/pkg/apis/stable"
	stablev1alpha1 "agones.dev/agones/pkg/apis/stable/v1alpha1"
	"agones.dev/agones/pkg/client/clientset/versioned"
	getterv1alpha1 "agones.dev/agones/pkg/client/clientset/versioned/typed/stable/v1alpha1"
	"agones.dev/agones/pkg/client/informers/externalversions"
	listerv1alpha1 "agones.dev/agones/pkg/client/listers/stable/v1alpha1"
	"agones.dev/agones/pkg/util/crd"
	"agones.dev/agones/pkg/util/runtime"
	"agones.dev/agones/pkg/util/workerqueue"
	"github.com/heptiolabs/healthcheck"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	extclientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/typed/apiextensions/v1beta1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
)

// Controller is a the GameServerSet controller
type Controller struct {
	logger              *logrus.Entry
	crdGetter           v1beta1.CustomResourceDefinitionInterface
	gameServerSetGetter getterv1alpha1.GameServerSetsGetter
	gameServerSetLister listerv1alpha1.GameServerSetLister
	gameServerSetSynced cache.InformerSynced
	fleetGetter         getterv1alpha1.FleetsGetter
	fleetLister         listerv1alpha1.FleetLister
	fleetSynced         cache.InformerSynced
	workerqueue         *workerqueue.WorkerQueue
	recorder            record.EventRecorder
}

// NewController returns a new fleets crd controller
func NewController(
	health healthcheck.Handler,
	kubeClient kubernetes.Interface,
	extClient extclientset.Interface,
	agonesClient versioned.Interface,
	agonesInformerFactory externalversions.SharedInformerFactory) *Controller {

	gameServerSets := agonesInformerFactory.Stable().V1alpha1().GameServerSets()
	gsSetInformer := gameServerSets.Informer()

	fleets := agonesInformerFactory.Stable().V1alpha1().Fleets()
	fInformer := fleets.Informer()

	c := &Controller{
		crdGetter:           extClient.ApiextensionsV1beta1().CustomResourceDefinitions(),
		gameServerSetGetter: agonesClient.StableV1alpha1(),
		gameServerSetLister: gameServerSets.Lister(),
		gameServerSetSynced: gsSetInformer.HasSynced,
		fleetGetter:         agonesClient.StableV1alpha1(),
		fleetLister:         fleets.Lister(),
		fleetSynced:         fInformer.HasSynced,
	}

	c.logger = runtime.NewLoggerWithType(c)
	c.workerqueue = workerqueue.NewWorkerQueue(c.syncFleet, c.logger, stable.GroupName+".FleetController")
	health.AddLivenessCheck("fleet-workerqueue", healthcheck.Check(c.workerqueue.Healthy))

	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(c.logger.Infof)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: kubeClient.CoreV1().Events("")})
	c.recorder = eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: "fleet-controller"})

	fInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: c.workerqueue.Enqueue,
		UpdateFunc: func(_, newObj interface{}) {
			c.workerqueue.Enqueue(newObj)
		},
	})

	gsSetInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: c.gameServerSetEventHandler,
		UpdateFunc: func(_, newObj interface{}) {
			gsSet := newObj.(*stablev1alpha1.GameServerSet)
			// ignore if already being deleted
			if gsSet.ObjectMeta.DeletionTimestamp.IsZero() {
				c.gameServerSetEventHandler(gsSet)
			}
		},
	})

	return c
}

// Run the Fleet controller. Will block until stop is closed.
// Runs threadiness number workers to process the rate limited queue
func (c *Controller) Run(threadiness int, stop <-chan struct{}) error {
	err := crd.WaitForEstablishedCRD(c.crdGetter, "fleets.stable.agones.dev", c.logger)
	if err != nil {
		return err
	}

	c.logger.Info("Wait for cache sync")
	if !cache.WaitForCacheSync(stop, c.gameServerSetSynced, c.fleetSynced) {
		return errors.New("failed to wait for caches to sync")
	}

	c.workerqueue.Run(threadiness, stop)
	return nil
}

// gameServerSetEventHandler enqueues the owning Fleet for this GameServerSet,
// assuming that it has one
func (c *Controller) gameServerSetEventHandler(obj interface{}) {
	gsSet := obj.(*stablev1alpha1.GameServerSet)
	ref := metav1.GetControllerOf(gsSet)
	if ref == nil {
		return
	}

	fleet, err := c.fleetLister.Fleets(gsSet.ObjectMeta.Namespace).Get(ref.Name)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			c.logger.WithField("ref", ref).Info("Owner Fleet no longer available for syncing")
		} else {
			runtime.HandleError(c.logger.WithField("fleet", fleet.ObjectMeta.Name).WithField("ref", ref),
				errors.Wrap(err, "error retrieving GameServerSet owner"))
		}
		return
	}
	c.workerqueue.Enqueue(fleet)
}

// syncFleet synchronised the fleet CRDs and configures/updates
// backing GameServerSets
func (c *Controller) syncFleet(key string) error {
	c.logger.WithField("key", key).Info("Synchronising")

	// Convert the namespace/name string into a distinct namespace and name
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		// don't return an error, as we don't want this retried
		runtime.HandleError(c.logger.WithField("key", key), errors.Wrapf(err, "invalid resource key"))
		return nil
	}

	fleet, err := c.fleetLister.Fleets(namespace).Get(name)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			c.logger.WithField("key", key).Info("Fleet is no longer available for syncing")
			return nil
		}
		return errors.Wrapf(err, "error retrieving fleet %s from namespace %s", name, namespace)
	}

	list, err := c.listGameServerSets(fleet)
	if err != nil {
		return err
	}

	var activeGsSet *stablev1alpha1.GameServerSet

	// if there isn't a GameServerSet yet, then create one
	if len(list) == 0 {
		activeGsSet = fleet.GameServerSet()
		activeGsSet, err = c.gameServerSetGetter.GameServerSets(fleet.ObjectMeta.Namespace).Create(activeGsSet)
		if err != nil {
			return errors.Wrapf(err, "error creating gameserverset for fleet %s", fleet.ObjectMeta.Name)
		}

		c.recorder.Eventf(fleet, corev1.EventTypeNormal, "CreatingGameServerSet",
			"Created GameServerSet %s", activeGsSet.ObjectMeta.Name)

	} else {
		// for now, we're ignoring any change to the template - will handle on the next PR
		// therefore, we are going to assume for the moment, that there is only ever one
		// GameServerSet for a Fleet - we will handle multiple GameServerSets in the next PR
		activeGsSet = list[0]
	}

	// if the replica count has changed, then update the GameServerSet
	if fleet.Spec.Replicas != activeGsSet.Spec.Replicas {
		gsSetCopy := activeGsSet.DeepCopy()
		gsSetCopy.Spec.Replicas = fleet.Spec.Replicas

		if gsSetCopy, err = c.gameServerSetGetter.GameServerSets(fleet.ObjectMeta.Namespace).Update(gsSetCopy); err != nil {
			return errors.Wrapf(err, "error updating replicas for gameserverset for fleet %s", fleet.ObjectMeta.Name)
		}
		c.recorder.Eventf(fleet, corev1.EventTypeNormal, "ScalingGameServerSet",
			"Scaling GameServerSet %s from %d to %d", gsSetCopy.ObjectMeta.Name, activeGsSet.Spec.Replicas, gsSetCopy.Spec.Replicas)
	}

	return c.updateFleetStatus(fleet)
}

// listGameServerSets lists all the GameServerSets for a given
// Fleet
func (c *Controller) listGameServerSets(f *stablev1alpha1.Fleet) ([]*stablev1alpha1.GameServerSet, error) {
	list, err := c.gameServerSetLister.List(labels.SelectorFromSet(labels.Set{stablev1alpha1.FleetGameServerSetLabel: f.ObjectMeta.Name}))
	if err != nil {
		return list, errors.Wrapf(err, "error listing gameserversets for fleet %s", f.ObjectMeta.Name)
	}

	var result []*stablev1alpha1.GameServerSet
	for _, gsSet := range list {
		if metav1.IsControlledBy(gsSet, f) {
			result = append(result, gsSet)
		}
	}

	return result, nil
}

// updateFleetStatus gets the GameServerSets for this Fleet and then
// calculates the counts for the status, and updates the Fleet
func (c *Controller) updateFleetStatus(fleet *stablev1alpha1.Fleet) error {
	list, err := c.listGameServerSets(fleet)
	if err != nil {
		return err
	}

	fCopy := fleet.DeepCopy()
	fCopy.Status.Replicas = 0
	fCopy.Status.ReadyReplicas = 0

	for _, gsSet := range list {
		fCopy.Status.Replicas += gsSet.Status.Replicas
		fCopy.Status.ReadyReplicas += gsSet.Status.ReadyReplicas
	}

	_, err = c.fleetGetter.Fleets(fCopy.Namespace).Update(fCopy)
	return errors.Wrapf(err, "error updating status of fleet %s", fCopy.ObjectMeta.Name)
}
