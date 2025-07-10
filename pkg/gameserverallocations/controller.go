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
	"io"
	"mime"
	"net/http"
	"os"
	"time"

	"github.com/heptiolabs/healthcheck"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/record"

	"agones.dev/agones/pkg/allocation/converters"
	pb "agones.dev/agones/pkg/allocation/go"
	allocationv1 "agones.dev/agones/pkg/apis/allocation/v1"
	"agones.dev/agones/pkg/client/clientset/versioned"
	"agones.dev/agones/pkg/client/informers/externalversions"
	"agones.dev/agones/pkg/gameserverallocations/distributedallocator/buffer"
	"agones.dev/agones/pkg/gameservers"
	"agones.dev/agones/pkg/util/apiserver"
	"agones.dev/agones/pkg/util/https"
	"agones.dev/agones/pkg/util/leader"
	"agones.dev/agones/pkg/util/runtime"
)

func init() {
	registerViews()
}

// Extensions is a GameServerAllocation controller within the Extensions service
type Extensions struct {
	api        *apiserver.APIServer
	baseLogger *logrus.Entry
	recorder   record.EventRecorder
	allocator  *Allocator

	requestBuffer chan *buffer.PendingRequest
}

// NewExtensions returns the extensions controller for a GameServerAllocation
func NewExtensions(apiServer *apiserver.APIServer,
	health healthcheck.Handler,
	counter *gameservers.PerNodeCounter,
	kubeClient kubernetes.Interface,
	kubeInformerFactory informers.SharedInformerFactory,
	agonesClient versioned.Interface,
	agonesInformerFactory externalversions.SharedInformerFactory,
	remoteAllocationTimeout time.Duration,
	totalAllocationTimeout time.Duration,
	allocationBatchWaitTime time.Duration,
) *Extensions {
	c := &Extensions{
		api: apiServer,
		allocator: NewAllocator(
			agonesInformerFactory.Multicluster().V1().GameServerAllocationPolicies(),
			kubeInformerFactory.Core().V1().Secrets(),
			agonesClient.AgonesV1(),
			kubeClient,
			NewAllocationCache(agonesInformerFactory.Agones().V1().GameServers(), counter, health),
			remoteAllocationTimeout,
			totalAllocationTimeout,
			allocationBatchWaitTime),

		requestBuffer: make(chan *buffer.PendingRequest, 1000),
	}
	c.baseLogger = runtime.NewLoggerWithType(c)

	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(c.baseLogger.Debugf)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: kubeClient.CoreV1().Events("")})
	c.recorder = eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: "GameServerAllocation-controller"})

	bufferedAllocatorEnabled := true
	if bufferedAllocatorEnabled {
		batchTimeout := 500 * time.Millisecond
		maxBatchSize := 100
		clientID := os.Getenv("POD_NAME")
		batchSource := buffer.BatchSourceFromPendingRequests(
			context.TODO(), c.requestBuffer, batchTimeout, maxBatchSize,
		)
		go leader.RunLeaderTracking(context.TODO(), kubeClient, "agones-processor-leader-election", "agones-system", 8443, func(clientCtx context.Context, addr string) {
			err := buffer.PullAndDispatchBatches(clientCtx, addr, clientID, batchSource)
			if err != nil {
				c.baseLogger.WithError(err).Error("processor client exited")
			}
		})
	}

	return c
}

// registers the api resource for gameserverallocation
func (c *Extensions) registerAPIResource(ctx context.Context) {
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
		bufferedAllocatorEnabled := true
		if bufferedAllocatorEnabled {
			return c.processBufferedAllocationRequest(ctx, w, r, n)
		}
		return c.processAllocationRequest(ctx, w, r, n)
	})
}

// Run runs this extensions controller. Will block until stop is closed.
// Ignores threadiness, as we only needs 1 worker for cache sync
func (c *Extensions) Run(ctx context.Context, _ int) error {
	if err := c.allocator.Run(ctx); err != nil {
		return err
	}

	c.registerAPIResource(ctx)

	return nil
}

func (c *Extensions) processAllocationRequest(ctx context.Context, w http.ResponseWriter, r *http.Request, namespace string) (err error) {
	if r.Body != nil {
		defer r.Body.Close() // nolint: errcheck
	}

	log := https.LogRequest(c.baseLogger, r)

	if r.Method != http.MethodPost {
		log.Warn("allocation handler only supports POST")
		http.Error(w, "Method not supported", http.StatusMethodNotAllowed)
		return nil
	}

	gsa, err := c.AllocationDeserialization(r, namespace)
	if err != nil {
		return err
	}

	result, err := c.allocator.Allocate(ctx, gsa)
	if err != nil {
		return err
	}
	var code int
	switch obj := result.(type) {
	case *metav1.Status:
		code = int(obj.Code)
	case *allocationv1.GameServerAllocation:
		code = http.StatusCreated
	default:
		code = http.StatusOK
	}

	err = c.Serialisation(r, w, result, code, scheme.Codecs)
	return err
}

// AllocationDeserialization processes the request and namespace, and attempts to deserialise its values
// into a GameServerAllocation. Returns an error if it fails for whatever reason.
func (c *Extensions) AllocationDeserialization(r *http.Request, namespace string) (*allocationv1.GameServerAllocation, error) {
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

	b, err := io.ReadAll(r.Body)
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

// Serialisation takes a runtime.Object, and serialise it to the ResponseWriter in the requested format
func (c *Extensions) Serialisation(r *http.Request, w http.ResponseWriter, obj k8sruntime.Object, statusCode int, codecs serializer.CodecFactory) error {
	info, err := apiserver.AcceptedSerializer(r, codecs)
	if err != nil {
		return errors.Wrapf(err, "failed to find Serialisation info for %T object", obj)
	}

	w.Header().Set("Content-Type", info.MediaType)
	// we have to do this here, so that the content type is set before we send a HTTP status header, as the WriteHeader
	// call will send data to the client.
	w.WriteHeader(statusCode)

	err = info.Serializer.Encode(obj, w)
	return errors.Wrapf(err, "error encoding %T", obj)
}

func (c *Extensions) processBufferedAllocationRequest(ctx context.Context, w http.ResponseWriter, r *http.Request, namespace string) error {
	if r.Body != nil {
		defer r.Body.Close()
	}
	if r.Method != http.MethodPost {
		http.Error(w, "Method not supported", http.StatusMethodNotAllowed)
		return nil
	}
	gsa, err := c.AllocationDeserialization(r, namespace)
	if err != nil {
		return err
	}
	in := converters.ConvertGSAToAllocationRequest(gsa)
	pr := &buffer.PendingRequest{
		Req:    in,
		RespCh: make(chan *pb.AllocationResponse, 1),
		ErrCh:  make(chan error, 1),
	}
	c.requestBuffer <- pr

	var result k8sruntime.Object
	var code int

	select {
	case resp := <-pr.RespCh:
		source := ""
		if resp != nil {
			source = resp.Source
		}
		result = converters.ConvertAllocationResponseToGSA(resp, source)
		code = http.StatusCreated
	case err := <-pr.ErrCh:
		c.baseLogger.WithField("request", in).WithError(err).Error("[Extensions] Error from processor")
		return err
	case <-ctx.Done():
		c.baseLogger.WithField("request", in).Error("[Extensions] Context cancelled while waiting for processor response")
		return ctx.Err()
	}

	return c.Serialisation(r, w, result, code, scheme.Codecs)
}
