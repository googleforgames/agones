// Copyright 2024 Google LLC All Rights Reserved.
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

// This code was autogenerated. Do not edit directly.

// Code generated by informer-gen. DO NOT EDIT.

package v1

import (
	"context"
	time "time"

	autoscalingv1 "agones.dev/agones/pkg/apis/autoscaling/v1"
	versioned "agones.dev/agones/pkg/client/clientset/versioned"
	internalinterfaces "agones.dev/agones/pkg/client/informers/externalversions/internalinterfaces"
	v1 "agones.dev/agones/pkg/client/listers/autoscaling/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	watch "k8s.io/apimachinery/pkg/watch"
	cache "k8s.io/client-go/tools/cache"
)

// FleetAutoscalerInformer provides access to a shared informer and lister for
// FleetAutoscalers.
type FleetAutoscalerInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() v1.FleetAutoscalerLister
}

type fleetAutoscalerInformer struct {
	factory          internalinterfaces.SharedInformerFactory
	tweakListOptions internalinterfaces.TweakListOptionsFunc
	namespace        string
}

// NewFleetAutoscalerInformer constructs a new informer for FleetAutoscaler type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewFleetAutoscalerInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers) cache.SharedIndexInformer {
	return NewFilteredFleetAutoscalerInformer(client, namespace, resyncPeriod, indexers, nil)
}

// NewFilteredFleetAutoscalerInformer constructs a new informer for FleetAutoscaler type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewFilteredFleetAutoscalerInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers, tweakListOptions internalinterfaces.TweakListOptionsFunc) cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.AutoscalingV1().FleetAutoscalers(namespace).List(context.TODO(), options)
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.AutoscalingV1().FleetAutoscalers(namespace).Watch(context.TODO(), options)
			},
		},
		&autoscalingv1.FleetAutoscaler{},
		resyncPeriod,
		indexers,
	)
}

func (f *fleetAutoscalerInformer) defaultInformer(client versioned.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	return NewFilteredFleetAutoscalerInformer(client, f.namespace, resyncPeriod, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}, f.tweakListOptions)
}

func (f *fleetAutoscalerInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&autoscalingv1.FleetAutoscaler{}, f.defaultInformer)
}

func (f *fleetAutoscalerInformer) Lister() v1.FleetAutoscalerLister {
	return v1.NewFleetAutoscalerLister(f.Informer().GetIndexer())
}
