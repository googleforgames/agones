// Copyright 2018 Google LLC All Rights Reserved.
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
	"testing"

	"github.com/stretchr/testify/assert"
	admregv1 "k8s.io/api/admissionregistration/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func TestFleetAutoscalerValidateUpdate(t *testing.T) {
	t.Parallel()

	t.Run("bad buffer size", func(t *testing.T) {
		fas := defaultFixture()
		fas.Spec.Policy.Buffer.BufferSize = intstr.FromInt(0)
		causes := fas.Validate(nil)

		assert.Len(t, causes, 1)
		assert.Equal(t, "bufferSize", causes[0].Field)
	})

	t.Run("bad min replicas", func(t *testing.T) {

		fas := defaultFixture()
		fas.Spec.Policy.Buffer.MinReplicas = 2

		causes := fas.Validate(nil)

		assert.Len(t, causes, 1)
		assert.Equal(t, "minReplicas", causes[0].Field)
	})

	t.Run("bad max replicas", func(t *testing.T) {
		fas := defaultFixture()
		fas.Spec.Policy.Buffer.MaxReplicas = 2
		causes := fas.Validate(nil)

		assert.Len(t, causes, 1)
		assert.Equal(t, "maxReplicas", causes[0].Field)
	})

	t.Run("minReplicas > maxReplicas", func(t *testing.T) {
		fas := defaultFixture()
		fas.Spec.Policy.Buffer.MinReplicas = 20
		causes := fas.Validate(nil)

		assert.Len(t, causes, 1)
		assert.Equal(t, "minReplicas", causes[0].Field)
	})

	t.Run("bufferSize good percent", func(t *testing.T) {
		fas := defaultFixture()
		fas.Spec.Policy.Buffer.MinReplicas = 1
		fas.Spec.Policy.Buffer.BufferSize = intstr.FromString("20%")
		causes := fas.Validate(nil)

		assert.Len(t, causes, 0)
	})

	t.Run("bufferSize bad percent", func(t *testing.T) {
		fas := defaultFixture()
		fas.Spec.Policy.Buffer.MinReplicas = 1

		fasCopy := fas.DeepCopy()
		fasCopy.Spec.Policy.Buffer.BufferSize = intstr.FromString("120%")
		causes := fasCopy.Validate(nil)
		assert.Len(t, causes, 1)
		assert.Equal(t, "bufferSize", causes[0].Field)

		fasCopy = fas.DeepCopy()
		fasCopy.Spec.Policy.Buffer.BufferSize = intstr.FromString("0%")
		causes = fasCopy.Validate(nil)
		assert.Len(t, causes, 1)
		assert.Equal(t, "bufferSize", causes[0].Field)

		fasCopy = fas.DeepCopy()
		fasCopy.Spec.Policy.Buffer.BufferSize = intstr.FromString("-10%")
		causes = fasCopy.Validate(nil)
		assert.Len(t, causes, 1)
		assert.Equal(t, "bufferSize", causes[0].Field)
		fasCopy = fas.DeepCopy()

		fasCopy.Spec.Policy.Buffer.BufferSize = intstr.FromString("notgood")
		causes = fasCopy.Validate(nil)
		assert.Len(t, causes, 1)
		assert.Equal(t, "bufferSize", causes[0].Field)
	})

	t.Run("bad min replicas with percentage value of bufferSize", func(t *testing.T) {
		fas := defaultFixture()
		fas.Spec.Policy.Buffer.BufferSize = intstr.FromString("10%")
		fas.Spec.Policy.Buffer.MinReplicas = 0

		causes := fas.Validate(nil)

		assert.Len(t, causes, 1)
		assert.Equal(t, "minReplicas", causes[0].Field)
	})

	t.Run("bad sync interval seconds", func(t *testing.T) {
		fas := defaultFixture()
		fas.Spec.Sync.FixedInterval.Seconds = 0

		causes := fas.Validate(nil)

		assert.Len(t, causes, 1)
		assert.Equal(t, "seconds", causes[0].Field)
	})
}
func TestFleetAutoscalerWebhookValidateUpdate(t *testing.T) {
	t.Parallel()

	t.Run("good service value", func(t *testing.T) {
		fas := webhookFixture()
		causes := fas.Validate(nil)

		assert.Len(t, causes, 0)
	})

	t.Run("good url value", func(t *testing.T) {
		fas := webhookFixture()
		url := "http://good.example.com"
		fas.Spec.Policy.Webhook.URL = &url
		fas.Spec.Policy.Webhook.Service = nil
		causes := fas.Validate(nil)

		assert.Len(t, causes, 0)
	})

	t.Run("bad URL and service value", func(t *testing.T) {
		fas := webhookFixture()
		fas.Spec.Policy.Webhook.URL = nil
		fas.Spec.Policy.Webhook.Service = nil
		causes := fas.Validate(nil)

		assert.Len(t, causes, 1)
		assert.Equal(t, "url", causes[0].Field)
	})

	t.Run("both URL and service value are used - fail", func(t *testing.T) {

		fas := webhookFixture()
		url := "123"
		fas.Spec.Policy.Webhook.URL = &url

		causes := fas.Validate(nil)

		assert.Len(t, causes, 1)
		assert.Equal(t, "url", causes[0].Field)
	})

	goodCaBundle := "\n-----BEGIN CERTIFICATE-----\nMIIDXjCCAkYCCQDvT9MAXwnuqDANBgkqhkiG9w0BAQsFADBxMQswCQYDVQQGEwJV\nUzETMBEGA1UECAwKQ2FsaWZvcm5pYTEWMBQGA1UEBwwNTW91bnRhaW4gVmlldzEP\nMA0GA1UECgwGQWdvbmVzMQ8wDQYDVQQLDAZBZ29uZXMxEzARBgNVBAMMCmFnb25l\ncy5kZXYwHhcNMTkwMTAzMTEwNTA0WhcNMjExMDIzMTEwNTA0WjBxMQswCQYDVQQG\nEwJVUzETMBEGA1UECAwKQ2FsaWZvcm5pYTEWMBQGA1UEBwwNTW91bnRhaW4gVmll\ndzEPMA0GA1UECgwGQWdvbmVzMQ8wDQYDVQQLDAZBZ29uZXMxEzARBgNVBAMMCmFn\nb25lcy5kZXYwggEiMA0GCSqGSIb3DQEBAQUAA4IBDwAwggEKAoIBAQDFwohJp3xK\n4iORkJXNO2KEkdrVYK7xpXTrPvZqLzoyMBOXi9b+lOKILUPaKtZ33GIwola31bHp\ni7V97vh3irIQVpap6uncesRTX0qk5Y70f7T6lByMKDsxi5ddea3ztAftH+PMYSLn\nE7H9276R1lvX8HZ0E2T4ea63PcVcTldw74ueEQr7HFMVucO+hHjgNJXDsWFUNppv\nxqWOvlIEDRdQzB1UYd13orqX0t514Ikp5Y3oNigXftDH+lZPlrWGsknMIDWr4DKP\n7NB1BZMfLFu/HXTGI9dK5Zc4T4GG4DBZqlgDPdzAXSBUT9cRQvbLrZ5+tUjOZK5E\nzEEIqyUo1+QdAgMBAAEwDQYJKoZIhvcNAQELBQADggEBABgtnWaWIDFCbKvhD8cF\nd5fvARFJcRl4dcChoqANeUXK4iNiCEPiDJb4xDGrLSVOeQ2IMbghqCwJfH93aqTr\n9kFQPvYbCt10TPQpmmh2//QjWGc7lxniFWR8pAVYdCGHqIAMvW2V2177quHsqc2I\nNTXyEUus0SDHLK8swLQxoCVw4fSq+kjMFW/3zOvMfh13rZH7Lo0gQyAUcuHM5U7g\nbhCZ3yVkDYpPxVv2vL0eyWUdLrQjYXyY7MWHPXvDozi3CtuBZlp6ulgeubi6jhHE\nIzuOM3qiLMJ/KG8MlIgGCwSX/x0vfO0/LtkZM7P1+yptSr/Se5QiZMtmpxC+DDWJ\n2xw=\n-----END CERTIFICATE-----"
	t.Run("good url and CABundle value", func(t *testing.T) {
		fas := webhookFixture()
		url := "https://good.example.com"
		fas.Spec.Policy.Webhook.URL = &url
		fas.Spec.Policy.Webhook.Service = nil
		fas.Spec.Policy.Webhook.CABundle = []byte(goodCaBundle)

		causes := fas.Validate(nil)
		assert.Len(t, causes, 0)
	})

	t.Run("https url and invalid CABundle value", func(t *testing.T) {
		fas := webhookFixture()
		url := "https://bad.example.com"
		fas.Spec.Policy.Webhook.URL = &url
		fas.Spec.Policy.Webhook.Service = nil
		fas.Spec.Policy.Webhook.CABundle = []byte("SomeInvalidCABundle")

		causes := fas.Validate(nil)
		assert.Len(t, causes, 1)
		assert.Equal(t, "caBundle", causes[0].Field)
	})

	t.Run("https url and missing CABundle value", func(t *testing.T) {
		fas := webhookFixture()
		url := "https://bad.example.com"
		fas.Spec.Policy.Webhook.URL = &url
		fas.Spec.Policy.Webhook.Service = nil
		fas.Spec.Policy.Webhook.CABundle = nil

		causes := fas.Validate(nil)
		assert.Len(t, causes, 1)
		assert.Equal(t, "caBundle", causes[0].Field)
	})

	t.Run("bad url value", func(t *testing.T) {
		fas := webhookFixture()
		url := "http:/bad.example.com%"
		fas.Spec.Policy.Webhook.URL = &url
		fas.Spec.Policy.Webhook.Service = nil
		fas.Spec.Policy.Webhook.CABundle = []byte(goodCaBundle)

		causes := fas.Validate(nil)
		assert.Len(t, causes, 1)
		assert.Equal(t, "url", causes[0].Field)
	})

}

func defaultFixture() *FleetAutoscaler {
	return customFixture(BufferPolicyType)
}

func webhookFixture() *FleetAutoscaler {
	return customFixture(WebhookPolicyType)
}

func customFixture(t FleetAutoscalerPolicyType) *FleetAutoscaler {
	res := &FleetAutoscaler{
		ObjectMeta: metav1.ObjectMeta{Name: "test"},
		Spec: FleetAutoscalerSpec{
			FleetName: "testing",
			Policy: FleetAutoscalerPolicy{
				Type: BufferPolicyType,
				Buffer: &BufferPolicy{
					BufferSize:  intstr.FromInt(5),
					MaxReplicas: 10,
				},
			},
			Sync: FleetAutoscalerSync{
				Type: FixedIntervalSyncType,
				FixedInterval: &FixedIntervalSync{
					Seconds: 30,
				},
			},
		},
	}
	switch t {
	case BufferPolicyType:
	case WebhookPolicyType:
		res.Spec.Policy.Type = WebhookPolicyType
		res.Spec.Policy.Buffer = nil
		url := "/scale"
		res.Spec.Policy.Webhook = &WebhookPolicy{
			Service: &admregv1.ServiceReference{
				Name:      "service1",
				Namespace: "default",
				Path:      &url,
			},
		}
	}
	return res
}
