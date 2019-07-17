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
	"sort"
	"strconv"
	"time"

	"agones.dev/agones/pkg/apis/agones"
	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	allocationv1 "agones.dev/agones/pkg/apis/allocation/v1"
	multiclusterv1alpha1 "agones.dev/agones/pkg/apis/multicluster/v1alpha1"
	"agones.dev/agones/pkg/client/clientset/versioned"
	getterv1 "agones.dev/agones/pkg/client/clientset/versioned/typed/agones/v1"
	"agones.dev/agones/pkg/client/informers/externalversions"
	listerv1 "agones.dev/agones/pkg/client/listers/agones/v1"
	multiclusterlisterv1alpha1 "agones.dev/agones/pkg/client/listers/multicluster/v1alpha1"
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
	secretClientCertName  = "tls.crt"
	secretClientKeyName   = "tls.key"
	secretCaCertName      = "ca.crt"
	maxBatchQueue         = 100
	maxBatchBeforeRefresh = 100
	batchWaitTime         = 500 * time.Millisecond
)

// request is an async request for allocation
type request struct {
	gsa      *allocationv1.GameServerAllocation
	response chan response
}

// response is an async response for a matching request
type response struct {
	request request
	gs      *agonesv1.GameServer
	err     error
}

// Controller is a the GameServerAllocation controller
type Controller struct {
	baseLogger       *logrus.Entry
	counter          *gameservers.PerNodeCounter
	readyGameServers gameServerCacheEntry
	// Instead of selecting the top one, controller selects a random one
	// from the topNGameServerCount of Ready gameservers
	topNGameServerCount    int
	gameServerSynced       cache.InformerSynced
	gameServerGetter       getterv1.GameServersGetter
	gameServerLister       listerv1.GameServerLister
	allocationPolicyLister multiclusterlisterv1alpha1.GameServerAllocationPolicyLister
	allocationPolicySynced cache.InformerSynced
	secretLister           corev1lister.SecretLister
	secretSynced           cache.InformerSynced
	stop                   <-chan struct{}
	workerqueue            *workerqueue.WorkerQueue
	recorder               record.EventRecorder
	pendingRequests        chan request
}

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

	agonesInformer := agonesInformerFactory.Agones().V1()
	c := &Controller{
		counter:                counter,
		topNGameServerCount:    topNGameServerCnt,
		gameServerSynced:       agonesInformer.GameServers().Informer().HasSynced,
		gameServerGetter:       agonesClient.AgonesV1(),
		gameServerLister:       agonesInformer.GameServers().Lister(),
		allocationPolicyLister: agonesInformerFactory.Multicluster().V1alpha1().GameServerAllocationPolicies().Lister(),
		allocationPolicySynced: agonesInformerFactory.Multicluster().V1alpha1().GameServerAllocationPolicies().Informer().HasSynced,
		secretLister:           kubeInformerFactory.Core().V1().Secrets().Lister(),
		secretSynced:           kubeInformerFactory.Core().V1().Secrets().Informer().HasSynced,
		pendingRequests:        make(chan request, maxBatchQueue),
	}
	c.baseLogger = runtime.NewLoggerWithType(c)
	c.workerqueue = workerqueue.NewWorkerQueue(c.syncGameServers, c.baseLogger, logfields.GameServerKey, agones.GroupName+".GameServerUpdateController")
	health.AddLivenessCheck("gameserverallocation-gameserver-workerqueue", healthcheck.Check(c.workerqueue.Healthy))

	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(c.baseLogger.Infof)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: kubeClient.CoreV1().Events("")})
	c.recorder = eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: "GameServerAllocation-controller"})

	c.registerAPIResource(apiServer)

	agonesInformer.GameServers().Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		UpdateFunc: func(oldObj, newObj interface{}) {
			// only interested in if the old / new state was/is Ready
			oldGs := oldObj.(*agonesv1.GameServer)
			newGs := newObj.(*agonesv1.GameServer)
			key, ok := c.getKey(newGs)
			if !ok {
				return
			}
			if newGs.IsBeingDeleted() {
				c.readyGameServers.Delete(key)
			} else if oldGs.Status.State == agonesv1.GameServerStateReady || newGs.Status.State == agonesv1.GameServerStateReady {
				if newGs.Status.State == agonesv1.GameServerStateReady {
					c.readyGameServers.Store(key, newGs)
				} else {
					c.readyGameServers.Delete(key)
				}
			}
		},
		DeleteFunc: func(obj interface{}) {
			gs, ok := obj.(*agonesv1.GameServer)
			if !ok {
				return
			}
			var key string
			if key, ok = c.getKey(gs); ok {
				c.readyGameServers.Delete(key)
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
	api.AddAPIResource(allocationv1.SchemeGroupVersion.String(), resource, c.allocationHandler)
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

	// workers and logic for batching allocations
	go c.runLocalAllocations(maxBatchQueue)

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

func (c *Controller) loggerForGameServerAllocation(gsa *allocationv1.GameServerAllocation) *logrus.Entry {
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
				Group:  allocationv1.SchemeGroupVersion.Group,
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
	var out *allocationv1.GameServerAllocation
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
func (c *Controller) allocateFromLocalCluster(gsa *allocationv1.GameServerAllocation) (*allocationv1.GameServerAllocation, error) {
	var gs *agonesv1.GameServer
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
		gsa.Status.State = allocationv1.GameServerAllocationUnAllocated
	} else if err == ErrConflictInGameServerSelection {
		gsa.Status.State = allocationv1.GameServerAllocationContention
	} else {
		gsa.ObjectMeta.Name = gs.ObjectMeta.Name
		gsa.Status.State = allocationv1.GameServerAllocationAllocated
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
func (c *Controller) applyMultiClusterAllocation(gsa *allocationv1.GameServerAllocation) (result *allocationv1.GameServerAllocation, err error) {

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
func (c *Controller) allocateFromRemoteCluster(gsa allocationv1.GameServerAllocation, connectionInfo *multiclusterv1alpha1.ClusterConnectionInfo, namespace string) (*allocationv1.GameServerAllocation, error) {
	var gsaResult allocationv1.GameServerAllocation

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
	for i, endpoint := range connectionInfo.AllocationEndpoints {
		response, err := client.Post(endpoint, "application/json", bytes.NewBuffer(body))
		if err != nil {
			return nil, err
		}
		defer response.Body.Close() // nolint: errcheck

		data, err := ioutil.ReadAll(response.Body)
		if err != nil {
			return nil, err
		}
		if response.StatusCode >= 500 && (i+1) < len(connectionInfo.AllocationEndpoints) {
			// If there is a server error try a different endpoint
			c.baseLogger.WithError(err).WithField("endpoint", endpoint).Warn("The request sent failed, trying next endpoint")
			continue
		}
		if response.StatusCode >= 400 {
			// For error responses return the body without deserializing to an object.
			return nil, errors.New(string(data))
		}

		err = json.Unmarshal(data, &gsaResult)
		if err != nil {
			return nil, err
		}
		break
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
func (c *Controller) allocationDeserialization(r *http.Request, namespace string) (*allocationv1.GameServerAllocation, error) {
	gsa := &allocationv1.GameServerAllocation{}

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

	gvk := allocationv1.SchemeGroupVersion.WithKind("GameServerAllocation")
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

// allocate allocated a GameServer from a given GameServerAllocation
// this sets up allocation through a batch process.
func (c *Controller) allocate(gsa *allocationv1.GameServerAllocation) (*agonesv1.GameServer, error) {
	// creates an allocation request. This contains the requested GameServerAllocation, as well as the
	// channel we expect the return values to come back for this GameServerAllocation
	req := request{gsa: gsa, response: make(chan response)}

	// this pushes the request into the batching process
	c.pendingRequests <- req

	select {
	case res := <-req.response: // wait for the batch to be completed
		return res.gs, res.err
	case <-c.stop:
		return nil, errors.New("shutting down")
	}
}

// runLocalAllocations is a blocking function that runs in a loop
// looking at c.requestBatches for batches of requests that are coming through.
func (c *Controller) runLocalAllocations(updateWorkerCount int) {
	// setup workers for allocation updates. Push response values into
	// this queue for concurrent updating of GameServers to Allocated
	updateQueue := c.allocationUpdateWorkers(updateWorkerCount)

	// Batch processing strategy:
	// We constantly loop around the below for loop. If nothing is found in c.pendingRequests, we move to
	// default: which will wait for half a second, to allow for some requests to backup in c.pendingRequests,
	// providing us with a batch of Allocation requests in that channel

	// Once we have 1 or more requests in c.pendingRequests (which is buffered to 100), we can start the batch process.

	// Assuming this is the first run (either entirely, or for a while), list will be nil, and therefore the first
	// thing that will be done is retrieving the Ready GameSerers and sorting them for this batch via
	// c.listSortedReadyGameServers(). This list is maintained as we flow through the batch.

	// We then use findGameServerForAllocation to loop around the sorted list of Ready GameServers to look for matches
	// against the preferred and required selectors of the GameServerAllocation. If there is an error, we immediately
	// pass that straight back to the response channel for this GameServerAllocation.

	// Assuming we find a matching GameServer to our GameServerAllocation, we remove it from the list and the backing
	// Ready GameServer cache.

	// We then pass the found GameServers into the updateQueue, where there are updateWorkerCount number of goroutines
	// waiting to concurrently attempt to move the GameServer into an Allocated state, and return the result to
	// GameServerAllocation request's response channel

	// Then we get the next item off the batch (c.pendingRequests), and do this all over again, but this time, we have
	// an already sorted list of GameServers, so we only need to find one that matches our GameServerAllocation
	// selectors, and put it into updateQueue

	// The tracking of requestCount >= maxBatchBeforeRefresh is necessary, because without it, at high enough load
	// the list of GameServers that we are using to allocate would never get refreshed (list = nil) with an updated
	// list of Ready GameServers, and you would eventually never be able to Allocate anything as long as the load
	// continued.

	var list []*agonesv1.GameServer
	requestCount := 0

	for {
		select {
		case req := <-c.pendingRequests:
			// refresh the list after every 100 allocations made in a single batch
			requestCount++
			if requestCount >= maxBatchBeforeRefresh {
				list = nil
				requestCount = 0
			}

			if list == nil {
				list = c.listSortedReadyGameServers()
			}

			gs, index, err := findGameServerForAllocation(req.gsa, list)
			if err != nil {
				req.response <- response{request: req, gs: nil, err: err}
				continue
			}
			// remove the game server that has been allocated
			list = append(list[:index], list[index+1:]...)

			key, _ := cache.MetaNamespaceKeyFunc(gs)
			if ok := c.readyGameServers.Delete(key); !ok {
				// this seems unlikely, but lets handle it just in case
				req.response <- response{request: req, gs: nil, err: ErrConflictInGameServerSelection}
				continue
			}

			updateQueue <- response{request: req, gs: gs.DeepCopy(), err: nil}

		case <-c.stop:
			return
		default:
			list = nil
			requestCount = 0
			// slow down cpu churn, and allow items to batch
			time.Sleep(batchWaitTime)
		}
	}
}

// allocationUpdateWorkers runs workerCount number of goroutines as workers to
// process each GameServer passed into the returned updateQueue
// Each worker will concurrently attempt to move the GameServer to an Allocated
// state and then respond to the initial request's response channel with the
// details of that update
func (c *Controller) allocationUpdateWorkers(workerCount int) chan<- response {
	updateQueue := make(chan response)

	for i := 0; i < workerCount; i++ {
		go func() {
			for {
				select {
				case res := <-updateQueue:
					gsCopy := res.gs.DeepCopy()
					c.patchMetadata(gsCopy, res.request.gsa.Spec.MetaPatch)
					gsCopy.Status.State = agonesv1.GameServerStateAllocated

					gs, err := c.gameServerGetter.GameServers(res.gs.ObjectMeta.Namespace).Update(gsCopy)
					if err != nil {
						key, _ := cache.MetaNamespaceKeyFunc(gs)
						// since we could not allocate, we should put it back
						c.readyGameServers.Store(key, gs)
						res.err = errors.Wrap(err, "error updating allocated gameserver")
					} else {
						res.gs = gs
						c.recorder.Event(res.gs, corev1.EventTypeNormal, string(res.gs.Status.State), "Allocated")
					}

					res.request.response <- res
				case <-c.stop:
					return
				}
			}
		}()
	}

	return updateQueue
}

// listSortedReadyGameServers returns a list of the cache ready gameservers
// sorted by most allocated to least
func (c *Controller) listSortedReadyGameServers() []*agonesv1.GameServer {
	length := c.readyGameServers.Len()
	if length == 0 {
		return []*agonesv1.GameServer{}
	}

	list := make([]*agonesv1.GameServer, 0, length)
	c.readyGameServers.Range(func(_ string, gs *agonesv1.GameServer) bool {
		list = append(list, gs)
		return true
	})
	counts := c.counter.Counts()

	sort.Slice(list, func(i, j int) bool {
		gs1 := list[i]
		gs2 := list[j]

		c1, ok := counts[gs1.Status.NodeName]
		if !ok {
			return false
		}

		c2, ok := counts[gs2.Status.NodeName]
		if !ok {
			return true
		}

		if c1.Allocated > c2.Allocated {
			return true
		}
		if c1.Allocated < c2.Allocated {
			return false
		}

		// prefer nodes that have the most Ready gameservers on them - they are most likely to be
		// completely filled and least likely target for scale down.
		if c1.Ready < c2.Ready {
			return false
		}
		if c1.Ready > c2.Ready {
			return true
		}

		// finally sort lexicographically, so we have a stable order
		return gs1.Status.NodeName < gs2.Status.NodeName
	})

	return list
}

// patch the labels and annotations of an allocated GameServer with metadata from a GameServerAllocation
func (c *Controller) patchMetadata(gs *agonesv1.GameServer, fam allocationv1.MetaPatch) {
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
	currGameservers := make(map[string]*agonesv1.GameServer)
	for _, gs := range gsList {
		if key, ok := c.getKey(gs); ok {
			currGameservers[key] = gs
		}
	}

	// first remove the gameservers are not in the list anymore
	tobeDeletedGSInCache := make([]string, 0)
	c.readyGameServers.Range(func(key string, gs *agonesv1.GameServer) bool {
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
			if !(gs.DeletionTimestamp.IsZero() && gs.Status.State == agonesv1.GameServerStateReady) {
				c.readyGameServers.Delete(key)
			} else if gs.ObjectMeta.ResourceVersion != gsCache.ObjectMeta.ResourceVersion {
				c.readyGameServers.Store(key, gs)
			}
		} else if gs.DeletionTimestamp.IsZero() && gs.Status.State == agonesv1.GameServerStateReady {
			c.readyGameServers.Store(key, gs)
		}
	}

	return nil
}

// getKey extract the key of gameserver object
func (c *Controller) getKey(gs *agonesv1.GameServer) (string, bool) {
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
func (c *Controller) getRandomlySelectedGS(gsa *allocationv1.GameServerAllocation, bestGSList []agonesv1.GameServer) *agonesv1.GameServer {
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
