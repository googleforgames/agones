// Copyright 2019 Google LLC All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package e2e

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"testing"
	"time"

	pb "agones.dev/agones/pkg/allocation/go"
	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	multiclusterv1 "agones.dev/agones/pkg/apis/multicluster/v1"
	"agones.dev/agones/pkg/util/runtime"
	e2e "agones.dev/agones/test/e2e/framework"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/status"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/uuid"
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
)

func TestAllocatorWithDeprecatedRequired(t *testing.T) {
	ctx := context.Background()

	ip, port := getAllocatorEndpoint(ctx, t)
	requestURL := fmt.Sprintf(allocatorReqURLFmt, ip, port)
	tlsCA := refreshAllocatorTLSCerts(ctx, t, ip)

	var flt *agonesv1.Fleet
	var err error
	if runtime.FeatureEnabled(runtime.FeaturePlayerAllocationFilter) {
		flt, err = createFleetWithOpts(ctx, framework.Namespace, func(f *agonesv1.Fleet) {
			f.Spec.Template.Spec.Players = &agonesv1.PlayersSpec{
				InitialCapacity: 10,
			}
		})
	} else {
		flt, err = createFleet(ctx, framework.Namespace)
	}
	assert.NoError(t, err)

	framework.AssertFleetCondition(t, flt, e2e.FleetReadyCount(flt.Spec.Replicas))
	request := &pb.AllocationRequest{
		Namespace:                    framework.Namespace,
		RequiredGameServerSelector:   &pb.GameServerSelector{MatchLabels: map[string]string{agonesv1.FleetNameLabel: flt.ObjectMeta.Name}},
		PreferredGameServerSelectors: []*pb.GameServerSelector{{MatchLabels: map[string]string{agonesv1.FleetNameLabel: flt.ObjectMeta.Name}}},
		Scheduling:                   pb.AllocationRequest_Packed,
		Metadata:                     &pb.MetaPatch{Labels: map[string]string{"gslabel": "allocatedbytest"}},
	}

	var response *pb.AllocationResponse
	// wait for the allocation system to come online
	err = wait.PollImmediate(2*time.Second, 5*time.Minute, func() (bool, error) {
		// create the grpc client each time, as we may end up looking at an old cert
		dialOpts, err := createRemoteClusterDialOption(ctx, allocatorClientSecretNamespace, allocatorClientSecretName, tlsCA)
		if err != nil {
			return false, err
		}

		conn, err := grpc.Dial(requestURL, dialOpts)
		if err != nil {
			logrus.WithError(err).Info("failing grpc.Dial")
			return false, nil
		}
		defer conn.Close() // nolint: errcheck

		grpcClient := pb.NewAllocationServiceClient(conn)
		response, err = grpcClient.Allocate(context.Background(), request)
		if err != nil {
			logrus.WithError(err).Info("failing Allocate request")
			return false, nil
		}
		validateAllocatorResponse(t, response)

		// let's do a re-allocation
		if runtime.FeatureEnabled(runtime.FeatureStateAllocationFilter) && runtime.FeatureEnabled(runtime.FeaturePlayerAllocationFilter) {
			logrus.Info("testing state allocation filter")
			// nolint:staticcheck
			request.PreferredGameServerSelectors[0].GameServerState = pb.GameServerSelector_ALLOCATED
			allocatedResponse, err := grpcClient.Allocate(context.Background(), request)
			require.NoError(t, err)
			require.Equal(t, response.GameServerName, allocatedResponse.GameServerName)
			validateAllocatorResponse(t, allocatedResponse)

			// do a capacity based allocation
			logrus.Info("testing capacity allocation filter")
			// nolint:staticcheck
			request.PreferredGameServerSelectors[0].Players = &pb.PlayerSelector{
				MinAvailable: 5,
				MaxAvailable: 10,
			}
			allocatedResponse, err = grpcClient.Allocate(context.Background(), request)
			require.NoError(t, err)
			require.Equal(t, response.GameServerName, allocatedResponse.GameServerName)
			validateAllocatorResponse(t, allocatedResponse)

			// do a capacity based allocation that should fail
			// nolint:staticcheck
			request.PreferredGameServerSelectors = nil
			// nolint:staticcheck
			request.RequiredGameServerSelector.GameServerState = pb.GameServerSelector_ALLOCATED
			// nolint:staticcheck
			request.RequiredGameServerSelector.Players = &pb.PlayerSelector{MinAvailable: 99, MaxAvailable: 200}

			allocatedResponse, err = grpcClient.Allocate(context.Background(), request)
			assert.Nil(t, allocatedResponse)
			status, ok := status.FromError(err)
			require.True(t, ok)
			assert.Equal(t, codes.ResourceExhausted, status.Code())
		}

		return true, nil
	})

	assert.NoError(t, err)
}

func TestAllocatorWithSelectors(t *testing.T) {
	ctx := context.Background()

	ip, port := getAllocatorEndpoint(ctx, t)
	requestURL := fmt.Sprintf(allocatorReqURLFmt, ip, port)
	tlsCA := refreshAllocatorTLSCerts(ctx, t, ip)

	var flt *agonesv1.Fleet
	var err error
	if runtime.FeatureEnabled(runtime.FeaturePlayerAllocationFilter) {
		flt, err = createFleetWithOpts(ctx, framework.Namespace, func(f *agonesv1.Fleet) {
			f.Spec.Template.Spec.Players = &agonesv1.PlayersSpec{
				InitialCapacity: 10,
			}
		})
	} else {
		flt, err = createFleet(ctx, framework.Namespace)
	}
	assert.NoError(t, err)

	framework.AssertFleetCondition(t, flt, e2e.FleetReadyCount(flt.Spec.Replicas))
	request := &pb.AllocationRequest{
		Namespace:           framework.Namespace,
		GameServerSelectors: []*pb.GameServerSelector{{MatchLabels: map[string]string{agonesv1.FleetNameLabel: flt.ObjectMeta.Name}}},
		Scheduling:          pb.AllocationRequest_Packed,
		Metadata:            &pb.MetaPatch{Labels: map[string]string{"gslabel": "allocatedbytest", "blue-frog.fred_thing": "test.dog_fred-blue"}},
	}

	var response *pb.AllocationResponse
	// wait for the allocation system to come online
	err = wait.PollImmediate(2*time.Second, 5*time.Minute, func() (bool, error) {
		// create the grpc client each time, as we may end up looking at an old cert
		dialOpts, err := createRemoteClusterDialOption(ctx, allocatorClientSecretNamespace, allocatorClientSecretName, tlsCA)
		if err != nil {
			return false, err
		}

		conn, err := grpc.Dial(requestURL, dialOpts)
		if err != nil {
			logrus.WithError(err).Info("failing grpc.Dial")
			return false, nil
		}
		defer conn.Close() // nolint: errcheck

		grpcClient := pb.NewAllocationServiceClient(conn)
		response, err = grpcClient.Allocate(context.Background(), request)
		if err != nil {
			logrus.WithError(err).Info("failing Allocate request")
			return false, nil
		}
		validateAllocatorResponse(t, response)

		// let's do a re-allocation
		if runtime.FeatureEnabled(runtime.FeatureStateAllocationFilter) && runtime.FeatureEnabled(runtime.FeaturePlayerAllocationFilter) {
			logrus.Info("testing state allocation filter")
			request.GameServerSelectors[0].GameServerState = pb.GameServerSelector_ALLOCATED
			allocatedResponse, err := grpcClient.Allocate(context.Background(), request)
			require.NoError(t, err)
			require.Equal(t, response.GameServerName, allocatedResponse.GameServerName)
			validateAllocatorResponse(t, allocatedResponse)

			// do a capacity based allocation
			logrus.Info("testing capacity allocation filter")
			request.GameServerSelectors[0].Players = &pb.PlayerSelector{
				MinAvailable: 5,
				MaxAvailable: 10,
			}
			allocatedResponse, err = grpcClient.Allocate(context.Background(), request)
			require.NoError(t, err)
			require.Equal(t, response.GameServerName, allocatedResponse.GameServerName)
			validateAllocatorResponse(t, allocatedResponse)

			// do a capacity based allocation that should fail
			request.GameServerSelectors[0].GameServerState = pb.GameServerSelector_ALLOCATED
			request.GameServerSelectors[0].Players = &pb.PlayerSelector{MinAvailable: 99, MaxAvailable: 200}

			allocatedResponse, err = grpcClient.Allocate(context.Background(), request)
			assert.Nil(t, allocatedResponse)
			status, ok := status.FromError(err)
			require.True(t, ok)
			assert.Equal(t, codes.ResourceExhausted, status.Code())
		}

		return true, nil
	})

	assert.NoError(t, err)
}

func TestRestAllocatorWithDeprecatedRequired(t *testing.T) {
	ctx := context.Background()

	ip, port := getAllocatorEndpoint(ctx, t)
	requestURL := fmt.Sprintf(allocatorReqURLFmt, ip, port)
	tlsCA := refreshAllocatorTLSCerts(ctx, t, ip)

	flt, err := createFleet(ctx, framework.Namespace)
	if !assert.Nil(t, err) {
		return
	}
	framework.AssertFleetCondition(t, flt, e2e.FleetReadyCount(flt.Spec.Replicas))
	request := &pb.AllocationRequest{
		Namespace:                    framework.Namespace,
		RequiredGameServerSelector:   &pb.GameServerSelector{MatchLabels: map[string]string{agonesv1.FleetNameLabel: flt.ObjectMeta.Name}},
		PreferredGameServerSelectors: []*pb.GameServerSelector{{MatchLabels: map[string]string{agonesv1.FleetNameLabel: flt.ObjectMeta.Name}}},
		Scheduling:                   pb.AllocationRequest_Packed,
		Metadata:                     &pb.MetaPatch{Labels: map[string]string{"gslabel": "allocatedbytest"}},
	}
	tlsCfg, err := getTLSConfig(ctx, allocatorClientSecretNamespace, allocatorClientSecretName, tlsCA)
	if !assert.Nil(t, err) {
		return
	}
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsCfg,
		},
	}
	jsonRes, err := json.Marshal(request)
	if !assert.Nil(t, err) {
		return
	}
	req, err := http.NewRequest("POST", "https://"+requestURL+"/gameserverallocation", bytes.NewBuffer(jsonRes))
	if !assert.Nil(t, err) {
		logrus.WithError(err).Info("failed to create rest request")
		return
	}

	// wait for the allocation system to come online
	err = wait.PollImmediate(2*time.Second, 5*time.Minute, func() (bool, error) {
		resp, err := client.Do(req)
		if err != nil {
			logrus.WithError(err).Info("failed Allocate rest request")
			return false, nil
		}
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			logrus.WithError(err).Info("failed to read Allocate response body")
			return false, nil
		}
		defer resp.Body.Close() // nolint: errcheck
		var response pb.AllocationResponse
		err = json.Unmarshal(body, &response)
		if err != nil {
			logrus.WithError(err).Info("failed to unmarshal Allocate response")
			return false, nil
		}
		validateAllocatorResponse(t, &response)
		return true, nil
	})

	assert.NoError(t, err)
}

func TestRestAllocatorWithSelectors(t *testing.T) {
	ctx := context.Background()

	ip, port := getAllocatorEndpoint(ctx, t)
	requestURL := fmt.Sprintf(allocatorReqURLFmt, ip, port)
	tlsCA := refreshAllocatorTLSCerts(ctx, t, ip)

	flt, err := createFleet(ctx, framework.Namespace)
	if !assert.Nil(t, err) {
		return
	}
	framework.AssertFleetCondition(t, flt, e2e.FleetReadyCount(flt.Spec.Replicas))
	request := &pb.AllocationRequest{
		Namespace:           framework.Namespace,
		GameServerSelectors: []*pb.GameServerSelector{{MatchLabels: map[string]string{agonesv1.FleetNameLabel: flt.ObjectMeta.Name}}},
		Scheduling:          pb.AllocationRequest_Packed,
		Metadata:            &pb.MetaPatch{Labels: map[string]string{"gslabel": "allocatedbytest", "blue-frog.fred_thing": "test.dog_fred-blue"}},
	}
	tlsCfg, err := getTLSConfig(ctx, allocatorClientSecretNamespace, allocatorClientSecretName, tlsCA)
	if !assert.Nil(t, err) {
		return
	}
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsCfg,
		},
	}
	jsonRes, err := json.Marshal(request)
	if !assert.Nil(t, err) {
		return
	}
	req, err := http.NewRequest("POST", "https://"+requestURL+"/gameserverallocation", bytes.NewBuffer(jsonRes))
	if !assert.Nil(t, err) {
		logrus.WithError(err).Info("failed to create rest request")
		return
	}

	// wait for the allocation system to come online
	var response pb.AllocationResponse
	err = wait.PollImmediate(2*time.Second, 5*time.Minute, func() (bool, error) {
		resp, err := client.Do(req)
		if err != nil {
			logrus.WithError(err).Info("failed Allocate rest request")
			return false, nil
		}
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			logrus.WithError(err).Info("failed to read Allocate response body")
			return false, nil
		}
		defer resp.Body.Close() // nolint: errcheck
		err = json.Unmarshal(body, &response)
		if err != nil {
			logrus.WithError(err).Info("failed to unmarshal Allocate response")
			return false, nil
		}
		validateAllocatorResponse(t, &response)
		return true, nil
	})
	require.NoError(t, err)

	gs, err := framework.AgonesClient.AgonesV1().GameServers(framework.Namespace).Get(ctx, response.GameServerName, metav1.GetOptions{})
	require.NoError(t, err)
	assert.Equal(t, agonesv1.GameServerStateAllocated, gs.Status.State)
	assert.Equal(t, "allocatedbytest", gs.ObjectMeta.Labels["gslabel"])
	assert.Equal(t, "test.dog_fred-blue", gs.ObjectMeta.Labels["blue-frog.fred_thing"])
}

// Tests multi-cluster allocation by reusing the same cluster but across namespace.
// Multi-cluster is represented as two namespaces A and B in the same cluster.
// Namespace A received the allocation request, but because namespace B has the highest priority, A will forward the request to B.
func TestAllocatorCrossNamespace(t *testing.T) {
	ctx := context.Background()

	ip, port := getAllocatorEndpoint(ctx, t)
	requestURL := fmt.Sprintf(allocatorReqURLFmt, ip, port)
	tlsCA := refreshAllocatorTLSCerts(ctx, t, ip)

	// Create namespaces A and B
	namespaceA := framework.Namespace // let's reuse an existing one
	copyDefaultAllocatorClientSecret(ctx, t, namespaceA)

	namespaceB := fmt.Sprintf("allocator-b-%s", uuid.NewUUID())
	err := framework.CreateNamespace(namespaceB)
	if !assert.Nil(t, err) {
		return
	}
	defer func() {
		if derr := framework.DeleteNamespace(namespaceB); derr != nil {
			t.Error(derr)
		}
	}()

	policyName := fmt.Sprintf("a-to-b-%s", uuid.NewUUID())
	p := &multiclusterv1.GameServerAllocationPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      policyName,
			Namespace: namespaceA,
		},
		Spec: multiclusterv1.GameServerAllocationPolicySpec{
			Priority: 1,
			Weight:   1,
			ConnectionInfo: multiclusterv1.ClusterConnectionInfo{
				SecretName:          allocatorClientSecretName,
				Namespace:           namespaceB,
				AllocationEndpoints: []string{ip},
				ServerCA:            tlsCA,
			},
		},
	}
	createAllocationPolicy(ctx, t, p)

	// Create a fleet in namespace B. Allocation should not happen in A according to policy
	flt, err := createFleet(ctx, namespaceB)
	if !assert.Nil(t, err) {
		return
	}
	framework.AssertFleetCondition(t, flt, e2e.FleetReadyCount(flt.Spec.Replicas))
	request := &pb.AllocationRequest{
		Namespace: namespaceA,
		// Enable multi-cluster setting
		MultiClusterSetting: &pb.MultiClusterSetting{Enabled: true},
		GameServerSelectors: []*pb.GameServerSelector{{MatchLabels: map[string]string{agonesv1.FleetNameLabel: flt.ObjectMeta.Name}}},
	}

	// wait for the allocation system to come online
	err = wait.PollImmediate(2*time.Second, 5*time.Minute, func() (bool, error) {
		// create the grpc client each time, as we may end up looking at an old cert
		dialOpts, err := createRemoteClusterDialOption(ctx, namespaceA, allocatorClientSecretName, tlsCA)
		if err != nil {
			return false, err
		}

		conn, err := grpc.Dial(requestURL, dialOpts)
		if err != nil {
			logrus.WithError(err).Info("failing grpc.Dial")
			return false, nil
		}
		defer conn.Close() // nolint: errcheck

		grpcClient := pb.NewAllocationServiceClient(conn)
		response, err := grpcClient.Allocate(context.Background(), request)
		if err != nil {
			logrus.WithError(err).Info("failing Allocate request")
			return false, nil
		}
		validateAllocatorResponse(t, response)
		return true, nil
	})

	assert.NoError(t, err)
}

func copyDefaultAllocatorClientSecret(ctx context.Context, t *testing.T, toNamespace string) {
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

func createAllocationPolicy(ctx context.Context, t *testing.T, p *multiclusterv1.GameServerAllocationPolicy) {
	t.Helper()

	mc := framework.AgonesClient.MulticlusterV1()
	policy, err := mc.GameServerAllocationPolicies(p.Namespace).Create(ctx, p, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("creating allocation policy failed: %s", err)
	}
	t.Logf("created allocation policy %v", policy)
}

func getAllocatorEndpoint(ctx context.Context, t *testing.T) (string, int32) {
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

// createRemoteClusterDialOption creates a grpc client dial option with proper certs to make a remote call.
//
//nolint:unparam
func createRemoteClusterDialOption(ctx context.Context, namespace, clientSecretName string, tlsCA []byte) (grpc.DialOption, error) {
	tlsConfig, err := getTLSConfig(ctx, namespace, clientSecretName, tlsCA)
	if err != nil {
		return nil, err
	}

	return grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)), nil
}

func getTLSConfig(ctx context.Context, namespace, clientSecretName string, tlsCA []byte) (*tls.Config, error) {
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

func createFleet(ctx context.Context, namespace string) (*agonesv1.Fleet, error) {
	return createFleetWithOpts(ctx, namespace, func(*agonesv1.Fleet) {})
}

func createFleetWithOpts(ctx context.Context, namespace string, opts func(fleet *agonesv1.Fleet)) (*agonesv1.Fleet, error) {
	fleets := framework.AgonesClient.AgonesV1().Fleets(namespace)
	fleet := defaultFleet(namespace)
	opts(fleet)
	return fleets.Create(ctx, fleet, metav1.CreateOptions{})
}

func refreshAllocatorTLSCerts(ctx context.Context, t *testing.T, host string) []byte {
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

func validateAllocatorResponse(t *testing.T, resp *pb.AllocationResponse) {
	t.Helper()
	if !assert.NotNil(t, resp) {
		return
	}
	assert.Greater(t, len(resp.Ports), 0)
	assert.NotEmpty(t, resp.GameServerName)
	assert.NotEmpty(t, resp.Address)
	assert.NotEmpty(t, resp.NodeName)
}
