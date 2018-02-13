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
	"fmt"
	"net/http"
	"time"

	"agones.dev/agones/pkg/apis/stable"
	stablev1alpha1 "agones.dev/agones/pkg/apis/stable/v1alpha1"
	"agones.dev/agones/pkg/client/clientset/versioned"
	getterv1alpha1 "agones.dev/agones/pkg/client/clientset/versioned/typed/stable/v1alpha1"
	"agones.dev/agones/pkg/client/informers/externalversions"
	listerv1alpha1 "agones.dev/agones/pkg/client/listers/stable/v1alpha1"
	"agones.dev/agones/pkg/health"
	"agones.dev/agones/pkg/util/runtime"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	apiv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	extclientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/typed/apiextensions/v1beta1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	corelisterv1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
)

var (
	errPodNotFound = errors.New("A Pod for this GameServer Was Not Found")
)

// Controller is a the main GameServer crd controller
type Controller struct {
	sidecarImage           string
	alwaysPullSidecarImage bool
	crdGetter              v1beta1.CustomResourceDefinitionInterface
	podGetter              typedcorev1.PodsGetter
	podLister              corelisterv1.PodLister
	gameServerGetter       getterv1alpha1.GameServersGetter
	gameServerLister       listerv1alpha1.GameServerLister
	gameServerSynced       cache.InformerSynced
	nodeLister             corelisterv1.NodeLister
	queue                  workqueue.RateLimitingInterface
	portAllocator          *PortAllocator
	monitor                health.Monitor
	server                 *http.Server
	recorder               record.EventRecorder
	// this allows for overwriting for testing purposes
	syncHandler func(string) error
}

// NewController returns a new gameserver crd controller
func NewController(minPort, maxPort int32,
	sidecarImage string,
	alwaysPullSidecarImage bool,
	kubeClient kubernetes.Interface,
	kubeInformerFactory informers.SharedInformerFactory,
	extClient extclientset.Interface,
	agonesClient versioned.Interface,
	agonesInformerFactory externalversions.SharedInformerFactory,
	monitor health.Monitor,
) *Controller {

	gameServers := agonesInformerFactory.Stable().V1alpha1().GameServers()
	gsInformer := gameServers.Informer()

	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(logrus.Infof)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: kubeClient.CoreV1().Events("")})
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: "gameserver-controller"})

	c := &Controller{
		sidecarImage:           sidecarImage,
		alwaysPullSidecarImage: alwaysPullSidecarImage,
		crdGetter:              extClient.ApiextensionsV1beta1().CustomResourceDefinitions(),
		podGetter:              kubeClient.CoreV1(),
		podLister:              kubeInformerFactory.Core().V1().Pods().Lister(),
		gameServerGetter:       agonesClient.StableV1alpha1(),
		gameServerLister:       gameServers.Lister(),
		gameServerSynced:       gsInformer.HasSynced,
		nodeLister:             kubeInformerFactory.Core().V1().Nodes().Lister(),
		queue:                  workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), stable.GroupName+".GameServerController"),
		portAllocator:          NewPortAllocator(minPort, maxPort, kubeInformerFactory, agonesInformerFactory),
		monitor:                monitor,
		recorder:               recorder,
	}

	gsInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: c.enqueueGameServer,
		UpdateFunc: func(oldObj, newObj interface{}) {
			// no point in processing unless there is a State change
			oldGs := oldObj.(*stablev1alpha1.GameServer)
			newGs := newObj.(*stablev1alpha1.GameServer)
			if oldGs.Status.State != newGs.Status.State || oldGs.ObjectMeta.DeletionTimestamp != newGs.ObjectMeta.DeletionTimestamp {
				c.enqueueGameServer(newGs)
			}
		},
	})

	// track pod deletions, for when GameServers are deleted
	kubeInformerFactory.Core().V1().Pods().Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		DeleteFunc: func(obj interface{}) {
			pod := obj.(*corev1.Pod)
			if stablev1alpha1.GameServerRolePodSelector.Matches(labels.Set(pod.ObjectMeta.Labels)) {
				if owner := metav1.GetControllerOf(pod); owner != nil {
					c.enqueueGameServer(cache.ExplicitKey(pod.ObjectMeta.Namespace + "/" + owner.Name))
				}
			}
		},
	})

	c.syncHandler = c.syncGameServer

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write([]byte("ok"))
		if err != nil {
			logrus.WithError(err).Error("could not send ok response on healthz")
			w.WriteHeader(http.StatusInternalServerError)
		}
	})

	c.server = &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	return c
}

// Run the GameServer controller. Will block until stop is closed.
// Runs threadiness number workers to process the rate limited queue
func (c Controller) Run(threadiness int, stop <-chan struct{}) error {
	defer c.queue.ShutDown()

	logrus.Info("Starting health check...")
	go func() {
		if err := c.server.ListenAndServe(); err != nil {
			if err == http.ErrServerClosed {
				logrus.WithError(err).Info("health check: http server closed")
			} else {
				err := errors.Wrap(err, "Could not listen on :8080")
				runtime.HandleError(logrus.WithError(err), err)
			}
		}
	}()
	defer c.server.Close() // nolint: errcheck

	err := c.waitForEstablishedCRD()
	if err != nil {
		return err
	}

	logrus.Info("Wait for cache sync")
	if !cache.WaitForCacheSync(stop, c.gameServerSynced) {
		return errors.New("failed to wait for caches to sync")
	}

	// Run the Port Allocator
	if err := c.portAllocator.Run(stop); err != nil {
		return err
	}

	// Run the Health Monitor
	go c.monitor.Run(stop)

	logrus.Info("Starting workers...")
	for i := 0; i < threadiness; i++ {
		go wait.Until(c.runWorker, time.Second, stop)
	}

	<-stop
	return nil
}

// enqueueGameServer puts the name of the GameServer in the
// queue to be processed. This should not be passed any object
// other than a GameServer.
func (c Controller) enqueueGameServer(obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		err := errors.Wrap(err, "Error creating key for object")
		runtime.HandleError(logrus.WithField("obj", obj), err)
		return
	}
	c.queue.AddRateLimited(key)
}

// runWorker is a long-running function that will continually call the
// processNextWorkItem function in order to read and process a message on the
// workqueue.
func (c *Controller) runWorker() {
	for c.processNextWorkItem() {
	}
}

func (c *Controller) processNextWorkItem() bool {
	obj, quit := c.queue.Get()
	if quit {
		return false
	}
	defer c.queue.Done(obj)

	logrus.WithField("obj", obj).Info("Processing obj")

	var key string
	var ok bool
	if key, ok = obj.(string); !ok {
		runtime.HandleError(logrus.WithField("obj", obj), errors.Errorf("expected string in queue, but got %T", obj))
		// this is a bad entry, we don't want to reprocess
		c.queue.Forget(obj)
		return true
	}

	if err := c.syncHandler(key); err != nil {
		// we don't forget here, because we want this to be retried via the queue
		runtime.HandleError(logrus.WithField("obj", obj), err)
		c.queue.AddRateLimited(obj)
		return true
	}

	c.queue.Forget(obj)
	return true
}

// syncGameServer synchronises the Pods for the GameServers.
// and reacts to status changes that can occur through the client SDK
func (c *Controller) syncGameServer(key string) error {
	logrus.WithField("key", key).Info("Synchronising")

	// Convert the namespace/name string into a distinct namespace and name
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		// don't return an error, as we don't want this retried
		runtime.HandleError(logrus.WithField("key", key), errors.Wrapf(err, "invalid resource key"))
		return nil
	}

	gs, err := c.gameServerLister.GameServers(namespace).Get(name)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			logrus.WithField("key", key).Info("GameServer is no longer available for syncing")
			return nil
		}
		return errors.Wrapf(err, "error retrieving GameServer %s from namespace %s", name, namespace)
	}

	if gs, err = c.syncGameServerDeletionTimestamp(gs); err != nil {
		return err
	}
	if gs, err = c.syncGameServerBlankState(gs); err != nil {
		return err
	}
	if gs, err = c.syncGameServerCreatingState(gs); err != nil {
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
func (c *Controller) syncGameServerDeletionTimestamp(gs *stablev1alpha1.GameServer) (*stablev1alpha1.GameServer, error) {
	if gs.ObjectMeta.DeletionTimestamp.IsZero() {
		return gs, nil
	}

	logrus.WithField("gs", gs).Info("Syncing with Deletion Timestamp")
	pods, err := c.listGameServerPods(gs)
	if err != nil {
		return gs, err
	}

	if len(pods) > 0 {
		logrus.WithField("pods", pods).WithField("gsName", gs.ObjectMeta.Name).Info("Found pods, deleting")
		for _, p := range pods {
			err := c.podGetter.Pods(p.ObjectMeta.Namespace).Delete(p.ObjectMeta.Name, nil)
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
	logrus.WithField("gs", gsCopy).Infof("No pods found, removing finalizer %s", stable.GroupName)
	gs, err = c.gameServerGetter.GameServers(gsCopy.ObjectMeta.Namespace).Update(gsCopy)
	return gs, errors.Wrapf(err, "error removing finalizer for GameServer %s", gsCopy.ObjectMeta.Name)
}

// syncGameServerBlankState applies default values to the the GameServer if its state is "" (blank)
// returns an updated GameServer
func (c *Controller) syncGameServerBlankState(gs *stablev1alpha1.GameServer) (*stablev1alpha1.GameServer, error) {
	if !(gs.Status.State == "" && gs.ObjectMeta.DeletionTimestamp.IsZero()) {
		return gs, nil
	}

	gsCopy := gs.DeepCopy()
	gsCopy.ApplyDefaults()

	// manage dynamic ports
	if gsCopy.Spec.PortPolicy == stablev1alpha1.Dynamic {
		port, err := c.portAllocator.Allocate()
		if err != nil {
			return gsCopy, errors.Wrapf(err, "error allocating port for GameServer %s", gsCopy.Name)
		}
		gsCopy.Spec.HostPort = port
	}

	logrus.WithField("gs", gsCopy).Info("Syncing Blank State")
	var err error
	gs, err = c.gameServerGetter.GameServers(gs.ObjectMeta.Namespace).Update(gsCopy)
	if err != nil {
		return gs, errors.Wrapf(err, "error updating GameServer %s to default values", gs.Name)
	}

	c.recorder.Event(gs, corev1.EventTypeNormal, string(gs.Status.State), "Defaults applied")
	return gs, nil
}

// syncGameServerCreatingState checks if the GameServer is in the Creating state, and if so
// creates a Pod for the GameServer and moves the state to Starting
func (c *Controller) syncGameServerCreatingState(gs *stablev1alpha1.GameServer) (*stablev1alpha1.GameServer, error) {
	if !(gs.Status.State == stablev1alpha1.Creating && gs.ObjectMeta.DeletionTimestamp.IsZero()) {
		return gs, nil
	}

	logrus.WithField("gs", gs).Info("Syncing Create State")

	// Maybe something went wrong, and the pod was created, but the state was never moved to Starting, so let's check
	ret, err := c.listGameServerPods(gs)
	if err != nil {
		return nil, err
	}

	if len(ret) == 0 {
		sidecar := c.sidecar(gs)
		pod, err := gs.Pod(sidecar)

		// this shouldn't happen, but if it does.
		if err != nil {
			logrus.WithField("gameserver", gs).WithError(err).Error("error creating pod from Game Server")
			return c.moveToErrorState(gs, err.Error())
		}

		c.addGameServerHealthCheck(gs, pod)

		logrus.WithField("pod", pod).Info("creating Pod for GameServer")
		pod, err = c.podGetter.Pods(gs.ObjectMeta.Namespace).Create(pod)
		if err != nil {
			if k8serrors.IsInvalid(err) {
				logrus.WithField("pod", pod).WithField("gameserver", gs).Errorf("Pod created is invalid")
				return c.moveToErrorState(gs, err.Error())
			}
			return gs, errors.Wrapf(err, "error creating Pod for GameServer %s", gs.Name)
		}
		c.recorder.Event(gs, corev1.EventTypeNormal, string(gs.Status.State),
			fmt.Sprintf("Pod %s created", pod.ObjectMeta.Name))
	}

	gsCopy := gs.DeepCopy()
	gsCopy.Status.State = stablev1alpha1.Starting
	gs, err = c.gameServerGetter.GameServers(gs.ObjectMeta.Namespace).Update(gsCopy)
	if err != nil {
		return gs, errors.Wrapf(err, "error updating GameServer %s to Starting state", gs.Name)
	}

	c.recorder.Event(gs, corev1.EventTypeNormal, string(gs.Status.State), "Synced")
	return gs, nil
}

// sidecar creates the sidecar container for a given GameServer
func (c *Controller) sidecar(gs *stablev1alpha1.GameServer) corev1.Container {
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
func (c *Controller) addGameServerHealthCheck(gs *stablev1alpha1.GameServer, pod *corev1.Pod) {
	if !gs.Spec.Health.Disabled {
		for i, c := range pod.Spec.Containers {
			logrus.WithField("c", c.Name).WithField("Container", gs.Spec.Container).Info("Checking name and container")
			if c.Name == gs.Spec.Container {
				logrus.WithField("liveness", c.LivenessProbe).Info("Found container")
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
					logrus.WithField("container", c).WithField("pod", pod).Info("Final pod")
					pod.Spec.Containers[i] = c
				}
				break
			}
		}
	}
}

// syncGameServerRequestReadyState checks if the Game Server is Requesting to be ready,
// and then adds the IP and Port information to the Status and marks the GameServer
// as Ready
func (c *Controller) syncGameServerRequestReadyState(gs *stablev1alpha1.GameServer) (*stablev1alpha1.GameServer, error) {
	if !(gs.Status.State == stablev1alpha1.RequestReady && gs.ObjectMeta.DeletionTimestamp.IsZero()) ||
		gs.Status.State == stablev1alpha1.Unhealthy {
		return gs, nil
	}

	logrus.WithField("gs", gs).Info("Syncing RequestReady State")

	pod, err := c.gameServerPod(gs)
	if err != nil {
		return gs, errors.Wrapf(err, "error getting pod for GameServer %s", gs.ObjectMeta.Name)
	}
	addr, err := c.Address(pod)
	if err != nil {
		return gs, errors.Wrapf(err, "error getting external Address for GameServer %s", gs.ObjectMeta.Name)
	}

	gsCopy := gs.DeepCopy()
	gsCopy.Status.State = stablev1alpha1.Ready
	gsCopy.Status.Address = addr
	gsCopy.Status.NodeName = pod.Spec.NodeName
	// HostPort is always going to be populated, even when dynamic
	// This will be a double up of information, but it will be easier to read
	gsCopy.Status.Port = gs.Spec.HostPort

	gs, err = c.gameServerGetter.GameServers(gs.ObjectMeta.Namespace).Update(gsCopy)
	if err != nil {
		return gs, errors.Wrapf(err, "error setting Ready, Port and Address on GameServer %s Status", gs.ObjectMeta.Name)
	}

	c.recorder.Event(gs, corev1.EventTypeNormal, string(gs.Status.State), "Address and Port populated")
	return gs, nil
}

// syncGameServerShutdownState deletes the GameServer (and therefore the backing Pod) if it is in shutdown state
func (c *Controller) syncGameServerShutdownState(gs *stablev1alpha1.GameServer) error {
	if !(gs.Status.State == stablev1alpha1.Shutdown && gs.ObjectMeta.DeletionTimestamp.IsZero()) {
		return nil
	}

	logrus.WithField("gs", gs).Info("Syncing Shutdown State")
	// Do it in the foreground, so the GameServer gets killed last
	p := metav1.DeletePropagationForeground
	err := c.gameServerGetter.GameServers(gs.ObjectMeta.Namespace).Delete(gs.ObjectMeta.Name, &metav1.DeleteOptions{PropagationPolicy: &p})
	if err != nil {
		return errors.Wrapf(err, "error deleting Game Server %s", gs.ObjectMeta.Name)
	}
	c.recorder.Event(gs, corev1.EventTypeNormal, string(gs.Status.State), "Deletion started")
	return nil
}

// moveToErrorState moves the GameServer to the error state
func (c Controller) moveToErrorState(gs *stablev1alpha1.GameServer, msg string) (*stablev1alpha1.GameServer, error) {
	copy := gs.DeepCopy()
	copy.Status.State = stablev1alpha1.Error

	gs, err := c.gameServerGetter.GameServers(gs.ObjectMeta.Namespace).Update(copy)
	if err != nil {
		return gs, errors.Wrapf(err, "error moving GameServer %s to Error State", gs.ObjectMeta.Name)
	}

	c.recorder.Event(gs, corev1.EventTypeWarning, string(gs.Status.State), msg)
	return gs, nil
}

// gameServerPod returns the Pod for this Game Server, or an error if there are none,
// or it cannot be determined (there are more than one, which should not happen)
func (c *Controller) gameServerPod(gs *stablev1alpha1.GameServer) (*corev1.Pod, error) {
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
func (c *Controller) listGameServerPods(gs *stablev1alpha1.GameServer) ([]*corev1.Pod, error) {
	pods, err := c.podLister.List(labels.SelectorFromSet(labels.Set{stablev1alpha1.GameServerPodLabel: gs.ObjectMeta.Name}))
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

// Address returns the IP that the given Pod is being run on
// This should be the externalIP, but if the externalIP is
// not set, it will fall back to the internalIP with a warning.
// (basically because minikube only has an internalIP)
func (c Controller) Address(pod *corev1.Pod) (string, error) {
	node, err := c.nodeLister.Get(pod.Spec.NodeName)
	if err != nil {
		return "", errors.Wrapf(err, "error retrieving node %s for Pod %s", node.ObjectMeta.Name, pod.ObjectMeta.Name)
	}

	for _, a := range node.Status.Addresses {
		if a.Type == corev1.NodeExternalIP {
			return a.Address, nil
		}
	}

	// minikube only has an InternalIP on a Node, so we'll fall back to that.
	logrus.WithField("node", node.ObjectMeta.Name).Warn("Could not find ExternalIP. Falling back to Internal")
	for _, a := range node.Status.Addresses {
		if a.Type == corev1.NodeInternalIP {
			return a.Address, nil
		}
	}

	return "", errors.Errorf("Could not find an Address for Node: %s", node.ObjectMeta.Name)
}

// waitForEstablishedCRD blocks until CRD comes to an Established state.
// Has a deadline of 60 seconds for this to occur.
func (c Controller) waitForEstablishedCRD() error {
	return wait.PollImmediate(500*time.Millisecond, 60*time.Second, func() (done bool, err error) {
		crd, err := c.crdGetter.Get("gameservers.stable.agones.dev", metav1.GetOptions{})
		if err != nil {
			return false, err
		}

		for _, cond := range crd.Status.Conditions {
			switch cond.Type {
			case apiv1beta1.Established:
				if cond.Status == apiv1beta1.ConditionTrue {
					logrus.WithField("crd", crd).Info("GameServer custom resource definition is established")
					return true, err
				}
			}
		}

		return false, nil
	})
}
