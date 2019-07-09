// Copyright 2019 Google LLC All Rights Reserved.
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
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation"
)

// Block of const Error messages
const (
	ErrContainerRequired        = "Container is required when using multiple containers in the pod template"
	ErrHostPortDynamic          = "HostPort cannot be specified with a Dynamic PortPolicy"
	ErrPortPolicyStatic         = "PortPolicy must be Static"
	ErrContainerPortRequired    = "ContainerPort must be defined for Dynamic and Static PortPolicies"
	ErrContainerPortPassthrough = "ContainerPort cannot be specified with Passthrough PortPolicy"
)

// crd is an interface to get Name and Kind of CRD
type crd interface {
	GetName() string
	GetObjectKind() schema.ObjectKind
}

// validateName Check NameSize of a CRD
func validateName(c crd) []metav1.StatusCause {
	var causes []metav1.StatusCause
	name := c.GetName()
	kind := c.GetObjectKind().GroupVersionKind().Kind
	// make sure the Name of a Fleet does not oversize the Label size in GSS and GS
	if len(name) > validation.LabelValueMaxLength {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Field:   fmt.Sprintf("Name"),
			Message: fmt.Sprintf("Length of %s '%s' name should be no more than 63 characters.", kind, name),
		})
	}
	return causes
}

// gsSpec is an interface which contains all necessary
// functions to perform common validations against it
type gsSpec interface {
	GetGameServerSpec() *GameServerSpec
}

// validateGSSpec Check GameserverSpec of a CRD
// Used by Fleet and Gameserverset
func validateGSSpec(gs gsSpec) []metav1.StatusCause {
	gsSpec := gs.GetGameServerSpec()
	gsSpec.ApplyDefaults()
	causes, _ := gsSpec.Validate("")

	return causes
}
