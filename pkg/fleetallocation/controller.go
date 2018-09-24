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

package fleetallocation

import (
	"encoding/json"
	"fmt"
	"sync"

	"agones.dev/agones/pkg/apis/stable"
	stablev1alpha1 "agones.dev/agones/pkg/apis/stable/v1alpha1"
	"agones.dev/agones/pkg/client/clientset/versioned"
	getterv1alpha1 "agones.dev/agones/pkg/client/clientset/versioned/typed/stable/v1alpha1"
	"agones.dev/agones/pkg/client/informers/externalversions"
	listerv1alpha1 "agones.dev/agones/pkg/client/listers/stable/v1alpha1"
	"agones.dev/agones/pkg/fleets"
	"agones.dev/agones/pkg/util/crd"
	"agones.dev/agones/pkg/util/runtime"
	"agones.dev/agones/pkg/util/webhooks"
	"github.com/mattbaird/jsonpatch"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	admv1beta1 "k8s.io/api/admission/v1beta1"
	corev1 "k8s.io/api/core/v1"
	extclientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/typed/apiextensions/v1beta1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
)

var (
	// ErrNoGameServerReady is returned when there are no Ready GameServers
	// available
	ErrNoGameServerReady = errors.New("Could not find a Ready GameServer")
)

// Controller is a the FleetAllocation controller
type Controller struct {
	logger                *logrus.Entry
	crdGetter             v1beta1.CustomResourceDefinitionInterface
	gameServerSynced      cache.InformerSynced
	gameServerGetter      getterv1alpha1.GameServersGetter
	gameServerLister      listerv1alpha1.GameServerLister
	gameServerSetLister   listerv1alpha1.GameServerSetLister
	fleetLister           listerv1alpha1.FleetLister
	fleetAllocationGetter getterv1alpha1.FleetAllocationsGetter
	fleetAllocationLister listerv1alpha1.FleetAllocationLister
	stop                  <-chan struct{}
	allocationMutex       *sync.Mutex
	recorder              record.EventRecorder
}

// NewController returns a controller for a FleetAllocation
func NewController(
	wh *webhooks.WebHook,
	allocationMutex *sync.Mutex,
	kubeClient kubernetes.Interface,
	extClient extclientset.Interface,
	agonesClient versioned.Interface,
	agonesInformerFactory externalversions.SharedInformerFactory) *Controller {

	agonesInformer := agonesInformerFactory.Stable().V1alpha1()
	c := &Controller{
		crdGetter:             extClient.ApiextensionsV1beta1().CustomResourceDefinitions(),
		gameServerSynced:      agonesInformer.GameServers().Informer().HasSynced,
		gameServerGetter:      agonesClient.StableV1alpha1(),
		gameServerLister:      agonesInformer.GameServers().Lister(),
		gameServerSetLister:   agonesInformer.GameServerSets().Lister(),
		fleetLister:           agonesInformer.Fleets().Lister(),
		fleetAllocationGetter: agonesClient.StableV1alpha1(),
		fleetAllocationLister: agonesInformer.FleetAllocations().Lister(),
		allocationMutex:       allocationMutex,
	}
	c.logger = runtime.NewLoggerWithType(c)

	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(c.logger.Infof)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: kubeClient.CoreV1().Events("")})
	c.recorder = eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: "fleetallocation-controller"})

	kind := stablev1alpha1.Kind("FleetAllocation")
	wh.AddHandler("/mutate", kind, admv1beta1.Create, c.creationMutationHandler)
	wh.AddHandler("/validate", kind, admv1beta1.Create, c.creationValidationHandler)
	wh.AddHandler("/validate", kind, admv1beta1.Update, c.mutationValidationHandler)

	return c
}

// Run runs this controller. This controller doesn't (currently)
// have a worker/queue, and as such, does not block.
func (c *Controller) Run(workers int, stop <-chan struct{}) error {
	err := crd.WaitForEstablishedCRD(c.crdGetter, "fleetallocations."+stable.GroupName, c.logger)
	if err != nil {
		return err
	}

	c.stop = stop
	return nil
}

// creationMutationHandler will intercept when a FleetAllocation is created, and allocate it a GameServer
// assuming that one is available. If not, it will reject the AdmissionReview.
func (c *Controller) creationMutationHandler(review admv1beta1.AdmissionReview) (admv1beta1.AdmissionReview, error) {
	c.logger.WithField("review", review).Info("creationMutationHandler")
	obj := review.Request.Object
	fa := &stablev1alpha1.FleetAllocation{}

	err := json.Unmarshal(obj.Raw, fa)
	if err != nil {
		return review, errors.Wrapf(err, "error unmarshalling original FleetAllocation json: %s", obj.Raw)
	}

	// When being called from the API the fa.ObjectMeta.Namespace isn't populated
	// (whereas it is from kubectl). So make sure to pull the namespace from the review
	fleet, err := c.fleetLister.Fleets(review.Request.Namespace).Get(fa.Spec.FleetName)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			logrus.WithError(err).WithField("fleetName", fa.Name).
				WithField("namespace", review.Request.Namespace).
				Warn("Could not find fleet for allocation. Skipping.")
			return review, nil
		}
		return review, errors.Wrapf(err, "error retrieving fleet %s", fa.Name)
	}

	gs, err := c.allocate(fleet, &fa.Spec.MetaPatch)
	if err != nil {
		review.Response.Allowed = false
		review.Response.Result = &metav1.Status{
			Status: metav1.StatusFailure,
			Reason: metav1.StatusReasonNotFound,
			Details: &metav1.StatusDetails{
				Name:  review.Request.Name,
				Group: review.Request.Kind.Group,
				Kind:  "GameServer",
			},
		}

		return review, nil
	}

	// When a GameServer is deleted, the FleetAllocation should go with it
	ref := metav1.NewControllerRef(gs, stablev1alpha1.SchemeGroupVersion.WithKind("GameServer"))
	fa.ObjectMeta.OwnerReferences = append(fa.ObjectMeta.OwnerReferences, *ref)

	fa.Status = stablev1alpha1.FleetAllocationStatus{GameServer: gs}

	newFA, err := json.Marshal(fa)
	if err != nil {
		return review, errors.Wrapf(err, "error marshalling FleetAllocation %s to json", fa.ObjectMeta.Name)
	}

	patch, err := jsonpatch.CreatePatch(obj.Raw, newFA)
	if err != nil {
		return review, errors.Wrapf(err, "error creating patch for FleetAllocation %s", fa.ObjectMeta.Name)
	}

	json, err := json.Marshal(patch)
	if err != nil {
		return review, errors.Wrapf(err, "error creating json for patch for FleetAllocation %s", gs.ObjectMeta.Name)
	}

	c.logger.WithField("fa", fa.ObjectMeta.Name).WithField("patch", string(json)).Infof("patch created!")

	pt := admv1beta1.PatchTypeJSONPatch
	review.Response.PatchType = &pt
	review.Response.Patch = json

	return review, nil
}

// creationValidationHandler intercepts the creation of a FleetAllocation, and if there is
// no Status > GameServer set, then we will assume that the Spec > fleetName is invalid
func (c *Controller) creationValidationHandler(review admv1beta1.AdmissionReview) (admv1beta1.AdmissionReview, error) {
	c.logger.WithField("review", review).Info("creationValidationHandler")
	obj := review.Request.Object
	fa := &stablev1alpha1.FleetAllocation{}
	if err := json.Unmarshal(obj.Raw, fa); err != nil {
		return review, errors.Wrapf(err, "error unmarshalling original FleetAllocation json: %s", obj.Raw)
	}

	// If there is no GameServer, we are assuming that is
	// because the fleetName is invalid. Any other error
	// option should be handled by the creationMutationHandler
	if fa.Status.GameServer == nil {
		review.Response.Allowed = false
		review.Response.Result = &metav1.Status{
			Status:  metav1.StatusFailure,
			Reason:  metav1.StatusReasonInvalid,
			Message: "Invalid FleetAllocation",
			Details: &metav1.StatusDetails{
				Name:  review.Request.Name,
				Group: review.Request.Kind.Group,
				Kind:  review.Request.Kind.Kind,
				Causes: []metav1.StatusCause{
					{Type: metav1.CauseTypeFieldValueNotFound,
						Message: fmt.Sprintf("Could not find fleet %s in namespace %s", fa.Spec.FleetName, review.Request.Namespace),
						Field:   "fleetName"}},
			},
		}
	}

	return review, nil
}

// mutationValidationHandler stops edits from happening to a
// FleetAllocation fleetName value
func (c *Controller) mutationValidationHandler(review admv1beta1.AdmissionReview) (admv1beta1.AdmissionReview, error) {
	c.logger.WithField("review", review).Info("mutationValidationHandler")

	newFA := &stablev1alpha1.FleetAllocation{}
	oldFA := &stablev1alpha1.FleetAllocation{}

	if err := json.Unmarshal(review.Request.Object.Raw, newFA); err != nil {
		return review, errors.Wrapf(err, "error unmarshalling new FleetAllocation json: %s", review.Request.Object.Raw)
	}

	if err := json.Unmarshal(review.Request.OldObject.Raw, oldFA); err != nil {
		return review, errors.Wrapf(err, "error unmarshalling old FleetAllocation json: %s", review.Request.Object.Raw)
	}

	if ok, causes := oldFA.ValidateUpdate(newFA); !ok {
		review.Response.Allowed = false
		details := metav1.StatusDetails{
			Name:   review.Request.Name,
			Group:  review.Request.Kind.Group,
			Kind:   review.Request.Kind.Kind,
			Causes: causes,
		}
		review.Response.Result = &metav1.Status{
			Status:  metav1.StatusFailure,
			Message: "FleetAllocation update is invalid",
			Reason:  metav1.StatusReasonInvalid,
			Details: &details,
		}
	}

	return review, nil
}

// allocate allocated a GameServer from a given Fleet
func (c *Controller) allocate(f *stablev1alpha1.Fleet, fam *stablev1alpha1.FleetAllocationMeta) (*stablev1alpha1.GameServer, error) {
	var allocation *stablev1alpha1.GameServer
	// can only allocate one at a time, as we don't want two separate processes
	// trying to allocate the same GameServer to different clients
	c.allocationMutex.Lock()
	defer c.allocationMutex.Unlock()

	// make sure we have the most up to date view of the world
	if !cache.WaitForCacheSync(c.stop, c.gameServerSynced) {
		return allocation, errors.New("error syncing GameServer cache")
	}
	gsList, err := fleets.ListGameServersByFleetOwner(c.gameServerLister, c.gameServerSetLister, f)
	if err != nil {
		return allocation, err
	}

	for _, gs := range gsList {
		if gs.Status.State == stablev1alpha1.Ready && gs.ObjectMeta.DeletionTimestamp.IsZero() {
			allocation = gs
			break
		}
	}

	if allocation == nil {
		return allocation, ErrNoGameServerReady
	}

	gsCopy := allocation.DeepCopy()
	gsCopy.Status.State = stablev1alpha1.Allocated

	if fam != nil {
		c.patchMetadata(gsCopy, fam)
	}

	gs, err := c.gameServerGetter.GameServers(f.ObjectMeta.Namespace).Update(gsCopy)
	if err != nil {
		return gs, errors.Wrapf(err, "error updating GameServer %s", gsCopy.ObjectMeta.Name)
	}
	c.recorder.Eventf(gs, corev1.EventTypeNormal, string(gs.Status.State), "Allocated from Fleet %s", f.ObjectMeta.Name)

	return gs, nil
}

// patch the labels and annotations of an allocated GameServer with metadata from a FleetAllocation
func (c *Controller) patchMetadata(gs *stablev1alpha1.GameServer, fam *stablev1alpha1.FleetAllocationMeta) {
	// patch ObjectMeta labels
	if fam.Labels != nil {
		if gs.ObjectMeta.Labels == nil {
			gs.ObjectMeta.Labels = make(map[string]string, len(fam.Labels))
		}
		for key, value := range fam.Labels {
			gs.ObjectMeta.Labels[key] = value
		}
	}
	// apply annotations patch
	if fam.Annotations != nil {
		if gs.ObjectMeta.Annotations == nil {
			gs.ObjectMeta.Annotations = make(map[string]string, len(fam.Annotations))
		}
		for key, value := range fam.Annotations {
			gs.ObjectMeta.Annotations[key] = value
		}
	}
}
