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

package gameserverallocations

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"strconv"
	"time"

	"agones.dev/agones/pkg/apis"
	"agones.dev/agones/pkg/apis/allocation/v1alpha1"
	multiclusterv1alpha1 "agones.dev/agones/pkg/apis/multicluster/v1alpha1"
	"agones.dev/agones/pkg/apis/stable"
	stablev1alpha1 "agones.dev/agones/pkg/apis/stable/v1alpha1"
	"agones.dev/agones/pkg/client/clientset/versioned"
	getterv1alpha1 "agones.dev/agones/pkg/client/clientset/versioned/typed/stable/v1alpha1"
	"agones.dev/agones/pkg/client/informers/externalversions"
	multiclusterlisterv1alpha1 "agones.dev/agones/pkg/client/listers/multicluster/v1alpha1"
	listerv1alpha1 "agones.dev/agones/pkg/client/listers/stable/v1alpha1"
	"agones.dev/agones/pkg/gameservers"
	"agones.dev/agones/pkg/util/apiserver"
	"agones.dev/agones/pkg/util/https"
	"agones.dev/agones/pkg/util/logfields"
	"agones.dev/agones/pkg/util/runtime"
	"agones.dev/agones/pkg/util/workerqueue"
	"github.com/heptiolabs/healthcheck"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	corev1lister "k8s.io/client-go/listers/core/v1"
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

const (
	secretClientCertName = "client.crt"
	secretClientKeyName  = "client.key"
	secretCaCertName     = "ca.crt"
)

// Controller is a the GameServerAllocation controller
type Controller struct {
	baseLogger       *logrus.Entry
	counter          *gameservers.PerNodeCounter
	readyGameServers gameServerCacheEntry
	// Instead of selecting the top one, controller selects a random one
	// from the topNGameServerCount of Ready gameservers
	topNGameServerCount    int
	gameServerSynced       cache.InformerSynced
	gameServerGetter       getterv1alpha1.GameServersGetter
	gameServerLister       listerv1alpha1.GameServerLister
	allocationPolicyLister multiclusterlisterv1alpha1.GameServerAllocationPolicyLister
	allocationPolicySynced cache.InformerSynced
	secretLister           corev1lister.SecretLister
	secretSynced           cache.InformerSynced
	stop                   <-chan struct{}
	workerqueue            *workerqueue.WorkerQueue
	recorder               record.EventRecorder
}

// findComparator is a comparator function specifically for the
// findReadyGameServerForAllocation method for determining
// scheduling strategy
type findComparator func(bestCount, currentCount gameservers.NodeCount) bool

var allocationRetry = wait.Backoff{
	Steps:    5,
	Duration: 10 * time.Millisecond,
	Factor:   1.0,
	Jitter:   0.1,
}

// NewController returns a controller for a GameServerAllocation
func NewController(apiServer *apiserver.APIServer,
	health healthcheck.Handler,
	counter *gameservers.PerNodeCounter,
	topNGameServerCnt int,
	kubeClient kubernetes.Interface,
	kubeInformerFactory informers.SharedInformerFactory,
	agonesClient versioned.Interface,
	agonesInformerFactory externalversions.SharedInformerFactory,
) *Controller {

	agonesInformer := agonesInformerFactory.Stable().V1alpha1()
	c := &Controller{
		counter:                counter,
		topNGameServerCount:    topNGameServerCnt,
		gameServerSynced:       agonesInformer.GameServers().Informer().HasSynced,
		gameServerGetter:       agonesClient.StableV1alpha1(),
		gameServerLister:       agonesInformer.GameServers().Lister(),
		allocationPolicyLister: agonesInformerFactory.Multicluster().V1alpha1().GameServerAllocationPolicies().Lister(),
		allocationPolicySynced: agonesInformerFactory.Multicluster().V1alpha1().GameServerAllocationPolicies().Informer().HasSynced,
		secretLister:           kubeInformerFactory.Core().V1().Secrets().Lister(),
		secretSynced:           kubeInformerFactory.Core().V1().Secrets().Informer().HasSynced,
	}
	c.baseLogger = runtime.NewLoggerWithType(c)
	c.workerqueue = workerqueue.NewWorkerQueue(c.syncGameServers, c.baseLogger, logfields.GameServerKey, stable.GroupName+".GameServerUpdateController")
	health.AddLivenessCheck("gameserverallocation-gameserver-workerqueue", healthcheck.Check(c.workerqueue.Healthy))

	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(c.baseLogger.Infof)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: kubeClient.CoreV1().Events("")})
	c.recorder = eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: "GameServerAllocation-controller"})

	c.registerAPIResource(apiServer)

	agonesInformer.GameServers().Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		UpdateFunc: func(oldObj, newObj interface{}) {
			// only interested in if the old / new state was/is Ready
			oldGs := oldObj.(*stablev1alpha1.GameServer)
			newGs := newObj.(*stablev1alpha1.GameServer)
			if oldGs.Status.State == stablev1alpha1.GameServerStateReady || newGs.Status.State == stablev1alpha1.GameServerStateReady {
				if key, ok := c.getKey(newGs); ok {
					if newGs.Status.State == stablev1alpha1.GameServerStateReady {
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

// Run runs this controller. Will block until stop is closed.
// Ignores threadiness, as we only needs 1 worker for cache sync
func (c *Controller) Run(_ int, stop <-chan struct{}) error {
	c.stop = stop
	c.baseLogger.Info("Wait for cache sync")
	if !cache.WaitForCacheSync(stop, c.gameServerSynced, c.secretSynced, c.allocationPolicySynced) {
		return errors.New("failed to wait for caches to sync")
	}

	// build the cache
	err := c.syncReadyGSServerCache()
	if err != nil {
		return err
	}

	// we don't want mutiple workers refresh cache at the same time so one worker will be better.
	// Also we don't expect to have too many failures when allocating
	c.workerqueue.Run(1, stop)

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

	// If multi-cluster setting is enabled, allocate base on the multicluster allocation policy.
	var out *v1alpha1.GameServerAllocation
	if gsa.Spec.MultiClusterSetting.Enabled {
		out, err = c.applyMultiClusterAllocation(gsa)
	} else {
		out, err = c.allocateFromLocalCluster(gsa)
	}

	if err != nil {
		return err
	}

	return c.serialisation(r, w, out, scheme.Codecs)
}

// allocateFromLocalCluster allocates gameservers from the local cluster.
func (c *Controller) allocateFromLocalCluster(gsa *v1alpha1.GameServerAllocation) (*v1alpha1.GameServerAllocation, error) {
	var gs *stablev1alpha1.GameServer
	err := Retry(allocationRetry, func() error {
		var err error
		gs, err = c.allocate(gsa)
		return err
	})

	if err != nil && err != ErrNoGameServerReady && err != ErrConflictInGameServerSelection {
		// this will trigger syncing of the cache (assuming cache might not be up to date)
		c.workerqueue.EnqueueImmediately(gs)
		return nil, err
	}

	if err == ErrNoGameServerReady {
		gsa.Status.State = v1alpha1.GameServerAllocationUnAllocated
	} else if err == ErrConflictInGameServerSelection {
		gsa.Status.State = v1alpha1.GameServerAllocationContention
	} else {
		gsa.ObjectMeta.Name = gs.ObjectMeta.Name
		gsa.Status.State = v1alpha1.GameServerAllocationAllocated
		gsa.Status.GameServerName = gs.ObjectMeta.Name
		gsa.Status.Ports = gs.Status.Ports
		gsa.Status.Address = gs.Status.Address
		gsa.Status.NodeName = gs.Status.NodeName
	}

	c.loggerForGameServerAllocation(gsa).Info("game server allocation")
	return gsa, nil
}

// applyMultiClusterAllocation retrieves allocation policies and iterate on policies.
// Then allocate gameservers from local or remote cluster accordingly.
func (c *Controller) applyMultiClusterAllocation(gsa *v1alpha1.GameServerAllocation) (result *v1alpha1.GameServerAllocation, err error) {

	selector := labels.Everything()
	if len(gsa.Spec.MultiClusterSetting.PolicySelector.MatchLabels)+len(gsa.Spec.MultiClusterSetting.PolicySelector.MatchExpressions) != 0 {
		selector, err = metav1.LabelSelectorAsSelector(&gsa.Spec.MultiClusterSetting.PolicySelector)
		if err != nil {
			return nil, err
		}
	}

	policies, err := c.allocationPolicyLister.GameServerAllocationPolicies(gsa.ObjectMeta.Namespace).List(selector)
	if err != nil {
		return nil, err
	} else if len(policies) == 0 {
		return nil, errors.New("no multi-cluster allocation policy is specified")
	}

	it := multiclusterv1alpha1.NewConnectionInfoIterator(policies)
	for {
		connectionInfo := it.Next()
		if connectionInfo == nil {
			break
		}
		if connectionInfo.ClusterName == gsa.ObjectMeta.ClusterName {
			result, err = c.allocateFromLocalCluster(gsa)
			c.baseLogger.Error(err)
		} else {
			result, err = c.allocateFromRemoteCluster(*gsa, connectionInfo, gsa.ObjectMeta.Namespace)
			c.baseLogger.Error(err)
		}
		if result != nil {
			return result, nil
		}
	}
	return nil, err
}

// allocateFromRemoteCluster allocates gameservers from a remote cluster by making
// an http call to allocation service in that cluster.
func (c *Controller) allocateFromRemoteCluster(gsa v1alpha1.GameServerAllocation, connectionInfo *multiclusterv1alpha1.ClusterConnectionInfo, namespace string) (*v1alpha1.GameServerAllocation, error) {
	var gsaResult v1alpha1.GameServerAllocation

	// TODO: handle converting error to apiserver error
	// TODO: cache the client
	client, err := c.createRemoteClusterRestClient(namespace, connectionInfo.SecretName)
	if err != nil {
		return nil, err
	}

	// Forward the game server allocation request to another cluster,
	// and disable multicluster settings to avoid the target cluster
	// forward the allocation request again.
	gsa.Spec.MultiClusterSetting.Enabled = false
	body, err := json.Marshal(gsa)
	if err != nil {
		return nil, err
	}

	// TODO: Retry on transient error --> response.StatusCode >= 500
	response, err := client.Post(connectionInfo.AllocationEndpoint, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	defer response.Body.Close() // nolint: errcheck

	data, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	if response.StatusCode >= 400 {
		// For error responses return the body without deserializing to an object.
		return nil, errors.New(string(data))
	}

	err = json.Unmarshal(data, &gsaResult)
	if err != nil {
		return nil, err
	}

	return &gsaResult, nil
}

// createRemoteClusterRestClient creates a rest client with proper certs to make a remote call.
func (c *Controller) createRemoteClusterRestClient(namespace, secretName string) (*http.Client, error) {
	clientCert, clientKey, caCert, err := c.getClientCertificates(namespace, secretName)
	if err != nil {
		return nil, err
	}
	if clientCert == nil || clientKey == nil {
		return nil, fmt.Errorf("missing client certificate key pair in secret %s", secretName)
	}

	// Load client cert
	cert, err := tls.X509KeyPair(clientCert, clientKey)
	if err != nil {
		return nil, err
	}

	tlsConfig := &tls.Config{Certificates: []tls.Certificate{cert}}
	if len(caCert) != 0 {
		// Load CA cert, if provided and trust the server certificate.
		// This is required for self-signed certs.
		tlsConfig.RootCAs = x509.NewCertPool()
		ca, err := x509.ParseCertificate(caCert)
		if err != nil {
			return nil, err
		}
		tlsConfig.RootCAs.AddCert(ca)
	}

	// Setup HTTPS client
	return &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
	}, nil
}

// getClientCertificates returns the client certificates and CA cert for remote allocation cluster call
func (c *Controller) getClientCertificates(namespace, secretName string) (clientCert, clientKey, caCert []byte, err error) {
	secret, err := c.secretLister.Secrets(namespace).Get(secretName)
	if err != nil {
		return nil, nil, nil, err
	}
	if secret == nil || len(secret.Data) == 0 {
		return nil, nil, nil, fmt.Errorf("secert %s does not have data", secretName)
	}

	// Create http client using cert
	clientCert = secret.Data[secretClientCertName]
	clientKey = secret.Data[secretClientKeyName]
	caCert = secret.Data[secretCaCertName]
	return clientCert, clientKey, caCert, nil
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

// allocate allocated a GameServer from a given Fleet
func (c *Controller) allocate(gsa *v1alpha1.GameServerAllocation) (*stablev1alpha1.GameServer, error) {
	var allocation *stablev1alpha1.GameServer
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

	key, _ := cache.MetaNamespaceKeyFunc(allocation)
	if ok := c.readyGameServers.Delete(key); !ok {
		return allocation, ErrConflictInGameServerSelection
	}

	gsCopy := allocation.DeepCopy()
	gsCopy.Status.State = stablev1alpha1.GameServerStateAllocated

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
	var bestCount *gameservers.NodeCount
	// the current GameServer from the node with the most GameServers (allocated, ready)
	var bestGS *stablev1alpha1.GameServer

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

	var bestGSList []stablev1alpha1.GameServer
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
	currGameservers := make(map[string]*stablev1alpha1.GameServer)
	for _, gs := range gsList {
		if key, ok := c.getKey(gs); ok {
			currGameservers[key] = gs
		}
	}

	// first remove the gameservers are not in the list anymore
	tobeDeletedGSInCache := make([]string, 0)
	c.readyGameServers.Range(func(key string, gs *stablev1alpha1.GameServer) bool {
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
			if !(gs.DeletionTimestamp.IsZero() && gs.Status.State == stablev1alpha1.GameServerStateReady) {
				c.readyGameServers.Delete(key)
			} else if gs.ObjectMeta.ResourceVersion != gsCache.ObjectMeta.ResourceVersion {
				c.readyGameServers.Store(key, gs)
			}
		} else if gs.DeletionTimestamp.IsZero() && gs.Status.State == stablev1alpha1.GameServerStateReady {
			c.readyGameServers.Store(key, gs)
		}
	}

	return nil
}

// selectGameServers selects the appropriate gameservers from cache based on selector.
func (c *Controller) selectGameServers(selector labels.Selector) (res []*stablev1alpha1.GameServer) {
	c.readyGameServers.Range(func(key string, gs *stablev1alpha1.GameServer) bool {
		if selector.Matches(labels.Set(gs.ObjectMeta.GetLabels())) {
			res = append(res, gs)
		}
		return true
	})
	return res
}

// getKey extract the key of gameserver object
func (c *Controller) getKey(gs *stablev1alpha1.GameServer) (string, bool) {
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
func (c *Controller) getRandomlySelectedGS(gsa *v1alpha1.GameServerAllocation, bestGSList []stablev1alpha1.GameServer) *stablev1alpha1.GameServer {
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
