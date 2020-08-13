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
	"context"
	"io/ioutil"
	"mime"
	"net/http"
	"time"

	allocationv1 "agones.dev/agones/pkg/apis/allocation/v1"
	"agones.dev/agones/pkg/client/clientset/versioned"
	"agones.dev/agones/pkg/client/informers/externalversions"
	"agones.dev/agones/pkg/gameservers"
	"agones.dev/agones/pkg/util/apiserver"
	"agones.dev/agones/pkg/util/https"
	"agones.dev/agones/pkg/util/runtime"
	"github.com/heptiolabs/healthcheck"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/tag"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/record"
)

// Controller is a the GameServerAllocation controller
type Controller struct {
	api        *apiserver.APIServer
	baseLogger *logrus.Entry
	recorder   record.EventRecorder
	allocator  *Allocator
}

// NewController returns a controller for a GameServerAllocation
func NewController(apiServer *apiserver.APIServer,
	health healthcheck.Handler,
	counter *gameservers.PerNodeCounter,
	kubeClient kubernetes.Interface,
	kubeInformerFactory informers.SharedInformerFactory,
	agonesClient versioned.Interface,
	agonesInformerFactory externalversions.SharedInformerFactory,
) *Controller {
	c := &Controller{
		api: apiServer,
		allocator: NewAllocator(
			agonesInformerFactory.Multicluster().V1().GameServerAllocationPolicies(),
			kubeInformerFactory.Core().V1().Secrets(),
			kubeClient,
			NewReadyGameServerCache(agonesInformerFactory.Agones().V1().GameServers(), agonesClient.AgonesV1(), counter, health)),
	}
	c.baseLogger = runtime.NewLoggerWithType(c)

	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(c.baseLogger.Debugf)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: kubeClient.CoreV1().Events("")})
	c.recorder = eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: "GameServerAllocation-controller"})

	return c
}

// registers the api resource for gameserverallocation
func (c *Controller) registerAPIResource(stop <-chan struct{}) {
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
	c.api.AddAPIResource(allocationv1.SchemeGroupVersion.String(), resource, func(w http.ResponseWriter, r *http.Request, n string) error {
		return c.processAllocationRequest(w, r, n, stop)
	})
}

// Run runs this controller. Will block until stop is closed.
// Ignores threadiness, as we only needs 1 worker for cache sync
func (c *Controller) Run(_ int, stop <-chan struct{}) error {
	if err := c.allocator.Start(stop); err != nil {
		return err
	}

	c.registerAPIResource(stop)

	return nil
}

func (c *Controller) processAllocationRequest(w http.ResponseWriter, r *http.Request, namespace string, stop <-chan struct{}) (err error) {
	latency := c.newMetrics(r.Context())
	defer func() {
		if err != nil {
			latency.setError()
		}
		latency.record()
	}()

	if r.Body != nil {
		defer r.Body.Close() // nolint: errcheck
	}

	log := https.LogRequest(c.baseLogger, r)

	if r.Method != http.MethodPost {
		log.Warn("allocation handler only supports POST")
		http.Error(w, "Method not supported", http.StatusMethodNotAllowed)
		latency.setError()
		return
	}

	gsa, err := c.allocationDeserialization(r, namespace)
	if err != nil {
		return err
	}

	latency.setRequest(gsa)

	result, err := c.allocator.Allocate(gsa, stop)
	if err != nil {
		return err
	}
	if status, ok := result.(*metav1.Status); ok {
		w.WriteHeader(int(status.Code))
	}

	latency.setResponse(result)
	err = c.serialisation(r, w, result, scheme.Codecs)
	return err
}

// newMetrics creates a new gsa latency recorder.
func (c *Controller) newMetrics(ctx context.Context) *metrics {
	ctx, err := tag.New(ctx, latencyTags...)
	if err != nil {
		c.baseLogger.WithError(err).Warn("failed to tag latency recorder.")
	}
	return &metrics{
		ctx:              ctx,
		gameServerLister: c.allocator.readyGameServerCache.gameServerLister,
		logger:           c.baseLogger,
		start:            time.Now(),
	}
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
	mt, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil {
		return gsa, errors.Wrap(err, "error parsing mediatype from a request header")
	}
	info, ok := k8sruntime.SerializerInfoForMediaType(mediaTypes, mt)
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
