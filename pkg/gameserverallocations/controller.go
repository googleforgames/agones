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
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"

	"agones.dev/agones/pkg/apis"
	"agones.dev/agones/pkg/apis/allocation/v1alpha1"
	stablev1alpha1 "agones.dev/agones/pkg/apis/stable/v1alpha1"
	"agones.dev/agones/pkg/client/clientset/versioned"
	getterv1alpha1 "agones.dev/agones/pkg/client/clientset/versioned/typed/stable/v1alpha1"
	"agones.dev/agones/pkg/client/informers/externalversions"
	listerv1alpha1 "agones.dev/agones/pkg/client/listers/stable/v1alpha1"
	"agones.dev/agones/pkg/util/apiserver"
	"agones.dev/agones/pkg/util/https"
	"agones.dev/agones/pkg/util/logfields"
	"agones.dev/agones/pkg/util/runtime"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
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
	baseLogger       *logrus.Entry
	counter          *AllocationCounter
	gameServerSynced cache.InformerSynced
	gameServerGetter getterv1alpha1.GameServersGetter
	gameServerLister listerv1alpha1.GameServerLister
	stop             <-chan struct{}
	allocationMutex  *sync.Mutex
	recorder         record.EventRecorder
}

// findComparator is a comparator function specifically for the
// findReadyGameServerForAllocation method for determining
// scheduling strategy
type findComparator func(bestCount, currentCount NodeCount) bool

// NewController returns a controller for a GameServerAllocation
func NewController(
	apiServer *apiserver.APIServer,
	allocationMutex *sync.Mutex,
	kubeClient kubernetes.Interface,
	kubeInformerFactory informers.SharedInformerFactory,
	agonesClient versioned.Interface,
	agonesInformerFactory externalversions.SharedInformerFactory) *Controller {

	agonesInformer := agonesInformerFactory.Stable().V1alpha1()
	c := &Controller{
		counter:          NewAllocationCounter(kubeInformerFactory, agonesInformerFactory),
		gameServerSynced: agonesInformer.GameServers().Informer().HasSynced,
		gameServerGetter: agonesClient.StableV1alpha1(),
		gameServerLister: agonesInformer.GameServers().Lister(),
		allocationMutex:  allocationMutex,
	}
	c.baseLogger = runtime.NewLoggerWithType(c)

	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(c.baseLogger.Infof)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: kubeClient.CoreV1().Events("")})
	c.recorder = eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: "GameServerAllocation-controller"})

	c.registerAPIResource(apiServer)

	return c
}

// registers the api resource for gameserverallocation
func (c *Controller) registerAPIResource(api *apiserver.APIServer) {
	resource := metav1.APIResource{
		Name:         "gameserverallocations",
		SingularName: "gameserverallocation",
		Namespaced:   true,
		Kind:         "GameServerAllocation",
		Verbs: []string{
			"create",
		},
		ShortNames: []string{"gsa"},
	}
	api.AddAPIResource(v1alpha1.SchemeGroupVersion.String(), resource, c.allocationHandler)
}

// Run runs this controller. Currently, does not block
// worker queue not implemented in this controller
func (c *Controller) Run(_ int, stop <-chan struct{}) error {
	err := c.counter.Run(stop)
	if err != nil {
		return err
	}

	c.stop = stop

	c.baseLogger.Info("Wait for cache sync")
	if !cache.WaitForCacheSync(stop, c.gameServerSynced) {
		return errors.New("failed to wait for caches to sync")
	}

	return nil
}

func (c *Controller) loggerForGameServerAllocationKey(key string) *logrus.Entry {
	return logfields.AugmentLogEntry(c.baseLogger, logfields.GameServerAllocationKey, key)
}

func (c *Controller) loggerForGameServerAllocation(gsa *v1alpha1.GameServerAllocation) *logrus.Entry {
	return c.loggerForGameServerAllocationKey(gsa.Namespace+"/"+gsa.Name).WithField("gsa", gsa)
}

// allocationHandler CRDHandler for allocating a gameserver. Only accepts POST
// commands
func (c *Controller) allocationHandler(w http.ResponseWriter, r *http.Request, namespace string) error {
	if r.Body != nil {
		defer r.Body.Close() // nolint: errcheck
	}

	log := https.LogRequest(c.baseLogger, r)

	if r.Method != http.MethodPost {
		log.Warn("allocation handler only supports POST")
		http.Error(w, "Method not supported", http.StatusMethodNotAllowed)
		return nil
	}

	gsa, err := c.allocationDeserialization(r, namespace)
	if err != nil {
		return err
	}

	// server side validation
	if causes, ok := gsa.Validate(); !ok {
		status := &metav1.Status{
			Status:  metav1.StatusFailure,
			Message: fmt.Sprintf("GameServerAllocation is invalid: Invalid value: %#v", gsa),
			Reason:  metav1.StatusReasonInvalid,
			Details: &metav1.StatusDetails{
				Kind:   "GameServerAllocation",
				Group:  v1alpha1.SchemeGroupVersion.Group,
				Causes: causes,
			},
			Code: http.StatusUnprocessableEntity,
		}

		var gvks []schema.GroupVersionKind
		gvks, _, err = apiserver.Scheme.ObjectKinds(status)
		if err != nil {
			return errors.Wrap(err, "could not find objectkinds for status")
		}

		status.TypeMeta = metav1.TypeMeta{Kind: gvks[0].Kind, APIVersion: gvks[0].Version}

		w.WriteHeader(http.StatusUnprocessableEntity)
		return c.serialisation(r, w, status, apiserver.Codecs)
	}

	gs, err := c.allocate(gsa)
	if err != nil && err != ErrNoGameServerReady {
		return err
	}

	if err == ErrNoGameServerReady {
		gsa.Status.State = v1alpha1.GameServerAllocationUnAllocated
	} else {
		gsa.ObjectMeta.Name = gs.ObjectMeta.Name
		gsa.Status.State = v1alpha1.GameServerAllocationAllocated
		gsa.Status.GameServerName = gs.ObjectMeta.Name
		gsa.Status.Ports = gs.Status.Ports
		gsa.Status.Address = gs.Status.Address
		gsa.Status.NodeName = gs.Status.NodeName
	}

	c.loggerForGameServerAllocation(gsa).Info("game server allocation")

	return c.serialisation(r, w, gsa, scheme.Codecs)
}

// allocate allocated a GameServer from a given Fleet
func (c *Controller) allocate(gsa *v1alpha1.GameServerAllocation) (*stablev1alpha1.GameServer, error) {
	var allocation *stablev1alpha1.GameServer
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
	case apis.Packed:
		comparator = packedComparator
	case apis.Distributed:
		comparator = distributedComparator
	}

	allocation, err := c.findReadyGameServerForAllocation(gsa, comparator)
	if err != nil {
		return allocation, err
	}

	gsCopy := allocation.DeepCopy()
	gsCopy.Status.State = stablev1alpha1.GameServerStateAllocated

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
func (c *Controller) patchMetadata(gs *stablev1alpha1.GameServer, fam v1alpha1.MetaPatch) {
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

// findReadyGameServerForAllocation returns the most appropriate GameServer from the set, taking into account
// preferred selectors, as well as the passed in comparator
func (c *Controller) findReadyGameServerForAllocation(gsa *v1alpha1.GameServerAllocation, comparator findComparator) (*stablev1alpha1.GameServer, error) {
	// track the best node count
	var bestCount *NodeCount
	// the current GameServer from the node with the most GameServers (allocated, ready)
	var bestGS *stablev1alpha1.GameServer

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
	allocatableRequired := map[string]*stablev1alpha1.GameServer{}
	allocatablePreferred := make([]map[string]*stablev1alpha1.GameServer, len(preferred))

	// build the index of possible allocatable GameServers
	for _, gs := range gsList {
		if gs.DeletionTimestamp.IsZero() && gs.Status.State == stablev1alpha1.GameServerStateReady {
			allocatableRequired[gs.Status.NodeName] = gs

			for i, p := range preferred {
				if p.Matches(labels.Set(gs.Labels)) {
					if allocatablePreferred[i] == nil {
						allocatablePreferred[i] = map[string]*stablev1alpha1.GameServer{}
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

// allocationDeserialization processes the request and namespace, and attempts to deserialise its values
// into a GameServerAllocation. Returns an error if it fails for whatever reason.
func (c *Controller) allocationDeserialization(r *http.Request, namespace string) (*v1alpha1.GameServerAllocation, error) {
	gsa := &v1alpha1.GameServerAllocation{}

	gvks, _, err := scheme.Scheme.ObjectKinds(gsa)
	if err != nil {
		return gsa, errors.Wrap(err, "error getting objectkinds for gameserverallocation")
	}

	gsa.TypeMeta = metav1.TypeMeta{Kind: gvks[0].Kind, APIVersion: gvks[0].Version}

	mediaTypes := scheme.Codecs.SupportedMediaTypes()
	info, ok := k8sruntime.SerializerInfoForMediaType(mediaTypes, r.Header.Get("Content-Type"))
	if !ok {
		return gsa, errors.New("Could not find deserializer")
	}

	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return gsa, errors.Wrap(err, "could not read body")
	}

	gvk := v1alpha1.SchemeGroupVersion.WithKind("GameServerAllocation")
	_, _, err = info.Serializer.Decode(b, &gvk, gsa)
	if err != nil {
		c.baseLogger.WithField("body", string(b)).Error("error decoding body")
		return gsa, errors.Wrap(err, "error decoding body")
	}

	gsa.ObjectMeta.Namespace = namespace
	gsa.ObjectMeta.CreationTimestamp = metav1.Now()
	gsa.ApplyDefaults()

	return gsa, nil
}

// serialisation takes a runtime.Object, and serislises it to the ResponseWriter in the requested format
func (c *Controller) serialisation(r *http.Request, w http.ResponseWriter, obj k8sruntime.Object, codecs serializer.CodecFactory) error {
	info, err := apiserver.AcceptedSerializer(r, codecs)
	if err != nil {
		return err
	}

	w.Header().Set("Content-Type", info.MediaType)
	err = info.Serializer.Encode(obj, w)
	return errors.Wrapf(err, "error encoding %T", obj)
}
