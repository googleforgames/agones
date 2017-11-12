// Copyright 2017 Google Inc. All Rights Reserved.
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

package main

import (
	"time"

	"github.com/agonio/agon/pkg/apis/stable/v1alpha1"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	apiv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/typed/apiextensions/v1beta1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

// NewController returns a new gameserver crd controller
func NewController(crds v1beta1.CustomResourceDefinitionInterface) *Controller {
	return &Controller{crds: crds}
}

// Controller is a gameserver crd controller
type Controller struct {
	crds v1beta1.CustomResourceDefinitionInterface
}

// Run the gameserver controller. Will block until stop is closed.
func (c Controller) Run(stop <-chan struct{}) error {
	err := c.createCRDIfDoesntExist()
	if err != nil {
		return err
	}
	err = c.waitForEstablishedCRD()
	if err != nil {
		return err
	}

	<-stop
	return nil
}

// createCRDIfDoesntExist creates the GameServer CRD if it doesn't exist.
// only returns an error if something goes wrong
func (c Controller) createCRDIfDoesntExist() error {
	crd, err := c.crds.Create(v1alpha1.GameServerCRD())
	if err != nil {
		if !k8serrors.IsAlreadyExists(err) {
			return errors.Wrap(err, "Error creating gameserver custom resource definition")
		}
		logrus.Info("gameserver custom resource definition already exists.")
	} else {
		logrus.WithField("crd", crd).Info("gameserver custom resource definition created successfully")
	}

	return nil
}

// waitForEstablishedCRD blocks until CRD comes to an Established state.
// Has a deadline of 60 seconds for this to occur.
func (c Controller) waitForEstablishedCRD() error {
	crdName := v1alpha1.GameServerCRD().ObjectMeta.Name
	return wait.PollImmediate(500*time.Millisecond, 60*time.Second, func() (done bool, err error) {
		crd, err := c.crds.Get(crdName, metav1.GetOptions{})
		if err != nil {
			return false, err
		}

		for _, cond := range crd.Status.Conditions {
			switch cond.Type {
			case apiv1beta1.Established:
				if cond.Status == apiv1beta1.ConditionTrue {
					logrus.WithField("crd", crd).Info("gameserver custom resource definition is established")
					return true, err
				}
			}
		}

		return false, nil
	})
}
