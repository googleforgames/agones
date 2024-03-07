// Copyright 2017 Google LLC All Rights Reserved.
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

package v1

import (
	"agones.dev/agones/pkg/apis/agones"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/scheme"
)

// VersionAnnotation is the key for version annotation
// associated with the CRD
const VersionAnnotation = agones.GroupName + "/sdk-version"

// SchemeGroupVersion is group version used to register these objects
var SchemeGroupVersion = schema.GroupVersion{Group: agones.GroupName, Version: "v1"}

// Kind takes an unqualified kind and returns back a Group qualified GroupKind
func Kind(kind string) schema.GroupKind {
	return SchemeGroupVersion.WithKind(kind).GroupKind()
}

// Resource takes an unqualified resource and returns a Group qualified GroupResource
func Resource(resource string) schema.GroupResource {
	return SchemeGroupVersion.WithResource(resource).GroupResource()
}

var (
	// SchemeBuilder registers our types
	SchemeBuilder = k8sruntime.NewSchemeBuilder(addKnownTypes)
	// AddToScheme local alias for SchemeBuilder.AddToScheme
	AddToScheme = SchemeBuilder.AddToScheme
)

func init() {
	if err := AddToScheme(scheme.Scheme); err != nil {
		panic(err)
	}
}

// Adds the list of known types to api.Scheme.
func addKnownTypes(apiScheme *k8sruntime.Scheme) error {
	apiScheme.AddKnownTypes(SchemeGroupVersion,
		&GameServer{},
		&GameServerList{},
		&GameServerSet{},
		&GameServerSetList{},
		&Fleet{},
		&FleetList{},
	)
	metav1.AddToGroupVersion(apiScheme, SchemeGroupVersion)
	return nil
}
