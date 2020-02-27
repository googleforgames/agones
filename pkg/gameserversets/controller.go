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

package gameserversets

import (
	"encoding/json"
	"sync"
	"time"

	"agones.dev/agones/pkg/apis"
	"agones.dev/agones/pkg/apis/agones"
	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	"agones.dev/agones/pkg/client/clientset/versioned"
	getterv1 "agones.dev/agones/pkg/client/clientset/versioned/typed/agones/v1"
	"agones.dev/agones/pkg/client/informers/externalversions"
	listerv1 "agones.dev/agones/pkg/client/listers/agones/v1"
	"agones.dev/agones/pkg/gameservers"
	"agones.dev/agones/pkg/util/crd"
	"agones.dev/agones/pkg/util/logfields"
	"agones.dev/agones/pkg/util/runtime"
	"agones.dev/agones/pkg/util/webhooks"
	"agones.dev/agones/pkg/util/workerqueue"
	"github.com/heptiolabs/healthcheck"
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
	// ErrNoGameServerSetOwner is returned when a GameServerSet can't be found as an owner
	// for a GameServer
	ErrNoGameServerSetOwner = errors.New("No GameServerSet owner for this GameServer")
)

const (
	maxCreationParalellism         = 16
	maxGameServerCreationsPerBatch = 64

	maxDeletionParallelism         = 64
	maxGameServerDeletionsPerBatch = 64

	// maxPodPendingCount is the maximum number of pending pods per game server set
	maxPodPendingCount = 5000
)

// Controller is a the GameServerSet controller
type Controller struct {
	baseLogger          *logrus.Entry
	counter             *gameservers.PerNodeCounter
	crdGetter           v1beta1.CustomResourceDefinitionInterface
	gameServerGetter    getterv1.GameServersGetter
	gameServerLister    listerv1.GameServerLister
	gameServerSynced    cache.InformerSynced
	gameServerSetGetter getterv1.GameServerSetsGetter
	gameServerSetLister listerv1.GameServerSetLister
	gameServerSetSynced cache.InformerSynced
	workerqueue         *workerqueue.WorkerQueue
	stop                <-chan struct{}
	recorder            record.EventRecorder
	stateCache          *gameServerStateCache
}

// NewController returns a new gameserverset crd controller
func NewController(
	wh *webhooks.WebHook,
	health healthcheck.Handler,
	counter *gameservers.PerNodeCounter,
	kubeClient kubernetes.Interface,
	extClient extclientset.Interface,
	agonesClient versioned.Interface,
	agonesInformerFactory externalversions.SharedInformerFactory) *Controller {

	gameServers := agonesInformerFactory.Agones().V1().GameServers()
	gsInformer := gameServers.Informer()
	gameServerSets := agonesInformerFactory.Agones().V1().GameServerSets()
	gsSetInformer := gameServerSets.Informer()

	c := &Controller{
		crdGetter:           extClient.ApiextensionsV1beta1().CustomResourceDefinitions(),
		counter:             counter,
		gameServerGetter:    agonesClient.AgonesV1(),
		gameServerLister:    gameServers.Lister(),
		gameServerSynced:    gsInformer.HasSynced,
		gameServerSetGetter: agonesClient.AgonesV1(),
		gameServerSetLister: gameServerSets.Lister(),
		gameServerSetSynced: gsSetInformer.HasSynced,
		stateCache:          &gameServerStateCache{},
	}

	c.baseLogger = runtime.NewLoggerWithType(c)
	c.workerqueue = workerqueue.NewWorkerQueueWithRateLimiter(c.syncGameServerSet, c.baseLogger, logfields.GameServerSetKey, agones.GroupName+".GameServerSetController", workerqueue.FastRateLimiter(3*time.Second))
	health.AddLivenessCheck("gameserverset-workerqueue", healthcheck.Check(c.workerqueue.Healthy))

	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(c.baseLogger.Debugf)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: kubeClient.CoreV1().Events("")})
	c.recorder = eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: "gameserverset-controller"})

	wh.AddHandler("/validate", agonesv1.Kind("GameServerSet"), admv1beta1.Create, c.creationValidationHandler)
	wh.AddHandler("/validate", agonesv1.Kind("GameServerSet"), admv1beta1.Update, c.updateValidationHandler)

	gsSetInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: c.workerqueue.Enqueue,
		UpdateFunc: func(oldObj, newObj interface{}) {
			oldGss := oldObj.(*agonesv1.GameServerSet)
			newGss := newObj.(*agonesv1.GameServerSet)
			if oldGss.Spec.Replicas != newGss.Spec.Replicas {
				c.workerqueue.Enqueue(newGss)
			}
		},
		DeleteFunc: func(gsSet interface{}) {
			c.stateCache.deleteGameServerSet(gsSet.(*agonesv1.GameServerSet))
		},
	})

	gsInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: c.gameServerEventHandler,
		UpdateFunc: func(oldObj, newObj interface{}) {
			gs := newObj.(*agonesv1.GameServer)
			// ignore if already being deleted
			if gs.ObjectMeta.DeletionTimestamp == nil {
				c.gameServerEventHandler(gs)
			}
		},
		DeleteFunc: c.gameServerEventHandler,
	})

	return c
}

// Run the GameServerSet controller. Will block until stop is closed.
// Runs threadiness number workers to process the rate limited queue
func (c *Controller) Run(workers int, stop <-chan struct{}) error {
	c.stop = stop

	err := crd.WaitForEstablishedCRD(c.crdGetter, "gameserversets."+agones.GroupName, c.baseLogger)
	if err != nil {
		return err
	}

	c.baseLogger.Debug("Wait for cache sync")
	if !cache.WaitForCacheSync(stop, c.gameServerSynced, c.gameServerSetSynced) {
		return errors.New("failed to wait for caches to sync")
	}

	c.workerqueue.Run(workers, stop)
	return nil
}

// updateValidationHandler that validates a GameServerSet when is updated
// Should only be called on gameserverset update operations.
func (c *Controller) updateValidationHandler(review admv1beta1.AdmissionReview) (admv1beta1.AdmissionReview, error) {
	c.baseLogger.WithField("review", review).Debug("updateValidationHandler")

	newGss := &agonesv1.GameServerSet{}
	oldGss := &agonesv1.GameServerSet{}

	newObj := review.Request.Object
	if err := json.Unmarshal(newObj.Raw, newGss); err != nil {
		return review, errors.Wrapf(err, "error unmarshalling new GameServerSet json: %s", newObj.Raw)
	}

	oldObj := review.Request.OldObject
	if err := json.Unmarshal(oldObj.Raw, oldGss); err != nil {
		return review, errors.Wrapf(err, "error unmarshalling old GameServerSet json: %s", oldObj.Raw)
	}

	causes, ok := oldGss.ValidateUpdate(newGss)
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
			Message: "GameServerSet update is invalid",
			Reason:  metav1.StatusReasonInvalid,
			Details: &details,
		}

		c.loggerForGameServerSet(newGss).WithField("review", review).Info("Invalid GameServerSet update")
		return review, nil
	}

	return review, nil
}

// creationValidationHandler that validates a GameServerSet when is created
// Should only be called on gameserverset create operations.
func (c *Controller) creationValidationHandler(review admv1beta1.AdmissionReview) (admv1beta1.AdmissionReview, error) {
	c.baseLogger.WithField("review", review).Debug("creationValidationHandler")

	newGss := &agonesv1.GameServerSet{}

	newObj := review.Request.Object
	if err := json.Unmarshal(newObj.Raw, newGss); err != nil {
		return review, errors.Wrapf(err, "error unmarshalling new GameServerSet json: %s", newObj.Raw)
	}

	causes, ok := newGss.Validate()
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
			Message: "GameServerSet update is invalid",
			Reason:  metav1.StatusReasonInvalid,
			Details: &details,
		}

		c.loggerForGameServerSet(newGss).WithField("review", review).Debug("Invalid GameServerSet update")
		return review, nil
	}

	return review, nil
}

func (c *Controller) gameServerEventHandler(obj interface{}) {
	gs, ok := obj.(*agonesv1.GameServer)
	if !ok {
		return
	}

	ref := metav1.GetControllerOf(gs)
	if ref == nil {
		return
	}
	gsSet, err := c.gameServerSetLister.GameServerSets(gs.ObjectMeta.Namespace).Get(ref.Name)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			c.baseLogger.WithField("ref", ref).Debug("Owner GameServerSet no longer available for syncing")
		} else {
			runtime.HandleError(c.baseLogger.WithField("gsKey", gs.ObjectMeta.Namespace+"/"+gs.ObjectMeta.Name).WithField("ref", ref),
				errors.Wrap(err, "error retrieving GameServer owner"))
		}
		return
	}
	c.workerqueue.EnqueueImmediately(gsSet)
}

func (c *Controller) loggerForGameServerSetKey(key string) *logrus.Entry {
	return logfields.AugmentLogEntry(c.baseLogger, logfields.GameServerSetKey, key)
}

func (c *Controller) loggerForGameServerSet(gsSet *agonesv1.GameServerSet) *logrus.Entry {
	gsSetName := "NilGameServerSet"
	if gsSet != nil {
		gsSetName = gsSet.Namespace + "/" + gsSet.Name
	}
	return c.loggerForGameServerSetKey(gsSetName).WithField("gss", gsSet)
}

// syncGameServer synchronises the GameServers for the Set,
// making sure there are aways as many GameServers as requested
func (c *Controller) syncGameServerSet(key string) error {
	// Convert the namespace/name string into a distinct namespace and name
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		// don't return an error, as we don't want this retried
		runtime.HandleError(c.loggerForGameServerSetKey(key), errors.Wrapf(err, "invalid resource key"))
		return nil
	}

	gsSet, err := c.gameServerSetLister.GameServerSets(namespace).Get(name)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			c.loggerForGameServerSetKey(key).Debug("GameServerSet is no longer available for syncing")
			return nil
		}
		return errors.Wrapf(err, "error retrieving GameServerSet %s from namespace %s", name, namespace)
	}

	list, err := ListGameServersByGameServerSetOwner(c.gameServerLister, gsSet)
	if err != nil {
		return err
	}

	list = c.stateCache.forGameServerSet(gsSet).reconcileWithUpdatedServerList(list)

	numServersToAdd, toDelete, isPartial := computeReconciliationAction(gsSet.Spec.Scheduling, list, c.counter.Counts(),
		int(gsSet.Spec.Replicas), maxGameServerCreationsPerBatch, maxGameServerDeletionsPerBatch, maxPodPendingCount)
	status := computeStatus(list)
	fields := logrus.Fields{}

	for _, gs := range list {
		key := "gsCount" + string(gs.Status.State)
		if gs.ObjectMeta.DeletionTimestamp != nil {
			key += "Deleted"
		}
		v, ok := fields[key]
		if !ok {
			v = 0
		}

		fields[key] = v.(int) + 1
	}
	c.loggerForGameServerSet(gsSet).
		WithField("targetReplicaCount", gsSet.Spec.Replicas).
		WithField("numServersToAdd", numServersToAdd).
		WithField("numServersToDelete", len(toDelete)).
		WithField("isPartial", isPartial).
		WithField("status", status).
		WithFields(fields).
		Debug("Reconciling GameServerSet")
	if isPartial {
		// we've determined that there's work to do, but we've decided not to do all the work in one shot
		// make sure we get a follow-up, by re-scheduling this GSS in the worker queue immediately before this
		// function returns
		defer c.workerqueue.EnqueueImmediately(gsSet)
	}

	if numServersToAdd > 0 {
		if err := c.addMoreGameServers(gsSet, numServersToAdd); err != nil {
			c.loggerForGameServerSet(gsSet).WithError(err).Warning("error adding game servers")
		}
	}

	if len(toDelete) > 0 {
		if err := c.deleteGameServers(gsSet, toDelete); err != nil {
			c.loggerForGameServerSet(gsSet).WithError(err).Warning("error deleting game servers")
		}
	}

	return c.syncGameServerSetStatus(gsSet, list)
}

// computeReconciliationAction computes the action to take to reconcile a game server set set given
// the list of game servers that were found and target replica count.
func computeReconciliationAction(strategy apis.SchedulingStrategy, list []*agonesv1.GameServer,
	counts map[string]gameservers.NodeCount, targetReplicaCount int, maxCreations int, maxDeletions int,
	maxPending int) (int, []*agonesv1.GameServer, bool) {
	var upCount int     // up == Ready or will become ready
	var deleteCount int // number of gameservers to delete

	// track the number of pods that are being created at any given moment by the GameServerSet
	// so we can limit it at a throughput that Kubernetes can handle
	var podPendingCount int // podPending == "up" but don't have a Pod running yet

	var potentialDeletions []*agonesv1.GameServer
	var toDelete []*agonesv1.GameServer

	scheduleDeletion := func(gs *agonesv1.GameServer) {
		toDelete = append(toDelete, gs)
		deleteCount--
	}

	handleGameServerUp := func(gs *agonesv1.GameServer) {
		if upCount >= targetReplicaCount {
			deleteCount++
		} else {
			upCount++
		}

		// Track gameservers that could be potentially deleted
		potentialDeletions = append(potentialDeletions, gs)
	}

	// pass 1 - count allocated/reserved servers only, since those can't be touched
	for _, gs := range list {
		if !gs.IsDeletable() {
			upCount++
		}
	}

	// pass 2 - handle all other statuses
	for _, gs := range list {
		if !gs.IsDeletable() {
			// already handled above
			continue
		}

		// GS being deleted don't count.
		if gs.IsBeingDeleted() {
			continue
		}

		switch gs.Status.State {
		case agonesv1.GameServerStatePortAllocation:
			podPendingCount++
			handleGameServerUp(gs)
		case agonesv1.GameServerStateCreating:
			podPendingCount++
			handleGameServerUp(gs)
		case agonesv1.GameServerStateStarting:
			podPendingCount++
			handleGameServerUp(gs)
		case agonesv1.GameServerStateScheduled:
			podPendingCount++
			handleGameServerUp(gs)
		case agonesv1.GameServerStateRequestReady:
			handleGameServerUp(gs)
		case agonesv1.GameServerStateReady:
			handleGameServerUp(gs)
		case agonesv1.GameServerStateReserved:
			handleGameServerUp(gs)

		// GameServerStateShutdown - already handled above
		// GameServerStateAllocated - already handled above
		case agonesv1.GameServerStateError, agonesv1.GameServerStateUnhealthy:
			scheduleDeletion(gs)
		default:
			// unrecognized state, assume it's up.
			handleGameServerUp(gs)
		}
	}

	var partialReconciliation bool
	var numServersToAdd int

	if upCount < targetReplicaCount {
		numServersToAdd = targetReplicaCount - upCount
		originalNumServersToAdd := numServersToAdd

		if numServersToAdd > maxCreations {
			numServersToAdd = maxCreations
		}

		if numServersToAdd+podPendingCount > maxPending {
			numServersToAdd = maxPending - podPendingCount
			if numServersToAdd < 0 {
				numServersToAdd = 0
			}
		}

		if originalNumServersToAdd != numServersToAdd {
			partialReconciliation = true
		}
	}

	if deleteCount > 0 {
		if strategy == apis.Packed {
			potentialDeletions = sortGameServersByLeastFullNodes(potentialDeletions, counts)
		} else {
			potentialDeletions = sortGameServersByNewFirst(potentialDeletions)
		}

		toDelete = append(toDelete, potentialDeletions[0:deleteCount]...)
	}

	if len(toDelete) > maxDeletions {
		toDelete = toDelete[0:maxDeletions]
		partialReconciliation = true
	}

	return numServersToAdd, toDelete, partialReconciliation
}

// addMoreGameServers adds diff more GameServers to the set
func (c *Controller) addMoreGameServers(gsSet *agonesv1.GameServerSet, count int) error {
	c.loggerForGameServerSet(gsSet).WithField("count", count).Info("Adding more gameservers")

	return parallelize(newGameServersChannel(count, gsSet), maxCreationParalellism, func(gs *agonesv1.GameServer) error {
		gs, err := c.gameServerGetter.GameServers(gs.Namespace).Create(gs)
		if err != nil {
			return errors.Wrapf(err, "error creating gameserver for gameserverset %s", gsSet.ObjectMeta.Name)
		}

		c.stateCache.forGameServerSet(gsSet).created(gs)
		c.recorder.Eventf(gsSet, corev1.EventTypeNormal, "SuccessfulCreate", "Created gameserver: %s", gs.ObjectMeta.Name)
		return nil
	})
}

func (c *Controller) deleteGameServers(gsSet *agonesv1.GameServerSet, toDelete []*agonesv1.GameServer) error {
	c.loggerForGameServerSet(gsSet).WithField("diff", len(toDelete)).Info("Deleting gameservers")

	return parallelize(gameServerListToChannel(toDelete), maxDeletionParallelism, func(gs *agonesv1.GameServer) error {
		// We should not delete the gameservers directly buy set their state to shutdown and let the gameserver controller to delete
		gsCopy := gs.DeepCopy()
		gsCopy.Status.State = agonesv1.GameServerStateShutdown
		_, err := c.gameServerGetter.GameServers(gs.Namespace).Update(gsCopy)
		if err != nil {
			return errors.Wrapf(err, "error updating gameserver %s from status %s to Shutdown status.", gs.ObjectMeta.Name, gs.Status.State)
		}

		c.stateCache.forGameServerSet(gsSet).deleted(gs)
		c.recorder.Eventf(gsSet, corev1.EventTypeNormal, "SuccessfulDelete", "Deleted gameserver in state %s: %v", gs.Status.State, gs.ObjectMeta.Name)
		return nil
	})
}

func newGameServersChannel(n int, gsSet *agonesv1.GameServerSet) chan *agonesv1.GameServer {
	gameServers := make(chan *agonesv1.GameServer)
	go func() {
		defer close(gameServers)

		for i := 0; i < n; i++ {
			gameServers <- gsSet.GameServer()
		}
	}()

	return gameServers
}

func gameServerListToChannel(list []*agonesv1.GameServer) chan *agonesv1.GameServer {
	gameServers := make(chan *agonesv1.GameServer)
	go func() {
		defer close(gameServers)

		for _, gs := range list {
			gameServers <- gs
		}
	}()

	return gameServers
}

// parallelize processes a channel of game server objects, invoking the provided callback for items in the channel with the specified degree of parallelism up to a limit.
// Returns nil if all callbacks returned nil or one of the error responses, not necessarily the first one.
func parallelize(gameServers chan *agonesv1.GameServer, parallelism int, work func(gs *agonesv1.GameServer) error) error {
	errch := make(chan error, parallelism)

	var wg sync.WaitGroup

	for i := 0; i < parallelism; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()
			for it := range gameServers {
				err := work(it)
				if err != nil {
					errch <- err
					break
				}
			}
		}()
	}
	wg.Wait()
	close(errch)

	for range gameServers {
		// drain any remaining game servers in the channel, in case we did not consume them all
	}

	// return first error from the channel, or nil if all successful.
	return <-errch
}

// syncGameServerSetStatus synchronises the GameServerSet State with active GameServer counts
func (c *Controller) syncGameServerSetStatus(gsSet *agonesv1.GameServerSet, list []*agonesv1.GameServer) error {
	return c.updateStatusIfChanged(gsSet, computeStatus(list))
}

// updateStatusIfChanged updates GameServerSet status if it's different than provided.
func (c *Controller) updateStatusIfChanged(gsSet *agonesv1.GameServerSet, status agonesv1.GameServerSetStatus) error {
	if gsSet.Status != status {
		gsSetCopy := gsSet.DeepCopy()
		gsSetCopy.Status = status
		_, err := c.gameServerSetGetter.GameServerSets(gsSet.ObjectMeta.Namespace).UpdateStatus(gsSetCopy)
		if err != nil {
			return errors.Wrapf(err, "error updating status on GameServerSet %s", gsSet.ObjectMeta.Name)
		}
	}
	return nil
}

// computeStatus computes the status of the game server set.
func computeStatus(list []*agonesv1.GameServer) agonesv1.GameServerSetStatus {
	var status agonesv1.GameServerSetStatus
	for _, gs := range list {
		if gs.IsBeingDeleted() {
			// don't count GS that are being deleted
			status.ShutdownReplicas++
			continue
		}

		status.Replicas++
		switch gs.Status.State {
		case agonesv1.GameServerStateReady:
			status.ReadyReplicas++
		case agonesv1.GameServerStateAllocated:
			status.AllocatedReplicas++
		case agonesv1.GameServerStateReserved:
			status.ReservedReplicas++
		}
	}

	return status
}
