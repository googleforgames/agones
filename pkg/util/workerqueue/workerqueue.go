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

// Package workerqueue extends client-go's workqueue
// functionality into an opinionated queue + worker model that
// is reusable
package workerqueue

import (
	"time"

	"agones.dev/agones/pkg/util/runtime"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

// Handler is the handler for processing the work queue
// This is usually a syncronisation handler for a controller or related
type Handler func(string) error

// WorkerQueue is an opinionated queue + worker for use
// with controllers and related and processing Kubernetes watched
// events and synchronising resources
type WorkerQueue struct {
	logger *logrus.Entry
	queue  workqueue.RateLimitingInterface
	// SyncHandler is exported to make testing easier (hack)
	SyncHandler Handler
}

// NewWorkerQueue returns a new worker queue for a given name
func NewWorkerQueue(handler Handler, logger *logrus.Entry, name string) *WorkerQueue {
	return &WorkerQueue{
		logger:      logger.WithField("queue", name),
		queue:       workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), name),
		SyncHandler: handler,
	}
}

// Enqueue puts the name of the runtime.Object in the
// queue to be processed. If you need to send through an
// explicit key, use an cache.ExplicitKey
func (wq *WorkerQueue) Enqueue(obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		err := errors.Wrap(err, "Error creating key for object")
		runtime.HandleError(wq.logger.WithField("obj", obj), err)
		return
	}
	wq.logger.WithField("key", key).Info("Enqueuing key")
	wq.queue.AddRateLimited(key)
}

// runWorker is a long-running function that will continually call the
// processNextWorkItem function in order to read and process a message on the
// workqueue.
func (wq *WorkerQueue) runWorker() {
	for wq.processNextWorkItem() {
	}
}

// processNextWorkItem processes the next work item.
// pretty self explanatory :)
func (wq *WorkerQueue) processNextWorkItem() bool {
	obj, quit := wq.queue.Get()
	if quit {
		return false
	}
	defer wq.queue.Done(obj)

	wq.logger.WithField("obj", obj).Info("Processing obj")

	var key string
	var ok bool
	if key, ok = obj.(string); !ok {
		runtime.HandleError(wq.logger.WithField("obj", obj), errors.Errorf("expected string in queue, but got %T", obj))
		// this is a bad entry, we don't want to reprocess
		wq.queue.Forget(obj)
		return true
	}

	if err := wq.SyncHandler(key); err != nil {
		// we don't forget here, because we want this to be retried via the queue
		runtime.HandleError(wq.logger.WithField("obj", obj), err)
		wq.queue.AddRateLimited(obj)
		return true
	}

	wq.queue.Forget(obj)
	return true
}

// Run the WorkerQueue processing via the Handler. Will block until stop is closed.
// Runs threadiness number workers to process the rate limited queue
func (wq *WorkerQueue) Run(threadiness int, stop <-chan struct{}) {
	defer wq.queue.ShutDown()

	wq.logger.WithField("threadiness", threadiness).Info("Starting workers...")
	for i := 0; i < threadiness; i++ {
		go wait.Until(wq.runWorker, time.Second, stop)
	}

	<-stop
	wq.logger.Info("...shutting down workers")
}
