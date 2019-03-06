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
	"math/rand"
	"strconv"
	"sync"
	"time"

	"agones.dev/agones/pkg/apis/stable"
	"agones.dev/agones/pkg/apis/stable/v1alpha1"
	"agones.dev/agones/pkg/client/clientset/versioned"
	getterv1alpha1 "agones.dev/agones/pkg/client/clientset/versioned/typed/stable/v1alpha1"
	"agones.dev/agones/pkg/client/informers/externalversions"
	listerv1alpha1 "agones.dev/agones/pkg/client/listers/stable/v1alpha1"
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
	corev1 "k8s.io/api/core/v1"
	extclientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/typed/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/wait"
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
	// ErrConflictInGameServerSelection is returned when the candidate gameserver already allocated
	ErrConflictInGameServerSelection = errors.New("The Gameserver was already allocated")
)

// Controller is a the GameServerAllocation controller
type Controller struct {
	baseLogger                 *logrus.Entry
	counter                    *AllocationCounter
	crdGetter                  v1beta1.CustomResourceDefinitionInterface
	gameServerSynced           cache.InformerSynced
	gameServerGetter           getterv1alpha1.GameServersGetter
	gameServerLister           listerv1alpha1.GameServerLister
	gameServerAllocationSynced cache.InformerSynced
	gameServerAllocationGetter getterv1alpha1.GameServerAllocationsGetter
	stop                       <-chan struct{}
	workerqueue                *workerqueue.WorkerQueue
	gsWorkerqueue              *workerqueue.WorkerQueue
	recorder                   record.EventRecorder
	readyGameServers           gameServerCacheEntry
	// Instead of selecting the top one, controller selects a random one
	// from the topNGameServerCount of Ready gameservers
	topNGameServerCount int
}

// gameserver cache to keep the Ready state gameserver.
type gameServerCacheEntry struct {
	mu    sync.RWMutex
	cache map[string]*v1alpha1.GameServer
}

// Store saves the data in the cache.
func (e *gameServerCacheEntry) Store(key string, gs *v1alpha1.GameServer) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.cache == nil {
		e.cache = map[string]*v1alpha1.GameServer{}
	}
	e.cache[key] = gs.DeepCopy()
}

// Delete deletes the data. If it exists returns true.
func (e *gameServerCacheEntry) Delete(key string) bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	ret := false
	if e.cache != nil {
		if _, ok := e.cache[key]; ok {
			delete(e.cache, key)
			ret = true
		}
	}

	return ret
}

// Load returns the data from cache. It return true if the value exists in the cache
func (e *gameServerCacheEntry) Load(key string) (*v1alpha1.GameServer, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	val, ok := e.cache[key]

	return val, ok
}

// Range extracts data from the cache based on provided function f.
func (e *gameServerCacheEntry) Range(f func(key string, gs *v1alpha1.GameServer) bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	for k, v := range e.cache {
		if !f(k, v) {
			break
		}
	}
}

// findComparator is a comparator function specifically for the
// findReadyGameServerForAllocation method for determining
// scheduling strategy
type findComparator func(bestCount, currentCount NodeCount) bool

var allocationRetry = wait.Backoff{
	Steps:    5,
	Duration: 10 * time.Millisecond,
	Factor:   1.0,
	Jitter:   0.1,
}

// NewController returns a controller for a GameServerAllocation
func NewController(wh *webhooks.WebHook,
	health healthcheck.Handler,
	kubeClient kubernetes.Interface,
	kubeInformerFactory informers.SharedInformerFactory,
	extClient extclientset.Interface,
	agonesClient versioned.Interface,
	agonesInformerFactory externalversions.SharedInformerFactory,
	topNGameServerCnt int,
) *Controller {

	agonesInformer := agonesInformerFactory.Stable().V1alpha1()
	c := &Controller{
		counter:                    NewAllocationCounter(kubeInformerFactory, agonesInformerFactory),
		crdGetter:                  extClient.ApiextensionsV1beta1().CustomResourceDefinitions(),
		gameServerSynced:           agonesInformer.GameServers().Informer().HasSynced,
		gameServerGetter:           agonesClient.StableV1alpha1(),
		gameServerLister:           agonesInformer.GameServers().Lister(),
		gameServerAllocationSynced: agonesInformer.GameServerAllocations().Informer().HasSynced,
		gameServerAllocationGetter: agonesClient.StableV1alpha1(),
		topNGameServerCount:        topNGameServerCnt,
	}
	c.baseLogger = runtime.NewLoggerWithType(c)
	c.workerqueue = workerqueue.NewWorkerQueue(c.syncDelete, c.baseLogger, logfields.GameServerAllocationKey, stable.GroupName+".GameServerAllocationController")
	c.gsWorkerqueue = workerqueue.NewWorkerQueue(c.syncGameServers, c.baseLogger, logfields.GameServerKey, stable.GroupName+".GameServerUpdateController")
	health.AddLivenessCheck("gameserverallocation-workerqueue", healthcheck.Check(c.workerqueue.Healthy))
	health.AddLivenessCheck("gameserverallocation-gameserver-workerqueue", healthcheck.Check(c.gsWorkerqueue.Healthy))

	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(c.baseLogger.Infof)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: kubeClient.CoreV1().Events("")})
	c.recorder = eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: "GameServerAllocation-controller"})

	kind := v1alpha1.Kind("GameServerAllocation")
	wh.AddHandler("/mutate", kind, admv1beta1.Create, c.creationMutationHandler)
	wh.AddHandler("/validate", kind, admv1beta1.Update, c.mutationValidationHandler)

	agonesInformer.GameServerAllocations().Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			gsa := obj.(*v1alpha1.GameServerAllocation)
			if gsa.Status.State == v1alpha1.GameServerAllocationUnAllocated || gsa.Status.State == v1alpha1.GameServerAllocationContention {
				c.workerqueue.Enqueue(gsa)
			}
		},
	})

	agonesInformer.GameServers().Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		UpdateFunc: func(oldObj, newObj interface{}) {
			// only interested in if the old / new state was/is Ready
			oldGs := oldObj.(*v1alpha1.GameServer)
			newGs := newObj.(*v1alpha1.GameServer)
			if oldGs.Status.State == v1alpha1.GameServerStateReady || newGs.Status.State == v1alpha1.GameServerStateReady {
				if key, ok := c.getKey(newGs); ok {
					if newGs.Status.State == v1alpha1.GameServerStateReady {
						c.readyGameServers.Store(key, newGs)
					} else {
						c.readyGameServers.Delete(key)
					}
				}
			}
		},
	})

	return c
}

// Run runs this controller. Will block until stop is closed.
// Runs threadiness number workers to process the rate limited queue
// Probably only needs 1 worker, as its just deleting unallocated GameServerAllocations
func (c *Controller) Run(workers int, stop <-chan struct{}) error {
	err := crd.WaitForEstablishedCRD(c.crdGetter, "gameserverallocations."+stable.GroupName, c.baseLogger)
	if err != nil {
		return err
	}

	err = c.counter.Run(stop)
	if err != nil {
		return err
	}

	c.stop = stop

	c.baseLogger.Info("Wait for cache sync")
	if !cache.WaitForCacheSync(stop, c.gameServerAllocationSynced, c.gameServerSynced) {
		return errors.New("failed to wait for caches to sync")
	}

	// build the cache
	err = c.syncReadyGSServerCache()
	if err != nil {
		return err
	}

	c.workerqueue.Run(workers, stop)

	// we don't want mutiple workers refresh cache at the same time so one worker will be better.
	// Also we don't expect to have too many failures when allocating
	c.gsWorkerqueue.Run(1, stop)

	return nil
}

func (c *Controller) loggerForGameServerKey(key string) *logrus.Entry {
	return logfields.AugmentLogEntry(c.baseLogger, logfields.GameServerKey, key)
}

func (c *Controller) loggerForGameServerAllocationKey(key string) *logrus.Entry {
	return logfields.AugmentLogEntry(c.baseLogger, logfields.GameServerAllocationKey, key)
}

func (c *Controller) loggerForGameServerAllocation(gsa *v1alpha1.GameServerAllocation) *logrus.Entry {
	gsaName := "NilGameServerAllocation"
	if gsa != nil {
		gsaName = gsa.Namespace + "/" + gsa.Name
	}
	return c.loggerForGameServerAllocationKey(gsaName).WithField("gsa", gsa)
}

// creationMutationHandler will intercept when a GameServerAllocation is created, and allocate it a GameServer
// assuming that one is available. If not, it will reject the AdmissionReview.
func (c *Controller) creationMutationHandler(review admv1beta1.AdmissionReview) (admv1beta1.AdmissionReview, error) {
	c.baseLogger.WithField("review", review).Info("creationMutationHandler")
	obj := review.Request.Object
	gsa := &v1alpha1.GameServerAllocation{}

	err := json.Unmarshal(obj.Raw, gsa)
	if err != nil {
		c.baseLogger.WithError(err).Error("error unmarshalling json")
		return review, errors.Wrapf(err, "error unmarshalling original GameServerAllocation json: %s", obj.Raw)
	}

	gsa.ApplyDefaults()
	var gs *v1alpha1.GameServer
	err = Retry(allocationRetry, func() error {
		gs, err = c.allocate(gsa)
		return err
	})

	if err != nil && err != ErrNoGameServerReady && err != ErrConflictInGameServerSelection {
		// this will trigger syncing of the cache (assuming cache might not be up to date)
		c.gsWorkerqueue.EnqueueImmediately(gs)

		return review, err
	}

	if err == ErrNoGameServerReady {
		gsa.Status.State = v1alpha1.GameServerAllocationUnAllocated
	} else if err == ErrConflictInGameServerSelection {
		gsa.Status.State = v1alpha1.GameServerAllocationContention
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
		c.baseLogger.WithError(err).Error("error marshalling")
		return review, errors.Wrapf(err, "error marshalling GameServerAllocation %s to json", gsa.ObjectMeta.Name)
	}

	patch, err := jsonpatch.CreatePatch(obj.Raw, newFA)
	if err != nil {
		c.baseLogger.WithError(err).Error("error creating the patch")
		return review, errors.Wrapf(err, "error creating patch for GameServerAllocation %s", gsa.ObjectMeta.Name)
	}

	json, err := json.Marshal(patch)
	if err != nil {
		c.baseLogger.WithError(err).Error("error creating the json for the patch")
		return review, errors.Wrapf(err, "error creating json for patch for GameServerAllocation %s", gs.ObjectMeta.Name)
	}

	c.loggerForGameServerAllocation(gsa).WithField("patch", string(json)).Infof("patch created!")

	pt := admv1beta1.PatchTypeJSONPatch
	review.Response.PatchType = &pt
	review.Response.Patch = json

	return review, nil
}

// GameServerAllocation fleetName value
// nolint: dupl
func (c *Controller) mutationValidationHandler(review admv1beta1.AdmissionReview) (admv1beta1.AdmissionReview, error) {
	c.baseLogger.WithField("review", review).Info("mutationValidationHandler")

	newGSA := &v1alpha1.GameServerAllocation{}
	oldGSA := &v1alpha1.GameServerAllocation{}

	if err := json.Unmarshal(review.Request.Object.Raw, newGSA); err != nil {
		return review, errors.Wrapf(err, "error unmarshalling new GameServerAllocation json: %s", review.Request.Object.Raw)
	}

	if err := json.Unmarshal(review.Request.OldObject.Raw, oldGSA); err != nil {
		return review, errors.Wrapf(err, "error unmarshalling old GameServerAllocation json: %s", review.Request.Object.Raw)
	}

	if causes, ok := oldGSA.ValidateUpdate(newGSA); !ok {
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

	key, _ := cache.MetaNamespaceKeyFunc(allocation)
	if ok := c.readyGameServers.Delete(key); !ok {
		return allocation, ErrConflictInGameServerSelection
	}

	gsCopy := allocation.DeepCopy()
	gsCopy.Status.State = v1alpha1.GameServerStateAllocated

	c.patchMetadata(gsCopy, gsa.Spec.MetaPatch)

	gs, err := c.gameServerGetter.GameServers(gsCopy.ObjectMeta.Namespace).Update(gsCopy)

	if err != nil {
		// since we could not allocate, we should put it back
		c.readyGameServers.Store(key, gs)
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
	c.loggerForGameServerAllocationKey(key).Info("Deleting gameserverallocation")
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		// don't return an error, as we don't want this retried
		runtime.HandleError(c.loggerForGameServerAllocationKey(key), errors.Wrapf(err, "invalid resource key"))
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

	gsList := c.selectGameServers(selector)

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

	bestGSList := []v1alpha1.GameServer{}
	for nodeName, gs := range allocationSet {
		count := counts[nodeName]
		// bestGS == nil: if there is no best GameServer, then this node & GameServer is the always the best
		if bestGS == nil || comparator(*bestCount, count) {
			bestCount = &count
			bestGS = gs
			bestGSList = append(bestGSList, *gs)
		}
	}

	if bestGS == nil {
		err = ErrNoGameServerReady
	} else {
		bestGS = c.getRandomlySelectedGS(gsa, bestGSList)
	}

	return bestGS, err
}

// syncGameServers synchronises the GameServers to Gameserver cache. This is called when a failure
// happened during the allocation. This method will sync and make sure the cache is up to date.
func (c *Controller) syncGameServers(key string) error {
	c.loggerForGameServerKey(key).Info("Refreshing Ready Gameserver cache")

	return c.syncReadyGSServerCache()
}

// syncReadyGSServerCache syncs the gameserver cache and updates the local cache for any changes.
func (c *Controller) syncReadyGSServerCache() error {
	c.baseLogger.Info("Wait for cache sync")
	if !cache.WaitForCacheSync(c.stop, c.gameServerSynced) {
		return errors.New("failed to wait for cache to sync")
	}

	// build the cache
	gsList, err := c.gameServerLister.List(labels.Everything())
	if err != nil {
		return errors.Wrap(err, "could not list GameServers")
	}

	// convert list of current gameservers to map for faster access
	currGameservers := make(map[string]*v1alpha1.GameServer)
	for _, gs := range gsList {
		if key, ok := c.getKey(gs); ok {
			currGameservers[key] = gs
		}
	}

	// first remove the gameservers are not in the list anymore
	tobeDeletedGSInCache := make([]string, 0)
	c.readyGameServers.Range(func(key string, gs *v1alpha1.GameServer) bool {
		if _, ok := currGameservers[key]; !ok {
			tobeDeletedGSInCache = append(tobeDeletedGSInCache, key)
		}
		return true
	})

	for _, staleGSKey := range tobeDeletedGSInCache {
		c.readyGameServers.Delete(staleGSKey)
	}

	// refresh the cache of possible allocatable GameServers
	for key, gs := range currGameservers {
		if gsCache, ok := c.readyGameServers.Load(key); ok {
			if !(gs.DeletionTimestamp.IsZero() && gs.Status.State == v1alpha1.GameServerStateReady) {
				c.readyGameServers.Delete(key)
			} else if gs.ObjectMeta.ResourceVersion != gsCache.ObjectMeta.ResourceVersion {
				c.readyGameServers.Store(key, gs)
			}
		} else if gs.DeletionTimestamp.IsZero() && gs.Status.State == v1alpha1.GameServerStateReady {
			c.readyGameServers.Store(key, gs)
		}
	}

	return nil
}

// selectGameServers selects the appropriate gameservers from cache based on selector.
func (c *Controller) selectGameServers(selector labels.Selector) (res []*v1alpha1.GameServer) {
	c.readyGameServers.Range(func(key string, gs *v1alpha1.GameServer) bool {
		if selector.Matches(labels.Set(gs.ObjectMeta.GetLabels())) {
			res = append(res, gs)
		}
		return true
	})
	return res
}

// getKey extract the key of gameserver object
func (c *Controller) getKey(gs *v1alpha1.GameServer) (string, bool) {
	var key string
	ok := true
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(gs); err != nil {
		ok = false
		err = errors.Wrap(err, "Error creating key for object")
		runtime.HandleError(c.baseLogger.WithField("obj", gs), err)
	}
	return key, ok
}

// Retry retries fn based on backoff provided.
func Retry(backoff wait.Backoff, fn func() error) error {
	var lastConflictErr error
	err := wait.ExponentialBackoff(backoff, func() (bool, error) {
		err := fn()
		switch {
		case err == nil:
			return true, nil
		case err == ErrNoGameServerReady:
			return true, err
		default:
			lastConflictErr = err
			return false, nil
		}
	})
	if err == wait.ErrWaitTimeout {
		err = lastConflictErr
	}
	return err
}

// getRandomlySelectedGS selects a GS from the set of Gameservers randomly. This will reduce the contentions
func (c *Controller) getRandomlySelectedGS(gsa *v1alpha1.GameServerAllocation, bestGSList []v1alpha1.GameServer) *v1alpha1.GameServer {
	seed, err := strconv.Atoi(gsa.ObjectMeta.ResourceVersion)
	if err != nil {
		seed = 1234567
	}

	ln := c.topNGameServerCount
	if ln > len(bestGSList) {
		ln = len(bestGSList)
	}

	startIndex := len(bestGSList) - ln
	bestGSList = bestGSList[startIndex:]
	index := rand.New(rand.NewSource(int64(seed))).Intn(ln)
	return &bestGSList[index]
}
