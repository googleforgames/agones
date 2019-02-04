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

package e2e

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	"agones.dev/agones/pkg/apis/stable/v1alpha1"
	e2e "agones.dev/agones/test/e2e/framework"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	admregv1b "k8s.io/api/admissionregistration/v1beta1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/wait"
)

var deletePropagationForeground = metav1.DeletePropagationForeground

var waitForDeletion = &metav1.DeleteOptions{
	PropagationPolicy: &deletePropagationForeground,
}

func TestAutoscalerBasicFunctions(t *testing.T) {
	t.Parallel()

	alpha1 := framework.AgonesClient.StableV1alpha1()
	fleets := alpha1.Fleets(defaultNs)
	flt, err := fleets.Create(defaultFleet())
	if assert.Nil(t, err) {
		defer fleets.Delete(flt.ObjectMeta.Name, nil) // nolint:errcheck
	}

	framework.WaitForFleetCondition(t, flt, e2e.FleetReadyCount(flt.Spec.Replicas))

	fleetautoscalers := alpha1.FleetAutoscalers(defaultNs)
	fas, err := fleetautoscalers.Create(defaultFleetAutoscaler(flt))
	if assert.Nil(t, err) {
		defer fleetautoscalers.Delete(fas.ObjectMeta.Name, nil) // nolint:errcheck
	} else {
		// if we could not create the autoscaler, their is no point going further
		logrus.Error("Failed creating autoscaler, aborting TestAutoscalerBasicFunctions")
		return
	}

	// the fleet autoscaler should scale the fleet up now up to BufferSize
	bufferSize := int32(fas.Spec.Policy.Buffer.BufferSize.IntValue())
	framework.WaitForFleetCondition(t, flt, e2e.FleetReadyCount(bufferSize))

	// patch the autoscaler to increase MinReplicas and watch the fleet scale up
	fas, err = patchFleetAutoscaler(fas, intstr.FromInt(int(bufferSize)), bufferSize+2, fas.Spec.Policy.Buffer.MaxReplicas)
	assert.Nil(t, err, "could not patch fleetautoscaler")

	// min replicas is now higher than buffer size, will scale to that level
	framework.WaitForFleetCondition(t, flt, e2e.FleetReadyCount(fas.Spec.Policy.Buffer.MinReplicas))

	// patch the autoscaler to remove MinReplicas and watch the fleet scale down to bufferSize
	fas, err = patchFleetAutoscaler(fas, intstr.FromInt(int(bufferSize)), 0, fas.Spec.Policy.Buffer.MaxReplicas)
	assert.Nil(t, err, "could not patch fleetautoscaler")

	bufferSize = int32(fas.Spec.Policy.Buffer.BufferSize.IntValue())
	framework.WaitForFleetCondition(t, flt, e2e.FleetReadyCount(bufferSize))

	// do an allocation and watch the fleet scale up
	fa := getAllocation(flt)
	fa, err = alpha1.FleetAllocations(defaultNs).Create(fa)
	assert.Nil(t, err)
	assert.Equal(t, v1alpha1.GameServerStateAllocated, fa.Status.GameServer.Status.State)
	framework.WaitForFleetCondition(t, flt, func(fleet *v1alpha1.Fleet) bool {
		return fleet.Status.AllocatedReplicas == 1
	})

	framework.WaitForFleetCondition(t, flt, e2e.FleetReadyCount(bufferSize))

	// patch autoscaler to switch to relative buffer size and check if the fleet adjusts
	_, err = patchFleetAutoscaler(fas, intstr.FromString("10%"), 1, fas.Spec.Policy.Buffer.MaxReplicas)
	assert.Nil(t, err, "could not patch fleetautoscaler")

	//10% with only one allocated GS means only one ready server
	framework.WaitForFleetCondition(t, flt, e2e.FleetReadyCount(1))

	// delete the allocated GameServer and watch the fleet scale down
	gp := int64(1)
	err = alpha1.GameServers(defaultNs).Delete(fa.Status.GameServer.ObjectMeta.Name, &metav1.DeleteOptions{GracePeriodSeconds: &gp})
	assert.Nil(t, err)
	framework.WaitForFleetCondition(t, flt, func(fleet *v1alpha1.Fleet) bool {
		return fleet.Status.AllocatedReplicas == 0 &&
			fleet.Status.ReadyReplicas == 1 &&
			fleet.Status.Replicas == 1
	})
}

// TestAutoscalerStressCreate creates many fleetautoscalers with random values
// to check if the creation validation works as expected and if the fleet scales
// to the expected number of replicas (when the creation is valid)
func TestAutoscalerStressCreate(t *testing.T) {
	t.Parallel()

	alpha1 := framework.AgonesClient.StableV1alpha1()
	fleets := alpha1.Fleets(defaultNs)
	flt, err := fleets.Create(defaultFleet())
	if assert.Nil(t, err) {
		defer fleets.Delete(flt.ObjectMeta.Name, nil) // nolint:errcheck
	}

	framework.WaitForFleetCondition(t, flt, e2e.FleetReadyCount(flt.Spec.Replicas))

	r := rand.New(rand.NewSource(1783))

	fleetautoscalers := alpha1.FleetAutoscalers(defaultNs)

	for i := 0; i < 5; i++ {
		fas := defaultFleetAutoscaler(flt)
		bufferSize := r.Int31n(5)
		minReplicas := r.Int31n(5)
		maxReplicas := r.Int31n(8)
		fas.Spec.Policy.Buffer.BufferSize = intstr.FromInt(int(bufferSize))
		fas.Spec.Policy.Buffer.MinReplicas = minReplicas
		fas.Spec.Policy.Buffer.MaxReplicas = maxReplicas

		valid := bufferSize > 0 &&
			fas.Spec.Policy.Buffer.MaxReplicas > 0 &&
			fas.Spec.Policy.Buffer.MaxReplicas >= bufferSize &&
			fas.Spec.Policy.Buffer.MinReplicas <= fas.Spec.Policy.Buffer.MaxReplicas &&
			(fas.Spec.Policy.Buffer.MinReplicas == 0 || fas.Spec.Policy.Buffer.MinReplicas >= bufferSize)

		// create a closure to have defered delete func called on each loop iteration.
		func() {
			fas, err := fleetautoscalers.Create(fas)
			if err == nil {
				defer fleetautoscalers.Delete(fas.ObjectMeta.Name, nil) // nolint:errcheck
				assert.True(t, valid,
					fmt.Sprintf("FleetAutoscaler created even if the parameters are NOT valid: %d %d %d",
						bufferSize,
						fas.Spec.Policy.Buffer.MinReplicas,
						fas.Spec.Policy.Buffer.MaxReplicas))

				expectedReplicas := bufferSize
				if expectedReplicas < fas.Spec.Policy.Buffer.MinReplicas {
					expectedReplicas = fas.Spec.Policy.Buffer.MinReplicas
				}
				if expectedReplicas > fas.Spec.Policy.Buffer.MaxReplicas {
					expectedReplicas = fas.Spec.Policy.Buffer.MaxReplicas
				}
				// the fleet autoscaler should scale the fleet now to expectedReplicas
				framework.WaitForFleetCondition(t, flt, e2e.FleetReadyCount(expectedReplicas))
			} else {
				assert.False(t, valid,
					fmt.Sprintf("FleetAutoscaler NOT created even if the parameters are valid: %d %d %d (%s)",
						bufferSize,
						minReplicas,
						maxReplicas, err))
			}
		}()
	}
}

// scaleFleet creates a patch to apply to a Fleet.
// easier for testing, as it removes object generational issues.
func patchFleetAutoscaler(fas *v1alpha1.FleetAutoscaler, bufferSize intstr.IntOrString, minReplicas int32, maxReplicas int32) (*v1alpha1.FleetAutoscaler, error) {
	var bufferSizeFmt string
	if bufferSize.Type == intstr.Int {
		bufferSizeFmt = fmt.Sprintf("%d", bufferSize.IntValue())
	} else {
		bufferSizeFmt = fmt.Sprintf(`"%s"`, bufferSize.String())
	}

	patch := fmt.Sprintf(
		`[{ "op": "replace", "path": "/spec/policy/buffer/bufferSize", "value": %s },`+
			`{ "op": "replace", "path": "/spec/policy/buffer/minReplicas", "value": %d },`+
			`{ "op": "replace", "path": "/spec/policy/buffer/maxReplicas", "value": %d }]`,
		bufferSizeFmt, minReplicas, maxReplicas)
	logrus.
		WithField("fleetautoscaler", fas.ObjectMeta.Name).
		WithField("bufferSize", bufferSize.String()).
		WithField("minReplicas", minReplicas).
		WithField("maxReplicas", maxReplicas).
		WithField("patch", patch).
		Info("Patching fleetautoscaler")

	fas, err := framework.AgonesClient.StableV1alpha1().FleetAutoscalers(defaultNs).Patch(fas.ObjectMeta.Name, types.JSONPatchType, []byte(patch))
	logrus.WithField("fleetautoscaler", fas).Info("Patched fleet autoscaler")
	return fas, err
}

// defaultFleetAutoscaler returns a default fleet autoscaler configuration for a given fleet
func defaultFleetAutoscaler(f *v1alpha1.Fleet) *v1alpha1.FleetAutoscaler {
	return &v1alpha1.FleetAutoscaler{
		ObjectMeta: metav1.ObjectMeta{Name: f.ObjectMeta.Name + "-autoscaler", Namespace: defaultNs},
		Spec: v1alpha1.FleetAutoscalerSpec{
			FleetName: f.ObjectMeta.Name,
			Policy: v1alpha1.FleetAutoscalerPolicy{
				Type: v1alpha1.BufferPolicyType,
				Buffer: &v1alpha1.BufferPolicy{
					BufferSize:  intstr.FromInt(3),
					MaxReplicas: 10,
				},
			},
		},
	}
}

func getAllocation(f *v1alpha1.Fleet) *v1alpha1.FleetAllocation {
	// get an allocation
	return &v1alpha1.FleetAllocation{
		ObjectMeta: metav1.ObjectMeta{GenerateName: "allocation-", Namespace: defaultNs},
		Spec: v1alpha1.FleetAllocationSpec{
			FleetName: f.ObjectMeta.Name,
		},
	}
}

//Test fleetautoscaler with webhook policy type
// scaling from Replicas equals to 1 to 2
func TestAutoscalerWebhook(t *testing.T) {
	t.Parallel()
	pod, svc := defaultAutoscalerWebhook()
	pod, err := framework.KubeClient.CoreV1().Pods(defaultNs).Create(pod)
	if assert.Nil(t, err) {
		defer framework.KubeClient.CoreV1().Pods(defaultNs).Delete(pod.ObjectMeta.Name, nil) // nolint:errcheck
	} else {
		// if we could not create the webhook pod, there is no point going further
		assert.FailNow(t, "Failed creating webhook pod, aborting TestAutoscalerWebhook")
	}
	svc.ObjectMeta.Name = ""
	svc.ObjectMeta.GenerateName = "test-service-"

	svc, err = framework.KubeClient.CoreV1().Services(defaultNs).Create(svc)
	if assert.Nil(t, err) {
		defer framework.KubeClient.CoreV1().Services(defaultNs).Delete(svc.ObjectMeta.Name, nil) // nolint:errcheck
	} else {
		// if we could not create the webhook service, there is no point going further
		assert.FailNow(t, "Failed creating webhook service, aborting TestAutoscalerWebhook")
	}

	alpha1 := framework.AgonesClient.StableV1alpha1()
	fleets := alpha1.Fleets(defaultNs)
	flt := defaultFleet()
	initialReplicasCount := int32(1)
	flt.Spec.Replicas = initialReplicasCount
	flt, err = fleets.Create(flt)
	if assert.Nil(t, err) {
		defer fleets.Delete(flt.ObjectMeta.Name, nil) // nolint:errcheck
	}

	framework.WaitForFleetCondition(t, flt, e2e.FleetReadyCount(flt.Spec.Replicas))

	fleetautoscalers := alpha1.FleetAutoscalers(defaultNs)
	fas := defaultFleetAutoscaler(flt)
	fas.Spec.Policy.Type = v1alpha1.WebhookPolicyType
	fas.Spec.Policy.Buffer = nil
	path := "scale"
	fas.Spec.Policy.Webhook = &v1alpha1.WebhookPolicy{
		Service: &admregv1b.ServiceReference{
			Name:      svc.ObjectMeta.Name,
			Namespace: defaultNs,
			Path:      &path,
		},
	}
	fas, err = fleetautoscalers.Create(fas)
	if assert.Nil(t, err) {
		defer fleetautoscalers.Delete(fas.ObjectMeta.Name, nil) // nolint:errcheck
	} else {
		// if we could not create the autoscaler, there is no point going further
		assert.FailNow(t, "Failed creating autoscaler, aborting TestAutoscalerWebhook")
	}
	fa := getAllocation(flt)
	fa, err = alpha1.FleetAllocations(defaultNs).Create(fa)
	assert.Nil(t, err)
	assert.Equal(t, v1alpha1.GameServerStateAllocated, fa.Status.GameServer.Status.State)
	framework.WaitForFleetCondition(t, flt, func(fleet *v1alpha1.Fleet) bool {
		return fleet.Status.AllocatedReplicas == 1
	})

	framework.WaitForFleetCondition(t, flt, func(fleet *v1alpha1.Fleet) bool {
		return fleet.Status.Replicas > initialReplicasCount
	})
}

var webhookKey = `
-----BEGIN RSA PRIVATE KEY-----
MIIEogIBAAKCAQEA2a+qgmnbyKsKHe/ayQSF17966ghXickvFJr8MtHsAVVfDPTu
K50wmDN5IlkIf5IHqM/4imtw2LXGgFlGIlIsytF8Cy8gW4nqJjZY0XSNmmnJF3Mc
O62Ptys0+JxsRqjkpkHzxV7atAdVUWqPzz97UPhcf62qUWxw/zyjA1InTj+kDxMZ
KzCqhetgCW2IGSsb6h4zub/3My/TytP+aDY5P5hHEl1C2ZNvLt+lweaF9TAQ3Pi1
XCIpp0HIHg0CviDMOxtKO44v+ZuYyuwBJ07f5ny/jzA3+BDmcXiH2drb4EjMzpEk
7oWfYpkoso++YB7JlDZBAON0ryaetgCHkuw8wwIDAQABAoIBAAOIvpPvdAoF/NwP
kNXCpQmjqjMyf3lVMtZ6za1lixdac3iaYWOD4c4Wx9iu6Vxo2ob7GWXl6KccDGT5
DhJwkxmX3ROxaC0USCDmsPp1kfb30LP4wnSVlMe8g9elcnyTMWMhnvuNVq+ljtUL
jdonhbEC1z2bbDB2Oj9qlJrxMoIqrqzLAE2ibC7hS9P70NKjAjgof6kWYLaMv7Bs
o6ONlrJL8jBfYDd82GZOXQf81WzMbV1wA6waiArSOKLrpDngnjGN38whCKg3R8yC
ysmCAzazSZpPmoiA9roNfcSxIsA8NFPWqTY8bOBlFnh+dm2qBPmfeUTgN8O+d+eM
gsAhCgECgYEA7kxq5A81SZDGoJuVYbeU74ORy1O6KfJ21nqJSABnxQp4tmOLSTSE
PT/UxVRVHCln1YlaZG0TGvrjLN9HOc8STJWfwORN+Dh2p37IlF6gkwrCWS6kWI5H
N9vxgxgI5m1p+Y1p4OOdgqTy2NlTjmK5tiwO0MErHwY1LwJSTcVByiECgYEA6dtG
vZ42nFA3WCTJhRG/WdWImg/gzBE2hqfKsoJ1Xw/qhBhO8jgNTB913Pmhby2Afj8T
dnB0EyjwD2vwVgrDafNr0BOutJEJ/dvgObRHJjDHzPhQ9nwxzpSDNkblV0jGoTL1
7iwyblkWOVUpMti3NS1W3Oqrq1LAcC3OZPUs0mMCgYBpIVuTC8aVkwKePqWTu7tA
Q8phaqnZ8bdN/jdshYlCW9FPnfEINdwVbYDAIel+iCHgCj3PynNAVuk8lbDFpz5K
fURChDaFyNtIH9373xd2Z6vATpyA2RxAX49YJ5Vdm23ChAnvBlwqE/1zf8WmLpYB
8cQDgwU0Jbf26k5HMzxIIQKBgFHiui6DS9QIMpjmqLmzsTEfmCl6Ddjm3hTghBVl
oPuccx219U7TWbSh/39U2bY4VJngNExwq/RZjVWZEhrOwgZDeijt+2q2rqz5ZNZP
zeoNgqi++nqUmkwfrKJAyOV7UjH3yi2PxEjnYOTKcRag0+YG7jeE5H+lBkVBhNfN
EdjJAoGAfNI/In1n1XG9n0N+fidouSGPBWL8zt+peQ6YvWNzyq5tlpfQeatwRbZJ
9vgyqxHWpiYZs44pjM9oahT4KeHO0OUqNQw9DLc4jNC6+eb/FwGNM1d7bft9s+e5
rney13WRt3xasYCzx0cl6+zJXI3DcY48O82EdI5am9vWsFpfzVc=
-----END RSA PRIVATE KEY-----`

var caPem = `
-----BEGIN CERTIFICATE-----
MIICuDCCAaACCQCodpAMm9SwJjANBgkqhkiG9w0BAQsFADAeMQswCQYDVQQGEwJV
UzEPMA0GA1UECwwGQWdvbmVzMB4XDTE5MDEwNDExNTE0NFoXDTIxMTAyNDExNTE0
NFowHjELMAkGA1UEBhMCVVMxDzANBgNVBAsMBkFnb25lczCCASIwDQYJKoZIhvcN
AQEBBQADggEPADCCAQoCggEBANCHyvwC96pd9SvAa1B/Eh6zG1x0KKWOiXm78Irx
+6yXwyaZl1Z5qQ1mFh98LHeYRd0YX3E2gzVylZoRU+kVDK4TsEsWKMPUiuZ41Ekt
po+alCzjP2ivsDfZ8a/vpr/wYgakXkVjPThjJROqNqHudN26UqAIbsNMZhRLd9QE
qKJ4O6aG5S1MSjdTFTqelrbf+CqsJhymdB3flFEEouq1Jj1KDhB4WZSSm/UJzB6G
4u3cpeBmcLUQGm6fQGobEA+yJZLhEWqpkwvUggB6dsXO1dSHexafiC9ETXlUtTag
5SbNy5hadVQUwgnwSBvv4vGKuQLWqgWsBrk0yZYxJNAoEyECAwEAATANBgkqhkiG
9w0BAQsFAAOCAQEAQ2H3ibQqf3A3DKiuxbHIDdnYzNVvgGaDZpiVr3nhrnyvle5X
GOaFm+27QF4VWoE36CLhXdzDZS8lJHcOXQnJ9O7cjOc91VhuKcfHx0KOaSZ0ySkT
vlKWk9A4Wh4a4AqYJW7gpTTtuPZrvw8Tk/n1ZXFNaWAx7yENNuWb88h4dAD5ZO4s
G9HrHvZnM3WC1AQp4CyZF5rCR6vEE9ddRiJor33zKe4hFBo7BENIYesseYqE+dp3
+H8MnK84Wx1TgSyVJy8yLmqiu2ui8ch1Hfxt8Zcpx7up6HFKFTlN9AyvTiv1a0X/
DU95y10v/hNW4Xzn02G4hkr8siGnHG+PJkOxAw==
-----END CERTIFICATE-----`

var webhookCrt = `
-----BEGIN CERTIFICATE-----
MIIC6TCCAdECCQCLqOrlK/jyADANBgkqhkiG9w0BAQsFADAeMQswCQYDVQQGEwJV
UzEPMA0GA1UECwwGQWdvbmVzMB4XDTE5MDEwNDEzMzczOVoXDTIwMDUxODEzMzcz
OVowTzEPMA0GA1UECgwGQWdvbmVzMQ8wDQYDVQQLDAZBZ29uZXMxKzApBgNVBAMM
ImF1dG9zY2FsZXItdGxzLXNlcnZpY2UuZGVmYXVsdC5zdmMwggEiMA0GCSqGSIb3
DQEBAQUAA4IBDwAwggEKAoIBAQDZr6qCadvIqwod79rJBIXXv3rqCFeJyS8Umvwy
0ewBVV8M9O4rnTCYM3kiWQh/kgeoz/iKa3DYtcaAWUYiUizK0XwLLyBbieomNljR
dI2aackXcxw7rY+3KzT4nGxGqOSmQfPFXtq0B1VRao/PP3tQ+Fx/rapRbHD/PKMD
UidOP6QPExkrMKqF62AJbYgZKxvqHjO5v/czL9PK0/5oNjk/mEcSXULZk28u36XB
5oX1MBDc+LVcIimnQcgeDQK+IMw7G0o7ji/5m5jK7AEnTt/mfL+PMDf4EOZxeIfZ
2tvgSMzOkSTuhZ9imSiyj75gHsmUNkEA43SvJp62AIeS7DzDAgMBAAEwDQYJKoZI
hvcNAQELBQADggEBAJXHdBh7fw62+fhNsbNbq6HAzigDjf2LrvmuIWlQE6qQnGkx
TVgf+ZnSxvv5u+inOVNkPwbQtoMlWqSBgHMFj3O2mFVnWvO1nj0ajzSN6GAZszws
ZUy8FCRIJbbyqhNsjB/x0ZXM4cpotgtuIe55h7psZU13f7GAuxE8E5anc44Tdufw
ccYzVogM+wEna/pHPOo3ITR4c2k7zVgrr75LkFokUK0fsgFVJ4zTsMP+kQ/UTVmt
kpXqAOUeQx4ZfwM0FI5Yj3Ox5/AsdZ/hNzszjnPFyjKAp+AWjCiDu6VcgFj+WW0L
T1HcD9NOEIwRUO04DY86+P4d0TFY/SwxAiMnwBQ=
-----END CERTIFICATE-----`

func TestTlsWebhook(t *testing.T) {
	t.Parallel()
	secr := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "autoscalersecret-",
		},
		Type: corev1.SecretTypeTLS,
		Data: make(map[string][]byte),
	}

	secr.Data[corev1.TLSCertKey] = []byte(webhookCrt)
	secr.Data[corev1.TLSPrivateKeyKey] = []byte(webhookKey)

	secrets := framework.KubeClient.CoreV1().Secrets(defaultNs)
	secr, err := secrets.Create(secr.DeepCopy())
	if assert.Nil(t, err) {
		defer secrets.Delete(secr.ObjectMeta.Name, nil) // nolint:errcheck
	}

	pod, svc := defaultAutoscalerWebhook()
	pod.Spec.Volumes = make([]corev1.Volume, 1)
	pod.Spec.Volumes[0] = corev1.Volume{
		Name: "secret-volume",
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName: secr.ObjectMeta.Name,
			},
		},
	}
	pod.Spec.Containers[0].VolumeMounts = []corev1.VolumeMount{{
		Name:      "secret-volume",
		MountPath: "/home/service/certs",
	}}
	pod, err = framework.KubeClient.CoreV1().Pods(defaultNs).Create(pod.DeepCopy())
	if assert.Nil(t, err) {
		defer framework.KubeClient.CoreV1().Pods(defaultNs).Delete(pod.ObjectMeta.Name, nil) // nolint:errcheck
	} else {
		// if we could not create the webhook, there is no point going further
		assert.FailNow(t, "Failed creating webhook pod, aborting TestTlsWebhook")
	}

	// since we're using statically-named service, perform a best-effort delete of a previous service
	err = framework.KubeClient.CoreV1().Services(defaultNs).Delete(svc.ObjectMeta.Name, waitForDeletion)
	if err != nil {
		assert.True(t, k8serrors.IsNotFound(err))
	}

	// making sure the service is really gone.
	err = wait.PollImmediate(2*time.Second, time.Minute, func() (bool, error) {
		_, err := framework.KubeClient.CoreV1().Services(defaultNs).Get(svc.ObjectMeta.Name, metav1.GetOptions{})
		return k8serrors.IsNotFound(err), nil
	})
	assert.Nil(t, err)

	svc, err = framework.KubeClient.CoreV1().Services(defaultNs).Create(svc.DeepCopy())
	if assert.Nil(t, err) {
		defer framework.KubeClient.CoreV1().Services(defaultNs).Delete(svc.ObjectMeta.Name, nil) // nolint:errcheck
	} else {
		// if we could not create the service, there is no point going further
		assert.FailNow(t, "Failed creating service, aborting TestTlsWebhook")
	}

	alpha1 := framework.AgonesClient.StableV1alpha1()
	fleets := alpha1.Fleets(defaultNs)
	flt := defaultFleet()
	initialReplicasCount := int32(1)
	flt.Spec.Replicas = initialReplicasCount
	flt, err = fleets.Create(flt.DeepCopy())
	if assert.Nil(t, err) {
		defer fleets.Delete(flt.ObjectMeta.Name, nil) // nolint:errcheck
	}

	framework.WaitForFleetCondition(t, flt, e2e.FleetReadyCount(flt.Spec.Replicas))

	fleetautoscalers := alpha1.FleetAutoscalers(defaultNs)
	fas := defaultFleetAutoscaler(flt)
	fas.Spec.Policy.Type = v1alpha1.WebhookPolicyType
	fas.Spec.Policy.Buffer = nil
	path := "scale"

	fas.Spec.Policy.Webhook = &v1alpha1.WebhookPolicy{
		Service: &admregv1b.ServiceReference{
			Name:      svc.ObjectMeta.Name,
			Namespace: defaultNs,
			Path:      &path,
		},
		CABundle: []byte(caPem),
	}
	fas, err = fleetautoscalers.Create(fas.DeepCopy())
	if assert.Nil(t, err) {
		defer fleetautoscalers.Delete(fas.ObjectMeta.Name, nil) // nolint:errcheck
	} else {
		// if we could not create the autoscaler, their is no point going further
		assert.FailNow(t, "Failed creating autoscaler, aborting TestTlsWebhook")
	}
	fa := getAllocation(flt)
	fa, err = alpha1.FleetAllocations(defaultNs).Create(fa.DeepCopy())
	assert.Nil(t, err)
	assert.Equal(t, v1alpha1.GameServerStateAllocated, fa.Status.GameServer.Status.State)
	framework.WaitForFleetCondition(t, flt, func(fleet *v1alpha1.Fleet) bool {
		return fleet.Status.AllocatedReplicas == 1
	})

	framework.WaitForFleetCondition(t, flt, func(fleet *v1alpha1.Fleet) bool {
		return fleet.Status.Replicas > initialReplicasCount
	})
}

func defaultAutoscalerWebhook() (*corev1.Pod, *corev1.Service) {
	l := make(map[string]string)
	appName := fmt.Sprintf("autoscaler-webhook-%v", time.Now().UnixNano())
	l["app"] = appName
	l[e2e.AutoCleanupLabelKey] = e2e.AutoCleanupLabelValue
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "auto-webhook-",
			Namespace:    defaultNs,
			Labels:       l,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{{Name: "webhook",
				Image:           "gcr.io/agones-images/autoscaler-webhook:0.2",
				ImagePullPolicy: corev1.PullAlways,
				Ports: []corev1.ContainerPort{{
					ContainerPort: 8000,
					Name:          "autoscaler",
				}},
			}},
		},
	}
	m := make(map[string]string)
	m["app"] = appName
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "autoscaler-tls-service",
			Namespace: defaultNs,
		},
		Spec: corev1.ServiceSpec{
			Selector: m,
			Ports: []corev1.ServicePort{{
				Name:       "newport",
				Port:       8000,
				TargetPort: intstr.IntOrString{StrVal: "autoscaler"},
			}},
		},
	}

	return pod, service
}
