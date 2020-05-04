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
	"context"
	"io/ioutil"
	"net/http"
	"os"
	"testing"

	pb "agones.dev/agones/pkg/allocation/go/v1alpha1"
	allocationv1 "agones.dev/agones/pkg/apis/allocation/v1"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	corev1 "k8s.io/api/core/v1"
	k8serror "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
)

func TestAllocateHandler(t *testing.T) {
	t.Parallel()

	h := serviceHandler{
		allocationCallback: func(gsa *allocationv1.GameServerAllocation) (k8sruntime.Object, error) {
			return &allocationv1.GameServerAllocation{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "default",
				},
				Status: allocationv1.GameServerAllocationStatus{
					State: allocationv1.GameServerAllocationContention,
				},
			}, nil
		},
	}

	request := &pb.AllocationRequest{
		Namespace: "ns",
		MultiClusterSetting: &pb.MultiClusterSetting{
			Enabled: true,
		},
	}

	response, err := h.Allocate(context.Background(), request)
	if !assert.NoError(t, err) {
		return
	}
	assert.Equal(t, pb.AllocationResponse_Contention, response.State)
}

func TestAllocateHandlerReturnsError(t *testing.T) {
	t.Parallel()

	h := serviceHandler{
		allocationCallback: func(gsa *allocationv1.GameServerAllocation) (k8sruntime.Object, error) {
			return nil, k8serror.NewBadRequest("error")
		},
	}

	request := &pb.AllocationRequest{}
	_, err := h.Allocate(context.Background(), request)
	if assert.Error(t, err) {
		assert.Equal(t, "error", err.Error())
	}
}

func TestHandlingStatus(t *testing.T) {
	t.Parallel()

	errorMessage := "GameServerAllocation is invalid"
	h := serviceHandler{
		allocationCallback: func(gsa *allocationv1.GameServerAllocation) (k8sruntime.Object, error) {
			return &metav1.Status{
				Status:  metav1.StatusFailure,
				Message: errorMessage,
				Reason:  metav1.StatusReasonInvalid,
				Details: &metav1.StatusDetails{
					Kind:  "GameServerAllocation",
					Group: allocationv1.SchemeGroupVersion.Group,
				},
				Code: http.StatusUnprocessableEntity,
			}, nil
		},
	}

	request := &pb.AllocationRequest{}
	_, err := h.Allocate(context.Background(), request)
	if !assert.Error(t, err, "expecting failure") {
		return
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Errorf("expecting status error: %v", err)
	}
	assert.Equal(t, 422, int(st.Code()))
	assert.Contains(t, st.Message(), errorMessage)
}

func TestBadReturnType(t *testing.T) {
	t.Parallel()

	h := serviceHandler{
		allocationCallback: func(gsa *allocationv1.GameServerAllocation) (k8sruntime.Object, error) {
			return &corev1.Secret{}, nil
		},
	}

	request := &pb.AllocationRequest{}
	_, err := h.Allocate(context.Background(), request)
	if !assert.Error(t, err, "expecting failure") {
		return
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Errorf("expecting status error: %v", err)
	}
	assert.Equal(t, codes.Internal, st.Code())
	assert.Contains(t, st.Message(), "internal server error")
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
