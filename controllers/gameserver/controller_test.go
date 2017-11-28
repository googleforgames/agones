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
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/agonio/agon/pkg/apis/stable"
	"github.com/agonio/agon/pkg/apis/stable/v1alpha1"
	"github.com/stretchr/testify/assert"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/fake"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	k8stesting "k8s.io/client-go/testing"
)

func TestControllerCreateCRDIfDoesntExist(t *testing.T) {
	t.Parallel()

	t.Run("CRD doesn't exist", func(t *testing.T) {
		con, cs := newFakeController()
		var crd *v1beta1.CustomResourceDefinition
		cs.AddReactor("create", "customresourcedefinitions", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
			a := action.(k8stesting.CreateAction)
			crd = a.GetObject().(*v1beta1.CustomResourceDefinition)
			return true, nil, nil
		})

		err := con.createCRDIfDoesntExist()
		assert.Nil(t, err, "CRD Should be created: %v", err)
		assert.Equal(t, v1alpha1.GameServerCRD(), crd)
	})

	t.Run("CRD does exist", func(t *testing.T) {
		con, cs := newFakeController()
		cs.AddReactor("create", "customresourcedefinitions", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
			err = k8serrors.NewAlreadyExists(schema.GroupResource{Group: stable.GroupName, Resource: "gameserver"}, "Foo")
			return true, nil, err
		})
		err := con.createCRDIfDoesntExist()
		assert.Nil(t, err, "CRD Should not be created, but not throw an error: %v", err)
	})

	t.Run("Something bad happens", func(t *testing.T) {
		con, cs := newFakeController()
		fixture := errors.New("this is a custom error")
		cs.AddReactor("create", "customresourcedefinitions", func(action k8stesting.Action) (bool, runtime.Object, error) {
			return true, nil, fixture
		})
		err := con.createCRDIfDoesntExist()
		assert.NotNil(t, err, "Custom error should be returned")
	})
}

func TestControllerWaitForEstablishedCRD(t *testing.T) {
	t.Parallel()

	crd := v1alpha1.GameServerCRD()
	crd.Status.Conditions = []v1beta1.CustomResourceDefinitionCondition{{
		Type:   v1beta1.Established,
		Status: v1beta1.ConditionTrue,
	}}

	t.Run("CRD already established", func(t *testing.T) {
		con, cs := newFakeController()
		cs.AddReactor("get", "customresourcedefinitions", func(action k8stesting.Action) (bool, runtime.Object, error) {
			return true, crd, nil
		})

		err := con.waitForEstablishedCRD()
		assert.Nil(t, err)
	})

	t.Run("CRD takes a second to become established", func(t *testing.T) {
		t.Parallel()
		con, cs := newFakeController()

		m := sync.RWMutex{}
		established := false

		cs.AddReactor("get", "customresourcedefinitions", func(action k8stesting.Action) (bool, runtime.Object, error) {
			m.RLock()
			defer m.RUnlock()
			if established {
				return true, crd, nil
			}
			return false, nil, nil
		})

		go func() {
			time.Sleep(3 * time.Second)
			m.Lock()
			defer m.Unlock()
			established = true
		}()

		err := con.waitForEstablishedCRD()
		assert.Nil(t, err)
	})
}

// newFakeController returns a controller, backed by the fake Clientset
func newFakeController() (*Controller, *fake.Clientset) {
	cs := &fake.Clientset{}
	return NewController(cs.ApiextensionsV1beta1().CustomResourceDefinitions()), cs
}
