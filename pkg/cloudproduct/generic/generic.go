// Copyright 2022 Google LLC All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package generic

import (
	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	"agones.dev/agones/pkg/client/informers/externalversions"
	"agones.dev/agones/pkg/portallocator"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
)

func New() (*generic, error) { return &generic{}, nil }

type generic struct{}

func (*generic) SyncPodPortsToGameServer(*agonesv1.GameServer, *corev1.Pod) error { return nil }

func (*generic) NewPortAllocator(minPort, maxPort int32,
	kubeInformerFactory informers.SharedInformerFactory,
	agonesInformerFactory externalversions.SharedInformerFactory) portallocator.Interface {
	return portallocator.New(minPort, maxPort, kubeInformerFactory, agonesInformerFactory)
}

func (*generic) ValidateGameServer(*agonesv1.GameServer) []metav1.StatusCause {
	return nil
}
