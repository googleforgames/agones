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

package gameservers

import (
	"encoding/json"
	"fmt"
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
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	corelisterv1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
)

var (
	errPodNotFound = errors.New("A Pod for this GameServer Was Not Found")
)

// Controller is a the main GameServer crd controller
type Controller struct {
	logger                 *logrus.Entry
	sidecarImage           string
	alwaysPullSidecarImage bool
	crdGetter              v1beta1.CustomResourceDefinitionInterface
	podGetter              typedcorev1.PodsGetter
	podLister              corelisterv1.PodLister
	podSynced              cache.InformerSynced
	gameServerGetter       getterv1alpha1.GameServersGetter
	gameServerLister       listerv1alpha1.GameServerLister
	gameServerSynced       cache.InformerSynced
	nodeLister             corelisterv1.NodeLister
	portAllocator          *PortAllocator
	healthController       *HealthController
	workerqueue            *workerqueue.WorkerQueue
	allocationMutex        *sync.Mutex
	stop                   <-chan struct{}
	recorder               record.EventRecorder
}

// NewController returns a new gameserver crd controller
func NewController(
	wh *webhooks.WebHook,
	health healthcheck.Handler,
	allocationMutex *sync.Mutex,
	minPort, maxPort int32,
	sidecarImage string,
	alwaysPullSidecarImage bool,
	kubeClient kubernetes.Interface,
	kubeInformerFactory informers.SharedInformerFactory,
	extClient extclientset.Interface,
	agonesClient versioned.Interface,
	agonesInformerFactory externalversions.SharedInformerFactory) *Controller {

	pods := kubeInformerFactory.Core().V1().Pods()
	gameServers := agonesInformerFactory.Stable().V1alpha1().GameServers()
	gsInformer := gameServers.Informer()

	c := &Controller{
		sidecarImage:           sidecarImage,
		alwaysPullSidecarImage: alwaysPullSidecarImage,
		allocationMutex:        allocationMutex,
		crdGetter:              extClient.ApiextensionsV1beta1().CustomResourceDefinitions(),
		podGetter:              kubeClient.CoreV1(),
		podLister:              pods.Lister(),
		podSynced:              pods.Informer().HasSynced,
		gameServerGetter:       agonesClient.StableV1alpha1(),
		gameServerLister:       gameServers.Lister(),
		gameServerSynced:       gsInformer.HasSynced,
		nodeLister:             kubeInformerFactory.Core().V1().Nodes().Lister(),
		portAllocator:          NewPortAllocator(minPort, maxPort, kubeInformerFactory, agonesInformerFactory),
		healthController:       NewHealthController(kubeClient, agonesClient, kubeInformerFactory, agonesInformerFactory),
	}

	c.logger = runtime.NewLoggerWithType(c)

	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(c.logger.Infof)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: kubeClient.CoreV1().Events("")})
	c.recorder = eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: "gameserver-controller"})

	c.workerqueue = workerqueue.NewWorkerQueue(c.syncGameServer, c.logger, stable.GroupName+".GameServerController")
	health.AddLivenessCheck("gameserver-workerqueue", healthcheck.Check(c.workerqueue.Healthy))

	wh.AddHandler("/mutate", v1alpha1.Kind("GameServer"), admv1beta1.Create, c.creationMutationHandler)
	wh.AddHandler("/validate", v1alpha1.Kind("GameServer"), admv1beta1.Create, c.creationValidationHandler)

	gsInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: c.workerqueue.Enqueue,
		UpdateFunc: func(oldObj, newObj interface{}) {
			// no point in processing unless there is a State change
			oldGs := oldObj.(*v1alpha1.GameServer)
			newGs := newObj.(*v1alpha1.GameServer)
			if oldGs.Status.State != newGs.Status.State || oldGs.ObjectMeta.DeletionTimestamp != newGs.ObjectMeta.DeletionTimestamp {
				c.workerqueue.Enqueue(newGs)
			}
		},
	})

	// track pod deletions, for when GameServers are deleted
	pods.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		UpdateFunc: func(oldObj, newObj interface{}) {
			oldPod := oldObj.(*corev1.Pod)
			if isGameServerPod(oldPod) {
				newPod := newObj.(*corev1.Pod)
				//  node name has changed -- i.e. it has been scheduled
				if oldPod.Spec.NodeName != newPod.Spec.NodeName {
					owner := metav1.GetControllerOf(newPod)
					c.workerqueue.Enqueue(cache.ExplicitKey(newPod.ObjectMeta.Namespace + "/" + owner.Name))
				}
			}
		},
		DeleteFunc: func(obj interface{}) {
			// Could be a DeletedFinalStateUnknown, in which case, just ignore it
			pod, ok := obj.(*corev1.Pod)
			if ok && isGameServerPod(pod) {
				owner := metav1.GetControllerOf(pod)
				c.workerqueue.Enqueue(cache.ExplicitKey(pod.ObjectMeta.Namespace + "/" + owner.Name))
			}
		},
	})

	return c
}

// creationMutationHandler is the handler for the mutating webhook that sets the
// the default values on the GameServer
// Should only be called on gameserver create operations.
// nolint:dupl
func (c *Controller) creationMutationHandler(review admv1beta1.AdmissionReview) (admv1beta1.AdmissionReview, error) {
	c.logger.WithField("review", review).Info("creationMutationHandler")

	obj := review.Request.Object
	gs := &v1alpha1.GameServer{}
	err := json.Unmarshal(obj.Raw, gs)
	if err != nil {
		return review, errors.Wrapf(err, "error unmarshalling original GameServer json: %s", obj.Raw)
	}

	// This is the main logic of this function
	// the rest is really just json plumbing
	gs.ApplyDefaults()

	newGS, err := json.Marshal(gs)
	if err != nil {
		return review, errors.Wrapf(err, "error marshalling default applied GameSever %s to json", gs.ObjectMeta.Name)
	}

	patch, err := jsonpatch.CreatePatch(obj.Raw, newGS)
	if err != nil {
		return review, errors.Wrapf(err, "error creating patch for GameServer %s", gs.ObjectMeta.Name)
	}

	json, err := json.Marshal(patch)
	if err != nil {
		return review, errors.Wrapf(err, "error creating json for patch for GameServer %s", gs.ObjectMeta.Name)
	}

	c.logger.WithField("gs", gs.ObjectMeta.Name).WithField("patch", string(json)).Infof("patch created!")

	pt := admv1beta1.PatchTypeJSONPatch
	review.Response.PatchType = &pt
	review.Response.Patch = json

	return review, nil
}

// creationValidationHandler that validates a GameServer when it is created
// Should only be called on gameserver create operations.
func (c *Controller) creationValidationHandler(review admv1beta1.AdmissionReview) (admv1beta1.AdmissionReview, error) {
	c.logger.WithField("review", review).Info("creationValidationHandler")

	obj := review.Request.Object
	gs := &v1alpha1.GameServer{}
	err := json.Unmarshal(obj.Raw, gs)
	if err != nil {
		return review, errors.Wrapf(err, "error unmarshalling original GameServer json: %s", obj.Raw)
	}

	ok, causes := gs.Validate()
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
			Message: "GameServer configuration is invalid",
			Reason:  metav1.StatusReasonInvalid,
			Details: &details,
		}

		c.logger.WithField("review", review).Info("Invalid GameServer")
		return review, nil
	}

	return review, nil
}

// Run the GameServer controller. Will block until stop is closed.
// Runs threadiness number workers to process the rate limited queue
func (c *Controller) Run(workers int, stop <-chan struct{}) error {
	c.stop = stop

	err := crd.WaitForEstablishedCRD(c.crdGetter, "gameservers.stable.agones.dev", c.logger)
	if err != nil {
		return err
	}

	c.logger.Info("Wait for cache sync")
	if !cache.WaitForCacheSync(stop, c.gameServerSynced) {
		return errors.New("failed to wait for caches to sync")
	}

	// Run the Port Allocator
	go func() {
		if err := c.portAllocator.Run(stop); err != nil {
			c.logger.WithError(err).Error("error running the port allocator")
		}
	}()

	// Run the Health Controller
	go c.healthController.Run(stop)

	c.workerqueue.Run(workers, stop)
	return nil
}

// syncGameServer synchronises the Pods for the GameServers.
// and reacts to status changes that can occur through the client SDK
func (c *Controller) syncGameServer(key string) error {
	c.logger.WithField("key", key).Info("Synchronising")

	// Convert the namespace/name string into a distinct namespace and name
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		// don't return an error, as we don't want this retried
		runtime.HandleError(c.logger.WithField("key", key), errors.Wrapf(err, "invalid resource key"))
		return nil
	}

	gs, err := c.gameServerLister.GameServers(namespace).Get(name)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			c.logger.WithField("key", key).Info("GameServer is no longer available for syncing")
			return nil
		}
		return errors.Wrapf(err, "error retrieving GameServer %s from namespace %s", name, namespace)
	}

	if gs, err = c.syncGameServerDeletionTimestamp(gs); err != nil {
		return err
	}
	if gs, err = c.syncGameServerPortAllocationState(gs); err != nil {
		return err
	}
	if gs, err = c.syncGameServerCreatingState(gs); err != nil {
		return err
	}
	if gs, err = c.syncGameServerStartingState(gs); err != nil {
		return err
	}
	if gs, err = c.syncGameServerRequestReadyState(gs); err != nil {
		return err
	}
	if err = c.syncGameServerShutdownState(gs); err != nil {
		return err
	}

	return nil
}

// syncGameServerDeletionTimestamp if the deletion timestamp is non-zero
// then do one of two things:
// - if the GameServer has Pods running, delete them
// - if there no pods, remove the finalizer
func (c *Controller) syncGameServerDeletionTimestamp(gs *v1alpha1.GameServer) (*v1alpha1.GameServer, error) {
	if gs.ObjectMeta.DeletionTimestamp.IsZero() {
		return gs, nil
	}

	c.logger.WithField("gs", gs).Info("Syncing with Deletion Timestamp")
	pods, err := c.listGameServerPods(gs)
	if err != nil {
		return gs, err
	}

	if len(pods) > 0 {
		c.logger.WithField("pods", pods).WithField("gsName", gs.ObjectMeta.Name).Info("Found pods, deleting")
		for _, p := range pods {
			err = c.podGetter.Pods(p.ObjectMeta.Namespace).Delete(p.ObjectMeta.Name, nil)
			if err != nil {
				return gs, errors.Wrapf(err, "error deleting pod for GameServer %s, %s", gs.ObjectMeta.Name, p.ObjectMeta.Name)
			}
			c.recorder.Event(gs, corev1.EventTypeNormal, string(gs.Status.State), fmt.Sprintf("Deleting Pod %s", p.ObjectMeta.Name))
		}
		return gs, nil
	}

	gsCopy := gs.DeepCopy()
	// remove the finalizer for this controller
	var fin []string
	for _, f := range gsCopy.ObjectMeta.Finalizers {
		if f != stable.GroupName {
			fin = append(fin, f)
		}
	}
	gsCopy.ObjectMeta.Finalizers = fin
	c.logger.WithField("gs", gsCopy).Infof("No pods found, removing finalizer %s", stable.GroupName)
	gs, err = c.gameServerGetter.GameServers(gsCopy.ObjectMeta.Namespace).Update(gsCopy)
	return gs, errors.Wrapf(err, "error removing finalizer for GameServer %s", gsCopy.ObjectMeta.Name)
}

// syncGameServerPortAllocationState gives a port to a dynamically allocating GameServer
func (c *Controller) syncGameServerPortAllocationState(gs *v1alpha1.GameServer) (*v1alpha1.GameServer, error) {
	if !(gs.Status.State == v1alpha1.PortAllocation && gs.ObjectMeta.DeletionTimestamp.IsZero()) {
		return gs, nil
	}

	gsCopy, err := c.portAllocator.Allocate(gs.DeepCopy())
	if err != nil {
		return gsCopy, errors.Wrapf(err, "error allocating port for GameServer %s", gsCopy.Name)
	}

	gsCopy.Status.State = v1alpha1.Creating
	c.recorder.Event(gs, corev1.EventTypeNormal, string(gs.Status.State), "Port allocated")

	c.logger.WithField("gs", gsCopy).Info("Syncing Port Allocation State")
	gs, err = c.gameServerGetter.GameServers(gs.ObjectMeta.Namespace).Update(gsCopy)
	if err != nil {
		// if the GameServer doesn't get updated with the port data, then put the port
		// back in the pool, as it will get retried on the next pass
		c.portAllocator.DeAllocate(gsCopy)
		return gs, errors.Wrapf(err, "error updating GameServer %s to default values", gs.Name)
	}

	return gs, nil
}

// syncGameServerCreatingState checks if the GameServer is in the Creating state, and if so
// creates a Pod for the GameServer and moves the state to Starting
func (c *Controller) syncGameServerCreatingState(gs *v1alpha1.GameServer) (*v1alpha1.GameServer, error) {
	if !(gs.Status.State == v1alpha1.Creating && gs.ObjectMeta.DeletionTimestamp.IsZero()) {
		return gs, nil
	}

	c.logger.WithField("gs", gs).Info("Syncing Create State")

	// Wait for pod cache sync, so that we don't end up with multiple pods for a GameServer
	if !(cache.WaitForCacheSync(c.stop, c.podSynced)) {
		return nil, errors.New("could not sync pod cache state")
	}

	// Maybe something went wrong, and the pod was created, but the state was never moved to Starting, so let's check
	ret, err := c.listGameServerPods(gs)
	if err != nil {
		return nil, err
	}

	if len(ret) == 0 {
		gs, err = c.createGameServerPod(gs)
		if err != nil || gs.Status.State == v1alpha1.Error {
			return gs, err
		}
	}

	gsCopy := gs.DeepCopy()
	gsCopy.Status.State = v1alpha1.Starting
	gs, err = c.gameServerGetter.GameServers(gs.ObjectMeta.Namespace).Update(gsCopy)
	if err != nil {
		return gs, errors.Wrapf(err, "error updating GameServer %s to Starting state", gs.Name)
	}
	return gs, nil
}

// createGameServerPod creates the backing Pod for a given GameServer
func (c *Controller) createGameServerPod(gs *v1alpha1.GameServer) (*v1alpha1.GameServer, error) {
	sidecar := c.sidecar(gs)
	var pod *corev1.Pod
	pod, err := gs.Pod(sidecar)

	// this shouldn't happen, but if it does.
	if err != nil {
		c.logger.WithField("gameserver", gs).WithError(err).Error("error creating pod from Game Server")
		gs, err = c.moveToErrorState(gs, err.Error())
		return gs, err
	}

	c.addGameServerHealthCheck(gs, pod)

	c.logger.WithField("pod", pod).Info("creating Pod for GameServer")
	pod, err = c.podGetter.Pods(gs.ObjectMeta.Namespace).Create(pod)
	if err != nil {
		if k8serrors.IsInvalid(err) {
			c.logger.WithField("pod", pod).WithField("gameserver", gs).Errorf("Pod created is invalid")
			gs, err = c.moveToErrorState(gs, err.Error())
			return gs, err
		}
		return gs, errors.Wrapf(err, "error creating Pod for GameServer %s", gs.Name)
	}
	c.recorder.Event(gs, corev1.EventTypeNormal, string(gs.Status.State),
		fmt.Sprintf("Pod %s created", pod.ObjectMeta.Name))

	return gs, nil
}

// sidecar creates the sidecar container for a given GameServer
func (c *Controller) sidecar(gs *v1alpha1.GameServer) corev1.Container {
	sidecar := corev1.Container{
		Name:  "agones-gameserver-sidecar",
		Image: c.sidecarImage,
		Env: []corev1.EnvVar{
			{
				Name:  "GAMESERVER_NAME",
				Value: gs.ObjectMeta.Name,
			},
			{
				Name: "POD_NAMESPACE",
				ValueFrom: &corev1.EnvVarSource{
					FieldRef: &corev1.ObjectFieldSelector{
						FieldPath: "metadata.namespace",
					},
				},
			},
		},
		LivenessProbe: &corev1.Probe{
			Handler: corev1.Handler{
				HTTPGet: &corev1.HTTPGetAction{
					Path: "/healthz",
					Port: intstr.FromInt(8080),
				},
			},
			InitialDelaySeconds: 3,
			PeriodSeconds:       3,
		},
	}
	if c.alwaysPullSidecarImage {
		sidecar.ImagePullPolicy = corev1.PullAlways
	}
	return sidecar
}

// addGameServerHealthCheck adds the http health check to the GameServer container
func (c *Controller) addGameServerHealthCheck(gs *v1alpha1.GameServer, pod *corev1.Pod) {
	if !gs.Spec.Health.Disabled {
		for i, c := range pod.Spec.Containers {
			if c.Name == gs.Spec.Container {
				if c.LivenessProbe == nil {
					c.LivenessProbe = &corev1.Probe{
						Handler: corev1.Handler{
							HTTPGet: &corev1.HTTPGetAction{
								Path: "/gshealthz",
								Port: intstr.FromInt(8080),
							},
						},
						InitialDelaySeconds: gs.Spec.Health.InitialDelaySeconds,
						PeriodSeconds:       gs.Spec.Health.PeriodSeconds,
						FailureThreshold:    gs.Spec.Health.FailureThreshold,
					}
					pod.Spec.Containers[i] = c
				}
				break
			}
		}
	}
}

// syncGameServerStartingState looks for a pod that has been scheduled for this GameServer
// and then sets the Status > Address and Ports values.
func (c *Controller) syncGameServerStartingState(gs *v1alpha1.GameServer) (*v1alpha1.GameServer, error) {
	if !(gs.Status.State == v1alpha1.Starting && gs.ObjectMeta.DeletionTimestamp.IsZero()) {
		return gs, nil
	}

	c.logger.WithField("gs", gs).Info("Syncing Starting State")

	// there should be a pod (although it may not have a scheduled container),
	// so if there is an error of any kind, then move this to queue backoff
	pod, err := c.gameServerPod(gs)
	if err != nil {
		return nil, err
	}

	gsCopy := gs.DeepCopy()
	// if we can't get the address, then go into queue backoff
	gsCopy, err = c.applyGameServerAddressAndPort(gsCopy, pod)
	if err != nil {
		return gs, err
	}

	gsCopy.Status.State = v1alpha1.Scheduled
	gs, err = c.gameServerGetter.GameServers(gs.ObjectMeta.Namespace).Update(gsCopy)
	if err != nil {
		return gs, errors.Wrapf(err, "error updating GameServer %s to Scheduled state", gs.Name)
	}
	c.recorder.Event(gs, corev1.EventTypeNormal, string(gs.Status.State), "Address and port populated")

	return gs, nil
}

// applyGameServerAddressAndPort gets the backing Pod for the GamesServer,
// and sets the allocated Address and Port values to it and returns it.
func (c *Controller) applyGameServerAddressAndPort(gs *v1alpha1.GameServer, pod *corev1.Pod) (*v1alpha1.GameServer, error) {
	addr, err := c.address(pod)
	if err != nil {
		return gs, errors.Wrapf(err, "error getting external address for GameServer %s", gs.ObjectMeta.Name)
	}

	gs.Status.Address = addr
	gs.Status.NodeName = pod.Spec.NodeName
	// HostPort is always going to be populated, even when dynamic
	// This will be a double up of information, but it will be easier to read
	gs.Status.Ports = make([]v1alpha1.GameServerStatusPort, len(gs.Spec.Ports))
	for i, p := range gs.Spec.Ports {
		gs.Status.Ports[i] = p.Status()
	}

	return gs, nil
}

// syncGameServerRequestReadyState checks if the Game Server is Requesting to be ready,
// and then adds the IP and Port information to the Status and marks the GameServer
// as Ready
func (c *Controller) syncGameServerRequestReadyState(gs *v1alpha1.GameServer) (*v1alpha1.GameServer, error) {
	if !(gs.Status.State == v1alpha1.RequestReady && gs.ObjectMeta.DeletionTimestamp.IsZero()) ||
		gs.Status.State == v1alpha1.Unhealthy {
		return gs, nil
	}

	c.logger.WithField("gs", gs).Info("Syncing RequestReady State")

	gsCopy := gs.DeepCopy()

	// if the address hasn't been populated, and the Ready request comes
	// before the controller has had a chance to do it, then
	// do it here instead
	addressPopulated := false
	if gs.Status.NodeName == "" {
		addressPopulated = true
		pod, err := c.gameServerPod(gs)
		// errPodNotFound should never happen, and if it does -- something bad happened,
		// so go into workerqueue backoff.
		if err != nil {
			return nil, err
		}
		gsCopy, err = c.applyGameServerAddressAndPort(gsCopy, pod)
		if err != nil {
			return gs, err
		}
	}

	gsCopy.Status.State = v1alpha1.Ready
	gs, err := c.gameServerGetter.GameServers(gs.ObjectMeta.Namespace).Update(gsCopy)
	if err != nil {
		return gs, errors.Wrapf(err, "error setting Ready, Port and address on GameServer %s Status", gs.ObjectMeta.Name)
	}

	if addressPopulated {
		c.recorder.Event(gs, corev1.EventTypeNormal, string(gs.Status.State), "Address and port populated")
	}
	c.recorder.Event(gs, corev1.EventTypeNormal, string(gs.Status.State), "SDK.Ready() executed")
	return gs, nil
}

// syncGameServerShutdownState deletes the GameServer (and therefore the backing Pod) if it is in shutdown state
func (c *Controller) syncGameServerShutdownState(gs *v1alpha1.GameServer) error {
	if !(gs.Status.State == v1alpha1.Shutdown && gs.ObjectMeta.DeletionTimestamp.IsZero()) {
		return nil
	}

	c.logger.WithField("gs", gs).Info("Syncing Shutdown State")
	// be explicit about where to delete. We only need to wait for the Pod to be removed, which we handle with our
	// own finalizer.
	p := metav1.DeletePropagationBackground
	c.allocationMutex.Lock()
	err := c.gameServerGetter.GameServers(gs.ObjectMeta.Namespace).Delete(gs.ObjectMeta.Name, &metav1.DeleteOptions{PropagationPolicy: &p})
	c.allocationMutex.Unlock()
	if err != nil {
		return errors.Wrapf(err, "error deleting Game Server %s", gs.ObjectMeta.Name)
	}
	c.recorder.Event(gs, corev1.EventTypeNormal, string(gs.Status.State), "Deletion started")
	return nil
}

// moveToErrorState moves the GameServer to the error state
func (c *Controller) moveToErrorState(gs *v1alpha1.GameServer, msg string) (*v1alpha1.GameServer, error) {
	copy := gs.DeepCopy()
	copy.Status.State = v1alpha1.Error

	gs, err := c.gameServerGetter.GameServers(gs.ObjectMeta.Namespace).Update(copy)
	if err != nil {
		return gs, errors.Wrapf(err, "error moving GameServer %s to Error State", gs.ObjectMeta.Name)
	}

	c.recorder.Event(gs, corev1.EventTypeWarning, string(gs.Status.State), msg)
	return gs, nil
}

// gameServerPod returns the Pod for this Game Server, or an error if there are none,
// or it cannot be determined (there are more than one, which should not happen)
func (c *Controller) gameServerPod(gs *v1alpha1.GameServer) (*corev1.Pod, error) {
	pods, err := c.listGameServerPods(gs)
	if err != nil {
		return nil, err
	}
	len := len(pods)
	if len == 0 {
		return nil, errPodNotFound
	}
	if len > 1 {
		return nil, errors.Errorf("Found %d pods for Game Server %s", len, gs.ObjectMeta.Name)
	}
	return pods[0], nil
}

// listGameServerPods returns all the Pods that the GameServer created.
// This should only ever be one.
func (c *Controller) listGameServerPods(gs *v1alpha1.GameServer) ([]*corev1.Pod, error) {
	pods, err := c.podLister.List(labels.SelectorFromSet(labels.Set{v1alpha1.GameServerPodLabel: gs.ObjectMeta.Name}))
	if err != nil {
		return pods, errors.Wrapf(err, "error checking if pod exists for GameServer %s", gs.Name)
	}

	// there is a small chance that the GameServer name is not unique, and a Pod for a previous
	// GameServer is has yet to Terminate so check its controller, just to be sure.
	var result []*corev1.Pod
	for _, p := range pods {
		if metav1.IsControlledBy(p, gs) {
			result = append(result, p)
		}
	}

	return result, nil
}

// address returns the IP that the given Pod is being run on
// This should be the externalIP, but if the externalIP is
// not set, it will fall back to the internalIP with a warning.
// (basically because minikube only has an internalIP)
func (c *Controller) address(pod *corev1.Pod) (string, error) {
	node, err := c.nodeLister.Get(pod.Spec.NodeName)
	if err != nil {
		return "", errors.Wrapf(err, "error retrieving node %s for Pod %s", pod.Spec.NodeName, pod.ObjectMeta.Name)
	}

	for _, a := range node.Status.Addresses {
		if a.Type == corev1.NodeExternalIP {
			return a.Address, nil
		}
	}

	// minikube only has an InternalIP on a Node, so we'll fall back to that.
	c.logger.WithField("node", node.ObjectMeta.Name).Warn("Could not find ExternalIP. Falling back to Internal")
	for _, a := range node.Status.Addresses {
		if a.Type == corev1.NodeInternalIP {
			return a.Address, nil
		}
	}

	return "", errors.Errorf("Could not find an address for Node: %s", node.ObjectMeta.Name)
}

// isGameServerPod returns if this Pod is a Pod that comes from a GameServer
func isGameServerPod(pod *corev1.Pod) bool {
	if v1alpha1.GameServerRolePodSelector.Matches(labels.Set(pod.ObjectMeta.Labels)) {
		owner := metav1.GetControllerOf(pod)
		return owner != nil && owner.Kind == "GameServer"
	}

	return false
}
