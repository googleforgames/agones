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
	"sync"

	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	"agones.dev/agones/pkg/client/informers/externalversions"
	"agones.dev/agones/pkg/cloudproduct/generic"
	"agones.dev/agones/pkg/cloudproduct/gke"
	"agones.dev/agones/pkg/portallocator"
	"agones.dev/agones/pkg/util/runtime"
	"github.com/pkg/errors"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
)

// The Cloud Product abstraction currently consists of disjoint interfaces:
// * ControllerHooks() is an interface available to the pkg/controller binaries
// * api hooks are separated into api group and registered directly with the API
//
// Users should bind flags with Bind{Flags,Env}, then call Initialize with a
// k8s client to initialize.

// ControllerHooksInterface provides a generic interface that abstracts cloud product
// specific functionality for pkg/controller packages.
type ControllerHooksInterface interface {
	// SyncPodPortsToGameServer runs after a Pod has been assigned to a Node and before we sync
	// Pod host ports to the GameServer status.
	SyncPodPortsToGameServer(*agonesv1.GameServer, *corev1.Pod) error

	// NewPortAllocator creates a PortAllocator. See gameservers.NewPortAllocator for parameters.
	NewPortAllocator(int32, int32, informers.SharedInformerFactory, externalversions.SharedInformerFactory) portallocator.Interface
}

const (
	cloudProductFlag = "cloud-product"

	// If --cloud-product=auto, auto-detect
	autoDetectString = "auto"

	genericProduct = "generic"
)

var (
	logger           = runtime.NewLoggerWithSource("cloudproduct")
	productDetectors = []func(context.Context, *kubernetes.Clientset) string{gke.Detect}

	// singleton cloud product interface
	cloudProduct ControllerHooksInterface

	initializeOnce sync.Once
)

// BindFlags binds the --cloud-product flag.
func BindFlags() {
	viper.SetDefault(cloudProductFlag, autoDetectString)
	pflag.String(cloudProductFlag, viper.GetString(cloudProductFlag), "Cloud product. Set to 'auto' to auto-detect, set to 'generic' to force generic behavior, set to 'gke-autopilot' for GKE Autopilot. Can also use CLOUD_PRODUCT env variable.")
}

// BindEnv binds the CLOUD_PRODUCT env variable.
func BindEnv() error {
	return viper.BindEnv(cloudProductFlag)
}

// Initialize initializes the cloud product from the viper flag.
// Caller must use BindFlags and BindEnv prior. If this function returns an
// error, caller should exit, as calling ControllerHooks() after a bad
// initialization will return a nil interface
func Initialize(ctx context.Context, kc *kubernetes.Clientset) (err error) {
	initializeOnce.Do(func() {
		err = initializeFromString(ctx, viper.GetString(cloudProductFlag), kc)
	})
	return err
}

// ControllerHooks returns the global cloud product controller hooks. If the global singleton
// is not set, we initialize the current cloud product to the "generic" product - this should
// only happen in unit tests.
func ControllerHooks() ControllerHooksInterface {
	initializeOnce.Do(func() {
		// If not already initialized, initialize using generic cloud product.
		runtime.Must(initializeFromString(context.Background(), "generic", nil))
	})
	return cloudProduct
}

// initializeFromString initializes the known hooks (controller, API) from a specific string.
// Must be called under initializeOnce.Do() to prevent re-initialization.
func initializeFromString(ctx context.Context, productString string, kc *kubernetes.Clientset) error {
	controllerHooks, v1hooks, err := newFromName(ctx, productString, kc)
	if err != nil {
		return err
	}
	cloudProduct = controllerHooks
	agonesv1.RegisterAPIHooks(v1hooks)
	return nil
}

// newFromName instantiates a new CloudProduct interface by product name.
func newFromName(ctx context.Context, product string, kc *kubernetes.Clientset) (ControllerHooksInterface, agonesv1.APIHooks, error) {
	product = autoDetect(ctx, product, kc)

	switch product {
	case "gke-autopilot":
		return gke.Autopilot()
	case genericProduct:
		return generic.New()
	}
	return nil, nil, errors.Errorf("unknown cloud product: %q", product)
}

func autoDetect(ctx context.Context, product string, kc *kubernetes.Clientset) string {
	if product != autoDetectString {
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
