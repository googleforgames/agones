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
package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"agones.dev/agones/pkg/apis/allocation/v1alpha1"
	agonesfake "agones.dev/agones/pkg/client/clientset/versioned/fake"
	"github.com/stretchr/testify/assert"
	k8serror "k8s.io/apimachinery/pkg/api/errors"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	k8stesting "k8s.io/client-go/testing"
)

func TestAllocateHandler(t *testing.T) {
	t.Parallel()

	fakeAgones := &agonesfake.Clientset{}
	h := httpHandler{
		agonesClient: fakeAgones,
		namespace:    "default",
	}

	fakeAgones.AddReactor("create", "gameserverallocations", func(action k8stesting.Action) (bool, k8sruntime.Object, error) {
		return true, &v1alpha1.GameServerAllocation{
			Status: v1alpha1.GameServerAllocationStatus{
				State: v1alpha1.GameServerAllocationContention,
			},
		}, nil
	})

	gsa := &v1alpha1.GameServerAllocation{}
	body, _ := json.Marshal(gsa)
	buf := bytes.NewBuffer(body)
	req, err := http.NewRequest(http.MethodPost, "/", buf)
	if !assert.Nil(t, err) {
		return
	}

	rec := httptest.NewRecorder()
	h.allocateHandler(rec, req)

	ret := &v1alpha1.GameServerAllocation{}
	assert.Equal(t, rec.Code, 200)
	assert.Equal(t, "application/json", rec.Header()["Content-Type"][0])
	err = json.Unmarshal(rec.Body.Bytes(), ret)
	assert.NoError(t, err)
	assert.Equal(t, v1alpha1.GameServerAllocationContention, ret.Status.State)
}

func TestAllocateHandlerReturnsError(t *testing.T) {
	t.Parallel()

	fakeAgones := &agonesfake.Clientset{}
	h := httpHandler{
		agonesClient: fakeAgones,
		namespace:    "default",
	}

	fakeAgones.AddReactor("create", "gameserverallocations", func(action k8stesting.Action) (bool, k8sruntime.Object, error) {
		return true, nil, k8serror.NewBadRequest("error")
	})

	gsa := &v1alpha1.GameServerAllocation{}
	body, _ := json.Marshal(gsa)
	buf := bytes.NewBuffer(body)
	req, err := http.NewRequest(http.MethodPost, "/", buf)
	if !assert.Nil(t, err) {
		return
	}

	rec := httptest.NewRecorder()
	h.allocateHandler(rec, req)
	assert.Equal(t, rec.Code, 400)
	assert.Contains(t, rec.Body.String(), "error")
}

func TestGettingCaCert(t *testing.T) {
	t.Parallel()

	file, err := ioutil.TempFile(".", "*.crt")
	if assert.Nil(t, err) {
		defer os.Remove(file.Name()) // nolint: errcheck
		_, err = file.WriteString(clientCert)
		if assert.Nil(t, err) {
			certPool, err := getCACertPool("./")
			if assert.Nil(t, err) {
				assert.Len(t, certPool.Subjects(), 1)
			}
		}
	}
}

var clientCert = `-----BEGIN CERTIFICATE-----
MIIDuzCCAqOgAwIBAgIUduDWtqpUsp3rZhCEfUrzI05laVIwDQYJKoZIhvcNAQEL
BQAwbTELMAkGA1UEBhMCR0IxDzANBgNVBAgMBkxvbmRvbjEPMA0GA1UEBwwGTG9u
ZG9uMRgwFgYDVQQKDA9HbG9iYWwgU2VjdXJpdHkxFjAUBgNVBAsMDUlUIERlcGFy
dG1lbnQxCjAIBgNVBAMMASowHhcNMTkwNTAyMjIzMDQ3WhcNMjkwNDI5MjIzMDQ3
WjBtMQswCQYDVQQGEwJHQjEPMA0GA1UECAwGTG9uZG9uMQ8wDQYDVQQHDAZMb25k
b24xGDAWBgNVBAoMD0dsb2JhbCBTZWN1cml0eTEWMBQGA1UECwwNSVQgRGVwYXJ0
bWVudDEKMAgGA1UEAwwBKjCCASIwDQYJKoZIhvcNAQEBBQADggEPADCCAQoCggEB
AKGDasjadVwe0bXUEQfZCkMEAkzn0qTud3RYytympmaS0c01SWFNZwPRO0rpdIOZ
fyXVXVOAhgmgCR6QuXySmyQIoYl/D6tVhc5r9FyWPIBtzQKCJTX0mZOZwMn22qvo
bfnDnVsZ1Ny3RLZIF3um3xovvePXyg1z7D/NvCogNuYpyUUEITPZX6ss5ods/U78
BxLhKrT8iyu61ZC+ZegbHQqFRngbeb348gE1JwKTslDfe4oH7tZ+bNDZxnGcvh9j
eyagpM0zys4gFfQf/vfD2aEsUJ+GesUQC6uGVoGnTFshFhBsAK6vpIQ4ZQujaJ0r
NKgJ/ccBJFiJXMCR44yWFY0CAwEAAaNTMFEwHQYDVR0OBBYEFEe1gDd8JpzgnvOo
1AEloAXxmxHCMB8GA1UdIwQYMBaAFEe1gDd8JpzgnvOo1AEloAXxmxHCMA8GA1Ud
EwEB/wQFMAMBAf8wDQYJKoZIhvcNAQELBQADggEBAI5GyuakVgunerCGCSN7Ghsr
ys9vJytbyT+BLmxNBPSXWQwcm3g9yCDdgf0Y3q3Eef7IEZu4I428318iLzhfKln1
ua4fxvmTFKJ65lQKNkc6Y4e3w1t+C2HOl6fOIVT231qsCoM5SAwQQpqAzEUj6kZl
x+3avw9KSlXqR/mCAkePyoKvprxeb6RVDdq92Ug0qzoAHLpvIkuHdlF0dNp6/kO0
1pVL0BqW+6UTimSSvH8F/cMeYKbkhpE1u2c/NtNwsR2jN4M9kl3KHqkynk67PfZv
pwlCqZx4M8FpdfCbOZeRLzClUBdD5qzev0L3RNUx7UJzEIN+4LCBv37DIojNOyA=
-----END CERTIFICATE-----
`
