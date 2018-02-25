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

// package crd contains utilities for working with
// CustomResourceDefinitions
package crd

import (
	"time"

	"github.com/sirupsen/logrus"
	apiv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	extv1beta1 "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/typed/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

// WaitForEstablishedCRD blocks until CRD comes to an Established state.
// Has a deadline of 60 seconds for this to occur.
func WaitForEstablishedCRD(crdGetter extv1beta1.CustomResourceDefinitionInterface, name string, logger *logrus.Entry) error {
	return wait.PollImmediate(time.Second, 60*time.Second, func() (done bool, err error) {
		crd, err := crdGetter.Get(name, v1.GetOptions{})
		if err != nil {
			return false, err
		}

		for _, cond := range crd.Status.Conditions {
			switch cond.Type {
			case apiv1beta1.Established:
				if cond.Status == apiv1beta1.ConditionTrue {
					logger.WithField("crd", crd.ObjectMeta.Name).Info("custom resource definition established")
					return true, err
				}
			}
		}

		return false, nil
	})
}
