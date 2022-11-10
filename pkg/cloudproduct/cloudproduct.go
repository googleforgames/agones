// Copyright 2022 Google LLC All Rights Reserved.
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

package cloudproduct

import (
	"context"
	"fmt"

	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	"agones.dev/agones/pkg/client/informers/externalversions"
	"agones.dev/agones/pkg/cloudproduct/generic"
	"agones.dev/agones/pkg/cloudproduct/gke"
	"agones.dev/agones/pkg/portallocator"
	"agones.dev/agones/pkg/util/runtime"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
)

// CloudProduct provides a generic interface that abstracts cloud product
// specific functionality. Users should call New() to instantiate a
// specific cloud product interface.
type CloudProduct interface {
	// SyncPodPortsToGameServer runs after a Pod has been assigned to a Node and before we sync
	// Pod host ports to the GameServer status.
	SyncPodPortsToGameServer(*agonesv1.GameServer, *corev1.Pod) error

	// ValidateGameServer is called by GameServer.Validate to allow for product specific validation.
	ValidateGameServer(*agonesv1.GameServer) []metav1.StatusCause

	// NewPortAllocator creates a PortAllocator. See gameservers.NewPortAllocator for parameters.
	NewPortAllocator(int32, int32, informers.SharedInformerFactory, externalversions.SharedInformerFactory) portallocator.Interface
}

const (
	// If --cloud-product=auto, auto-detect
	AutoDetect = "auto"

	genericProduct = "generic"
)

var (
	logger           = runtime.NewLoggerWithSource("cloudproduct")
	productDetectors = []func(context.Context, *kubernetes.Clientset) string{gke.Detect}
)

// New instantiates a new CloudProduct interface by product name.
func New(ctx context.Context, product string, kc *kubernetes.Clientset) (CloudProduct, error) {
	product = autoDetect(ctx, product, kc)

	switch product {
	case "gke-autopilot":
		return gke.Autopilot()
	case genericProduct:
		return generic.New()
	}
	return nil, fmt.Errorf("unknown cloud product: %q", product)
}

func autoDetect(ctx context.Context, product string, kc *kubernetes.Clientset) string {
	if product != AutoDetect {
		logger.Infof("Cloud product forced to %q, skipping auto-detection", product)
		return product
	}
	for _, detect := range productDetectors {
		product = detect(ctx, kc)
		if product != "" {
			logger.Infof("Cloud product detected as %q", product)
			return product
		}
	}
	logger.Infof("Cloud product defaulted to %q", genericProduct)
	return genericProduct
}

// MustNewGeneric returns the "generic" cloud product, panicking if an error is encountered.
func MustNewGeneric(ctx context.Context) CloudProduct {
	c, err := New(ctx, genericProduct, nil)
	runtime.Must(err)
	return c
}
