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

	"fmt"

	"github.com/agonio/agon/pkg/apis/stable"
	"github.com/agonio/agon/pkg/apis/stable/v1alpha1"
	agonfake "github.com/agonio/agon/pkg/client/clientset/versioned/fake"
	"github.com/agonio/agon/pkg/client/informers/externalversions"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	extfake "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/fake"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/informers"
	kubefake "k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
	"k8s.io/client-go/tools/cache"
)

func TestControllerCreateCRDIfDoesntExist(t *testing.T) {
	t.Parallel()

	t.Run("CRD doesn't exist", func(t *testing.T) {
		con, mocks := newFakeController()
		var crd *v1beta1.CustomResourceDefinition
		mocks.extClient.AddReactor("create", "customresourcedefinitions", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
			a := action.(k8stesting.CreateAction)
			crd = a.GetObject().(*v1beta1.CustomResourceDefinition)
			return true, nil, nil
		})

		err := con.createCRDIfDoesntExist()
		assert.Nil(t, err, "CRD Should be created: %v", err)
		assert.Equal(t, v1alpha1.GameServerCRD(), crd)
	})

	t.Run("CRD does exist", func(t *testing.T) {
		con, mocks := newFakeController()
		mocks.extClient.AddReactor("create", "customresourcedefinitions", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
			err = k8serrors.NewAlreadyExists(schema.GroupResource{Group: stable.GroupName, Resource: "gameserver"}, "Foo")
			return true, nil, err
		})
		err := con.createCRDIfDoesntExist()
		assert.Nil(t, err, "CRD Should not be created, but not throw an error: %v", err)
	})

	t.Run("Something bad happens", func(t *testing.T) {
		con, mocks := newFakeController()
		fixture := errors.New("this is a custom error")
		mocks.extClient.AddReactor("create", "customresourcedefinitions", func(action k8stesting.Action) (bool, runtime.Object, error) {
			return true, nil, fixture
		})
		err := con.createCRDIfDoesntExist()
		assert.NotNil(t, err, "Custom error should be returned")
	})
}

func TestControllerWaitForEstablishedCRD(t *testing.T) {
	t.Parallel()
	crd := newEstablishedCRD()
	t.Run("CRD already established", func(t *testing.T) {
		con, mocks := newFakeController()
		mocks.extClient.AddReactor("get", "customresourcedefinitions", func(action k8stesting.Action) (bool, runtime.Object, error) {
			return true, crd, nil
		})

		err := con.waitForEstablishedCRD()
		assert.Nil(t, err)
	})

	t.Run("CRD takes a second to become established", func(t *testing.T) {
		t.Parallel()
		con, mocks := newFakeController()

		m := sync.RWMutex{}
		established := false

		mocks.extClient.AddReactor("get", "customresourcedefinitions", func(action k8stesting.Action) (bool, runtime.Object, error) {
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

func TestSyncGameServer(t *testing.T) {
	t.Parallel()

	t.Run("Creating a new GameServer", func(t *testing.T) {
		c, mocks := newFakeController()
		updateCount := 0
		podCreated := false
		fixture := v1alpha1.GameServer{ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "default"},
			Spec: newSingeContainerSpec()}

		mocks.kubeClient.AddReactor("create", "pods", func(action k8stesting.Action) (bool, runtime.Object, error) {
			ca := action.(k8stesting.CreateAction)
			pod := ca.GetObject().(*corev1.Pod)
			podCreated = true
			assert.Equal(t, fixture.ObjectMeta.Name+"-", pod.ObjectMeta.GenerateName)
			return false, pod, nil
		})
		mocks.agonClient.AddReactor("list", "gameservers", func(action k8stesting.Action) (bool, runtime.Object, error) {
			gameServers := &v1alpha1.GameServerList{Items: []v1alpha1.GameServer{fixture}}
			return true, gameServers, nil
		})
		mocks.agonClient.AddReactor("update", "gameservers", func(action k8stesting.Action) (bool, runtime.Object, error) {
			ua := action.(k8stesting.UpdateAction)
			gs := ua.GetObject().(*v1alpha1.GameServer)
			updateCount++
			expectedState := v1alpha1.State("notastate")
			switch updateCount {
			case 1:
				expectedState = v1alpha1.CreatingState
			case 2:
				expectedState = v1alpha1.StartingState
			}

			assert.Equal(t, expectedState, gs.Status.State)

			return true, gs, nil
		})

		stop := startInformers(c, mocks)
		defer close(stop)

		err := c.syncHandler("default/test")
		assert.Nil(t, err)
		assert.Equal(t, 2, updateCount, "update reactor should twice")
		assert.True(t, podCreated, "pod should be created")
	})

	t.Run("When a GameServer has been deleted, the sync operation should be a noop", func(t *testing.T) {
		c, mocks := newFakeController()
		fixture := &v1alpha1.GameServer{ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "default"},
			Spec:   newSingeContainerSpec(),
			Status: v1alpha1.GameServerStatus{State: v1alpha1.ReadyState}}
		agonWatch := watch.NewFake()
		podAction := false

		mocks.agonClient.AddWatchReactor("gameservers", k8stesting.DefaultWatchReactor(agonWatch, nil))
		mocks.kubeClient.AddReactor("*", "pods", func(action k8stesting.Action) (bool, runtime.Object, error) {
			if action.GetVerb() == "update" || action.GetVerb() == "delete" || action.GetVerb() == "create" || action.GetVerb() == "patch" {
				podAction = true
			}
			return false, nil, nil
		})

		stop := startInformers(c, mocks)
		defer close(stop)

		agonWatch.Delete(fixture)

		err := c.syncGameServer("default/test")
		assert.Nil(t, err, fmt.Sprintf("Shouldn't be an error from syncGameServer: %+v", err))
		assert.False(t, podAction, "Nothing should happen to a Pod")
	})
}

func TestWatchGameServers(t *testing.T) {
	t.Parallel()

	c, mocks := newFakeController()
	fixture := v1alpha1.GameServer{ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "default"}}
	fakeWatch := watch.NewFake()
	mocks.agonClient.AddWatchReactor("gameservers", k8stesting.DefaultWatchReactor(fakeWatch, nil))
	mocks.extClient.AddReactor("get", "customresourcedefinitions", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, newEstablishedCRD(), nil
	})

	received := make(chan bool)

	c.syncHandler = func(name string) error {
		defer close(received)
		assert.Equal(t, "default/test", name)
		return nil
	}

	stop := startInformers(c, mocks)
	defer close(stop)

	go func() {
		err := c.Run(1, stop)
		assert.Nil(t, err, "Run should not error")
	}()

	fakeWatch.Add(&fixture)
	<-received
}

func TestSyncGameServerBlankState(t *testing.T) {

	t.Run("GameServer with a blank initial state", func(t *testing.T) {
		c, mocks := newFakeController()
		fixture := &v1alpha1.GameServer{ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "default"}, Spec: newSingeContainerSpec()}
		updated := false

		mocks.agonClient.AddReactor("update", "gameservers", func(action k8stesting.Action) (bool, runtime.Object, error) {
			updated = true
			ua := action.(k8stesting.UpdateAction)
			gs := ua.GetObject().(*v1alpha1.GameServer)
			assert.Equal(t, fixture.ObjectMeta.Name, gs.ObjectMeta.Name)
			assert.Equal(t, fixture.ObjectMeta.Namespace, gs.ObjectMeta.Namespace)
			return true, gs, nil
		})

		result, err := c.syncGameServerBlankState(fixture)
		assert.Nil(t, err, "sync should not error")
		assert.True(t, updated, "update should occur")
		assert.Equal(t, fixture.ObjectMeta.Name, result.ObjectMeta.Name)
		assert.Equal(t, fixture.ObjectMeta.Namespace, result.ObjectMeta.Namespace)
		assert.Equal(t, v1alpha1.CreatingState, result.Status.State)
	})

	t.Run("Gameserver with unknown state", func(t *testing.T) {
		testWithUnknownState(t, func(c *Controller, fixture *v1alpha1.GameServer) (*v1alpha1.GameServer, error) {
			return c.syncGameServerBlankState(fixture)
		})
	})
}

func TestSyncGameServerCreatingState(t *testing.T) {
	t.Parallel()

	newFixture := func() *v1alpha1.GameServer {
		fixture := &v1alpha1.GameServer{ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "default"},
			Spec: newSingeContainerSpec(), Status: v1alpha1.GameServerStatus{State: v1alpha1.CreatingState}}
		fixture.ApplyDefaults()
		return fixture
	}

	t.Run("Syncing from Created State, with no issues", func(t *testing.T) {
		c, mocks := newFakeController()
		fixture := newFixture()
		podCreated := false
		gsUpdated := false
		mocks.kubeClient.AddReactor("create", "pods", func(action k8stesting.Action) (bool, runtime.Object, error) {
			podCreated = true
			ca := action.(k8stesting.CreateAction)
			pod := ca.GetObject().(*corev1.Pod)

			assert.Equal(t, fixture.ObjectMeta.Name+"-", pod.ObjectMeta.GenerateName)
			assert.Equal(t, fixture.ObjectMeta.Namespace, pod.ObjectMeta.Namespace)
			assert.Equal(t, "gameserver", pod.ObjectMeta.Labels[stable.GroupName+"/role"])
			assert.Equal(t, fixture.ObjectMeta.Name, pod.ObjectMeta.Labels[gameServerPodLabel])
			assert.True(t, metav1.IsControlledBy(pod, fixture))
			assert.Equal(t, fixture.Spec.HostPort, pod.Spec.Containers[0].Ports[0].HostPort)
			assert.Equal(t, fixture.Spec.ContainerPort, pod.Spec.Containers[0].Ports[0].ContainerPort)
			assert.Equal(t, corev1.Protocol("UDP"), pod.Spec.Containers[0].Ports[0].Protocol)

			return true, pod, nil
		})
		mocks.agonClient.AddReactor("update", "gameservers", func(action k8stesting.Action) (bool, runtime.Object, error) {
			gsUpdated = true
			ua := action.(k8stesting.UpdateAction)
			gs := ua.GetObject().(*v1alpha1.GameServer)
			assert.Equal(t, v1alpha1.StartingState, gs.Status.State)
			return true, gs, nil
		})

		err := c.syncGameServerCreatingState(fixture)
		assert.Nil(t, err)
		assert.True(t, podCreated, "Pod should have been created")
		assert.True(t, gsUpdated, "GameServer should have been updated")
	})

	t.Run("Previously started sync, created Pod, but didn't move to Starting", func(t *testing.T) {
		c, mocks := newFakeController()
		fixture := newFixture()
		podCreated := false
		gsUpdated := false
		fakeWatch := watch.NewFake()
		pod := corev1.Pod{
			ObjectMeta: *fixture.ObjectMeta.DeepCopy(),
		}
		pod.ObjectMeta.Labels = map[string]string{gameServerPodLabel: fixture.ObjectMeta.Name}

		mocks.kubeClient.AddWatchReactor("pods", k8stesting.DefaultWatchReactor(fakeWatch, nil))
		mocks.kubeClient.AddReactor("list", "pods", func(action k8stesting.Action) (bool, runtime.Object, error) {
			return true, &corev1.PodList{Items: []corev1.Pod{pod}}, nil
		})
		mocks.kubeClient.AddReactor("create", "pods", func(action k8stesting.Action) (bool, runtime.Object, error) {
			podCreated = true
			return true, nil, nil
		})
		mocks.agonClient.AddReactor("update", "gameservers", func(action k8stesting.Action) (bool, runtime.Object, error) {
			gsUpdated = true
			ua := action.(k8stesting.UpdateAction)
			gs := ua.GetObject().(*v1alpha1.GameServer)
			assert.Equal(t, v1alpha1.StartingState, gs.Status.State)
			return true, gs, nil
		})

		stop := startInformers(c, mocks)
		defer close(stop)

		fakeWatch.Add(&pod)

		err := c.syncGameServerCreatingState(fixture)
		assert.Nil(t, err)
		assert.False(t, podCreated, "Pod should not have been created")
		assert.True(t, gsUpdated, "GameServer should have been updated")
	})

	t.Run("Gameserver with unknown state", func(t *testing.T) {
		testWithUnknownState(t, func(c *Controller, fixture *v1alpha1.GameServer) (*v1alpha1.GameServer, error) {
			return fixture, c.syncGameServerCreatingState(fixture)
		})
	})
}

// testWithUnknownState runs a test with a state that doesn't exist, to ensure a handler
// doesn't do process anything beyond the state it is meant to handle.
func testWithUnknownState(t *testing.T, f func(*Controller, *v1alpha1.GameServer) (*v1alpha1.GameServer, error)) {
	c, mocks := newFakeController()
	fixture := &v1alpha1.GameServer{ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "default"},
		Spec: newSingeContainerSpec(), Status: v1alpha1.GameServerStatus{State: "ThisStateDoesNotExist"}}
	fixture.ApplyDefaults()
	updated := false
	mocks.agonClient.AddReactor("update", "gameservers", func(action k8stesting.Action) (bool, runtime.Object, error) {
		updated = true
		return true, nil, nil
	})

	result, err := f(c, fixture)
	assert.Nil(t, err, "sync should not error")
	assert.False(t, updated, "update should occur")
	assert.Equal(t, fixture, result)
}

// holder for all my fakes and mocks
type mocks struct {
	kubeClient             *kubefake.Clientset
	kubeInformationFactory informers.SharedInformerFactory
	extClient              *extfake.Clientset
	agonClient           *agonfake.Clientset
	agonInformerFactory  externalversions.SharedInformerFactory
}

// newFakeController returns a controller, backed by the fake Clientset
func newFakeController() (*Controller, mocks) {
	kubeClient := &kubefake.Clientset{}
	kubeInformationFactory := informers.NewSharedInformerFactory(kubeClient, 30*time.Second)
	extClient := &extfake.Clientset{}
	agonClient := &agonfake.Clientset{}
	agonInformerFactory := externalversions.NewSharedInformerFactory(agonClient, 30*time.Second)

	return NewController(kubeClient, kubeInformationFactory, extClient, agonClient, agonInformerFactory),
		mocks{
			kubeClient:             kubeClient,
			kubeInformationFactory: kubeInformationFactory,
			extClient:              extClient,
			agonClient:           agonClient,
			agonInformerFactory:  agonInformerFactory}
}

func newSingeContainerSpec() v1alpha1.GameServerSpec {
	return v1alpha1.GameServerSpec{
		ContainerPort: 7777,
		HostPort:      9999,
		PortPolicy:    v1alpha1.StaticPortPolicy,
		Template: corev1.PodTemplateSpec{
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{{Name: "container", Image: "container/image"}},
			},
		},
	}
}

func newEstablishedCRD() *v1beta1.CustomResourceDefinition {
	crd := v1alpha1.GameServerCRD()
	crd.Status.Conditions = []v1beta1.CustomResourceDefinitionCondition{{
		Type:   v1beta1.Established,
		Status: v1beta1.ConditionTrue,
	}}

	return crd
}

func startInformers(c *Controller, mocks mocks) chan struct{} {
	stop := make(chan struct{})
	mocks.kubeInformationFactory.Start(stop)
	mocks.agonInformerFactory.Start(stop)

	logrus.Info("Wait for cache sync")
	if !cache.WaitForCacheSync(stop, c.gameServerSynced) {
		panic("Cache never synced")
	}

	return stop
}
