// Copyright 2020 Google LLC All Rights Reserved.
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
package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"flag"
	"fmt"
	"os"
	"strconv"
	"sync"
	"time"

	pb "agones.dev/agones/pkg/allocation/go"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func main() {
	keyFile := flag.String("key", "missing key", "the private key file for the client certificate in PEM format")
	certFile := flag.String("cert", "missing cert", "the public key file for the client certificate in PEM format")
	cacertFile := flag.String("cacert", "missing cacert", "the CA cert file for server signing certificate in PEM format")
	externalIP := flag.String("ip", "missing external IP", "the external IP for allocator server")
	port := flag.String("port", "443", "the port for allocator server")
	namespace := flag.String("namespace", "default", "the game server kubernetes namespace")
	multicluster := flag.Bool("multicluster", false, "set to true to enable the multi-cluster allocation")
	requestsPerSecond := flag.String("requestsPerSecond", "5", "the rate limited number of game server allocation requests per second")
	totalRequests := flag.String("totalRequests", "1000", "the total number of allocation requests to make")
	flag.Parse()
	endpoint := *externalIP + ":" + *port
	cert, err := os.ReadFile(*certFile)
	if err != nil {
		panic(err)
	}
	key, err := os.ReadFile(*keyFile)
	if err != nil {
		panic(err)
	}
	cacert, err := os.ReadFile(*cacertFile)
	if err != nil {
		panic(err)
	}
	throttleRate, err := strconv.Atoi(*requestsPerSecond)
	if err != nil {
		panic(err)
	}
	numRequests, err := strconv.Atoi(*totalRequests)
	if err != nil {
		panic(err)
	}

	request := &pb.AllocationRequest{
		Namespace: *namespace,
		GameServerSelectors: []*pb.GameServerSelector{
			{
				GameServerState: pb.GameServerSelector_ALLOCATED,
				MatchLabels: map[string]string{
					"version": "1.2.3",
				},
				Counters: map[string]*pb.CounterSelector{
					"players": {
						MinAvailable: 1,
					},
				},
			},
			{
				GameServerState: pb.GameServerSelector_READY,
				MatchLabels: map[string]string{
					"version": "1.2.3",
				},
				Counters: map[string]*pb.CounterSelector{
					"players": {
						MinAvailable: 1,
					},
				},
			},
		},
		Counters: map[string]*pb.CounterAction{
			"players": {
				Action: &wrapperspb.StringValue{Value: "Increment"},
				Amount: &wrapperspb.Int64Value{Value: 1},
			},
		},
		MultiClusterSetting: &pb.MultiClusterSetting{
			Enabled: *multicluster,
		},
		Priorities: []*pb.Priority{
			{Type: pb.Priority_Counter,
				Key:   "players",
				Order: pb.Priority_Ascending},
		},
		Scheduling: pb.AllocationRequest_Distributed,
	}
	// 	Scheduling: pb.AllocationRequest_Packed,
	// }

	dialOpts, err := createRemoteClusterDialOption(cert, key, cacert)
	if err != nil {
		panic(err)
	}
	conn, err := grpc.NewClient(endpoint, dialOpts)
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	grpcClient := pb.NewAllocationServiceClient(conn)
	rateLimit := time.Second / time.Duration(throttleRate) // throttle rate of calls per second
	fmt.Println("rateLimit", rateLimit)
	throttle := time.Tick(rateLimit)
	start :=  time.Now()
	fails := 0
	successes := 0

	// Make numRequests number of concurrent rate-limited game server allocation requests.
	// Rate limit to prevent failures due to resource conflicts.
  var wg sync.WaitGroup
	for range numRequests {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-throttle  // rate limit client calls
			err := allocateGameServer(grpcClient, request)
			if err != nil {
				fails += 1
			} else {
				successes += 1
			}
		}()
	}
	wg.Wait()
	diff := time.Now().Sub(start)
	fmt.Println("time spent", diff)
	fmt.Println("successes", successes)
	fmt.Println("fails", fails)
}

func allocateGameServer(client pb.AllocationServiceClient, request *pb.AllocationRequest) error {
	ctx := context.Background()
	_, err := client.Allocate(ctx, request)
	return err
}

// createRemoteClusterDialOption creates a grpc client dial option with TLS configuration.
func createRemoteClusterDialOption(clientCert, clientKey, caCert []byte) (grpc.DialOption, error) {
	// Load client cert
	cert, err := tls.X509KeyPair(clientCert, clientKey)
	if err != nil {
		return nil, err
	}
	tlsConfig := &tls.Config{Certificates: []tls.Certificate{cert}}
	if len(caCert) != 0 {
		// Load CA cert, if provided and trust the server certificate.
		// This is required for self-signed certs.
		tlsConfig.RootCAs = x509.NewCertPool()
		if !tlsConfig.RootCAs.AppendCertsFromPEM(caCert) {
			return nil, errors.New("only PEM format is accepted for server CA")
		}
	}
	return grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)), nil
}
