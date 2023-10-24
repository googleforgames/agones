// Copyright 2023 Google LLC All Rights Reserved.
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

// Package allochelper is a package for helper function that is used by e2e tests
package allochelper

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"testing"
	"time"

	pb "agones.dev/agones/pkg/allocation/go"
	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	multiclusterv1 "agones.dev/agones/pkg/apis/multicluster/v1"
	e2e "agones.dev/agones/test/e2e/framework"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/wait"
)

const (
	agonesSystemNamespace          = "agones-system"
	allocatorServiceName           = "agones-allocator"
	allocatorTLSName               = "allocator-tls"
	tlsCrtTag                      = "tls.crt"
	tlsKeyTag                      = "tls.key"
	allocatorReqURLFmt             = "%s:%d"
	allocatorClientSecretName      = "allocator-client.default"
	allocatorClientSecretNamespace = "default"
	replicasCount                  = 5
)

// CopyDefaultAllocatorClientSecret copys the allocator client secret
func CopyDefaultAllocatorClientSecret(ctx context.Context, t *testing.T, toNamespace string, framework *e2e.Framework) {
	kubeCore := framework.KubeClient.CoreV1()
	clientSecret, err := kubeCore.Secrets(allocatorClientSecretNamespace).Get(ctx, allocatorClientSecretName, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Could not retrieve default allocator client secret %s/%s: %v", allocatorClientSecretNamespace, allocatorClientSecretName, err)
	}
	clientSecret.ObjectMeta.Namespace = toNamespace
	clientSecret.ResourceVersion = ""
	_, err = kubeCore.Secrets(toNamespace).Create(ctx, clientSecret, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Could not copy default allocator client %s/%s secret to namespace %s: %v", allocatorClientSecretNamespace, allocatorClientSecretName, toNamespace, err)
	}
}

// CreateAllocationPolicy create a allocation policy
func CreateAllocationPolicy(ctx context.Context, t *testing.T, framework *e2e.Framework, p *multiclusterv1.GameServerAllocationPolicy) {
	t.Helper()

	mc := framework.AgonesClient.MulticlusterV1()
	policy, err := mc.GameServerAllocationPolicies(p.Namespace).Create(ctx, p, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("creating allocation policy failed: %s", err)
	}
	t.Logf("created allocation policy %v", policy)
}

// GetAllocatorEndpoint gets the allocator LB endpoint
func GetAllocatorEndpoint(ctx context.Context, t *testing.T, framework *e2e.Framework) (string, int32) {
	kubeCore := framework.KubeClient.CoreV1()
	svc, err := kubeCore.Services(agonesSystemNamespace).Get(ctx, allocatorServiceName, metav1.GetOptions{})
	if !assert.Nil(t, err) {
		t.FailNow()
	}
	if !assert.NotNil(t, svc.Status.LoadBalancer) {
		t.FailNow()
	}
	if !assert.Equal(t, 1, len(svc.Status.LoadBalancer.Ingress)) {
		t.FailNow()
	}
	if !assert.NotNil(t, 0, svc.Status.LoadBalancer.Ingress[0].IP) {
		t.FailNow()
	}

	port := svc.Spec.Ports[0]
	return svc.Status.LoadBalancer.Ingress[0].IP, port.Port
}

// CreateRemoteClusterDialOption creates a grpc client dial option with proper certs to make a remote call.
//
//nolint:unparam
func CreateRemoteClusterDialOption(ctx context.Context, namespace, clientSecretName string, tlsCA []byte, framework *e2e.Framework) (grpc.DialOption, error) {
	tlsConfig, err := GetTLSConfig(ctx, namespace, clientSecretName, tlsCA, framework)
	if err != nil {
		return nil, err
	}

	return grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)), nil
}

// GetTLSConfig gets the namesapce client secret
func GetTLSConfig(ctx context.Context, namespace, clientSecretName string, tlsCA []byte, framework *e2e.Framework) (*tls.Config, error) {
	kubeCore := framework.KubeClient.CoreV1()
	clientSecret, err := kubeCore.Secrets(namespace).Get(ctx, clientSecretName, metav1.GetOptions{})
	if err != nil {
		return nil, errors.Errorf("getting client secret %s/%s failed: %s", namespace, clientSecretName, err)
	}

	// Create http client using cert
	clientCert := clientSecret.Data[tlsCrtTag]
	clientKey := clientSecret.Data[tlsKeyTag]
	if clientCert == nil || clientKey == nil {
		return nil, errors.New("missing certificate")
	}

	// Load client cert
	cert, err := tls.X509KeyPair(clientCert, clientKey)
	if err != nil {
		return nil, err
	}

	rootCA := x509.NewCertPool()
	if !rootCA.AppendCertsFromPEM(tlsCA) {
		return nil, errors.New("could not append PEM format CA cert")
	}

	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      rootCA,
	}, nil
}

// CreateFleet creates a game server fleet
func CreateFleet(ctx context.Context, namespace string, framework *e2e.Framework) (*agonesv1.Fleet, error) {
	return CreateFleetWithOpts(ctx, namespace, framework, func(*agonesv1.Fleet) {})
}

// CreateFleetWithOpts creates a game server fleet with the designated options
func CreateFleetWithOpts(ctx context.Context, namespace string, framework *e2e.Framework, opts func(fleet *agonesv1.Fleet)) (*agonesv1.Fleet, error) {
	fleets := framework.AgonesClient.AgonesV1().Fleets(namespace)
	fleet := defaultFleet(namespace, framework)
	opts(fleet)
	return fleets.Create(ctx, fleet, metav1.CreateOptions{})
}

// RefreshAllocatorTLSCerts refreshes the allocator TLS cert with a newly generated cert
func RefreshAllocatorTLSCerts(ctx context.Context, t *testing.T, host string, framework *e2e.Framework) []byte {
	t.Helper()

	pub, priv := generateTLSCertPair(t, host)
	// verify key pair
	if _, err := tls.X509KeyPair(pub, priv); err != nil {
		t.Fatalf("generated key pair failed create cert: %s", err)
	}

	kubeCore := framework.KubeClient.CoreV1()

	require.Eventually(t, func() bool {
		s, err := kubeCore.Secrets(agonesSystemNamespace).Get(ctx, allocatorTLSName, metav1.GetOptions{})
		if err != nil {
			t.Logf("failed getting secret %s/%s failed: %s", agonesSystemNamespace, allocatorTLSName, err)
			return false
		}

		s.Data[tlsCrtTag] = pub
		s.Data[tlsKeyTag] = priv
		if _, err := kubeCore.Secrets(agonesSystemNamespace).Update(ctx, s, metav1.UpdateOptions{}); err != nil {
			t.Logf("failed updating secrets failed: %s", err)
			return false
		}

		return true
	}, time.Minute, time.Second, "Could not update Secret")

	t.Logf("Allocator TLS is refreshed with public CA: %s for endpoint %s", string(pub), host)
	return pub
}

func generateTLSCertPair(t *testing.T, host string) ([]byte, []byte) {
	t.Helper()

	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generating RSA key failed: %s", err)
	}

	notBefore := time.Now()
	notAfter := notBefore.Add(time.Hour)

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		t.Fatalf("generating serial number failed: %s", err)
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName:   host,
			Organization: []string{"testing"},
		},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		SignatureAlgorithm:    x509.SHA1WithRSA,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	if host != "" {
		template.IPAddresses = []net.IP{net.ParseIP(host)}
	}
	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		t.Fatalf("creating certificate failed: %s", err)
	}
	pemPubBytes := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	privBytes, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		t.Fatalf("marshalling private key failed: %v", err)
	}
	pemPrivBytes := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privBytes})

	return pemPubBytes, pemPrivBytes
}

// ValidateAllocatorResponse validates the response returned by the allcoator
func ValidateAllocatorResponse(t *testing.T, resp *pb.AllocationResponse) {
	t.Helper()
	if !assert.NotNil(t, resp) {
		return
	}
	assert.Greater(t, len(resp.Ports), 0)
	assert.NotEmpty(t, resp.GameServerName)
	assert.NotEmpty(t, resp.Address)
	assert.NotEmpty(t, resp.Addresses)
	assert.NotEmpty(t, resp.NodeName)
	assert.NotEmpty(t, resp.Metadata.Labels)
	assert.NotEmpty(t, resp.Metadata.Annotations)
}

// DeleteAgonesAllocatorPod deletes a Agones allocator pod
func DeleteAgonesAllocatorPod(ctx context.Context, podName string, framework *e2e.Framework) error {
	return DeleteAgonesPod(ctx, podName, "agones-system", framework)
}

// DeleteAgonesPod deletes an Agones pod with the specified namespace and podname
func DeleteAgonesPod(ctx context.Context, podName string, namespace string, framework *e2e.Framework) error {
	policy := metav1.DeletePropagationBackground
	err := framework.KubeClient.CoreV1().Pods(namespace).Delete(ctx, podName,
		metav1.DeleteOptions{PropagationPolicy: &policy})
	return err
}

// GetAgonesAllocatorPods returns all the Agones allocator pods
func GetAgonesAllocatorPods(ctx context.Context, framework *e2e.Framework) (*corev1.PodList, error) {
	opts := metav1.ListOptions{LabelSelector: labels.Set{"multicluster.agones.dev/role": "allocator"}.String()}
	return framework.KubeClient.CoreV1().Pods("agones-system").List(ctx, opts)
}

// GetAllocatorClient creates a client and ensure that it can be connected to
func GetAllocatorClient(ctx context.Context, t *testing.T, framework *e2e.Framework) (pb.AllocationServiceClient, error) {
	ip, port := GetAllocatorEndpoint(ctx, t, framework)
	requestURL := fmt.Sprintf(allocatorReqURLFmt, ip, port)
	tlsCA := RefreshAllocatorTLSCerts(ctx, t, ip, framework)

	flt, err := CreateFleet(ctx, framework.Namespace, framework)
	if !assert.Nil(t, err) {
		return nil, err
	}
	framework.AssertFleetCondition(t, flt, e2e.FleetReadyCount(flt.Spec.Replicas))

	dialOpts, err := CreateRemoteClusterDialOption(ctx, allocatorClientSecretNamespace, allocatorClientSecretName, tlsCA, framework)
	if err != nil {
		return nil, err
	}

	conn, err := grpc.Dial(requestURL, dialOpts)
	require.NoError(t, err, "Failed grpc.Dial")
	go func() {
		<-ctx.Done()
		conn.Close() // nolint: errcheck
	}()

	grpcClient := pb.NewAllocationServiceClient(conn)

	request := &pb.AllocationRequest{
		Namespace:                    framework.Namespace,
		PreferredGameServerSelectors: []*pb.GameServerSelector{{MatchLabels: map[string]string{agonesv1.FleetNameLabel: flt.ObjectMeta.Name}}},
		Scheduling:                   pb.AllocationRequest_Packed,
		Metadata:                     &pb.MetaPatch{Labels: map[string]string{"gslabel": "allocatedbytest"}},
	}

	var response *pb.AllocationResponse
	err = wait.PollUntilContextTimeout(context.Background(), 2*time.Second, 5*time.Minute, true, func(ctx context.Context) (bool, error) {
		response, err = grpcClient.Allocate(context.Background(), request)
		if err != nil {
			logrus.WithError(err).Info("failing Allocate request")
			return false, nil
		}
		ValidateAllocatorResponse(t, response)
		return true, nil
	})
	if err != nil {
		return nil, err
	}

	return grpcClient, nil
}

// CleanupNamespaces cleans up the framework namespace
func CleanupNamespaces(ctx context.Context, framework *e2e.Framework) error {
	// list all e2e namespaces
	opts := metav1.ListOptions{LabelSelector: labels.Set(e2e.NamespaceLabel).String()}
	list, err := framework.KubeClient.CoreV1().Namespaces().List(ctx, opts)
	if err != nil {
		return err
	}

	// loop through them, and delete them
	for _, ns := range list.Items {
		if err := framework.DeleteNamespace(ns.ObjectMeta.Name); err != nil {
			cause := errors.Cause(err)
			if k8serrors.IsConflict(cause) {
				logrus.WithError(cause).Warn("namespace already being deleted")
				continue
			}
			// here just in case we need to catch other errors
			logrus.WithField("reason", k8serrors.ReasonForError(cause)).Info("cause for namespace deletion error")
			return cause
		}
	}

	return nil
}

// From fleet_test
// defaultFleet returns a default fleet configuration
func defaultFleet(namespace string, framework *e2e.Framework) *agonesv1.Fleet {
	gs := framework.DefaultGameServer(namespace)
	return fleetWithGameServerSpec(&gs.Spec, namespace)
}

// fleetWithGameServerSpec returns a fleet with specified gameserver spec
func fleetWithGameServerSpec(gsSpec *agonesv1.GameServerSpec, namespace string) *agonesv1.Fleet {
	return &agonesv1.Fleet{
		ObjectMeta: metav1.ObjectMeta{GenerateName: "simple-fleet-1.0", Namespace: namespace},
		Spec: agonesv1.FleetSpec{
			Replicas: replicasCount,
			Template: agonesv1.GameServerTemplateSpec{
				Spec: *gsSpec,
			},
		},
	}
}
