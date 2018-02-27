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

package gameservers

import (
	"context"
	"time"

	"io"
	"io/ioutil"
	"net/http"
	"testing"

	"agones.dev/agones/pkg/apis/stable/v1alpha1"
	agonesfake "agones.dev/agones/pkg/client/clientset/versioned/fake"
	"agones.dev/agones/pkg/client/informers/externalversions"
	"agones.dev/agones/pkg/sdk"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/metadata"
	corev1 "k8s.io/api/core/v1"
	extfake "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/fake"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/informers"
	kubefake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
)

// holder for all my fakes and mocks
type mocks struct {
	kubeClient             *kubefake.Clientset
	kubeInformationFactory informers.SharedInformerFactory
	extClient              *extfake.Clientset
	agonesClient           *agonesfake.Clientset
	agonesInformerFactory  externalversions.SharedInformerFactory
	fakeRecorder           *record.FakeRecorder
}

func newMocks() mocks {
	kubeClient := &kubefake.Clientset{}
	kubeInformationFactory := informers.NewSharedInformerFactory(kubeClient, 30*time.Second)
	extClient := &extfake.Clientset{}
	agonesClient := &agonesfake.Clientset{}
	agonesInformerFactory := externalversions.NewSharedInformerFactory(agonesClient, 30*time.Second)
	m := mocks{
		kubeClient:             kubeClient,
		kubeInformationFactory: kubeInformationFactory,
		extClient:              extClient,
		agonesClient:           agonesClient,
		agonesInformerFactory:  agonesInformerFactory,
		fakeRecorder:           record.NewFakeRecorder(10),
	}
	return m
}

func startInformers(mocks mocks, sync ...cache.InformerSynced) (<-chan struct{}, context.CancelFunc) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	stop := ctx.Done()

	mocks.kubeInformationFactory.Start(stop)
	mocks.agonesInformerFactory.Start(stop)

	logrus.Info("Wait for cache sync")
	if !cache.WaitForCacheSync(stop, sync...) {
		panic("Cache never synced")
	}

	return stop, cancel
}

func newSingleContainerSpec() v1alpha1.GameServerSpec {
	return v1alpha1.GameServerSpec{
		ContainerPort: 7777,
		HostPort:      9999,
		PortPolicy:    v1alpha1.Static,
		Template: corev1.PodTemplateSpec{
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{{Name: "container", Image: "container/image"}},
			},
		},
	}
}

// mockStream is the mock of the SDK_HealthServer for streaming
type mockStream struct {
	msgs chan *sdk.Empty
}

func newMockStream() *mockStream {
	return &mockStream{msgs: make(chan *sdk.Empty)}
}

func (m *mockStream) SendAndClose(*sdk.Empty) error {
	return nil
}

func (m *mockStream) Recv() (*sdk.Empty, error) {
	empty, ok := <-m.msgs
	if ok {
		return empty, nil
	}
	return empty, io.EOF
}

func (m *mockStream) SetHeader(metadata.MD) error {
	panic("implement me")
}

func (m *mockStream) SendHeader(metadata.MD) error {
	panic("implement me")
}

func (m *mockStream) SetTrailer(metadata.MD) {
	panic("implement me")
}

func (m *mockStream) Context() context.Context {
	panic("implement me")
}

func (m *mockStream) SendMsg(msg interface{}) error {
	panic("implement me")
}

func (m *mockStream) RecvMsg(msg interface{}) error {
	panic("implement me")
}

func testHTTPHealth(t *testing.T, url string, expectedResponse string, expectedStatus int) {
	// do a poll, because this code could run before the health check becomes live
	err := wait.PollImmediate(time.Second, 20*time.Second, func() (done bool, err error) {
		resp, err := http.Get(url)
		if err != nil {
			logrus.WithError(err).Error("Error connecting to %v", url)
			return false, nil
		}

		assert.NotNil(t, resp)
		if resp != nil {
			defer resp.Body.Close()
			body, err := ioutil.ReadAll(resp.Body)
			assert.Nil(t, err, "(%s) read response error should be nil: %v", url, err)
			assert.Equal(t, expectedStatus, resp.StatusCode, "url: %s", url)
			assert.Equal(t, []byte(expectedResponse), body, "(%s) response body should be '%s'", url, expectedResponse)
		}

		return true, nil
	})
	assert.Nil(t, err, "Timeout on %s health check, %v", url, err)
}
