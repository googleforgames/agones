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

package gameserverallocations

import (
	"encoding/json"
	"sync"

	"agones.dev/agones/pkg/apis/stable"
	"agones.dev/agones/pkg/apis/stable/v1alpha1"
	"agones.dev/agones/pkg/client/clientset/versioned"
	getterv1alpha1 "agones.dev/agones/pkg/client/clientset/versioned/typed/stable/v1alpha1"
	"agones.dev/agones/pkg/client/informers/externalversions"
	listerv1alpha1 "agones.dev/agones/pkg/client/listers/stable/v1alpha1"
	"agones.dev/agones/pkg/util/crd"
	"agones.dev/agones/pkg/util/runtime"
	"agones.dev/agones/pkg/util/webhooks"
	"agones.dev/agones/pkg/util/workerqueue"
	"github.com/heptiolabs/healthcheck"
	"github.com/mattbaird/jsonpatch"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	admv1beta1 "k8s.io/api/admission/v1beta1"
	corev1 "k8s.io/api/core/v1"
	extclientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/typed/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/informers"
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

// Controller is a the GameServerAllocation controller
type Controller struct {
	logger                     *logrus.Entry
	counter                    *AllocationCounter
	crdGetter                  v1beta1.CustomResourceDefinitionInterface
	gameServerSynced           cache.InformerSynced
	gameServerGetter           getterv1alpha1.GameServersGetter
	gameServerLister           listerv1alpha1.GameServerLister
	gameServerAllocationSynced cache.InformerSynced
	gameServerAllocationGetter getterv1alpha1.GameServerAllocationsGetter
	stop                       <-chan struct{}
	allocationMutex            *sync.Mutex
	workerqueue                *workerqueue.WorkerQueue
	recorder                   record.EventRecorder
}

// findComparator is a comparator function specifically for the
// findReadyGameServerForAllocation method for determining
// scheduling strategy
type findComparator func(bestCount, currentCount NodeCount) bool

// NewController returns a controller for a GameServerAllocation
func NewController(wh *webhooks.WebHook,
	health healthcheck.Handler,
	allocationMutex *sync.Mutex,
	kubeClient kubernetes.Interface,
	kubeInformerFactory informers.SharedInformerFactory,
	extClient extclientset.Interface,
	agonesClient versioned.Interface,
	agonesInformerFactory externalversions.SharedInformerFactory) *Controller {

	agonesInformer := agonesInformerFactory.Stable().V1alpha1()
	c := &Controller{
		counter:                    NewAllocationCounter(kubeInformerFactory, agonesInformerFactory),
		crdGetter:                  extClient.ApiextensionsV1beta1().CustomResourceDefinitions(),
		gameServerSynced:           agonesInformer.GameServers().Informer().HasSynced,
		gameServerGetter:           agonesClient.StableV1alpha1(),
		gameServerLister:           agonesInformer.GameServers().Lister(),
		gameServerAllocationSynced: agonesInformer.GameServerAllocations().Informer().HasSynced,
		gameServerAllocationGetter: agonesClient.StableV1alpha1(),
		allocationMutex:            allocationMutex,
	}
	c.logger = runtime.NewLoggerWithType(c)
	c.workerqueue = workerqueue.NewWorkerQueue(c.syncDelete, c.logger, stable.GroupName+".GameServerAllocationController")
	health.AddLivenessCheck("gameserverallocation-workerqueue", healthcheck.Check(c.workerqueue.Healthy))

	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(c.logger.Infof)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: kubeClient.CoreV1().Events("")})
	c.recorder = eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: "GameServerAllocation-controller"})

	kind := v1alpha1.Kind("GameServerAllocation")
	wh.AddHandler("/mutate", kind, admv1beta1.Create, c.creationMutationHandler)
	wh.AddHandler("/validate", kind, admv1beta1.Update, c.mutationValidationHandler)

	agonesInformer.GameServerAllocations().Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			gsa := obj.(*v1alpha1.GameServerAllocation)
			if gsa.Status.State == v1alpha1.GameServerAllocationUnAllocated {
				c.workerqueue.Enqueue(gsa)
			}
		},
	})

	return c
}

// Run runs this controller. Will block until stop is closed.
// Runs threadiness number workers to process the rate limited queue
// Probably only needs 1 worker, as its just deleting unallocated GameServerAllocations
func (c *Controller) Run(workers int, stop <-chan struct{}) error {
	err := crd.WaitForEstablishedCRD(c.crdGetter, "gameserverallocations."+stable.GroupName, c.logger)
	if err != nil {
		return err
	}

	err = c.counter.Run(stop)
	if err != nil {
		return err
	}

	c.stop = stop

	c.logger.Info("Wait for cache sync")
	if !cache.WaitForCacheSync(stop, c.gameServerAllocationSynced) {
		return errors.New("failed to wait for caches to sync")
	}

	c.workerqueue.Run(workers, stop)
	return nil
}

// creationMutationHandler will intercept when a GameServerAllocation is created, and allocate it a GameServer
// assuming that one is available. If not, it will reject the AdmissionReview.
func (c *Controller) creationMutationHandler(review admv1beta1.AdmissionReview) (admv1beta1.AdmissionReview, error) {
	c.logger.WithField("review", review).Info("creationMutationHandler")
	obj := review.Request.Object
	gsa := &v1alpha1.GameServerAllocation{}

	err := json.Unmarshal(obj.Raw, gsa)
	if err != nil {
		c.logger.WithError(err).Error("error unmarchaslling json")
		return review, errors.Wrapf(err, "error unmarshalling original GameServerAllocation json: %s", obj.Raw)
	}

	gsa.ApplyDefaults()
	gs, err := c.allocate(gsa)
	if err != nil && err != ErrNoGameServerReady {
		return review, err
	}

	if err == ErrNoGameServerReady {
		gsa.Status.State = v1alpha1.GameServerAllocationUnAllocated
	} else {
		// When a GameServer is deleted, the GameServerAllocation should go with it
		ref := metav1.NewControllerRef(gs, v1alpha1.SchemeGroupVersion.WithKind("GameServer"))
		gsa.ObjectMeta.OwnerReferences = append(gsa.ObjectMeta.OwnerReferences, *ref)
		gsa.Status.State = v1alpha1.GameServerAllocationAllocated
		gsa.Status.GameServerName = gs.ObjectMeta.Name
		gsa.Status.Ports = gs.Status.Ports
		gsa.Status.Address = gs.Status.Address
		gsa.Status.NodeName = gs.Status.NodeName
	}

	newFA, err := json.Marshal(gsa)
	if err != nil {
		c.logger.WithError(err).Error("error marshalling")
		return review, errors.Wrapf(err, "error marshalling GameServerAllocation %s to json", gsa.ObjectMeta.Name)
	}

	patch, err := jsonpatch.CreatePatch(obj.Raw, newFA)
	if err != nil {
		c.logger.WithError(err).Error("error creating the patch")
		return review, errors.Wrapf(err, "error creating patch for GameServerAllocation %s", gsa.ObjectMeta.Name)
	}

	json, err := json.Marshal(patch)
	if err != nil {
		c.logger.WithError(err).Error("error creating the json for the patch")
		return review, errors.Wrapf(err, "error creating json for patch for GameServerAllocation %s", gs.ObjectMeta.Name)
	}

	c.logger.WithField("gsa", gsa.ObjectMeta.Name).WithField("patch", string(json)).Infof("patch created!")

	pt := admv1beta1.PatchTypeJSONPatch
	review.Response.PatchType = &pt
	review.Response.Patch = json

	return review, nil
}

// GameServerAllocation fleetName value
// nolint: dupl
func (c *Controller) mutationValidationHandler(review admv1beta1.AdmissionReview) (admv1beta1.AdmissionReview, error) {
	c.logger.WithField("review", review).Info("mutationValidationHandler")

	newGSA := &v1alpha1.GameServerAllocation{}
	oldGSA := &v1alpha1.GameServerAllocation{}

	if err := json.Unmarshal(review.Request.Object.Raw, newGSA); err != nil {
		return review, errors.Wrapf(err, "error unmarshalling new GameServerAllocation json: %s", review.Request.Object.Raw)
	}

	if err := json.Unmarshal(review.Request.OldObject.Raw, oldGSA); err != nil {
		return review, errors.Wrapf(err, "error unmarshalling old GameServerAllocation json: %s", review.Request.Object.Raw)
	}

	if ok, causes := oldGSA.ValidateUpdate(newGSA); !ok {
		review.Response.Allowed = false
		details := metav1.StatusDetails{
			Name:   review.Request.Name,
			Group:  review.Request.Kind.Group,
			Kind:   review.Request.Kind.Kind,
			Causes: causes,
		}
		review.Response.Result = &metav1.Status{
			Status:  metav1.StatusFailure,
			Message: "GameServerAllocation update is invalid",
			Reason:  metav1.StatusReasonInvalid,
			Details: &details,
		}
	}

	return review, nil
}

// allocate allocated a GameServer from a given Fleet
func (c *Controller) allocate(gsa *v1alpha1.GameServerAllocation) (*v1alpha1.GameServer, error) {
	var allocation *v1alpha1.GameServer
	// can only allocate one at a time, as we don't want two separate processes
	// trying to allocate the same GameServer to different clients
	c.allocationMutex.Lock()
	defer c.allocationMutex.Unlock()

	// make sure we have the most up to date view of the world
	if !cache.WaitForCacheSync(c.stop, c.gameServerSynced) {
		return allocation, errors.New("error syncing GameServer cache")
	}

	var comparator findComparator

	switch gsa.Spec.Scheduling {
	case v1alpha1.Packed:
		comparator = packedComparator
	case v1alpha1.Distributed:
		comparator = distributedComparator
	}

	allocation, err := c.findReadyGameServerForAllocation(gsa, comparator)
	if err != nil {
		return allocation, err
	}

	gsCopy := allocation.DeepCopy()
	gsCopy.Status.State = v1alpha1.GameServerStateAllocated

	c.patchMetadata(gsCopy, gsa.Spec.MetaPatch)

	patch, err := allocation.Patch(gsCopy)
	if err != nil {
		return allocation, err
	}

	gs, err := c.gameServerGetter.GameServers(gsCopy.ObjectMeta.Namespace).
		Patch(gsCopy.ObjectMeta.Name, types.JSONPatchType, patch)
	if err != nil {
		return gs, errors.Wrapf(err, "error updating GameServer %s", gsCopy.ObjectMeta.Name)
	}
	c.recorder.Event(gs, corev1.EventTypeNormal, string(gs.Status.State), "Allocated")

	return gs, nil
}

// patch the labels and annotations of an allocated GameServer with metadata from a GameServerAllocation
func (c *Controller) patchMetadata(gs *v1alpha1.GameServer, fam v1alpha1.MetaPatch) {
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

// syncDelete takes unallocated GameServerAllocations, and deletes them!
func (c *Controller) syncDelete(key string) error {
	c.logger.WithField("key", key).Info("Deleting gameserverallocation")
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		// don't return an error, as we don't want this retried
		runtime.HandleError(c.logger.WithField("key", key), errors.Wrapf(err, "invalid resource key"))
		return nil
	}

	err = c.gameServerAllocationGetter.GameServerAllocations(namespace).Delete(name, nil)
	return errors.Wrapf(err, "could not delete GameServerAllocation %s", key)
}

// findReadyGameServerForAllocation returns the most appropriate GameServer from the set, taking into account
// preferred selectors, as well as the passed in comparator
func (c *Controller) findReadyGameServerForAllocation(gsa *v1alpha1.GameServerAllocation, comparator findComparator) (*v1alpha1.GameServer, error) {
	// track the best node count
	var bestCount *NodeCount
	// the current GameServer from the node with the most GameServers (allocated, ready)
	var bestGS *v1alpha1.GameServer

	selector, err := metav1.LabelSelectorAsSelector(&gsa.Spec.Required)
	if err != nil {
		return bestGS, errors.Wrapf(err, "could not convert GameServer %s GameServerAllocation selector", gsa.ObjectMeta.Name)
	}

	gsList, err := c.gameServerLister.List(selector)
	if err != nil {
		return bestGS, errors.Wrapf(err, "could not list GameServers for GameServerAllocation %s", gsa.ObjectMeta.Name)
	}

	preferred, err := gsa.Spec.PreferredSelectors()
	if err != nil {
		return bestGS, errors.Wrapf(err, "could not create preferred selectors for GameServerAllocation %s", gsa.ObjectMeta.Name)
	}

	counts := c.counter.Counts()

	// track potential GameServers, one for each node
	allocatableRequired := map[string]*v1alpha1.GameServer{}
	allocatablePreferred := make([]map[string]*v1alpha1.GameServer, len(preferred))

	// build the index of possible allocatable GameServers
	for _, gs := range gsList {
		if gs.DeletionTimestamp.IsZero() && gs.Status.State == v1alpha1.GameServerStateReady {
			allocatableRequired[gs.Status.NodeName] = gs

			for i, p := range preferred {
				if p.Matches(labels.Set(gs.Labels)) {
					if allocatablePreferred[i] == nil {
						allocatablePreferred[i] = map[string]*v1alpha1.GameServer{}
					}
					allocatablePreferred[i][gs.Status.NodeName] = gs
				}
			}
		}
	}

	allocationSet := allocatableRequired

	// check if there is any preferred options available
	for _, set := range allocatablePreferred {
		if len(set) > 0 {
			allocationSet = set
			break
		}
	}

	for nodeName, gs := range allocationSet {
		count := counts[nodeName]
		// bestGS == nil: if there is no best GameServer, then this node & GameServer is the always the best
		if bestGS == nil || comparator(*bestCount, count) {
			bestCount = &count
			bestGS = gs
		}
	}

	if bestGS == nil {
		err = ErrNoGameServerReady
	}

	return bestGS, err
}
