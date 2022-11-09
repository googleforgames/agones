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

package fleetautoscalers

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	"agones.dev/agones/pkg/apis/autoscaling"
	autoscalingv1 "agones.dev/agones/pkg/apis/autoscaling/v1"
	"agones.dev/agones/pkg/client/clientset/versioned"
	typedagonesv1 "agones.dev/agones/pkg/client/clientset/versioned/typed/agones/v1"
	typedautoscalingv1 "agones.dev/agones/pkg/client/clientset/versioned/typed/autoscaling/v1"
	"agones.dev/agones/pkg/client/informers/externalversions"
	listeragonesv1 "agones.dev/agones/pkg/client/listers/agones/v1"
	listerautoscalingv1 "agones.dev/agones/pkg/client/listers/autoscaling/v1"
	"agones.dev/agones/pkg/util/crd"
	"agones.dev/agones/pkg/util/logfields"
	"agones.dev/agones/pkg/util/runtime"
	"agones.dev/agones/pkg/util/webhooks"
	"agones.dev/agones/pkg/util/workerqueue"
	"github.com/heptiolabs/healthcheck"
	"github.com/mattbaird/jsonpatch"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	extclientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	apiextclientv1 "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/typed/apiextensions/v1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/clock"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
)

// fasThread is used for tracking each Fleet's autoscaling jobs
//
//nolint:govet // ignore fieldalignment, one per fleet autoscaler
type fasThread struct {
	generation int64
	cancel     context.CancelFunc
}

// Controller is the FleetAutoscaler controller
//
//nolint:govet // ignore fieldalignment, singleton
type Controller struct {
	baseLogger            *logrus.Entry
	clock                 clock.Clock
	crdGetter             apiextclientv1.CustomResourceDefinitionInterface
	fasThreads            map[types.UID]fasThread
	fasThreadMutex        sync.Mutex
	fleetGetter           typedagonesv1.FleetsGetter
	fleetLister           listeragonesv1.FleetLister
	fleetSynced           cache.InformerSynced
	fleetAutoscalerGetter typedautoscalingv1.FleetAutoscalersGetter
	fleetAutoscalerLister listerautoscalingv1.FleetAutoscalerLister
	fleetAutoscalerSynced cache.InformerSynced
	workerqueue           *workerqueue.WorkerQueue
	recorder              record.EventRecorder
}

// NewController returns a controller for a FleetAutoscaler
func NewController(
	wh *webhooks.WebHook,
	health healthcheck.Handler,
	kubeClient kubernetes.Interface,
	extClient extclientset.Interface,
	agonesClient versioned.Interface,
	agonesInformerFactory externalversions.SharedInformerFactory) *Controller {

	autoscaler := agonesInformerFactory.Autoscaling().V1().FleetAutoscalers()
	fleetInformer := agonesInformerFactory.Agones().V1().Fleets()
	c := &Controller{
		clock:                 clock.RealClock{},
		crdGetter:             extClient.ApiextensionsV1().CustomResourceDefinitions(),
		fasThreads:            map[types.UID]fasThread{},
		fasThreadMutex:        sync.Mutex{},
		fleetGetter:           agonesClient.AgonesV1(),
		fleetLister:           fleetInformer.Lister(),
		fleetSynced:           fleetInformer.Informer().HasSynced,
		fleetAutoscalerGetter: agonesClient.AutoscalingV1(),
		fleetAutoscalerLister: autoscaler.Lister(),
		fleetAutoscalerSynced: autoscaler.Informer().HasSynced,
	}
	c.baseLogger = runtime.NewLoggerWithType(c)
	c.workerqueue = workerqueue.NewWorkerQueueWithRateLimiter(c.syncFleetAutoscaler, c.baseLogger, logfields.FleetAutoscalerKey, autoscaling.GroupName+".FleetAutoscalerController", workerqueue.FastRateLimiter(3*time.Second))
	health.AddLivenessCheck("fleetautoscaler-workerqueue", healthcheck.Check(c.workerqueue.Healthy))

	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(c.baseLogger.Debugf)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: kubeClient.CoreV1().Events("")})
	c.recorder = eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: "fleetautoscaler-controller"})

	kind := autoscalingv1.Kind("FleetAutoscaler")
	wh.AddHandler("/mutate", kind, admissionv1.Create, c.mutationHandler)
	wh.AddHandler("/mutate", kind, admissionv1.Update, c.mutationHandler)
	wh.AddHandler("/validate", kind, admissionv1.Create, c.validationHandler)
	wh.AddHandler("/validate", kind, admissionv1.Update, c.validationHandler)

	autoscaler.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			if runtime.FeatureEnabled(runtime.FeatureCustomFasSyncInterval) {
				c.addFasThread(obj.(*autoscalingv1.FleetAutoscaler), true)
			} else {
				c.workerqueue.Enqueue(obj)
			}
		},
		UpdateFunc: func(_, newObj interface{}) {
			if runtime.FeatureEnabled(runtime.FeatureCustomFasSyncInterval) {
				c.updateFasThread(newObj.(*autoscalingv1.FleetAutoscaler))
			} else {
				c.workerqueue.Enqueue(newObj)
			}
		},
		DeleteFunc: func(obj interface{}) {
			if runtime.FeatureEnabled(runtime.FeatureCustomFasSyncInterval) {
				// Could be a DeletedFinalStateUnknown, in which case, just ignore it
				fas, ok := obj.(*autoscalingv1.FleetAutoscaler)
				if !ok {
					return
				}
				c.deleteFasThread(fas, true)
			}
		},
	})

	return c
}

// Run the FleetAutoscaler controller. Will block until stop is closed.
// Runs threadiness number workers to process the rate limited queue
func (c *Controller) Run(ctx context.Context, workers int) error {
	err := crd.WaitForEstablishedCRD(ctx, c.crdGetter, "fleetautoscalers."+autoscaling.GroupName, c.baseLogger)
	if err != nil {
		return err
	}

	c.baseLogger.Debug("Wait for cache sync")
	if !cache.WaitForCacheSync(ctx.Done(), c.fleetSynced, c.fleetAutoscalerSynced) {
		return errors.New("failed to wait for caches to sync")
	}

	go func() {
		// clean all go routines when ctx is Done
		<-ctx.Done()
		c.fasThreadMutex.Lock()
		defer c.fasThreadMutex.Unlock()
		for _, thread := range c.fasThreads {
			thread.cancel()
		}
	}()

	c.workerqueue.Run(ctx, workers)
	return nil
}

func (c *Controller) loggerForFleetAutoscalerKey(key string) *logrus.Entry {
	return logfields.AugmentLogEntry(c.baseLogger, logfields.FleetAutoscalerKey, key)
}

func (c *Controller) loggerForFleetAutoscaler(fas *autoscalingv1.FleetAutoscaler) *logrus.Entry {
	fasName := "NilFleetAutoScaler"
	if fas != nil {
		fasName = fas.Namespace + "/" + fas.Name
	}
	return c.loggerForFleetAutoscalerKey(fasName).WithField("fas", fas)
}

// creationMutationHandler is the handler for the mutating webhook that sets the
// the default values on the FleetAutoscaler
func (c *Controller) mutationHandler(review admissionv1.AdmissionReview) (admissionv1.AdmissionReview, error) {
	obj := review.Request.Object
	fas := &autoscalingv1.FleetAutoscaler{}
	err := json.Unmarshal(obj.Raw, fas)
	if err != nil {
		c.baseLogger.WithField("review", review).WithError(err).Error("validationHandler")
		return review, errors.Wrapf(err, "error unmarshalling original FleetAutoscaler json: %s", obj.Raw)
	}

	fas.ApplyDefaults()

	newFas, err := json.Marshal(fas)
	if err != nil {
		return review, errors.Wrapf(err, "error marshalling default applied FleetAutoscaler %s to json", fas.ObjectMeta.Name)
	}

	patch, err := jsonpatch.CreatePatch(obj.Raw, newFas)
	if err != nil {
		return review, errors.Wrapf(err, "error creating patch for FleetAutoscaler %s", fas.ObjectMeta.Name)
	}

	jsonPatch, err := json.Marshal(patch)
	if err != nil {
		return review, errors.Wrapf(err, "error creating json for patch for FleetAutoScaler %s", fas.ObjectMeta.Name)
	}

	pt := admissionv1.PatchTypeJSONPatch
	review.Response.PatchType = &pt
	review.Response.Patch = jsonPatch

	return review, nil
}

// validationHandler will intercept when a FleetAutoscaler is created, and
// validate its settings.
func (c *Controller) validationHandler(review admissionv1.AdmissionReview) (admissionv1.AdmissionReview, error) {
	obj := review.Request.Object
	fas := &autoscalingv1.FleetAutoscaler{}
	err := json.Unmarshal(obj.Raw, fas)
	if err != nil {
		c.baseLogger.WithField("review", review).WithError(err).Error("validationHandler")
		return review, errors.Wrapf(err, "error unmarshalling original FleetAutoscaler json: %s", obj.Raw)
	}
	fas.ApplyDefaults()
	var causes []metav1.StatusCause
	causes = fas.Validate(causes)
	if len(causes) != 0 {
		review.Response.Allowed = false
		details := metav1.StatusDetails{
			Name:   review.Request.Name,
			Group:  review.Request.Kind.Group,
			Kind:   review.Request.Kind.Kind,
			Causes: causes,
		}
		review.Response.Result = &metav1.Status{
			Status:  metav1.StatusFailure,
			Message: "FleetAutoscaler is invalid",
			Reason:  metav1.StatusReasonInvalid,
			Details: &details,
		}
	}

	return review, nil
}

// syncFleetAutoscaler syncs FleetAutoScale according to different sync type
func (c *Controller) syncFleetAutoscaler(ctx context.Context, key string) error {
	c.loggerForFleetAutoscalerKey(key).Debug("Synchronising")

	fas, err := c.getFleetAutoscalerByKey(key)
	if err != nil {
		return err
	}

	if fas == nil {
		// just in case we don't catch a delete event for some reason, use this as a
		// failsafe to ensure we don't end up leaking goroutines.
		if runtime.FeatureEnabled(runtime.FeatureCustomFasSyncInterval) {
			return c.cleanFasThreads(key)
		}
		return nil
	}

	// Retrieve the fleet by spec name
	fleet, err := c.fleetLister.Fleets(fas.Namespace).Get(fas.Spec.FleetName)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			c.loggerForFleetAutoscaler(fas).Debug("Could not find fleet for autoscaler. Skipping.")

			c.recorder.Eventf(fas, corev1.EventTypeWarning, "FailedGetFleet",
				"could not fetch fleet: %s", fas.Spec.FleetName)

			// don't retry. Pick it up next sync.
			err = nil
		}

		if err := c.updateStatusUnableToScale(ctx, fas); err != nil {
			return err
		}

		return err
	}

	// Don't do anything, the fleet is marked for deletion
	if !fleet.DeletionTimestamp.IsZero() {
		return nil
	}

	currentReplicas := fleet.Status.Replicas
	desiredReplicas, scalingLimited, err := computeDesiredFleetSize(fas, fleet)
	if err != nil {
		c.recorder.Eventf(fas, corev1.EventTypeWarning, "FleetAutoscaler",
			"Error calculating desired fleet size on FleetAutoscaler %s. Error: %s", fas.ObjectMeta.Name, err.Error())

		if err := c.updateStatusUnableToScale(ctx, fas); err != nil {
			return err
		}
		return errors.Wrapf(err, "error calculating autoscaling fleet: %s", fleet.ObjectMeta.Name)
	}

	// Scale the fleet to the new size
	if err = c.scaleFleet(ctx, fas, fleet, desiredReplicas); err != nil {
		return errors.Wrapf(err, "error autoscaling fleet %s to %d replicas", fas.Spec.FleetName, desiredReplicas)
	}

	return c.updateStatus(ctx, fas, currentReplicas, desiredReplicas, desiredReplicas != fleet.Spec.Replicas, scalingLimited)
}

// getFleetAutoscalerByKey gets the Fleet Autoscaler by key
// a nil FleetAutoscaler returned indicates that an attempt to sync should not be retried, e.g.  if the FleetAutoscaler no longer exists.
func (c *Controller) getFleetAutoscalerByKey(key string) (*autoscalingv1.FleetAutoscaler, error) {
	// Convert the namespace/name string into a distinct namespace and name
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		// don't return an error, as we don't want this retried
		runtime.HandleError(c.loggerForFleetAutoscalerKey(key), errors.Wrapf(err, "invalid resource key"))
		return nil, nil
	}
	fas, err := c.fleetAutoscalerLister.FleetAutoscalers(namespace).Get(name)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			c.loggerForFleetAutoscalerKey(key).Debug(fmt.Sprintf("FleetAutoscaler %s from namespace %s is no longer available for syncing", name, namespace))
			return nil, nil
		}
		return nil, errors.Wrapf(err, "error retrieving FleetAutoscaler %s from namespace %s", name, namespace)
	}
	return fas, nil
}

// scaleFleet scales the fleet of the autoscaler to a new number of replicas
func (c *Controller) scaleFleet(ctx context.Context, fas *autoscalingv1.FleetAutoscaler, f *agonesv1.Fleet, replicas int32) error {
	if replicas != f.Spec.Replicas {
		fCopy := f.DeepCopy()
		fCopy.Spec.Replicas = replicas
		fCopy, err := c.fleetGetter.Fleets(f.ObjectMeta.Namespace).Update(ctx, fCopy, metav1.UpdateOptions{})
		if err != nil {
			c.recorder.Eventf(fas, corev1.EventTypeWarning, "AutoScalingFleetError",
				"Error on scaling fleet %s from %d to %d. Error: %s", fCopy.ObjectMeta.Name, f.Spec.Replicas, fCopy.Spec.Replicas, err.Error())
			return errors.Wrapf(err, "error updating replicas for fleet %s", f.ObjectMeta.Name)
		}

		c.recorder.Eventf(fas, corev1.EventTypeNormal, "AutoScalingFleet",
			"Scaling fleet %s from %d to %d", fCopy.ObjectMeta.Name, f.Spec.Replicas, fCopy.Spec.Replicas)
	}

	return nil
}

// updateStatus updates the status of the given FleetAutoscaler
func (c *Controller) updateStatus(ctx context.Context, fas *autoscalingv1.FleetAutoscaler, currentReplicas int32, desiredReplicas int32, scaled bool, scalingLimited bool) error {
	fasCopy := fas.DeepCopy()
	fasCopy.Status.AbleToScale = true
	fasCopy.Status.ScalingLimited = scalingLimited
	fasCopy.Status.CurrentReplicas = currentReplicas
	fasCopy.Status.DesiredReplicas = desiredReplicas
	if scaled {
		now := metav1.NewTime(time.Now())
		fasCopy.Status.LastScaleTime = &now
	}

	if !apiequality.Semantic.DeepEqual(fas.Status, fasCopy.Status) {
		if scalingLimited {
			// scalingLimited indicates that the calculated scale would be above or below the range defined by MinReplicas and MaxReplicas
			msg := "Scaling fleet %s was limited to minimum size of %d"
			if currentReplicas > desiredReplicas {
				msg = "Scaling fleet %s was limited to maximum size of %d"
			}

			c.recorder.Eventf(fas, corev1.EventTypeWarning, "ScalingLimited", msg, fas.Spec.FleetName, desiredReplicas)
		}

		_, err := c.fleetAutoscalerGetter.FleetAutoscalers(fas.ObjectMeta.Namespace).UpdateStatus(ctx, fasCopy, metav1.UpdateOptions{})
		if err != nil {
			return errors.Wrapf(err, "error updating status for fleetautoscaler %s", fas.ObjectMeta.Name)
		}
	}

	return nil
}

// updateStatus updates the status of the given FleetAutoscaler in the case we're not able to scale
func (c *Controller) updateStatusUnableToScale(ctx context.Context, fas *autoscalingv1.FleetAutoscaler) error {
	fasCopy := fas.DeepCopy()
	fasCopy.Status.AbleToScale = false
	fasCopy.Status.ScalingLimited = false
	fasCopy.Status.CurrentReplicas = 0
	fasCopy.Status.DesiredReplicas = 0

	if !apiequality.Semantic.DeepEqual(fas.Status, fasCopy.Status) {
		_, err := c.fleetAutoscalerGetter.FleetAutoscalers(fas.ObjectMeta.Namespace).UpdateStatus(ctx, fasCopy, metav1.UpdateOptions{})
		if err != nil {
			return errors.Wrapf(err, "error updating status for fleetautoscaler %s", fas.ObjectMeta.Name)
		}
	}

	return nil
}

// addFasThread creates a ticker that enqueues the FleetAutoscaler for it's configured interval.
// If `lock` is set to true, the function will do appropriate locking for this operation. If set to `false`
// make sure to lock the operation with the c.fasThreadMutex for the execution of this command.
func (c *Controller) addFasThread(fas *autoscalingv1.FleetAutoscaler, lock bool) {
	log := c.loggerForFleetAutoscaler(fas)
	log.WithField("seconds", fas.Spec.Sync.FixedInterval.Seconds).Debug("Thread for Autoscaler created")

	duration := time.Duration(fas.Spec.Sync.FixedInterval.Seconds) * time.Second

	// store against the UID, as there is no guarantee the name is unique over time.
	ctx, cancel := context.WithCancel(context.Background())
	thread := fasThread{
		cancel:     cancel,
		generation: fas.Generation,
	}

	if lock {
		c.fasThreadMutex.Lock()
		defer c.fasThreadMutex.Unlock()
	}

	// Seems unlikely that concurrent events could fire at the same time for the same UID,
	// but just in case, let's check.
	if _, ok := c.fasThreads[fas.ObjectMeta.UID]; ok {
		return
	}
	c.fasThreads[fas.ObjectMeta.UID] = thread

	// do immediate enqueue on addition to have an autoscale fire on addition.
	c.workerqueue.Enqueue(fas)
	// Add to queue for each duration period, until cancellation occurs.
	// Workerqueue will handle if multiple attempts are made to add an existing item to the queue, and retries on failure
	// etc.
	go func() {
		wait.Until(func() {
			c.workerqueue.Enqueue(fas)
		}, duration, ctx.Done())
	}()
}

// updateFasThread will replace the queueing thread if the generation has changes on the FleetAutoscaler.
func (c *Controller) updateFasThread(fas *autoscalingv1.FleetAutoscaler) {
	c.fasThreadMutex.Lock()
	defer c.fasThreadMutex.Unlock()

	thread, ok := c.fasThreads[fas.ObjectMeta.UID]
	if !ok {
		// maybe the controller crashed and we are only getting update events at this point, so let's add
		// the thread back in
		c.addFasThread(fas, false)
		return
	}

	if fas.Generation != thread.generation {
		c.loggerForFleetAutoscaler(fas).WithField("generation", thread.generation).
			Debug("Fleet autoscaler generation updated, recreating thread")
		c.deleteFasThread(fas, false)
		c.addFasThread(fas, false)
	}
}

// deleteFasThread removes a FleetAutoScaler sync routine.
// If `lock` is set to true, the function will do appropriate locking for this operation. If set to `false`
// make sure to lock the operation with the c.fasThreadMutex for the execution of this command.
func (c *Controller) deleteFasThread(fas *autoscalingv1.FleetAutoscaler, lock bool) {
	c.loggerForFleetAutoscaler(fas).Debug("Thread for Autoscaler removed")

	if lock {
		c.fasThreadMutex.Lock()
		defer c.fasThreadMutex.Unlock()
	}

	if thread, ok := c.fasThreads[fas.ObjectMeta.UID]; ok {
		thread.cancel()
		delete(c.fasThreads, fas.ObjectMeta.UID)
	}
}

// cleanFasThreads will delete any fasThread that no longer
// can be tied to a FleetAutoscaler instance.
func (c *Controller) cleanFasThreads(key string) error {
	c.baseLogger.WithField("key", key).Debug("Doing full autoscaler thread cleanup")
	namespace, _, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return errors.Wrap(err, "attempting to clean all fleet autoscaler threads")
	}

	fasList, err := c.fleetAutoscalerLister.FleetAutoscalers(namespace).List(labels.Everything())
	if err != nil {
		return errors.Wrap(err, "attempting to clean all fleet autoscaler threads")
	}

	c.fasThreadMutex.Lock()
	defer c.fasThreadMutex.Unlock()

	keys := map[types.UID]bool{}
	for k := range c.fasThreads {
		keys[k] = true
	}

	for _, fas := range fasList {
		delete(keys, fas.ObjectMeta.UID)
	}

	// any key that doesn't match to an existing UID, stop it.
	for k := range keys {
		c.fasThreads[k].cancel()
		delete(c.fasThreads, k)
	}

	return nil
}
