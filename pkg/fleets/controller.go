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

package fleets

import (
	"encoding/json"
	"fmt"
	"time"

	"agones.dev/agones/pkg/apis/agones"
	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	"agones.dev/agones/pkg/client/clientset/versioned"
	getterv1 "agones.dev/agones/pkg/client/clientset/versioned/typed/agones/v1"
	"agones.dev/agones/pkg/client/informers/externalversions"
	listerv1 "agones.dev/agones/pkg/client/listers/agones/v1"
	"agones.dev/agones/pkg/util/crd"
	"agones.dev/agones/pkg/util/logfields"
	"agones.dev/agones/pkg/util/runtime"
	"agones.dev/agones/pkg/util/webhooks"
	"agones.dev/agones/pkg/util/workerqueue"
	"github.com/heptiolabs/healthcheck"
	"github.com/mattbaird/jsonpatch"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	admv1beta1 "k8s.io/api/admission/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	extclientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/typed/apiextensions/v1beta1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
)

// Controller is a the GameServerSet controller
type Controller struct {
	baseLogger          *logrus.Entry
	crdGetter           v1beta1.CustomResourceDefinitionInterface
	gameServerSetGetter getterv1.GameServerSetsGetter
	gameServerSetLister listerv1.GameServerSetLister
	gameServerSetSynced cache.InformerSynced
	fleetGetter         getterv1.FleetsGetter
	fleetLister         listerv1.FleetLister
	fleetSynced         cache.InformerSynced
	workerqueue         *workerqueue.WorkerQueue
	recorder            record.EventRecorder
}

// NewController returns a new fleets crd controller
func NewController(
	wh *webhooks.WebHook,
	health healthcheck.Handler,
	kubeClient kubernetes.Interface,
	extClient extclientset.Interface,
	agonesClient versioned.Interface,
	agonesInformerFactory externalversions.SharedInformerFactory) *Controller {

	gameServerSets := agonesInformerFactory.Agones().V1().GameServerSets()
	gsSetInformer := gameServerSets.Informer()

	fleets := agonesInformerFactory.Agones().V1().Fleets()
	fInformer := fleets.Informer()

	c := &Controller{
		crdGetter:           extClient.ApiextensionsV1beta1().CustomResourceDefinitions(),
		gameServerSetGetter: agonesClient.AgonesV1(),
		gameServerSetLister: gameServerSets.Lister(),
		gameServerSetSynced: gsSetInformer.HasSynced,
		fleetGetter:         agonesClient.AgonesV1(),
		fleetLister:         fleets.Lister(),
		fleetSynced:         fInformer.HasSynced,
	}

	c.baseLogger = runtime.NewLoggerWithType(c)
	c.workerqueue = workerqueue.NewWorkerQueueWithRateLimiter(c.syncFleet, c.baseLogger, logfields.FleetKey, agones.GroupName+".FleetController", workerqueue.FastRateLimiter(3*time.Second))
	health.AddLivenessCheck("fleet-workerqueue", healthcheck.Check(c.workerqueue.Healthy))

	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(c.baseLogger.Debugf)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: kubeClient.CoreV1().Events("")})
	c.recorder = eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: "fleet-controller"})

	wh.AddHandler("/mutate", agonesv1.Kind("Fleet"), admv1beta1.Create, c.creationMutationHandler)
	wh.AddHandler("/validate", agonesv1.Kind("Fleet"), admv1beta1.Create, c.creationValidationHandler)
	wh.AddHandler("/validate", agonesv1.Kind("Fleet"), admv1beta1.Update, c.creationValidationHandler)

	fInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: c.workerqueue.Enqueue,
		UpdateFunc: func(_, newObj interface{}) {
			c.workerqueue.Enqueue(newObj)
		},
	})

	gsSetInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: c.gameServerSetEventHandler,
		UpdateFunc: func(_, newObj interface{}) {
			gsSet := newObj.(*agonesv1.GameServerSet)
			// ignore if already being deleted
			if gsSet.ObjectMeta.DeletionTimestamp.IsZero() {
				c.gameServerSetEventHandler(gsSet)
			}
		},
	})

	return c
}

// creationMutationHandler is the handler for the mutating webhook that sets the
// the default values on the Fleet
// Should only be called on fleet create operations.
// nolint:dupl
func (c *Controller) creationMutationHandler(review admv1beta1.AdmissionReview) (admv1beta1.AdmissionReview, error) {
	c.baseLogger.WithField("review", review).Debug("creationMutationHandler")

	obj := review.Request.Object
	fleet := &agonesv1.Fleet{}
	err := json.Unmarshal(obj.Raw, fleet)
	if err != nil {
		return review, errors.Wrapf(err, "error unmarshalling original Fleet json: %s", obj.Raw)
	}

	// This is the main logic of this function
	// the rest is really just json plumbing
	fleet.ApplyDefaults()

	newFleet, err := json.Marshal(fleet)
	if err != nil {
		return review, errors.Wrapf(err, "error marshalling default applied Fleet %s to json", fleet.ObjectMeta.Name)
	}

	patch, err := jsonpatch.CreatePatch(obj.Raw, newFleet)
	if err != nil {
		return review, errors.Wrapf(err, "error creating patch for Fleet %s", fleet.ObjectMeta.Name)
	}

	jsn, err := json.Marshal(patch)
	if err != nil {
		return review, errors.Wrapf(err, "error creating json for patch for Fleet %s", fleet.ObjectMeta.Name)
	}

	c.loggerForFleet(fleet).WithField("patch", string(jsn)).Infof("patch created!")

	pt := admv1beta1.PatchTypeJSONPatch
	review.Response.PatchType = &pt
	review.Response.Patch = jsn

	return review, nil
}

// creationValidationHandler that validates a Fleet when it is created
// Should only be called on Fleet create and Update operations.
func (c *Controller) creationValidationHandler(review admv1beta1.AdmissionReview) (admv1beta1.AdmissionReview, error) {
	c.baseLogger.WithField("review", review).Debug("creationValidationHandler")

	obj := review.Request.Object
	fleet := &agonesv1.Fleet{}
	err := json.Unmarshal(obj.Raw, fleet)
	if err != nil {
		return review, errors.Wrapf(err, "error unmarshalling original Fleet json: %s", obj.Raw)
	}

	causes, ok := fleet.Validate()
	if !ok {
		review.Response.Allowed = false
		details := metav1.StatusDetails{
			Name:   review.Request.Name,
			Group:  review.Request.Kind.Group,
			Kind:   review.Request.Kind.Kind,
			Causes: causes,
		}
		review.Response.Result = &metav1.Status{
			Status:  metav1.StatusFailure,
			Message: "Fleet configuration is invalid",
			Reason:  metav1.StatusReasonInvalid,
			Details: &details,
		}

		c.loggerForFleet(fleet).WithField("review", review).Warn("Invalid Fleet")
		return review, nil
	}

	return review, nil
}

// Run the Fleet controller. Will block until stop is closed.
// Runs threadiness number workers to process the rate limited queue
func (c *Controller) Run(workers int, stop <-chan struct{}) error {
	err := crd.WaitForEstablishedCRD(c.crdGetter, "fleets.agones.dev", c.baseLogger)
	if err != nil {
		return err
	}

	c.baseLogger.Debug("Wait for cache sync")
	if !cache.WaitForCacheSync(stop, c.gameServerSetSynced, c.fleetSynced) {
		return errors.New("failed to wait for caches to sync")
	}

	c.workerqueue.Run(workers, stop)
	return nil
}

func (c *Controller) loggerForFleetKey(key string) *logrus.Entry {
	return logfields.AugmentLogEntry(c.baseLogger, logfields.FleetKey, key)
}

func (c *Controller) loggerForFleet(f *agonesv1.Fleet) *logrus.Entry {
	fleetName := "NilFleet"
	if f != nil {
		fleetName = f.ObjectMeta.Namespace + "/" + f.ObjectMeta.Name
	}
	return c.loggerForFleetKey(fleetName).WithField("fleet", f)
}

// gameServerSetEventHandler enqueues the owning Fleet for this GameServerSet,
// assuming that it has one
func (c *Controller) gameServerSetEventHandler(obj interface{}) {
	gsSet := obj.(*agonesv1.GameServerSet)
	ref := metav1.GetControllerOf(gsSet)
	if ref == nil {
		return
	}

	fleet, err := c.fleetLister.Fleets(gsSet.ObjectMeta.Namespace).Get(ref.Name)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			c.baseLogger.WithField("ref", ref).Warn("Owner Fleet no longer available for syncing")
		} else {
			runtime.HandleError(c.loggerForFleet(fleet).WithField("ref", ref),
				errors.Wrap(err, "error retrieving GameServerSet owner"))
		}
		return
	}
	c.workerqueue.Enqueue(fleet)
}

// syncFleet synchronised the fleet CRDs and configures/updates
// backing GameServerSets
func (c *Controller) syncFleet(key string) error {
	c.loggerForFleetKey(key).Debug("Synchronising")

	// Convert the namespace/name string into a distinct namespace and name
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		// don't return an error, as we don't want this retried
		runtime.HandleError(c.loggerForFleetKey(key), errors.Wrapf(err, "invalid resource key"))
		return nil
	}

	fleet, err := c.fleetLister.Fleets(namespace).Get(name)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			c.loggerForFleetKey(key).Debug("Fleet is no longer available for syncing")
			return nil
		}
		return errors.Wrapf(err, "error retrieving fleet %s from namespace %s", name, namespace)
	}

	list, err := ListGameServerSetsByFleetOwner(c.gameServerSetLister, fleet)
	if err != nil {
		return err
	}

	active, rest := c.filterGameServerSetByActive(fleet, list)

	// if there isn't an active gameServerSet, create one (but don't persist yet)
	if active == nil {
		c.loggerForFleet(fleet).Debug("could not find active GameServerSet, creating")
		active = fleet.GameServerSet()
	}

	replicas, err := c.applyDeploymentStrategy(fleet, active, rest)
	if err != nil {
		return err
	}
	if err := c.deleteEmptyGameServerSets(fleet, rest); err != nil {
		return err
	}

	if err := c.upsertGameServerSet(fleet, active, replicas); err != nil {
		return err
	}
	return c.updateFleetStatus(fleet)
}

// upsertGameServerSet if the GameServerSet is new, insert it
// if the replicas do not match the active
// GameServerSet, then update it
func (c *Controller) upsertGameServerSet(fleet *agonesv1.Fleet, active *agonesv1.GameServerSet, replicas int32) error {
	if active.ObjectMeta.UID == "" {
		active.Spec.Replicas = replicas
		gsSets := c.gameServerSetGetter.GameServerSets(active.ObjectMeta.Namespace)
		gsSet, err := gsSets.Create(active)
		if err != nil {
			return errors.Wrapf(err, "error creating gameserverset for fleet %s", fleet.ObjectMeta.Name)
		}

		// extra step which is needed to set
		// default values for GameServerSet Status Subresource
		gsSetCopy := gsSet.DeepCopy()
		gsSetCopy.Status.ReadyReplicas = 0
		gsSetCopy.Status.Replicas = 0
		gsSetCopy.Status.AllocatedReplicas = 0
		_, err = gsSets.UpdateStatus(gsSetCopy)
		if err != nil {
			return errors.Wrapf(err, "error updating status of gameserverset for fleet %s",
				fleet.ObjectMeta.Name)
		}

		c.recorder.Eventf(fleet, corev1.EventTypeNormal, "CreatingGameServerSet",
			"Created GameServerSet %s", gsSet.ObjectMeta.Name)
		return nil
	}

	if replicas != active.Spec.Replicas || active.Spec.Scheduling != fleet.Spec.Scheduling {
		gsSetCopy := active.DeepCopy()
		gsSetCopy.Spec.Replicas = replicas
		gsSetCopy.Spec.Scheduling = fleet.Spec.Scheduling
		gsSetCopy, err := c.gameServerSetGetter.GameServerSets(fleet.ObjectMeta.Namespace).Update(gsSetCopy)
		if err != nil {
			return errors.Wrapf(err, "error updating replicas for gameserverset for fleet %s", fleet.ObjectMeta.Name)
		}
		c.recorder.Eventf(fleet, corev1.EventTypeNormal, "ScalingGameServerSet",
			"Scaling active GameServerSet %s from %d to %d", gsSetCopy.ObjectMeta.Name, active.Spec.Replicas, gsSetCopy.Spec.Replicas)
	}

	return nil
}

// applyDeploymentStrategy applies the Fleet > Spec > Deployment strategy to all the non-active
// GameServerSets that are passed in
func (c *Controller) applyDeploymentStrategy(fleet *agonesv1.Fleet, active *agonesv1.GameServerSet, rest []*agonesv1.GameServerSet) (int32, error) {
	// if there is nothing `rest`, then it's either brand Fleet, or we can just jump to the fleet value,
	// since there is nothing else scaling down at this point

	if len(rest) == 0 {
		return fleet.Spec.Replicas, nil
	}

	switch fleet.Spec.Strategy.Type {
	case appsv1.RecreateDeploymentStrategyType:
		return c.recreateDeployment(fleet, rest)
	case appsv1.RollingUpdateDeploymentStrategyType:
		return c.rollingUpdateDeployment(fleet, active, rest)
	}

	return 0, errors.Errorf("unexpected deployment strategy type: %s", fleet.Spec.Strategy.Type)
}

// deleteEmptyGameServerSets deletes all GameServerServerSets
// That have `Status > Replicas` of 0
func (c *Controller) deleteEmptyGameServerSets(fleet *agonesv1.Fleet, list []*agonesv1.GameServerSet) error {
	p := metav1.DeletePropagationBackground
	for _, gsSet := range list {
		if gsSet.Status.Replicas == 0 && gsSet.Status.ShutdownReplicas == 0 {
			err := c.gameServerSetGetter.GameServerSets(gsSet.ObjectMeta.Namespace).Delete(gsSet.ObjectMeta.Name, &metav1.DeleteOptions{PropagationPolicy: &p})
			if err != nil {
				return errors.Wrapf(err, "error updating gameserverset %s", gsSet.ObjectMeta.Name)
			}

			c.recorder.Eventf(fleet, corev1.EventTypeNormal, "DeletingGameServerSet", "Deleting inactive GameServerSet %s", gsSet.ObjectMeta.Name)
		}
	}

	return nil
}

// recreateDeployment applies the recreate deployment strategy to all non-active
// GameServerSets, and return the replica count for the active GameServerSet
func (c *Controller) recreateDeployment(fleet *agonesv1.Fleet, rest []*agonesv1.GameServerSet) (int32, error) {
	for _, gsSet := range rest {
		if gsSet.Spec.Replicas == 0 {
			continue
		}
		c.loggerForFleet(fleet).WithField("gameserverset", gsSet.ObjectMeta.Name).Debug("applying recreate deployment: scaling to 0")
		gsSetCopy := gsSet.DeepCopy()
		gsSetCopy.Spec.Replicas = 0
		if _, err := c.gameServerSetGetter.GameServerSets(gsSetCopy.ObjectMeta.Namespace).Update(gsSetCopy); err != nil {
			return 0, errors.Wrapf(err, "error updating gameserverset %s", gsSetCopy.ObjectMeta.Name)
		}
		c.recorder.Eventf(fleet, corev1.EventTypeNormal, "ScalingGameServerSet",
			"Scaling inactive GameServerSet %s from %d to %d", gsSetCopy.ObjectMeta.Name, gsSet.Spec.Replicas, gsSetCopy.Spec.Replicas)
	}

	return fleet.LowerBoundReplicas(fleet.Spec.Replicas - agonesv1.SumStatusAllocatedReplicas(rest)), nil
}

// rollingUpdateDeployment will do the rolling update of the old GameServers
// through to the new ones, based on the fleet.Spec.Strategy.RollingUpdate configuration
// and return the replica count for the active GameServerSet
func (c *Controller) rollingUpdateDeployment(fleet *agonesv1.Fleet, active *agonesv1.GameServerSet, rest []*agonesv1.GameServerSet) (int32, error) {
	replicas, err := c.rollingUpdateActive(fleet, active, rest)
	if err != nil {
		return replicas, err
	}
	if err := c.rollingUpdateRest(fleet, rest); err != nil {
		return replicas, err
	}
	return replicas, nil
}

// rollingUpdateActive applies the rolling update to the active GameServerSet
// and returns what its replica value should be set to
func (c *Controller) rollingUpdateActive(fleet *agonesv1.Fleet, active *agonesv1.GameServerSet, rest []*agonesv1.GameServerSet) (int32, error) {
	replicas := active.Spec.Replicas
	// always leave room for Allocated GameServers
	sumAllocated := agonesv1.SumStatusAllocatedReplicas(rest)

	// if the active spec replicas are greater than or equal the fleet spec replicas, then we don't
	// need to another rolling update upwards.
	// Likewise if the active spec replicas don't equal the active status replicas, this means we are
	// in the middle of a rolling update, and should wait for it to complete.

	if active.Spec.Replicas != active.Status.Replicas {
		return replicas, nil
	}
	if active.Spec.Replicas >= (fleet.Spec.Replicas - sumAllocated) {
		return fleet.Spec.Replicas - sumAllocated, nil
	}

	r, err := intstr.GetValueFromIntOrPercent(fleet.Spec.Strategy.RollingUpdate.MaxSurge, int(fleet.Spec.Replicas), true)
	if err != nil {
		return replicas, errors.Wrapf(err, "error calculating scaling gameserverset: %s", fleet.ObjectMeta.Name)
	}
	surge := int32(r)

	// make sure we don't end up with more than the configured max surge
	maxSurge := surge + fleet.Spec.Replicas
	replicas = fleet.UpperBoundReplicas(replicas + surge)
	total := agonesv1.SumStatusReplicas(rest) + replicas
	if total > maxSurge {
		replicas = fleet.LowerBoundReplicas(replicas - (total - maxSurge))
	}

	// make room for allocated game servers, but not over the fleet replica count
	if replicas+sumAllocated > fleet.Spec.Replicas {
		replicas = fleet.LowerBoundReplicas(fleet.Spec.Replicas - sumAllocated)
	}

	c.loggerForFleet(fleet).WithField("gameserverset", active.ObjectMeta.Name).WithField("replicas", replicas).
		Debug("applying rolling update to active gameserverset")

	return replicas, nil
}

// rollingUpdateRest applies the rolling update to the inactive GameServerSets
func (c *Controller) rollingUpdateRest(fleet *agonesv1.Fleet, rest []*agonesv1.GameServerSet) error {
	if len(rest) == 0 {
		return nil
	}

	r, err := intstr.GetValueFromIntOrPercent(fleet.Spec.Strategy.RollingUpdate.MaxUnavailable, int(fleet.Spec.Replicas), true)
	if err != nil {
		return errors.Wrapf(err, "error calculating scaling gameserverset: %s", fleet.ObjectMeta.Name)
	}
	unavailable := int32(r)

	for _, gsSet := range rest {
		// if the status.Replicas are less than or equal to 0, then that means we are done
		// scaling this GameServerSet down, and can therefore exit/move to the next one.
		if gsSet.Status.Replicas <= 0 {
			continue
		}

		// If the Spec.Replicas does not equal the Status.Replicas for this GameServerSet, this means
		// that the rolling down process is currently ongoing, and we should therefore exit so we can wait for it to finish
		if gsSet.Spec.Replicas != gsSet.Status.Replicas {
			break
		}
		gsSetCopy := gsSet.DeepCopy()
		if gsSet.Status.ShutdownReplicas == 0 {
			gsSetCopy.Spec.Replicas = fleet.LowerBoundReplicas(gsSetCopy.Spec.Replicas - unavailable)

			c.loggerForFleet(fleet).Debug(fmt.Sprintf("Shutdownreplicas %d", gsSet.Status.ShutdownReplicas))
			c.loggerForFleet(fleet).WithField("gameserverset", gsSet.ObjectMeta.Name).WithField("replicas", gsSetCopy.Spec.Replicas).
				Debug("applying rolling update to inactive gameserverset")

			if _, err := c.gameServerSetGetter.GameServerSets(gsSetCopy.ObjectMeta.Namespace).Update(gsSetCopy); err != nil {
				return errors.Wrapf(err, "error updating gameserverset %s", gsSetCopy.ObjectMeta.Name)
			}
			c.recorder.Eventf(fleet, corev1.EventTypeNormal, "ScalingGameServerSet",
				"Scaling inactive GameServerSet %s from %d to %d", gsSetCopy.ObjectMeta.Name, gsSet.Spec.Replicas, gsSetCopy.Spec.Replicas)

			// let's update just one at a time, slightly slower, but a simpler solution that doesn't require us
			// to make sure we don't overshoot the amount that is being shutdown at any given point and time
			break
		}
	}
	return nil
}

// updateFleetStatus gets the GameServerSets for this Fleet and then
// calculates the counts for the status, and updates the Fleet
func (c *Controller) updateFleetStatus(fleet *agonesv1.Fleet) error {
	c.loggerForFleet(fleet).Debug("Update Fleet Status")

	list, err := ListGameServerSetsByFleetOwner(c.gameServerSetLister, fleet)
	if err != nil {
		return err
	}

	fCopy, err := c.fleetGetter.Fleets(fleet.ObjectMeta.Namespace).Get(fleet.ObjectMeta.GetName(), metav1.GetOptions{})
	if err != nil {
		return err
	}
	fCopy.Status.Replicas = 0
	fCopy.Status.ReadyReplicas = 0
	fCopy.Status.ReservedReplicas = 0
	fCopy.Status.AllocatedReplicas = 0

	for _, gsSet := range list {
		fCopy.Status.Replicas += gsSet.Status.Replicas
		fCopy.Status.ReadyReplicas += gsSet.Status.ReadyReplicas
		fCopy.Status.ReservedReplicas += gsSet.Status.ReservedReplicas
		fCopy.Status.AllocatedReplicas += gsSet.Status.AllocatedReplicas
	}
	_, err = c.fleetGetter.Fleets(fCopy.ObjectMeta.Namespace).UpdateStatus(fCopy)
	return errors.Wrapf(err, "error updating status of fleet %s", fCopy.ObjectMeta.Name)
}

// filterGameServerSetByActive returns the active GameServerSet (or nil if it
// doesn't exist) and then the rest of the GameServerSets that are controlled
// by this Fleet
func (c *Controller) filterGameServerSetByActive(fleet *agonesv1.Fleet, list []*agonesv1.GameServerSet) (*agonesv1.GameServerSet, []*agonesv1.GameServerSet) {
	var active *agonesv1.GameServerSet
	var rest []*agonesv1.GameServerSet

	for _, gsSet := range list {
		if apiequality.Semantic.DeepEqual(gsSet.Spec.Template, fleet.Spec.Template) {
			active = gsSet
		} else {
			rest = append(rest, gsSet)
		}
	}

	return active, rest
}
