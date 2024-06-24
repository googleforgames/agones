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

	"agones.dev/agones/pkg/util/runtime"
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
		causes := fas.Validate()

		assert.Len(t, causes, 1)
		assert.Equal(t, "spec.policy.buffer.bufferSize", causes[0].Field)
	})

	t.Run("bad min replicas", func(t *testing.T) {
		fas := defaultFixture()
		fas.Spec.Policy.Buffer.MinReplicas = 2

		causes := fas.Validate()

		assert.Len(t, causes, 1)
		assert.Equal(t, "spec.policy.buffer.minReplicas", causes[0].Field)
	})

	t.Run("bad max replicas", func(t *testing.T) {
		fas := defaultFixture()
		fas.Spec.Policy.Buffer.MaxReplicas = 2
		causes := fas.Validate()

		assert.Len(t, causes, 1)
		assert.Equal(t, "spec.policy.buffer.maxReplicas", causes[0].Field)
	})

	t.Run("minReplicas > maxReplicas", func(t *testing.T) {
		fas := defaultFixture()
		fas.Spec.Policy.Buffer.MinReplicas = 20
		causes := fas.Validate()

		assert.Len(t, causes, 1)
		assert.Equal(t, "spec.policy.buffer.minReplicas", causes[0].Field)
	})

	t.Run("bufferSize good percent", func(t *testing.T) {
		fas := defaultFixture()
		fas.Spec.Policy.Buffer.MinReplicas = 1
		fas.Spec.Policy.Buffer.BufferSize = intstr.FromString("20%")
		causes := fas.Validate()

		assert.Len(t, causes, 0)
	})

	t.Run("bufferSize bad percent", func(t *testing.T) {
		fas := defaultFixture()
		fas.Spec.Policy.Buffer.MinReplicas = 1

		fasCopy := fas.DeepCopy()
		fasCopy.Spec.Policy.Buffer.BufferSize = intstr.FromString("120%")
		causes := fasCopy.Validate()
		assert.Len(t, causes, 1)
		assert.Equal(t, "spec.policy.buffer.bufferSize", causes[0].Field)

		fasCopy = fas.DeepCopy()
		fasCopy.Spec.Policy.Buffer.BufferSize = intstr.FromString("0%")
		causes = fasCopy.Validate()
		assert.Len(t, causes, 1)
		assert.Equal(t, "spec.policy.buffer.bufferSize", causes[0].Field)

		fasCopy = fas.DeepCopy()
		fasCopy.Spec.Policy.Buffer.BufferSize = intstr.FromString("-10%")
		causes = fasCopy.Validate()
		assert.Len(t, causes, 1)
		assert.Equal(t, "spec.policy.buffer.bufferSize", causes[0].Field)
		fasCopy = fas.DeepCopy()

		fasCopy.Spec.Policy.Buffer.BufferSize = intstr.FromString("notgood")
		causes = fasCopy.Validate()
		assert.Len(t, causes, 1)
		assert.Equal(t, "spec.policy.buffer.bufferSize", causes[0].Field)
	})

	t.Run("bad min replicas with percentage value of bufferSize", func(t *testing.T) {
		fas := defaultFixture()
		fas.Spec.Policy.Buffer.BufferSize = intstr.FromString("10%")
		fas.Spec.Policy.Buffer.MinReplicas = 0

		causes := fas.Validate()

		assert.Len(t, causes, 1)
		assert.Equal(t, "spec.policy.buffer.minReplicas", causes[0].Field)
	})

	t.Run("bad sync interval seconds", func(t *testing.T) {

		fas := defaultFixture()
		fas.Spec.Sync.FixedInterval.Seconds = 0

		causes := fas.Validate()

		assert.Len(t, causes, 1)
		assert.Equal(t, "spec.sync.fixedInterval.seconds", causes[0].Field)
	})
}
func TestFleetAutoscalerWebhookValidateUpdate(t *testing.T) {
	t.Parallel()

	t.Run("good service value", func(t *testing.T) {
		fas := webhookFixture()
		causes := fas.Validate()

		assert.Len(t, causes, 0)
	})

	t.Run("good url value", func(t *testing.T) {
		fas := webhookFixture()
		url := "http://good.example.com"
		fas.Spec.Policy.Webhook.URL = &url
		fas.Spec.Policy.Webhook.Service = nil
		causes := fas.Validate()

		assert.Len(t, causes, 0)
	})

	t.Run("bad URL and service value", func(t *testing.T) {
		fas := webhookFixture()
		fas.Spec.Policy.Webhook.URL = nil
		fas.Spec.Policy.Webhook.Service = nil
		causes := fas.Validate()

		assert.Len(t, causes, 1)
		assert.Equal(t, "spec.policy.webhook", causes[0].Field)
	})

	t.Run("both URL and service value are used - fail", func(t *testing.T) {

		fas := webhookFixture()
		url := "123"
		fas.Spec.Policy.Webhook.URL = &url

		causes := fas.Validate()

		assert.Len(t, causes, 1)
		assert.Equal(t, "spec.policy.webhook.url", causes[0].Field)
	})

	goodCaBundle := "\n-----BEGIN CERTIFICATE-----\nMIIDXjCCAkYCCQDvT9MAXwnuqDANBgkqhkiG9w0BAQsFADBxMQswCQYDVQQGEwJV\nUzETMBEGA1UECAwKQ2FsaWZvcm5pYTEWMBQGA1UEBwwNTW91bnRhaW4gVmlldzEP\nMA0GA1UECgwGQWdvbmVzMQ8wDQYDVQQLDAZBZ29uZXMxEzARBgNVBAMMCmFnb25l\ncy5kZXYwHhcNMTkwMTAzMTEwNTA0WhcNMjExMDIzMTEwNTA0WjBxMQswCQYDVQQG\nEwJVUzETMBEGA1UECAwKQ2FsaWZvcm5pYTEWMBQGA1UEBwwNTW91bnRhaW4gVmll\ndzEPMA0GA1UECgwGQWdvbmVzMQ8wDQYDVQQLDAZBZ29uZXMxEzARBgNVBAMMCmFn\nb25lcy5kZXYwggEiMA0GCSqGSIb3DQEBAQUAA4IBDwAwggEKAoIBAQDFwohJp3xK\n4iORkJXNO2KEkdrVYK7xpXTrPvZqLzoyMBOXi9b+lOKILUPaKtZ33GIwola31bHp\ni7V97vh3irIQVpap6uncesRTX0qk5Y70f7T6lByMKDsxi5ddea3ztAftH+PMYSLn\nE7H9276R1lvX8HZ0E2T4ea63PcVcTldw74ueEQr7HFMVucO+hHjgNJXDsWFUNppv\nxqWOvlIEDRdQzB1UYd13orqX0t514Ikp5Y3oNigXftDH+lZPlrWGsknMIDWr4DKP\n7NB1BZMfLFu/HXTGI9dK5Zc4T4GG4DBZqlgDPdzAXSBUT9cRQvbLrZ5+tUjOZK5E\nzEEIqyUo1+QdAgMBAAEwDQYJKoZIhvcNAQELBQADggEBABgtnWaWIDFCbKvhD8cF\nd5fvARFJcRl4dcChoqANeUXK4iNiCEPiDJb4xDGrLSVOeQ2IMbghqCwJfH93aqTr\n9kFQPvYbCt10TPQpmmh2//QjWGc7lxniFWR8pAVYdCGHqIAMvW2V2177quHsqc2I\nNTXyEUus0SDHLK8swLQxoCVw4fSq+kjMFW/3zOvMfh13rZH7Lo0gQyAUcuHM5U7g\nbhCZ3yVkDYpPxVv2vL0eyWUdLrQjYXyY7MWHPXvDozi3CtuBZlp6ulgeubi6jhHE\nIzuOM3qiLMJ/KG8MlIgGCwSX/x0vfO0/LtkZM7P1+yptSr/Se5QiZMtmpxC+DDWJ\n2xw=\n-----END CERTIFICATE-----"
	t.Run("good url and CABundle value", func(t *testing.T) {
		fas := webhookFixture()
		url := "https://good.example.com"
		fas.Spec.Policy.Webhook.URL = &url
		fas.Spec.Policy.Webhook.Service = nil
		fas.Spec.Policy.Webhook.CABundle = []byte(goodCaBundle)

		causes := fas.Validate()
		assert.Len(t, causes, 0)
	})

	t.Run("https url and invalid CABundle value", func(t *testing.T) {
		fas := webhookFixture()
		url := "https://bad.example.com"
		fas.Spec.Policy.Webhook.URL = &url
		fas.Spec.Policy.Webhook.Service = nil
		fas.Spec.Policy.Webhook.CABundle = []byte("SomeInvalidCABundle")

		causes := fas.Validate()
		assert.Len(t, causes, 1)
		assert.Equal(t, "spec.policy.webhook.caBundle", causes[0].Field)
	})

	t.Run("https url and missing CABundle value", func(t *testing.T) {
		fas := webhookFixture()
		url := "https://bad.example.com"
		fas.Spec.Policy.Webhook.URL = &url
		fas.Spec.Policy.Webhook.Service = nil
		fas.Spec.Policy.Webhook.CABundle = nil

		causes := fas.Validate()
		assert.Len(t, causes, 1)
		assert.Equal(t, "spec.policy.webhook.caBundle", causes[0].Field)
	})

	t.Run("bad url value", func(t *testing.T) {
		fas := webhookFixture()
		url := "http:/bad.example.com%"
		fas.Spec.Policy.Webhook.URL = &url
		fas.Spec.Policy.Webhook.Service = nil
		fas.Spec.Policy.Webhook.CABundle = []byte(goodCaBundle)

		causes := fas.Validate()
		assert.Len(t, causes, 1)
		assert.Equal(t, "spec.policy.webhook.url", causes[0].Field)
	})

}

// nolint:dupl  // Linter errors on lines are duplicate of TestFleetAutoscalerListValidateUpdate
func TestFleetAutoscalerCounterValidateUpdate(t *testing.T) {
	t.Parallel()

	modifiedFAS := func(f func(*FleetAutoscalerPolicy)) *FleetAutoscaler {
		fas := counterFixture()
		f(&fas.Spec.Policy)
		return fas
	}

	testCases := map[string]struct {
		fas          *FleetAutoscaler
		featureFlags string
		wantLength   int
		wantField    string
	}{
		"feature gate not turned on": {
			fas:          counterFixture(),
			featureFlags: string(runtime.FeatureCountsAndLists) + "=false",
			wantLength:   1,
			wantField:    "spec.policy.counter",
		},
		"nil parameters": {
			fas: modifiedFAS(func(fap *FleetAutoscalerPolicy) {
				fap.Counter = nil
			}),
			featureFlags: string(runtime.FeatureCountsAndLists) + "=true",
			wantLength:   1,
			wantField:    "spec.policy.counter",
		},
		"minCapacity size too large": {
			fas: modifiedFAS(func(fap *FleetAutoscalerPolicy) {
				fap.Counter.MinCapacity = int64(11)
			}),
			featureFlags: string(runtime.FeatureCountsAndLists) + "=true",
			wantLength:   1,
			wantField:    "spec.policy.counter.minCapacity",
		},
		"bufferSize size too small": {
			fas: modifiedFAS(func(fap *FleetAutoscalerPolicy) {
				fap.Counter.BufferSize = intstr.FromInt(0)
			}),
			featureFlags: string(runtime.FeatureCountsAndLists) + "=true",
			wantLength:   1,
			wantField:    "spec.policy.counter.bufferSize",
		},
		"maxCapacity size too small": {
			fas: modifiedFAS(func(fap *FleetAutoscalerPolicy) {
				fap.Counter.MaxCapacity = int64(4)
			}),
			featureFlags: string(runtime.FeatureCountsAndLists) + "=true",
			wantLength:   1,
			wantField:    "spec.policy.counter.maxCapacity",
		},
		"minCapacity size too small": {
			fas: modifiedFAS(func(fap *FleetAutoscalerPolicy) {
				fap.Counter.MinCapacity = int64(4)
			}),
			featureFlags: string(runtime.FeatureCountsAndLists) + "=true",
			wantLength:   1,
			wantField:    "spec.policy.counter.minCapacity",
		},
		"bufferSize percentage OK": {
			fas: modifiedFAS(func(fap *FleetAutoscalerPolicy) {
				fap.Counter.BufferSize.Type = intstr.String
				fap.Counter.BufferSize = intstr.FromString("99%")
				fap.Counter.MinCapacity = 10
			}),
			featureFlags: string(runtime.FeatureCountsAndLists) + "=true",
			wantLength:   0,
		},
		"bufferSize percentage can't parse": {
			fas: modifiedFAS(func(fap *FleetAutoscalerPolicy) {
				fap.Counter.BufferSize.Type = intstr.String
				fap.Counter.BufferSize = intstr.FromString("99.0%")
				fap.Counter.MinCapacity = 1
			}),
			featureFlags: string(runtime.FeatureCountsAndLists) + "=true",
			wantLength:   1,
			wantField:    "spec.policy.counter.bufferSize",
		},
		"bufferSize percentage and MinCapacity too small": {
			fas: modifiedFAS(func(fap *FleetAutoscalerPolicy) {
				fap.Counter.BufferSize.Type = intstr.String
				fap.Counter.BufferSize = intstr.FromString("0%")
			}),
			featureFlags: string(runtime.FeatureCountsAndLists) + "=true",
			wantLength:   2,
			wantField:    "spec.policy.counter.bufferSize",
		},
		"bufferSize percentage too large": {
			fas: modifiedFAS(func(fap *FleetAutoscalerPolicy) {
				fap.Counter.BufferSize.Type = intstr.String
				fap.Counter.BufferSize = intstr.FromString("100%")
				fap.Counter.MinCapacity = 10
			}),
			featureFlags: string(runtime.FeatureCountsAndLists) + "=true",
			wantLength:   1,
			wantField:    "spec.policy.counter.bufferSize",
		},
	}

	runtime.FeatureTestMutex.Lock()
	defer runtime.FeatureTestMutex.Unlock()

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			err := runtime.ParseFeatures(tc.featureFlags)
			assert.NoError(t, err)

			causes := tc.fas.Validate()

			assert.Len(t, causes, tc.wantLength)
			if tc.wantLength > 0 {
				assert.Equal(t, tc.wantField, causes[0].Field)
			}
		})
	}
}

// nolint:dupl  // Linter errors on lines are duplicate of TestFleetAutoscalerCounterValidateUpdate
func TestFleetAutoscalerListValidateUpdate(t *testing.T) {
	t.Parallel()

	modifiedFAS := func(f func(*FleetAutoscalerPolicy)) *FleetAutoscaler {
		fas := listFixture()
		f(&fas.Spec.Policy)
		return fas
	}

	testCases := map[string]struct {
		fas          *FleetAutoscaler
		featureFlags string
		wantLength   int
		wantField    string
	}{
		"feature gate not turned on": {
			fas:          listFixture(),
			featureFlags: string(runtime.FeatureCountsAndLists) + "=false",
			wantLength:   1,
			wantField:    "spec.policy.list",
		},
		"nil parameters": {
			fas: modifiedFAS(func(fap *FleetAutoscalerPolicy) {
				fap.List = nil
			}),
			featureFlags: string(runtime.FeatureCountsAndLists) + "=true",
			wantLength:   1,
			wantField:    "spec.policy.list",
		},
		"minCapacity size too large": {
			fas: modifiedFAS(func(fap *FleetAutoscalerPolicy) {
				fap.List.MinCapacity = int64(11)
			}),
			featureFlags: string(runtime.FeatureCountsAndLists) + "=true",
			wantLength:   1,
			wantField:    "spec.policy.list.minCapacity",
		},
		"bufferSize size too small": {
			fas: modifiedFAS(func(fap *FleetAutoscalerPolicy) {
				fap.List.BufferSize = intstr.FromInt(0)
			}),
			featureFlags: string(runtime.FeatureCountsAndLists) + "=true",
			wantLength:   1,
			wantField:    "spec.policy.list.bufferSize",
		},
		"maxCapacity size too small": {
			fas: modifiedFAS(func(fap *FleetAutoscalerPolicy) {
				fap.List.MaxCapacity = int64(4)
			}),
			featureFlags: string(runtime.FeatureCountsAndLists) + "=true",
			wantLength:   1,
			wantField:    "spec.policy.list.maxCapacity",
		},
		"minCapacity size too small": {
			fas: modifiedFAS(func(fap *FleetAutoscalerPolicy) {
				fap.List.MinCapacity = int64(4)
			}),
			featureFlags: string(runtime.FeatureCountsAndLists) + "=true",
			wantLength:   1,
			wantField:    "spec.policy.list.minCapacity",
		},
		"bufferSize percentage OK": {
			fas: modifiedFAS(func(fap *FleetAutoscalerPolicy) {
				fap.List.BufferSize.Type = intstr.String
				fap.List.BufferSize = intstr.FromString("99%")
				fap.List.MinCapacity = 1
			}),
			featureFlags: string(runtime.FeatureCountsAndLists) + "=true",
			wantLength:   0,
		},
		"bufferSize percentage can't parse": {
			fas: modifiedFAS(func(fap *FleetAutoscalerPolicy) {
				fap.List.BufferSize.Type = intstr.String
				fap.List.BufferSize = intstr.FromString("99.0%")
				fap.List.MinCapacity = 1
			}),
			featureFlags: string(runtime.FeatureCountsAndLists) + "=true",
			wantLength:   1,
			wantField:    "spec.policy.list.bufferSize",
		},
		"bufferSize percentage and MinCapacity too small": {
			fas: modifiedFAS(func(fap *FleetAutoscalerPolicy) {
				fap.List.BufferSize.Type = intstr.String
				fap.List.BufferSize = intstr.FromString("0%")
			}),
			featureFlags: string(runtime.FeatureCountsAndLists) + "=true",
			wantLength:   2,
			wantField:    "spec.policy.list.bufferSize",
		},
		"bufferSize percentage too large": {
			fas: modifiedFAS(func(fap *FleetAutoscalerPolicy) {
				fap.List.BufferSize.Type = intstr.String
				fap.List.BufferSize = intstr.FromString("100%")
				fap.List.MinCapacity = 1
			}),
			featureFlags: string(runtime.FeatureCountsAndLists) + "=true",
			wantLength:   1,
			wantField:    "spec.policy.list.bufferSize",
		},
	}

	runtime.FeatureTestMutex.Lock()
	defer runtime.FeatureTestMutex.Unlock()

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			err := runtime.ParseFeatures(tc.featureFlags)
			assert.NoError(t, err)

			causes := tc.fas.Validate()

			assert.Len(t, causes, tc.wantLength)
			if tc.wantLength > 0 {
				assert.Equal(t, tc.wantField, causes[0].Field)
			}
		})
	}
}

func TestFleetAutoscalerChainValidateUpdate(t *testing.T) {
	t.Parallel()

	modifiedFAS := func(f func(*FleetAutoscalerPolicy)) *FleetAutoscaler {
		fas := chainFixture()
		f(&fas.Spec.Policy)
		return fas
	}

	testCases := map[string]struct {
		fas          *FleetAutoscaler
		featureFlags string
		wantLength   int
		wantField    string
	}{
		"feature gate not turned on": {
			fas:          chainFixture(),
			featureFlags: string(runtime.FeatureScheduledAutoscaler) + "=false",
			wantLength:   1,
			wantField:    "spec.policy.chain",
		},
		"empty policy": {
			fas: modifiedFAS(func(fap *FleetAutoscalerPolicy) {
				fap.Chain.Items[0].Policy = FleetAutoscalerPolicy{}
			}),
			featureFlags: string(runtime.FeatureScheduledAutoscaler) + "=true",
			wantLength:   1,
			wantField:    "spec.policy.chain.items[0].policy",
		},
		"nested chain policy not allowed": {
			fas: modifiedFAS(func(fap *FleetAutoscalerPolicy) {
				fap.Chain.Items[0].Policy.Chain = &ChainPolicy{}
			}),
			featureFlags: string(runtime.FeatureScheduledAutoscaler) + "=true",
			wantLength:   1,
			wantField:    "spec.policy.chain.items[0].policy.chain",
		},
		"invalid nested policy": {
			fas: modifiedFAS(func(fap *FleetAutoscalerPolicy) {
				fap.Chain.Items[0].Policy.Buffer.MinReplicas = 20
			}
			),
			featureFlags: string(runtime.FeatureScheduledAutoscaler) + "=true",
			wantLength:   1,
			wantField:    "spec.policy.chain.items[0].policy.buffer.minReplicas",
		},
		"invalid time zone": {
			fas: modifiedFAS(func(fap *FleetAutoscalerPolicy) {
				fap.Chain.Items[0].Schedule.Timezone = "invalid"
			}),
			featureFlags: string(runtime.FeatureScheduledAutoscaler) + "=true",
			wantLength:   1,
			wantField:    "spec.policy.chain.items[0].schedule.timezone",
		},
		"invalid start time": {
			fas: modifiedFAS(func(fap *FleetAutoscalerPolicy) {
				fap.Chain.Items[0].Schedule.Between.Start = "invalid"
			}),
			featureFlags: string(runtime.FeatureScheduledAutoscaler) + "=true",
			wantLength:   1,
			wantField:    "spec.policy.chain.items[0].schedule.between.start",
		},
		"invalid end time": {
			fas: modifiedFAS(func(fap *FleetAutoscalerPolicy) {
				fap.Chain.Items[0].Schedule.Between.End = "invalid"
			}),
			featureFlags: string(runtime.FeatureScheduledAutoscaler) + "=true",
			wantLength:   1,
			wantField:    "spec.policy.chain.items[0].schedule.between.end",
		},
		"invalid start cron": {
			fas: modifiedFAS(func(fap *FleetAutoscalerPolicy) {
				fap.Chain.Items[0].Schedule.ActivePeriod.StartCron = "invalid"
			}),
			featureFlags: string(runtime.FeatureScheduledAutoscaler) + "=true",
			wantLength:   1,
			wantField:    "spec.policy.chain.items[0].schedule.activePeriod.startCron",
		},
		"invalid duration": {
			fas: modifiedFAS(func(fap *FleetAutoscalerPolicy) {
				fap.Chain.Items[0].Schedule.ActivePeriod.Duration = "invalid"
			}),
			featureFlags: string(runtime.FeatureScheduledAutoscaler) + "=true",
			wantLength:   1,
			wantField:    "spec.policy.chain.items[0].schedule.activePeriod.duration",
		},
	}

	runtime.FeatureTestMutex.Lock()
	defer runtime.FeatureTestMutex.Unlock()

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			err := runtime.ParseFeatures(tc.featureFlags)
			assert.NoError(t, err)

			causes := tc.fas.Validate()

			assert.Len(t, causes, tc.wantLength)
			if tc.wantLength > 0 {
				assert.Equal(t, tc.wantField, causes[0].Field)
			}
		})
	}
}

func TestFleetAutoscalerApplyDefaults(t *testing.T) {
	fas := &FleetAutoscaler{}

	// gate
	assert.Nil(t, fas.Spec.Sync)

	fas.ApplyDefaults()
	assert.NotNil(t, fas.Spec.Sync)
	assert.Equal(t, FixedIntervalSyncType, fas.Spec.Sync.Type)
	assert.Equal(t, defaultIntervalSyncSeconds, fas.Spec.Sync.FixedInterval.Seconds)

	// Test apply defaults is idempotent -- calling ApplyDefaults more than one time does not change the original result.
	fas.ApplyDefaults()
	assert.NotNil(t, fas.Spec.Sync)
	assert.Equal(t, FixedIntervalSyncType, fas.Spec.Sync.Type)
	assert.Equal(t, defaultIntervalSyncSeconds, fas.Spec.Sync.FixedInterval.Seconds)
}

func defaultFixture() *FleetAutoscaler {
	return customFixture(BufferPolicyType)
}

func webhookFixture() *FleetAutoscaler {
	return customFixture(WebhookPolicyType)
}

func counterFixture() *FleetAutoscaler {
	return customFixture(CounterPolicyType)
}

func listFixture() *FleetAutoscaler {
	return customFixture(ListPolicyType)
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
			Sync: &FleetAutoscalerSync{
				Type: FixedIntervalSyncType,
				FixedInterval: FixedIntervalSync{
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
	case CounterPolicyType:
		res.Spec.Policy.Type = CounterPolicyType
		res.Spec.Policy.Buffer = nil
		res.Spec.Policy.Counter = &CounterPolicy{
			BufferSize:  intstr.FromInt(5),
			MaxCapacity: 10,
		}
	case ListPolicyType:
		res.Spec.Policy.Type = ListPolicyType
		res.Spec.Policy.Buffer = nil
		res.Spec.Policy.List = &ListPolicy{
			BufferSize:  intstr.FromInt(5),
			MaxCapacity: 10,
		}
	}
	return res
}
